#!/usr/bin/env bash
# Fast iteration for frontend-only changes.
# Build SvelteKit locally + rsync web/build/ to the Mac Mini's Frontend dir.
# Total: ~3 seconds. No Docker rebuild needed.
#
# Prereq (one-time): container started with -v .../Frontend:/frontend:ro and
# -e FRAMES_FRONTEND_DIR=/frontend.
set -euo pipefail

cd "$(dirname "$0")/../web"
echo "=== building frontend ==="
npm run build 2>&1 | tail -3

echo "=== syncing to mac mini ==="
rsync -a --delete build/ server@servermini.local:/Users/server/Docker/Frames/Frontend/
echo "done — refresh your browser"
