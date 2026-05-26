#!/usr/bin/env bash
set -euo pipefail

# 关闭低内存静态部署时，只需要处理 static-runtime 下记录的 Go 后端进程。
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PID_DIR="${ROOT_DIR}/tmp/static-runtime/pids"
API_PORT="${API_PORT:-9999}"

# 按 PID 文件停止静态部署服务。
stop_pid() {
  local name="$1"
  local pid_file="$2"
  if [[ ! -f "${pid_file}" ]]; then
    return 0
  fi

  local pid
  pid="$(cat "${pid_file}")"
  if kill -0 "${pid}" 2>/dev/null; then
    printf '[static-down] stopping %s (PID %s)\n' "${name}" "${pid}"
    kill "${pid}" 2>/dev/null || true
  fi
  rm -f "${pid_file}"
}

# PID 文件缺失时兜底清理监听 API_PORT 的进程。
stop_port_listener() {
  local port="$1"
  local pids

  pids="$(lsof -tiTCP:"${port}" -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -z "${pids}" ]]; then
    return 0
  fi

  printf '[static-down] stopping listener on port %s (%s)\n' "${port}" "${pids}"
  kill ${pids} 2>/dev/null || true
}

# 先按脚本记录的 PID 停止，再按端口做兜底清理。
stop_pid server "${PID_DIR}/server.pid"
stop_port_listener "${API_PORT}"
