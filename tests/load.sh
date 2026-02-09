#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/_lib.sh"

require_cmd curl
require_cmd awk
require_cmd xargs

LISTEN_PORT="${LISTEN_PORT:-18089}"
export RATE_LIMIT_RPS=10000
export RATE_LIMIT_BURST=20000

start_server

BASE_URL="$(base_url)"
TARGET_PID="$(cat "${PID_FILE}")"

same_qs="name=Example%20GmbH&iban=DE12500105170648489890&bic=INGDDEFFXXX&amount=49.90"

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

metrics_loop &
METRICS_PID=$!

echo "Load test: same request"
REQS="${REQS:-200}"
CONCURRENCY="${CONCURRENCY:-20}"

seq 1 "${REQS}" | xargs -I{} -P "${CONCURRENCY}" \
  curl -sS -o /dev/null -w "%{http_code}\n" \
  "${BASE_URL}/sepa-qr?${same_qs}" |
  awk '{count[$1]++} END {for (c in count) print c, count[c]}' | sort -n

echo "Load test: mixed requests"
seq 1 "${REQS}" | xargs -I{} -P "${CONCURRENCY}" \
  bash -c 'i="$1"; qs="name=Example%20GmbH&iban=DE12500105170648489890&bic=INGDDEFFXXX&amount=49.90&remittance_text=Order%20'"'"'"${i}"'"'"'"; curl -sS -o /dev/null -w "%{http_code}\n" "'"${BASE_URL}"'/sepa-qr?${qs}"' _ {} |
  awk '{count[$1]++} END {for (c in count) print c, count[c]}' | sort -n

if [[ -n "${METRICS_PID:-}" ]]; then
  wait "${METRICS_PID}" >/dev/null 2>&1 || true
fi
