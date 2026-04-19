# Frames — Design Document

**Date:** 2026-04-18
**Status:** Approved (brainstorm phase)
**Next step:** Implementation plan (via `writing-plans` skill)

---

## 1. Overview

**Frames** is a self-hosted, Docker-packaged media library application. It provides a web frontend on top of an existing Finder-driven folder structure of photos, videos, and other media files. The filesystem is the source of truth; Frames mirrors, indexes, and serves it.

### Goals

- Browse a 500k+ file, 5 TB+ personal/family media library over the web
- Reflect Finder folder structure literally — a folder created in Finder appears in the app after a scan
- Fast browsing with aggressive thumbnail caching
- Support all common formats (JPEG, PNG, HEIC, RAW formats, MP4, MOV, PDFs, and other file types)
- Multi-user accounts (login required); all logged-in users see the same library
- Two sharing mechanisms:
  - External share-links (no account required) with expiration, password, download, and upload options
  - Internal sharing markers between accounts as navigation hints
- Basic file-operations from the frontend: upload, create folder, rename, move, delete
- Simple deployment: `docker compose up`

### Non-goals (v1)

- Face recognition / AI tagging
- Albums as a separate concept from folders
- Photo editing that persists edits
- Multi-tenant (multiple libraries per instance)
- Native mobile apps
- Sync clients (Lightroom plugin, etc.)
- Email notifications

---

## 2. Architecture

Single Go binary bundling API and frontend, talking to SQLite, coordinating a scanner and a thumbnail worker pool. One Docker service, three volumes.

```
┌─────────────────────────────────────────────────────────┐
│                   Docker Compose                         │
│                                                           │
│  ┌─────────────────────────────────────────────────┐    │
│  │   frames (single Go binary)                      │    │
│  │                                                   │    │
│  │   ┌─────────┐  ┌──────────┐  ┌──────────────┐  │    │
│  │   │ HTTP API│  │ Scanner  │  │ Thumb worker │  │    │
│  │   │ + embed │  │ (scheduled│ │ (async queue)│  │    │
│  │   │ frontend│  │ + manual) │ └──────────────┘  │    │
│  │   └─────────┘  └──────────┘                    │    │
│  │        │            │              │            │    │
│  │        └────────────┴──────────────┘            │    │
│  │                     │                            │    │
│  │              ┌──────┴──────┐                     │    │
│  │              │   SQLite    │                     │    │
│  │              └─────────────┘                     │    │
│  └─────────────────────────────────────────────────┘    │
│         │                 │                    │         │
│   ┌─────┴─────┐     ┌────┴─────┐        ┌────┴─────┐   │
│   │ /photos   │     │ /cache   │        │ /data    │   │
│   │ (bind mnt)│     │ (volume) │        │ (volume) │   │
│   │ read+write│     │ thumbs   │        │ SQLite DB│   │
│   └───────────┘     └──────────┘        └──────────┘   │
└──────────────────────────────────────────────────────────┘
```

### Tech stack

- **Backend**: Go (`net/http`, `chi` router, `embed` for bundled frontend assets)
- **Image processing**: libvips via [`govips`](https://github.com/davidbyttow/govips)
- **RAW processing**: `libraw` — embedded JPEG extraction for fast thumbnails; full rendering for previews
- **Video**: `ffmpeg` for frame extraction and metadata
- **Database**: SQLite (WAL mode)
- **Frontend**: SvelteKit (static build, embedded in Go binary); `svelte-virtual` for grid virtualization
- **Auth**: session cookies (HttpOnly, SameSite=Lax, Secure); argon2id password hashing
- **Packaging**: multi-stage Docker build on Alpine; target image size ~50 MB

### Volumes

| Mount | Purpose | Backup target? |
|-------|---------|----------------|
| `/photos` | Bind-mount to the user's Finder library (read+write) | No — user manages externally |
| `/cache` | Thumbnails and previews | No — regenerable |
| `/data` | SQLite database file (`frames.db`) | Yes |

---

## 3. Data model

SQLite schema (simplified — drop nullability/length details as needed during implementation):

```sql
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  username TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,          -- argon2id
  is_admin BOOLEAN DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE folders (
  id INTEGER PRIMARY KEY,
  parent_id INTEGER REFERENCES folders(id) ON DELETE CASCADE,
  path TEXT UNIQUE NOT NULL,            -- relative path from photos root
  name TEXT NOT NULL,
  mtime INTEGER NOT NULL,
  item_count INTEGER DEFAULT 0,
  last_scanned_at DATETIME
);
CREATE INDEX idx_folders_parent ON folders(parent_id);
CREATE INDEX idx_folders_path ON folders(path);

CREATE TABLE files (
  id INTEGER PRIMARY KEY,
  folder_id INTEGER NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
  filename TEXT NOT NULL,
  relative_path TEXT UNIQUE NOT NULL,
  size INTEGER NOT NULL,
  mtime INTEGER NOT NULL,
  mime_type TEXT,
  kind TEXT NOT NULL,                    -- 'image' | 'raw' | 'video' | 'other'
  taken_at DATETIME,                     -- EXIF DateTimeOriginal
  width INTEGER,
  height INTEGER,
  camera_make TEXT,
  camera_model TEXT,
  orientation INTEGER,
  duration_ms INTEGER,                   -- video only
  thumb_status TEXT DEFAULT 'pending',   -- 'pending' | 'ready' | 'failed'
  thumb_attempts INTEGER DEFAULT 0,      -- retry counter, capped at 3
  preview_status TEXT DEFAULT 'pending',
  preview_attempts INTEGER DEFAULT 0,
  UNIQUE(folder_id, filename)
);
CREATE INDEX idx_files_folder ON files(folder_id);
CREATE INDEX idx_files_taken_at ON files(taken_at);
CREATE INDEX idx_files_relpath ON files(relative_path);

CREATE TABLE shares (
  id INTEGER PRIMARY KEY,
  token TEXT UNIQUE NOT NULL,            -- 32-char URL-safe random
  folder_id INTEGER NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
  created_by INTEGER NOT NULL REFERENCES users(id),
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  expires_at DATETIME,                   -- NULL = never
  password_hash TEXT,                    -- NULL = no password
  allow_download BOOLEAN DEFAULT 1,
  allow_upload BOOLEAN DEFAULT 0,
  revoked_at DATETIME                    -- NULL = active
);

CREATE TABLE folder_shares (
  folder_id INTEGER NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
  shared_with_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  shared_by INTEGER NOT NULL REFERENCES users(id),
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (folder_id, shared_with_user_id)
);

CREATE TABLE scan_jobs (
  id INTEGER PRIMARY KEY,
  type TEXT NOT NULL,                    -- 'full' | 'incremental' | 'folder'
  started_at DATETIME NOT NULL,
  finished_at DATETIME,
  files_scanned INTEGER DEFAULT 0,
  files_added INTEGER DEFAULT 0,
  files_updated INTEGER DEFAULT 0,
  files_removed INTEGER DEFAULT 0,
  error TEXT
);

CREATE TABLE sessions (
  token TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at DATETIME NOT NULL
);
```

---

## 4. Filesystem scanner

### Strategy

**Incremental, mtime-driven traversal.** Full rescans are expensive at 500k+ files, so the common path must avoid touching unchanged directories.

1. Walk `/photos` with `filepath.WalkDir`.
2. For each folder: compare on-disk `mtime` to stored `mtime`. If equal, skip descent into that folder's files.
3. For changed folders: list files, diff against DB (by filename + mtime + size), compute upserts and deletions.
4. Batch writes in transactions of ~1000 rows to keep memory flat.
5. Metadata extraction (EXIF, dimensions) is lazy: on insert, set `thumb_status='pending'` and let the worker pool extract metadata together with thumbnail generation.

### Schedule

- **Incremental scan**: every `FRAMES_SCAN_INTERVAL` (default 5 minutes)
- **Full scan**: nightly at `FRAMES_FULL_SCAN_CRON` (default `0 3 * * *`) — same as incremental but does not trust stored `mtime` (handles clock skew and mounted-volume edge cases)
- **Manual**: `POST /api/scan` endpoint, UI "Rescan now" button

### Change-handling semantics

- **New file detected**: inserted with `thumb_status='pending'`, metadata extracted by worker
- **Modified file** (mtime or size changed): same row updated, thumb/preview regenerated, derivatives reclaimed from cache
- **Missing file**: hard-deleted from `files` table. `ON DELETE CASCADE` cleans up related rows. Cached derivatives in `/cache/thumb/xx/{id}.webp` and `/cache/preview/xx/{id}.webp` are removed by the same transaction. See "Open questions" for a tombstone alternative.
- **New folder**: inserted; scan descends into it immediately
- **Missing folder**: hard-deleted with cascade to files

---

## 5. Thumbnail/preview pipeline

### Sizes

| Size | Longest edge | Use | Format |
|------|--------------|-----|--------|
| `thumb` | 256 px | Grid view | WebP q=75 |
| `preview` | 2048 px | Lightbox single-view | WebP q=85 |
| `original` | — | Download, direct view of raw files | passthrough |

### Cache layout

Content-addressable by `files.id`, sharded by first two hex chars to keep directory sizes manageable:

```
/cache/
├── thumb/
│   ├── 00/
│   │   ├── 0001a2b3.webp
│   │   └── 0001c4d5.webp
│   └── 01/...
├── preview/
│   └── 00/...
└── tmp/                                  # atomic writes: rename from here
```

### Generation

A pool of N goroutines (default `runtime.NumCPU()`) pulls from a priority queue of `files.id` rows with `thumb_status='pending'`:

- **Image** (JPEG, PNG, HEIC): libvips decode → resize → WebP encode
- **RAW**: libraw extracts embedded JPEG → libvips resize → WebP (fast path, ~50 ms). Full-render preview via libraw is generated **on demand** when the lightbox requests it, not eagerly.
- **Video**: ffmpeg extracts frame at 3 s (or 10% if shorter) → libvips → WebP
- **Other** (PDF, audio, etc.): generic icon; no thumbnail generated

Writes go to `/cache/tmp/` then `rename(2)` into the final path — atomic per POSIX.

### Priorities

1. Thumbnails for files in currently-viewed folders (pushed by folder-fetch API)
2. On-demand previews (requested by lightbox)
3. Background: remaining pending thumbs

### Serving

- `GET /api/thumb/{id}` — serves `/cache/thumb/xx/id.webp`. If pending, returns 202 with Retry-After or blocks up to 3 s waiting.
- `GET /api/preview/{id}` — same for preview size. Triggers full render for RAW if needed.
- `GET /api/original/{id}` — streams original from `/photos` with Range support.
- ETag: `{id}-{mtime}`. `Cache-Control: public, max-age=31536000, immutable` on derivatives.
- Frontend uses `<img loading="lazy">` to only request thumbs entering the viewport.

### Retry and failure

Maximum 3 attempts per file. After failure, `thumb_status='failed'`; frontend shows a generic placeholder. Admin UI offers "Retry failed thumbs" button.

---

## 6. Authentication and permissions

### Auth flow

- `POST /api/login` — verify argon2id hash, create session, set HttpOnly `session` cookie (30-day expiry)
- `POST /api/logout` — delete session row
- Rate limit: 5 login attempts per 15 min per IP (in-memory map)
- Session tokens are random 32 bytes, stored in `sessions` table

### Admin bootstrapping

On first startup, if no admin exists, read `FRAMES_ADMIN_USERNAME` and `FRAMES_ADMIN_PASSWORD` env vars and create that user. Alternative: `docker exec frames frames create-admin` CLI command.

### Roles

- **admin**: manage users, see all shares, trigger scans, destructive bulk operations
- **user**: browse, download, create own shares, basic file operations (rename, move, delete, create folder)

### Permissions

All logged-in users see the entire library. The filesystem is the single source of truth for what exists; permissions are flat within the app and access control lives only in login state, not per-folder.

File operations (rename, move, delete, create folder) are allowed for all logged-in users. Admin is required for: user management, viewing all shares system-wide, scan-interval configuration changes, and retrying or purging files whose thumbnail generation has failed.

### CSRF & path safety

- Double-submit cookie CSRF token required on all non-GET endpoints
- All file paths are normalized and validated to remain within `/photos` (reject `..`, symlinks to outside, etc.)

---

## 7. Sharing

### External share-links

Created from UI: right-click folder → "Share via link".

Options per share:
- **Expiration**: 7, 30, 90 days, or never
- **Password**: optional; prompt before access, unlocked via share-scoped cookie
- **Allow download**: individual downloads + "Download all as ZIP". The ZIP is streamed on-the-fly without buffering the whole archive to disk, and file bytes are copied through unchanged (originals, no re-encoding or thumbnail substitution).
- **Allow upload**: when enabled, anonymous uploaders drop files into `/photos/{sharePath}/Uploads/{uploaderName|anonymous}/` (path configurable per share)
- **Revoke**: owner can deactivate any time (`revoked_at` set); after revocation, link returns 410 Gone

URL format: `https://{FRAMES_PUBLIC_URL}/s/{token}` where token is 32-char URL-safe random.

### Share-view UI

- No sidebar, no admin controls, no settings
- Scoped to the shared folder and its descendants (crumbs start at shared folder)
- Grid and lightbox work the same as authenticated view
- Rate limited: 100 req/min per token

### Internal shares (navigation hint)

User A marks folder X as "shared with" user B. User B gets a ⭐ marker in the sidebar and a "Shared with me" section for quick access. No notification/email in v1. No extra permissions granted — it's purely a bookmark.

---

## 8. Frontend

### Layout (Explorer-style)

- **Left sidebar**: folder tree (lazy-loaded children), drag-over highlight, right-click context menu, "Shared with me" section at top
- **Top bar**: clickable breadcrumb, debounced search input (`/api/search?q=&date_from=&date_to=&camera=`), "+ Upload" button, sort toggle (date/name/size), density toggle (3 levels)
- **Main**: virtualized grid (`svelte-virtual`), lazy `<img loading="lazy">` thumbs, shift-click multi-select, right-click context menu (Open, Download, Rename, Move to..., Delete, Share...)
- **Lightbox**: deep-linkable at `/file/{id}`, arrow L/R navigation, info panel with EXIF and shares list, download button, zoom+pan, native `<video>` with Range-based seeking

### Routes

| Path | Purpose |
|------|---------|
| `/` | Redirects to `/browse` or `/login` |
| `/login` | Login form |
| `/browse` | Root folder view |
| `/browse/{path}` | Specific folder |
| `/file/{id}` | Single-file lightbox (overlays browse) |
| `/shares` | My shares management |
| `/settings` | Profile, password change |
| `/admin` | Admin: users, scan status, failed thumbs (admin-only) |
| `/s/{token}` | Public share view (no auth required) |

### Upload

Drag-drop onto main view or folder-tree node. Chunked uploads (tus protocol or a simple chunked POST flow) for files > 100 MB. Per-file progress bar. Files land in the currently-selected folder.

### Mobile (< 768 px)

- Sidebar collapses into a hamburger drawer
- Grid density reduces to 2–3 columns
- Lightbox goes fullscreen with swipe L/R navigation
- Upload via the native file picker (opens the camera roll on iOS/Android)

---

## 9. Docker & deployment

### `docker-compose.yml` example

```yaml
services:
  frames:
    image: frames:latest
    container_name: frames
    ports:
      - "8080:8080"
    volumes:
      - /Users/niel/Pictures/Library:/photos:rw
      - frames_cache:/cache
      - frames_data:/data
    environment:
      - FRAMES_BIND=:8080
      - FRAMES_PHOTOS_ROOT=/photos
      - FRAMES_CACHE_DIR=/cache
      - FRAMES_DATA_DIR=/data
      - FRAMES_SCAN_INTERVAL=5m
      - FRAMES_FULL_SCAN_CRON=0 3 * * *
      - FRAMES_WORKERS=4
      - FRAMES_SESSION_SECRET=${FRAMES_SESSION_SECRET}
      - FRAMES_PUBLIC_URL=https://frames.example.com
      - FRAMES_ADMIN_USERNAME=${FRAMES_ADMIN_USERNAME}
      - FRAMES_ADMIN_PASSWORD=${FRAMES_ADMIN_PASSWORD}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/healthz"]
      interval: 30s

volumes:
  frames_cache:
  frames_data:
```

### Image build (multi-stage)

1. `node:alpine` — build SvelteKit to static files
2. `golang:alpine` — build Go binary with `embed` bundling stage 1 output
3. `alpine:latest` — install `libvips`, `libraw`, `ffmpeg`; copy binary; set entrypoint

### TLS

Not handled inside the image. README documents Caddy/Traefik examples for HTTPS termination. The `FRAMES_PUBLIC_URL` env var is used when generating share-link URLs.

---

## 10. Configuration

| Variable | Default | Purpose |
|----------|---------|---------|
| `FRAMES_BIND` | `:8080` | Listen address |
| `FRAMES_PHOTOS_ROOT` | `/photos` | Media root |
| `FRAMES_CACHE_DIR` | `/cache` | Thumbnails and previews |
| `FRAMES_DATA_DIR` | `/data` | SQLite DB directory |
| `FRAMES_SCAN_INTERVAL` | `5m` | Incremental scan period |
| `FRAMES_FULL_SCAN_CRON` | `0 3 * * *` | Full rescan schedule |
| `FRAMES_WORKERS` | `runtime.NumCPU()` | Thumbnail worker count |
| `FRAMES_SESSION_SECRET` | — (required) | Session signing secret |
| `FRAMES_PUBLIC_URL` | — (required) | Base URL for share-link generation |
| `FRAMES_MAX_UPLOAD_SIZE` | `5GB` | Authenticated upload limit |
| `FRAMES_SHARE_UPLOAD_MAX` | `500MB` | Anonymous upload-via-share limit |
| `FRAMES_ADMIN_USERNAME` | — (bootstrap) | First-run admin username |
| `FRAMES_ADMIN_PASSWORD` | — (bootstrap) | First-run admin password |

---

## 11. Out-of-scope and future work

### Explicitly not in v1

- Face recognition / AI tagging
- Album concept separate from folders
- Persistent photo edits (crop, rotate, color)
- Multi-tenant (multiple libraries per instance)
- Native mobile apps
- Sync clients (Lightroom plugin, etc.)
- Email notifications

### Candidates for v2

- Geo-map view from EXIF GPS data
- Smarter search filters (dominant color, recognized objects)
- Duplicate-file detection
- Email notifications for internal shares

---

## 12. Open questions (to resolve during implementation)

1. **Tombstones vs hard deletes**: when the scanner notices a file is gone, is it deleted or tombstoned (to preserve share-link integrity)? Default: hard delete, cascade share-links. Revisit if we see share-breakage.
2. **Upload chunking protocol**: `tus` (mature spec, extra client library) vs custom chunked POST. Lean toward `tus` for resumability but pick during implementation.
3. **Password reset**: v1 has no self-service password reset. Admin resets via CLI or admin UI. Acceptable for family use; reconsider for broader deployments.

---
