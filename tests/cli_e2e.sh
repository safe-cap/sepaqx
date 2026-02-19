#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/_lib.sh"

build_if_needed

failures=0
total=0

expect_ok() {
  local code="$1"
  local label="$2"
  total=$((total + 1))
  if [[ "${code}" -ne 0 ]]; then
    echo "FAIL: ${label} expected exit 0, got ${code}"
    failures=$((failures + 1))
  else
    echo "OK: ${label}"
  fi
}

expect_fail() {
  local code="$1"
  local label="$2"
  total=$((total + 1))
  if [[ "${code}" -eq 0 ]]; then
    echo "FAIL: ${label} expected non-zero exit, got 0"
    failures=$((failures + 1))
  else
    echo "OK: ${label}"
  fi
}

expect_file_nonempty() {
  local path="$1"
  local label="$2"
  total=$((total + 1))
  if [[ -s "${path}" ]]; then
    echo "OK: ${label}"
  else
    echo "FAIL: ${label} missing/empty file: ${path}"
    failures=$((failures + 1))
  fi
}

expect_contains() {
  local text="$1"
  local pattern="$2"
  local label="$3"
  total=$((total + 1))
  if printf "%s" "${text}" | grep -q "${pattern}"; then
    echo "OK: ${label}"
  else
    echo "FAIL: ${label} missing pattern: ${pattern}"
    failures=$((failures + 1))
  fi
}

TMP_DIR="$(mktemp -d)"
out_png="${TMP_DIR}/one.png"
out_json="${TMP_DIR}/one.json"
batch_json="${TMP_DIR}/batch.json"
batch_out="${TMP_DIR}/batch-out.json"
batch_png_dir="${TMP_DIR}/png-out"
batch_png_json="${TMP_DIR}/batch-png.json"

echo "CLI single png"
set +e
"${BIN}" generate \
  --name "Example GmbH" \
  --iban "DE12500105170648489890" \
  --bic "INGDDEFFXXX" \
  --amount "49.90" \
  --out "${out_png}"
code=$?
set -e
expect_ok "${code}" "single png"
expect_file_nonempty "${out_png}" "single png file"

echo "CLI single payload"
set +e
payload="$("${BIN}" generate \
  --name "Example GmbH" \
  --iban "DE12500105170648489890" \
  --bic "INGDDEFFXXX" \
  --amount "49.90" \
  --format payload)"
code=$?
set -e
expect_ok "${code}" "single payload"
expect_contains "${payload}" "BCD" "payload contains BCD"
expect_contains "${payload}" "EUR49.90" "payload contains amount"

echo "CLI single json"
set +e
json_out="$("${BIN}" generate \
  --name "Example GmbH" \
  --iban "DE12500105170648489890" \
  --bic "INGDDEFFXXX" \
  --amount "49.90" \
  --format json)"
code=$?
set -e
expect_ok "${code}" "single json"
expect_contains "${json_out}" "\"ok\":true" "single json ok"
expect_contains "${json_out}" "\"amount_cents\":4990" "single json amount_cents"

echo "CLI unsupported scheme"
set +e
"${BIN}" generate \
  --scheme "pix" \
  --name "Example GmbH" \
  --iban "DE12500105170648489890" \
  --bic "INGDDEFFXXX" \
  --amount "49.90" \
  --out "${TMP_DIR}/bad.png" >/dev/null 2>&1
code=$?
set -e
expect_fail "${code}" "unsupported scheme"

echo "CLI explicit amount format"
set +e
payload_fmt="$("${BIN}" generate \
  --name "Example GmbH" \
  --iban "DE12500105170648489890" \
  --bic "INGDDEFFXXX" \
  --amount "49,90" \
  --amount-format "eur_comma" \
  --format payload)"
code=$?
set -e
expect_ok "${code}" "amount format eur_comma"
expect_contains "${payload_fmt}" "EUR49.90" "amount format normalized payload"

echo "CLI unsupported amount format"
set +e
"${BIN}" generate \
  --name "Example GmbH" \
  --iban "DE12500105170648489890" \
  --bic "INGDDEFFXXX" \
  --amount "49.90" \
  --amount-format "custom_profile" \
  --out "${TMP_DIR}/bad-format.png" >/dev/null 2>&1
code=$?
set -e
expect_fail "${code}" "unsupported amount format"

cat >"${batch_json}" <<EOF
[
  {"name":"Example GmbH","iban":"DE12500105170648489890","bic":"INGDDEFFXXX","amount":"49.90"},
  {"name":"","iban":"DE12500105170648489890","bic":"INGDDEFFXXX","amount":"49.90"}
]
EOF

echo "CLI batch json partial failure"
set +e
"${BIN}" generate --input "${batch_json}" --format json >"${batch_out}" 2>/dev/null
code=$?
set -e
expect_fail "${code}" "batch json partial failure exit"
batch_text="$(cat "${batch_out}")"
expect_contains "${batch_text}" "\"ok\":false" "batch summary ok=false"
expect_contains "${batch_text}" "\"total\":2" "batch summary total"
expect_contains "${batch_text}" "\"failed\":1" "batch summary failed"

echo "CLI batch png no stdout"
set +e
"${BIN}" generate --input "${batch_json}" --format png --out - >/dev/null 2>&1
code=$?
set -e
expect_fail "${code}" "batch png out dash unsupported"

echo "CLI batch png success"
cat >"${batch_json}" <<EOF
[
  {"name":"Example GmbH","iban":"DE12500105170648489890","bic":"INGDDEFFXXX","amount":"49.90"}
]
EOF
set +e
"${BIN}" generate --input "${batch_json}" --format png --out "${batch_png_dir}" >"${batch_png_json}"
code=$?
set -e
expect_ok "${code}" "batch png success exit"
expect_file_nonempty "${batch_png_dir}/sepa-qr-1.png" "batch png output file"
batch_png_text="$(cat "${batch_png_json}")"
expect_contains "${batch_png_text}" "\"ok\":true" "batch png summary ok=true"
expect_contains "${batch_png_text}" "\"succeeded\":1" "batch png summary succeeded"

if [[ "${TESTS_HIDE_TOTAL:-0}" != "1" ]]; then
  echo
  echo "Total: ${total}, Failures: ${failures}"
fi
if [[ "${failures}" -ne 0 ]]; then
  exit 1
fi
