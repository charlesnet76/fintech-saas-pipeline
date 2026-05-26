#!/usr/bin/env bash
set -euo pipefail
echo "Setting up local dev environment..."
docker compose -f infrastructure/docker/docker-compose.yml up -d
echo "Done."
