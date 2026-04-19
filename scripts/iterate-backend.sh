#!/usr/bin/env bash
# Iteration for backend (Go) changes.
# Rsyncs source to the Mac Mini, rebuilds the Docker image there, and
# force-recreates the frames container with the fresh image while preserving
# the mounts + env of whatever Portainer stack config you deployed.
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

# Force-recreate the frames container with the fresh image. We dump the
# existing container's run config to JSON, rewrite it with `docker run`-
# compatible flags via `jq`, stop the old container, and run the new one.
echo "=== force-recreate container ==="
ssh "$REMOTE_HOST" '
export PATH=/usr/local/bin:$PATH
if ! docker ps -a --filter name=^frames$ --format "{{.Names}}" | grep -q frames; then
  echo "No frames container present — deploy your Portainer stack first."
  exit 0
fi

# Extract config one field at a time via --format Go templates (no jq needed).
binds=$(docker inspect frames --format "{{range .HostConfig.Binds}}{{println .}}{{end}}")
vols=$(docker inspect frames --format "{{range .Mounts}}{{if eq .Type \"volume\"}}{{println .Name \":\" .Destination}}{{end}}{{end}}" | tr -d " ")
envs=$(docker inspect frames --format "{{range .Config.Env}}{{println .}}{{end}}")
ports=$(docker inspect frames --format "{{range \$cp, \$hps := .HostConfig.PortBindings}}{{range \$hps}}{{println .HostPort \":\" $cp}}{{end}}{{end}}" | sed "s/ //g; s|/[a-z]*||g")

docker stop frames >/dev/null
docker rm frames >/dev/null

run_args=(run -d --name frames --restart unless-stopped)
while IFS= read -r b; do [ -n "$b" ] && run_args+=("-v" "$b"); done <<< "$binds"
while IFS= read -r v; do [ -n "$v" ] && run_args+=("-v" "$v"); done <<< "$vols"
while IFS= read -r p; do [ -n "$p" ] && run_args+=("-p" "$p"); done <<< "$ports"
# Env is one name=value per line — easy because container env doesn'\''t contain newlines.
while IFS= read -r e; do [ -n "$e" ] && run_args+=("-e" "$e"); done <<< "$envs"
run_args+=(ghcr.io/nielheesakkers/frames:latest)

docker "${run_args[@]}" >/dev/null
sleep 2
docker ps --filter name=^frames$ --format "table {{.Status}}"
'
echo "done — refresh your browser"
