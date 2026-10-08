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
	"fmt"
	"hash"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/cshum/imagor/imagorpath"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cshum/imagor-mcp/internal/pathgen"
)

// Config is the runtime configuration, read from the environment so that the
// signing secret never travels through the MCP tool surface.
type Config struct {
	// Secret is IMAGOR_SECRET. Empty means path-out only: the server returns
	// unsigned paths and the caller signs them.
	Secret string
	// SignerType is IMAGOR_SIGNER_TYPE: sha1 (default), sha256 or sha512.
	SignerType string
	// Truncate is IMAGOR_SIGNER_TRUNCATE, in characters. 0 means no truncation.
	Truncate int
	// BaseURL is the optional imagor origin, used to return absolute URLs.
	BaseURL string
}

// ConfigFromEnv reads configuration from the process environment.
func ConfigFromEnv() Config {
	truncate, _ := strconv.Atoi(os.Getenv("IMAGOR_SIGNER_TRUNCATE"))
	return Config{
		Secret:     os.Getenv("IMAGOR_SECRET"),
		SignerType: strings.ToLower(strings.TrimSpace(os.Getenv("IMAGOR_SIGNER_TYPE"))),
		Truncate:   truncate,
		BaseURL:    strings.TrimSuffix(os.Getenv("IMAGOR_BASE_URL"), "/"),
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
	Path   string `json:"path" jsonschema:"the imagor path, without a leading slash"`
	URL    string `json:"url,omitempty" jsonschema:"full request path including signature, to append to the imagor origin"`
	Signed bool   `json:"signed" jsonschema:"whether a signature was applied"`
	Note   string `json:"note,omitempty" jsonschema:"human-readable notes about the result"`
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
		if strings.TrimSpace(f.Name) == "" {
			return imagorpath.Params{}, fmt.Errorf("filter name must not be empty")
		}
		filters = append(filters, imagorpath.Filter{Name: f.Name, Args: f.Args})
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

// build validates the composition and renders it.
func (c Config) build(in TransformInput) (BuildOutput, error) {
	p, err := in.toParams()
	if err != nil {
		return BuildOutput{}, err
	}
	if err := pathgen.Validate(p); err != nil {
		return BuildOutput{}, err
	}
	signer, err := c.Signer()
	if err != nil {
		return BuildOutput{}, err
	}

	out := BuildOutput{Path: pathgen.Build(p)}
	if signer != nil {
		out.URL = pathgen.BuildURL(p, signer)
		out.Signed = true
	} else {
		out.Note = "no IMAGOR_SECRET configured: returning an unsigned path for the caller to sign"
	}
	if c.BaseURL != "" && out.URL != "" {
		out.Note = "prepend " + c.BaseURL + "/ to url, or use path with your own origin"
	}
	return out, nil
}

// BuildURLTool returns a signed URL for a transformation.
func (c Config) BuildURLTool(_ context.Context, _ *mcp.CallToolRequest, in TransformInput) (*mcp.CallToolResult, BuildOutput, error) {
	out, err := c.build(in)
	if err != nil {
		return nil, BuildOutput{}, err
	}
	if !out.Signed {
		return nil, BuildOutput{}, fmt.Errorf("IMAGOR_SECRET is not set, so no signed URL can be produced; use build_imagor_path and sign in the caller")
	}
	return nil, out, nil
}

// BuildPathTool returns an unsigned imagor path.
func (c Config) BuildPathTool(_ context.Context, _ *mcp.CallToolRequest, in TransformInput) (*mcp.CallToolResult, BuildOutput, error) {
	out, err := c.build(in)
	if err != nil {
		return nil, BuildOutput{}, err
	}
	out.URL = ""
	out.Signed = false
	return nil, out, nil
}

// ValidateInput is a path to check.
type ValidateInput struct {
	Path string `json:"path" jsonschema:"an imagor path to validate, without a leading slash and without any signature prefix"`
}

// ValidateOutput reports whether the path survives a parse/regenerate round trip.
type ValidateOutput struct {
	Valid    bool     `json:"valid" jsonschema:"whether the path round-trips unchanged"`
	Path     string   `json:"path" jsonschema:"the path as parsed"`
	Rebuilt  string   `json:"rebuilt" jsonschema:"the path regenerated from the parsed parameters"`
	Stable   bool     `json:"stable" jsonschema:"whether regenerating the parsed path reproduces the input exactly"`
	Problems []string `json:"problems,omitempty" jsonschema:"human-readable differences found"`
	Summary  string   `json:"summary" jsonschema:"a short description of the parsed transformation"`
}

// ValidateTool parses a path and reports whether it is well formed.
func (c Config) ValidateTool(_ context.Context, _ *mcp.CallToolRequest, in ValidateInput) (*mcp.CallToolResult, ValidateOutput, error) {
	path := strings.TrimPrefix(strings.TrimSpace(in.Path), "/")
	if path == "" {
		return nil, ValidateOutput{}, fmt.Errorf("path is required")
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
		out.Valid = false
		out.Problems = append(out.Problems, fmt.Sprintf("regenerating the parsed path produced %q, which differs from the input", rebuilt))
	}
	if err := pathgen.Validate(p); err != nil {
		out.Valid = false
		out.Problems = append(out.Problems, err.Error())
	}
	return nil, out, nil
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
	SignerType   string     `json:"signer_type"`
	Truncate     int        `json:"truncate"`
	Signed       bool       `json:"signing_enabled"`
	BaseURL      string     `json:"base_url,omitempty"`
	PathParams   []ParamDoc `json:"path_params" jsonschema:"path grammar parameters, reflected from the imagor path type"`
	HAlignValues []string   `json:"h_align_values"`
	VAlignValues []string   `json:"v_align_values"`
	TrimByValues []string   `json:"trim_by_values"`
	Limitations  []string   `json:"limitations"`
}

// DescribeTool reports the supported grammar and the server's signing mode.
//
// The parameter list is reflected from imagor's own Params type, so it cannot
// drift from the implementation. The filter catalogue is NOT included, because
// imagor does not currently expose its filter registry in a machine-readable
// form and a hand-maintained copy would silently rot.
func (c Config) DescribeTool(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, DescribeOutput, error) {
	out := DescribeOutput{
		SignerType:   c.SignerType,
		Truncate:     c.Truncate,
		Signed:       c.Secret != "",
		BaseURL:      c.BaseURL,
		PathParams:   reflectParams(),
		HAlignValues: []string{imagorpath.HAlignLeft, imagorpath.HAlignRight},
		VAlignValues: []string{imagorpath.VAlignTop, imagorpath.VAlignBottom},
		TrimByValues: []string{imagorpath.TrimByTopLeft, imagorpath.TrimByBottomRight},
		Limitations: []string{
			"the filter catalogue is not machine-readable in imagor today, so this server does not enumerate it; pass filter names and arguments directly",
			"this server does not upload, store or render images; it only constructs paths and signatures",
		},
	}
	if out.SignerType == "" && c.Secret != "" {
		out.SignerType = "sha1"
	}
	return nil, out, nil
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
		Name: "describe_imagor",
		Description: "Describe the imagor path grammar this server targets, including the parameters " +
			"reflected from imagor's own path type and the current signing mode.",
	}, cfg.DescribeTool)
}
