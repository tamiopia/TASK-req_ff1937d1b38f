#!/usr/bin/env bash
# run-tests.sh — single-command test runner
# Usage: ./scripts/run-tests.sh
# Runs: backend unit+integration tests, then Playwright e2e tests
# Requires: Docker + either `docker-compose` (v1) or `docker compose` (v2 plugin)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_DIR="${REPO_ROOT}/deploy/docker"
ENV_FILE="${REPO_ROOT}/.env"

if docker compose version >/dev/null 2>&1; then
  COMPOSE=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE=(docker-compose)
else
  echo "error: neither 'docker compose' nor 'docker-compose' is available" >&2
  exit 1
fi

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "[setup] .env not found — copying from .env.example"
  cp "${REPO_ROOT}/.env.example" "${ENV_FILE}"
fi

cd "${COMPOSE_DIR}"

echo ""
echo "══════════════════════════════════════════"
echo "  Step 1/3 — Build images"
echo "══════════════════════════════════════════"
"${COMPOSE[@]}" \
  -f docker-compose.yml \
  -f docker-compose.test.yml \
  --env-file "${ENV_FILE}" \
  build --parallel

echo ""
echo "══════════════════════════════════════════"
echo "  Step 2/3 — Backend unit + integration tests"
echo "══════════════════════════════════════════"
"${COMPOSE[@]}" \
  -f docker-compose.yml \
  -f docker-compose.test.yml \
  --env-file "${ENV_FILE}" \
  run --rm backend-test

echo ""
echo "══════════════════════════════════════════"
echo "  Step 3/3 — Playwright e2e tests"
echo "══════════════════════════════════════════"
"${COMPOSE[@]}" \
  -f docker-compose.yml \
  -f docker-compose.test.yml \
  --env-file "${ENV_FILE}" \
  run --rm frontend-test

echo ""
echo "✓ All tests passed."
