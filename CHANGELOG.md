# Frames — Changelog

All notable changes land here, newest on top. Version bumps follow a simple 0.1 increment per shipped change.

## 0.5.0 — 2026-04-19

- Sidebar folder tree now shows the **total** item count including all
  descendants, not just direct files. A container folder like `2018 Berlin`
  that holds only subfolders (`JPG`, `RAW`) previously read `0`; it now
  shows the sum across the subtree. Single recursive CTE per expansion —
  still fast.

## 0.4.0 — 2026-04-19

- Dev iteration is **~16× faster** for frontend-only changes. New `FRAMES_FRONTEND_DIR` env var: when set, the binary serves the SvelteKit build from that directory instead of the embedded FS, so frontend edits go live after a single `npm run build && rsync` (~3 s total, no Docker rebuild).
- `scripts/iterate-frontend.sh` and `scripts/iterate-backend.sh` — pick the right speed tier depending on what you changed.

## 0.3.0 — 2026-04-19

- Recursive folder tree in the sidebar — drill into any depth. Clicking a
  folder name now navigates AND expands it inline, so siblings stay
  visible and you can see the full path you're in. The chevron on the
  left still toggles expansion without navigating.

## 0.2.0 — 2026-04-19

- Inside a folder, show a "Foto's (n)" heading above the grid and an
  explicit empty-state hint when the folder has no direct files. Prevents
  the "only subfolders, nothing else visible" confusion at a glance.

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
