#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

docker compose -f docker/docker-compose.yml -f docker/docker-compose.dev.yml -p stock \
  exec -T db psql -v ON_ERROR_STOP=1 -U appuser -d app < docker/postgres/seed.sql

echo "Seed data inserted successfully."
