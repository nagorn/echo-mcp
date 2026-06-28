#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEFAULT_TMP_DIR="${TMPDIR:-/tmp}"
DEFAULT_TMP_DIR="${DEFAULT_TMP_DIR%/}"
CACHE_DIR="${ECHO_MCP_OPENAPI_SMOKE_CACHE_DIR:-${DEFAULT_TMP_DIR}}"
GO_CACHE_DIR="${GOCACHE:-${DEFAULT_TMP_DIR}/echo-mcp-go-cache}"

STRIPE_URL="${ECHO_MCP_STRIPE_OPENAPI_URL:-https://raw.githubusercontent.com/stripe/openapi/master/latest/openapi.spec3.json}"
GITHUB_URL="${ECHO_MCP_GITHUB_OPENAPI_URL:-https://raw.githubusercontent.com/github/rest-api-description/ef4e98d7fcad5ec4476fd73b4d536557524f3c57/descriptions/api.github.com/api.github.com.json}"

STRIPE_FILE="${ECHO_MCP_STRIPE_OPENAPI_FILE:-${CACHE_DIR}/echo-mcp-stripe-latest-openapi.spec3.json}"
GITHUB_FILE="${ECHO_MCP_GITHUB_OPENAPI_FILE:-${CACHE_DIR}/echo-mcp-github-api.openapi.json}"

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "missing required command: ${name}" >&2
    exit 1
  fi
}

fetch_if_missing() {
  local name="$1"
  local url="$2"
  local file="$3"

  if [ -s "${file}" ]; then
    echo "${name}: using cached ${file}"
    return
  fi

  mkdir -p "$(dirname "${file}")"
  echo "${name}: ${file} not found; downloading ${url}"
  if ! curl -fsS "${url}" -o "${file}"; then
    echo "${name}: failed to download ${url}" >&2
    echo "Provide a cached file with ECHO_MCP_${name}_OPENAPI_FILE or restore network access." >&2
    exit 1
  fi
}

operation_count() {
  jq '[.paths[] | to_entries[] | select(.key | IN("get","put","post","delete","options","head","patch","trace"))] | length' "$1"
}

schema_count() {
  jq '(.components.schemas // {}) | length' "$1"
}

print_metadata() {
  local name="$1"
  local url="$2"
  local file="$3"

  echo
  echo "== ${name} OpenAPI metadata =="
  echo "source_url=${url}"
  echo "local_file=${file}"
  echo "sha256=$(shasum -a 256 "${file}" | awk '{print $1}')"
  echo "file_size_bytes=$(wc -c < "${file}" | tr -d ' ')"
  echo "openapi_version=$(jq -r '.openapi' "${file}")"
  echo "operation_count=$(operation_count "${file}")"
  echo "schemas_count=$(schema_count "${file}")"
}

require_command curl
require_command jq
require_command shasum
require_command go

fetch_if_missing STRIPE "${STRIPE_URL}" "${STRIPE_FILE}"
fetch_if_missing GITHUB "${GITHUB_URL}" "${GITHUB_FILE}"

print_metadata "Stripe" "${STRIPE_URL}" "${STRIPE_FILE}"
print_metadata "GitHub REST" "${GITHUB_URL}" "${GITHUB_FILE}"

echo
echo "== Synthetic compatibility corpus =="
(
  cd "${ROOT_DIR}"
  GOCACHE="${GO_CACHE_DIR}" \
    go test -count=1 ./internal/contract -run 'TestOpenAPICompatibilityCorpus'
)

echo
echo "== Real-provider compatibility probes =="
(
  cd "${ROOT_DIR}"
  ECHO_MCP_STRIPE_OPENAPI_FILE="${STRIPE_FILE}" \
  ECHO_MCP_GITHUB_OPENAPI_FILE="${GITHUB_FILE}" \
  GOCACHE="${GO_CACHE_DIR}" \
    go test -count=1 ./internal/contract -run 'TestOpenAPIValidatorAccepts(StripePaymentIntent|GitHubMeta)WhenFixtureAvailable'
)

echo
echo "OpenAPI compatibility smoke completed."
