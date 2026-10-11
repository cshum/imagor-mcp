package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
