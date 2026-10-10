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
	"errors"
	"fmt"
	"hash"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/cshum/imagor/imagorpath"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	filtercat "github.com/cshum/imagor-mcp/internal/filters"
	"github.com/cshum/imagor-mcp/internal/pathgen"
)

// Mode is how the server produces request paths.
type Mode string

const (
	// ModeSigned signs every path with IMAGOR_SECRET. Use in production.
	ModeSigned Mode = "signed"
	// ModeUnsafe emits "unsafe/" paths and signs nothing. Development only;
	// the target imagor must itself be running with IMAGOR_UNSAFE=1.
	ModeUnsafe Mode = "unsafe"
	// ModePathOut returns bare paths for the caller to sign. The server holds
	// no secret at all.
	ModePathOut Mode = "path_out"
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
		return ModePathOut, nil
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

// FilterInput is one imagor filter and its arguments.
type FilterInput struct {
	Name string `json:"name" jsonschema:"filter name, for example fill, format, quality, blur, rotate, invert, watermark, text, image, max_age"`
	Args string `json:"args,omitempty" jsonschema:"comma-separated arguments for the filter, for example \"white\" for fill, \"jpeg\" for format, \"80\" for quality. Omit for filters that take none."`
}

// TransformInput describes an image transformation. Every field maps onto a
// concept in the imagor path grammar; there are no invented parameters.
type TransformInput struct {
	Image string `json:"image" jsonschema:"source image: a URL or path such as \"example.com/a.jpg\", or a synthetic source such as \"color:red\" or \"color:none\""`

	Width  int `json:"width,omitempty" jsonschema:"target width in pixels; 0 means automatic"`
	Height int `json:"height,omitempty" jsonschema:"target height in pixels; 0 means automatic"`

	FitIn         bool `json:"fit_in,omitempty" jsonschema:"fit within width x height instead of cropping to fill"`
	FullFitIn     bool `json:"full_fit_in,omitempty" jsonschema:"fit using the largest dimension; requires fit_in"`
	AdaptiveFitIn bool `json:"adaptive_fit_in,omitempty" jsonschema:"invert width and height when that yields better definition; requires fit_in"`
	Stretch       bool `json:"stretch,omitempty" jsonschema:"stretch to exactly width x height, ignoring aspect ratio"`
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

	Filters []FilterInput `json:"filters,omitempty" jsonschema:"ordered filters, applied left to right"`

	ExpireUnixMillis int64 `json:"expire_unix_millis,omitempty" jsonschema:"add an expire() filter so the URL stops working after this unix-milliseconds timestamp"`
	MaxAgeSeconds    int   `json:"max_age_seconds,omitempty" jsonschema:"add a max_age() filter limiting how long the response may be cached"`
}

// BuildOutput is the result of building a path or URL.
type BuildOutput struct {
	Mode        string   `json:"mode" jsonschema:"how the path was produced: signed, unsafe or path_out"`
	Path        string   `json:"path" jsonschema:"the imagor path, without any signature or unsafe prefix"`
	URL         string   `json:"url,omitempty" jsonschema:"a request path ready to send to imagor, including its signature or unsafe prefix; empty in path_out mode"`
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
		filters = append(filters, imagorpath.Filter{Name: name, Args: f.Args})
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

// compositionWarnings reports combinations that pass validation but will not do
// what the caller expects. Argument-count checking cannot catch these, because
// each filter involved is individually well formed.
func compositionWarnings(p imagorpath.Params) []string {
	if !imagorpath.HasFilter(p, "quality") {
		return nil
	}
	for _, f := range p.Filters {
		if f.Name != "format" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(f.Args)) {
		case "jpeg", "jpg", "webp", "avif", "heif":
			return nil // a lossy format is requested, so quality applies
		}
	}
	return []string{
		"quality() only affects lossy output formats. No lossy format() filter is present, so the " +
			"result depends on the client's Accept header and may be lossless PNG, in which case " +
			"quality has no effect. Add format(jpeg), format(webp) or format(avif) to be certain.",
	}
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
	default:
		out.Note = "path_out mode: sign this path with your own secret before serving it"
	}
	if c.BaseURL != "" && out.URL != "" {
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
	if out.URL == "" {
		return nil, BuildOutput{}, fmt.Errorf(
			"this server runs in path_out mode, so it holds no signing secret and cannot produce a request URL; use build_imagor_path and sign the result in the caller, or configure IMAGOR_SECRET")
	}
	return nil, out, nil
}

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
	Mode          string     `json:"mode" jsonschema:"signed, unsafe or path_out"`
	ImagorVersion string     `json:"imagor_version" jsonschema:"the imagor release the filter catalogue was generated from"`
	FilterCount   int        `json:"filter_count" jsonschema:"number of filters accepted by the tools"`
	SignerType    string     `json:"signer_type"`
	Truncate      int        `json:"truncate"`
	Signed        bool       `json:"signing_enabled"`
	BaseURL       string     `json:"base_url,omitempty"`
	PathParams    []ParamDoc `json:"path_params" jsonschema:"path grammar parameters, reflected from the imagor path type"`
	HAlignValues  []string   `json:"h_align_values"`
	VAlignValues  []string   `json:"v_align_values"`
	TrimByValues  []string   `json:"trim_by_values"`
	Limitations   []string   `json:"limitations"`
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
	if out.SignerType == "" {
		out.SignerType = "sha1"
	}
	switch mode {
	case ModeUnsafe:
		out.Limitations = append(out.Limitations,
			"UNSAFE MODE: this server emits unsigned URLs and only works against an imagor instance running with IMAGOR_UNSAFE=1. Never use it in production.")
	case ModePathOut:
		out.Limitations = append(out.Limitations,
			"path_out mode: this server holds no signing secret, so build_imagor_url is unavailable. Sign the path returned by build_imagor_path in the caller.")
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
	Arguments []string `json:"arguments,omitempty"`
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
			Arguments:   spec.Arguments,
		})
	}
	out.Count = len(out.Filters)
	if out.Count == 0 {
		out.Note = fmt.Sprintf("no filter name contains %q", in.NameContains)
	}
	return nil, out, nil
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
			"URL is returned, so an invalid combination is refused rather than emitted.",
	}, cfg.BuildURLTool)

	mcp.AddTool(s, &mcp.Tool{
		Name: "build_imagor_path",
		Description: "Build an unsigned imagor path from typed transformation parameters, for the " +
			"caller to sign with its own secret. Use this when the MCP server should not hold " +
			"the signing secret.",
	}, cfg.BuildPathTool)

	mcp.AddTool(s, &mcp.Tool{
		Name: "validate_imagor_path",
		Description: "Parse an imagor path and report whether it round-trips unchanged, plus a short " +
			"summary of the transformation it describes. Use this to check a path before serving it.",
	}, cfg.ValidateTool)

	mcp.AddTool(s, &mcp.Tool{
		Name: "list_imagor_filters",
		Description: "List the imagor filters this server accepts, with each filter's argument count. " +
			"imagor ignores unrecognised filters and wrong argument counts silently — the request " +
			"returns 200 with the image unchanged — so use this to pick filters that exist instead " +
			"of guessing names.",
	}, cfg.ListFiltersTool)

	mcp.AddTool(s, &mcp.Tool{
		Name: "describe_imagor",
		Description: "Describe the imagor path grammar this server targets, including the parameters " +
			"reflected from imagor's own path type and the current signing mode.",
	}, cfg.DescribeTool)
}
