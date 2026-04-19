#!/usr/bin/env bash
# Iteration for backend (Go) changes.
# Rsync source to Mac Mini, rebuild Docker image, restart container.
# Total: ~50 seconds (Docker build dominates). Use this when you change code
# in internal/, cmd/, go.mod, or Dockerfile.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REMOTE_HOST="server@servermini.local"
REMOTE_DIR="/Users/server/Docker/frames-src"
REMOTE_FRONTEND="/Users/server/Docker/Frames/Frontend"

cd "$REPO_ROOT"

# The container's FRAMES_FRONTEND_DIR overrides the embedded frontend, so
# any UI changes must also land in the bind-mount dir. Build + sync first
# so that by the time the container restarts, both backend and frontend
# are in sync.
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
# Restart the existing Portainer-managed container so it picks up the new
# image + the frontend dir update. We deliberately do NOT re-create the
# container so we don't clobber any compose/env changes you made in Portainer.
ssh "$REMOTE_HOST" "export PATH=/usr/local/bin:\$PATH; docker restart frames 2>/dev/null || echo 'No frames container running yet — deploy your Portainer stack.'"
echo "done — refresh your browser"
