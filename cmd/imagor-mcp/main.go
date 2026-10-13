// Command imagor-mcp is an MCP server that exposes imagor's image pipeline to
// AI agents as typed, validated tools.
//
// It does not upload, store or render images. It constructs imagor paths and,
// when configured with a signing secret, signs them. Every path is validated
// by round-tripping through imagor's own parser before it is returned.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cshum/imagor-template/internal/tools"
)

// version is the release tag, injected by -ldflags "-X main.version=...".
var version = ""

// resolveVersion reports which build this is. Release binaries carry the tag in
// version. A `go install ...@vX.Y.Z` build gets no ldflags, so it falls back to
// the module version the toolchain records, and a working tree build to "dev".
func resolveVersion() string {
	if version != "" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}

func main() {
	v := resolveVersion()
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-version", "--version", "version":
			fmt.Println(v)
			return
		case "-h", "-help", "--help", "help":
			fmt.Fprintf(os.Stderr, `imagor-mcp %s

An MCP server exposing imagor's image pipeline as typed tools.

Usage:
  imagor-mcp            run the server on stdio (spawned by an MCP client)
  imagor-mcp version    print the version

Environment:
  IMAGOR_SECRET            signing secret, required unless IMAGOR_UNSAFE is set
  IMAGOR_SIGNER_TYPE       sha1 (default), sha256 or sha512
  IMAGOR_SIGNER_TRUNCATE   signature length in characters
  IMAGOR_BASE_URL          optional imagor origin, reported in tool output

Example MCP client configuration:

  {
    "mcpServers": {
      "imagor": {
        "command": "imagor-mcp",
        "env": {
          "IMAGOR_SECRET": "your-secret-here",
          "IMAGOR_SIGNER_TYPE": "sha256",
          "IMAGOR_SIGNER_TRUNCATE": "40"
        }
      }
    }
  }
`, v)
			return
		}
	}

	cfg := tools.ConfigFromEnv()
	mode, err := cfg.Mode()
	if err != nil {
		log.Fatalf("imagor-mcp: %v", err)
	}
	if _, err := cfg.Signer(); err != nil {
		log.Fatalf("imagor-mcp: %v", err)
	}
	if mode == tools.ModeUnsafe {
		fmt.Fprintln(os.Stderr, "imagor-mcp: running in UNSAFE mode; emitted URLs carry no signature")
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "imagor-mcp",
		Version: v,
	}, &mcp.ServerOptions{
		// A client that reads only the tool list never learns the order a path
		// is written in or how overlays are placed, and imagor answers 200 when
		// a filter does not apply, so a composition can be wrong and look fine.
		Instructions: "Builds and verifies imagor image URLs. Read the " + tools.SkillURI +
			" resource before composing anything: it carries the path grammar, the placement " +
			"rules and a working method that do not fit in a tool description. Use " +
			"verify_imagor_url to check a composition, since a request succeeds even when a " +
			"filter did nothing.",
	})

	tools.Register(server, cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("imagor-mcp: %v", err)
	}
}
