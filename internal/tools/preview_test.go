package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// pngBytes is a small valid PNG, so the preview path can be exercised without a
// real renderer.
func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			img.Set(x, y, color.RGBA{R: 200, G: 30, B: 30, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// previewServer answers the metadata request with a report and everything else
// with an image, which is the shape a real imagor has.
func previewServer(t *testing.T, meta string, img []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "meta") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(meta))
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(img)
	}))
}

// textOf returns the text half of a result.
func textOf(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

// A composition can be reported as applied and still be wrong: the filters ran,
// the dimensions are right, and the result is not what was asked for. A caller
// that can see images needs the rendered result to tell those apart.
func TestVerifyAttachesTheRenderedResult(t *testing.T) {
	srv := previewServer(t, `{"width":8,"height":8,"filters":[{"name":"format","processed":true}]}`, pngBytes(t))
	defer srv.Close()

	cfg := Config{BaseURL: srv.URL, Unsafe: true}
	res, out, err := cfg.VerifyURLTool(context.Background(), nil, VerifyInput{
		TransformInput: TransformInput{Image: "https://example.com/a.jpg", Width: 8, Height: 8},
	})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(res.Content) != 2 {
		t.Fatalf("want a text part and an image part, got %d", len(res.Content))
	}
	img, ok := res.Content[1].(*mcp.ImageContent)
	if !ok {
		t.Fatalf("the second part is %T, want an image", res.Content[1])
	}
	if img.MIMEType != "image/png" {
		t.Errorf("mime type %q, want image/png", img.MIMEType)
	}
	if len(img.Data) == 0 {
		t.Error("the image part carries no data")
	}
	// The wire form is what a client sees, so check it is a real image block.
	raw, err := json.Marshal(img)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"type":"image"`) {
		t.Errorf("the image does not marshal as an image block: %s", raw[:80])
	}
	if out.Status != 200 || !out.Verified {
		t.Errorf("the report was lost when the preview was attached: %+v", out)
	}
}

// The preview is asked for through a scaled path. Images cost tokens by area, so
// attaching the full render would spend several times as much for no more answer
// to whether a panel sits in the right place.
func TestVerifyAsksForAScaledPreview(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.URL.Path)
		if strings.Contains(r.URL.Path, "meta") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"width":8,"height":8,"filters":[]}`))
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes(t))
	}))
	defer srv.Close()

	cfg := Config{BaseURL: srv.URL, Unsafe: true}
	if _, _, err := cfg.VerifyURLTool(context.Background(), nil, VerifyInput{
		TransformInput: TransformInput{Image: "https://example.com/a.jpg", Width: 1800, Height: 942},
	}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	var preview string
	for _, p := range got {
		if !strings.Contains(p, "meta") {
			preview = p
		}
	}
	if preview == "" {
		t.Fatal("no preview was requested")
	}
	if !strings.Contains(preview, "fit-in/768x768/") {
		t.Errorf("the preview is not scaled: %s", preview)
	}
	// The scaled request must scale the composition rather than replace it: the
	// full-size path is what the report describes, and it has to be inside the
	// preview request as the source.
	if !strings.Contains(preview, "1800x942") {
		t.Errorf("the preview path lost the composition it is previewing: %s", preview)
	}
}

// The url is what the caller serves and the attachment is not it, so the report has to
// say so. This was added after a real session where the agent showed the picture and
// dropped the url, leaving the user with an image they could not serve.
func TestAttachedPreviewNamesTheURLAsTheDeliverable(t *testing.T) {
	out := VerifyOutput{Status: 200, Width: 1200, Height: 630,
		AbsoluteURL: "http://localhost:8000/unsafe/1200x630/filters:image(a/b,c,d)/a.jpg"}
	txt := verifySummary(out, "", true, 64)
	for _, want := range []string{
		out.AbsoluteURL,
		"Give the user the url",
		"it is the output",
		"scaled preview",
	} {
		if !strings.Contains(txt, want) {
			t.Errorf("missing %q in:\n%s", want, txt)
		}
	}

	// With nothing attached there is no preview to distinguish from the file, so the
	// instruction would be noise.
	bare := verifySummary(out, "", false, 0)
	if strings.Contains(bare, "Give the user the url") {
		t.Errorf("the instruction belongs only where a preview was attached:\n%s", bare)
	}
	if !strings.Contains(bare, out.AbsoluteURL) {
		t.Error("the url is reported whether or not a preview came back")
	}
}

// An attached image is delivered to the client whether or not the model wants
// it, so a caller that only wants the report can say so.
func TestVerifyCanBeAskedForNoPreview(t *testing.T) {
	srv := previewServer(t, `{"width":8,"height":8,"filters":[]}`, pngBytes(t))
	defer srv.Close()

	no := false
	cfg := Config{BaseURL: srv.URL, Unsafe: true}
	res, out, err := cfg.VerifyURLTool(context.Background(), nil, VerifyInput{
		TransformInput: TransformInput{Image: "https://example.com/a.jpg", Width: 8, Height: 8},
		Preview:        &no,
	})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if res != nil && len(res.Content) > 1 {
		t.Error("an image was attached when the caller asked for none")
	}
	if !out.Verified {
		t.Error("asking for no preview should not cost the report")
	}
}

// The image must reach a client as decodable bytes. A []byte field is already
// base64 on the wire, so encoding it here as well would leave a client holding
// base64 after one decode - a picture that never renders, from a tool that
// reported success.
func TestVerifyAttachesDecodableImageBytes(t *testing.T) {
	want := pngBytes(t)
	srv := previewServer(t, `{"width":8,"height":8,"filters":[]}`, want)
	defer srv.Close()

	cfg := Config{BaseURL: srv.URL, Unsafe: true}
	res, _, err := cfg.VerifyURLTool(context.Background(), nil, VerifyInput{
		TransformInput: TransformInput{Image: "https://example.com/a.jpg", Width: 8, Height: 8},
	})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	img, ok := res.Content[1].(*mcp.ImageContent)
	if !ok {
		t.Fatalf("second part is %T, want an image", res.Content[1])
	}
	// Decoding the marshalled wire form once has to yield the original bytes. If
	// the data were encoded twice, one decode would still leave base64 text.
	raw, err := json.Marshal(img)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	got, err := base64.StdEncoding.DecodeString(wire.Data)
	if err != nil {
		t.Fatalf("the wire data is not base64: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("one decode did not yield the image: got %d bytes starting %q, want %d starting %q",
			len(got), got[:min(8, len(got))], len(want), want[:8])
	}
	if _, err := png.Decode(bytes.NewReader(got)); err != nil {
		t.Errorf("the decoded bytes are not a PNG: %v", err)
	}
}

// A result too large to attach is reported as such rather than silently dropped,
// and the report still arrives - the picture is a convenience on top of it.
func TestVerifySaysWhyThereIsNoPreview(t *testing.T) {
	srv := previewServer(t, `{"width":1,"height":1,"filters":[]}`,
		bytes.Repeat([]byte{0}, previewBytes+1024))
	defer srv.Close()

	cfg := Config{BaseURL: srv.URL, Unsafe: true}
	res, out, err := cfg.VerifyURLTool(context.Background(), nil, VerifyInput{
		TransformInput: TransformInput{Image: "https://example.com/a.jpg", Width: 1, Height: 1},
	})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(res.Content) != 1 {
		t.Fatalf("an oversized result should not be attached, got %d parts", len(res.Content))
	}
	if !out.Verified {
		t.Error("the report should survive a preview that could not be attached")
	}
	if !strings.Contains(textOf(t, res), "larger than") {
		t.Errorf("the caller is not told why there is no picture: %s", textOf(t, res))
	}
}
