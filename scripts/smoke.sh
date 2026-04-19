#!/usr/bin/env bash
# scripts/smoke.sh — brings the stack up, creates a test photo, verifies thumb generation
set -euo pipefail

export FRAMES_SESSION_SECRET=${FRAMES_SESSION_SECRET:-dev-secret-32-characters-xxxxxxxxx}
export FRAMES_PUBLIC_URL=${FRAMES_PUBLIC_URL:-http://localhost:8080}
export FRAMES_ADMIN_USERNAME=${FRAMES_ADMIN_USERNAME:-admin}
export FRAMES_ADMIN_PASSWORD=${FRAMES_ADMIN_PASSWORD:-admin1234567}

mkdir -p photos
# seed one tiny test image
if command -v vips >/dev/null 2>&1; then
  vips black photos/test.jpg 64 64 >/dev/null 2>&1 || true
fi

docker compose up -d --build
trap 'docker compose down' EXIT

# Wait for healthz.
for i in $(seq 1 30); do
  if curl -sf http://localhost:8080/healthz > /dev/null; then break; fi
  sleep 1
done
curl -sf http://localhost:8080/healthz

# Trigger a manual scan (requires login via cookie jar).
jar=$(mktemp)
curl -sf -c "$jar" http://localhost:8080/api/me > /dev/null || true
CSRF=$(grep frames_csrf "$jar" | awk '{print $7}')
curl -sf -b "$jar" -c "$jar" -H "X-CSRF-Token: $CSRF" \
  -H "Content-Type: application/json" \
  -d '{"username":"'"$FRAMES_ADMIN_USERNAME"'","password":"'"$FRAMES_ADMIN_PASSWORD"'"}' \
  http://localhost:8080/api/login > /dev/null
curl -sf -b "$jar" -H "X-CSRF-Token: $CSRF" -X POST http://localhost:8080/api/scan > /dev/null

echo "smoke ok"
