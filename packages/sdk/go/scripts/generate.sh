#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PACKAGE_DIR="${ROOT_DIR}/packages/sdk/go"
SPEC_PATH="${ROOT_DIR}/packages/server/openapi.json"
GENERATOR="openapi-generator-cli"
GEN_DIR="${PACKAGE_DIR}/gen"

# Ensure the OpenAPI spec is fresh
pnpm --filter @agate/server generate:openapi

# Clean previous generation artifacts
rm -rf "${GEN_DIR}"
mkdir -p "${GEN_DIR}"

# Generate Go client + models
pnpm exec ${GENERATOR} generate \
  -i "${SPEC_PATH}" \
  -g go \
  -o "${GEN_DIR}" \
  --additional-properties=packageName=agateclient,isGoSubmodule=true,enumClassPrefix=true

# Remove unused scaffolding
rm -f "${GEN_DIR}/go.mod" "${GEN_DIR}/go.sum" "${GEN_DIR}/git_push.sh"
rm -rf "${GEN_DIR}/test"

# Tidy Go module to pull dependencies discovered by the generator
(cd "${PACKAGE_DIR}" && go mod tidy)
