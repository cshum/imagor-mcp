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
	cmd.Env = append(os.Environ(), "IMAGOR_SECRET=", "IMAGOR_SIGNER_TYPE=", "IMAGOR_SIGNER_TRUNCATE=")
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
	for _, want := range []string{"build_imagor_url", "build_imagor_path", "validate_imagor_path", "describe_imagor"} {
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
