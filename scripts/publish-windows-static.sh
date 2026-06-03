#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_DIR="${ROOT_DIR}/deploy/windows-static"
BUILD_DIR="${ROOT_DIR}/tmp/publish-windows-static"
SERVER_BIN="${BUILD_DIR}/gva-server.exe"
PUSH=0
COMMIT_MESSAGE=""

log() {
  printf '[publish-windows-static] %s\n' "$*"
}

fail() {
  printf '[publish-windows-static] ERROR: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'USAGE'
Usage: ./scripts/publish-windows-static.sh [--push] [--message "commit message"]

Builds a Windows static deployment tree under deploy/windows-static.

Options:
  --push              git add -f, commit, and push generated artifacts to main
  --message MESSAGE   commit message used with --push
  -h, --help          show this help
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --push)
      PUSH=1
      shift
      ;;
    --message|-m)
      [[ $# -ge 2 ]] || fail "--message requires a value"
      COMMIT_MESSAGE="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
done

need_command() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

copy_dir() {
  local source="$1"
  local destination="$2"
  [[ -d "${source}" ]] || fail "missing directory: ${source}"
  mkdir -p "$(dirname "${destination}")"
  cp -R "${source}" "${destination}"
}

need_command git
need_command go
need_command npm

CURRENT_BRANCH="$(git -C "${ROOT_DIR}" branch --show-current)"
[[ -n "${CURRENT_BRANCH}" ]] || fail "could not determine current git branch"
if [[ "${PUSH}" == "1" && "${CURRENT_BRANCH}" != "main" ]]; then
  fail "--push must be run from main; current branch is ${CURRENT_BRANCH}"
fi

GIT_COMMIT="$(git -C "${ROOT_DIR}" rev-parse --short HEAD 2>/dev/null || true)"
BUILD_TIME="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
VERSION="${GIT_COMMIT:-local}"
if [[ -z "${COMMIT_MESSAGE}" ]]; then
  COMMIT_MESSAGE="build: publish Windows static artifacts ${VERSION}"
fi

log "Installing web dependencies when needed"
pushd "${ROOT_DIR}/web" >/dev/null
if [[ ! -d node_modules || ! -f node_modules/.static-deps-stamp || package.json -nt node_modules/.static-deps-stamp || package-lock.json -nt node_modules/.static-deps-stamp ]]; then
  npm install --prefer-offline --no-audit --no-fund
  touch node_modules/.static-deps-stamp
else
  log "web dependency cache hit"
fi

log "Building web/dist with npm run build:static"
npm run build:static
popd >/dev/null

[[ -f "${ROOT_DIR}/web/dist/index.html" ]] || fail "web/dist/index.html was not generated"

log "Cross-compiling Windows server binary"
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"
pushd "${ROOT_DIR}/server" >/dev/null
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-X main.Version=${VERSION}" -o "${SERVER_BIN}" .
popd >/dev/null
[[ -f "${SERVER_BIN}" ]] || fail "server binary was not generated: ${SERVER_BIN}"

log "Refreshing ${DEPLOY_DIR}"
rm -rf "${DEPLOY_DIR}"
mkdir -p \
  "${DEPLOY_DIR}/scripts" \
  "${DEPLOY_DIR}/server/uploads/file" \
  "${DEPLOY_DIR}/web" \
  "${DEPLOY_DIR}/tmp/static-runtime/bin"

cp "${ROOT_DIR}/scripts/static-up.ps1" "${DEPLOY_DIR}/scripts/static-up.ps1"
cp "${ROOT_DIR}/scripts/static-down.ps1" "${DEPLOY_DIR}/scripts/static-down.ps1"
cp "${ROOT_DIR}/server/config.local.example.yaml" "${DEPLOY_DIR}/server/config.local.example.yaml"
copy_dir "${ROOT_DIR}/server/resource" "${DEPLOY_DIR}/server/resource"
copy_dir "${ROOT_DIR}/web/dist" "${DEPLOY_DIR}/web/dist"
cp "${SERVER_BIN}" "${DEPLOY_DIR}/tmp/static-runtime/bin/gva-server.exe"
touch "${DEPLOY_DIR}/server/uploads/.gitkeep" "${DEPLOY_DIR}/server/uploads/file/.gitkeep"

cat > "${DEPLOY_DIR}/BUILD_INFO.txt" <<INFO
git_commit=${GIT_COMMIT:-unknown}
git_branch=${CURRENT_BRANCH}
built_at_utc=${BUILD_TIME}
builder_os=$(uname -s)
builder_arch=$(uname -m)
go_version=$(go version)
node_version=$(node -v)
npm_version=$(npm -v)
INFO

log "Prebuilt Windows static package is ready:"
log "  ${DEPLOY_DIR}"
log "  ${DEPLOY_DIR}/web/dist/index.html"
log "  ${DEPLOY_DIR}/tmp/static-runtime/bin/gva-server.exe"

if [[ "${PUSH}" != "1" ]]; then
  log "Skipping git push. Run with --push to commit and push artifacts."
  exit 0
fi

log "Staging generated artifacts"
git -C "${ROOT_DIR}" add .gitignore readme-win.md scripts/install-windows.ps1 scripts/win-lowmem-deploy.ps1 scripts/publish-windows-static.sh
git -C "${ROOT_DIR}" add -f deploy/windows-static

if git -C "${ROOT_DIR}" diff --cached --quiet; then
  log "No staged changes to commit"
else
  git -C "${ROOT_DIR}" commit -m "${COMMIT_MESSAGE}"
fi

log "Pushing ${CURRENT_BRANCH} to origin"
git -C "${ROOT_DIR}" push origin "${CURRENT_BRANCH}"
