#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="${ROOT_DIR}/.bin"
BIN="${BIN_DIR}/sepaqx"
LOG_DIR="${ROOT_DIR}/log"
TMP_DIR=""
PID_FILE=""
LOG_FILE=""

cleanup() {
  if [[ -n "${PID_FILE}" && -f "${PID_FILE}" ]]; then
    kill "$(cat "${PID_FILE}")" >/dev/null 2>&1 || true
  fi
  if [[ -n "${TMP_DIR}" ]]; then
    rm -rf "${TMP_DIR}"
  fi
}
trap cleanup EXIT

build_if_needed() {
  if [[ ! -x "${BIN}" ]]; then
    echo "Building binary (missing)..."
    mkdir -p "${BIN_DIR}"
    (cd "${ROOT_DIR}" && go build -o "${BIN}" .)
    return
  fi
  if find "${ROOT_DIR}" -name '*.go' -newer "${BIN}" | grep -q .; then
    echo "Building binary (source changed)..."
    mkdir -p "${BIN_DIR}"
    (cd "${ROOT_DIR}" && go build -o "${BIN}" .)
  fi
}

start_server() {
  build_if_needed

  TMP_DIR="$(mktemp -d)"
  PID_FILE="${TMP_DIR}/sepaqx.pid"
  mkdir -p "${LOG_DIR}"
  rotate_logs "${LOG_DIR}" "sepaqx-" 7
  LOG_FILE="${LOG_DIR}/sepaqx-$(date +%Y%m%d-%H%M%S).log"

  export LISTEN_IP=127.0.0.1
  export LISTEN_PORT="${LISTEN_PORT:-18089}"
  export TLS_ENABLED=false
  export ACCESS_LOG=false

  "${BIN}" >"${LOG_FILE}" 2>&1 &
  echo $! >"${PID_FILE}"

  for _ in $(seq 1 20); do
    if curl -sS "http://127.0.0.1:${LISTEN_PORT}/health" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.1
  done
  echo "Server did not start. Log:"
  cat "${LOG_FILE}"
  return 1
}

base_url() {
  echo "http://127.0.0.1:${LISTEN_PORT}"
}

require_cmd() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "Missing required command: ${cmd}"
    exit 1
  fi
}

rotate_logs() {
  local dir="$1"
  local prefix="$2"
  local keep="$3"
  if ! command -v ls >/dev/null 2>&1; then
    return 0
  fi
  local files
  files=$(ls -1t "${dir}/${prefix}"* 2>/dev/null || true)
  if [[ -z "${files}" ]]; then
    return 0
  fi
  local count=0
  local f
  while IFS= read -r f; do
    count=$((count + 1))
    if [[ "${count}" -gt "${keep}" ]]; then
      rm -f "${f}" || true
    fi
  done <<<"${files}"
}
