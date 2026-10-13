package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// A signature disagreement has one likely cause, and the settings that disagree
// are invisible in the response. Without naming them the caller is left to hunt
// through ports and versions.
func TestVerifyNamesTheSignerOnASignatureMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"url signature mismatch","status":403}`))
	}))
	defer srv.Close()

	c := Config{Secret: "a-secret", BaseURL: srv.URL}
	_, out, err := c.VerifyURLTool(context.Background(), nil, VerifyInput{Image: "http://x/a.png"})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if out.Status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", out.Status)
	}
	for _, want := range []string{"IMAGOR_SIGNER_TYPE", "IMAGOR_SIGNER_TRUNCATE", "configured to match"} {
		if !strings.Contains(out.Note, want) {
			t.Errorf("the note does not mention %q:\n%s", want, out.Note)
		}
	}
	// The server's own body still has to survive into the note.
	if !strings.Contains(out.Note, "url signature mismatch") {
		t.Errorf("the note dropped the server's body:\n%s", out.Note)
	}
}

// A 403 that is not about a signature must not blame the signer.
func TestVerifyDoesNotBlameTheSignerForAnUnrelatedForbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"source host not allowed","status":403}`))
	}))
	defer srv.Close()

	c := Config{Secret: "a-secret", BaseURL: srv.URL}
	_, out, err := c.VerifyURLTool(context.Background(), nil, VerifyInput{Image: "http://x/a.png"})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if strings.Contains(out.Note, "IMAGOR_SIGNER_TYPE") {
		t.Errorf("a non-signature 403 was blamed on the signer:\n%s", out.Note)
	}
}

// An absent field and an empty one mean different things: absent means the
// release predates the field, empty means the request had nothing to report. The
// note has to say which one it saw rather than infer a version.
func TestVerifySaysWhatItObservedWhenTheFieldIsAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"width":10,"height":10}`))
	}))
	defer srv.Close()

	c := Config{Secret: "a-secret", BaseURL: srv.URL}
	_, out, err := c.VerifyURLTool(context.Background(), nil, VerifyInput{Image: "http://x/a.png"})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if out.Server.ReportsFilters {
		t.Error("it reported the server as reporting, when the field was absent")
	}
	if !strings.Contains(out.Note, `"filters"`) {
		t.Errorf("the note does not name the field it looked for:\n%s", out.Note)
	}
}

// The field present but empty is a different case, and must not be reported as
// an old server.
func TestVerifyDistinguishesAnEmptyReportFromAnAbsentOne(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"width":10,"height":10,"filters":[]}`))
	}))
	defer srv.Close()

	c := Config{Secret: "a-secret", BaseURL: srv.URL}
	_, out, err := c.VerifyURLTool(context.Background(), nil, VerifyInput{Image: "http://x/a.png"})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !out.Server.ReportsFilters || !out.Server.Compatible {
		t.Errorf("an empty report should still count as reporting: %+v", out.Server)
	}
	if !strings.Contains(out.Note, "no filters were requested") {
		t.Errorf("the note does not say the request had nothing to report:\n%s", out.Note)
	}
}

// An unsafe server sends no signature at all, so a signature refusal means the
// other side is not accepting unsigned requests. That is a different fault from
// a signer disagreement, and the advice has to say which one it is.
func TestVerifyBlamesTheModeWhenThisServerIsUnsafe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"url signature mismatch","status":403}`))
	}))
	defer srv.Close()

	c := Config{Unsafe: true, BaseURL: srv.URL}
	_, out, err := c.VerifyURLTool(context.Background(), nil, VerifyInput{Image: "http://x/a.png"})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	for _, want := range []string{"unsafe mode", "IMAGOR_UNSAFE=1", "IMAGOR_SECRET"} {
		if !strings.Contains(out.Note, want) {
			t.Errorf("the note does not mention %q:\n%s", want, out.Note)
		}
	}
	if strings.Contains(out.Note, "IMAGOR_SIGNER_TYPE") {
		t.Errorf("it advised a signer this server is not using:\n%s", out.Note)
	}
}

// A filter the server recognised and left the image unchanged is the failure
// this whole tool exists to catch: the request succeeds, the image comes back,
// and the filter did not run. A note is not enough, because a caller checking
// for an error would not read it.
func TestVerifyRefusesADeclinedFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"width":300,"height":300,"filters":[{"name":"blur","processed":false}]}`))
	}))
	defer srv.Close()

	c := Config{Secret: "a-secret", BaseURL: srv.URL}
	_, _, err := c.VerifyURLTool(context.Background(), nil, VerifyInput{
		Image:   "http://x/a.png",
		Filters: []FilterInput{{Name: "blur", Args: "0"}},
	})
	if err == nil {
		t.Fatal("a declined filter did not fail the verify, so a caller checking for an error sees success")
	}
	for _, want := range []string{"blur", "left the image unchanged", "200", "allow_declined"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %q:\n%s", want, err)
		}
	}
}

// A filter that applied is not a fault, and must not be reported as one.
func TestVerifyAcceptsAnAppliedFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"width":300,"height":300,"filters":[{"name":"blur","processed":true}]}`))
	}))
	defer srv.Close()

	c := Config{Secret: "a-secret", BaseURL: srv.URL}
	_, out, err := c.VerifyURLTool(context.Background(), nil, VerifyInput{
		Image:   "http://x/a.png",
		Filters: []FilterInput{{Name: "blur", Args: "5"}},
	})
	if err != nil {
		t.Fatalf("an applied filter failed the verify: %v", err)
	}
	if !strings.Contains(out.Note, "applied") {
		t.Errorf("the note does not say it applied:\n%s", out.Note)
	}
}

// A filter can be deliberately conditional, and the caller has to be able to say
// so without losing the report.
func TestVerifyAllowsADeclineWhenTheCallerSaysSo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"width":300,"height":300,"filters":[{"name":"trim","processed":false}]}`))
	}))
	defer srv.Close()

	allow := true
	c := Config{Secret: "a-secret", BaseURL: srv.URL}
	_, out, err := c.VerifyURLTool(context.Background(), nil, VerifyInput{
		Image:         "http://x/a.png",
		Filters:       []FilterInput{{Name: "trim", Args: ""}},
		AllowDeclined: &allow,
	})
	if err != nil {
		t.Fatalf("the caller allowed a decline and still got an error: %v", err)
	}
	if !out.Verified {
		t.Error("the report was dropped when the decline was allowed")
	}
	if !strings.Contains(out.Note, "trim") {
		t.Errorf("the note does not name the declined filter:\n%s", out.Note)
	}
}

// A decline inside a loaded path is the same failure as one at the top level,
// and harder to see: image() nests its own filters, so the walk has to descend.
func TestVerifyRefusesADeclineInsideALoadedPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"width":300,"height":300,"filters":[` +
			`{"name":"image","processed":true,"filters":[{"name":"blur","processed":false}]}]}`))
	}))
	defer srv.Close()

	c := Config{Secret: "a-secret", BaseURL: srv.URL}
	_, _, err := c.VerifyURLTool(context.Background(), nil, VerifyInput{
		Image:   "http://x/a.png",
		Filters: []FilterInput{{Name: "image", Args: "100x100/filters:blur(0)/http://x/b.png,0,0"}},
	})
	if err == nil {
		t.Fatal("a decline nested inside a loaded path was not reported as a fault")
	}
	if !strings.Contains(err.Error(), "blur") {
		t.Errorf("the error does not name the nested filter:\n%s", err)
	}
}

// The image URL must not carry the meta segment, whatever the caller asked for.
//
// The preview loads the image URL as an image. When the caller passed meta:true,
// toParams carried it through, the image URL became the metadata URL, and the
// preview asked the server to decode a JSON document - a 500 that arrived as a
// note rather than an error, so the report still came back and the tests passed.
func TestVerifyKeepsTheMetaSegmentOutOfTheImageURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/meta/") {
			_, _ = w.Write([]byte(`{"width":300,"height":300,"filters":[{"name":"blur","processed":true}]}`))
			return
		}
		// anything that is not the metadata request is the preview, and it must
		// be an image or the preview fails
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(onePixelPNG)
	}))
	defer srv.Close()

	c := Config{Secret: "a-secret", BaseURL: srv.URL}
	res, out, err := c.VerifyURLTool(context.Background(), nil, VerifyInput{
		Image:   "http://x/a.png",
		Filters: []FilterInput{{Name: "blur", Args: "5"}},
		Meta:    true,
	})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if strings.Contains(out.URL, "/meta/") {
		t.Errorf("the image URL carries the meta segment: %q", out.URL)
	}
	if !strings.Contains(out.MetaURL, "meta") {
		t.Errorf("the meta URL should carry it: %q", out.MetaURL)
	}
	// The preview is attached as content, so its presence is the assertion: an
	// image came back. A 500 leaves the text report intact, which is why this
	// has to check the attachment rather than the note.
	attached := false
	for _, c := range res.Content {
		if _, ok := c.(*mcp.ImageContent); ok {
			attached = true
		}
	}
	if !attached {
		t.Errorf("no preview image came back; the summary was %q",
			res.Content[0].(*mcp.TextContent).Text)
	}
}

// onePixelPNG is a real 1x1 PNG. The preview path checks that what came back is
// an image, so a stub returning a PNG signature and nothing else is rejected.
var onePixelPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
	0x0d, 0x49, 0x44, 0x41, 0x54, 0x78, 0xda, 0x63, 0xfc, 0xcf, 0xc0, 0x50,
	0x0f, 0x00, 0x04, 0x85, 0x01, 0x80, 0x84, 0xa9, 0x8c, 0x21, 0x00, 0x00,
	0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}
