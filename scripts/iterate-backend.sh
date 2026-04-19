#!/usr/bin/env bash
# Iteration for backend (Go) changes.
# Rsyncs source to the Mac Mini, rebuilds the Docker image there, then
# force-recreates the frames container with the fresh image so migrations
# and new code actually take effect.
# Total: ~90 seconds.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REMOTE_HOST="server@servermini.local"
REMOTE_DIR="/Users/server/Docker/frames-src"
REMOTE_FRONTEND="/Users/server/Docker/Frames/Frontend"

cd "$REPO_ROOT"

echo "=== build + sync frontend ==="
( cd web && npm run build 2>&1 | tail -2 )
rsync -a --delete web/build/ "$REMOTE_HOST:$REMOTE_FRONTEND/"

echo "=== rsync source ==="
rsync -a --delete \
  --exclude='.git' --exclude='.superpowers' \
  --exclude='internal/frontend/dist' \
  --exclude='web/node_modules' --exclude='web/build' --exclude='web/.svelte-kit' \
  --exclude='/photos' --exclude='/cache' --exclude='/data' \
  --exclude='*.db*' --exclude='/frames' \
  ./ "$REMOTE_HOST:$REMOTE_DIR/"

echo "=== rebuild image on mac mini ==="
ssh "$REMOTE_HOST" "
export PATH=/usr/local/bin:\$PATH
cd $REMOTE_DIR
sed -i '' 's|^# syntax=docker/dockerfile:.*||' Dockerfile
time DOCKER_BUILDKIT=0 docker build -t ghcr.io/nielheesakkers/frames:latest -t frames:dev . 2>&1 | tail -4
"

# Force-recreate the frames container. Config below mirrors the Portainer
# stack (NAS-mounted photos, cache/data/frontend bind mounts, env vars).
# If you change the Portainer compose (different volume name, different
# NAS host, etc.), update these values too.
echo "=== force-recreate container ==="
ssh "$REMOTE_HOST" '
export PATH=/usr/local/bin:$PATH
docker rm -f frames >/dev/null 2>&1 || true
docker run -d --name frames --restart unless-stopped \
  -p 2342:8080 \
  -v frames_photos_nas:/photos:rw \
  -v /Users/server/Docker/Frames/Cache:/cache \
  -v /Users/server/Docker/Frames/Data:/data \
  -v /Users/server/Docker/Frames/Frontend:/frontend:ro \
  -e FRAMES_SESSION_SECRET="hfbeu*WE#YRhv8d0hvvbosdfb" \
  -e FRAMES_PUBLIC_URL="https://frames.heesakkers.com" \
  -e FRAMES_ADMIN_USERNAME="niel" \
  -e FRAMES_ADMIN_PASSWORD="Puno@4074" \
  -e FRAMES_TRUST_PROXY="1" \
  -e FRAMES_FRONTEND_DIR="/frontend" \
  ghcr.io/nielheesakkers/frames:latest >/dev/null
sleep 2
docker ps --filter name=^frames$ --format "table {{.Status}}"
'
echo "done — refresh your browser"
