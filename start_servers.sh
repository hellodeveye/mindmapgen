#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HTTP_PORT="${HTTP_PORT:-8080}"
SSE_ADDR="${MCP_ADDR:-}"

usage() {
  cat <<USAGE
Usage: ${0##*/} [--http-port PORT] [--sse-addr HOST:PORT]

Environment overrides:
  HTTP_PORT                  Default HTTP server port (8080)
  MCP_ADDR                   Default SSE server address (":8082")
  MCP_PORT                   Alternative way to set SSE port
  MCP_BASE_URL               Optional base URL for SSE endpoint metadata
  MCP_BASE_PATH              Optional path prefix for SSE endpoints (default "/mcp")
  MCP_SSE_ENDPOINT           Optional SSE stream path (default "/sse")
  MCP_MESSAGE_ENDPOINT       Optional message path (default "/message")
  MCP_KEEP_ALIVE             Set to true/1 to enable SSE keep-alives
  MCP_KEEP_ALIVE_INTERVAL    Duration string for keep-alive interval (e.g. "15s")
USAGE
}

while (($#)); do
  case "$1" in
    --http-port)
      HTTP_PORT="$2"
      shift 2
      ;;
    --sse-addr)
      SSE_ADDR="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ -z "$SSE_ADDR" ]]; then
  if [[ -n "${MCP_PORT:-}" ]]; then
    SSE_ADDR=":${MCP_PORT}"
  else
    SSE_ADDR=":8082"
  fi
fi

HTTP_CMD=(go run . -port "$HTTP_PORT")
SSE_CMD=(go run ./cmd/mcp-server -addr "$SSE_ADDR")

[[ -n "${MCP_BASE_URL:-}" ]] && SSE_CMD+=(-base-url "$MCP_BASE_URL")
[[ -n "${MCP_BASE_PATH:-}" ]] && SSE_CMD+=(-base-path "$MCP_BASE_PATH")
[[ -n "${MCP_SSE_ENDPOINT:-}" ]] && SSE_CMD+=(-sse-endpoint "$MCP_SSE_ENDPOINT")
[[ -n "${MCP_MESSAGE_ENDPOINT:-}" ]] && SSE_CMD+=(-message-endpoint "$MCP_MESSAGE_ENDPOINT")

enable_keep_alive=false
if [[ -n "${MCP_KEEP_ALIVE_INTERVAL:-}" ]]; then
  enable_keep_alive=true
fi
if [[ -n "${MCP_KEEP_ALIVE:-}" ]]; then
  lower_keep_alive="$(printf '%s' "${MCP_KEEP_ALIVE}" | tr '[:upper:]' '[:lower:]')"
  case "$lower_keep_alive" in
    1|true|yes|on)
      enable_keep_alive=true
      ;;
  esac
fi

if [[ "$enable_keep_alive" == true ]]; then
  SSE_CMD+=(-keep-alive=true)
  if [[ -n "${MCP_KEEP_ALIVE_INTERVAL:-}" ]]; then
    SSE_CMD+=(-keep-alive-interval "$MCP_KEEP_ALIVE_INTERVAL")
  fi
fi

PIDS=()
cleanup() {
  for pid in "${PIDS[@]}"; do
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
    fi
  done
}

wait_remaining() {
  for pid in "${PIDS[@]}"; do
    wait "$pid" 2>/dev/null || true
  done
}

terminate() {
  cleanup
  wait_remaining
  exit 1
}
trap terminate INT TERM

cd "$ROOT_DIR"

echo "Starting HTTP server on port $HTTP_PORT"
"${HTTP_CMD[@]}" &
PIDS+=($!)

echo "Starting MCP SSE server on $SSE_ADDR"
"${SSE_CMD[@]}" &
PIDS+=($!)

status=0
for pid in "${PIDS[@]}"; do
  if wait "$pid"; then
    continue
  else
    status=$?
    break
  fi
done

trap - INT TERM

if [[ $status -ne 0 ]]; then
  cleanup
fi

wait_remaining

exit $status
