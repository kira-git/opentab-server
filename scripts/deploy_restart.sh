#!/usr/bin/env bash
set -euo pipefail

# Pull latest server code, run tests, stop the old process and start OpenTab server.
# Run this script on the Linux server from the repository root.

APP_MODE="${APP_MODE:-postgres}"
DATABASE_URL="${DATABASE_URL:-postgres://opentab:opentab123@localhost:5432/opentab?sslmode=disable}"
HOST="${HOST:-0.0.0.0}"
PORT="${PORT:-8080}"

echo "Pulling latest code..."
git pull

echo "Running tests..."
go test ./...

echo "Stopping old server processes..."
pkill -f "go run ./cmd/server" || true
pkill -f "/tmp/go-build.*/exe/server" || true
pkill -f "/root/.cache/go-build.*/exe/server" || true
pkill -f "opentab-server" || true

sleep 1

echo "Starting server on ${HOST}:${PORT}..."
APP_MODE="${APP_MODE}" \
DATABASE_URL="${DATABASE_URL}" \
HOST="${HOST}" \
PORT="${PORT}" \
nohup go run ./cmd/server > server.out.log 2> server.err.log &

sleep 2

echo "Checking health..."
curl -fsS "http://127.0.0.1:${PORT}/health"
echo
echo "Server started. Logs:"
echo "  tail -f server.out.log"
echo "  tail -f server.err.log"
