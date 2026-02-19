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
LOAD_MIN_200_RATE="${LOAD_MIN_200_RATE:-100}"

summarize_codes() {
  awk '
    {count[$1]++; total++}
    END {
      ok = count["200"] + 0
      ok_rate = (total > 0 ? (ok * 100.0 / total) : 0)
      for (c in count) {
        printf "%s %d\n", c, count[c]
      }
      printf "__TOTAL__ %d\n", total
      printf "__OK__ %d\n", ok
      printf "__OK_RATE__ %.2f\n", ok_rate
    }' | sort -n
}

print_load_summary() {
  local label="$1"
  local stats="$2"

  local total ok ok_rate
  total="$(printf "%s\n" "${stats}" | awk '$1=="__TOTAL__" {print $2}')"
  ok="$(printf "%s\n" "${stats}" | awk '$1=="__OK__" {print $2}')"
  ok_rate="$(printf "%s\n" "${stats}" | awk '$1=="__OK_RATE__" {print $2}')"

  echo "${label} status counts:"
  printf "%s\n" "${stats}" | awk '$1 !~ /^__/' | sort -n
  echo "${label} summary: total=${total} ok_200=${ok} ok_200_rate=${ok_rate}% threshold=${LOAD_MIN_200_RATE}%"

  awk -v r="${ok_rate}" -v t="${LOAD_MIN_200_RATE}" 'BEGIN{exit !(r+0 >= t+0)}'
}

same_stats="$(
  seq 1 "${REQS}" | xargs -I{} -P "${CONCURRENCY}" \
    curl -sS -o /dev/null -w "%{http_code}\n" \
    "${BASE_URL}/sepa-qr?${same_qs}" | summarize_codes
)"
print_load_summary "same-request" "${same_stats}"

echo "Load test: mixed requests"
mixed_stats="$(
  seq 1 "${REQS}" | xargs -I{} -P "${CONCURRENCY}" \
    bash -c 'i="$1"; qs="name=Example%20GmbH&iban=DE12500105170648489890&bic=INGDDEFFXXX&amount=49.90&remittance_text=Order%20'"'"'"${i}"'"'"'"; curl -sS -o /dev/null -w "%{http_code}\n" "'"${BASE_URL}"'/sepa-qr?${qs}"' _ {} | summarize_codes
)"
print_load_summary "mixed-request" "${mixed_stats}"

echo "Load checks: PASS"

if [[ -n "${METRICS_PID:-}" ]]; then
  wait "${METRICS_PID}" >/dev/null 2>&1 || true
fi
