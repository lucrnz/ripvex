#!/usr/bin/env bash

# End-to-end checks for ripvex --max-redirs handling using public httpbin endpoints.
# Requirements:
#   - httpbin.org reachable from the network
#   - ripvex built locally

set -euo pipefail


PATH="$PWD/build:$PATH"
BASE_URL="https://httpbin.org"

# check ripvex  command is available
if ! command -v ripvex &> /dev/null; then
  echo "error: ripvex command not found. Run 'make build' to build it." >&2
  exit 1
fi

# create a temporary directory for the test
tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmpdir}"
}
trap cleanup EXIT

pass=0
fail=0

note() {
  printf "[info] %s\n" "$*"
}

pass_case() {
  printf "[pass] %s\n" "$*"
  pass=$((pass + 1))
}

fail_case() {
  printf "[fail] %s\n" "$*"
  fail=$((fail + 1))
}

run_case() {
  local name="$1"
  local url="$2"
  local max_redirs="$3"
  local expect_status="$4"
  local expect_substr="${5:-}"

  local out_file="${tmpdir}/${name}.out"
  local err_file="${tmpdir}/${name}.err"

  note "running ${name}: url=${url}, max-redirs=${max_redirs}, expect_status=${expect_status}"

  if ripvex \
    --url "${url}" \
    --max-redirs "${max_redirs}" \
    --output "${out_file}" \
    --quiet; then
    status=0
  else
    status=$?
  fi

  if [[ "${status}" -ne "${expect_status}" ]]; then
    fail_case "${name} (status ${status}, expected ${expect_status})"
    printf "stderr:\n" >&2
    sed 's/^/  /' <"${err_file}" >&2 || true
    return
  fi

  if [[ -n "${expect_substr}" ]]; then
    if ! grep -qi -- "${expect_substr}" "${err_file}"; then
      fail_case "${name} (missing expected stderr substring: ${expect_substr})"
      printf "stderr:\n" >&2
      sed 's/^/  /' <"${err_file}" >&2 || true
      return
    fi
  fi

  pass_case "${name}"
}

# Test matrix
run_case "redirect-3-max-5" \
  "${BASE_URL}/redirect/3" \
  5 \
  0

run_case "redirect-5-max-3" \
  "${BASE_URL}/redirect/5" \
  3 \
  1 \
  "stopped after 3 redirects"

run_case "redirect-0-max-0" \
  "${BASE_URL}/redirect/0" \
  0 \
  0

run_case "redirect-1-max-0" \
  "${BASE_URL}/redirect/1" \
  0 \
  1 \
  "stopped after 0 redirects"

run_case "absolute-redirect-2-max-2" \
  "${BASE_URL}/absolute-redirect/2" \
  2 \
  0

echo
echo "Summary: pass=${pass} fail=${fail}"

if [[ "${fail}" -ne 0 ]]; then
  exit 1
fi

exit 0

