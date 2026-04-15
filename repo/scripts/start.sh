#!/usr/bin/env bash
# start.sh — single-command application launcher
# Usage: ./scripts/start.sh
# Brings up MySQL, runs migrations, seeds data, then starts the backend API and frontend.
# Requires: Docker + docker-compose

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_DIR="${REPO_ROOT}/deploy/docker"
ENV_FILE="${REPO_ROOT}/.env"

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

docker-compose \
  -f docker-compose.yml \
  --env-file "${ENV_FILE}" \
  up --build
