#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PID_FILE="$ROOT_DIR/.mindmap_servers.pid"

usage() {
  cat <<USAGE
Usage: ${0##*/} [start|restart] [OPTIONS]

Commands:
  start                  Launch the HTTP and MCP services (default)
  restart                Stop any running services started by this script, then launch again

Options:
  --http-port PORT       HTTP server port (env HTTP_PORT, default 8080)
  --mcp-addr HOST:PORT   Address for MCP HTTP server (env MCP_ADDR or MCP_PORT, default :8082)
  -h, --help             Show this help message and exit

Environment overrides:
  HTTP_PORT, MCP_ADDR, MCP_PORT, MCP_BASE_PATH,
  MCP_KEEP_ALIVE, MCP_KEEP_ALIVE_INTERVAL
USAGE
}

command="start"
if (($# > 0)); then
  case "$1" in
    start|restart)
      command="$1"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --*)
      command="start"
      ;;
    *)
      echo "Unknown command: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
fi

HTTP_PORT="${HTTP_PORT:-8080}"
MCP_ADDR_VAL="${MCP_ADDR:-}"

while (($#)); do
  case "$1" in
    --http-port)
      HTTP_PORT="$2"
      shift 2
      ;;
    --mcp-addr)
      MCP_ADDR_VAL="$2"
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

if [[ -z "$MCP_ADDR_VAL" ]]; then
  if [[ -n "${MCP_PORT:-}" ]]; then
    MCP_ADDR_VAL=":${MCP_PORT}"
  else
    MCP_ADDR_VAL=":8082"
  fi
fi

HTTP_CMD=(go run . -port "$HTTP_PORT")
MCP_CMD=(go run ./cmd/mcp-server -addr "$MCP_ADDR_VAL")
[[ -n "${MCP_BASE_PATH:-}" ]] && MCP_CMD+=(-base-path "$MCP_BASE_PATH")

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
  MCP_CMD+=(-keep-alive=true)
  if [[ -n "${MCP_KEEP_ALIVE_INTERVAL:-}" ]]; then
    MCP_CMD+=(-keep-alive-interval "$MCP_KEEP_ALIVE_INTERVAL")
  fi
fi

read_pid_file() {
  if [[ -f "$PID_FILE" ]]; then
    mapfile -t existing_pids <"$PID_FILE" || existing_pids=()
  else
    existing_pids=()
  fi
}

process_alive() {
  local pid="$1"
  kill -0 "$pid" 2>/dev/null
}

stop_running() {
  read_pid_file
  if ((${#existing_pids[@]} == 0)); then
    return
  fi
  for pid in "${existing_pids[@]}"; do
    if process_alive "$pid"; then
      echo "Stopping process $pid"
      kill "$pid" 2>/dev/null || true
    fi
  done
  for pid in "${existing_pids[@]}"; do
    if process_alive "$pid"; then
      wait "$pid" 2>/dev/null || true
    fi
  done
  rm -f "$PID_FILE"
}

ensure_not_running() {
  read_pid_file
  for pid in "${existing_pids[@]:-}"; do
    if process_alive "$pid"; then
      echo "Services appear to be running already (PID $pid). Use restart." >&2
      exit 1
    fi
  done
  rm -f "$PID_FILE"
}

PIDS=()
cleanup() {
  for pid in "${PIDS[@]}"; do
    if process_alive "$pid"; then
      kill "$pid" 2>/dev/null || true
    fi
  done
}

wait_remaining() {
  for pid in "${PIDS[@]}"; do
    wait "$pid" 2>/dev/null || true
  done
  rm -f "$PID_FILE"
}

terminate() {
  cleanup
  wait_remaining
  exit 1
}

start_services() {
  ensure_not_running

  trap terminate INT TERM

  cd "$ROOT_DIR"

  echo "Starting HTTP server on port $HTTP_PORT"
  "${HTTP_CMD[@]}" &
  PIDS+=($!)

  echo "Starting MCP HTTP server on $MCP_ADDR_VAL"
  "${MCP_CMD[@]}" &
  PIDS+=($!)

  printf '%s\n' "${PIDS[@]}" >"$PID_FILE"

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
}

case "$command" in
  start)
    start_services
    ;;
  restart)
    stop_running
    start_services
    ;;
  *)
    echo "Unexpected command: $command" >&2
    exit 1
    ;;
esac
