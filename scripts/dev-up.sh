#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNTIME_DIR="${ROOT_DIR}/tmp/dev-runtime"
LOG_DIR="${RUNTIME_DIR}/logs"
PID_DIR="${RUNTIME_DIR}/pids"
BIN_DIR="${RUNTIME_DIR}/bin"
GO_CACHE_DIR="${RUNTIME_DIR}/go-cache"
GO_MOD_CACHE_DIR="${RUNTIME_DIR}/go-mod"
SERVER_DIR="${ROOT_DIR}/server"
WEB_DIR="${ROOT_DIR}/web"
SERVER_BIN="${BIN_DIR}/gva-server"
WEB_DEPS_STAMP="${WEB_DIR}/node_modules/.deps-stamp"
CONFIG_TEMPLATE="${SERVER_DIR}/config.local.example.yaml"
CONFIG_FILE="${SERVER_DIR}/config.local.yaml"

API_PORT="${API_PORT:-9999}"
WEB_PORT="${WEB_PORT:-8080}"
WEB_HOST="${WEB_HOST:-127.0.0.1}"
MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-123456a}"
MYSQL_DATABASE="${MYSQL_DATABASE:-amazon_admin}"
MYSQL_SOCKET="${MYSQL_SOCKET:-}"
REDIS_HOST="${REDIS_HOST:-127.0.0.1}"
REDIS_PORT="${REDIS_PORT:-6379}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-123456}"
SERVER_READY_TIMEOUT="${SERVER_READY_TIMEOUT:-120}"
WEB_READY_TIMEOUT="${WEB_READY_TIMEOUT:-120}"

mkdir -p "${LOG_DIR}" "${PID_DIR}" "${BIN_DIR}" "${GO_CACHE_DIR}" "${GO_MOD_CACHE_DIR}"

MYSQL_ACCESS_MODE=""

log() {
  printf '[dev-up] %s\n' "$*"
}

fail() {
  printf '[dev-up] %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    fail "missing required command: $1"
  fi
}

remove_stale_pid() {
  local pid_file="$1"
  if [[ -f "${pid_file}" ]] && ! pid_is_running "${pid_file}"; then
    rm -f "${pid_file}"
  fi
}

tail_log_excerpt() {
  local log_file="$1"
  if [[ -f "${log_file}" ]]; then
    printf '[dev-up] recent log from %s:\n' "${log_file}" >&2
    tail -n 40 "${log_file}" >&2 || true
  fi
}

port_listener_summary() {
  local port="$1"
  lsof -nP -iTCP:"${port}" -sTCP:LISTEN 2>/dev/null | tail -n +2 | awk '{print $1 " pid=" $2 " " $9}' | paste -sd '; ' -
}

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

pid_is_running() {
  local pid_file="$1"
  [[ -f "${pid_file}" ]] && kill -0 "$(cat "${pid_file}")" 2>/dev/null
}

port_is_busy() {
  lsof -nP -iTCP:"$1" -sTCP:LISTEN >/dev/null 2>&1
}

wait_for_command() {
  local message="$1"
  local attempts="$2"
  shift 2
  local index
  for ((index = 1; index <= attempts; index++)); do
    if "$@" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  fail "${message}"
}

wait_for_process_ready() {
  local name="$1"
  local attempts="$2"
  local pid_file="$3"
  local log_file="$4"
  local port="$5"
  shift 5
  local index
  for ((index = 1; index <= attempts; index++)); do
    if "$@" >/dev/null 2>&1; then
      return 0
    fi
    if [[ -f "${pid_file}" ]] && ! pid_is_running "${pid_file}"; then
      tail_log_excerpt "${log_file}"
      fail "${name} exited before becoming ready"
    fi
    sleep 1
  done
  tail_log_excerpt "${log_file}"
  if port_is_busy "${port}"; then
    fail "${name} did not become ready; current listener(s) on port ${port}: $(port_listener_summary "${port}")"
  fi
  fail "${name} did not become ready"
}

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

build_server_binary() {
  if ! server_sources_changed; then
    log "Reusing cached server binary"
    return 0
  fi

  log "Building server binary"
  (
    cd "${SERVER_DIR}"
    env -u GOROOT GOCACHE="${GO_CACHE_DIR}" GOMODCACHE="${GO_MOD_CACHE_DIR}" go build -o "${SERVER_BIN}" .
  )
}

prepare_dev_prerequisites() {
  local web_prepare_log="${LOG_DIR}/web-prepare.log"
  local server_prepare_log="${LOG_DIR}/server-prepare.log"
  local web_pid
  local server_pid
  local status=0

  : > "${web_prepare_log}"
  : > "${server_prepare_log}"

  log "Preparing web dependencies and server binary"
  (
    ensure_web_dependencies
  ) > "${web_prepare_log}" 2>&1 &
  web_pid=$!

  (
    build_server_binary
  ) > "${server_prepare_log}" 2>&1 &
  server_pid=$!

  if ! wait "${web_pid}"; then
    tail_log_excerpt "${web_prepare_log}"
    status=1
  fi

  if ! wait "${server_pid}"; then
    tail_log_excerpt "${server_prepare_log}"
    status=1
  fi

  if [[ "${status}" -ne 0 ]]; then
    fail "failed to prepare development prerequisites"
  fi
}

ensure_formula() {
  local formula="$1"
  if brew list --versions "${formula}" >/dev/null 2>&1; then
    return 0
  fi
  log "Installing Homebrew formula: ${formula}"
  brew install "${formula}"
}

start_service() {
  local formula="$1"
  log "Starting Homebrew service: ${formula}"
  brew services start "${formula}" >/dev/null 2>&1 || true
}

ensure_mysql_ready() {
  if detect_mysql_access_mode; then
    return 0
  fi

  start_service mysql
  wait_for_command "mysql did not become ready" 30 detect_mysql_access_mode
}

detect_mysql_access_mode() {
  resolve_mysql_socket
  if mysqladmin --protocol=TCP -h "${MYSQL_HOST}" -P "${MYSQL_PORT}" -u "${MYSQL_USER}" "-p${MYSQL_PASSWORD}" ping >/dev/null 2>&1; then
    MYSQL_ACCESS_MODE="password"
    return 0
  fi
  if [[ -n "${MYSQL_SOCKET}" ]] && mysqladmin --socket="${MYSQL_SOCKET}" -u "${MYSQL_USER}" ping >/dev/null 2>&1; then
    MYSQL_ACCESS_MODE="socket-no-password"
    return 0
  fi
  if mysqladmin --protocol=TCP -h "${MYSQL_HOST}" -P "${MYSQL_PORT}" -u "${MYSQL_USER}" ping >/dev/null 2>&1; then
    MYSQL_ACCESS_MODE="tcp-no-password"
    return 0
  fi
  return 1
}

mysql_exec() {
  case "${MYSQL_ACCESS_MODE}" in
    password)
      mysql --protocol=TCP -h "${MYSQL_HOST}" -P "${MYSQL_PORT}" -u "${MYSQL_USER}" "-p${MYSQL_PASSWORD}" "$@"
      ;;
    socket-no-password)
      if [[ -n "${MYSQL_SOCKET}" ]]; then
        mysql --socket="${MYSQL_SOCKET}" -u "${MYSQL_USER}" "$@"
      else
        mysql -u "${MYSQL_USER}" "$@"
      fi
      ;;
    tcp-no-password)
      mysql --protocol=TCP -h "${MYSQL_HOST}" -P "${MYSQL_PORT}" -u "${MYSQL_USER}" "$@"
      ;;
    *)
      fail "mysql access mode is not initialized"
      ;;
  esac
}

resolve_mysql_socket() {
  if [[ -n "${MYSQL_SOCKET}" && -S "${MYSQL_SOCKET}" ]]; then
    return 0
  fi
  MYSQL_SOCKET=""
  if command -v mysql_config >/dev/null 2>&1; then
    local socket_path
    socket_path="$(mysql_config --socket 2>/dev/null || true)"
    if [[ -n "${socket_path}" && -S "${socket_path}" ]]; then
      MYSQL_SOCKET="${socket_path}"
      return 0
    fi
  fi
  local candidate
  for candidate in /tmp/mysql.sock /opt/homebrew/var/mysql/mysql.sock /usr/local/var/mysql/mysql.sock; do
    if [[ -S "${candidate}" ]]; then
      MYSQL_SOCKET="${candidate}"
      return 0
    fi
  done
}

ensure_mysql_password() {
  if ! detect_mysql_access_mode; then
    local socket_hint="${MYSQL_SOCKET:-not-found}"
    local listener_hint
    listener_hint="$(port_listener_summary "${MYSQL_PORT}")"
    fail "unable to connect to MySQL as ${MYSQL_USER} (listener: ${listener_hint:-none}; socket: ${socket_hint}). Check MYSQL_PASSWORD / MYSQL_SOCKET."
  fi
  if [[ "${MYSQL_ACCESS_MODE}" == "password" ]]; then
    return 0
  fi

  log "Configuring MySQL root password for local development"
  mysql_exec -e "ALTER USER 'root'@'localhost' IDENTIFIED BY '${MYSQL_PASSWORD}';" >/dev/null 2>&1 || true
  mysql_exec -e "CREATE USER IF NOT EXISTS 'root'@'127.0.0.1' IDENTIFIED BY '${MYSQL_PASSWORD}';" >/dev/null 2>&1 || true
  mysql_exec -e "ALTER USER 'root'@'127.0.0.1' IDENTIFIED BY '${MYSQL_PASSWORD}';" >/dev/null 2>&1 || true
  mysql_exec -e "GRANT ALL PRIVILEGES ON *.* TO 'root'@'127.0.0.1' WITH GRANT OPTION; FLUSH PRIVILEGES;" >/dev/null 2>&1 || true

  MYSQL_ACCESS_MODE=""
  if ! detect_mysql_access_mode || [[ "${MYSQL_ACCESS_MODE}" != "password" ]]; then
    fail "failed to set MySQL root password to the expected development default"
  fi
}

redis_ping() {
  redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" ping | grep -qx 'PONG'
}

ensure_local_config() {
  if [[ -f "${CONFIG_FILE}" ]]; then
    return 0
  fi
  cp "${CONFIG_TEMPLATE}" "${CONFIG_FILE}"
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

config_uses_redis() {
  local source_file="${CONFIG_FILE}"
  if [[ ! -f "${source_file}" ]]; then
    source_file="${CONFIG_TEMPLATE}"
  fi
  awk '
    /^[[:space:]]*use-redis:/ {
      value=$2
      gsub(/#.*/, "", value)
      gsub(/[[:space:]]+/, "", value)
      print tolower(value)
      exit
    }
  ' "${source_file}" | grep -qx 'true'
}

redis_listener_is_ready() {
  local listeners
  listeners="$(lsof -nP -iTCP:"${REDIS_PORT}" -sTCP:LISTEN 2>/dev/null | tail -n +2 | awk '{print $1}')"
  [[ -n "${listeners}" ]] && echo "${listeners}" | grep -Eq '^(redis|redis-server|redis-ser)$'
}

ensure_redis_ready() {
  if ! config_uses_redis; then
    log "Skipping Redis startup because local config has use-redis: false"
    return 0
  fi

  require_cmd redis-cli
  ensure_formula redis
  start_service redis

  if redis_ping >/dev/null 2>&1; then
    return 0
  fi
  if redis_listener_is_ready; then
    log "Redis listener already present on port ${REDIS_PORT} ($(port_listener_summary "${REDIS_PORT}")), continuing"
    return 0
  fi
  wait_for_command "redis did not become ready" 30 redis_ping
}

write_runtime_summary() {
  cat <<EOF

项目已启动：
  前端: http://127.0.0.1:${WEB_PORT}
  登录页: http://127.0.0.1:${WEB_PORT}/#/login
  物流页: http://127.0.0.1:${WEB_PORT}/#/amazon/logisticsQuote
  后端: http://127.0.0.1:${API_PORT}
  健康检查: http://127.0.0.1:${API_PORT}/health

默认账号：
  用户名: admin
  密码: ${ADMIN_PASSWORD}
EOF
}

server_health() {
  curl -fsS "http://127.0.0.1:${API_PORT}/health" | grep -qx '"ok"'
}

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
    if server_health >/dev/null 2>&1; then
      log "Server already reachable on port ${API_PORT} ($(port_listener_summary "${API_PORT}"))"
      return 0
    fi
    fail "server port ${API_PORT} is already in use by another process: $(port_listener_summary "${API_PORT}")"
  fi

  [[ -x "${SERVER_BIN}" ]] || fail "server binary not found at ${SERVER_BIN}; build step did not complete"

  command=$(cat <<EOF
cd "${SERVER_DIR}"
env GVA_CONFIG=config.local.yaml "${SERVER_BIN}"
EOF
)

  log "Starting gin-vue-admin server"
  : > "${log_file}"
  spawn_detached "${log_file}" "${pid_file}" "${command}"
}

check_need_init() {
  curl -fsS -X POST "http://127.0.0.1:${API_PORT}/init/checkdb"
}

init_db() {
  local payload
  payload=$(cat <<EOF
{"adminPassword":"${ADMIN_PASSWORD}","dbType":"mysql","host":"${MYSQL_HOST}","port":"${MYSQL_PORT}","userName":"${MYSQL_USER}","password":"${MYSQL_PASSWORD}","dbName":"${MYSQL_DATABASE}"}
EOF
)

  log "Initializing GVA database"
  curl -fsS \
    -H 'Content-Type: application/json' \
    -X POST \
    -d "${payload}" \
    "http://127.0.0.1:${API_PORT}/init/initdb" >/dev/null
}

ensure_server_initialized() {
  local check_response
  check_response="$(check_need_init)"
  if [[ "${check_response}" == *'"needInit":true'* ]]; then
    init_db
    wait_for_command "server did not recover after initdb" "${SERVER_READY_TIMEOUT}" server_health
  fi
}

ensure_web_dependencies() {
  if [[ -d "${WEB_DIR}/node_modules" && -f "${WEB_DEPS_STAMP}" && ! "${WEB_DIR}/package.json" -nt "${WEB_DEPS_STAMP}" && ! "${WEB_DIR}/package-lock.json" -nt "${WEB_DEPS_STAMP}" ]]; then
    return 0
  fi
  log "Installing web dependencies"
  (
    cd "${WEB_DIR}"
    npm install --prefer-offline --no-audit --no-fund
    touch "${WEB_DEPS_STAMP}"
  )
}

start_web() {
  local pid_file="${PID_DIR}/web.pid"
  local log_file="${LOG_DIR}/web.log"
  local command
  local vite_bin="${WEB_DIR}/node_modules/.bin/vite"

  remove_stale_pid "${pid_file}"

  if pid_is_running "${pid_file}"; then
    log "Web already running (PID $(cat "${pid_file}"))"
    return 0
  fi
  if port_is_busy "${WEB_PORT}"; then
    if web_health >/dev/null 2>&1; then
      log "Web already reachable on port ${WEB_PORT} ($(port_listener_summary "${WEB_PORT}"))"
      return 0
    fi
    fail "web port ${WEB_PORT} is already in use by another process: $(port_listener_summary "${WEB_PORT}")"
  fi

  [[ -x "${vite_bin}" ]] || fail "vite executable not found at ${vite_bin}; run npm install in web/"

command=$(cat <<EOF
cd "${WEB_DIR}"
env BROWSER=none VITE_AUTO_OPEN=false "${vite_bin}" --host "${WEB_HOST}" --port "${WEB_PORT}" --mode development
EOF
)

  log "Starting web dev server"
  : > "${log_file}"
  spawn_detached "${log_file}" "${pid_file}" "${command}"
}

web_health() {
  curl -fsS "http://127.0.0.1:${WEB_PORT}" >/dev/null
}

main() {
  require_cmd brew
  require_cmd curl
  require_cmd go
  require_cmd npm
  require_cmd python3
  require_cmd lsof

  ensure_local_config
  sync_api_port_to_local_config
  ensure_formula mysql
  require_cmd mysql
  require_cmd mysqladmin

  ensure_mysql_ready
  ensure_redis_ready
  ensure_mysql_password

  prepare_dev_prerequisites

  start_server
  start_web

  wait_for_process_ready "server" "${SERVER_READY_TIMEOUT}" "${PID_DIR}/server.pid" "${LOG_DIR}/server.log" "${API_PORT}" server_health
  ensure_server_initialized
  wait_for_process_ready "web" "${WEB_READY_TIMEOUT}" "${PID_DIR}/web.pid" "${LOG_DIR}/web.log" "${WEB_PORT}" web_health

  write_runtime_summary
}

main "$@"
