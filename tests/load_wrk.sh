#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/_lib.sh"

require_cmd wrk
require_cmd ps

LISTEN_PORT="${LISTEN_PORT:-18089}"
export RATE_LIMIT_RPS="${RATE_LIMIT_RPS:-10000}"
export RATE_LIMIT_BURST="${RATE_LIMIT_BURST:-20000}"
export PPROF_ENABLED="${PPROF_ENABLED:-0}"
export PPROF_LISTEN="${PPROF_LISTEN:-127.0.0.1:6060}"

start_server

BASE_URL="$(base_url)"
TARGET_PID="$(cat "${PID_FILE}")"

METRICS_DIR="${METRICS_DIR:-${SCRIPT_DIR}/../log}"
mkdir -p "${METRICS_DIR}"
rotate_logs "${METRICS_DIR}" "metrics-" 7
METRICS_FILE="${METRICS_DIR}/metrics-$(date +%Y%m%d-%H%M%S).log"

METRICS_INTERVAL_SEC="${METRICS_INTERVAL_SEC:-5}"
METRICS_SAMPLES="${METRICS_SAMPLES:-0}"

metrics_loop() {
  if [[ "${METRICS_SAMPLES}" -le 0 ]]; then
    return 0
  fi
  for _ in $(seq 1 "${METRICS_SAMPLES}"); do
    if ! ps -p "${TARGET_PID}" >/dev/null 2>&1; then
      return 0
    fi
    ps -o pid,rss,vsz,pcpu,pmem,etime,cmd -p "${TARGET_PID}" | tee -a "${METRICS_FILE}"
    sleep "${METRICS_INTERVAL_SEC}"
  done
}

threads="${THREADS:-12}"
conns="${CONNS:-200}"
duration="${DURATION:-5m}"

metrics_loop &
METRICS_PID=$!

wrk -t"${threads}" -c"${conns}" -d"${duration}" -s "${SCRIPT_DIR}/wrk_status.lua" \
  "${BASE_URL}/sepa-qr?name=Example%20GmbH&iban=DE12500105170648489890&bic=INGDDEFFXXX&amount=49.90"

if [[ -n "${METRICS_PID:-}" ]]; then
  wait "${METRICS_PID}" >/dev/null 2>&1 || true
fi

if [[ "${PPROF_HEAP:-0}" == "1" ]]; then
  mkdir -p "${METRICS_DIR}"
  rotate_logs "${METRICS_DIR}" "pprof-heap-" 7
  out="${METRICS_DIR}/pprof-heap-$(date +%Y%m%d-%H%M%S).pb.gz"
  curl -sS "http://${PPROF_LISTEN}/debug/pprof/heap" -o "${out}"
  echo "Saved heap profile to ${out}"
  if command -v go >/dev/null 2>&1; then
    go tool pprof -top -unit=bytes "${out}" || true
  fi
fi
