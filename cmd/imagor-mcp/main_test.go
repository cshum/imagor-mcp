package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "imagor-mcp-test")
	if err != nil {
		panic(err)
	}
	binPath = filepath.Join(dir, "imagor-mcp")
	if out, err := exec.Command("go", "build", "-o", binPath, ".").CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		panic("build failed: " + string(out))
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// connect spawns the real binary over stdio and returns a live session.
func connect(t *testing.T, env map[string]string) *mcp.ClientSession {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(), "IMAGOR_SECRET=", "IMAGOR_UNSAFE=", "IMAGOR_SIGNER_TYPE=", "IMAGOR_SIGNER_TRUNCATE=", "IMAGOR_BASE_URL=", "IMAGOR_MCP_EXTRA_FILTERS=")
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// callTool invokes a tool and decodes its structured content into out.
func callTool(t *testing.T, s *mcp.ClientSession, name string, args any, out any) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return err
	}
	if res.IsError {
		var b strings.Builder
		for _, c := range res.Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				b.WriteString(tc.Text)
			}
		}
		return &toolError{b.String()}
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

type toolError struct{ msg string }

func (e *toolError) Error() string { return e.msg }

func TestToolsAreRegisteredWithSchemas(t *testing.T) {
	s := connect(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := s.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	got := map[string]bool{}
	for _, tool := range res.Tools {
		got[tool.Name] = true
		if tool.InputSchema == nil {
			t.Errorf("tool %s has no input schema", tool.Name)
		}
		if tool.Description == "" {
			t.Errorf("tool %s has no description", tool.Name)
		}
	}
	for _, want := range []string{"build_imagor_url", "build_imagor_path", "validate_imagor_path", "list_imagor_filters", "describe_imagor"} {
		if !got[want] {
			t.Errorf("tool %s not registered", want)
		}
	}
}

// The signature produced through the MCP tool surface must match the value
// independently verified against a running imagor instance.
func TestBuildURLMatchesVerifiedValue(t *testing.T) {
	s := connect(t, map[string]string{
		"IMAGOR_SECRET":          "mysecret",
		"IMAGOR_SIGNER_TYPE":     "sha256",
		"IMAGOR_SIGNER_TRUNCATE": "40",
	})

	var out struct {
		Path   string `json:"path"`
		URL    string `json:"url"`
		Signed bool   `json:"signed"`
	}
	err := callTool(t, s, "build_imagor_url", map[string]any{
		"image":   "raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png",
		"width":   500,
		"height":  500,
		"v_align": "top",
		"filters": []map[string]any{
			{"name": "fill", "args": "white"},
			{"name": "format", "args": "jpeg"},
			{"name": "quality", "args": "80"},
		},
	}, &out)
	if err != nil {
		t.Fatalf("build_imagor_url: %v", err)
	}

	const wantURL = "jxr-nEUK64wb7evFuOx1qCc52zHvFrJff3eYyyS4/500x500/top/filters:fill(white):format(jpeg):quality(80)/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png"
	if out.URL != wantURL {
		t.Fatalf("signed URL mismatch\n got: %s\nwant: %s", out.URL, wantURL)
	}
	if !out.Signed {
		t.Error("expected signed=true")
	}
}

func TestPathOutModeWithoutSecret(t *testing.T) {
	s := connect(t, nil)

	var out struct {
		URL    string `json:"url"`
		Signed bool   `json:"signed"`
		Note   string `json:"note"`
	}
	err := callTool(t, s, "build_imagor_path", map[string]any{
		"image":  "example.com/a.jpg",
		"width":  320,
		"height": 240,
		"smart":  true,
	}, &out)
	if err != nil {
		t.Fatalf("build_imagor_path: %v", err)
	}
	if out.Signed || out.URL != "" {
		t.Errorf("path-out mode must not sign: %+v", out)
	}
	if out.Note == "" {
		t.Error("expected a note explaining the unsigned result")
	}
}

func TestSignedURLRefusedWithoutSecret(t *testing.T) {
	s := connect(t, nil)
	err := callTool(t, s, "build_imagor_url", map[string]any{
		"image": "example.com/a.jpg",
		"width": 100,
	}, &struct{}{})
	if err == nil {
		t.Fatal("expected build_imagor_url to refuse when no secret is configured")
	}
}

// Invalid compositions must be refused, not emitted.
func TestInvalidCompositionRefused(t *testing.T) {
	s := connect(t, map[string]string{
		"IMAGOR_SECRET":      "mysecret",
		"IMAGOR_SIGNER_TYPE": "sha256",
	})

	for name, args := range map[string]map[string]any{
		"full_fit_in without fit_in": {"image": "a.com/a.jpg", "full_fit_in": true},
		"trim_by without trim":       {"image": "a.com/a.jpg", "trim_by": "top-left"},
		"bad h_align":                {"image": "a.com/a.jpg", "h_align": "middle"},
		"missing image":              {"width": 100},
	} {
		t.Run(name, func(t *testing.T) {
			if err := callTool(t, s, "build_imagor_url", args, &struct{}{}); err == nil {
				t.Fatal("expected an error, got none")
			}
		})
	}
}

func TestValidateTool(t *testing.T) {
	s := connect(t, nil)

	var out struct {
		Valid    bool     `json:"valid"`
		Stable   bool     `json:"stable"`
		Summary  string   `json:"summary"`
		Problems []string `json:"problems"`
	}
	err := callTool(t, s, "validate_imagor_path", map[string]any{
		"path": "500x500/top/filters:fill(white):format(jpeg):quality(80)/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png",
	}, &out)
	if err != nil {
		t.Fatalf("validate_imagor_path: %v", err)
	}
	if !out.Valid || !out.Stable {
		t.Fatalf("expected a valid stable path, got valid=%v stable=%v problems=%v", out.Valid, out.Stable, out.Problems)
	}
	if !strings.Contains(out.Summary, "500x500") || !strings.Contains(out.Summary, "fill") {
		t.Errorf("summary missing detail: %q", out.Summary)
	}
}

// The grammar description must be derived from imagor's own types, so it
// cannot drift from what the tools actually build.
func TestDescribeReflectsImagorParams(t *testing.T) {
	s := connect(t, nil)

	var out struct {
		PathParams []struct {
			Name   string `json:"name"`
			GoType string `json:"go_type"`
		} `json:"path_params"`
		Signed      bool     `json:"signing_enabled"`
		Limitations []string `json:"limitations"`
	}
	if err := callTool(t, s, "describe_imagor", map[string]any{}, &out); err != nil {
		t.Fatalf("describe_imagor: %v", err)
	}

	names := map[string]bool{}
	for _, p := range out.PathParams {
		names[p.Name] = true
	}
	for _, want := range []string{"image", "width", "height", "filters", "v_align", "crop_left"} {
		if !names[want] {
			t.Errorf("path param %q missing from reflected grammar", want)
		}
	}
	if out.Signed {
		t.Error("expected signing_enabled=false without a secret")
	}
	if len(out.Limitations) == 0 {
		t.Error("expected the filter-catalogue limitation to be declared")
	}
}

// Unsafe mode must produce a genuinely usable "unsafe/" request path, and must
// never claim it is signed.
func TestUnsafeMode(t *testing.T) {
	s := connect(t, map[string]string{"IMAGOR_UNSAFE": "1"})

	var out struct {
		Mode   string `json:"mode"`
		Path   string `json:"path"`
		URL    string `json:"url"`
		Signed bool   `json:"signed"`
		Note   string `json:"note"`
	}
	if err := callTool(t, s, "build_imagor_url", map[string]any{
		"image":  "example.com/a.jpg",
		"width":  320,
		"height": 240,
		"smart":  true,
	}, &out); err != nil {
		t.Fatalf("build_imagor_url in unsafe mode: %v", err)
	}
	if out.Mode != "unsafe" {
		t.Errorf("mode = %q, want unsafe", out.Mode)
	}
	if !strings.HasPrefix(out.URL, "unsafe/") {
		t.Errorf("unsafe URL must carry the unsafe/ prefix, got %q", out.URL)
	}
	if out.Signed {
		t.Error("unsafe mode must not report signed=true")
	}
	if out.URL != "unsafe/"+out.Path {
		t.Errorf("URL %q is not the unsafe prefix plus path %q", out.URL, out.Path)
	}
	if !strings.Contains(out.Note, "IMAGOR_UNSAFE") {
		t.Errorf("note should warn about the target instance, got %q", out.Note)
	}
}

// build_imagor_path must stay usable in every mode, since it is the fallback
// for callers that sign themselves.
func TestPathToolWorksInUnsafeAndSignedModes(t *testing.T) {
	for name, env := range map[string]map[string]string{
		"unsafe": {"IMAGOR_UNSAFE": "1"},
		"signed": {"IMAGOR_SECRET": "s", "IMAGOR_SIGNER_TYPE": "sha256"},
	} {
		t.Run(name, func(t *testing.T) {
			s := connect(t, env)
			var out struct {
				Path        string `json:"path"`
				URL         string `json:"url"`
				AbsoluteURL string `json:"absolute_url"`
				Signed      bool   `json:"signed"`
			}
			if err := callTool(t, s, "build_imagor_path", map[string]any{
				"image": "example.com/a.jpg",
				"width": 100,
			}, &out); err != nil {
				t.Fatalf("build_imagor_path: %v", err)
			}
			if out.Path == "" {
				t.Error("path must not be empty")
			}
			if out.URL != "" || out.AbsoluteURL != "" || out.Signed {
				t.Errorf("path tool must return an unprefixed path: %+v", out)
			}
		})
	}
}

// Setting both a secret and unsafe mode states opposite intentions, so the
// server must refuse to start rather than pick one.
func TestSecretAndUnsafeTogetherRefuseStartup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath)
	cmd.Env = append(os.Environ(), "IMAGOR_SECRET=mysecret", "IMAGOR_UNSAFE=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected startup to fail, got success with output: %s", out)
	}
	if !strings.Contains(string(out), "contradict") {
		t.Errorf("expected a contradiction error, got: %s", out)
	}
}

func TestAbsoluteURLUsesBaseURL(t *testing.T) {
	s := connect(t, map[string]string{
		"IMAGOR_SECRET":          "mysecret",
		"IMAGOR_SIGNER_TYPE":     "sha256",
		"IMAGOR_SIGNER_TRUNCATE": "40",
		"IMAGOR_BASE_URL":        "https://img.example.com/", // trailing slash must be trimmed
	})

	var out struct {
		URL         string `json:"url"`
		AbsoluteURL string `json:"absolute_url"`
	}
	if err := callTool(t, s, "build_imagor_url", map[string]any{
		"image": "example.com/a.jpg",
		"width": 100,
	}, &out); err != nil {
		t.Fatalf("build_imagor_url: %v", err)
	}
	want := "https://img.example.com/" + out.URL
	if out.AbsoluteURL != want {
		t.Errorf("absolute_url = %q, want %q", out.AbsoluteURL, want)
	}
	if strings.Contains(out.AbsoluteURL, "img.example.com//") {
		t.Errorf("trailing slash on the base URL was not trimmed: %q", out.AbsoluteURL)
	}
}

// The whole reason the catalogue exists: an invented filter would otherwise
// produce a valid, correctly-signed URL that returns 200 with the image
// unchanged, and neither the agent nor the caller could tell.
func TestInventedFilterIsRefusedByTool(t *testing.T) {
	s := connect(t, map[string]string{
		"IMAGOR_SECRET":          "mysecret",
		"IMAGOR_SIGNER_TYPE":     "sha256",
		"IMAGOR_SIGNER_TRUNCATE": "40",
	})

	err := callTool(t, s, "build_imagor_url", map[string]any{
		"image":   "example.com/a.jpg",
		"width":   200,
		"filters": []map[string]any{{"name": "vintage", "args": "70"}},
	}, &struct{}{})
	if err == nil {
		t.Fatal("build_imagor_url must refuse a filter imagor would silently ignore")
	}
	if !strings.Contains(err.Error(), "unknown filter") {
		t.Errorf("error should explain the problem, got: %v", err)
	}
}

func TestWrongArgumentCountIsRefusedByTool(t *testing.T) {
	s := connect(t, map[string]string{
		"IMAGOR_SECRET":          "mysecret",
		"IMAGOR_SIGNER_TYPE":     "sha256",
		"IMAGOR_SIGNER_TRUNCATE": "40",
	})

	err := callTool(t, s, "build_imagor_url", map[string]any{
		"image":   "example.com/a.jpg",
		"width":   200,
		"filters": []map[string]any{{"name": "blur", "args": "1,2,3"}},
	}, &struct{}{})
	if err == nil {
		t.Fatal("build_imagor_url must refuse blur with too many arguments")
	}
	if !strings.Contains(err.Error(), "at most") {
		t.Errorf("error should state the limit, got: %v", err)
	}
}

// A legitimate filter must still work, so the strictness does not block real use.
func TestLegitimateFiltersStillAccepted(t *testing.T) {
	s := connect(t, map[string]string{
		"IMAGOR_SECRET":          "mysecret",
		"IMAGOR_SIGNER_TYPE":     "sha256",
		"IMAGOR_SIGNER_TRUNCATE": "40",
	})

	var out struct {
		URL string `json:"url"`
	}
	err := callTool(t, s, "build_imagor_url", map[string]any{
		"image":  "example.com/a.jpg",
		"width":  200,
		"height": 200,
		"filters": []map[string]any{
			{"name": "fill", "args": "white"},
			{"name": "blur", "args": "5"},
			{"name": "grayscale"},
			{"name": "format", "args": "jpeg"},
			{"name": "quality", "args": "80"},
		},
	}, &out)
	if err != nil {
		t.Fatalf("legitimate filters were refused: %v", err)
	}
	if !strings.Contains(out.URL, "blur(5)") || !strings.Contains(out.URL, "grayscale()") {
		t.Errorf("expected the filters in the URL, got %q", out.URL)
	}
}

func TestListFiltersTool(t *testing.T) {
	s := connect(t, nil)

	var out struct {
		ImagorVersion string `json:"imagor_version"`
		Count         int    `json:"count"`
		Filters       []struct {
			Name       string `json:"name"`
			Signature  string `json:"signature"`
			MinArgs    int    `json:"min_args"`
			MaxArgs    int    `json:"max_args"`
			Documented bool   `json:"documented"`
		} `json:"filters"`
	}
	if err := callTool(t, s, "list_imagor_filters", map[string]any{}, &out); err != nil {
		t.Fatalf("list_imagor_filters: %v", err)
	}
	if out.ImagorVersion == "" {
		t.Error("imagor_version must be reported so callers can detect a stale catalogue")
	}
	if out.Count < 40 || out.Count != len(out.Filters) {
		t.Fatalf("expected at least 40 filters, got count=%d len=%d", out.Count, len(out.Filters))
	}
	byName := map[string]int{}
	for i, f := range out.Filters {
		byName[f.Name] = i
	}
	i, ok := byName["text"]
	if !ok {
		t.Fatal("text should be in the catalogue")
	}
	if out.Filters[i].Signature == "" || out.Filters[i].MaxArgs < 3 {
		t.Errorf("text should advertise its argument range, got %+v", out.Filters[i])
	}
}

func TestListFiltersFiltersByName(t *testing.T) {
	s := connect(t, nil)

	var out struct {
		Count   int `json:"count"`
		Filters []struct {
			Name string `json:"name"`
		} `json:"filters"`
	}
	if err := callTool(t, s, "list_imagor_filters", map[string]any{"name_contains": "strip"}, &out); err != nil {
		t.Fatalf("list_imagor_filters: %v", err)
	}
	if out.Count == 0 {
		t.Fatal("expected strip_* filters")
	}
	for _, f := range out.Filters {
		if !strings.Contains(f.Name, "strip") {
			t.Errorf("filter %q does not match the requested substring", f.Name)
		}
	}
}

// quality() is arity-valid on its own but has no effect on lossless output.
// Verified against imagor: no format filter yields PNG, where quality(30) and
// quality() both return byte-identical output to no filter at all.
func TestQualityWithoutLossyFormatWarns(t *testing.T) {
	s := connect(t, map[string]string{
		"IMAGOR_SECRET":          "mysecret",
		"IMAGOR_SIGNER_TYPE":     "sha256",
		"IMAGOR_SIGNER_TRUNCATE": "40",
	})

	var warned struct {
		Warnings []string `json:"warnings"`
	}
	if err := callTool(t, s, "build_imagor_url", map[string]any{
		"image":   "example.com/a.jpg",
		"width":   200,
		"filters": []map[string]any{{"name": "quality", "args": "30"}},
	}, &warned); err != nil {
		t.Fatalf("build_imagor_url: %v", err)
	}
	if len(warned.Warnings) == 0 {
		t.Fatal("quality without a lossy format should warn that it has no effect")
	}
	if !strings.Contains(warned.Warnings[0], "format(jpeg)") {
		t.Errorf("warning should suggest a fix, got %q", warned.Warnings[0])
	}

	var quiet struct {
		Warnings []string `json:"warnings"`
	}
	if err := callTool(t, s, "build_imagor_url", map[string]any{
		"image": "example.com/a.jpg",
		"width": 200,
		"filters": []map[string]any{
			{"name": "format", "args": "jpeg"},
			{"name": "quality", "args": "30"},
		},
	}, &quiet); err != nil {
		t.Fatalf("build_imagor_url: %v", err)
	}
	if len(quiet.Warnings) != 0 {
		t.Errorf("quality with format(jpeg) should not warn, got %v", quiet.Warnings)
	}
}

// Too-few arguments is the larger silent-failure class: imagor returns the
// image unchanged with a 200 response for blur(), rotate(), brightness(),
// hue(), contrast(), saturation(), format() and quality(). The tool must
// refuse these rather than emit an inert URL.
func TestMissingRequiredArgumentsRefused(t *testing.T) {
	s := connect(t, map[string]string{
		"IMAGOR_SECRET":          "mysecret",
		"IMAGOR_SIGNER_TYPE":     "sha256",
		"IMAGOR_SIGNER_TRUNCATE": "40",
	})

	for _, name := range []string{"blur", "rotate", "brightness", "hue", "contrast", "saturation", "format", "quality"} {
		t.Run(name, func(t *testing.T) {
			err := callTool(t, s, "build_imagor_url", map[string]any{
				"image":   "example.com/a.jpg",
				"width":   200,
				"filters": []map[string]any{{"name": name}},
			}, &struct{}{})
			if err == nil {
				t.Fatalf("%s() with no arguments must be refused: imagor ignores it silently", name)
			}
		})
	}
}
