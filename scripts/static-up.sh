#!/usr/bin/env bash
set -euo pipefail

# 项目根目录和静态部署运行目录，运行产物统一放到 tmp/static-runtime。
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNTIME_DIR="${ROOT_DIR}/tmp/static-runtime"
LOG_DIR="${RUNTIME_DIR}/logs"
PID_DIR="${RUNTIME_DIR}/pids"
BIN_DIR="${RUNTIME_DIR}/bin"
GO_CACHE_DIR="${RUNTIME_DIR}/go-cache"
GO_MOD_CACHE_DIR="${RUNTIME_DIR}/go-mod"
SERVER_DIR="${ROOT_DIR}/server"
WEB_DIR="${ROOT_DIR}/web"
WEB_DIST="${WEB_DIR}/dist"
SERVER_BIN="${BIN_DIR}/gva-server"
WEB_DEPS_STAMP="${WEB_DIR}/node_modules/.static-deps-stamp"
CONFIG_TEMPLATE="${SERVER_DIR}/config.local.example.yaml"
CONFIG_FILE="${SERVER_DIR}/config.local.yaml"

# 低内存静态部署只暴露一个端口：Go 后端同时提供 API 和前端静态页。
API_PORT="${API_PORT:-9999}"
# STATIC_BUILD=0 时复用已有 web/dist，适合 2C2G 目标机。
STATIC_BUILD="${STATIC_BUILD:-1}"
# SERVER_BUILD=0 时复用已有 Go 二进制，适合提前构建后再迁移。
SERVER_BUILD="${SERVER_BUILD:-1}"
SERVER_READY_TIMEOUT="${SERVER_READY_TIMEOUT:-120}"

# 确保日志、PID、二进制和 Go 缓存目录存在。
mkdir -p "${LOG_DIR}" "${PID_DIR}" "${BIN_DIR}" "${GO_CACHE_DIR}" "${GO_MOD_CACHE_DIR}"

# 统一输出脚本日志前缀。
log() {
  printf '[static-up] %s\n' "$*"
}

# 失败时输出错误并终止脚本。
fail() {
  printf '[static-up] %s\n' "$*" >&2
  exit 1
}

# 检查外部命令是否存在，缺失时尽早失败。
require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    fail "missing required command: $1"
  fi
}

# 服务启动失败时输出最近日志，方便定位 MySQL、端口或配置问题。
tail_log_excerpt() {
  local log_file="$1"
  if [[ -f "${log_file}" ]]; then
    printf '[static-up] recent log from %s:\n' "${log_file}" >&2
    tail -n 40 "${log_file}" >&2 || true
  fi
}

# 根据 PID 文件判断进程是否仍在运行。
pid_is_running() {
  local pid_file="$1"
  [[ -f "${pid_file}" ]] && kill -0 "$(cat "${pid_file}")" 2>/dev/null
}

# 清理已经失效的 PID 文件，避免误判服务已启动。
remove_stale_pid() {
  local pid_file="$1"
  if [[ -f "${pid_file}" ]] && ! pid_is_running "${pid_file}"; then
    rm -f "${pid_file}"
  fi
}

# 判断端口是否已被其他进程监听。
port_is_busy() {
  lsof -nP -iTCP:"$1" -sTCP:LISTEN >/dev/null 2>&1
}

# 输出端口占用摘要，用于报错信息。
port_listener_summary() {
  local port="$1"
  lsof -nP -iTCP:"${port}" -sTCP:LISTEN 2>/dev/null | tail -n +2 | awk '{print $1 " pid=" $2 " " $9}' | paste -sd '; ' -
}

# 通过 python3 创建独立会话，避免脚本退出后 Go 服务被当前终端回收。
spawn_detached() {
  local log_file="$1"
  local pid_file="$2"
  local command="$3"
  python3 - "${log_file}" "${pid_file}" "${command}" <<'PY'
import pathlib
import subprocess
import sys

log_path = pathlib.Path(sys.argv[1])
pid_path = pathlib.Path(sys.argv[2])
command = sys.argv[3]

log_path.parent.mkdir(parents=True, exist_ok=True)
with log_path.open("ab") as log_handle:
    process = subprocess.Popen(
        ["bash", "-lc", command],
        stdin=subprocess.DEVNULL,
        stdout=log_handle,
        stderr=subprocess.STDOUT,
        start_new_session=True,
    )
pid_path.write_text(f"{process.pid}\n")
PY
}

# 后端健康检查，确认 API 服务已经可用。
server_health() {
  curl -fsS "http://127.0.0.1:${API_PORT}/health" | grep -qx '"ok"'
}

# 前端首页检查，确认 Go 已经托管 web/dist。
web_health() {
  curl -fsS "http://127.0.0.1:${API_PORT}/" | grep -q '<!doctype html\|<html'
}

# 等待 Go 后端和前端静态页同时可用。
wait_for_process_ready() {
  local pid_file="$1"
  local log_file="$2"
  local index
  for ((index = 1; index <= SERVER_READY_TIMEOUT; index++)); do
    if server_health >/dev/null 2>&1 && web_health >/dev/null 2>&1; then
      return 0
    fi
    if [[ -f "${pid_file}" ]] && ! pid_is_running "${pid_file}"; then
      tail_log_excerpt "${log_file}"
      fail "server exited before becoming ready"
    fi
    sleep 1
  done

  tail_log_excerpt "${log_file}"
  fail "server did not become ready on port ${API_PORT}"
}

# 如果本地配置不存在，复制模板生成 server/config.local.yaml。
ensure_local_config() {
  if [[ -f "${CONFIG_FILE}" ]]; then
    return 0
  fi
  cp "${CONFIG_TEMPLATE}" "${CONFIG_FILE}"
  log "Created server/config.local.yaml from template; edit MySQL settings if needed"
}

# 保持 config.local.yaml 的 system.addr 和脚本健康检查端口一致。
sync_api_port_to_local_config() {
  python3 - "${CONFIG_FILE}" "${API_PORT}" <<'PY'
import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
api_port = sys.argv[2]
lines = path.read_text().splitlines()
patched = []
in_system = False
for line in lines:
    if re.match(r"^system:\s*$", line):
        in_system = True
        patched.append(line)
        continue
    if in_system and re.match(r"^[^\s#].*:\s*$", line):
        in_system = False
    if in_system and re.match(r"^\s*addr:", line):
        patched.append(f"    addr: {api_port}")
        continue
    patched.append(line)
path.write_text("\n".join(patched) + "\n")
PY
}

# 按 package.json 变更判断是否需要重新安装前端依赖。
ensure_web_dependencies() {
  if [[ -d "${WEB_DIR}/node_modules" && -f "${WEB_DEPS_STAMP}" && ! "${WEB_DIR}/package.json" -nt "${WEB_DEPS_STAMP}" ]]; then
    return 0
  fi

  log "Installing web dependencies"
  (
    cd "${WEB_DIR}"
    npm install --prefer-offline --no-audit --no-fund
    touch "${WEB_DEPS_STAMP}"
  )
}

# 构建或复用 web/dist；2C2G 机器建议通过 STATIC_BUILD=0 跳过。
build_static_web() {
  if [[ "${STATIC_BUILD}" == "0" ]]; then
    [[ -f "${WEB_DIST}/index.html" ]] || fail "web/dist/index.html not found; run npm run build:static first"
    log "Reusing existing web/dist"
    return 0
  fi

  require_cmd npm
  ensure_web_dependencies
  log "Building static web assets"
  (
    cd "${WEB_DIR}"
    npm run build:static
  )
}

# 判断 Go 源码、go.mod 或 go.sum 是否比已有二进制更新。
server_sources_changed() {
  if [[ ! -x "${SERVER_BIN}" ]]; then
    return 0
  fi
  if [[ "${SERVER_DIR}/go.mod" -nt "${SERVER_BIN}" || "${SERVER_DIR}/go.sum" -nt "${SERVER_BIN}" ]]; then
    return 0
  fi

  local newer_source
  newer_source="$(find "${SERVER_DIR}" -type f -name '*.go' -newer "${SERVER_BIN}" -print -quit)"
  [[ -n "${newer_source}" ]]
}

# 构建或复用 Go 后端二进制；目标机内存紧张时可用 SERVER_BUILD=0 跳过。
build_server_binary() {
  if [[ "${SERVER_BUILD}" == "0" && -x "${SERVER_BIN}" ]]; then
    log "Reusing existing server binary"
    return 0
  fi
  if [[ "${SERVER_BUILD}" != "0" ]] && ! server_sources_changed; then
    log "Reusing cached server binary"
    return 0
  fi

  require_cmd go
  log "Building server binary"
  (
    cd "${SERVER_DIR}"
    env -u GOROOT GOCACHE="${GO_CACHE_DIR}" GOMODCACHE="${GO_MOD_CACHE_DIR}" go build -o "${SERVER_BIN}" .
  )
}

# 启动单进程静态部署：Go 同时提供后端 API 和前端静态资源。
start_server() {
  local pid_file="${PID_DIR}/server.pid"
  local log_file="${LOG_DIR}/server.log"
  local command

  remove_stale_pid "${pid_file}"

  if pid_is_running "${pid_file}"; then
    log "Server already running (PID $(cat "${pid_file}"))"
    return 0
  fi
  if port_is_busy "${API_PORT}"; then
    if server_health >/dev/null 2>&1 && web_health >/dev/null 2>&1; then
      log "Static deployment already reachable on port ${API_PORT} ($(port_listener_summary "${API_PORT}"))"
      return 0
    fi
    fail "port ${API_PORT} is already in use by another process: $(port_listener_summary "${API_PORT}")"
  fi

  [[ -x "${SERVER_BIN}" ]] || fail "server binary not found at ${SERVER_BIN}"
  [[ -f "${WEB_DIST}/index.html" ]] || fail "web/dist/index.html not found"

  log "Starting server with static web root"
  : > "${log_file}"
  # 注入 GVA_STATIC_ROOT，让后端注册 web/dist 静态路由。
  printf -v command 'cd %q && exec env GVA_CONFIG=config.local.yaml GVA_STATIC_ROOT=%q %q' "${SERVER_DIR}" "${WEB_DIST}" "${SERVER_BIN}"
  spawn_detached "${log_file}" "${pid_file}" "${command}"

  wait_for_process_ready "${pid_file}" "${log_file}"
}

# 输出静态部署访问地址和停止命令。
write_runtime_summary() {
  cat <<EOF

静态部署已启动：
  访问地址: http://127.0.0.1:${API_PORT}/#/login
  后端接口: http://127.0.0.1:${API_PORT}
  健康检查: http://127.0.0.1:${API_PORT}/health
  日志文件: ${LOG_DIR}/server.log

停止服务：
  ./scripts/static-down.sh
EOF
}

# 主流程：检查工具、准备配置、构建静态资源、构建后端并启动服务。
main() {
  require_cmd bash
  require_cmd curl
  require_cmd lsof
  require_cmd python3

  ensure_local_config
  sync_api_port_to_local_config
  build_static_web
  build_server_binary
  start_server
  write_runtime_summary
}

main "$@"
