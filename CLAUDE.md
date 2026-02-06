# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

```bash
# Build all packages
go build ./...

# Run tests
go test ./...

# Run a single test
go test -run TestName ./internal/parser

# Run HTTP server (default port 8080)
go run .
go run . -port 3000

# Run CLI tool
go run ./cmd/mindmapgen -i examples/map.txt -o output.png
go run ./cmd/mindmapgen -raw "mindmap\n  root((Topic))\n    Child" -o output.png -theme dark -layout both

# Run MCP server
go run ./cmd/mcp-server -addr :8082
```

## Architecture Overview

This is a Go application that generates PNG mind map images from indented text or Mermaid mindmap syntax. It provides three interfaces: CLI, HTTP API, and MCP (Model Context Protocol) server.

### Entry Points

- `main.go` - HTTP server with embedded static files, serves web UI and REST API
- `cmd/mindmapgen/main.go` - CLI tool for file-based or raw text mind map generation
- `cmd/mcp-server/main.go` - MCP SSE server for AI tool integration

### Core Pipeline

1. **Parser** (`internal/parser/parser.go`) - Parses indented text or Mermaid mindmap syntax into a tree of `Node` structs. Handles both tab and space indentation, detects format automatically.

2. **Drawer** (`internal/drawer/drawer.go`) - Renders the node tree to PNG using `fogleman/gg`. Supports:
   - Layout directions: `right`, `left`, `both` (balanced split)
   - Standard and sketch (hand-drawn) rendering styles
   - Embedded SimHei font for Chinese text support

3. **Theme System** (`internal/theme/`) - YAML-based theme configuration loaded from embedded `themes/*.yaml` files. Themes define colors, node styles (root/level1/level2/leaf), layout parameters, and optional sketch style settings.

### Key Packages

- `pkg/types/node.go` - Core `Node` struct representing mind map tree nodes
- `pkg/server/server.go` - HTTP mux setup with API routes and static file serving
- `pkg/mcp/server.go` - MCP server implementation using `mark3labs/mcp-go`
- `api/handler.go` - HTTP handlers for `/api/gen` and `/api/themes`
- `internal/storage/r2.go` - Cloudflare R2 storage client for image uploads (optional)

### Environment Variables

For R2 storage (optional, enables `media=url` and MCP image uploads):
- `R2_ACCOUNT_ID`
- `R2_ACCESS_KEY_ID`
- `R2_ACCESS_KEY_SECRET`
- `R2_BUCKET_NAME`
- `R2_PUBLIC_URL`
