#!/usr/bin/env bash
set -euo pipefail

# Reset dirty test records and rewrite sanitized demo data.
# The application tables must already exist. Start the Go server once after a
# fresh database reset so GORM can AutoMigrate and seed base users/tabs first.

DB_NAME="${OPENTAB_DB_NAME:-opentab}"
DB_USER="${OPENTAB_DB_USER:-opentab}"
DB_HOST="${OPENTAB_DB_HOST:-localhost}"
DB_PORT="${OPENTAB_DB_PORT:-5432}"
DB_PASSWORD="${OPENTAB_DB_PASSWORD:-opentab123}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if ! command -v psql >/dev/null 2>&1; then
  echo "psql not found. Install PostgreSQL client first."
  exit 1
fi

echo "Resetting sanitized demo data in database '${DB_NAME}'..."
PGPASSWORD="${DB_PASSWORD}" psql \
  -h "${DB_HOST}" \
  -p "${DB_PORT}" \
  -U "${DB_USER}" \
  -d "${DB_NAME}" \
  -v ON_ERROR_STOP=1 \
  -f "${SCRIPT_DIR}/reset_demo_data.sql"

echo "Demo data is ready."
