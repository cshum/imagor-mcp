package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cshum/imagor-mcp/internal/pathgen"
	"github.com/cshum/imagor/imagorpath"
)

// nestedProbePath is the smallest request whose outcome distinguishes 1.9.10
// from everything before it. It composes nothing and renders one pixel.
//
// path:<imagor path> is syntax only from 1.9.10. An older server does not
// recognise the prefix, so the token is read as part of the image name and the
// request fails while parsing the path rather than while loading it. Measured
// against a real 1.9.9:
//
//	1x1/path:1x1/color:red                  400 {"message":"invalid"}
//	fit-in/100x100/path:200x200/color:blue  400 {"message":"invalid"}
//	path:color:red                          400 {"message":"invalid"}
//
// and 200 for the same three against 1.9.10. A plain 200x200/color:blue against
// that same 1.9.9 answered 200, so a 400 here is the syntax and not the server.
const nestedProbePath = "1x1/path:1x1/color:red"

// probeTimeout is separate from verifyTimeout: the probe asks whether the server
// understands a syntax, and a slow image should not be read as an old server.
const probeTimeout = 10 * time.Second

// nestedSupport is the outcome of asking a server about the nested path source.
// Three states, not two, because "could not tell" is neither.
type nestedSupport int

const (
	// nestedUnknown means the probe got an answer that says nothing about the
	// version: a 403 from a signing disagreement, a 5xx from a broken server, a
	// timeout, a refused connection. It must not be reported as an old server.
	nestedUnknown nestedSupport = iota
	// nestedYes means the server accepted the syntax.
	nestedYes
	// nestedNo means the server rejected it while parsing, which is what a
	// release before 1.9.10 does.
	nestedNo
)

// probeNestedPath asks the imagor at BaseURL whether it understands a nested
// path source, which needs imagor 1.9.10 or later.
//
// The answer comes from behaviour rather than a version string, because imagor
// publishes none: it logs its version at startup and no endpoint reports it. The
// probe is therefore the only way to tell, and what it detects is exact — a
// server that predates the syntax cannot parse the token at all.
func (c Config) probeNestedPath(ctx context.Context) (nestedSupport, error) {
	if c.BaseURL == "" {
		return nestedUnknown, fmt.Errorf(
			"IMAGOR_BASE_URL must be set to check what the imagor behind it supports")
	}
	signer, err := c.Signer()
	if err != nil {
		return nestedUnknown, err
	}
	// Built through pathgen like every other request, so the probe is signed the
	// same way and a signing mistake cannot read as a missing feature. Parse and
	// Generate round-trip this path byte-exactly, which a test pins: a
	// normalising rewrite would probe different syntax than it claims to.
	url := c.BaseURL + "/" + pathgen.BuildURL(imagorpath.Parse(nestedProbePath), signer)

	reqCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nestedUnknown, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nestedUnknown, fmt.Errorf("probing %s for nested path support: %w", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))

	switch resp.StatusCode {
	case http.StatusOK:
		return nestedYes, nil
	case http.StatusBadRequest:
		// 400 is what the parser returns for syntax it does not know. Require the
		// body to say so before concluding: a 400 for some other reason would
		// otherwise be read as an old server and send a caller after a version
		// that is not the problem.
		if m := errorMessage(body); m != "" && m != "invalid" {
			return nestedUnknown, fmt.Errorf(
				"probing %s returned 400 %q, which is not the parse refusal this probe "+
					"looks for, so it says nothing about the version", url, m)
		}
		return nestedNo, nil
	default:
		return nestedUnknown, fmt.Errorf(
			"probing %s for nested path support returned %d %s: %s says nothing about the "+
				"version, so this is not reported as an old server",
			url, resp.StatusCode, http.StatusText(resp.StatusCode),
			strings.TrimSpace(string(body)))
	}
}

// NestedPathSupported reports whether the server understands a nested path
// source. An inconclusive probe is an error rather than false, so a caller can
// tell "no" from "could not tell".
func (c Config) NestedPathSupported(ctx context.Context) (bool, error) {
	switch support, err := c.probeNestedPath(ctx); support {
	case nestedYes:
		return true, nil
	case nestedNo:
		return false, nil
	default:
		return false, err
	}
}

// RequireNestedPath returns the error to show a caller whose imagor predates the
// nested path source, and nil when it does not.
//
// The message names the release, because that is the one thing a caller can act
// on. The bare failure is a 400 with the body {"message":"invalid"}, which names
// no part of the path and is indistinguishable from any other parse error.
//
// Only a definite no refuses. An inconclusive probe returns nil: a 403 means a
// signing disagreement and a 5xx means a broken server, and refusing on those
// would replace the error naming the real fault with one naming a version the
// server may already have.
func (c Config) RequireNestedPath(ctx context.Context) error {
	support, _ := c.probeNestedPath(ctx)
	if support != nestedNo {
		return nil
	}
	return c.nestedPathRefusal()
}

// nestedPathRefusal is the message for a server known to predate the syntax.
func (c Config) nestedPathRefusal() error {
	return fmt.Errorf(
		"this needs imagor %s or later: the imagor at %s does not understand a nested path "+
			"source (path:<imagor path>), so a path using one is rejected while parsing rather "+
			"than loaded, as 400 invalid, naming no part of the path. Upgrade that server to %s "+
			"or later, or point IMAGOR_BASE_URL at one that is",
		RequiresImagor, c.BaseURL, RequiresImagor)
}

// errorMessage pulls the message out of an imagor error body, or "" when the
// body is not one.
func errorMessage(body []byte) string {
	var e struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &e) != nil {
		return ""
	}
	return e.Message
}
