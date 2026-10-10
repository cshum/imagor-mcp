package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var binPath string

// unsafeEnv selects a mode for tests that do not care which one. The server
// refuses to start without a mode, so there is no default to fall back on.
var unsafeEnv = map[string]string{"IMAGOR_UNSAFE": "1"}

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
	s := connect(t, unsafeEnv)
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
	s := connect(t, unsafeEnv)

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
		t.Errorf("build_imagor_path must not sign or return a request URL: %+v", out)
	}
	if out.Note == "" {
		t.Error("expected a note explaining the unsigned result")
	}
}

// Unsafe mode must never produce a signature: the server signs only when it holds
// a secret, and a mode is required, so there is no unsigned-with-secret state.
func TestUnsafeModeEmitsNoSignature(t *testing.T) {
	s := connect(t, unsafeEnv)

	var out struct {
		Mode   string `json:"mode"`
		URL    string `json:"url"`
		Signed bool   `json:"signed"`
	}
	if err := callTool(t, s, "build_imagor_url", map[string]any{
		"image": "example.com/a.jpg",
		"width": 100,
	}, &out); err != nil {
		t.Fatalf("build_imagor_url: %v", err)
	}
	if out.Mode != "unsafe" || out.Signed {
		t.Errorf("mode=%q signed=%v, want unsafe and unsigned", out.Mode, out.Signed)
	}
	if !strings.HasPrefix(out.URL, "unsafe/") {
		t.Errorf("url %q does not carry the unsafe prefix", out.URL)
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
	s := connect(t, unsafeEnv)

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

// A path nested in image() is parsed with the same grammar as the outer path and
// loaded as an image, so a fault inside it fails the whole request. The
// validator has to reach it. The first case is the one imagor answers with a DNS
// failure, and it reported valid:true until this check existed.
func TestValidateToolChecksNestedPaths(t *testing.T) {
	s := connect(t, unsafeEnv)

	var out struct {
		Valid    bool     `json:"valid"`
		Problems []string `json:"problems"`
		Warnings []string `json:"warnings"`
	}

	const ref = "image-cache.goxip.com/banner-w500.jpg"
	encoded := base64.RawURLEncoding.EncodeToString([]byte("example.com/a,b.jpg"))

	for _, tc := range []struct {
		name        string
		path        string
		valid       bool
		wantProblem string
		wantWarning string
	}{
		{
			name:        "filters before parameters in a nested path",
			path:        "fit-in/400x300/filters:image(filters:crop(0,0,0.5,1)/fit-in/300x300/" + ref + ",20,20)/color:white",
			valid:       false,
			wantProblem: "parameters come before filters",
		},
		{
			name:  "parameters before filters in a nested path",
			path:  "fit-in/400x300/filters:image(fit-in/300x300/filters:crop(0,0,0.5,1)/" + ref + ",20,20)/color:white",
			valid: true,
		},
		{
			name:        "unknown filter name",
			path:        "400x400/filters:frmt(jpeg)/example.com/a.jpg",
			valid:       false,
			wantProblem: "unknown filter",
		},
		{
			name:        "unknown filter inside a nested path",
			path:        "filters:image(fit-in/100x100/filters:frmt(jpeg)/example.com/a.jpg,10,10)/color:white",
			valid:       false,
			wantProblem: "unknown filter",
		},
		{
			name:  "b64 encoded nested url with a comma",
			path:  "filters:image(b64:" + encoded + ",10,10)/color:white",
			valid: true,
		},
		{
			name:        "unsafe prefix is not part of the path",
			path:        "unsafe/fit-in/100x100/example.com/a.jpg",
			valid:       true,
			wantWarning: "unsafe/",
		},
		{
			// The crop pair is a parameter, so writing it after the filters
			// makes imagor read it as part of the image path.
			name:        "crop written after the filters",
			path:        "fit-in/300x300/filters:upscale()/201x65:289x99/example.com/a.jpg",
			valid:       false,
			wantProblem: "parameters come before filters",
		},
		{
			name:        "filters inside watermark() never run",
			path:        "filters:watermark(filters:blur(3)/example.com/logo.png,10,10,50)/example.com/a.jpg",
			valid:       false,
			wantProblem: "runs no filters",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := callTool(t, s, "validate_imagor_path", map[string]any{"path": tc.path}, &out); err != nil {
				t.Fatalf("validate_imagor_path: %v", err)
			}
			if out.Valid != tc.valid {
				t.Fatalf("valid = %v, want %v; problems = %v", out.Valid, tc.valid, out.Problems)
			}
			if tc.wantProblem != "" && !mentions(out.Problems, tc.wantProblem) {
				t.Errorf("problems %v do not mention %q", out.Problems, tc.wantProblem)
			}
			if tc.wantWarning != "" && !mentions(out.Warnings, tc.wantWarning) {
				t.Errorf("warnings %v do not mention %q", out.Warnings, tc.wantWarning)
			}
		})
	}
}

// The server produces signed or unsafe paths, so it must refuse to start without
// a mode rather than come up in a state where its tools cannot produce a URL.
func TestServerRequiresAMode(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  []string
		want string
	}{
		{"neither set", []string{"IMAGOR_SECRET=", "IMAGOR_UNSAFE="}, "no mode selected"},
		{"both set", []string{"IMAGOR_SECRET=s3cret", "IMAGOR_UNSAFE=1"}, "IMAGOR_SECRET"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binPath)
			cmd.Env = append(os.Environ(),
				"IMAGOR_SECRET=", "IMAGOR_UNSAFE=", "IMAGOR_SIGNER_TYPE=",
				"IMAGOR_SIGNER_TRUNCATE=", "IMAGOR_BASE_URL=", "IMAGOR_MCP_EXTRA_FILTERS=")
			cmd.Env = append(cmd.Env, tc.env...)

			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("the server started with %s; output: %s", tc.name, out)
			}
			if !strings.Contains(string(out), tc.want) {
				t.Errorf("the failure does not explain itself, want %q in: %s", tc.want, out)
			}
			if !strings.Contains(string(out), "IMAGOR_UNSAFE") {
				t.Errorf("the failure does not name the environment variables: %s", out)
			}
		})
	}
}

func mentions(lines []string, want string) bool {
	for _, l := range lines {
		if strings.Contains(l, want) {
			return true
		}
	}
	return false
}

// verify_imagor_url reports what the server did with each filter, including the
// filters inside an image() path, and must not turn a server that reports nothing
// into a failure: unknown is not wrong.
func TestVerifyURLTool(t *testing.T) {
	var served atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, served.Load().(string))
	}))
	defer srv.Close()

	s := connect(t, map[string]string{
		"IMAGOR_UNSAFE":   "1",
		"IMAGOR_BASE_URL": srv.URL,
	})

	type outcome struct {
		Name    string    `json:"name"`
		Outcome string    `json:"outcome"`
		Filters []outcome `json:"filters"`
	}
	var out struct {
		Mode            string `json:"mode"`
		Path            string `json:"path"`
		URL             string `json:"url"`
		AbsoluteURL     string `json:"absolute_url"`
		MetaURL         string `json:"meta_url"`
		MetaAbsoluteURL string `json:"meta_absolute_url"`
		Verified        bool   `json:"verified"`
		Server          struct {
			Version          string `json:"version"`
			ReportsVersion   bool   `json:"reports_version"`
			ReportsFilters   bool   `json:"reports_filters"`
			CatalogueVersion string `json:"catalogue_version"`
		} `json:"server"`
		Status  int       `json:"status"`
		Note    string    `json:"note"`
		Results []outcome `json:"results"`
	}
	args := map[string]any{
		"image": "example.com/a.jpg", "width": 400, "height": 300, "fit_in": true,
		"filters": []map[string]any{
			{"name": "image", "args": "fit-in/100x100/filters:blur(3):rotate()/example.com/b.jpg,10,10"},
			{"name": "quality", "args": "80"},
		},
	}

	t.Run("the server reports outcomes", func(t *testing.T) {
		served.Store(`{"version":"v1.9.7","filters":[` +
			`{"name":"image","processed":true,"filters":[` +
			`{"name":"blur","processed":true},{"name":"rotate","processed":false}]},` +
			`{"name":"quality","processed":true}]}`)

		if err := callTool(t, s, "verify_imagor_url", args, &out); err != nil {
			t.Fatalf("verify_imagor_url: %v", err)
		}
		if !out.Verified || out.Status != 200 {
			t.Fatalf("expected a verified 200, got verified=%v status=%d note=%q", out.Verified, out.Status, out.Note)
		}
		// Both added fields are there, so the caller can trust what it read.
		if !out.Server.ReportsVersion || !out.Server.ReportsFilters {
			t.Errorf("server = %+v, want both fields reported", out.Server)
		}
		if out.Server.Version != out.Server.CatalogueVersion {
			t.Errorf("server version %q and catalogue version %q should agree here",
				out.Server.Version, out.Server.CatalogueVersion)
		}
		if strings.Contains(out.Note, "may differ") || strings.Contains(out.Note, "predates") {
			t.Errorf("note %q warns about a version it matches", out.Note)
		}
		// Both URLs come back, and they differ: a signature covers the whole
		// path, so the metadata URL is not derivable from the image URL.
		if out.URL == "" || out.MetaURL == "" {
			t.Fatalf("expected both URLs, got url=%q meta_url=%q", out.URL, out.MetaURL)
		}
		if out.URL == out.MetaURL {
			t.Errorf("the image URL and the metadata URL are the same, so one of them is wrong")
		}
		if !strings.Contains(out.MetaURL, "/meta/") {
			t.Errorf("meta_url %q does not address the metadata", out.MetaURL)
		}
		if strings.Contains(out.URL, "/meta/") {
			t.Errorf("url %q addresses metadata rather than the image", out.URL)
		}
		if !strings.HasSuffix(out.AbsoluteURL, out.URL) || !strings.HasSuffix(out.MetaAbsoluteURL, out.MetaURL) {
			t.Errorf("absolute URLs do not match their paths: %q / %q", out.AbsoluteURL, out.MetaAbsoluteURL)
		}
		if len(out.Results) != 2 {
			t.Fatalf("expected 2 results, got %+v", out.Results)
		}
		if out.Results[0].Outcome != "applied" {
			t.Errorf("image() outcome = %q, want applied", out.Results[0].Outcome)
		}
		if len(out.Results[0].Filters) != 2 ||
			out.Results[0].Filters[0].Outcome != "applied" ||
			out.Results[0].Filters[1].Outcome != "declined" {
			t.Errorf("nested outcomes = %+v, want blur applied and rotate declined", out.Results[0].Filters)
		}
		if out.Results[1].Outcome != "applied" {
			t.Errorf("quality outcome = %q, want applied", out.Results[1].Outcome)
		}
		if !strings.Contains(out.Note, "3 applied") || !strings.Contains(out.Note, "1 recognised") {
			t.Errorf("note %q does not summarise the result", out.Note)
		}
	})

	t.Run("the server reports a different release", func(t *testing.T) {
		served.Store(`{"version":"v1.9.6","filters":[{"name":"image","processed":true},{"name":"quality","processed":true}]}`)

		if err := callTool(t, s, "verify_imagor_url", args, &out); err != nil {
			t.Fatalf("verify_imagor_url: %v", err)
		}
		if out.Server.Version != "v1.9.6" || !out.Server.ReportsVersion {
			t.Errorf("server = %+v, want the reported release", out.Server)
		}
		if !strings.Contains(out.Note, "v1.9.6") || !strings.Contains(out.Note, "may differ") {
			t.Errorf("note %q does not warn that the reference describes another release", out.Note)
		}
	})

	t.Run("the server reports outcomes but no version", func(t *testing.T) {
		served.Store(`{"filters":[{"name":"image","processed":true},{"name":"quality","processed":true}]}`)

		if err := callTool(t, s, "verify_imagor_url", args, &out); err != nil {
			t.Fatalf("verify_imagor_url: %v", err)
		}
		if out.Server.ReportsVersion {
			t.Errorf("server = %+v, want no version reported", out.Server)
		}
		if !out.Server.ReportsFilters {
			t.Errorf("server = %+v, want the filters field reported", out.Server)
		}
		if !strings.Contains(out.Note, "did not report a version") {
			t.Errorf("note %q does not say the version is missing", out.Note)
		}
		if !strings.Contains(out.Note, "predates the release that reports one") {
			t.Errorf("note %q does not say what the server predates", out.Note)
		}
		// The catalogue's release is not what the server predates: a server can
		// be exactly that release and still predate the field.
		if strings.Contains(out.Note, "predates "+out.Server.CatalogueVersion) {
			t.Errorf("note %q claims the server predates the release the catalogue describes", out.Note)
		}
	})

	t.Run("the server reports no outcomes", func(t *testing.T) {
		served.Store(`{"format":"jpeg","width":400}`)

		if err := callTool(t, s, "verify_imagor_url", args, &out); err != nil {
			t.Fatalf("verify_imagor_url: %v", err)
		}
		if out.Verified {
			t.Fatalf("a server with no filter report must not verify")
		}
		for _, r := range out.Results {
			if r.Outcome != "unknown" {
				t.Errorf("outcome = %q, want unknown: absence is not failure", r.Outcome)
			}
		}
		if out.Server.ReportsFilters {
			t.Errorf("server = %+v, want no filters field reported", out.Server)
		}
		if !strings.Contains(out.Note, "unknown rather than wrong") {
			t.Errorf("note %q does not explain the absence", out.Note)
		}
	})

	t.Run("the server errors", func(t *testing.T) {
		served.Store(`{"message":"boom"}`)
		srvErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"message":"boom","status":500}`)
		}))
		defer srvErr.Close()
		s500 := connect(t, map[string]string{"IMAGOR_UNSAFE": "1", "IMAGOR_BASE_URL": srvErr.URL})

		if err := callTool(t, s500, "verify_imagor_url", args, &out); err != nil {
			t.Fatalf("verify_imagor_url: %v", err)
		}
		if out.Verified || out.Status != 500 {
			t.Fatalf("expected verified=false status=500, got verified=%v status=%d", out.Verified, out.Status)
		}
		if !strings.Contains(out.Note, "500") {
			t.Errorf("note %q does not report the status", out.Note)
		}
	})
}

// The grammar description must be derived from imagor's own types, so it
// cannot drift from what the tools actually build.
func TestDescribeReflectsImagorParams(t *testing.T) {
	s := connect(t, unsafeEnv)

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

// Excess arguments warn rather than refuse. Measured on imagor v1.9.7,
// grayscale(1) still applies while blur(1,2,3) is ignored, so refusing both
// blocked a filter that works. The warning carries the uncertainty instead.
func TestExcessArgumentsWarnRatherThanRefuse(t *testing.T) {
	s := connect(t, map[string]string{
		"IMAGOR_SECRET":          "mysecret",
		"IMAGOR_SIGNER_TYPE":     "sha256",
		"IMAGOR_SIGNER_TRUNCATE": "40",
	})

	for _, name := range []string{"blur", "grayscale"} {
		t.Run(name, func(t *testing.T) {
			var out struct {
				URL      string   `json:"url"`
				Warnings []string `json:"warnings"`
			}
			err := callTool(t, s, "build_imagor_url", map[string]any{
				"image":   "example.com/a.jpg",
				"width":   200,
				"filters": []map[string]any{{"name": name, "args": "1,2,3"}},
			}, &out)
			if err != nil {
				t.Fatalf("%s with extra arguments must not be refused: %v", name, err)
			}
			if out.URL == "" {
				t.Error("expected a URL alongside the warning")
			}
			if len(out.Warnings) == 0 {
				t.Fatalf("%s with extra arguments should warn", name)
			}
			if !strings.Contains(out.Warnings[0], name) {
				t.Errorf("warning should name the filter, got %q", out.Warnings[0])
			}
		})
	}
}

// Regression: grayscale(1) was refused before the strictness was graduated,
// even though imagor applies it.
func TestOverSpecifiedButWorkingCompositionBuilds(t *testing.T) {
	s := connect(t, map[string]string{
		"IMAGOR_SECRET":          "mysecret",
		"IMAGOR_SIGNER_TYPE":     "sha256",
		"IMAGOR_SIGNER_TRUNCATE": "40",
	})

	var out struct {
		URL      string   `json:"url"`
		Warnings []string `json:"warnings"`
	}
	if err := callTool(t, s, "build_imagor_url", map[string]any{
		"image":   "example.com/a.jpg",
		"width":   200,
		"filters": []map[string]any{{"name": "grayscale", "args": "1"}},
	}, &out); err != nil {
		t.Fatalf("grayscale(1) works on imagor and must not be refused: %v", err)
	}
	if !strings.Contains(out.URL, "grayscale(1)") {
		t.Errorf("expected the filter in the URL, got %q", out.URL)
	}
	if len(out.Warnings) != 1 {
		t.Errorf("expected exactly one warning, got %v", out.Warnings)
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
	s := connect(t, unsafeEnv)

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
	s := connect(t, unsafeEnv)

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
