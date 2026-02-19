#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
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

failures=0
total=0

expect_status() {
  local expected="$1"
  local actual="$2"
  local label="$3"
  total=$((total + 1))
  if [[ "${expected}" != "${actual}" ]]; then
    echo "FAIL: ${label} expected ${expected}, got ${actual}"
    failures=$((failures + 1))
  else
    echo "OK: ${label}"
  fi
}

post_json() {
  local body="$1"
  curl -sS -o /dev/null -w "%{http_code}" \
    -X POST "${BASE_URL}/sepa-qr" \
    -H "Content-Type: application/json" \
    -d "${body}"
}

post_validate() {
  local body="$1"
  curl -sS -w "\n%{http_code}" \
    -X POST "${BASE_URL}/sepa-qr/validate" \
    -H "Content-Type: application/json" \
    -d "${body}"
}

get_query() {
  local qs="$1"
  curl -sS -o /dev/null -w "%{http_code}" \
    "${BASE_URL}/sepa-qr?${qs}"
}

get_headers() {
  local qs="$1"
  curl -sS -I "${BASE_URL}/sepa-qr?${qs}"
}

head_query() {
  local qs="$1"
  curl -sS -o /dev/null -w "%{http_code}" \
    -I "${BASE_URL}/sepa-qr?${qs}"
}

head_bare() {
  curl -sS -o /dev/null -w "%{http_code}" -I "${BASE_URL}/sepa-qr"
}

options_call() {
  curl -sS -o /dev/null -w "%{http_code}" -X OPTIONS "${BASE_URL}/sepa-qr"
}

health_call() {
  curl -sS -o /dev/null -w "%{http_code}" "${BASE_URL}/health"
}

ready_call() {
  curl -sS -o /dev/null -w "%{http_code}" "${BASE_URL}/readyz"
}

ready_body_with_code() {
  curl -sS -w "\n%{http_code}" "${BASE_URL}/readyz"
}

valid_name="Example GmbH"
valid_iban="DE12500105170648489890"
valid_bic="INGDDEFFXXX"
valid_amount="49.90"

base_payload() {
  cat <<EOF
{"name":"${valid_name}","iban":"${valid_iban}","bic":"${valid_bic}","amount":"${valid_amount}"}
EOF
}

payload_with() {
  local extra="$1"
  cat <<EOF
{"name":"${valid_name}","iban":"${valid_iban}","bic":"${valid_bic}","amount":"${valid_amount}",${extra}}
EOF
}

echo "Smoke: /health"
expect_status 200 "$(health_call)" "GET /health"
expect_status 200 "$(ready_call)" "GET /readyz ready"

echo "Options /sepa-qr"
expect_status 204 "$(options_call)" "OPTIONS /sepa-qr"

echo "POST valid base"
expect_status 200 "$(post_json "$(base_payload)")" "POST valid base"

echo "POST with optional fields"
expect_status 200 "$(post_json "$(payload_with "\"purpose\":\"GDDS\"")")" "POST purpose"
expect_status 200 "$(post_json "$(payload_with "\"scheme\":\"epc_sct\"")")" "POST scheme epc_sct"
expect_status 200 "$(post_json "$(payload_with "\"amount\":\"49,90\",\"amount_format\":\"eur_comma\"")")" "POST amount_format eur_comma"
expect_status 200 "$(post_json "$(payload_with "\"remittance_reference\":\"RF18539007547034\"")")" "POST remittance_reference"
expect_status 200 "$(post_json "$(payload_with "\"remittance_text\":\"Order 2026-0001\"")")" "POST remittance_text"
expect_status 200 "$(post_json "$(payload_with "\"information\":\"Invoice 0001\"")")" "POST information"

echo "POST invalid combinations"
expect_status 400 "$(post_json "$(payload_with "\"remittance_reference\":\"RF18539007547034\",\"remittance_text\":\"Both\"")")" "POST remittance both"

echo "POST invalid data"
expect_status 400 "$(post_json "{\"name\":\"\",\"iban\":\"${valid_iban}\",\"bic\":\"${valid_bic}\",\"amount\":\"${valid_amount}\"}")" "POST missing name"
expect_status 400 "$(post_json "{\"name\":\"${valid_name}\",\"iban\":\"\",\"bic\":\"${valid_bic}\",\"amount\":\"${valid_amount}\"}")" "POST missing iban"
expect_status 400 "$(post_json "{\"name\":\"${valid_name}\",\"iban\":\"INVALID\",\"bic\":\"${valid_bic}\",\"amount\":\"${valid_amount}\"}")" "POST invalid iban"
expect_status 400 "$(post_json "{\"name\":\"${valid_name}\",\"iban\":\"${valid_iban}\",\"bic\":\"\",\"amount\":\"${valid_amount}\"}")" "POST missing bic"
expect_status 400 "$(post_json "{\"name\":\"${valid_name}\",\"iban\":\"${valid_iban}\",\"bic\":\"INVALID\",\"amount\":\"${valid_amount}\"}")" "POST invalid bic"
expect_status 400 "$(post_json "{\"name\":\"${valid_name}\",\"iban\":\"${valid_iban}\",\"bic\":\"${valid_bic}\",\"amount\":\"\"}")" "POST missing amount"
expect_status 400 "$(post_json "{\"name\":\"${valid_name}\",\"iban\":\"${valid_iban}\",\"bic\":\"${valid_bic}\",\"amount\":\"0\"}")" "POST amount zero"
expect_status 400 "$(post_json "{\"name\":\"${valid_name}\",\"iban\":\"${valid_iban}\",\"bic\":\"${valid_bic}\",\"amount\":\"-1\"}")" "POST amount negative"
expect_status 400 "$(post_json "{\"name\":\"${valid_name}\",\"iban\":\"${valid_iban}\",\"bic\":\"${valid_bic}\",\"amount\":\"999999999999\"}")" "POST amount too large"
expect_status 400 "$(post_json "{\"name\":\"${valid_name}\",\"iban\":\"${valid_iban}\",\"bic\":\"${valid_bic}\",\"amount\":\"1.234\"}")" "POST amount precision"
expect_status 400 "$(post_json "{\"name\":\"${valid_name}\",\"iban\":\"${valid_iban}\",\"bic\":\"${valid_bic}\",\"amount\":\"abc\"}")" "POST amount non-numeric"
expect_status 400 "$(post_json "{\"scheme\":\"pix\",\"name\":\"${valid_name}\",\"iban\":\"${valid_iban}\",\"bic\":\"${valid_bic}\",\"amount\":\"${valid_amount}\"}")" "POST scheme unsupported"
expect_status 400 "$(post_json "{\"name\":\"${valid_name}\",\"iban\":\"${valid_iban}\",\"bic\":\"${valid_bic}\",\"amount\":\"49,90\",\"amount_format\":\"eur_dot\"}")" "POST amount_format mismatch"
expect_status 400 "$(post_json "{\"name\":\"${valid_name}\",\"iban\":\"${valid_iban}\",\"bic\":\"${valid_bic}\",\"amount\":\"49.90\",\"amount_format\":\"custom_profile\"}")" "POST amount_format unsupported"
expect_status 400 "$(post_json "{\"name\":\"${valid_name}\",\"iban\":\"${valid_iban}\",\"bic\":\"${valid_bic}\",\"amount\":\"1O,5\"}")" "POST amount OCR strict mode"
expect_status 400 "$(post_json "{bad-json}")" "POST invalid json"

echo "Validation API"
resp="$(post_validate "$(base_payload)")"
body="$(printf "%s" "${resp}" | sed '$d')"
code="$(printf "%s" "${resp}" | tail -n 1)"
expect_status 200 "${code}" "POST /sepa-qr/validate valid"
if ! printf "%s" "${body}" | grep -q '"ok":true'; then
  echo "FAIL: validate ok body"
  failures=$((failures + 1))
else
  echo "OK: validate ok body"
fi

resp="$(post_validate "{bad-json}")"
body="$(printf "%s" "${resp}" | sed '$d')"
code="$(printf "%s" "${resp}" | tail -n 1)"
expect_status 400 "${code}" "POST /sepa-qr/validate invalid json"
if ! printf "%s" "${body}" | grep -q '"ok":false'; then
  echo "FAIL: validate invalid json body"
  failures=$((failures + 1))
else
  echo "OK: validate invalid json body"
fi

resp="$(post_validate "{\"name\":\"\",\"iban\":\"${valid_iban}\",\"bic\":\"${valid_bic}\",\"amount\":\"${valid_amount}\"}")"
body="$(printf "%s" "${resp}" | sed '$d')"
code="$(printf "%s" "${resp}" | tail -n 1)"
expect_status 400 "${code}" "POST /sepa-qr/validate invalid input"
if ! printf "%s" "${body}" | grep -q '"ok":false'; then
  echo "FAIL: validate invalid input body"
  failures=$((failures + 1))
else
  echo "OK: validate invalid input body"
fi

echo "Validation rate limit"
cleanup
trap cleanup EXIT
export RATE_LIMIT_RPS=1
export RATE_LIMIT_BURST=1
start_server
BASE_URL="$(base_url)"
post_validate "$(base_payload)" >/dev/null
code="$(post_validate "$(base_payload)" | tail -n 1)"
expect_status 429 "${code}" "POST /sepa-qr/validate rate limited"

echo "Restore rate limit"
cleanup
trap cleanup EXIT
export RATE_LIMIT_RPS=1000
export RATE_LIMIT_BURST=2000
start_server
BASE_URL="$(base_url)"

echo "Readiness strict mode (REQUIRE_KEYS)"
cleanup
trap cleanup EXIT
bad_keys_file="$(mktemp)"
printf '{bad-json}\n' >"${bad_keys_file}"
export KEYS_FILE="${bad_keys_file}"
export REQUIRE_KEYS=true
start_server
BASE_URL="$(base_url)"
resp="$(ready_body_with_code)"
body="$(printf "%s" "${resp}" | sed '$d')"
code="$(printf "%s" "${resp}" | tail -n 1)"
expect_status 503 "${code}" "GET /readyz require keys invalid"
if ! printf "%s" "${body}" | grep -qi "not ready"; then
  echo "FAIL: /readyz require keys invalid body"
  failures=$((failures + 1))
else
  echo "OK: /readyz require keys invalid body"
fi

resp="$(curl -sS -H "Accept: application/json" -w "\n%{http_code}" "${BASE_URL}/readyz")"
body="$(printf "%s" "${resp}" | sed '$d')"
code="$(printf "%s" "${resp}" | tail -n 1)"
expect_status 503 "${code}" "GET /readyz require keys invalid json"
if ! printf "%s" "${body}" | grep -q '"ok":false'; then
  echo "FAIL: /readyz require keys invalid json body"
  failures=$((failures + 1))
else
  echo "OK: /readyz require keys invalid json body"
fi

rm -f "${bad_keys_file}"
unset REQUIRE_KEYS
export KEYS_FILE="${ROOT_DIR}/examples/keys.json.example"
cleanup
trap cleanup EXIT
start_server
BASE_URL="$(base_url)"
expect_status 200 "$(ready_call)" "GET /readyz ready after valid keys"

echo "GET/HEAD with query"
qs="name=Example%20GmbH&iban=${valid_iban}&bic=${valid_bic}&amount=49.90"
expect_status 200 "$(get_query "${qs}")" "GET valid query"
expect_status 200 "$(head_query "${qs}")" "HEAD valid query"
expect_status 200 "$(get_query "${qs}&scheme=epc_sct")" "GET scheme epc_sct"
expect_status 400 "$(get_query "${qs}&amount=50.00")" "GET duplicate amount query"
qs_comma="name=Example%20GmbH&iban=${valid_iban}&bic=${valid_bic}&amount=49,90"
expect_status 200 "$(get_query "${qs_comma}&amount_format=eur_comma")" "GET amount_format eur_comma"
expect_status 400 "$(get_query "${qs_comma}&amount_format=eur_dot")" "GET amount_format mismatch"
expect_status 400 "$(get_query "${qs}&scheme=pix")" "GET scheme unsupported"

echo "HEAD bare should succeed without params"
expect_status 200 "$(head_bare)" "HEAD /sepa-qr"

echo "Auth header with invalid key should fail"
expect_status 401 "$(curl -sS -o /dev/null -w "%{http_code}" -H "Authorization: Bearer invalid" "${BASE_URL}/sepa-qr?${qs}")" "GET invalid auth"

echo "Query API key"
for key in example-api-key-1 example-api-key-2 example-api-key-3 example-api-key-4; do
  qs_key="${qs}&api_key=${key}"
  expect_status 200 "$(get_query "${qs_key}")" "GET with api_key ${key}"

  hdrs="$(get_headers "${qs_key}")"
  if ! printf "%s" "${hdrs}" | grep -qi "content-type: image/png"; then
    echo "FAIL: GET with api_key ${key} content-type"
    failures=$((failures + 1))
  else
    echo "OK: GET with api_key ${key} content-type"
  fi
done

echo "Require API key mode"
cleanup
trap cleanup EXIT
export REQUIRE_API_KEY=true
start_server
BASE_URL="$(base_url)"

expect_status 401 "$(get_query "${qs}")" "GET public require api key"
expect_status 401 "$(head_bare)" "HEAD bare require api key"
expect_status 401 "$(post_json "$(base_payload)")" "POST public require api key"

resp="$(post_validate "$(base_payload)")"
body="$(printf "%s" "${resp}" | sed '$d')"
code="$(printf "%s" "${resp}" | tail -n 1)"
expect_status 401 "${code}" "POST /sepa-qr/validate require api key"
if ! printf "%s" "${body}" | grep -q '"ok":false'; then
  echo "FAIL: validate require api key body"
  failures=$((failures + 1))
else
  echo "OK: validate require api key body"
fi

expect_status 200 "$(curl -sS -o /dev/null -w "%{http_code}" -H "X-API-Key: example-api-key-1" "${BASE_URL}/sepa-qr?${qs}")" "GET with header api key"
expect_status 200 "$(curl -sS -o /dev/null -w "%{http_code}" -H "X-API-Key: example-api-key-1" -H "Content-Type: application/json" -X POST "${BASE_URL}/sepa-qr" -d "$(base_payload)")" "POST with header api key"
resp="$(curl -sS -H "X-API-Key: example-api-key-1" -H "Content-Type: application/json" -w "\n%{http_code}" -X POST "${BASE_URL}/sepa-qr/validate" -d "$(base_payload)")"
body="$(printf "%s" "${resp}" | sed '$d')"
code="$(printf "%s" "${resp}" | tail -n 1)"
expect_status 200 "${code}" "POST /sepa-qr/validate with header api key"
if ! printf "%s" "${body}" | grep -q '"ok":true'; then
  echo "FAIL: validate ok body with api key"
  failures=$((failures + 1))
else
  echo "OK: validate ok body with api key"
fi

cleanup
trap cleanup EXIT
unset REQUIRE_API_KEY
export AMOUNT_LENIENT_OCR=true
start_server
BASE_URL="$(base_url)"

echo "Amount lenient OCR mode"
expect_status 200 "$(post_json "{\"name\":\"${valid_name}\",\"iban\":\"${valid_iban}\",\"bic\":\"${valid_bic}\",\"amount\":\"1O,5\"}")" "POST amount OCR lenient mode"

echo "Error JSON format"
qs_bad="name=Example%20GmbH&iban=${valid_iban}&bic=${valid_bic}&amount=0&format=json"
resp="$(curl -sS -w "\n%{http_code}" "${BASE_URL}/sepa-qr?${qs_bad}")"
body="$(printf "%s" "${resp}" | sed '$d')"
code="$(printf "%s" "${resp}" | tail -n 1)"
expect_status 400 "${code}" "GET error format=json"
if ! printf "%s" "${body}" | grep -q '"error_code"'; then
  echo "FAIL: error format=json body"
  failures=$((failures + 1))
else
  echo "OK: error format=json body"
fi

echo "Error JSON via Accept"
resp="$(curl -sS -H "Accept: application/json" -w "\n%{http_code}" "${BASE_URL}/sepa-qr?${qs_bad}")"
body="$(printf "%s" "${resp}" | sed '$d')"
code="$(printf "%s" "${resp}" | tail -n 1)"
expect_status 400 "${code}" "GET error Accept: application/json"
if ! printf "%s" "${body}" | grep -q '"error_code"'; then
  echo "FAIL: error Accept json body"
  failures=$((failures + 1))
else
  echo "OK: error Accept json body"
fi

if [[ "${TESTS_HIDE_TOTAL:-0}" != "1" ]]; then
  echo
  echo "Total: ${total}, Failures: ${failures}"
fi
if [[ "${failures}" -ne 0 ]]; then
  exit 1
fi
