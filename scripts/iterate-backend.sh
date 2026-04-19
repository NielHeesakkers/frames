#!/usr/bin/env bash
# Iteration for backend (Go) changes.
# Rsync source to Mac Mini, rebuild Docker image, restart container.
# Total: ~50 seconds (Docker build dominates). Use this when you change code
# in internal/, cmd/, go.mod, or Dockerfile.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REMOTE_HOST="server@servermini.local"
REMOTE_DIR="/Users/server/Docker/frames-src"

cd "$REPO_ROOT"
echo "=== rsync source ==="
rsync -a --delete \
  --exclude='.git' --exclude='.superpowers' \
  --exclude='internal/frontend/dist' \
  --exclude='web/node_modules' --exclude='web/build' --exclude='web/.svelte-kit' \
  --exclude='/photos' --exclude='/cache' --exclude='/data' \
  --exclude='*.db*' --exclude='/frames' \
  ./ "$REMOTE_HOST:$REMOTE_DIR/"

echo "=== rebuild + restart on mac mini ==="
ssh "$REMOTE_HOST" "
export PATH=/usr/local/bin:\$PATH
cd $REMOTE_DIR
# strip the 'syntax=' line (docker desktop keychain issue in ssh sessions)
sed -i '' 's|^# syntax=docker/dockerfile:.*||' Dockerfile
time DOCKER_BUILDKIT=0 docker build -t frames:dev . 2>&1 | tail -4
docker rm -f frames 2>&1 | tail -1
docker run -d --name frames --restart unless-stopped \\
  -p 2342:8080 \\
  -v /Users/server/Desktop/Photo:/photos:rw \\
  -v /Users/server/Docker/Frames/Cache:/cache \\
  -v /Users/server/Docker/Frames/Data:/data \\
  -v /Users/server/Docker/Frames/Frontend:/frontend:ro \\
  -e FRAMES_SESSION_SECRET='hfbeu*WE#YRhv8d0hvvbosdfb' \\
  -e FRAMES_PUBLIC_URL='https://frames.heesakkers.com' \\
  -e FRAMES_ADMIN_USERNAME='niel' \\
  -e FRAMES_ADMIN_PASSWORD='Puno@4074' \\
  -e FRAMES_TRUST_PROXY='1' \\
  -e FRAMES_FRONTEND_DIR='/frontend' \\
  frames:dev
sleep 2
docker ps --filter name=frames --format 'table {{.Status}}'
"
echo "done — refresh your browser"
