#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

JSON_OUTPUT=0
if [[ "${1:-}" == "--json" ]]; then
  JSON_OUTPUT=1
  shift
fi

cmd="${1:-all}"
shift || true

print_section() {
  local title="$1"
  echo
  echo "== ${title} =="
}

run_suite() {
  local title="$1"
  local script="$2"
  shift 2

  local status="PASS"
  print_section "${title}"
  if ! "${script}" "$@"; then
    status="FAIL"
  fi
  echo "== ${title} Completed: ${status} =="

  SUITE_STATUS="${status}"
}

print_json_summary() {
  local matrix="$1"
  local cli="$2"
  local load="$3"
  local exit_code="$4"
  local cmd_name="$5"
  printf '{"command":"%s","matrix":"%s","cli_e2e":"%s","load":"%s","exit_code":%d}\n' \
    "${cmd_name}" "${matrix}" "${cli}" "${load}" "${exit_code}"
}

status_to_bit() {
  local status="$1"
  local bit="$2"
  if [[ "${status}" == "FAIL" ]]; then
    echo "${bit}"
  else
    echo "0"
  fi
}

SUITE_STATUS="PASS"

case "${cmd}" in
all)
  export TESTS_HIDE_TOTAL=1

  run_suite "Matrix Suite" "${SCRIPT_DIR}/api_matrix.sh" "$@"
  matrix_status="${SUITE_STATUS}"

  run_suite "CLI E2E Suite" "${SCRIPT_DIR}/cli_e2e.sh" "$@"
  cli_status="${SUITE_STATUS}"

  run_suite "Load Suite" "${SCRIPT_DIR}/load.sh" "$@"
  load_status="${SUITE_STATUS}"

  exit_code=0
  exit_code=$((exit_code | $(status_to_bit "${matrix_status}" 1)))
  exit_code=$((exit_code | $(status_to_bit "${cli_status}" 2)))
  exit_code=$((exit_code | $(status_to_bit "${load_status}" 4)))

  echo
  echo "== Test Summary =="
  echo "matrix:   ${matrix_status}"
  echo "cli-e2e:  ${cli_status}"
  echo "load:     ${load_status}"
  if [[ "${exit_code}" -eq 0 ]]; then
    echo "overall:  PASS"
  else
    echo "overall:  FAIL (exit_code=${exit_code})"
  fi

  if [[ "${JSON_OUTPUT}" -eq 1 ]]; then
    print_json_summary "${matrix_status}" "${cli_status}" "${load_status}" "${exit_code}" "all"
  fi
  exit "${exit_code}"
  ;;
matrix)
  export TESTS_HIDE_TOTAL=0
  run_suite "Matrix Suite" "${SCRIPT_DIR}/api_matrix.sh" "$@"
  matrix_status="${SUITE_STATUS}"
  exit_code=$(($(status_to_bit "${matrix_status}" 1)))
  if [[ "${JSON_OUTPUT}" -eq 1 ]]; then
    print_json_summary "${matrix_status}" "SKIP" "SKIP" "${exit_code}" "matrix"
  fi
  exit "${exit_code}"
  ;;
cli-e2e)
  export TESTS_HIDE_TOTAL=0
  run_suite "CLI E2E Suite" "${SCRIPT_DIR}/cli_e2e.sh" "$@"
  cli_status="${SUITE_STATUS}"
  exit_code=$(($(status_to_bit "${cli_status}" 2)))
  if [[ "${JSON_OUTPUT}" -eq 1 ]]; then
    print_json_summary "SKIP" "${cli_status}" "SKIP" "${exit_code}" "cli-e2e"
  fi
  exit "${exit_code}"
  ;;
load)
  export TESTS_HIDE_TOTAL=0
  run_suite "Load Suite" "${SCRIPT_DIR}/load.sh" "$@"
  load_status="${SUITE_STATUS}"
  exit_code=$(($(status_to_bit "${load_status}" 4)))
  if [[ "${JSON_OUTPUT}" -eq 1 ]]; then
    print_json_summary "SKIP" "SKIP" "${load_status}" "${exit_code}" "load"
  fi
  exit "${exit_code}"
  ;;
load-wrk)
  print_section "Load WRK Suite"
  "${SCRIPT_DIR}/load_wrk.sh" "$@"
  ;;
load-hey)
  print_section "Load HEY Suite"
  "${SCRIPT_DIR}/load_hey.sh" "$@"
  ;;
*)
  echo "Usage: tests/run.sh [--json] [all|matrix|cli-e2e|load|load-wrk|load-hey]"
  exit 1
  ;;
esac
