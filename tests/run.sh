#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cmd="${1:-all}"
shift || true

case "${cmd}" in
all)
  "${SCRIPT_DIR}/api_matrix.sh" "$@"
  "${SCRIPT_DIR}/load.sh" "$@"
  ;;
matrix)
  "${SCRIPT_DIR}/api_matrix.sh" "$@"
  ;;
load)
  "${SCRIPT_DIR}/load.sh" "$@"
  ;;
load-wrk)
  "${SCRIPT_DIR}/load_wrk.sh" "$@"
  ;;
load-hey)
  "${SCRIPT_DIR}/load_hey.sh" "$@"
  ;;
*)
  echo "Usage: tests/run.sh [all|matrix|load|load-wrk|load-hey]"
  exit 1
  ;;
esac
