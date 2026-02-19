#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

fail=0

list_files() {
  local glob="$1"
  if command -v rg >/dev/null 2>&1; then
    rg --files -g "${glob}"
  else
    find . -type f -name "${glob}" -print | sed 's#^\./##'
  fi
}

search_eol_issues() {
  local pattern="$1"
  shift
  if command -v rg >/dev/null 2>&1; then
    rg -n --color=never "${pattern}" "$@" || true
  else
    grep -HnE "${pattern}" "$@" || true
  fi
}

collect_issues_for_files() {
  local pattern="$1"
  shift
  local out=""
  local f
  for f in "$@"; do
    [[ -f "${f}" ]] || continue
    local res
    res="$(search_eol_issues "${pattern}" "${f}")"
    if [[ -n "${res}" ]]; then
      out+="${res}"$'\n'
    fi
  done
  printf '%s' "${out}"
}

echo "== Format Check: Go =="
go_unfmt="$(gofmt -l .)"
if [[ -n "${go_unfmt}" ]]; then
  echo "FAIL: gofmt found unformatted files:"
  printf '%s\n' "${go_unfmt}"
  fail=1
else
  echo "OK: gofmt"
fi

echo
echo "== Format Check: Shell =="
if ! command -v shfmt >/dev/null 2>&1; then
  echo "FAIL: shfmt is not installed"
  echo "Install with: go install mvdan.cc/sh/v3/cmd/shfmt@latest"
  fail=1
else
  shell_files="$(list_files '*.sh')"
  if [[ -n "${shell_files}" ]]; then
    mapfile -t shell_files_arr < <(printf '%s\n' "${shell_files}")
    if ! shfmt -d "${shell_files_arr[@]}"; then
      echo "FAIL: shfmt found issues"
      fail=1
    else
      echo "OK: shfmt"
    fi
  else
    echo "OK: no shell files"
  fi
fi

echo
echo "== Format Check: Docs/Text =="
text_ext_globs=(
  "*.yml"
  "*.yaml"
  "*.json"
  "*.toml"
  "*.env"
  "*.txt"
)
text_files=()
for g in "${text_ext_globs[@]}"; do
  while IFS= read -r f; do
    text_files+=("$f")
  done < <(list_files "${g}")
done

if [[ "${#text_files[@]}" -gt 0 ]]; then
  mapfile -t text_files_unique < <(printf '%s\n' "${text_files[@]}" | sort -u)
  text_with_trailing_ws="$(collect_issues_for_files '[[:blank:]]+$' "${text_files_unique[@]}")"
  if [[ -n "${text_with_trailing_ws}" ]]; then
    echo "FAIL: trailing whitespace found in docs/text files:"
    printf '%s\n' "${text_with_trailing_ws}"
    fail=1
  else
    echo "OK: no trailing whitespace in docs/text files"
  fi
else
  echo "OK: no docs/text files"
fi

md_files="$(list_files '*.md')"
if [[ -n "${md_files}" ]]; then
  # Allow Markdown hard line breaks (exactly two spaces at EOL).
  # Flag tabs at EOL and 3+ spaces at EOL.
  mapfile -t md_files_arr < <(printf '%s\n' "${md_files}")
  md_bad_spaces="$(collect_issues_for_files ' {3,}$' "${md_files_arr[@]}")"
  md_bad_tabs="$(collect_issues_for_files $'\t+$' "${md_files_arr[@]}")"
  md_bad_eol="${md_bad_spaces}"
  if [[ -n "${md_bad_tabs}" ]]; then
    md_bad_eol+="${md_bad_tabs}"$'\n'
  fi
  if [[ -n "${md_bad_eol}" ]]; then
    echo "FAIL: markdown EOL formatting issues found:"
    printf '%s\n' "${md_bad_eol}"
    fail=1
  else
    echo "OK: markdown EOL formatting"
  fi
else
  echo "OK: no markdown files"
fi

echo
if [[ "${fail}" -ne 0 ]]; then
  echo "Format check failed"
  exit 1
fi
echo "Format check passed"
