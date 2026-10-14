package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cshum/imagor-mcp/internal/pathgen"
	"github.com/cshum/imagor/imagorpath"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// FitTextInput asks for the largest font size at which a string still fits a box.
type FitTextInput struct {
	Text      string `json:"text" jsonschema:"the string to fit; it may contain %0A for a hard line break"`
	BoxWidth  int    `json:"box_width" jsonschema:"width of the area the text has to fit, in pixels"`
	BoxHeight int    `json:"box_height" jsonschema:"height of that area; the wrapped text may not exceed it"`
	Font      string `json:"font" jsonschema:"a Pango font with hyphens for spaces, family first and size last, e.g. Open-Sans-bold or Liberation-Serif-24"`
	MaxSize   int    `json:"max_size" jsonschema:"largest size to try, in points"`
	MinSize   int    `json:"min_size,omitempty" jsonschema:"smallest size to accept; the search stops here and reports that it does not fit"`
	Color     string `json:"color,omitempty" jsonschema:"text colour; affects nothing measurable, and defaults to black"`
}

// FitTextOutput reports the size that fits and how the text lays out at it.
type FitTextOutput struct {
	Font   string `json:"font" jsonschema:"the font argument to pass to text(), family and size together"`
	Size   int    `json:"size"`
	Lines  int    `json:"lines" jsonschema:"how many lines the text wrapped to at this size"`
	Width  int    `json:"width" jsonschema:"measured ink width in pixels"`
	Height int    `json:"height" jsonschema:"measured ink height in pixels"`
	Fits   bool   `json:"fits" jsonschema:"whether a size at or above min_size fits the box"`
	Probes int    `json:"probes" jsonschema:"how many renders the search needed"`
	Note   string `json:"note,omitempty"`
}

// FitTextTool finds the largest size at which text fits a box.
//
// imagor wraps text but never shrinks it, so a template that works for a short
// title silently overflows on a longer one. The only way to know the size is to
// render and measure, which is what this does: it renders the text on a
// transparent canvas at a candidate size, decodes the result, measures the ink,
// and searches for the largest size that fits. Rendering on a transparent canvas
// means the ink bounds are the text's own, with no background to separate out.
func (c Config) FitTextTool(ctx context.Context, _ *mcp.CallToolRequest, in FitTextInput) (*mcp.CallToolResult, FitTextOutput, error) {
	if c.BaseURL == "" {
		return nil, FitTextOutput{}, fmt.Errorf(
			"fit_text measures by rendering, so it needs IMAGOR_BASE_URL to reach a running imagor")
	}
	if strings.TrimSpace(in.Text) == "" {
		return nil, FitTextOutput{}, fmt.Errorf("fit_text needs some text to fit")
	}
	if in.BoxWidth <= 0 || in.BoxHeight <= 0 {
		return nil, FitTextOutput{}, fmt.Errorf("fit_text needs a box_width and box_height in pixels")
	}
	if in.Font == "" {
		in.Font = "sans"
	}
	if in.MaxSize <= 0 {
		in.MaxSize = 96
	}
	if in.MinSize <= 0 {
		in.MinSize = 12
	}
	if in.MinSize > in.MaxSize {
		return nil, FitTextOutput{}, fmt.Errorf("min_size %d is larger than max_size %d", in.MinSize, in.MaxSize)
	}

	probes := 0
	measure := func(size int) (ink, error) {
		probes++
		return c.measureText(ctx, in, size)
	}

	// The search is on the size alone, and the result is monotonic: if a size
	// fits, every smaller one does too. That makes a binary search exact, and
	// the number of renders logarithmic rather than linear.
	lo, hi := in.MinSize, in.MaxSize
	best := 0
	var bestInk ink
	for lo <= hi {
		mid := (lo + hi) / 2
		m, err := measure(mid)
		if err != nil {
			return nil, FitTextOutput{}, err
		}
		if m.height <= in.BoxHeight && m.width <= in.BoxWidth {
			best, bestInk = mid, m
			lo = mid + 1
			continue
		}
		hi = mid - 1
	}

	if best == 0 {
		// Nothing in range fits. Report the smallest size's shape so the caller
		// can see how far over it is rather than only that it failed.
		m, err := measure(in.MinSize)
		if err != nil {
			return nil, FitTextOutput{}, err
		}
		return nil, FitTextOutput{
			Font: fmt.Sprintf("%s-%d", in.Font, in.MinSize), Size: in.MinSize,
			Lines: m.lines, Width: m.width, Height: m.height, Fits: false, Probes: probes,
			Note: fmt.Sprintf("even at min_size %d the text measures %dx%d, over the %dx%d box. "+
				"Shorten the text, widen the box, or allow a smaller min_size.",
				in.MinSize, m.width, m.height, in.BoxWidth, in.BoxHeight),
		}, nil
	}

	out := FitTextOutput{
		Font: fmt.Sprintf("%s-%d", in.Font, best), Size: best,
		Lines: bestInk.lines, Width: bestInk.width, Height: bestInk.height,
		Fits: true, Probes: probes,
	}
	if best == in.MaxSize {
		out.Note = fmt.Sprintf("the text still fits at max_size %d, so a larger size would work too", in.MaxSize)
	}
	return nil, out, nil
}

// ink is the measured extent of rendered text.
type ink struct {
	width, height, lines int
}

// measureText renders the string at one size and reports the ink it produced.
func (c Config) measureText(ctx context.Context, in FitTextInput, size int) (ink, error) {
	colour := in.Color
	if colour == "" {
		colour = "000000"
	}
	// The probe canvas has to be tall enough to hold the text at any size the
	// search tries, or imagor clips it and the ink measured would shrink with the
	// canvas rather than with the font. The wrap width is what constrains the
	// text, so the height only has to be generous.
	canvasH := in.MaxSize * 20
	if canvasH < in.BoxHeight*4 {
		canvasH = in.BoxHeight * 4
	}
	// A comma separates filter arguments, and other characters are awkward to
	// keep in a path, so the text goes in as base64url. imagor accepts a b64:
	// prefix for exactly this, which is safer than escaping character by
	// character: a comma in the text otherwise reads as the next argument and
	// imagor renders nothing at all.
	// Pango reads the string as markup, so a bare & or < is a parse error rather
	// than a character. Escape those before encoding.
	escaped := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(in.Text)
	text := "b64:" + base64.RawURLEncoding.EncodeToString([]byte(escaped))
	path := fmt.Sprintf("%dx%d/filters:format(png):text(%s,0,0,%s-%d,%s,0,normal,%d,low,normal,word)/color:none",
		in.BoxWidth, canvasH, text, in.Font, size, colour, in.BoxWidth)

	// Sign through the same route the build tools use, so a probe is authorised
	// exactly as the composition it is measuring would be.
	var signer imagorpath.Signer
	if mode, err := c.Mode(); err != nil {
		return ink{}, err
	} else if mode == ModeSigned {
		if signer, err = c.Signer(); err != nil {
			return ink{}, err
		}
	}
	url := c.BaseURL + "/" + pathgen.BuildURL(imagorpath.Parse(path), signer)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ink{}, err
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ink{}, fmt.Errorf("rendering a probe from %s: %w", c.BaseURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return ink{}, fmt.Errorf("reading the probe from %s: %w", c.BaseURL, err)
	}
	if resp.StatusCode != http.StatusOK {
		return ink{}, fmt.Errorf("the probe render returned %d: %s", resp.StatusCode, firstLine(body))
	}
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return ink{}, fmt.Errorf("the probe render was not a decodable image: %w", err)
	}
	return measureInk(img), nil
}

// measureInk finds the bounding box of the non-transparent pixels. The probe is
// rendered on a transparent canvas, so anything opaque is text.
func measureInk(img image.Image) ink {
	b := img.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	rows := make([]bool, b.Max.Y-b.Min.Y)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			rows[y-b.Min.Y] = true
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if maxX < minX || maxY < minY {
		return ink{}
	}
	lines := 0
	for i, r := range rows {
		if r && (i == 0 || !rows[i-1]) {
			lines++
		}
	}
	return ink{width: maxX - minX + 1, height: maxY - minY + 1, lines: lines}
}
