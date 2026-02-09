#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/_lib.sh"

require_cmd hey
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

conns="${CONCURRENCY:-200}"
duration="${DURATION:-5m}"

metrics_loop &
METRICS_PID=$!

hey_out="$(mktemp)"
hey -z "${duration}" -c "${conns}" \
  "${BASE_URL}/sepa-qr?name=Example%20GmbH&iban=DE12500105170648489890&bic=INGDDEFFXXX&amount=49.90" | tee "${hey_out}"

duration_to_seconds() {
  local d="$1"
  case "${d}" in
  *h) echo $((${d%h} * 3600)) ;;
  *m) echo $((${d%m} * 60)) ;;
  *s) echo $((${d%s})) ;;
  *) echo "${d}" ;;
  esac
}

secs="$(duration_to_seconds "${duration}")"
if [[ "${secs}" -gt 0 ]]; then
  ok_err="$(awk '
    /^\s*\[[0-9]+\]/ {
      gsub(/[\[\]]/,"",$1);
      code=$1;
      count=$2;
      total+=count;
      if (code==200) ok+=count;
    }
    END { if (total>0) printf "%d %d", ok, total }
  ' "${hey_out}")"
  if [[ -n "${ok_err}" ]]; then
    ok="$(echo "${ok_err}" | awk '{print $1}')"
    total="$(echo "${ok_err}" | awk '{print $2}')"
    err=$((total - ok))
    awk -v ok="${ok}" -v err="${err}" -v secs="${secs}" 'BEGIN {
      printf "ok/s: %.2f\n", ok/secs;
      printf "err/s: %.2f\n", err/secs;
    }'
  fi
fi

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

rm -f "${hey_out}" || true
