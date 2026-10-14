package tools

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cshum/imagor-template/internal/pathgen"
	"github.com/cshum/imagor/imagorpath"
)

// The probe path must survive Parse and Generate byte-exactly. If a rewrite
// normalised it, the probe would test different syntax than it claims to, and
// could pass against a server that does not accept the real thing.
func TestNestedProbePathRoundTrips(t *testing.T) {
	if got := pathgen.Build(imagorpath.Parse(nestedProbePath)); got != nestedProbePath {
		t.Fatalf("probe path changed by Parse+Generate\n in: %s\nout: %s", nestedProbePath, got)
	}
}

// A server that answers 200 to the probe supports the syntax.
func TestNestedPathSupportedTrue(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := Config{Unsafe: true, BaseURL: srv.URL}
	ok, err := c.NestedPathSupported(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("200 must report supported")
	}
	if !strings.Contains(got, "path:1x1") {
		t.Fatalf("probe did not carry the nested source: %s", got)
	}
	if !strings.HasPrefix(got, "/unsafe/") {
		t.Fatalf("probe was not routed through the signer route: %s", got)
	}
}

// 400 is what a pre-1.9.10 server returns: it cannot parse the token.
func TestNestedPathSupportedFalseOn400(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"invalid","status":400}`))
	}))
	defer srv.Close()

	c := Config{Unsafe: true, BaseURL: srv.URL}
	ok, err := c.NestedPathSupported(context.Background())
	if err != nil {
		t.Fatalf("400 must be an answer, not an error: %v", err)
	}
	if ok {
		t.Fatal("400 must report unsupported")
	}
}

// Anything else is inconclusive. Reporting a 500 as "old server" would send a
// caller after the wrong problem, so it must be an error and must name the
// status rather than the version.
func TestNestedPathProbeInconclusiveIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := Config{Unsafe: true, BaseURL: srv.URL}
	_, err := c.NestedPathSupported(context.Background())
	if err == nil {
		t.Fatal("a 500 must be an error, not a verdict")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("the error must name the status: %v", err)
	}
	if strings.Contains(err.Error(), RequiresImagor+" or later") {
		t.Fatalf("an inconclusive probe must not be reported as an old server: %v", err)
	}
}

// The refusal names the release, because that is the one thing a caller can act
// on, and it names the server it checked.
func TestRequireNestedPathNamesTheRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := Config{Unsafe: true, BaseURL: srv.URL}
	err := c.RequireNestedPath(context.Background())
	if err == nil {
		t.Fatal("an unsupported server must be refused")
	}
	for _, want := range []string{RequiresImagor, srv.URL} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error must mention %q: %v", want, err)
		}
	}
}

// And it passes silently when the server supports it.
func TestRequireNestedPathPassesOnSupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := Config{Unsafe: true, BaseURL: srv.URL}
	if err := c.RequireNestedPath(context.Background()); err != nil {
		t.Fatalf("a supported server must not be refused: %v", err)
	}
}

// Without a base URL there is nothing to probe, and that is an error rather
// than a false verdict.
func TestNestedPathSupportedNeedsABaseURL(t *testing.T) {
	c := Config{Unsafe: true}
	if _, err := c.NestedPathSupported(context.Background()); err == nil {
		t.Fatal("no base URL must be an error, not false")
	} else if !errors.Is(err, err) || !strings.Contains(err.Error(), "IMAGOR_BASE_URL") {
		t.Fatalf("error must name the missing setting: %v", err)
	}
}

// The 403 case is the one that caught a real bug. A stub answering 403 is a
// signing disagreement, not an old server, and refusing it would replace the
// error naming the signer with one naming a version the server may already have.
// This pins that an inconclusive probe does NOT refuse.
func TestRequireNestedPathDoesNotRefuseOnAnInconclusiveProbe(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
	}{
		{"forbidden", http.StatusForbidden},
		{"not found", http.StatusNotFound},
		{"server error", http.StatusInternalServerError},
		{"not implemented", http.StatusNotImplemented},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()

			c := Config{Unsafe: true, BaseURL: srv.URL}
			if err := c.RequireNestedPath(context.Background()); err != nil {
				t.Fatalf("a %d probe says nothing about the version, so it must not refuse: %v",
					tc.status, err)
			}
			// And it must not claim support either.
			ok, err := c.NestedPathSupported(context.Background())
			if ok {
				t.Fatal("an inconclusive probe must not report support")
			}
			if err == nil {
				t.Fatal("an inconclusive probe must report an error when asked directly")
			}
		})
	}
}

// A 400 whose body is not the parse refusal says nothing about the version
// either, so it must not be read as an old server.
func TestProbeDoesNotReadAnUnrelated400AsAnOldServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"something else entirely","status":400}`))
	}))
	defer srv.Close()

	c := Config{Unsafe: true, BaseURL: srv.URL}
	if err := c.RequireNestedPath(context.Background()); err != nil {
		t.Fatalf("an unrelated 400 must not refuse: %v", err)
	}
	ok, err := c.NestedPathSupported(context.Background())
	if ok || err == nil {
		t.Fatal("an unrelated 400 must be inconclusive, not a verdict")
	}
}
