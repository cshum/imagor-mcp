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
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cshum/imagor-mcp/internal/tools"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "0.1.0-dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-version", "--version", "version":
			fmt.Println(version)
			return
		case "-h", "--help", "help":
			fmt.Fprintf(os.Stderr, `imagor-mcp %s

An MCP server exposing imagor's image pipeline as typed tools.

Usage:
  imagor-mcp            run the server on stdio (spawned by an MCP client)
  imagor-mcp version    print the version

Environment:
  IMAGOR_SECRET            signing secret. Unset means path-out only.
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
`, version)
			return
		}
	}

	cfg := tools.ConfigFromEnv()
	if _, err := cfg.Signer(); err != nil {
		log.Fatalf("imagor-mcp: %v", err)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "imagor-mcp",
		Version: version,
	}, nil)

	tools.Register(server, cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("imagor-mcp: %v", err)
	}
}
