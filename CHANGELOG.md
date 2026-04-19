# Frames — Changelog

All notable changes land here, newest on top. Version bumps follow a simple 0.1 increment per shipped change.

## 0.1.0 — 2026-04-19

Initial working release deployed to the Mac Mini M4.

- Go backend: SQLite (WAL), chi router, argon2id auth, CSRF, session + login rate limit
- Filesystem scanner: incremental mtime-driven walk, transactional bulk inserts, scheduled + manual rescans
- Thumbnail pipeline: vipsthumbnail + ffmpeg + exiftool, priority queue, bounded worker pool
- RAW files: fallback via exiftool's embedded JPEG when libvips lacks libraw support
- Sharing: internal markers between users + external share-links with expiration, password, ZIP download, anonymous upload
- SvelteKit frontend: login, folder tree sidebar, breadcrumb, grid, lightbox, context menu, upload dialog, move picker, share dialog, shares page, public share view, settings, admin, search, mobile polish
- Ignored sidecar file types: `.xmp`, `.thm`
- Docker image published to `ghcr.io/nielheesakkers/frames` (native arm64) via GitHub Actions
