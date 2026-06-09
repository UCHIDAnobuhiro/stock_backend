#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."

docker compose -f docker/docker-compose.yml -p stock \
  exec -T db psql -v ON_ERROR_STOP=1 -U appuser -d app < db/seed/seed.sql

echo "Seed data inserted successfully."
