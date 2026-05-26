#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PID_DIR="${ROOT_DIR}/tmp/dev-runtime/pids"
API_PORT="${API_PORT:-9999}"
WEB_PORT="${WEB_PORT:-8080}"

stop_pid() {
  local name="$1"
  local pid_file="$2"
  if [[ ! -f "${pid_file}" ]]; then
    return 0
  fi

  local pid
  pid="$(cat "${pid_file}")"
  if kill -0 "${pid}" 2>/dev/null; then
    printf '[dev-down] stopping %s (PID %s)\n' "${name}" "${pid}"
    kill "${pid}" 2>/dev/null || true
  fi
  rm -f "${pid_file}"
}

stop_port_listener() {
  local name="$1"
  local port="$2"
  local pids

  pids="$(lsof -tiTCP:"${port}" -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -z "${pids}" ]]; then
    return 0
  fi

  printf '[dev-down] stopping %s listener on port %s (%s)\n' "${name}" "${port}" "${pids}"
  kill ${pids} 2>/dev/null || true
}

stop_pid server "${PID_DIR}/server.pid"
stop_pid web "${PID_DIR}/web.pid"
stop_port_listener server "${API_PORT}"
stop_port_listener web "${WEB_PORT}"
