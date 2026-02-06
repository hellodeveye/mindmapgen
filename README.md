# mindmapgen

Generate PNG mind maps from indented text or Mermaid mindmap syntax. Supports CLI, HTTP API, and MCP tool usage.

## CLI

Generate a PNG from a file:

```sh
go run ./cmd/mindmapgen -i examples/map.txt -o artifacts/map.png
```

Generate from raw text:

```sh
go run ./cmd/mindmapgen -raw "mindmap\n  root((Main Topic))\n    Subtopic" -o artifacts/map.png
```

Select theme and layout:

```sh
go run ./cmd/mindmapgen -i examples/map.txt -o artifacts/map.png -theme dark -layout both
```

Layout options: `right` (default), `left`, `both`.

## HTTP API

Generate a PNG:

```sh
curl -X POST "http://localhost:8080/api/gen?media=raw&theme=default&layout=both" \
  -H "Content-Type: text/plain" \
  --data-binary $'mindmap\n  root((Main Topic))\n    Subtopic'
```

List themes:

```sh
curl "http://localhost:8080/api/themes"
```

## MCP

Tool: `generate_mindmap`

Parameters:
- `content` (string, required)
- `theme` (string, optional)
- `layout` (string, optional: `right`, `left`, `both`)
