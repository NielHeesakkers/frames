# Frames — Changelog

All notable changes land here, newest on top. Version bumps follow a simple 0.1 increment per shipped change.

## 0.13.0 — 2026-04-19

- Folder tree now auto-collapses siblings on navigation. Going from
  `folder01/subfolder` to `folder02` collapses `folder01` and its
  children; only the ancestor chain of the current path stays open.
  Manual chevron toggles still work until the next navigation.

## 0.12.0 — 2026-04-19

- Folder tree remembers which folders you expanded. Expansion state is
  persisted to `localStorage` and restored on page load, so a refresh
  never collapses branches you had open. Tree rows also navigate via a
  proper `<a href>` now (the chevron still toggles expansion only).

## 0.11.0 — 2026-04-19

- Folder tree in the sidebar auto-expands to the current path on page
  load. If the URL is `/browse/2018 Berlin/JPG`, the tree shows
  `2018 Berlin` expanded with `JPG` highlighted — no more losing your
  position in the tree after a refresh.

## 0.10.0 — 2026-04-19

- **Refresh at a nested folder URL now actually loads that folder.** The
  previous version imported the root `/browse` component into the
  `[...path]` route and read `$page.params` inside — which silently did
  not populate on reload. Extracted the view into
  `$lib/components/BrowseView.svelte` with an explicit `path` prop, so
  both `/browse` and `/browse/[...path]` pages pass the path in directly
  and the component can't drift from the URL.

## 0.9.0 — 2026-04-19

- **Folder navigation now changes the URL** — clicking a subfolder card
  was using `preventDefault` and only updating the local store, so the
  URL never followed. Refresh would therefore take you back to whatever
  the URL still pointed at (often root's first folder). Fixed — subfolder
  cards and breadcrumb links now navigate the URL normally; the path is
  derived from `$page.params.path`.
- **Folder listings show everything** — backend limit cap raised from
  1,000 to 50,000 per folder, and the frontend requests 50k. Leaf folders
  with thousands of photos render in full. For the rare folder beyond
  50k, pagination is still needed (not in v0.9).

## 0.8.0 — 2026-04-19

- Browse view now drives the current folder from the URL, so a page reload
  (F5 / Cmd-R) keeps you in the same folder instead of bouncing back to
  root. Fixes the onMount that always reset to `''`.

## 0.7.0 — 2026-04-19

- Home view cleaned up: "Laatste toegevoegde mappen" and "Alle mappen"
  sections removed from root — the folder tree in the sidebar already
  covers navigation. Home now shows only the latest photos.
- Inside a container folder that has no direct photos of its own (like
  `2018 Berlin` that only holds `RAW` and `JPG` subfolders), the FOTO'S
  section now previews the 10 most recently added photos from anywhere
  in that subtree instead of showing an empty-state message.
- Backend: `/api/latest` gained an optional `path` query parameter that
  scopes results to a folder's entire subtree via a recursive CTE.

## 0.6.0 — 2026-04-19

- Admin Scan section cleaned up. Two side-by-side cards with a one-line
  explanation of what each scan does, and a single action button. Raw
  JSON status dump removed.

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
