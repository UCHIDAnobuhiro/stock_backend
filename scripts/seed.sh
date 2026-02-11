#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

docker compose -f docker/docker-compose.yml -f docker/docker-compose.dev.yml -p stock \
  exec -T db mysql -uappuser -papppass app < docker/mysql/seed.sql

echo "Seed data inserted successfully."
