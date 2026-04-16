#!/usr/bin/env bash
# start.sh — single-command application launcher
# Usage: ./scripts/start.sh
# Brings up MySQL, runs migrations, seeds data, then starts the backend API and frontend.
# Requires: Docker + either `docker-compose` (v1) or `docker compose` (v2 plugin)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_DIR="${REPO_ROOT}/deploy/docker"
ENV_FILE="${REPO_ROOT}/.env"

# Resolve docker compose invocation: prefer v2 (the `docker compose` plugin),
# fall back to v1 (`docker-compose` binary). Anything else is an error.
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
echo "  Starting Service Portal"
echo "══════════════════════════════════════════"
echo "  Frontend: http://localhost:5173"
echo "  Backend:  http://localhost:8080"
echo "  Health:   http://localhost:8080/health"
echo "══════════════════════════════════════════"
echo ""

"${COMPOSE[@]}" \
  -f docker-compose.yml \
  --env-file "${ENV_FILE}" \
  up --build
