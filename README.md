# Frames

Self-hosted media library with a web frontend over an existing Finder folder structure.

## Quick start (local Docker)

```bash
git clone https://github.com/NielHeesakkers/frames && cd frames

cp .env.example .env
# Edit .env and fill in real values. At minimum:
#   FRAMES_SESSION_SECRET=$(openssl rand -base64 32)
#   FRAMES_ADMIN_PASSWORD=<something strong>
#   FRAMES_PHOTOS_HOST_PATH=/absolute/path/to/your/photos    # optional; defaults to ./photos

docker compose --env-file .env up -d
```

Open http://localhost:8080 and log in with the admin credentials from `.env`.

`docker compose up` pulls the pre-built image from `ghcr.io/nielheesakkers/frames:latest`. To build locally from source instead:

```bash
docker compose build && docker compose up -d
```

## Deploying with Portainer (or any Docker host)

The published image at `ghcr.io/nielheesakkers/frames:latest` means Portainer can deploy without access to the source code.

1. **GHCR access**: the package is private by default because the repo is private. Two options:
   - **Make the package public** (Go to https://github.com/users/NielHeesakkers/packages/container/frames → Package settings → "Change package visibility" → Public). The image is fine to publish publicly even while the repo stays private.
   - **Keep it private and add registry credentials** in Portainer (Registries → Add registry → GHCR, username=your GitHub handle, password=a PAT with `read:packages`).
2. In Portainer: **Stacks → Add stack → Web editor** (not Git repo — the image is self-contained now). Paste the `docker-compose.yml` and edit:
   - `FRAMES_PHOTOS_HOST_PATH` → absolute path on the Portainer host (e.g. `/mnt/photos`)
   - All `FRAMES_*` env vars — set real values
3. Deploy.

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

Set `FRAMES_PUBLIC_URL=https://frames.example.com` so share-link URLs are correct. If the proxy is on the same Docker network, also set `FRAMES_TRUST_PROXY=1` so per-IP rate limits see the real client IP.

## Environment variables

See `.env.example` for the full list. Details in [`docs/superpowers/specs/2026-04-18-frames-design.md`](docs/superpowers/specs/2026-04-18-frames-design.md) §10.

## Development

```bash
make build    # build binary
make test     # run all Go tests
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
