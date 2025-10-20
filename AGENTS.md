# Repository Guidelines

## Project Structure & Module Organization
- `cmd/mindmapgen` hosts the CLI entry point for turning text into a PNG mind map; use the bundled `input.txt` and `output.png` when smoke-testing.
- `cmd/mcp-server` is the MCP-facing stub—extend this when wiring the generator into external agents.
- `internal/` contains core logic: `parser` for markdown-like syntax, `layout` for node positioning, `drawer` for rendering, `storage` for R2 integration, and `theme` for palette presets.
- `pkg/` exposes reusable building blocks (`server`, `mcp`, `types`) consumed by both the CLI and the HTTP service defined in `main.go`.
- Static assets live in `static/`; deployment manifests live under `k8s/`.

## Build, Test, and Development Commands
- `go run ./cmd/mindmapgen -i examples/map.txt -o artifacts/map.png` generates a map locally; add `-theme` or `-b` as needed.
- `go run .` starts the HTTP server on `:8080`; export the `R2_*` variables first when exercising upload paths.
- `go build ./cmd/mindmapgen` and `go build ./...` must stay clean to keep release pipelines green.
- `go test ./...` is the fast path; add `-race` or `-run Parser` for focused checks during debugging.

## Coding Style & Naming Conventions
- Follow Go defaults: tabs for indentation, `camelCase` for locals, `CamelCase` for exported symbols, and `snake_case` for filenames when pairing with assets.
- Run `go fmt ./...` (or `gofmt -w`) before pushing; pair with `golangci-lint run` if you have it installed locally.
- Keep functions single-purpose; share logic via `internal/` packages instead of cross-importing `cmd/`.

## Testing Guidelines
- Co-locate tests beside implementations (`*_test.go`); mimic existing names like `parser_test.go` and `drawer_test.go` for clarity.
- Exercise new parsing rules with focused unit tests and add golden-style samples when updating rendering.
- Aim for passing `go test ./... -coverprofile=coverage.out`; flag regressions if coverage drops materially around parser/layout code.

## Commit & Pull Request Guidelines
- Use Conventional Commit prefixes seen in history (`feat:`, `fix:`, `refactor:`); include scope when touching a single package.
- Draft PRs with a concise summary, testing notes (`go test ./...` output), and screenshots/Base64 snippets when UI or rendering changes.
- Link Jira/GitHub issues in the description and request review from owners of the affected package before merging.

## Security & Configuration Tips
- Never commit credentials; provide `.env.example` updates instead. R2 access requires `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_ACCESS_KEY_SECRET`, `R2_BUCKET_NAME`, and `R2_DOMAIN`.
- When sharing logs, scrub base64 payloads and signed URLs—prefer reproducer text files under `cmd/mindmapgen/`.
