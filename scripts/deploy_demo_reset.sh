#!/usr/bin/env bash
set -euo pipefail

# Demo deployment script for the cloud server.
# It discards local repository changes on the server, pulls the latest GitHub
# code, resets PostgreSQL, rebuilds tables through the Go server, writes
# sanitized demo data, and keeps the new service running with nohup.

GIT_BRANCH="${GIT_BRANCH:-main}"
APP_MODE="${APP_MODE:-postgres}"
DATABASE_URL="${DATABASE_URL:-postgres://opentab:opentab123@localhost:5432/opentab?sslmode=disable}"
HOST="${HOST:-0.0.0.0}"
PORT="${PORT:-8080}"
AI_SERVICE_BASE_URL="${AI_SERVICE_BASE_URL:-http://127.0.0.1:8081}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

echo "Pulling latest code from origin/${GIT_BRANCH}..."
git fetch origin "${GIT_BRANCH}"
git reset --hard "origin/${GIT_BRANCH}"
chmod +x scripts/*.sh

echo "Stopping old OpenTab server processes..."
pkill -f "go run ./cmd/server" || true
pkill -f "/tmp/go-build.*/exe/server" || true
pkill -f "/root/.cache/go-build.*/exe/server" || true
pkill -f "opentab-server" || true
sleep 1

echo "Resetting PostgreSQL database..."
OPENTAB_DB_RESET=true ./scripts/init_postgres_linux.sh

echo "Running tests..."
go test ./...

echo "Starting server on ${HOST}:${PORT}..."
APP_MODE="${APP_MODE}" \
DATABASE_URL="${DATABASE_URL}" \
HOST="${HOST}" \
PORT="${PORT}" \
AI_SERVICE_BASE_URL="${AI_SERVICE_BASE_URL}" \
nohup go run ./cmd/server > server.out.log 2> server.err.log &

echo "Waiting for server health..."
for i in {1..30}; do
  if curl -fsS "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
    echo "Server health check passed."
    break
  fi
  if [[ "${i}" == "30" ]]; then
    echo "Server health check failed."
    tail -n 80 server.err.log || true
    exit 1
  fi
  sleep 1
done

echo "Writing sanitized demo data..."
./scripts/reset_demo_data.sh

echo "Deployment finished."
echo "Health:"
curl -fsS "http://127.0.0.1:${PORT}/health"
echo
echo "Logs:"
echo "  tail -f ${ROOT_DIR}/server.out.log"
echo "  tail -f ${ROOT_DIR}/server.err.log"
