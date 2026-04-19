# Frames — Changelog

All notable changes land here, newest on top. Version bumps follow a simple 0.1 increment per shipped change.

## 0.29.0 — 2026-04-19

- Justified grid rewritten as pure CSS (flex-grow per aspect ratio).
  No JS layout pass, so it reflows live with every pixel of browser
  resize. Every row fills the container width exactly; a phantom
  flex-filler absorbs slack on half-full trailing rows so items don't
  stretch to 2× width.

## 0.28.0 — 2026-04-19

- Grid width is now driven by `bind:clientWidth`, so the justified rows
  layout recomputes reliably on browser resize, sidebar toggles, zoom,
  and devtools open/close. Removed the Grid's inner `overflow-y: auto`
  too so there's no double scrollbar — the outer page-scroll handles it.

## 0.27.0 — 2026-04-19

- "Oorspronkelijke verhouding" grid now reflows on browser resize. Added
  a `window.resize` listener as a safety net around the existing
  `ResizeObserver`, plus a re-measure on file/shape changes so the first
  paint after switching layouts is always right.

## 0.26.0 — 2026-04-19

- Toolbar choices (sort, density, thumbnail shape) are now persisted to
  `localStorage` and restored on page load. Added a small `persisted()`
  helper in `stores.ts` so the three stores share the same mechanism.

## 0.25.0 — 2026-04-19

- Sidebar footer simplified to a single **Settings** link. The Settings,
  Admin and Shares pages now share a top tab bar (via a `(settings)`
  route group) so switching between them is one click. Admin tab shows
  only for admins.
- **Admin stats**: new `Statistieken` block on the admin page with total
  files, folders, rated, per-kind counts, photo volume and cache size,
  last scan info. Backend `/api/admin/stats` returns the numbers.
- **Lightbox slideshow**: play/pause toggle + interval select (2/4/7/10 s)
  in the top bar. Space key toggles. Auto-stops at the last photo.
- **Keyboard shortcuts overlay**: `?` in the lightbox shows a cheatsheet.
- **Justified rows layout** when thumbs are in "Oorspronkelijke
  verhouding" mode — each row is scaled to fill the container width
  exactly (Google Photos style), while preserving each photo's aspect
  ratio. Trailing rows are not up-stretched.

## 0.24.0 — 2026-04-19

- Hover on a grid thumbnail shows an overlay with filename, resolution,
  and file size along the top (existing hover rating stays at the bottom).
- Build now tags `sqlite_fts5` so the FTS5 virtual table added in 0.23
  actually loads at runtime (mattn/go-sqlite3 gates FTS5 behind a build
  tag). Dockerfile + Makefile updated.

## 0.23.0 — 2026-04-19

- **Thumbnail shape toggle** in the toolbar: "Squares" (uniform crop,
  default) or "Oorspronkelijke verhouding" (justified rows preserving
  each photo's aspect ratio).
- **Video hover preview**: hovering a video thumbnail plays the original
  muted + looped, replacing the still-frame thumb in place.
- **FTS5 full-text search** over filename + relative_path. Migration
  `0003_fts5.sql` creates the virtual table with triggers to stay in
  sync. Search now uses prefix-MATCH (`berl*` matches `berlin`).
- **Multi-file shares**: select photos in the grid (shift/⌘-click),
  toolbar shows "N geselecteerd" + "Share selected" → share dialog
  opens scoped to those file IDs. Public view, media access and ZIP
  download all respect the scoped list. Migration `0004_file_shares.sql`
  adds an optional `file_ids` column to `shares`.

## 0.22.0 — 2026-04-19

- Lightbox photo vertically centered via explicit flex centering.

## 0.21.0 — 2026-04-19

- Rating on hover: hovering a thumbnail in the grid reveals a 5-star row
  along the bottom — click to rate without opening the lightbox.
  Keyboard shortcuts 0–5 also work while hovering. Existing ratings stay
  visible as filled stars; rating a zero-rated item keeps the overlay
  only while the mouse is over the thumb.

## 0.20.0 — 2026-04-19

- **Timeline headers in the grid** when sorted by capture date: sticky
  `Augustus 2024` labels between months. Groups come from the files'
  `taken_at` on the client, so no extra backend roundtrips.
- **Zoom + pan in the lightbox**: Ctrl/⌘ + scroll zooms, drag to pan
  when zoomed, `+` / `-` / `0` keys, double-click to toggle 2×, zoom
  percentage badge with a reset button. Touch-swipe still works at 1×.
- **Filmstrip** at the bottom of the lightbox showing neighbor thumbs,
  highlighted + auto-scrolled to the active file. Click a thumb to jump.
- **Star ratings 0–5** per file. Click in the lightbox rating row; the
  grid shows filled stars on the bottom-left corner of thumbs. New sort
  option "Op rating" in the toolbar. DB migration `0002_ratings.sql`
  adds an indexed `rating` column.

## 0.19.0 — 2026-04-19

- New admin "Onderhoud" section with two maintenance actions:
  - **Clear cache** — wipes the thumbnail/preview directories and flips
    every file back to `thumb_status='pending'`. Worker regenerates on
    the next scan.
  - **Reset library** — full wipe: cache plus the entire folder/file/
    scan-job/shares index. Use after changing the container's photos
    root mount, then click "Run full scan" to re-index. Users and
    settings are kept.

## 0.18.0 — 2026-04-19

- Share dialog: "Copy link" now copies AND closes the dialog in one
  click. Primary action rightmost, with a secondary "Done" in case you
  just want to dismiss. Falls back to `execCommand('copy')` on non-HTTPS
  contexts so copying still works over plain HTTP.

## 0.17.0 — 2026-04-19

- Lightbox close (✕ / Esc) now navigates to the containing folder
  instead of walking `history.back()`. Previously, after flipping through
  photos with → → →, clicking ✕ would just undo one step instead of
  actually closing the preview.

## 0.16.0 — 2026-04-19

- Lightbox layout fix: previous version had 4 grid items in a 3-column
  grid, so the EXIF panel got auto-placed into a second row, squished to
  60 px wide, showing just "P." and "13 van 137". Switched to an
  explicit 4-column grid (nav | media | nav | info), centered image,
  fully opaque background, and bumped the z-index so the (app) shell
  doesn't show through.

## 0.15.0 — 2026-04-19

- Lightbox redesign: close button (✕ top-right, also Esc), ← → actually
  navigate now (reactive index, 50 k siblings loaded so even large
  folders work), swipe on touch, position indicator (e.g. "17 of 137").
- Detailed EXIF panel on the right: camera, lens, focal length,
  aperture, shutter, ISO, dimensions, size, type, GPS (linked to
  OpenStreetMap), software, relative path. Read on demand via exiftool
  in the `/api/file/{id}` response — no DB migration needed.
- Backend: new `ReadDetailedEXIF` helper and `exif` field on the file
  DTO.

## 0.14.0 — 2026-04-19

- Clicking a folder in the sidebar now also expands that folder, so its
  subfolders become visible without a separate chevron click.

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
