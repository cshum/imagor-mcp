// Package tools defines the MCP tool surface for imagor-mcp.
//
// The tools are deliberately not a generic "sign anything" endpoint. Each
// tool builds a path from typed, validated parameters; nothing here can be
// used to sign an arbitrary string.
package tools

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	imagormcp "github.com/cshum/imagor-template"
	"github.com/cshum/imagor/imagorpath"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	filtercat "github.com/cshum/imagor-template/internal/filters"
	"github.com/cshum/imagor-template/internal/pathgen"
)

// Mode is how the server produces request paths.
type Mode string

const (
	// ModeSigned signs every path with IMAGOR_SECRET. Use in production.
	ModeSigned Mode = "signed"
	// ModeUnsafe emits "unsafe/" paths and signs nothing. Development only;
	// the target imagor must itself be running with IMAGOR_UNSAFE=1.
	ModeUnsafe Mode = "unsafe"
)

// Config is the runtime configuration, read from the environment so that the
// signing secret never travels through the MCP tool surface.
type Config struct {
	// Secret is IMAGOR_SECRET. Setting it selects ModeSigned.
	Secret string
	// Unsafe is IMAGOR_UNSAFE. Setting it selects ModeUnsafe, and it must be
	// explicit: the server never falls back to unsigned URLs on its own.
	Unsafe bool
	// SignerType is IMAGOR_SIGNER_TYPE: sha1 (default), sha256 or sha512.
	SignerType string
	// Truncate is IMAGOR_SIGNER_TRUNCATE, in characters. 0 means no truncation.
	Truncate int
	// BaseURL is the optional imagor origin. When set, tool output carries an
	// absolute URL alongside the request path.
	BaseURL string
}

// ConfigFromEnv reads configuration from the process environment.
func ConfigFromEnv() Config {
	truncate, _ := strconv.Atoi(strings.TrimSpace(os.Getenv("IMAGOR_SIGNER_TRUNCATE")))
	return Config{
		Secret:     strings.TrimSpace(os.Getenv("IMAGOR_SECRET")),
		Unsafe:     isTruthy(os.Getenv("IMAGOR_UNSAFE")),
		SignerType: strings.ToLower(strings.TrimSpace(os.Getenv("IMAGOR_SIGNER_TYPE"))),
		Truncate:   truncate,
		BaseURL:    strings.TrimSuffix(strings.TrimSpace(os.Getenv("IMAGOR_BASE_URL")), "/"),
	}
}

// signingMismatchAdvice explains a signature refusal in terms of this server's
// own settings. A signature covers the whole path, so a refusal means this
// server and the imagor it is pointed at disagree about how to sign - and the
// settings that disagree are not visible in the response. There are two shapes:
// this server signs and the other expects something else, or this server sends
// no signature at all because it is in unsafe mode.
func (c Config) signingMismatchAdvice() string {
	if c.Unsafe {
		return "This server is in unsafe mode, so it sends no signature at all, and the imagor " +
			"it is pointed at is not accepting unsigned requests. Set IMAGOR_UNSAFE=1 on that " +
			"imagor, or give this server IMAGOR_SECRET so it signs. Check that before anything else."
	}
	return fmt.Sprintf("This server signs with IMAGOR_SIGNER_TYPE=%s and IMAGOR_SIGNER_TRUNCATE=%d, "+
		"and the secret it holds must be the same one that imagor was started with. The running "+
		"imagor has to be configured to match on all three. Check those before anything else.",
		c.signerName(), c.Truncate)
}

// signerName reports the signing algorithm in the form a caller would set it,
// so an error can name the setting rather than an empty string.
func (c Config) signerName() string {
	if c.SignerType == "" {
		return "sha1 (the default)"
	}
	return c.SignerType
}

// isTruthy mirrors imagor's own boolean environment parsing.
func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// Mode reports which of the three modes the configuration selects.
//
// IMAGOR_SECRET and IMAGOR_UNSAFE state opposite intentions, so setting both
// is refused rather than silently resolved. Guessing which one the operator
// meant is how an unsigned deployment reaches production.
func (c Config) Mode() (Mode, error) {
	switch {
	case c.Secret != "" && c.Unsafe:
		return "", fmt.Errorf(
			"IMAGOR_SECRET and IMAGOR_UNSAFE are both set and contradict each other: " +
				"set IMAGOR_SECRET for signed URLs, or IMAGOR_UNSAFE alone for development, but not both")
	case c.Secret != "":
		return ModeSigned, nil
	case c.Unsafe:
		return ModeUnsafe, nil
	default:
		return "", fmt.Errorf(
			"no mode selected: set IMAGOR_SECRET to sign request paths, or IMAGOR_UNSAFE=1 to target an unsafe imagor instance. This server produces signed or unsafe paths, so it cannot run without one of them")
	}
}

// Signer returns the configured signer, or nil when no secret is set.
func (c Config) Signer() (imagorpath.Signer, error) {
	if c.Secret == "" {
		return nil, nil
	}
	var alg func() hash.Hash
	switch c.SignerType {
	case "", "sha1":
		alg = sha1.New
	case "sha256":
		alg = sha256.New
	case "sha512":
		alg = sha512.New
	default:
		return nil, fmt.Errorf("unsupported IMAGOR_SIGNER_TYPE %q (want sha1, sha256 or sha512)", c.SignerType)
	}
	return imagorpath.NewHMACSigner(alg, c.Truncate, c.Secret), nil
}

// textArgs builds the argument string for a filter, encoding a text() argument
// when one was given separately.
//
// A text argument travels in the path, where a comma separates arguments and a
// question mark starts a query string. A real title containing either does not
// survive: a comma truncates it and imagor renders nothing while still returning
// 200, and a question mark changes the path a signature covers, so the request is
// refused as a signature mismatch. base64url keeps the text out of the path
// entirely. Pango then receives the decoded string as markup, so markup
// characters are escaped first.
// sourceArgs builds the argument string for a filter whose first argument is a
// nested source. The encoding rules live in encoding.go, alongside the ones the
// template layer uses, so both encode a value the same way.
func sourceArgs(source, tail string) string {
	if source == "" {
		return tail
	}
	encoded := encodeSourceValue(source)
	if tail == "" {
		return encoded
	}
	return encoded + "," + tail
}

func textArgs(text, tail string) string {
	if text == "" {
		return tail
	}
	encoded := encodeTextValue(text)
	if tail == "" {
		return encoded
	}
	return encoded + "," + tail
}

// FilterInput is one imagor filter and its arguments.
type FilterInput struct {
	Name string `json:"name" jsonschema:"filter name, for example fill, format, quality, blur, rotate, invert, watermark, text, image, max_age"`
	Args string `json:"args,omitempty" jsonschema:"comma-separated arguments for the filter, for example \"white\" for fill, \"jpeg\" for format, \"80\" for quality. Omit for filters that take none."`
	// Text is for text() only, and exists because a title is the one argument
	// that cannot safely travel as a comma-separated string.
	Text string `json:"text,omitempty" jsonschema:"for text(): the string to render. Prefer this to args whenever the text comes from data, because a comma in it truncates the string silently with a 200 response and a question mark changes the path a signature covers. label() is the legacy form of text()"`
	// Source is for image() and watermark(), whose first argument is a nested
	// path or URL. A ? in it ends the path the signature covers, and a , splits
	// the argument list: measured, a URL with a ? gave 403 and one with a ,
	// gave 404. Both go through b64 here.
	Source string `json:"source,omitempty" jsonschema:"for image(): the nested source, a path or URL. Prefer this to args whenever the source is a URL that may contain ? or , - written raw those give 403 and 404. watermark() is the legacy form of image()"`
}

// TransformInput describes an image transformation. Every field maps onto a
// concept in the imagor path grammar; there are no invented parameters.
type TransformInput struct {
	Image string `json:"image" jsonschema:"source image: a URL or path such as \"example.com/a.jpg\", or a synthetic source such as \"color:red\" or \"color:none\""`

	Width  int `json:"width,omitempty" jsonschema:"target width in pixels; 0 means automatic"`
	Height int `json:"height,omitempty" jsonschema:"target height in pixels; 0 means automatic"`

	FitIn         bool `json:"fit_in,omitempty" jsonschema:"fit inside width x height instead of cropping to fill; the result can be smaller than the box on one side"`
	FullFitIn     bool `json:"full_fit_in,omitempty" jsonschema:"cover the box without cropping: one side matches width x height and the other exceeds it, keeping the aspect. Use for an image that must fill a panel without cutting into it. Requires fit_in, and add upscale() when the source is smaller than the box"`
	AdaptiveFitIn bool `json:"adaptive_fit_in,omitempty" jsonschema:"invert width and height when that yields better definition; requires fit_in"`
	Stretch       bool `json:"stretch,omitempty" jsonschema:"stretch to exactly width x height, ignoring aspect ratio, which distorts the image; use full_fit_in to fill a box of a different shape without distorting it"`
	Smart         bool `json:"smart,omitempty" jsonschema:"use content-aware smart crop"`

	Trim          bool   `json:"trim,omitempty" jsonschema:"trim surrounding space using the top-left pixel colour"`
	TrimBy        string `json:"trim_by,omitempty" jsonschema:"trim anchor, either \"top-left\" or \"bottom-right\""`
	TrimTolerance int    `json:"trim_tolerance,omitempty" jsonschema:"trim tolerance from 0 to 255"`

	CropLeft   float64 `json:"crop_left,omitempty" jsonschema:"crop left edge: a fraction 0-1 of the width, or absolute pixels when greater than 1"`
	CropTop    float64 `json:"crop_top,omitempty" jsonschema:"crop top edge: a fraction 0-1 of the height, or absolute pixels when greater than 1"`
	CropRight  float64 `json:"crop_right,omitempty" jsonschema:"crop right edge: a fraction 0-1 of the width, or absolute pixels when greater than 1"`
	CropBottom float64 `json:"crop_bottom,omitempty" jsonschema:"crop bottom edge: a fraction 0-1 of the height, or absolute pixels when greater than 1"`

	HAlign string `json:"h_align,omitempty" jsonschema:"horizontal alignment, either \"left\" or \"right\""`
	VAlign string `json:"v_align,omitempty" jsonschema:"vertical alignment, either \"top\" or \"bottom\""`

	HFlip bool `json:"h_flip,omitempty" jsonschema:"mirror horizontally"`
	VFlip bool `json:"v_flip,omitempty" jsonschema:"mirror vertically"`
	Meta  bool `json:"meta,omitempty" jsonschema:"return image metadata only, without processing the image"`

	Filters []FilterInput `json:"filters,omitempty" jsonschema:"ordered filters, applied left to right. Arguments are validated against imagor's catalogue: call list_imagor_filters for a filter's signature, description and accepted argument forms. image() loads its argument as its own canvas, so positions inside it are relative to the overlay"`

	ExpireUnixMillis int64 `json:"expire_unix_millis,omitempty" jsonschema:"add an expire() filter so the URL stops working after this unix-milliseconds timestamp"`
	MaxAgeSeconds    int   `json:"max_age_seconds,omitempty" jsonschema:"add a max_age() filter limiting how long the response may be cached"`
}

// BuildOutput is the result of building a path or URL.
type BuildOutput struct {
	Mode        string   `json:"mode" jsonschema:"how the path was produced: signed or unsafe"`
	Path        string   `json:"path" jsonschema:"the imagor path, without any signature or unsafe prefix"`
	URL         string   `json:"url,omitempty" jsonschema:"a request path ready to send to imagor, including its signature or unsafe prefix"`
	AbsoluteURL string   `json:"absolute_url,omitempty" jsonschema:"the full URL to fetch, present only when IMAGOR_BASE_URL is configured"`
	Signed      bool     `json:"signed" jsonschema:"whether the URL carries a signature"`
	Note        string   `json:"note,omitempty" jsonschema:"human-readable notes about the result"`
	Warnings    []string `json:"warnings,omitempty" jsonschema:"compositions that are valid but will not do what was intended"`
}

// toParams converts tool input into imagorpath.Params, appending the
// convenience filters last so they apply after the caller's filters.
func (in TransformInput) toParams() (imagorpath.Params, error) {
	if strings.TrimSpace(in.Image) == "" {
		return imagorpath.Params{}, fmt.Errorf("image is required")
	}
	if in.TrimBy != "" && in.TrimBy != imagorpath.TrimByTopLeft && in.TrimBy != imagorpath.TrimByBottomRight {
		return imagorpath.Params{}, fmt.Errorf("trim_by must be %q or %q", imagorpath.TrimByTopLeft, imagorpath.TrimByBottomRight)
	}
	if in.HAlign != "" && in.HAlign != imagorpath.HAlignLeft && in.HAlign != imagorpath.HAlignRight {
		return imagorpath.Params{}, fmt.Errorf("h_align must be %q or %q", imagorpath.HAlignLeft, imagorpath.HAlignRight)
	}
	if in.VAlign != "" && in.VAlign != imagorpath.VAlignTop && in.VAlign != imagorpath.VAlignBottom {
		return imagorpath.Params{}, fmt.Errorf("v_align must be %q or %q", imagorpath.VAlignTop, imagorpath.VAlignBottom)
	}
	if (in.FullFitIn || in.AdaptiveFitIn) && !in.FitIn {
		return imagorpath.Params{}, fmt.Errorf("full_fit_in and adaptive_fit_in require fit_in")
	}
	if in.TrimBy != "" && !in.Trim {
		return imagorpath.Params{}, fmt.Errorf("trim_by requires trim")
	}
	if in.Width < 0 || in.Height < 0 {
		return imagorpath.Params{}, fmt.Errorf("width and height must not be negative; use h_flip or v_flip to mirror")
	}

	filters := make(imagorpath.Filters, 0, len(in.Filters)+2)
	for _, f := range in.Filters {
		name := strings.TrimSpace(f.Name)
		if name == "" {
			return imagorpath.Params{}, fmt.Errorf("filter name must not be empty")
		}
		// imagor ignores unknown filter names and missing required arguments
		// silently, returning an unchanged image with a 200 response, so those
		// are refused here. Excess arguments only warn — see filterWarnings —
		// because imagor's tolerance differs per filter.
		if outcome, msg := filtercat.Check(name, f.Args); outcome == filtercat.Rejected {
			return imagorpath.Params{}, errors.New(msg)
		}
		filters = append(filters, imagorpath.Filter{Name: name, Args: textArgs(f.Text, sourceArgs(f.Source, f.Args))})
	}
	if in.MaxAgeSeconds > 0 {
		filters = append(filters, imagorpath.Filter{Name: "max_age", Args: strconv.Itoa(in.MaxAgeSeconds)})
	}
	if in.ExpireUnixMillis > 0 {
		filters = append(filters, imagorpath.Filter{Name: "expire", Args: strconv.FormatInt(in.ExpireUnixMillis, 10)})
	}

	return imagorpath.Params{
		Image:         in.Image,
		Width:         in.Width,
		Height:        in.Height,
		FitIn:         in.FitIn,
		FullFitIn:     in.FullFitIn,
		AdaptiveFitIn: in.AdaptiveFitIn,
		Stretch:       in.Stretch,
		Smart:         in.Smart,
		Trim:          in.Trim,
		TrimBy:        in.TrimBy,
		TrimTolerance: in.TrimTolerance,
		CropLeft:      in.CropLeft,
		CropTop:       in.CropTop,
		CropRight:     in.CropRight,
		CropBottom:    in.CropBottom,
		HAlign:        in.HAlign,
		VAlign:        in.VAlign,
		HFlip:         in.HFlip,
		VFlip:         in.VFlip,
		Meta:          in.Meta,
		Filters:       filters,
	}, nil
}

// filterWarnings reports filters that are known to imagor but carry more
// arguments than documented. Refusing these blocked filters that work, so they
// are surfaced instead: grayscale(1) still applies, while blur(1,2,3) is
// ignored outright, and only the caller can tell which case it is in.
func filterWarnings(p imagorpath.Params) []string {
	var warnings []string
	for _, f := range p.Filters {
		if outcome, msg := filtercat.Check(f.Name, f.Args); outcome == filtercat.Warned {
			warnings = append(warnings, msg)
		}
	}
	return warnings
}

// compositionWarnings reports compositions that are valid but will not do what
// was intended. Each is a case where imagor answers 200 and the caller cannot
// tell from the response that something was discarded.
func compositionWarnings(p imagorpath.Params) []string {
	var out []string
	if imagorpath.HasFilter(p, "quality") && !hasLossyFormat(p) {
		out = append(out,
			"quality() only affects lossy output formats. No lossy format() filter is present, so the "+
				"result depends on the client's Accept header and may be lossless PNG, in which case "+
				"quality has no effect. Add format(jpeg), format(webp) or format(avif) to be certain.")
	}
	// The two crops run in opposite orders. The filter crops after the resize,
	// so asking for dimensions alongside it throws the resize away and the
	// output is the crop's size. The path crop is the other order: it takes a
	// region of the source and the dimensions then scale what is left.
	if imagorpath.HasFilter(p, "crop") && (p.Width > 0 || p.Height > 0) {
		out = append(out,
			"crop() runs after the resize, so the requested dimensions are discarded and the output is "+
				"the crop's size. To crop the source and then scale it, use the crop path parameters "+
				"(crop_left, crop_top, crop_right, crop_bottom) instead of the filter.")
	}
	return append(out, positionWarnings(p.Filters)...)
}

// genericFaces maps a generic family to the face imagor should actually use. sans,
// serif and monospace resolve to DejaVu, which is the widest face the official image
// carries, so type sized against it overflows a box drawn for anything narrower. A
// caller who writes "sans" means "the default sans", not "DejaVu specifically".
var genericFaces = map[string]string{
	"sans":      "Open-Sans",
	"serif":     "Liberation-Serif",
	"monospace": "Liberation-Mono",
}

// resolveFont swaps a leading generic family for the face worth using, and reports it.
// The replacement is by name, so DejaVu remains reachable by asking for it directly.
func resolveFont(font string) (string, string) {
	head, rest := font, ""
	if i := strings.Index(font, "-"); i > 0 {
		head, rest = font[:i], font[i:]
	}
	face, ok := genericFaces[strings.ToLower(head)]
	if !ok {
		return font, ""
	}
	return face + rest, fmt.Sprintf(
		"font %q resolved to %q: the generic is DejaVu, the widest face imagor carries, "+
			"and type sized for it overflows a narrower box. Ask for DejaVu-Sans by name "+
			"to keep it.", font, face+rest)
}

// fontWarning reports a font argument whose size will be silently discarded. Pango
// reads the family first and the size last, so a number anywhere else is ignored:
// "sans-40-bold" renders at the default size. imagor answers 200 either way, so the
// only evidence is the type looking wrong.
//
// This reads the font the caller wrote, not the built path: by the time a font is in
// the path the fitter has appended a size, so the path form is not what was asked for.
func fontWarning(font string) string {
	parts := strings.Split(font, "-")
	if len(parts) < 3 {
		return ""
	}
	for i, tok := range parts[1 : len(parts)-1] {
		if _, err := strconv.Atoi(tok); err != nil {
			continue
		}
		// The fix is to move the size to the end, not to add a second one.
		moved := append(append([]string{}, parts[:i+1]...), parts[i+2:]...)
		moved = append(moved, tok)
		return fmt.Sprintf(
			"text() font %q names a size that is not last, so Pango ignores it and the text "+
				"renders at the default size. Write %s instead.",
			font, strings.Join(moved, "-"))
	}
	return ""
}

// positionWarnings reports a position written in a form imagor reads as zero.
//
// full and full-N belong to the dimension segment. The position grammar has no
// such form and parses one as 0, so the overlay lands at the origin and the
// request still returns 200 - the same shape of failure as a comma in text.
// Measured: y=full-220 on a 630px canvas put a title at the top.
func positionWarnings(filters imagorpath.Filters) []string {
	var out []string
	for _, f := range filters {
		switch f.Name {
		case "image", "text", "watermark", "label":
			// All four take the source or text first, then x, then y.
			args := imagorpath.SplitArgs(f.Args)
			for i, role := range []string{"x", "y"} {
				idx := i + 1
				if idx >= len(args) {
					continue
				}
				if looksLikeADimension(args[idx]) {
					out = append(out, fmt.Sprintf(
						"%s() has %q for its %s. full and full-N are dimension forms: the position grammar "+
							"has no such form and reads one as 0, so the overlay lands at the origin and the "+
							"request still returns 200. Use pixels, a percentage such as 20p, or an "+
							"alignment such as center.", f.Name, args[idx], role))
				}
			}
			// A nested path has its own filters, and the same trap.
			if inner, ok := nestedPathOf(f); ok {
				nested := imagorpath.Parse("/" + inner)
				out = append(out, positionWarnings(nested.Filters)...)
			}
		}
	}
	return out
}

// looksLikeADimension reports a value written in the dimension segment's forms.
func looksLikeADimension(v string) bool {
	v = strings.TrimSpace(v)
	if v == "full" || v == "f" {
		return true
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(v, "full"), "f")
	return strings.HasPrefix(rest, "-") && rest != v
}

// hasLossyFormat reports whether the filters ask for a format that loses detail,
// which is what makes quality() meaningful.
func hasLossyFormat(p imagorpath.Params) bool {
	for _, f := range p.Filters {
		if f.Name != "format" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(f.Args)) {
		case "jpeg", "jpg", "webp", "avif", "heif":
			return true
		}
	}
	return false
}

// build validates the composition and renders it in the configured mode.
func (c Config) build(in TransformInput) (BuildOutput, error) {
	mode, err := c.Mode()
	if err != nil {
		return BuildOutput{}, err
	}
	p, err := in.toParams()
	if err != nil {
		return BuildOutput{}, err
	}
	if err := pathgen.Validate(p); err != nil {
		return BuildOutput{}, err
	}

	out := BuildOutput{Mode: string(mode), Path: pathgen.Build(p)}
	out.Warnings = append(compositionWarnings(p), filterWarnings(p)...)
	switch mode {
	case ModeSigned:
		signer, err := c.Signer()
		if err != nil {
			return BuildOutput{}, err
		}
		out.URL = pathgen.BuildURL(p, signer)
		out.Signed = true
	case ModeUnsafe:
		// A nil signer makes imagorpath emit the "unsafe/" prefix, which is
		// exactly what a target imagor running with IMAGOR_UNSAFE expects.
		out.URL = pathgen.BuildURL(p, nil)
		out.Note = "unsafe mode: this URL carries no signature and only works against an imagor instance running with IMAGOR_UNSAFE=1"
	}
	if c.BaseURL != "" {
		out.AbsoluteURL = c.BaseURL + "/" + out.URL
	}
	return out, nil
}

// BuildURLTool returns a request URL ready to send to imagor.
func (c Config) BuildURLTool(_ context.Context, _ *mcp.CallToolRequest, in TransformInput) (*mcp.CallToolResult, BuildOutput, error) {
	out, err := c.build(in)
	if err != nil {
		return nil, BuildOutput{}, err
	}
	return nil, out, nil
}

// What the running imagor reported for one filter.
const (
	// OutcomeApplied means the filter ran and changed the image.
	OutcomeApplied = "applied"
	// OutcomeDeclined means the filter was recognised but left the image
	// unchanged, as blur() does without a sigma.
	OutcomeDeclined = "declined"
	// OutcomeNotReported means the server did not mention the filter, because
	// this processor does not handle it or the deployment disabled it.
	OutcomeNotReported = "not_reported"
	// OutcomeUnknown means the server did not report filter outcomes at all, so
	// nothing can be concluded. It is not a failure.
	OutcomeUnknown = "unknown"
)

// VerifyInput is a composition to check against a running imagor.
type VerifyInput struct {
	TransformInput
	// Preview attaches the rendered result so a caller that can see images can
	// compare it against its reference. Defaults to on, and can be turned off
	// when the caller only wants the report: an attached image is delivered to
	// the client whether or not the model wants it, so the choice belongs to
	// the caller rather than being made for it.
	Preview *bool `json:"preview,omitempty" jsonschema:"attach the rendered result as an image, so a composition that succeeded but came out wrong can be seen. Defaults to true"`
	// PreviewRegion attaches one part of the result at its native resolution
	// instead of the whole result scaled down. A scaled preview shows the layout
	// and loses small text, so this is how a caller checks a label or a price
	// without paying for the whole render.
	PreviewRegion string `json:"preview_region,omitempty" jsonschema:"attach part of the result at native resolution as x,y,w,h in result pixels, for checking text and fine detail that a scaled preview cannot show. Takes precedence over preview"`
	// A filter the server recognised and left the image unchanged is the failure
	// this tool exists to catch: the request succeeds, the image is wrong, and
	// nothing says so. It is an error by default, because a caller that checks
	// the result should not have to also read a note. Set this when a filter is
	// deliberately conditional - one that applies only to some inputs - so the
	// decline is expected rather than a fault.
	AllowDeclined *bool `json:"allow_declined,omitempty" jsonschema:"return the report instead of an error when a filter was recognised but left the image unchanged. Set this for a filter that is deliberately conditional; leave it unset to treat a decline as the failure it usually is"`
}

// FilterOutcome is what the server reported for one filter in the URL.
type FilterOutcome struct {
	Name    string `json:"name"`
	Args    string `json:"args,omitempty"`
	Outcome string `json:"outcome" jsonschema:"applied, declined, not_reported, or unknown when the server reports no filter outcomes"`
	// Filters carries outcomes for a path this filter loaded, nested the way the
	// server reports them. It is an any so the schema generator is not asked to
	// describe a recursive type; the value written is always []FilterOutcome.
	Filters any `json:"filters,omitempty" jsonschema:"outcomes for the filters inside a path this filter loaded, nested as the server reports them"`
}

// RequiresImagor is the oldest imagor release this server works against end to
// end. Saying which release is needed is more useful than reporting an absent
// field or a puzzling 400: a caller can act on a version.
//
//	1.9.8   metadata reports what it did with each filter, which verify_imagor_url
//	        reads. A server that does not report it cannot be verified against.
//	1.9.10  a nested path source (path:<imagor path>) is resolved in-process. On
//	        an older server "path:" is not syntax, so the whole token is read as
//	        the image name and the request fails with 400 invalid.
//
// 1.9.10 is the base because it is the release that made a composition safe to
// preview without the server being able to reach itself.
//
// This is a contract, not a derived value, so it moves by hand when the behaviour
// moves. It is not the catalogue version, which describes the path grammar and is
// generated from a released imagor.
const RequiresImagor = "1.9.10"

// VerifyOutput is what the running imagor did with each filter, alongside both
// URLs for the composition: the image itself, and the metadata that was read.
//
// The two carry different signatures, because a signature covers the whole path
// including the meta segment, so neither URL can be derived from the other.
type VerifyOutput struct {
	BuildOutput
	MetaURL         string `json:"meta_url,omitempty" jsonschema:"the request path for the metadata that was read; its signature differs from the image URL because the signature covers the whole path"`
	MetaAbsoluteURL string `json:"meta_absolute_url,omitempty" jsonschema:"the full metadata URL that was fetched, present only when IMAGOR_BASE_URL is configured"`
	// Verified is false when the server did not report filter outcomes, which
	// leaves the result unknown rather than wrong: an imagor older than the
	// release that reports them, or a response replayed from result storage.
	Verified bool            `json:"verified"`
	Status   int             `json:"status" jsonschema:"the HTTP status the metadata request returned"`
	Results  []FilterOutcome `json:"results,omitempty"`
	// Width and Height are the dimensions the server reported for the result.
	// Without them a caller can check that the filters ran but not that the
	// image came out the size it intended, which is what a composition mistake
	// usually changes.
	Width  int `json:"width,omitempty" jsonschema:"the result width the server reported, 0 when it reported none"`
	Height int `json:"height,omitempty" jsonschema:"the result height the server reported, 0 when it reported none"`
	// Server records which of the fields this tool depends on the server
	// actually returned. Without it a caller cannot tell a server that does not
	// report something from a server that had nothing to report, since an
	// absent field and an empty one read the same.
	Server ServerInfo `json:"server"`
}

// ServerInfo is what the server's metadata said about itself, and which of the
// fields this tool needs were there at all.
type ServerInfo struct {
	// ReportsFilters is whether the server returned the filter report at all.
	// The field is always present from the release that reports outcomes, empty
	// when there is nothing to report, so its absence means the server predates
	// it and cannot be verified against.
	ReportsFilters bool `json:"reports_filters" jsonschema:"whether the server reported what it did with each filter; the field is present from the release that reports outcomes, so absence means the server predates it"`
	// CatalogueVersion is the release this server's filter reference describes,
	// so a caller can see the two side by side.
	CatalogueVersion string `json:"catalogue_version" jsonschema:"the imagor release this server's filter reference was generated from"`
	// RequiresImagor is the oldest release this server works against end to end.
	// A server older than this is refused rather than half-supported.
	RequiresImagor string `json:"requires_imagor" jsonschema:"the oldest imagor release this server works against end to end"`
	// NestedPath is whether the server understands a nested path source, which
	// needs imagor 1.9.10. False means it predates the syntax: a path using one
	// is rejected while parsing, so this is reported rather than inferred.
	NestedPath bool `json:"nested_path" jsonschema:"whether the server understands a nested path source (path:<imagor path>), which needs imagor 1.9.10 or later"`
	// ProbeNote is why NestedPath is false when the probe could not reach a
	// verdict. Absent when the probe answered, including when it answered no.
	ProbeNote string `json:"probe_note,omitempty" jsonschema:"why the nested path probe was inconclusive, when it was; absent when the probe answered"`
	// Compatible is whether the server reported what this tool needs. False
	// means nothing could be verified, which is not the same as a failure.
	Compatible bool `json:"compatible" jsonschema:"whether the server reported the fields this tool needs"`
}

// reportedFilter mirrors the filter report a server returns in a /meta response.
type reportedFilter struct {
	Name      string           `json:"name"`
	Processed bool             `json:"processed"`
	Filters   []reportedFilter `json:"filters"`
}

// VerifyURLTool fetches the metadata for a composition from a running imagor and
// reports what it did with each filter, so a caller can tell a filter that
// applied from one the server ignored.
//
// The report is a field on /meta, so this needs IMAGOR_BASE_URL and a way to
// sign the request: IMAGOR_SECRET, or IMAGOR_UNSAFE for an unsafe instance.
// A server that does not report outcomes is not a failure: every outcome comes
// back as unknown, and Verified is false.
func (c Config) VerifyURLTool(ctx context.Context, _ *mcp.CallToolRequest, in VerifyInput) (*mcp.CallToolResult, VerifyOutput, error) {
	if c.BaseURL == "" {
		return nil, VerifyOutput{}, fmt.Errorf(
			"verification fetches the result from a running imagor, so IMAGOR_BASE_URL must be set to its origin")
	}
	// Probe once for the nested path source, and use the one answer twice:
	// refuse an older server before the composition is built, and report what
	// the probe found. Every path this server emits may carry a nested source,
	// and a server that predates the syntax rejects one while parsing, as 400
	// invalid naming no part of the path — which reads as a bad composition
	// rather than an old server. Only a definite no refuses; see
	// RequireNestedPath for why an inconclusive probe does not.
	support, probeErr := c.probeNestedPath(ctx)
	if support == nestedNo {
		return nil, VerifyOutput{}, c.nestedPathRefusal()
	}
	mode, err := c.Mode()
	if err != nil {
		return nil, VerifyOutput{}, err
	}
	p, err := in.toParams()
	if err != nil {
		return nil, VerifyOutput{}, err
	}
	if err := pathgen.Validate(p); err != nil {
		return nil, VerifyOutput{}, err
	}
	var signer imagorpath.Signer
	if mode == ModeSigned {
		if signer, err = c.Signer(); err != nil {
			return nil, VerifyOutput{}, err
		}
	}
	// The report is part of the metadata response, and a signature covers the
	// whole path, so the metadata request needs its own signature with meta in
	// the path. It cannot be derived from the image URL.
	//
	// The image URL must be built from params with Meta OFF, whatever the caller
	// asked for. toParams carries the caller's Meta through, so building the image
	// URL from p directly produced a URL that asked the server for metadata - and
	// the preview, which loads that URL as an image, got a JSON document and a 500.
	imageParams := p
	imageParams.Meta = false
	imageURL := pathgen.BuildURL(imageParams, signer)
	metaParams := p
	metaParams.Meta = true
	metaURL := pathgen.BuildURL(metaParams, signer)

	out := VerifyOutput{
		BuildOutput: BuildOutput{
			Mode:   string(mode),
			Path:   pathgen.Build(p),
			URL:    imageURL,
			Signed: mode == ModeSigned,
		},
		MetaURL: metaURL,
	}
	checked := c.BaseURL + "/" + metaURL
	out.AbsoluteURL = c.BaseURL + "/" + imageURL
	out.MetaAbsoluteURL = checked
	// NestedPath reports what the probe actually found. It is false both for a
	// server that answered no and for one the probe could not reach a verdict
	// on, which is why the note below carries the reason when there is one.
	out.Server = ServerInfo{
		CatalogueVersion: filtercat.ImagorVersion,
		RequiresImagor:   RequiresImagor,
		NestedPath:       support == nestedYes,
	}
	if support != nestedYes && probeErr != nil {
		out.Server.ProbeNote = probeErr.Error()
	}

	reqCtx, cancel := context.WithTimeout(ctx, verifyTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, checked, nil)
	if err != nil {
		return nil, VerifyOutput{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, VerifyOutput{}, fmt.Errorf("fetching %s: %w", checked, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMetaBytes))
	if err != nil {
		return nil, VerifyOutput{}, fmt.Errorf("reading the response from %s: %w", checked, err)
	}

	out.Status = resp.StatusCode
	if resp.StatusCode != http.StatusOK {
		out.Note = fmt.Sprintf("the server returned %d, so nothing could be verified; the body was %s",
			resp.StatusCode, firstLine(body))
		// A signature covers the whole path, so this is what a signer disagreement
		// looks like from outside: same secret, different algorithm or length. The
		// settings are not visible in the response, so name them.
		if resp.StatusCode == http.StatusForbidden &&
			strings.Contains(strings.ToLower(string(body)), "signature") {
			out.Note += ". " + c.signingMismatchAdvice()
		}
		out.Results = outcomes(p.Filters, nil, OutcomeUnknown)
		return nil, out, nil
	}

	// A missing field and an empty one mean different things: absent means this
	// server does not report outcomes, empty means it reported none.
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, VerifyOutput{}, fmt.Errorf("the response from %s was not JSON: %w", checked, err)
	}
	raw, present := envelope["filters"]
	// Which of the fields this tool needs were there at all. An absent field and
	// an empty one read the same, so a caller cannot tell a server that does not
	// report something from a server that had nothing to report.
	out.Server.ReportsFilters = present
	out.Server.Compatible = present
	if v, ok := envelope["width"]; ok {
		_ = json.Unmarshal(v, &out.Width)
	}
	if v, ok := envelope["height"]; ok {
		_ = json.Unmarshal(v, &out.Height)
	}
	if !present {
		out.Note = fmt.Sprintf(
			"the metadata body had no %q field, so nothing can be verified against it. A server that "+
				"reports filter outcomes always sends that field, an empty list when nothing ran, and "+
				"that difference is how a reporting server is told from one that predates the field. "+
				"That needs imagor %s or later, so check the release behind %s if you expected a report. "+
				"This is not a failure: the composition may be perfectly valid, it simply cannot be "+
				"checked here.", "filters", RequiresImagor, checked)
		out.Results = outcomes(p.Filters, nil, OutcomeUnknown)
		return nil, out, nil
	}
	var reported []reportedFilter
	if err := json.Unmarshal(raw, &reported); err != nil {
		return nil, VerifyOutput{}, fmt.Errorf("the filters field from %s was not a list: %w", checked, err)
	}
	out.Verified = true
	out.Results = outcomes(p.Filters, reported, OutcomeNotReported)
	out.Note = summarizeOutcomes(out.Results)
	// The server recognised the filter and left the image unchanged. This is the
	// failure the whole tool exists to catch: a 200, a plausible image, and
	// nothing to say the filter did not run. Reporting it in a note leaves the
	// caller to notice, and a caller checking for an error would not - so it is
	// an error, and the opt-out is for a filter that is genuinely conditional.
	if in.AllowDeclined == nil || !*in.AllowDeclined {
		if declined := declinedNames(out.Results); len(declined) > 0 {
			return nil, out, fmt.Errorf(
				"%s recognised but left the image unchanged: %s. imagor applied every other filter "+
					"and returned 200, so the result is a valid image and not the one this path "+
					"describes. Either the filter's arguments do not apply to this source - a blur "+
					"with no effect, a trim on an image with nothing to trim - or it needs an "+
					"argument it did not get. Set allow_declined when the filter is deliberately "+
					"conditional and the decline is expected",
				plural(len(declined), "filter"), strings.Join(declined, ", "))
		}
	}
	// The report says what the server did; the preview says what it looks like.
	// A caller that can see images can compare the result against its reference
	// and fix a composition that succeeded but came out wrong, which the report
	// alone cannot reveal. It is asked for through a scaled path, because images
	// cost tokens by area and the full render costs several times the preview for
	// no more answer to "is this panel in the right place".
	if in.Preview != nil && !*in.Preview {
		return nil, out, nil
	}
	previewParams := p
	previewParams.Image = out.AbsoluteURL
	previewParams.Path = ""
	// The preview asks for the composed image, so it must not carry Meta either.
	// previewParams starts as a copy of p, and toParams carries the caller's Meta
	// through - so a caller that asked for metadata got a preview request with
	// meta in its own path, the server answered with the metadata document, and
	// the preview reported that the result was not an image.
	previewParams.Meta = false
	// The composition is the source of the preview, so the preview's own filter
	// list must not repeat the composition's. Carrying p.Filters here applied the
	// overlays and text a second time, on top of the already-composed source, and
	// the nested path's arguments then read as filter arguments of the outer path.
	previewParams.Filters = nil
	// preview() is added to the preview request and never to the composition. It
	// tells imagor this is a look rather than the result: the render is not
	// written to result storage, and the request opts into the in-memory cache so
	// a second look at the same composition does not decode it again. The caller's
	// own URL is untouched, so a preview never becomes the artifact.
	//
	// It is not exposed as a parameter. Whether a deployment caches is its own
	// configuration, and a caller composing a banner has no business deciding it.
	if region := in.PreviewRegion; region != "" {
		// A region is attached at native resolution, so there is no scaling and
		// no fit-in: the caller asked for a part of the result in order to read
		// it, and scaling it back down would undo the point of asking.
		sel, err := regionFilter(region, out.Width, out.Height)
		if err != nil {
			return nil, out, nil
		}
		// Only the crop: the composition already carries its own filters.
		previewParams.Filters = imagorpath.Filters{sel}
	} else {
		// The whole result is attached scaled down, because images cost tokens by
		// area and a layout question is answered at preview size. The composition
		// is passed in as a nested source rather than having its dimensions
		// replaced, so the caller is shown the layout the report describes.
		previewParams.FitIn = true
		previewParams.Width, previewParams.Height = previewMaxEdge, previewMaxEdge
	}
	previewParams.Filters = append(append(imagorpath.Filters{}, previewParams.Filters...),
		imagorpath.Filter{Name: "preview"})
	data, mime, previewNote := c.previewImage(ctx, c.BaseURL+"/"+pathgen.BuildURL(previewParams, signer))
	// The caller is told how much the preview was scaled, so it knows which
	// coordinate system it is looking at when it asks for a region.
	scale := 0
	if previewParams.FitIn && out.Width > 0 && out.Height > 0 {
		scale = previewMaxEdge * 100 / max(out.Width, out.Height)
	}
	return withPreview(out, data, mime, previewNote, scale), out, nil
}

// regionFilter turns "x,y,w,h" in result pixels into the crop a caller asked for.
//
// The crop is expressed in percentages because that is what the filter takes,
// and the result's dimensions are what the report already carried. A region that
// does not parse, or that falls outside the result, yields no preview rather than
// a crop of the wrong place: a wrong region is worse than none, because a caller
// would read it as evidence about the composition.
func regionFilter(region string, width, height int) (imagorpath.Filter, error) {
	if width <= 0 || height <= 0 {
		return imagorpath.Filter{}, fmt.Errorf("the result has no dimensions to crop against")
	}
	parts := strings.Split(region, ",")
	if len(parts) != 4 {
		return imagorpath.Filter{}, fmt.Errorf("a region is x,y,w,h; got %q", region)
	}
	var v [4]float64
	for i, s := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return imagorpath.Filter{}, fmt.Errorf("a region is x,y,w,h; %q is not a number", s)
		}
		v[i] = f
	}
	x, y, w, h := v[0], v[1], v[2], v[3]
	if w <= 0 || h <= 0 || x < 0 || y < 0 || x+w > float64(width) || y+h > float64(height) {
		return imagorpath.Filter{}, fmt.Errorf(
			"the region %s is not inside the %dx%d result", region, width, height)
	}
	// crop() takes left, top, width and height in pixels, not right and bottom
	// and not percentages. Passing the far edge as the third and fourth argument
	// asks for a crop that size, and a percentage form is not understood at all -
	// both silently return the whole image rather than failing.
	return imagorpath.Filter{
		Name: "crop",
		Args: strings.Join([]string{
			strconv.Itoa(int(x)),
			strconv.Itoa(int(y)),
			strconv.Itoa(int(w)),
			strconv.Itoa(int(h)),
		}, ","),
	}, nil
}

// outcomes walks the requested filters against the reported ones. The report
// holds one entry per filter the processor handled, in URL order, so entries are
// consumed in step: a name the report never mentions is absent rather than
// failed, since another processor in the chain may handle it.
func outcomes(requested []imagorpath.Filter, reported []reportedFilter, missing string) []FilterOutcome {
	var out []FilterOutcome
	next := 0
	for _, f := range requested {
		o := FilterOutcome{Name: f.Name, Args: f.Args, Outcome: missing}
		if next < len(reported) && reported[next].Name == f.Name {
			r := reported[next]
			next++
			o.Outcome = OutcomeDeclined
			if r.Processed {
				o.Outcome = OutcomeApplied
			}
			if nested, ok := nestedPathOf(f); ok && len(r.Filters) > 0 {
				o.Filters = outcomes(imagorpath.Parse(nested).Filters, r.Filters, OutcomeNotReported)
			}
		}
		out = append(out, o)
	}
	return out
}

// declinedNames lists the filters the server recognised and left the image
// unchanged, walking nested outcomes so a decline inside a loaded path counts
// too. A filter inside image() that did nothing is the same failure as one at
// the top level, and harder to see.
func declinedNames(results []FilterOutcome) []string {
	var out []string
	var walk func([]FilterOutcome)
	walk = func(rs []FilterOutcome) {
		for _, r := range rs {
			if r.Outcome == OutcomeDeclined {
				out = append(out, r.Name)
			}
			if nested, ok := r.Filters.([]FilterOutcome); ok {
				walk(nested)
			}
		}
	}
	walk(results)
	return out
}

// plural picks the singular or plural form for a count, so a message about one
// filter does not read as if it were about several.
func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// summarizeOutcomes states what was found in one line, so a caller that reads
// only the note still gets the answer. A decline is named rather than counted:
// "1 recognised but left the image unchanged" does not say which one, and the
// point of the report is to identify the filter that did nothing.
func summarizeOutcomes(results []FilterOutcome) string {
	var applied, missing int
	var declined []string
	var walk func([]FilterOutcome)
	walk = func(rs []FilterOutcome) {
		for _, r := range rs {
			switch r.Outcome {
			case OutcomeApplied:
				applied++
			case OutcomeDeclined:
				declined = append(declined, r.Name)
			default:
				missing++
			}
			if nested, ok := r.Filters.([]FilterOutcome); ok {
				walk(nested)
			}
		}
	}
	walk(results)
	switch {
	case applied == 0 && len(declined) == 0 && missing == 0:
		return "no filters were requested, so there is nothing to verify"
	case missing == 0 && len(declined) == 0:
		return fmt.Sprintf("all %d filter(s) applied", applied)
	case missing == 0:
		return fmt.Sprintf("%d applied, %s recognised but left the image unchanged: %s",
			applied, plural(len(declined), "filter"), strings.Join(declined, ", "))
	default:
		return fmt.Sprintf("%d applied, %d declined (%s), %d not reported by this processor",
			applied, len(declined), strings.Join(declined, ", "), missing)
	}
}

func firstLine(body []byte) string {
	s := strings.TrimSpace(string(body))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

const (
	verifyTimeout = 30 * time.Second
	maxMetaBytes  = 4 << 20
)

// BuildPathTool returns the bare imagor path, with no signature or prefix.
func (c Config) BuildPathTool(_ context.Context, _ *mcp.CallToolRequest, in TransformInput) (*mcp.CallToolResult, BuildOutput, error) {
	out, err := c.build(in)
	if err != nil {
		return nil, BuildOutput{}, err
	}
	out.URL = ""
	out.AbsoluteURL = ""
	out.Signed = false
	out.Note = "bare path, with no signature and no unsafe prefix"
	return nil, out, nil
}

// ValidateInput is a path to check.
type ValidateInput struct {
	Path string `json:"path" jsonschema:"an imagor path to validate, without a leading slash and without any signature prefix"`
}

// ValidateOutput reports whether the path is well formed, including the paths
// that image() loads, which imagor parses and processes as images in their own
// right.
type ValidateOutput struct {
	Valid    bool     `json:"valid" jsonschema:"whether the path is well formed and would not be silently ignored"`
	Path     string   `json:"path" jsonschema:"the path as parsed"`
	Rebuilt  string   `json:"rebuilt" jsonschema:"the path regenerated from the parsed parameters"`
	Stable   bool     `json:"stable" jsonschema:"whether regenerating the parsed path reproduces the input exactly"`
	Problems []string `json:"problems,omitempty" jsonschema:"reasons imagor would ignore this path or fail to load it"`
	Warnings []string `json:"warnings,omitempty" jsonschema:"things imagor accepts but may not act on"`
	Summary  string   `json:"summary" jsonschema:"a short description of the parsed transformation"`
}

// maxNestDepth bounds recursion: an image() argument is an imagor path, which
// may itself contain image() filters.
const maxNestDepth = 8

// ValidateTool parses a path and reports whether it is well formed.
//
// Everything reported here comes from the path alone, because a log is not a
// reliable signal in production: imagor's own parser, the filter catalogue, and
// the grammar of the paths that filters load.
func (c Config) ValidateTool(_ context.Context, _ *mcp.CallToolRequest, in ValidateInput) (*mcp.CallToolResult, ValidateOutput, error) {
	path := strings.TrimPrefix(strings.TrimSpace(in.Path), "/")
	if path == "" {
		return nil, ValidateOutput{}, fmt.Errorf("path is required")
	}
	var notes []string
	// unsafe/ selects the mode at the server rather than describing an
	// operation, so it is not part of the path and does not survive a round trip.
	if rest, ok := strings.CutPrefix(path, "unsafe/"); ok {
		path = rest
		notes = append(notes, `the leading "unsafe/" selects the server mode and is not part of the path; validated without it`)
	}
	p := imagorpath.Parse(path)
	rebuilt := pathgen.Build(p)

	out := ValidateOutput{
		Valid:   true,
		Path:    path,
		Rebuilt: rebuilt,
		Stable:  rebuilt == path,
		Summary: summarize(p),
	}
	if !out.Stable {
		out.Problems = append(out.Problems, fmt.Sprintf("regenerating the parsed path produced %q, which differs from the input", rebuilt))
	}
	if err := pathgen.Validate(p); err != nil {
		out.Problems = append(out.Problems, err.Error())
	}
	problems, warnings := checkPath(p, 0)
	out.Problems = append(out.Problems, problems...)
	out.Warnings = append(notes, warnings...)
	out.Valid = len(out.Problems) == 0
	return nil, out, nil
}

// checkPath reports what imagor would ignore or fail to load in a parsed path,
// and recurses into the paths that image() loads. Those are parsed with the same
// grammar as the outer path and processed as images, so a fault inside one fails
// the whole request.
func checkPath(p imagorpath.Params, depth int) (problems, warnings []string) {
	if seg, _, ok := strings.Cut(p.Image, "/"); ok && isParamSegment(seg) {
		problems = append(problems, fmt.Sprintf(
			"the image is read as %q, whose first segment %q is a parameter: parameters come before "+
				"filters, so imagor would try to load this as a URL and fail", p.Image, seg))
	}
	for _, f := range p.Filters {
		switch outcome, msg := filtercat.Check(f.Name, f.Args); outcome {
		case filtercat.Rejected:
			problems = append(problems, msg)
		case filtercat.Warned:
			warnings = append(warnings, msg)
		}

		nested, ok := nestedPathOf(f)
		if !ok {
			// watermark() loads its argument as a plain image URL, so anything
			// written as a filter inside it is never run: imagor would try to
			// fetch the whole string as a URL.
			if f.Name == "watermark" {
				if arg, ok := firstArg(f); ok {
					if nw := imagorpath.Parse(arg); len(nw.Filters) > 0 {
						problems = append(problems, fmt.Sprintf(
							"watermark loads %q as an image URL and runs no filters, so the %d filter(s) "+
								"written inside it never run; image() takes a full path", arg, len(nw.Filters)))
					}
				}
			}
			continue
		}
		if depth >= maxNestDepth {
			problems = append(problems, fmt.Sprintf("filter %q nests deeper than %d levels", f.Name, maxNestDepth))
			continue
		}
		np := imagorpath.Parse(nested)
		if np.Image == "" {
			problems = append(problems, fmt.Sprintf("filter %q loads %q, which names no image", f.Name, nested))
			continue
		}
		subProblems, subWarnings := checkPath(np, depth+1)
		for _, s := range subProblems {
			problems = append(problems, fmt.Sprintf("%s loads %q: %s", f.Name, nested, s))
		}
		for _, s := range subWarnings {
			warnings = append(warnings, fmt.Sprintf("%s loads %q: %s", f.Name, nested, s))
		}
	}
	return problems, warnings
}

// nestedPathOf returns the imagor path a filter loads, if it loads one. image()
// is the only one: it parses its argument as a path and processes it as an image
// in its own right. watermark() loads its argument as a plain image URL.
func nestedPathOf(f imagorpath.Filter) (string, bool) {
	if f.Name != "image" {
		return "", false
	}
	return firstArg(f)
}

// firstArg returns a filter's first argument with query escaping and b64:
// encoding resolved, the way imagor resolves them.
func firstArg(f imagorpath.Filter) (string, bool) {
	args := imagorpath.SplitArgs(f.Args)
	if len(args) == 0 {
		return "", false
	}
	arg := args[0]
	if unescaped, err := url.QueryUnescape(arg); err == nil {
		arg = unescaped
	}
	if encoded, ok := strings.CutPrefix(arg, "b64:"); ok {
		// imagor keeps the literal value when decoding fails.
		if raw, err := base64.RawURLEncoding.DecodeString(encoded); err == nil {
			arg = string(raw)
		}
	}
	if arg == "" || arg == "none" {
		return "", false
	}
	return arg, true
}

// paramSegments are the path segments imagor reads as parameters rather than as
// the start of an image reference.
var paramSegments = map[string]bool{
	"fit-in": true, "stretch": true, "full-fit-in": true, "adaptive-fit-in": true,
	"smart": true, "meta": true, "trim": true,
	"left": true, "right": true, "top": true, "bottom": true, "center": true,
	"top-left": true, "bottom-right": true,
}

// isParamSegment reports whether a segment is a parameter. A path whose image
// begins with one had its parameters written after its filters.
func isParamSegment(seg string) bool {
	return paramSegments[seg] || isDimSegment(seg) || isCropSegment(seg)
}

// isCropSegment reports whether a segment is a crop pair, leftxtop:rightxbottom,
// as in 201x65:289x99 or 0x0:0.5x1.
func isCropSegment(seg string) bool {
	top, bottom, ok := strings.Cut(seg, ":")
	if !ok {
		return false
	}
	return isDimSegment(top) && isDimSegment(bottom)
}

// isDimSegment reports whether a segment is a dimension or alignment pair such
// as 300x300, 300x, x300 or 0.5x0.5.
func isDimSegment(seg string) bool {
	left, right, ok := strings.Cut(seg, "x")
	if !ok || (left == "" && right == "") {
		return false
	}
	return isNumber(left) && isNumber(right)
}

func isNumber(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && r != '.' {
			return false
		}
	}
	return true
}

func summarize(p imagorpath.Params) string {
	var b strings.Builder
	if p.Meta {
		b.WriteString("metadata-only request; ")
	}
	switch {
	case p.Width > 0 && p.Height > 0:
		fmt.Fprintf(&b, "resize to %dx%d", p.Width, p.Height)
	case p.Width > 0:
		fmt.Fprintf(&b, "resize to width %d", p.Width)
	case p.Height > 0:
		fmt.Fprintf(&b, "resize to height %d", p.Height)
	default:
		b.WriteString("no resize")
	}
	for _, mode := range []struct {
		on   bool
		text string
	}{
		{p.FitIn, " fit-in"},
		{p.FullFitIn, " full-fit-in"},
		{p.AdaptiveFitIn, " adaptive-fit-in"},
		{p.Stretch, " stretch"},
		{p.Smart, " smart-crop"},
		{p.Trim, " trim"},
		{p.HFlip, " h-flip"},
		{p.VFlip, " v-flip"},
	} {
		if mode.on {
			b.WriteString(mode.text)
		}
	}
	if imagorpath.HasCrop(p) {
		fmt.Fprintf(&b, " crop(%v,%v,%v,%v)", p.CropLeft, p.CropTop, p.CropRight, p.CropBottom)
	}
	if len(p.Filters) > 0 {
		names := make([]string, 0, len(p.Filters))
		for _, f := range p.Filters {
			names = append(names, f.Name)
		}
		fmt.Fprintf(&b, "; filters: %s", strings.Join(names, ", "))
	}
	fmt.Fprintf(&b, "; source: %s", p.Image)
	return b.String()
}

// ParamDoc documents one path parameter, derived by reflection.
type ParamDoc struct {
	Name   string `json:"name"`
	GoType string `json:"go_type"`
}

// DescribeOutput describes the imagor path grammar this server targets.
type DescribeOutput struct {
	Mode          string     `json:"mode" jsonschema:"signed or unsafe"`
	ImagorVersion string     `json:"imagor_version" jsonschema:"the imagor release the filter catalogue was generated from"`
	FilterCount   int        `json:"filter_count" jsonschema:"number of filters accepted by the tools"`
	SignerType    string     `json:"signer_type" jsonschema:"signature algorithm; must match IMAGOR_SIGNER_TYPE on the imagor this server points at"`
	Truncate      int        `json:"truncate" jsonschema:"signature length in characters; must match IMAGOR_SIGNER_TRUNCATE on that imagor"`
	Signed        bool       `json:"signing_enabled"`
	SetupNote     string     `json:"setup_note" jsonschema:"what has to agree between this server and the imagor it points at, and what a disagreement looks like"`
	BaseURL       string     `json:"base_url,omitempty"`
	PathParams    []ParamDoc `json:"path_params" jsonschema:"path grammar parameters, reflected from the imagor path type"`
	HAlignValues  []string   `json:"h_align_values"`
	VAlignValues  []string   `json:"v_align_values"`
	TrimByValues  []string   `json:"trim_by_values"`
	Limitations   []string   `json:"limitations"`
}

// SkillOutput carries the composition skill.
type SkillOutput struct {
	Skill string `json:"skill"`
}

// SkillTool returns the composition skill. The server also serves it as a
// resource, but a host that reads only tools/list never sees a resource, and
// that host is the one that most needs this.
func (c Config) SkillTool(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, SkillOutput, error) {
	return nil, SkillOutput{Skill: imagormcp.Skill}, nil
}

// DescribeTool reports the supported grammar and the server's signing mode.
//
// The parameter list is reflected from imagor's own Params type, so it cannot
// drift from the implementation. The filter catalogue is NOT included, because
// imagor does not currently expose its filter registry in a machine-readable
// form and a hand-maintained copy would silently rot.
func (c Config) DescribeTool(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, DescribeOutput, error) {
	mode, err := c.Mode()
	if err != nil {
		return nil, DescribeOutput{}, err
	}
	out := DescribeOutput{
		Mode:          string(mode),
		ImagorVersion: filtercat.ImagorVersion,
		FilterCount:   len(filtercat.Names()),
		SignerType:    c.SignerType,
		Truncate:      c.Truncate,
		Signed:        c.Secret != "",
		BaseURL:       c.BaseURL,
		PathParams:    reflectParams(),
		HAlignValues:  []string{imagorpath.HAlignLeft, imagorpath.HAlignRight},
		VAlignValues:  []string{imagorpath.VAlignTop, imagorpath.VAlignBottom},
		TrimByValues:  []string{imagorpath.TrimByTopLeft, imagorpath.TrimByBottomRight},
		Limitations: []string{
			"this server does not upload, store or render images; it only constructs paths and signatures",
			"filter names and argument counts are validated against imagor " + filtercat.ImagorVersion + "; imagor itself ignores unrecognised filters silently",
		},
	}
	out.SetupNote = fmt.Sprintf(
		"This server is in %s mode, and the imagor it points at has to be set up the same way. "+
			"Three settings must agree: which of IMAGOR_UNSAFE or IMAGOR_SECRET authorises a "+
			"request, IMAGOR_SIGNER_TYPE (%s here) and IMAGOR_SIGNER_TRUNCATE (%d here). A "+
			"mismatch is not reported at startup: it returns 403 with 'url signature mismatch'. "+
			"A signature covers the whole path, so any difference looks the same, and neither "+
			"side can read the other's secret - make a request to confirm they agree.",
		mode, c.signerName(), c.Truncate)
	if out.SignerType == "" {
		out.SignerType = "sha1"
	}
	switch mode {
	case ModeUnsafe:
		out.Limitations = append(out.Limitations,
			"UNSAFE MODE: this server emits unsigned URLs and only works against an imagor instance running with IMAGOR_UNSAFE=1. Never use it in production.")
	}
	return nil, out, nil
}

// ListFiltersInput selects which filters to return.
type ListFiltersInput struct {
	NameContains string `json:"name_contains,omitempty" jsonschema:"optional case-insensitive substring to filter names by, for example \"strip\" to find the metadata filters"`
}

// FilterDoc documents one filter.
type FilterDoc struct {
	Name       string `json:"name"`
	Signature  string `json:"signature" jsonschema:"imagor's documented signature, for example \\\"image(imagorpath, x, y[, alpha[, blend_mode]])\\\"; the argument count when undocumented"`
	MinArgs    int    `json:"min_args"`
	MaxArgs    int    `json:"max_args" jsonschema:"-1 means the signature is undocumented and the argument count is not validated"`
	Documented bool   `json:"documented" jsonschema:"false when the filter is registered in imagor but absent from its filter reference"`
	// Description is imagor's own summary of the filter, carried from its
	// reference so the two cannot drift apart.
	Description string `json:"description,omitempty"`
	// Arguments are the reference's descriptions of each argument, in signature
	// order, which is where the accepted forms are written down.
	Arguments []FilterArgument `json:"arguments,omitempty"`
}

// FilterArgument documents one argument of a filter.
type FilterArgument struct {
	// Name is the argument as it appears in the signature.
	Name string `json:"name"`
	// Lead is what the argument means.
	Lead string `json:"lead"`
	// Forms are the accepted forms, kept as a list because they are
	// alternatives: a caller needs to pick one, not read them as one sentence.
	Forms []string `json:"forms,omitempty" jsonschema:"the accepted forms, one per line of imagor's reference"`
}

// filterArguments carries the reference's argument descriptions into the tool
// output, keeping each accepted form separate.
func filterArguments(args []filtercat.Argument) []FilterArgument {
	if len(args) == 0 {
		return nil
	}
	out := make([]FilterArgument, 0, len(args))
	for _, a := range args {
		out = append(out, FilterArgument{Name: a.Name, Lead: a.Lead, Forms: a.Forms})
	}
	return out
}

// ListFiltersOutput is the filter catalogue.
type ListFiltersOutput struct {
	ImagorVersion string      `json:"imagor_version" jsonschema:"the imagor release the catalogue was generated from"`
	Count         int         `json:"count"`
	Filters       []FilterDoc `json:"filters"`
	Note          string      `json:"note,omitempty"`
}

// ListFiltersTool returns the filter catalogue.
//
// Argument counts come from imagor's own filter reference and are enforced by
// build_imagor_url, so an agent chooses from filters that exist rather than
// guessing names that imagor would ignore.
func (c Config) ListFiltersTool(_ context.Context, _ *mcp.CallToolRequest, in ListFiltersInput) (*mcp.CallToolResult, ListFiltersOutput, error) {
	needle := strings.ToLower(strings.TrimSpace(in.NameContains))
	out := ListFiltersOutput{ImagorVersion: filtercat.ImagorVersion}
	for _, name := range filtercat.Names() {
		if needle != "" && !strings.Contains(strings.ToLower(name), needle) {
			continue
		}
		// A filter a caller cannot read a signature for is worse than one it
		// cannot see: it invites a guess. superseded filters work and still parse,
		// so they stay available to validation and to paths that already use them,
		// but they are not offered here.
		if superseded(name) {
			continue
		}
		spec, ok := filtercat.Lookup(name)
		if !ok {
			continue
		}
		out.Filters = append(out.Filters, FilterDoc{
			Name:        name,
			Signature:   signatureFor(name, spec),
			MinArgs:     spec.MinArgs,
			MaxArgs:     spec.MaxArgs,
			Documented:  spec.Documented,
			Description: spec.Description,
			Arguments:   filterArguments(spec.Arguments),
		})
	}
	out.Count = len(out.Filters)
	if out.Count == 0 {
		out.Note = fmt.Sprintf("no filter name contains %q", in.NameContains)
	}
	return nil, out, nil
}

// superseded reports filters that work but should not be offered.
//
// label() is undocumented and text() accepts everything it does at the same
// argument positions 1-5, then adds blending, wrapping, alignment, spacing and
// dpi. Nothing label does is unavailable, and its signature is not in imagor's
// reference, so a caller that sees it listed can only guess at the arguments.
func superseded(name string) bool {
	return name == "label"
}

// signatureFor renders a filter's accepted argument count for humans.
func signatureFor(name string, spec filtercat.Spec) string {
	if spec.Signature != "" {
		return spec.Signature
	}
	switch {
	case !spec.Documented || spec.MaxArgs < 0:
		return name + "(argument count not validated)"
	case spec.MinArgs == 0 && spec.MaxArgs == 0:
		return name + "()"
	case spec.MinArgs == spec.MaxArgs:
		return fmt.Sprintf("%s(%d arg)", name, spec.MinArgs)
	default:
		return fmt.Sprintf("%s(%d to %d args)", name, spec.MinArgs, spec.MaxArgs)
	}
}

// reflectParams lists the imagor path grammar parameters from imagor's own type.
func reflectParams() []ParamDoc {
	t := reflect.TypeOf(imagorpath.Params{})
	docs := make([]ParamDoc, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := strings.Split(f.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		docs = append(docs, ParamDoc{Name: name, GoType: f.Type.String()})
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Name < docs[j].Name })
	return docs
}

// Register adds every tool to the server.
func Register(s *mcp.Server, cfg Config) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "build_imagor_url",
		Description: "Build a signed imagor image URL from typed transformation parameters. " +
			"Requires IMAGOR_SECRET to be configured. The composition is validated before the " +
			"URL is returned, so an invalid combination is refused rather than emitted. " +
			"Read get_imagor_skill before composing anything, and call list_imagor_filters for each " +
			"filter's signature, description and accepted argument forms, which are generated " +
			"from imagor's own filter reference. Build and verify in a loop: a request returns " +
			"200 whether or not a filter did anything, so call verify_imagor_url on the result " +
			"rather than assuming it worked.",
	}, cfg.BuildURLTool)

	mcp.AddTool(s, &mcp.Tool{
		Name: "verify_imagor_url",
		Description: "Fetch the result for a composition from the running imagor and report what it " +
			"did with each filter: applied, declined, or not reported, plus the dimensions the " +
			"server reported. Answers whether a composition actually does what it says, which the " +
			"build tools cannot know on their own. Takes the same typed parameters as " +
			"build_imagor_url, not a URL. Requires IMAGOR_BASE_URL. " +
			"A server that does not report filter outcomes returns unknown for every filter and " +
			"verified: false, which means the result is unknown rather than wrong. Compare the " +
			"reported dimensions and outcomes against what you intended, and fetch the result to " +
			"compare it against your reference: a composition can succeed and still be wrong. " +
			"The rendered result is attached as an image by default, so you can see what you built " +
			"rather than only reading what the server did with it - use it to check placement, " +
			"cropping and text against your reference, and iterate until it matches. The image is " +
			"scaled down, so it shows the layout and not fine detail. Pass preview: false to get " +
			"the report alone when you do not need to look.",
	}, cfg.VerifyURLTool)

	mcp.AddTool(s, &mcp.Tool{
		Name: "build_imagor_path",
		Description: "Build an unsigned imagor path from typed transformation parameters, for the " +
			"caller to sign with its own secret. Use this when the MCP server should not hold " +
			"the signing secret. Read get_imagor_skill before composing anything, and call " +
			"list_imagor_filters for each filter's signature, description and accepted " +
			"argument forms.",
	}, cfg.BuildPathTool)

	mcp.AddTool(s, &mcp.Tool{
		Name: "validate_imagor_path",
		Description: "Parse an imagor path and report whether it round-trips unchanged, plus a short " +
			"summary of the transformation it describes. Use this to check a path before serving it.",
	}, cfg.ValidateTool)

	mcp.AddTool(s, &mcp.Tool{
		Name: "list_imagor_filters",
		Description: "List the imagor filters this server accepts, with each filter's signature, " +
			"description and accepted argument forms, generated from imagor's own filter reference. " +
			"imagor ignores unrecognised filters and wrong argument counts silently — the request " +
			"returns 200 with the image unchanged — so use this to pick filters that exist instead " +
			"of guessing names.",
	}, cfg.ListFiltersTool)

	mcp.AddTool(s, &mcp.Tool{
		Name: "describe_imagor",
		Description: "Describe the imagor path grammar this server targets, including the parameters " +
			"reflected from imagor's own path type and the current signing mode.",
	}, cfg.DescribeTool)

	mcp.AddTool(s, &mcp.Tool{
		Name: "fit_text",
		Description: "Find the largest font size at which a string still fits a box. imagor wraps " +
			"text but never shrinks it, so a template that suits a short title silently overflows on " +
			"a longer one. This renders the text on a transparent canvas at candidate sizes, measures " +
			"the ink, and searches for the largest size that fits. Pass the returned font argument " +
			"straight to text(). Requires IMAGOR_BASE_URL.",
	}, cfg.FitTextTool)

	mcp.AddTool(s, &mcp.Tool{
		Name: "render_template",
		Description: "Render a layout at a given canvas size. Geometry is written the way imagor " +
			"writes it, so slots can be positioned and sized relative to the canvas with full and " +
			"full-N, and one template then serves several sizes. A text slot given a height is fitted " +
			"with fit_text against the box its geometry resolves to, because a font size is absolute " +
			"and would otherwise be relatively smaller on a larger canvas. Values fill {{placeholders}}. " +
			"Returns the path, the URL, and what each slot resolved to.",
	}, cfg.RenderTemplateTool)

	mcp.AddTool(s, &mcp.Tool{
		Name: "parse_template",
		Description: "Read an imagor path back into a template, so one you already have can be edited " +
			"rather than retyped. This is the inverse of render_template and the two are meant to be " +
			"used together: render, look, adjust the JSON, render again. Read get_imagor_skill before " +
			"composing anything. Note that the geometry comes back absolute - a path records the " +
			"numbers a template resolved to, not the forms it was written in.",
	}, cfg.ParseTemplateTool)

	mcp.AddTool(s, &mcp.Tool{
		Name: "get_imagor_skill",
		Description: "Read this before composing anything. The order a path is written in, how " +
			"overlays and text are placed, what the resize modes actually do, and the things a " +
			"200 response will not tell you.",
	}, cfg.SkillTool)

	// A recipe is the agent's intermediate channel: it is what gets handed to
	// render_template, not what a reader of the cookbook page wants. So it arrives
	// by name from the binary rather than being pasted around, which also means a
	// client cannot hold a template from a version the server is not running.
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_cookbook_recipe",
		Description: "Fetch a ready-made layout by name: canvas, slots and values, ready " +
			"for render_template. **Call it with no name first** to get the list - what " +
			"each layout is for, its size, and which values it takes - then call it again " +
			"with the one you want. Prefer starting from a recipe over composing from " +
			"nothing: they encode spacing and type sizes that took iteration to get right.",
	}, cfg.CookbookTool)

	// The skill travels with the server. A client that only reads tools/list
	// never sees it, and it carries what a tool description cannot: the order a
	// path is written in, how overlays and text are placed, and a working method.
	s.AddResource(&mcp.Resource{
		URI:  SourcesURI,
		Name: "imagor sample sources",
		Description: "Ready-made images for a composition, with their dimensions: " +
			"products, portraits, textures, scenes, and edge cases for testing fit and crop.",
		MIMEType: "text/markdown",
	}, func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      SourcesURI,
				MIMEType: "text/markdown",
				Text:     imagormcp.Sources,
			}},
		}, nil
	})

	s.AddResource(&mcp.Resource{
		URI:  SkillURI,
		Name: "imagor composition skill",
		Description: "How to compose and debug imagor URLs with these tools: the order a path is " +
			"written in, placement and scaling, and a working method.",
		MIMEType: "text/markdown",
	}, func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      SkillURI,
				MIMEType: "text/markdown",
				Text:     imagormcp.Skill,
			}},
		}, nil
	})

	// The cookbook index: names, what each recipe is for, and the values it takes. The
	// templates are not in here - a reader choosing a layout wants the table, and the
	// one that renders wants the recipe, which arrives through the tool.
	s.AddResource(&mcp.Resource{
		URI:  CookbookURI,
		Name: "imagor cookbook",
		Description: "Fourteen ready-made layouts with what each is for, its size, and " +
			"which values it takes. Fetch one with get_cookbook_recipe.",
		MIMEType: "text/markdown",
	}, func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      CookbookURI,
				MIMEType: "text/markdown",
				Text:     imagormcp.CookbookIndex,
			}},
		}, nil
	})
}

// SkillURI is where the server serves its own skill.
const SkillURI = "imagor-mcp://skill"

// SourcesURI is where the server serves the fixture table. It is a second resource
// rather than part of the skill because every agent that touches this server reads the
// skill, while only an agent choosing what to put in an image field wants sixteen rows
// of names and dimensions.
const SourcesURI = "imagor-mcp://sources"

// CookbookURI is the index of ready-made layouts: the names and what each is for.
// The recipe itself comes from get_cookbook_recipe, because a template is an
// intermediate channel rather than something a reader wants to see.
const CookbookURI = "imagor-mcp://cookbook"
