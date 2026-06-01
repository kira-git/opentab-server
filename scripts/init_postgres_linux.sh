#!/usr/bin/env bash
set -euo pipefail

# Create the PostgreSQL database and role required by OpenTab.
# Tables and seed data are created by the Go server through GORM AutoMigrate + Seed.

DB_NAME="${OPENTAB_DB_NAME:-opentab}"
DB_USER="${OPENTAB_DB_USER:-opentab}"
DB_PASSWORD="${OPENTAB_DB_PASSWORD:-opentab123}"

if ! command -v psql >/dev/null 2>&1; then
  echo "psql not found. Install PostgreSQL client/server first."
  exit 1
fi

run_psql() {
  if command -v sudo >/dev/null 2>&1; then
    sudo -u postgres psql "$@"
  else
    psql -U postgres "$@"
  fi
}

echo "Creating PostgreSQL role '${DB_USER}' and database '${DB_NAME}'..."

ROLE_EXISTS="$(run_psql -tAc "SELECT 1 FROM pg_roles WHERE rolname = '${DB_USER}'" || true)"
if [[ "${ROLE_EXISTS}" != "1" ]]; then
  run_psql -v ON_ERROR_STOP=1 -c "CREATE ROLE ${DB_USER} WITH LOGIN PASSWORD '${DB_PASSWORD}';"
else
  run_psql -v ON_ERROR_STOP=1 -c "ALTER ROLE ${DB_USER} WITH LOGIN PASSWORD '${DB_PASSWORD}';"
fi

DB_EXISTS="$(run_psql -tAc "SELECT 1 FROM pg_database WHERE datname = '${DB_NAME}'" || true)"
if [[ "${DB_EXISTS}" != "1" ]]; then
  run_psql -v ON_ERROR_STOP=1 -c "CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};"
fi

run_psql -v ON_ERROR_STOP=1 -d "${DB_NAME}" -c "GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};"
run_psql -v ON_ERROR_STOP=1 -d "${DB_NAME}" -c "GRANT ALL ON SCHEMA public TO ${DB_USER};"

cat <<EOF

PostgreSQL database is ready.

Use this DATABASE_URL when starting the server:

postgres://${DB_USER}:${DB_PASSWORD}@localhost:5432/${DB_NAME}?sslmode=disable

Example:

APP_MODE=postgres \\
DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@localhost:5432/${DB_NAME}?sslmode=disable" \\
HOST=0.0.0.0 \\
PORT=8080 \\
./opentab-server

EOF
