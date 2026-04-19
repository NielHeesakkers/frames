# Frames

Self-hosted media library with a web frontend over an existing Finder folder structure.

## Quick start

```bash
git clone https://github.com/NielHeesakkers/frames && cd frames

export FRAMES_SESSION_SECRET=$(head -c 32 /dev/urandom | base64)
export FRAMES_PUBLIC_URL=http://localhost:8080
export FRAMES_ADMIN_USERNAME=niel
export FRAMES_ADMIN_PASSWORD='pick-something-strong'

mkdir -p photos
docker compose up --build
```

Open http://localhost:8080 and log in.

## Volumes

| Volume | Purpose | Backup? |
|--------|---------|---------|
| `/photos` | Your media root (bind-mount to your Finder library) | — (managed externally) |
| `/cache` | Thumbnails and previews | no — regenerable |
| `/data` | SQLite DB (`frames.db`) | **yes** |

## TLS

Run behind Caddy/Traefik/Nginx. Example Caddyfile:

```
frames.example.com {
  reverse_proxy frames:8080
}
```

Set `FRAMES_PUBLIC_URL=https://frames.example.com` so share-link URLs are correct.

## Environment variables

See [`docs/superpowers/specs/2026-04-18-frames-design.md`](docs/superpowers/specs/2026-04-18-frames-design.md) §10.

## Development

```bash
make build    # build binary
make test     # run all tests
make web      # build frontend into internal/frontend/dist
make run      # start binary (expects env vars)
```

Local dev with hot-reload frontend:

```bash
cd web && npm run dev   # proxies /api to :8080
# in another shell:
FRAMES_SESSION_SECRET=dev FRAMES_PUBLIC_URL=http://localhost:5173 \
  FRAMES_PHOTOS_ROOT=$(pwd)/photos FRAMES_DATA_DIR=$(pwd)/data \
  FRAMES_CACHE_DIR=$(pwd)/cache FRAMES_ADMIN_USERNAME=niel FRAMES_ADMIN_PASSWORD=dev12345 \
  go run ./cmd/frames
```

## Design

See [`docs/superpowers/specs/2026-04-18-frames-design.md`](docs/superpowers/specs/2026-04-18-frames-design.md).
