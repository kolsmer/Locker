#!/bin/sh
set -eu

DB_HOST="${DB_HOST:-postgres}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="${DB_NAME:-locker}"

export PGPASSWORD="$DB_PASSWORD"

until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres; do
  echo "waiting for postgres..."
  sleep 2
done

DB_EXISTS="$(psql "postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/postgres?sslmode=disable" -Atqc "SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'")"
if [ "$DB_EXISTS" != "1" ]; then
  echo "database ${DB_NAME} not found, creating..."
  psql "postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/postgres?sslmode=disable" -v ON_ERROR_STOP=1 -c "CREATE DATABASE \"${DB_NAME}\""
fi

exec goose -dir /app/migrations postgres "postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable" up
