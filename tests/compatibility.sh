#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=tests/_lib.sh
source "${SCRIPT_DIR}/_lib.sh"

require_cmd curl

LISTEN_PORT="${LISTEN_PORT:-18089}"
export RATE_LIMIT_RPS=1000
export RATE_LIMIT_BURST=2000
export ALLOW_QUERY_API_KEY=true
export AMOUNT_LENIENT_OCR=false
export KEYS_FILE="${ROOT_DIR}/examples/keys.json.example"

start_server
BASE_URL="$(base_url)"

total=0
failures=0

expect_status() {
  local expected="$1"
  local actual="$2"
  local label="$3"
  total=$((total + 1))
  if [[ "${actual}" == "${expected}" ]]; then
    echo "OK: ${label}"
  else
    echo "FAIL: ${label} (expected ${expected}, got ${actual})"
    failures=$((failures + 1))
  fi
}

expect_contains() {
  local haystack="$1"
  local needle="$2"
  local label="$3"
  total=$((total + 1))
  if printf "%s" "${haystack}" | grep -Fq "${needle}"; then
    echo "OK: ${label}"
  else
    echo "FAIL: ${label} (missing ${needle})"
    failures=$((failures + 1))
  fi
}

get_code() {
  local path="$1"
  curl -sS -o /dev/null -w "%{http_code}" "${BASE_URL}${path}"
}

post_validate() {
  local json="$1"
  curl -sS -X POST "${BASE_URL}/sepa-qr/validate" \
    -H "Content-Type: application/json" \
    -d "${json}"
}

echo "Compatibility: stable endpoint behavior"
expect_status 200 "$(get_code "/health")" "GET /health"
expect_status 200 "$(get_code "/healthz")" "GET /healthz"
expect_status 200 "$(get_code "/readyz")" "GET /readyz"
expect_status 200 "$(get_code "/version")" "GET /version"
expect_status 204 "$(curl -sS -o /dev/null -w "%{http_code}" -X OPTIONS "${BASE_URL}/sepa-qr")" "OPTIONS /sepa-qr"

echo "Compatibility: stable generate contract"
base_qs="name=Example%20GmbH&iban=DE12500105170648489890&bic=INGDDEFFXXX&amount=49.90&remittance_reference=INV-1"
expect_status 200 "$(get_code "/sepa-qr?${base_qs}")" "GET /sepa-qr valid"
expect_status 200 "$(curl -sS -o /dev/null -w "%{http_code}" -I "${BASE_URL}/sepa-qr?${base_qs}")" "HEAD /sepa-qr valid"
expect_status 200 "$(curl -sS -o /dev/null -w "%{http_code}" -X POST "${BASE_URL}/sepa-qr" -H "Content-Type: application/json" -d "{\"name\":\"Example GmbH\",\"iban\":\"DE12500105170648489890\",\"bic\":\"INGDDEFFXXX\",\"amount\":\"49.90\",\"remittance_reference\":\"INV-1\"}")" "POST /sepa-qr valid"

echo "Compatibility: stable error JSON shape"
resp="$(curl -sS -X POST "${BASE_URL}/sepa-qr/validate" -H "Content-Type: application/json" -d '{"name":"","iban":"DE12500105170648489890","bic":"INGDDEFFXXX","amount":"49.90"}')"
expect_contains "${resp}" "\"ok\":false" "validate error has ok=false"
expect_contains "${resp}" "\"error_code\"" "validate error has error_code"
expect_contains "${resp}" "\"details\"" "validate error has details"
expect_contains "${resp}" "\"field\"" "validate error has field"
expect_contains "${resp}" "\"request_id\"" "validate error has request_id"

echo "Compatibility: stable validate success shape"
resp="$(post_validate '{"name":"Example GmbH","iban":"DE12500105170648489890","bic":"INGDDEFFXXX","amount":"49.90","remittance_reference":"INV-1"}')"
expect_contains "${resp}" "\"ok\":true" "validate success has ok=true"
expect_contains "${resp}" "\"request_id\"" "validate success has request_id"

if [[ "${TESTS_HIDE_TOTAL:-0}" != "1" ]]; then
  echo
  echo "Total: ${total}, Failures: ${failures}"
fi

if [[ "${failures}" -ne 0 ]]; then
  exit 1
fi
