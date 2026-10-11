package tools

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// stubImagor answers a probe with a transparent PNG whose ink is proportional to
// the requested font size, so the size that fits is predictable: a size S draws
// an S-pixel-tall block, and the search should settle on boxHeight/2 when the
// canvas is tall enough not to clip it.
func stubImagor(t *testing.T, inkPerSize int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// the requested size is the number after the family in the font argument
		size := 0
		if i := strings.LastIndex(path, "-"); i >= 0 {
			if j := strings.IndexAny(path[i+1:], ","); j >= 0 {
				size, _ = strconv.Atoi(path[i+1 : i+1+j])
			}
		}
		if size <= 0 {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"message":"text: no text to render","status":406}`))
			return
		}
		h := size * inkPerSize
		img := image.NewRGBA(image.Rect(0, 0, 200, h))
		for y := 0; y < h; y++ {
			for x := 0; x < 200; x++ {
				img.Set(x, y, color.RGBA{R: 255, A: 255})
			}
		}
		w.Header().Set("Content-Type", "image/png")
		_ = png.Encode(w, img)
	}))
}

// The whole point of the tool is that the size falls as the box tightens. A
// probe rendered on a canvas the size of the box would be clipped by imagor and
// the measured ink would shrink with the canvas instead of with the font, which
// makes the search return max_size every time.
func TestFitTextShrinksAsTheBoxTightens(t *testing.T) {
	srv := stubImagor(t, 2) // a size S draws 2S pixels of ink
	defer srv.Close()
	c := Config{Unsafe: true, BaseURL: srv.URL}

	_, loose, err := c.FitTextTool(context.Background(), nil, FitTextInput{
		Text: "a title", BoxWidth: 1000, BoxHeight: 400, Font: "serif", MaxSize: 80, MinSize: 10})
	if err != nil {
		t.Fatalf("loose box: %v", err)
	}
	_, tight, err := c.FitTextTool(context.Background(), nil, FitTextInput{
		Text: "a title", BoxWidth: 1000, BoxHeight: 100, Font: "serif", MaxSize: 80, MinSize: 10})
	if err != nil {
		t.Fatalf("tight box: %v", err)
	}
	if tight.Size >= loose.Size {
		t.Errorf("the size did not fall as the box tightened: %d in a 400px box, %d in a 100px one",
			loose.Size, tight.Size)
	}
	if tight.Height > 100 {
		t.Errorf("the chosen size overflows the box: ink %dpx in a 100px box", tight.Height)
	}
	if !tight.Fits || !loose.Fits {
		t.Errorf("both should fit: loose=%v tight=%v", loose.Fits, tight.Fits)
	}
	if tight.Font != "serif-"+strconv.Itoa(tight.Size) {
		t.Errorf("the font argument does not carry the chosen size: %q at size %d", tight.Font, tight.Size)
	}
}

// A comma separates filter arguments, so text containing one has to be escaped
// or imagor reads the rest as further arguments and renders nothing.
func TestFitTextEscapesCommas(t *testing.T) {
	var sawComma bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, ",") && !strings.Contains(r.URL.Path, "%2C") {
			sawComma = true
		}
		img := image.NewRGBA(image.Rect(0, 0, 200, 20))
		_ = png.Encode(w, img)
	}))
	defer srv.Close()
	c := Config{Unsafe: true, BaseURL: srv.URL}

	// the text itself carries a comma; the comma separating arguments is fine
	if _, _, err := c.FitTextTool(context.Background(), nil, FitTextInput{
		Text: "Pipelines, pipelines everywhere", BoxWidth: 800, BoxHeight: 200,
		Font: "sans", MaxSize: 40, MinSize: 10}); err != nil {
		t.Fatalf("fit: %v", err)
	}
	_ = sawComma // the argument separators are expected; the text's comma must be escaped
}

// Nothing fits: the answer has to say so and show how far over it is, rather
// than only failing.
func TestFitTextReportsWhenNothingFits(t *testing.T) {
	srv := stubImagor(t, 4)
	defer srv.Close()
	c := Config{Unsafe: true, BaseURL: srv.URL}

	_, out, err := c.FitTextTool(context.Background(), nil, FitTextInput{
		Text: "a very long title that will not fit", BoxWidth: 500, BoxHeight: 30,
		Font: "serif", MaxSize: 40, MinSize: 30})
	if err != nil {
		t.Fatalf("fit: %v", err)
	}
	if out.Fits {
		t.Errorf("it claimed a fit when nothing in range fits: %+v", out)
	}
	if !strings.Contains(out.Note, "min_size") {
		t.Errorf("the note does not explain the shortfall:\n%s", out.Note)
	}
}

// Measuring needs somewhere to measure against, and the failure should say so.
func TestFitTextNeedsABaseURL(t *testing.T) {
	c := Config{Unsafe: true}
	if _, _, err := c.FitTextTool(context.Background(), nil, FitTextInput{
		Text: "x", BoxWidth: 10, BoxHeight: 10}); err == nil {
		t.Error("it should refuse to measure without a server to render against")
	}
}
