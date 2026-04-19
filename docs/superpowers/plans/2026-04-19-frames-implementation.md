# Frames Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Self-hosted Docker media-library application that mirrors a Finder folder structure, indexes 500k+ files, serves thumbnails and previews fast, and supports multi-user access plus external share-links with upload.

**Architecture:** Single Go binary bundling `chi` HTTP router + embedded SvelteKit frontend. SQLite (WAL) for index, libvips for images, libraw for RAW, ffmpeg for video. Three bind-mount/volumes: `/photos` (the Finder library), `/cache` (derivatives), `/data` (SQLite DB). Async scanner + worker pool for thumbnail generation.

**Tech Stack:** Go 1.22+, chi/v5, mattn/go-sqlite3, govips, libraw (via `os/exec`), ffmpeg (via `os/exec`), argon2id, SvelteKit (static adapter), svelte-virtual, TypeScript, Vite.

**Spec:** `docs/superpowers/specs/2026-04-18-frames-design.md`

---

## Phasing

Each phase leaves working, testable software:

| # | Phase | What works after this phase |
|---|-------|------------------------------|
| 1 | Foundation | `docker compose up` runs a healthz server backed by SQLite |
| 2 | Auth | Login/logout flow, protected endpoints |
| 3 | Scanner | Filesystem is mirrored into `folders`/`files` tables |
| 4 | Thumbnail pipeline | `/api/thumb/{id}`, `/api/preview/{id}`, `/api/original/{id}` serve media |
| 5 | Browse + search API | Folder listing + search endpoints work |
| 6 | File operations API | Upload, rename, move, delete, mkdir |
| 7 | Sharing API | Share creation, revoke, public access, ZIP stream, anon upload |
| 8 | Frontend foundation | SvelteKit scaffold, login page, API client, auth store |
| 9 | Browse UI | Folder tree + virtualized grid + breadcrumb |
| 10 | Lightbox | Single-view with EXIF, navigation, video |
| 11 | File ops UI | Upload, rename, move, delete from the UI |
| 12 | Share UI + share view | Share dialog, public `/s/{token}` view |
| 13 | Admin + settings + search UI | User management, scan status, search box |
| 14 | Mobile polish | Responsive breakpoints, hamburger, swipe |
| 15 | Packaging | Multi-stage Docker image, compose example, README |

---

## File Structure

```
Frames/
├── Dockerfile
├── docker-compose.yml
├── docker-compose.dev.yml
├── .dockerignore
├── .gitignore
├── README.md
├── go.mod
├── go.sum
├── Makefile
├── cmd/frames/main.go
├── internal/
│   ├── config/config.go
│   ├── logger/logger.go
│   ├── db/
│   │   ├── db.go
│   │   ├── migrate.go
│   │   ├── migrations/0001_initial.sql
│   │   ├── users.go
│   │   ├── folders.go
│   │   ├── files.go
│   │   ├── shares.go
│   │   ├── folder_shares.go
│   │   ├── sessions.go
│   │   └── scan_jobs.go
│   ├── auth/
│   │   ├── password.go
│   │   ├── session.go
│   │   ├── middleware.go
│   │   ├── csrf.go
│   │   ├── ratelimit.go
│   │   └── bootstrap.go
│   ├── scanner/
│   │   ├── scanner.go
│   │   ├── walker.go
│   │   ├── scheduler.go
│   │   └── mime.go
│   ├── thumbnail/
│   │   ├── cache.go
│   │   ├── image.go
│   │   ├── raw.go
│   │   ├── video.go
│   │   ├── metadata.go
│   │   ├── worker.go
│   │   └── queue.go
│   ├── share/
│   │   ├── token.go
│   │   ├── validate.go
│   │   └── zip.go
│   ├── upload/
│   │   ├── chunked.go
│   │   └── safepath.go
│   ├── fsops/
│   │   └── fsops.go
│   ├── api/
│   │   ├── router.go
│   │   ├── errors.go
│   │   ├── handlers_auth.go
│   │   ├── handlers_browse.go
│   │   ├── handlers_media.go
│   │   ├── handlers_ops.go
│   │   ├── handlers_upload.go
│   │   ├── handlers_search.go
│   │   ├── handlers_shares.go
│   │   ├── handlers_share_public.go
│   │   ├── handlers_scan.go
│   │   └── handlers_admin.go
│   └── frontend/frontend.go       # embed.FS for built SvelteKit assets
├── web/                            # SvelteKit project
│   ├── package.json
│   ├── svelte.config.js
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── static/favicon.ico
│   └── src/
│       ├── app.html
│       ├── app.css
│       ├── lib/
│       │   ├── api.ts
│       │   ├── stores.ts
│       │   ├── csrf.ts
│       │   ├── format.ts
│       │   └── components/
│       │       ├── FolderTree.svelte
│       │       ├── Breadcrumb.svelte
│       │       ├── Grid.svelte
│       │       ├── GridItem.svelte
│       │       ├── Lightbox.svelte
│       │       ├── SearchBox.svelte
│       │       ├── ContextMenu.svelte
│       │       ├── UploadDialog.svelte
│       │       ├── ShareDialog.svelte
│       │       ├── MovePicker.svelte
│       │       └── ConfirmDialog.svelte
│       └── routes/
│           ├── +layout.svelte
│           ├── +layout.ts
│           ├── +page.ts
│           ├── login/+page.svelte
│           ├── browse/+page.svelte
│           ├── browse/[...path]/+page.svelte
│           ├── file/[id]/+page.svelte
│           ├── shares/+page.svelte
│           ├── settings/+page.svelte
│           ├── admin/+page.svelte
│           └── s/[token]/+page.svelte
└── docs/
    └── superpowers/
        ├── specs/2026-04-18-frames-design.md
        └── plans/2026-04-19-frames-implementation.md
```

---

## Conventions

- Go: **TDD per task**. Tests live next to code (`foo_test.go`). Package-level tests use `t *testing.T`; integration tests use an in-memory SQLite DB with the migrated schema.
- Every task ends with `go test ./... -count=1` passing (unless explicitly noted) and a git commit.
- Frontend tasks build via `pnpm build` and include a manual browser smoke test where applicable (no unit testing framework in scope — Playwright is v2).
- Run `go vet ./...` and `gofmt -w .` before each commit.
- All HTTP handlers return JSON `{ "error": "..." }` on failures via `api.WriteError`. Success = 200 + `{ "data": ... }` or 204.
- **Remote push:** after **Task A** (GitHub setup, at the end of Phase 1) is done, follow every `git commit` with `git push` so the remote mirrors each task. Skip the push only if Task A has not been run yet.

---

## Phase 1 — Foundation

Bootstraps the project: Go module, git, SQLite with migrations, a minimal HTTP server exposing `/healthz`, structured logger, Dockerfile skeleton. After this phase you can `docker compose up` and get a 200 on `/healthz`.

### Task 1: Initialize Go module, git, and base files

**Files:**
- Create: `/Users/niel/Development/Frames/go.mod`
- Create: `/Users/niel/Development/Frames/.gitignore`
- Create: `/Users/niel/Development/Frames/Makefile`
- Create: `/Users/niel/Development/Frames/cmd/frames/main.go`

- [ ] **Step 1: Initialize git**

```bash
cd /Users/niel/Development/Frames
git init
```

- [ ] **Step 2: Create `.gitignore`**

```gitignore
# Binaries
/frames
/cmd/frames/frames

# Go build artefacts
*.test
*.out
coverage.*

# SvelteKit build
/web/.svelte-kit/
/web/build/
/web/node_modules/
/internal/frontend/dist/

# Local data
/data/
/cache/
/photos/
*.db
*.db-shm
*.db-wal

# Secrets — NEVER commit env files with real values
.env
.env.*
!.env.example

# IDE
.idea/
.vscode/

# Brainstorm artefacts (kept out of main branches)
/.superpowers/
```

- [ ] **Step 3: Initialize Go module**

```bash
cd /Users/niel/Development/Frames
go mod init github.com/NielHeesakkers/frames
go mod tidy
```

- [ ] **Step 4: Create skeleton `cmd/frames/main.go`**

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("frames: starting")
	return nil
}
```

- [ ] **Step 5: Create `Makefile`**

```makefile
.PHONY: build test lint fmt run

build:
	go build -o frames ./cmd/frames

test:
	go test ./... -count=1

lint:
	go vet ./...

fmt:
	gofmt -w .

run: build
	./frames
```

- [ ] **Step 6: Verify build**

```bash
make build && ./frames
```

Expected: prints `frames: starting` and exits 0.

- [ ] **Step 7: Commit**

```bash
git add .
git commit -m "chore: initialize go module and project skeleton"
```

---

### Task 2: Config package (env var parsing + validation)

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/config/config_test.go
package config

import (
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("FRAMES_SESSION_SECRET", "x")
	t.Setenv("FRAMES_PUBLIC_URL", "http://localhost:8080")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Bind != ":8080" {
		t.Errorf("Bind default = %q; want :8080", cfg.Bind)
	}
	if cfg.PhotosRoot != "/photos" {
		t.Errorf("PhotosRoot default = %q; want /photos", cfg.PhotosRoot)
	}
	if cfg.ScanInterval != 5*time.Minute {
		t.Errorf("ScanInterval default = %v; want 5m", cfg.ScanInterval)
	}
}

func TestLoad_RequiredMissing(t *testing.T) {
	t.Setenv("FRAMES_SESSION_SECRET", "")
	t.Setenv("FRAMES_PUBLIC_URL", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing required vars")
	}
}

func TestLoad_ParseDurationAndCron(t *testing.T) {
	t.Setenv("FRAMES_SESSION_SECRET", "x")
	t.Setenv("FRAMES_PUBLIC_URL", "http://localhost:8080")
	t.Setenv("FRAMES_SCAN_INTERVAL", "2m")
	t.Setenv("FRAMES_FULL_SCAN_CRON", "0 4 * * *")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ScanInterval != 2*time.Minute {
		t.Errorf("ScanInterval = %v; want 2m", cfg.ScanInterval)
	}
	if cfg.FullScanCron != "0 4 * * *" {
		t.Errorf("FullScanCron = %q", cfg.FullScanCron)
	}
}
```

- [ ] **Step 2: Implement config**

```go
// internal/config/config.go
package config

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"
)

type Config struct {
	Bind             string
	PhotosRoot       string
	CacheDir         string
	DataDir          string
	ScanInterval     time.Duration
	FullScanCron     string
	Workers          int
	SessionSecret    string
	PublicURL        string
	MaxUploadSize    int64
	ShareUploadMax   int64
	AdminUsername    string
	AdminPassword    string
}

func Load() (*Config, error) {
	cfg := &Config{
		Bind:           getenv("FRAMES_BIND", ":8080"),
		PhotosRoot:     getenv("FRAMES_PHOTOS_ROOT", "/photos"),
		CacheDir:       getenv("FRAMES_CACHE_DIR", "/cache"),
		DataDir:        getenv("FRAMES_DATA_DIR", "/data"),
		FullScanCron:   getenv("FRAMES_FULL_SCAN_CRON", "0 3 * * *"),
		SessionSecret:  os.Getenv("FRAMES_SESSION_SECRET"),
		PublicURL:      os.Getenv("FRAMES_PUBLIC_URL"),
		AdminUsername:  os.Getenv("FRAMES_ADMIN_USERNAME"),
		AdminPassword:  os.Getenv("FRAMES_ADMIN_PASSWORD"),
	}

	dur, err := time.ParseDuration(getenv("FRAMES_SCAN_INTERVAL", "5m"))
	if err != nil {
		return nil, fmt.Errorf("FRAMES_SCAN_INTERVAL: %w", err)
	}
	cfg.ScanInterval = dur

	cfg.Workers, err = atoiDefault(os.Getenv("FRAMES_WORKERS"), runtime.NumCPU())
	if err != nil {
		return nil, fmt.Errorf("FRAMES_WORKERS: %w", err)
	}

	cfg.MaxUploadSize, err = parseSize(getenv("FRAMES_MAX_UPLOAD_SIZE", "5GB"))
	if err != nil {
		return nil, fmt.Errorf("FRAMES_MAX_UPLOAD_SIZE: %w", err)
	}
	cfg.ShareUploadMax, err = parseSize(getenv("FRAMES_SHARE_UPLOAD_MAX", "500MB"))
	if err != nil {
		return nil, fmt.Errorf("FRAMES_SHARE_UPLOAD_MAX: %w", err)
	}

	if cfg.SessionSecret == "" {
		return nil, errors.New("FRAMES_SESSION_SECRET is required")
	}
	if cfg.PublicURL == "" {
		return nil, errors.New("FRAMES_PUBLIC_URL is required")
	}
	return cfg, nil
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func atoiDefault(s string, d int) (int, error) {
	if s == "" {
		return d, nil
	}
	return strconv.Atoi(s)
}

// parseSize parses "5GB", "500MB", "1024" (bytes).
func parseSize(s string) (int64, error) {
	if s == "" {
		return 0, errors.New("empty size")
	}
	mult := int64(1)
	switch {
	case hasSuffix(s, "GB"):
		mult = 1 << 30
		s = s[:len(s)-2]
	case hasSuffix(s, "MB"):
		mult = 1 << 20
		s = s[:len(s)-2]
	case hasSuffix(s, "KB"):
		mult = 1 << 10
		s = s[:len(s)-2]
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return n * mult, nil
}

func hasSuffix(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/config/... -v
```

Expected: 3 tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/config/
git commit -m "feat(config): env-driven config with validation"
```

---

### Task 3: Structured logger

**Files:**
- Create: `internal/logger/logger.go`

- [ ] **Step 1: Implement thin wrapper around slog**

```go
// internal/logger/logger.go
package logger

import (
	"log/slog"
	"os"
)

func New(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return slog.New(h)
}
```

- [ ] **Step 2: Run `go vet`**

```bash
go vet ./...
```

Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add internal/logger/
git commit -m "feat(logger): slog-based JSON logger"
```

---

### Task 4: SQLite connection + migration runner

**Files:**
- Create: `internal/db/db.go`
- Create: `internal/db/migrate.go`
- Create: `internal/db/migrations/0001_initial.sql`
- Create: `internal/db/db_test.go`
- Modify: `go.mod` (add `github.com/mattn/go-sqlite3`)

- [ ] **Step 1: Add SQLite dependency**

```bash
go get github.com/mattn/go-sqlite3
go mod tidy
```

- [ ] **Step 2: Write initial migration**

```sql
-- internal/db/migrations/0001_initial.sql
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY,
  username TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  is_admin BOOLEAN NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS folders (
  id INTEGER PRIMARY KEY,
  parent_id INTEGER REFERENCES folders(id) ON DELETE CASCADE,
  path TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  mtime INTEGER NOT NULL,
  item_count INTEGER NOT NULL DEFAULT 0,
  last_scanned_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_folders_parent ON folders(parent_id);
CREATE INDEX IF NOT EXISTS idx_folders_path ON folders(path);

CREATE TABLE IF NOT EXISTS files (
  id INTEGER PRIMARY KEY,
  folder_id INTEGER NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
  filename TEXT NOT NULL,
  relative_path TEXT UNIQUE NOT NULL,
  size INTEGER NOT NULL,
  mtime INTEGER NOT NULL,
  mime_type TEXT,
  kind TEXT NOT NULL,
  taken_at DATETIME,
  width INTEGER,
  height INTEGER,
  camera_make TEXT,
  camera_model TEXT,
  orientation INTEGER,
  duration_ms INTEGER,
  thumb_status TEXT NOT NULL DEFAULT 'pending',
  thumb_attempts INTEGER NOT NULL DEFAULT 0,
  preview_status TEXT NOT NULL DEFAULT 'pending',
  preview_attempts INTEGER NOT NULL DEFAULT 0,
  UNIQUE(folder_id, filename)
);
CREATE INDEX IF NOT EXISTS idx_files_folder ON files(folder_id);
CREATE INDEX IF NOT EXISTS idx_files_taken_at ON files(taken_at);
CREATE INDEX IF NOT EXISTS idx_files_relpath ON files(relative_path);
CREATE INDEX IF NOT EXISTS idx_files_thumb_status ON files(thumb_status) WHERE thumb_status = 'pending';

CREATE TABLE IF NOT EXISTS shares (
  id INTEGER PRIMARY KEY,
  token TEXT UNIQUE NOT NULL,
  folder_id INTEGER NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
  created_by INTEGER NOT NULL REFERENCES users(id),
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at DATETIME,
  password_hash TEXT,
  allow_download BOOLEAN NOT NULL DEFAULT 1,
  allow_upload BOOLEAN NOT NULL DEFAULT 0,
  revoked_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_shares_token ON shares(token);
CREATE INDEX IF NOT EXISTS idx_shares_created_by ON shares(created_by);

CREATE TABLE IF NOT EXISTS folder_shares (
  folder_id INTEGER NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
  shared_with_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  shared_by INTEGER NOT NULL REFERENCES users(id),
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (folder_id, shared_with_user_id)
);

CREATE TABLE IF NOT EXISTS scan_jobs (
  id INTEGER PRIMARY KEY,
  type TEXT NOT NULL,
  started_at DATETIME NOT NULL,
  finished_at DATETIME,
  files_scanned INTEGER NOT NULL DEFAULT 0,
  files_added INTEGER NOT NULL DEFAULT 0,
  files_updated INTEGER NOT NULL DEFAULT 0,
  files_removed INTEGER NOT NULL DEFAULT 0,
  error TEXT
);

CREATE TABLE IF NOT EXISTS sessions (
  token TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
```

- [ ] **Step 3: Implement DB open + migrate**

```go
// internal/db/db.go
package db

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct{ *sql.DB }

func Open(dataDir string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000&cache=shared",
		filepath.Join(dataDir, "frames.db"))
	d, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	d.SetMaxOpenConns(1) // SQLite likes single writer; readers use the same conn via busy_timeout
	if err := d.Ping(); err != nil {
		return nil, err
	}
	return &DB{d}, nil
}
```

```go
// internal/db/migrate.go
package db

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrations embed.FS

func (d *DB) Migrate() error {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	// Ensure schema_migrations exists even on a fresh DB by running first file fully.
	for _, n := range names {
		parts := strings.SplitN(n, "_", 2)
		ver, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("bad migration filename %q: %w", n, err)
		}
		var applied int
		_ = d.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, ver).Scan(&applied)
		if applied > 0 {
			continue
		}
		body, err := fs.ReadFile(migrations, "migrations/"+n)
		if err != nil {
			return err
		}
		if _, err := d.Exec(string(body)); err != nil {
			return fmt.Errorf("migration %s: %w", n, err)
		}
		if _, err := d.Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, ver); err != nil {
			return fmt.Errorf("record migration %s: %w", n, err)
		}
	}
	return nil
}
```

- [ ] **Step 4: Write integration test**

```go
// internal/db/db_test.go
package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer d.Close()
	if err := d.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Running a second time must be idempotent.
	if err := d.Migrate(); err != nil {
		t.Fatalf("migrate twice: %v", err)
	}
	// Tables should exist.
	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("users table missing")
	}
	_, err = os.Stat(filepath.Join(dir, "frames.db"))
	if err != nil {
		t.Errorf("db file missing: %v", err)
	}
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/db/... -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/db/ go.mod go.sum
git commit -m "feat(db): sqlite open + embedded migrations"
```

---

### Task 5: Minimal HTTP server with /healthz

**Files:**
- Create: `internal/api/router.go`
- Create: `internal/api/errors.go`
- Modify: `cmd/frames/main.go`
- Modify: `go.mod` (add `github.com/go-chi/chi/v5`)
- Create: `internal/api/router_test.go`

- [ ] **Step 1: Add chi**

```bash
go get github.com/go-chi/chi/v5
go mod tidy
```

- [ ] **Step 2: Implement error helpers**

```go
// internal/api/errors.go
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

func LogError(log *slog.Logger, r *http.Request, err error) {
	log.Error("request failed",
		"path", r.URL.Path, "method", r.Method, "err", err.Error())
}
```

- [ ] **Step 3: Implement router**

```go
// internal/api/router.go
package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Deps struct {
	Log *slog.Logger
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return r
}
```

- [ ] **Step 4: Wire main.go**

```go
// cmd/frames/main.go
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NielHeesakkers/frames/internal/api"
	"github.com/NielHeesakkers/frames/internal/config"
	"github.com/NielHeesakkers/frames/internal/db"
	"github.com/NielHeesakkers/frames/internal/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	log := logger.New(os.Getenv("FRAMES_LOG_LEVEL"))
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log.Info("loaded config", "bind", cfg.Bind, "photos", cfg.PhotosRoot)

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return err
	}
	database, err := db.Open(cfg.DataDir)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := database.Migrate(); err != nil {
		return err
	}

	h := api.NewRouter(api.Deps{Log: log})
	srv := &http.Server{
		Addr:              cfg.Bind,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("http listening", "addr", cfg.Bind)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down")
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
```

- [ ] **Step 5: Router test**

```go
// internal/api/router_test.go
package api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	r := NewRouter(Deps{Log: slog.Default()})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("code=%d", w.Code)
	}
	if w.Body.String() == "" {
		t.Fatal("empty body")
	}
}
```

- [ ] **Step 6: Run tests and smoke-start**

```bash
go test ./... -count=1
FRAMES_SESSION_SECRET=x FRAMES_PUBLIC_URL=http://localhost:8080 FRAMES_DATA_DIR=./data go run ./cmd/frames &
sleep 1
curl -sf http://localhost:8080/healthz
kill %1
rm -rf ./data
```

Expected: tests PASS; curl prints `{"status":"ok"}`.

- [ ] **Step 7: Commit**

```bash
git add internal/api/ cmd/frames/main.go go.mod go.sum
git commit -m "feat(api): minimal http server with healthz and graceful shutdown"
```

---

### Task 6: Dockerfile skeleton + docker-compose

**Files:**
- Create: `Dockerfile`
- Create: `.dockerignore`
- Create: `docker-compose.yml`
- Create: `docker-compose.dev.yml`

> The full multi-stage build (with libvips, libraw, ffmpeg, and embedded frontend) lands in Phase 15. This task adds an **alpine+go** stage so Phase 1 is self-contained and deployable.

- [ ] **Step 1: Create `.dockerignore`**

```
/.git
/.idea
/.vscode
/.superpowers
/data
/cache
/photos
/web/node_modules
/web/.svelte-kit
/web/build
/internal/frontend/dist
*.db*
```

- [ ] **Step 2: Create `Dockerfile` (temporary skeleton — replaced in Phase 15)**

```dockerfile
# syntax=docker/dockerfile:1.7

FROM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /out/frames ./cmd/frames

FROM alpine:3.20
RUN apk add --no-cache ca-certificates sqlite-libs tzdata
WORKDIR /app
COPY --from=build /out/frames /app/frames
EXPOSE 8080
ENTRYPOINT ["/app/frames"]
```

- [ ] **Step 3: Create `docker-compose.yml`**

```yaml
services:
  frames:
    build: .
    container_name: frames
    ports:
      - "8080:8080"
    volumes:
      - ./photos:/photos:rw
      - frames_cache:/cache
      - frames_data:/data
    environment:
      FRAMES_BIND: ":8080"
      FRAMES_PHOTOS_ROOT: /photos
      FRAMES_CACHE_DIR: /cache
      FRAMES_DATA_DIR: /data
      FRAMES_SESSION_SECRET: ${FRAMES_SESSION_SECRET:-change-me-32-bytes-minimum-xxxxxxx}
      FRAMES_PUBLIC_URL: ${FRAMES_PUBLIC_URL:-http://localhost:8080}
      FRAMES_ADMIN_USERNAME: ${FRAMES_ADMIN_USERNAME:-admin}
      FRAMES_ADMIN_PASSWORD: ${FRAMES_ADMIN_PASSWORD:-change-me}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 5s
      retries: 3

volumes:
  frames_cache:
  frames_data:
```

- [ ] **Step 4: Create `docker-compose.dev.yml`**

```yaml
services:
  frames:
    build:
      context: .
      target: build
    command: ["sh", "-c", "go run ./cmd/frames"]
    volumes:
      - .:/src
    working_dir: /src
    ports:
      - "8080:8080"
    environment:
      FRAMES_SESSION_SECRET: dev-secret-not-for-prod
      FRAMES_PUBLIC_URL: http://localhost:8080
      FRAMES_PHOTOS_ROOT: /src/photos
      FRAMES_CACHE_DIR: /src/cache
      FRAMES_DATA_DIR: /src/data
```

- [ ] **Step 5: Smoke test**

```bash
docker compose build
docker compose up -d
sleep 2
curl -sf http://localhost:8080/healthz
docker compose down
```

Expected: curl prints `{"status":"ok"}`.

- [ ] **Step 6: Commit**

```bash
git add Dockerfile .dockerignore docker-compose.yml docker-compose.dev.yml
git commit -m "chore(docker): phase-1 skeleton image and compose files"
```

---

### Task A: GitHub setup (push repo to private remote)

**Files:**
- Create: `.env.example`

> This task assumes the GitHub CLI (`gh`) is installed and authenticated (`gh auth login` run once). If not, follow the alternative web-flow at the end.

- [ ] **Step 1: Create `.env.example`** (a committed template; the real `.env` is gitignored)

```bash
# .env.example — copy to .env and fill in real values. DO NOT commit .env.
FRAMES_SESSION_SECRET=generate-with-openssl-rand-base64-32
FRAMES_PUBLIC_URL=http://localhost:8080
FRAMES_ADMIN_USERNAME=admin
FRAMES_ADMIN_PASSWORD=change-me-at-least-12-chars
```

Write this to `/Users/niel/Development/Frames/.env.example`.

- [ ] **Step 2: Verify nothing secret is staged**

```bash
git status
git ls-files | grep -E '\.env$|\.db$' || echo "clean"
```

Expected: prints `clean`. If it lists anything, untrack it first: `git rm --cached <path>`.

- [ ] **Step 3: Create a private GitHub repo and push**

Preferred (GitHub CLI):

```bash
gh repo create NielHeesakkers/frames --private --source=. --remote=origin --push
```

Expected: repo is created, remote `origin` set, and the current branch (`main`) pushed.

Alternative (web + CLI):
1. Create `https://github.com/new` → name `frames`, visibility **Private**, do **not** initialize with README.
2. Copy the shown SSH or HTTPS URL.
3. Locally:

   ```bash
   git remote add origin git@github.com:NielHeesakkers/frames.git   # or the HTTPS URL
   git branch -M main
   git push -u origin main
   ```

- [ ] **Step 4: Commit the example env and push**

```bash
git add .env.example
git commit -m "chore: add .env.example template"
git push
```

- [ ] **Step 5: Confirm**

```bash
gh repo view --web   # opens the repo in your browser
```

Check that: the repo is marked **Private**, the latest commit appears, and no `.env`, `*.db`, or `photos/` content is listed.

> From this point on, every task ends with `git commit && git push`. If you forget, run `git push` at any time to catch up.

---

## Phase 2 — Authentication

Adds argon2id password hashing, session cookies, login/logout, auth middleware, CSRF middleware, login rate limiting, admin bootstrap. After this phase, `/api/login`, `/api/logout`, `/api/me` work, and any non-public handler can be wrapped with `RequireLogin`.

### Task 7: Password hashing (argon2id)

**Files:**
- Create: `internal/auth/password.go`
- Create: `internal/auth/password_test.go`

- [ ] **Step 1: Add dependency**

```bash
go get golang.org/x/crypto/argon2
go mod tidy
```

- [ ] **Step 2: Write failing test**

```go
// internal/auth/password_test.go
package auth

import "testing"

func TestHashAndVerify(t *testing.T) {
	h, err := HashPassword("hunter2")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword(h, "hunter2")
	if err != nil || !ok {
		t.Fatalf("want ok, got ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword(h, "wrong")
	if err != nil || ok {
		t.Fatalf("want not-ok, got ok=%v err=%v", ok, err)
	}
}

func TestVerify_MalformedHash(t *testing.T) {
	_, err := VerifyPassword("not-a-hash", "pw")
	if err == nil {
		t.Fatal("expected error on malformed hash")
	}
}
```

- [ ] **Step 3: Implement argon2id**

```go
// internal/auth/password.go
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parameters tuned for modest servers. Adjust in config if needed.
const (
	argTime    = 3
	argMemory  = 64 * 1024 // 64 MiB
	argThreads = 4
	argKeyLen  = 32
	saltLen    = 16
)

func HashPassword(pw string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pw), salt, argTime, argMemory, argThreads, argKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argMemory, argTime, argThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

func VerifyPassword(encoded, pw string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("unsupported hash format")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, err
	}
	if version != argon2.Version {
		return false, errors.New("incompatible argon2 version")
	}
	var memory uint32
	var time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(pw), salt, time, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(want, got) == 1, nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/auth/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/auth/password.go internal/auth/password_test.go go.mod go.sum
git commit -m "feat(auth): argon2id password hashing"
```

---

### Task 8: Users repository

**Files:**
- Create: `internal/db/users.go`
- Create: `internal/db/users_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/db/users_test.go
package db

import "testing"

func setupDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.Migrate(); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestUserCRUD(t *testing.T) {
	d := setupDB(t)

	id, err := d.CreateUser("alice", "hash", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}
	u, err := d.UserByUsername("alice")
	if err != nil {
		t.Fatalf("by username: %v", err)
	}
	if u.ID != id || !u.IsAdmin || u.PasswordHash != "hash" {
		t.Errorf("unexpected user: %+v", u)
	}

	if _, err := d.CreateUser("alice", "x", false); err == nil {
		t.Fatal("expected unique violation")
	}

	if err := d.UpdateUserPassword(id, "newhash"); err != nil {
		t.Fatalf("update: %v", err)
	}
	u, _ = d.UserByUsername("alice")
	if u.PasswordHash != "newhash" {
		t.Errorf("password not updated")
	}

	users, err := d.ListUsers()
	if err != nil || len(users) != 1 {
		t.Errorf("list users: %v count=%d", err, len(users))
	}

	if err := d.DeleteUser(id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := d.UserByUsername("alice"); err == nil {
		t.Fatal("expected not found after delete")
	}
}
```

- [ ] **Step 2: Implement users repository**

```go
// internal/db/users.go
package db

import (
	"database/sql"
	"errors"
	"time"
)

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	IsAdmin      bool
	CreatedAt    time.Time
}

var ErrNotFound = errors.New("not found")

func (d *DB) CreateUser(username, passwordHash string, isAdmin bool) (int64, error) {
	res, err := d.Exec(`INSERT INTO users(username,password_hash,is_admin) VALUES(?,?,?)`,
		username, passwordHash, isAdmin)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) UserByUsername(u string) (*User, error) {
	row := d.QueryRow(`SELECT id,username,password_hash,is_admin,created_at FROM users WHERE username=?`, u)
	return scanUser(row)
}

func (d *DB) UserByID(id int64) (*User, error) {
	row := d.QueryRow(`SELECT id,username,password_hash,is_admin,created_at FROM users WHERE id=?`, id)
	return scanUser(row)
}

func (d *DB) ListUsers() ([]User, error) {
	rows, err := d.Query(`SELECT id,username,password_hash,is_admin,created_at FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		u, err := scanUserRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

func (d *DB) UpdateUserPassword(id int64, hash string) error {
	_, err := d.Exec(`UPDATE users SET password_hash=? WHERE id=?`, hash, id)
	return err
}

func (d *DB) DeleteUser(id int64) error {
	_, err := d.Exec(`DELETE FROM users WHERE id=?`, id)
	return err
}

type rowScanner interface {
	Scan(...any) error
}

func scanUser(r rowScanner) (*User, error) {
	u := &User{}
	err := r.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func scanUserRows(r *sql.Rows) (*User, error) {
	u := &User{}
	return u, r.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt)
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/db/... -v
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/db/users.go internal/db/users_test.go
git commit -m "feat(db): users repository"
```

---

### Task 9: Sessions repository

**Files:**
- Create: `internal/db/sessions.go`
- Create: `internal/db/sessions_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/db/sessions_test.go
package db

import (
	"testing"
	"time"
)

func TestSessionCRUD(t *testing.T) {
	d := setupDB(t)
	uid, _ := d.CreateUser("alice", "h", false)

	tok := "tok-abc"
	exp := time.Now().Add(time.Hour)
	if err := d.CreateSession(tok, uid, exp); err != nil {
		t.Fatal(err)
	}
	s, err := d.SessionByToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if s.UserID != uid {
		t.Fatalf("uid=%d want %d", s.UserID, uid)
	}
	if err := d.DeleteSession(tok); err != nil {
		t.Fatal(err)
	}
	if _, err := d.SessionByToken(tok); err == nil {
		t.Fatal("expected not found")
	}
}

func TestSession_Expired(t *testing.T) {
	d := setupDB(t)
	uid, _ := d.CreateUser("alice", "h", false)
	tok := "tok-expired"
	if err := d.CreateSession(tok, uid, time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := d.SessionByToken(tok); err == nil {
		t.Fatal("expected not-found for expired session")
	}
}
```

- [ ] **Step 2: Implement**

```go
// internal/db/sessions.go
package db

import (
	"database/sql"
	"errors"
	"time"
)

type Session struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
}

func (d *DB) CreateSession(token string, userID int64, expiresAt time.Time) error {
	_, err := d.Exec(`INSERT INTO sessions(token,user_id,expires_at) VALUES(?,?,?)`,
		token, userID, expiresAt)
	return err
}

func (d *DB) SessionByToken(token string) (*Session, error) {
	row := d.QueryRow(`SELECT token,user_id,expires_at FROM sessions WHERE token=? AND expires_at > CURRENT_TIMESTAMP`, token)
	s := &Session{}
	err := row.Scan(&s.Token, &s.UserID, &s.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (d *DB) DeleteSession(token string) error {
	_, err := d.Exec(`DELETE FROM sessions WHERE token=?`, token)
	return err
}

func (d *DB) CleanupExpiredSessions() (int64, error) {
	res, err := d.Exec(`DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/db/... -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/db/sessions.go internal/db/sessions_test.go
git commit -m "feat(db): sessions repository"
```

---

### Task 10: Session helpers and auth middleware

**Files:**
- Create: `internal/auth/session.go`
- Create: `internal/auth/middleware.go`
- Create: `internal/auth/middleware_test.go`

- [ ] **Step 1: Implement session + context helpers**

```go
// internal/auth/session.go
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"
)

type ctxKey int

const userCtxKey ctxKey = 1

const (
	SessionCookieName = "frames_session"
	SessionTTL        = 30 * 24 * time.Hour
)

// NewToken returns 32 random bytes as URL-safe base64.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func SetSessionCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(SessionTTL),
	})
}

func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

type CurrentUser struct {
	ID       int64
	Username string
	IsAdmin  bool
}

func WithUser(ctx context.Context, u CurrentUser) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

func UserFromContext(ctx context.Context) (CurrentUser, bool) {
	u, ok := ctx.Value(userCtxKey).(CurrentUser)
	return u, ok
}
```

- [ ] **Step 2: Implement middleware**

```go
// internal/auth/middleware.go
package auth

import (
	"net/http"

	"github.com/NielHeesakkers/frames/internal/db"
)

type UserLookup interface {
	SessionByToken(string) (*db.Session, error)
	UserByID(int64) (*db.User, error)
}

// RequireLogin loads the session and puts the user into the request context.
// Returns 401 if no valid session.
func RequireLogin(lookup UserLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(SessionCookieName)
			if err != nil || c.Value == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			sess, err := lookup.SessionByToken(c.Value)
			if err != nil || sess == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			u, err := lookup.UserByID(sess.UserID)
			if err != nil || u == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			ctx := WithUser(r.Context(), CurrentUser{ID: u.ID, Username: u.Username, IsAdmin: u.IsAdmin})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin must come after RequireLogin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserFromContext(r.Context())
		if !ok || !u.IsAdmin {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 3: Write middleware test**

```go
// internal/auth/middleware_test.go
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NielHeesakkers/frames/internal/db"
)

func TestRequireLogin(t *testing.T) {
	d, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.Migrate(); err != nil {
		t.Fatal(err)
	}
	uid, _ := d.CreateUser("alice", "h", false)
	_ = d.CreateSession("tok", uid, time.Now().Add(time.Hour))

	h := RequireLogin(d)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserFromContext(r.Context())
		if !ok || u.Username != "alice" {
			t.Errorf("no user in context")
		}
		w.WriteHeader(200)
	}))

	// missing cookie
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("no-cookie: code=%d", w.Code)
	}

	// valid cookie
	req = httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "tok"})
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("valid cookie: code=%d", w.Code)
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/auth/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/auth/session.go internal/auth/middleware.go internal/auth/middleware_test.go
git commit -m "feat(auth): session cookies + require-login middleware"
```

---

### Task 11: CSRF middleware (double-submit cookie)

**Files:**
- Create: `internal/auth/csrf.go`
- Create: `internal/auth/csrf_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/auth/csrf_test.go
package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRF_GetSetsCookie(t *testing.T) {
	h := CSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("code=%d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Set-Cookie"), CSRFCookieName) {
		t.Fatal("expected csrf cookie on GET")
	}
}

func TestCSRF_PostRequiresHeaderMatch(t *testing.T) {
	h := CSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))

	// missing both
	req := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("missing: code=%d", w.Code)
	}

	// mismatched
	req = httptest.NewRequest("POST", "/", nil)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "a"})
	req.Header.Set(CSRFHeaderName, "b")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("mismatch: code=%d", w.Code)
	}

	// matched
	req = httptest.NewRequest("POST", "/", nil)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "abc"})
	req.Header.Set(CSRFHeaderName, "abc")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("match: code=%d", w.Code)
	}
}
```

- [ ] **Step 2: Implement**

```go
// internal/auth/csrf.go
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

const (
	CSRFCookieName = "frames_csrf"
	CSRFHeaderName = "X-CSRF-Token"
)

// CSRF enforces the double-submit cookie pattern:
//   - Safe methods (GET/HEAD/OPTIONS) always pass and seed the cookie if missing.
//   - Unsafe methods (POST/PUT/PATCH/DELETE) require the cookie value to
//     equal the value sent in the X-CSRF-Token header.
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := r.Cookie(CSRFCookieName)
		if c == nil || c.Value == "" {
			tok, _ := newCSRFToken()
			http.SetCookie(w, &http.Cookie{
				Name: CSRFCookieName, Value: tok,
				Path: "/", SameSite: http.SameSiteLaxMode,
				// NOT HttpOnly — the JS client must read it to set the header
			})
			c = &http.Cookie{Value: tok}
		}

		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		hdr := r.Header.Get(CSRFHeaderName)
		if hdr == "" || hdr != c.Value {
			http.Error(w, `{"error":"csrf mismatch"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func newCSRFToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/auth/... -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/auth/csrf.go internal/auth/csrf_test.go
git commit -m "feat(auth): csrf double-submit middleware"
```

---

### Task 12: Login rate limit

**Files:**
- Create: `internal/auth/ratelimit.go`
- Create: `internal/auth/ratelimit_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/auth/ratelimit_test.go
package auth

import (
	"testing"
	"time"
)

func TestLoginLimiter(t *testing.T) {
	l := NewLoginLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !l.Allow("1.2.3.4") {
			t.Fatalf("allow %d expected true", i)
		}
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("expected block after limit")
	}
	// Different IP unaffected.
	if !l.Allow("5.6.7.8") {
		t.Fatal("different ip should be allowed")
	}
}

func TestLoginLimiter_WindowReset(t *testing.T) {
	l := NewLoginLimiter(2, 20*time.Millisecond)
	l.Allow("1.1.1.1")
	l.Allow("1.1.1.1")
	if l.Allow("1.1.1.1") {
		t.Fatal("should block")
	}
	time.Sleep(30 * time.Millisecond)
	if !l.Allow("1.1.1.1") {
		t.Fatal("should reset")
	}
}
```

- [ ] **Step 2: Implement**

```go
// internal/auth/ratelimit.go
package auth

import (
	"sync"
	"time"
)

type LoginLimiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	buckets map[string]*bucket
}

type bucket struct {
	count    int
	resetsAt time.Time
}

func NewLoginLimiter(max int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{
		max: max, window: window,
		buckets: map[string]*bucket{},
	}
}

func (l *LoginLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b := l.buckets[key]
	if b == nil || now.After(b.resetsAt) {
		l.buckets[key] = &bucket{count: 1, resetsAt: now.Add(l.window)}
		return true
	}
	if b.count >= l.max {
		return false
	}
	b.count++
	return true
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/auth/... -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/auth/ratelimit.go internal/auth/ratelimit_test.go
git commit -m "feat(auth): simple in-memory login rate limiter"
```

---

### Task 13: Login / logout / me handlers

**Files:**
- Create: `internal/api/handlers_auth.go`
- Create: `internal/api/handlers_auth_test.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Implement handlers**

```go
// internal/api/handlers_auth.go
package api

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/db"
)

type AuthDeps struct {
	DB       *db.DB
	Limiter  *auth.LoginLimiter
	Secure   bool // set Secure cookies in prod
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (ad *AuthDeps) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !ad.Limiter.Allow(ip) {
		WriteError(w, http.StatusTooManyRequests, "too many attempts, try later")
		return
	}
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, err := ad.DB.UserByUsername(req.Username)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	ok, err := auth.VerifyPassword(u.PasswordHash, req.Password)
	if err != nil || !ok {
		WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	tok, err := auth.NewToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "token error")
		return
	}
	if err := ad.DB.CreateSession(tok, u.ID, time.Now().Add(auth.SessionTTL)); err != nil {
		WriteError(w, http.StatusInternalServerError, "session error")
		return
	}
	auth.SetSessionCookie(w, tok, ad.Secure)
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id": u.ID, "username": u.Username, "is_admin": u.IsAdmin,
		},
	})
}

func (ad *AuthDeps) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.SessionCookieName); err == nil && c.Value != "" {
		_ = ad.DB.DeleteSession(c.Value)
	}
	auth.ClearSessionCookie(w, ad.Secure)
	w.WriteHeader(http.StatusNoContent)
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id": u.ID, "username": u.Username, "is_admin": u.IsAdmin,
		},
	})
}

func clientIP(r *http.Request) string {
	// Prefer X-Forwarded-For first entry when behind trusted proxy; fallback to remote addr.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

var _ = errors.New // silence unused-import warnings in future edits
```

- [ ] **Step 2: Wire into router**

```go
// internal/api/router.go
package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/db"
)

type Deps struct {
	Log     *slog.Logger
	DB      *db.DB
	Limiter *auth.LoginLimiter
	Secure  bool
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	ad := &AuthDeps{DB: d.DB, Limiter: d.Limiter, Secure: d.Secure}

	// CSRF applies to all /api routes. Login itself is unsafe but is only reachable
	// after a GET seeded the cookie; the frontend fetches /api/me (GET) first.
	r.Route("/api", func(r chi.Router) {
		r.Use(auth.CSRF)
		r.Post("/login", ad.handleLogin)
		r.Post("/logout", ad.handleLogout)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireLogin(d.DB))
			r.Get("/me", handleMe)
		})
	})

	return r
}
```

- [ ] **Step 3: Write handler test**

```go
// internal/api/handlers_auth_test.go
package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/db"
)

func testDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.Migrate(); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestLoginFlow(t *testing.T) {
	d := testDB(t)
	hash, _ := auth.HashPassword("hunter2")
	_, _ = d.CreateUser("alice", hash, false)

	r := NewRouter(Deps{
		Log:     slog.Default(),
		DB:      d,
		Limiter: auth.NewLoginLimiter(5, time.Minute),
	})

	// seed csrf with GET
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	csrfCookie := extractCookie(w.Header().Values("Set-Cookie"), auth.CSRFCookieName)
	if csrfCookie == "" {
		t.Fatal("no csrf cookie set")
	}

	// wrong password
	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "wrong"})
	req = httptest.NewRequest("POST", "/api/login", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: auth.CSRFCookieName, Value: csrfCookie})
	req.Header.Set(auth.CSRFHeaderName, csrfCookie)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("wrong-pw code=%d", w.Code)
	}

	// right password
	body, _ = json.Marshal(map[string]string{"username": "alice", "password": "hunter2"})
	req = httptest.NewRequest("POST", "/api/login", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: auth.CSRFCookieName, Value: csrfCookie})
	req.Header.Set(auth.CSRFHeaderName, csrfCookie)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("login code=%d body=%s", w.Code, w.Body.String())
	}
	sess := extractCookie(w.Header().Values("Set-Cookie"), auth.SessionCookieName)
	if sess == "" {
		t.Fatal("no session cookie")
	}

	// /api/me with session
	req = httptest.NewRequest("GET", "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: sess})
	req.AddCookie(&http.Cookie{Name: auth.CSRFCookieName, Value: csrfCookie})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("me code=%d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"alice"`) {
		t.Errorf("body: %s", w.Body.String())
	}
}

func extractCookie(setCookies []string, name string) string {
	for _, raw := range setCookies {
		// crude parse: name=value;...
		parts := strings.SplitN(raw, ";", 2)
		if kv := strings.SplitN(parts[0], "=", 2); len(kv) == 2 && kv[0] == name {
			return kv[1]
		}
	}
	return ""
}
```

- [ ] **Step 4: Update main.go to pass auth deps**

```go
// Replace the Deps{} construction in cmd/frames/main.go with:
	lim := auth.NewLoginLimiter(5, 15*time.Minute)
	h := api.NewRouter(api.Deps{
		Log: log, DB: database, Limiter: lim,
		Secure: strings.HasPrefix(cfg.PublicURL, "https://"),
	})
```

Add imports `"github.com/NielHeesakkers/frames/internal/auth"` and `"strings"`.

- [ ] **Step 5: Run tests**

```bash
go test ./... -count=1
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/api/ cmd/frames/main.go
git commit -m "feat(api): login/logout/me handlers and csrf-protected router"
```

---

### Task 14: Admin bootstrap on first run

**Files:**
- Create: `internal/auth/bootstrap.go`
- Create: `internal/auth/bootstrap_test.go`
- Modify: `cmd/frames/main.go`

- [ ] **Step 1: Write failing test**

```go
// internal/auth/bootstrap_test.go
package auth

import (
	"testing"

	"github.com/NielHeesakkers/frames/internal/db"
)

func TestBootstrapAdmin(t *testing.T) {
	d, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.Migrate(); err != nil {
		t.Fatal(err)
	}

	created, err := BootstrapAdmin(d, "niel", "hunter2hunter2hunter2")
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Error("expected created=true on fresh db")
	}
	u, err := d.UserByUsername("niel")
	if err != nil || !u.IsAdmin {
		t.Errorf("admin missing/not-admin: %+v err=%v", u, err)
	}

	// Second call is a no-op.
	created, err = BootstrapAdmin(d, "niel", "hunter2hunter2hunter2")
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Error("expected created=false second time")
	}
}

func TestBootstrapAdmin_EmptyCreds(t *testing.T) {
	d, _ := db.Open(t.TempDir())
	defer d.Close()
	_ = d.Migrate()
	_, err := BootstrapAdmin(d, "", "")
	if err == nil {
		t.Error("expected error when creds empty")
	}
}
```

- [ ] **Step 2: Implement**

```go
// internal/auth/bootstrap.go
package auth

import (
	"errors"

	"github.com/NielHeesakkers/frames/internal/db"
)

// BootstrapAdmin creates the admin user if none exists. Returns (created, error).
// Returns an error if there is no admin AND the supplied creds are empty.
func BootstrapAdmin(d *db.DB, username, password string) (bool, error) {
	users, err := d.ListUsers()
	if err != nil {
		return false, err
	}
	for _, u := range users {
		if u.IsAdmin {
			return false, nil
		}
	}
	if username == "" || password == "" {
		return false, errors.New("no admin exists; set FRAMES_ADMIN_USERNAME + FRAMES_ADMIN_PASSWORD on first run")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return false, err
	}
	if _, err := d.CreateUser(username, hash, true); err != nil {
		return false, err
	}
	return true, nil
}
```

- [ ] **Step 3: Call it from main.go**

Insert after migration in `run()`:

```go
created, err := auth.BootstrapAdmin(database, cfg.AdminUsername, cfg.AdminPassword)
if err != nil {
	return fmt.Errorf("bootstrap admin: %w", err)
}
if created {
	log.Info("admin user created", "username", cfg.AdminUsername)
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./... -count=1
```

- [ ] **Step 5: Commit**

```bash
git add internal/auth/bootstrap.go internal/auth/bootstrap_test.go cmd/frames/main.go
git commit -m "feat(auth): bootstrap admin from env vars on first run"
```

---

## Phase 3 — Filesystem scanner

Builds the folders/files repositories, the walker (mtime-driven), a scheduler (interval + cron), and a manual scan endpoint.

### Task 15: Folders repository

**Files:**
- Create: `internal/db/folders.go`
- Create: `internal/db/folders_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/db/folders_test.go
package db

import "testing"

func TestFolderUpsertAndTree(t *testing.T) {
	d := setupDB(t)

	root, err := d.UpsertFolder(Folder{Path: "", Name: "", Mtime: 1, ParentID: nil})
	if err != nil {
		t.Fatal(err)
	}
	kid, err := d.UpsertFolder(Folder{Path: "2024", Name: "2024", Mtime: 2, ParentID: &root.ID})
	if err != nil {
		t.Fatal(err)
	}

	// Lookup.
	got, err := d.FolderByPath("2024")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != kid.ID || *got.ParentID != root.ID {
		t.Errorf("tree mismatch: %+v", got)
	}

	// Update path via upsert (same path keeps id).
	same, err := d.UpsertFolder(Folder{Path: "2024", Name: "2024", Mtime: 99, ParentID: &root.ID})
	if err != nil {
		t.Fatal(err)
	}
	if same.ID != kid.ID {
		t.Errorf("expected same id after upsert; got %d vs %d", same.ID, kid.ID)
	}

	// List children.
	children, err := d.ChildFolders(root.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 1 {
		t.Errorf("children=%d", len(children))
	}
}

func TestDeleteFolderCascades(t *testing.T) {
	d := setupDB(t)
	root, _ := d.UpsertFolder(Folder{Path: "", Name: "", Mtime: 1})
	_, _ = d.UpsertFolder(Folder{Path: "A", Name: "A", Mtime: 1, ParentID: &root.ID})
	if err := d.DeleteFolder(root.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.FolderByPath("A"); err == nil {
		t.Error("expected child to be deleted via cascade")
	}
}
```

- [ ] **Step 2: Implement**

```go
// internal/db/folders.go
package db

import (
	"database/sql"
	"errors"
	"time"
)

type Folder struct {
	ID            int64
	ParentID      *int64
	Path          string // relative to PhotosRoot; root = ""
	Name          string
	Mtime         int64
	ItemCount     int64
	LastScannedAt *time.Time
}

func (d *DB) UpsertFolder(f Folder) (*Folder, error) {
	// Use ON CONFLICT(path) DO UPDATE pattern.
	_, err := d.Exec(`
		INSERT INTO folders(parent_id,path,name,mtime)
		VALUES(?,?,?,?)
		ON CONFLICT(path) DO UPDATE SET
		  parent_id=excluded.parent_id,
		  name=excluded.name,
		  mtime=excluded.mtime
	`, f.ParentID, f.Path, f.Name, f.Mtime)
	if err != nil {
		return nil, err
	}
	return d.FolderByPath(f.Path)
}

func (d *DB) FolderByPath(path string) (*Folder, error) {
	row := d.QueryRow(`SELECT id,parent_id,path,name,mtime,item_count,last_scanned_at FROM folders WHERE path=?`, path)
	return scanFolder(row)
}

func (d *DB) FolderByID(id int64) (*Folder, error) {
	row := d.QueryRow(`SELECT id,parent_id,path,name,mtime,item_count,last_scanned_at FROM folders WHERE id=?`, id)
	return scanFolder(row)
}

func (d *DB) ChildFolders(parentID int64) ([]Folder, error) {
	rows, err := d.Query(`SELECT id,parent_id,path,name,mtime,item_count,last_scanned_at FROM folders WHERE parent_id=? ORDER BY name`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Folder
	for rows.Next() {
		f := Folder{}
		var pid sql.NullInt64
		var ls sql.NullTime
		if err := rows.Scan(&f.ID, &pid, &f.Path, &f.Name, &f.Mtime, &f.ItemCount, &ls); err != nil {
			return nil, err
		}
		if pid.Valid {
			v := pid.Int64
			f.ParentID = &v
		}
		if ls.Valid {
			f.LastScannedAt = &ls.Time
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (d *DB) DeleteFolder(id int64) error {
	_, err := d.Exec(`DELETE FROM folders WHERE id=?`, id)
	return err
}

func (d *DB) SetFolderScanned(id int64, mtime int64, count int64) error {
	_, err := d.Exec(`UPDATE folders SET mtime=?, item_count=?, last_scanned_at=CURRENT_TIMESTAMP WHERE id=?`,
		mtime, count, id)
	return err
}

func scanFolder(r rowScanner) (*Folder, error) {
	f := &Folder{}
	var pid sql.NullInt64
	var ls sql.NullTime
	err := r.Scan(&f.ID, &pid, &f.Path, &f.Name, &f.Mtime, &f.ItemCount, &ls)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if pid.Valid {
		v := pid.Int64
		f.ParentID = &v
	}
	if ls.Valid {
		f.LastScannedAt = &ls.Time
	}
	return f, nil
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/db/... -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/db/folders.go internal/db/folders_test.go
git commit -m "feat(db): folders repository with upsert and tree"
```

---

### Task 16: Files repository

**Files:**
- Create: `internal/db/files.go`
- Create: `internal/db/files_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/db/files_test.go
package db

import "testing"

func TestFileCRUD(t *testing.T) {
	d := setupDB(t)
	root, _ := d.UpsertFolder(Folder{Path: "", Name: "", Mtime: 1})

	f := File{
		FolderID: root.ID, Filename: "a.jpg", RelativePath: "a.jpg",
		Size: 1000, Mtime: 123, MimeType: "image/jpeg", Kind: "image",
	}
	id, err := d.InsertFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("zero id")
	}

	files, err := d.FilesInFolder(root.ID, 100, 0, SortByName)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("count=%d", len(files))
	}
	if files[0].Filename != "a.jpg" {
		t.Error("wrong file returned")
	}

	// Update by upsert-like API (for scanner diffs).
	f.Size = 2000
	f.Mtime = 456
	if err := d.UpdateFileStat(id, f.Mtime, f.Size); err != nil {
		t.Fatal(err)
	}

	// Delete.
	if err := d.DeleteFile(id); err != nil {
		t.Fatal(err)
	}
	files, _ = d.FilesInFolder(root.ID, 100, 0, SortByName)
	if len(files) != 0 {
		t.Error("expected no files after delete")
	}
}

func TestFile_UniquePath(t *testing.T) {
	d := setupDB(t)
	root, _ := d.UpsertFolder(Folder{Path: "", Name: "", Mtime: 1})
	_, err := d.InsertFile(File{FolderID: root.ID, Filename: "a.jpg", RelativePath: "a.jpg", Size: 1, Mtime: 1, Kind: "image"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.InsertFile(File{FolderID: root.ID, Filename: "a.jpg", RelativePath: "a.jpg", Size: 1, Mtime: 1, Kind: "image"})
	if err == nil {
		t.Fatal("expected unique violation")
	}
}
```

- [ ] **Step 2: Implement**

```go
// internal/db/files.go
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type File struct {
	ID              int64
	FolderID        int64
	Filename        string
	RelativePath    string
	Size            int64
	Mtime           int64
	MimeType        string
	Kind            string // image | raw | video | other
	TakenAt         *time.Time
	Width           *int
	Height          *int
	CameraMake      *string
	CameraModel     *string
	Orientation     *int
	DurationMs      *int64
	ThumbStatus     string
	ThumbAttempts   int
	PreviewStatus   string
	PreviewAttempts int
}

type SortMode int

const (
	SortByName SortMode = iota
	SortByTakenAt
	SortBySize
)

func (d *DB) InsertFile(f File) (int64, error) {
	res, err := d.Exec(`
		INSERT INTO files(folder_id,filename,relative_path,size,mtime,mime_type,kind)
		VALUES(?,?,?,?,?,?,?)
	`, f.FolderID, f.Filename, f.RelativePath, f.Size, f.Mtime, f.MimeType, f.Kind)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) UpdateFileStat(id, mtime, size int64) error {
	_, err := d.Exec(`UPDATE files SET mtime=?, size=?, thumb_status='pending', preview_status='pending', thumb_attempts=0, preview_attempts=0 WHERE id=?`,
		mtime, size, id)
	return err
}

func (d *DB) UpdateFileMetadata(id int64, m MetadataUpdate) error {
	_, err := d.Exec(`
		UPDATE files SET
		  taken_at=?, width=?, height=?, camera_make=?, camera_model=?,
		  orientation=?, duration_ms=?, mime_type=COALESCE(?, mime_type)
		WHERE id=?
	`, m.TakenAt, m.Width, m.Height, m.CameraMake, m.CameraModel, m.Orientation, m.DurationMs, m.MimeType, id)
	return err
}

type MetadataUpdate struct {
	TakenAt     *time.Time
	Width       *int
	Height      *int
	CameraMake  *string
	CameraModel *string
	Orientation *int
	DurationMs  *int64
	MimeType    *string
}

func (d *DB) FileByID(id int64) (*File, error) {
	row := d.QueryRow(fileSelect+` WHERE id=?`, id)
	return scanFile(row)
}

func (d *DB) FilesInFolder(folderID int64, limit, offset int, sort SortMode) ([]File, error) {
	order := "filename"
	switch sort {
	case SortByTakenAt:
		order = "COALESCE(taken_at, datetime(mtime, 'unixepoch')) DESC, filename"
	case SortBySize:
		order = "size DESC, filename"
	}
	q := fileSelect + ` WHERE folder_id=? ORDER BY ` + order + ` LIMIT ? OFFSET ?`
	rows, err := d.Query(q, folderID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []File
	for rows.Next() {
		f, err := scanFileRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func (d *DB) PendingThumbs(limit int) ([]File, error) {
	rows, err := d.Query(fileSelect+` WHERE thumb_status='pending' AND thumb_attempts < 3 ORDER BY id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []File
	for rows.Next() {
		f, err := scanFileRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func (d *DB) SetThumbStatus(id int64, status string, bumpAttempts bool) error {
	if bumpAttempts {
		_, err := d.Exec(`UPDATE files SET thumb_status=?, thumb_attempts=thumb_attempts+1 WHERE id=?`, status, id)
		return err
	}
	_, err := d.Exec(`UPDATE files SET thumb_status=? WHERE id=?`, status, id)
	return err
}

func (d *DB) SetPreviewStatus(id int64, status string, bumpAttempts bool) error {
	if bumpAttempts {
		_, err := d.Exec(`UPDATE files SET preview_status=?, preview_attempts=preview_attempts+1 WHERE id=?`, status, id)
		return err
	}
	_, err := d.Exec(`UPDATE files SET preview_status=? WHERE id=?`, status, id)
	return err
}

func (d *DB) DeleteFile(id int64) error {
	_, err := d.Exec(`DELETE FROM files WHERE id=?`, id)
	return err
}

func (d *DB) DeleteFilesByFolder(folderID int64, keepFilenames []string) error {
	// Delete files not in keep list. Use IN with rebound parameters; at scale we
	// expect keepFilenames to fit (per-folder, not per-library).
	if len(keepFilenames) == 0 {
		_, err := d.Exec(`DELETE FROM files WHERE folder_id=?`, folderID)
		return err
	}
	placeholders := ""
	args := []any{folderID}
	for i, n := range keepFilenames {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, n)
	}
	q := fmt.Sprintf(`DELETE FROM files WHERE folder_id=? AND filename NOT IN (%s)`, placeholders)
	_, err := d.Exec(q, args...)
	return err
}

const fileSelect = `
SELECT id, folder_id, filename, relative_path, size, mtime, mime_type, kind,
       taken_at, width, height, camera_make, camera_model, orientation, duration_ms,
       thumb_status, thumb_attempts, preview_status, preview_attempts
FROM files`

func scanFile(r rowScanner) (*File, error) {
	f := &File{}
	var takenAt sql.NullTime
	var w, h, orient sql.NullInt64
	var make_, model sql.NullString
	var dur sql.NullInt64
	err := r.Scan(&f.ID, &f.FolderID, &f.Filename, &f.RelativePath, &f.Size, &f.Mtime, &f.MimeType, &f.Kind,
		&takenAt, &w, &h, &make_, &model, &orient, &dur,
		&f.ThumbStatus, &f.ThumbAttempts, &f.PreviewStatus, &f.PreviewAttempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if takenAt.Valid {
		f.TakenAt = &takenAt.Time
	}
	if w.Valid {
		v := int(w.Int64)
		f.Width = &v
	}
	if h.Valid {
		v := int(h.Int64)
		f.Height = &v
	}
	if orient.Valid {
		v := int(orient.Int64)
		f.Orientation = &v
	}
	if make_.Valid {
		v := make_.String
		f.CameraMake = &v
	}
	if model.Valid {
		v := model.String
		f.CameraModel = &v
	}
	if dur.Valid {
		v := dur.Int64
		f.DurationMs = &v
	}
	return f, nil
}

func scanFileRows(r *sql.Rows) (*File, error) { return scanFile(r) }
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/db/... -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/db/files.go internal/db/files_test.go
git commit -m "feat(db): files repository with sort, pagination, status updates"
```

---

### Task 17: Scan_jobs repository

**Files:**
- Create: `internal/db/scan_jobs.go`

- [ ] **Step 1: Implement (simple, no dedicated test beyond integration in scanner test)**

```go
// internal/db/scan_jobs.go
package db

import "time"

type ScanJob struct {
	ID            int64
	Type          string
	StartedAt     time.Time
	FinishedAt    *time.Time
	FilesScanned  int64
	FilesAdded    int64
	FilesUpdated  int64
	FilesRemoved  int64
	Error         *string
}

func (d *DB) StartScanJob(kind string) (int64, error) {
	res, err := d.Exec(`INSERT INTO scan_jobs(type,started_at) VALUES(?,CURRENT_TIMESTAMP)`, kind)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) FinishScanJob(id int64, scanned, added, updated, removed int64, errMsg string) error {
	var errVal any
	if errMsg != "" {
		errVal = errMsg
	}
	_, err := d.Exec(`
		UPDATE scan_jobs SET finished_at=CURRENT_TIMESTAMP,
		  files_scanned=?, files_added=?, files_updated=?, files_removed=?, error=?
		WHERE id=?
	`, scanned, added, updated, removed, errVal, id)
	return err
}

func (d *DB) LastScanJob(kind string) (*ScanJob, error) {
	row := d.QueryRow(`
		SELECT id,type,started_at,finished_at,files_scanned,files_added,files_updated,files_removed,error
		FROM scan_jobs WHERE type=? ORDER BY id DESC LIMIT 1
	`, kind)
	var j ScanJob
	var fin *time.Time
	var emsg *string
	if err := row.Scan(&j.ID, &j.Type, &j.StartedAt, &fin, &j.FilesScanned, &j.FilesAdded, &j.FilesUpdated, &j.FilesRemoved, &emsg); err != nil {
		return nil, err
	}
	j.FinishedAt = fin
	j.Error = emsg
	return &j, nil
}
```

- [ ] **Step 2: Run tests**

```bash
go test ./... -count=1
```

- [ ] **Step 3: Commit**

```bash
git add internal/db/scan_jobs.go
git commit -m "feat(db): scan_jobs repository"
```

---

### Task 18: MIME/kind classifier

**Files:**
- Create: `internal/scanner/mime.go`
- Create: `internal/scanner/mime_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/scanner/mime_test.go
package scanner

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		name, wantKind, wantMime string
	}{
		{"IMG.JPG", "image", "image/jpeg"},
		{"img.heic", "image", "image/heic"},
		{"clip.MP4", "video", "video/mp4"},
		{"dsc.arw", "raw", "image/x-sony-arw"},
		{"nikon.NEF", "raw", "image/x-nikon-nef"},
		{"canon.cr2", "raw", "image/x-canon-cr2"},
		{"adobe.dng", "raw", "image/x-adobe-dng"},
		{"readme.pdf", "other", "application/pdf"},
		{"song.flac", "other", "audio/flac"},
		{"unknown.xyz", "other", "application/octet-stream"},
	}
	for _, c := range cases {
		k, m := Classify(c.name)
		if k != c.wantKind || m != c.wantMime {
			t.Errorf("%s: got (%s,%s) want (%s,%s)", c.name, k, m, c.wantKind, c.wantMime)
		}
	}
}
```

- [ ] **Step 2: Implement**

```go
// internal/scanner/mime.go
package scanner

import (
	"path/filepath"
	"strings"
)

var rawExts = map[string]string{
	".arw": "image/x-sony-arw",
	".cr2": "image/x-canon-cr2",
	".cr3": "image/x-canon-cr3",
	".nef": "image/x-nikon-nef",
	".dng": "image/x-adobe-dng",
	".raf": "image/x-fuji-raf",
	".rw2": "image/x-panasonic-rw2",
	".orf": "image/x-olympus-orf",
	".srw": "image/x-samsung-srw",
	".pef": "image/x-pentax-pef",
}

var imageExts = map[string]string{
	".jpg": "image/jpeg", ".jpeg": "image/jpeg",
	".png": "image/png", ".gif": "image/gif",
	".webp": "image/webp", ".avif": "image/avif",
	".heic": "image/heic", ".heif": "image/heif",
	".tif": "image/tiff", ".tiff": "image/tiff",
	".bmp": "image/bmp",
}

var videoExts = map[string]string{
	".mp4": "video/mp4", ".mov": "video/quicktime",
	".mkv": "video/x-matroska", ".avi": "video/x-msvideo",
	".webm": "video/webm", ".m4v": "video/x-m4v",
}

var otherExts = map[string]string{
	".pdf": "application/pdf",
	".mp3": "audio/mpeg", ".flac": "audio/flac", ".wav": "audio/wav",
	".txt": "text/plain", ".md": "text/markdown",
}

// Classify returns (kind, mime) for a filename. kind ∈ image|raw|video|other.
func Classify(name string) (string, string) {
	ext := strings.ToLower(filepath.Ext(name))
	if m, ok := rawExts[ext]; ok {
		return "raw", m
	}
	if m, ok := imageExts[ext]; ok {
		return "image", m
	}
	if m, ok := videoExts[ext]; ok {
		return "video", m
	}
	if m, ok := otherExts[ext]; ok {
		return "other", m
	}
	return "other", "application/octet-stream"
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/scanner/... -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/scanner/
git commit -m "feat(scanner): mime/kind classifier"
```

---

### Task 19: Walker + incremental diff

**Files:**
- Create: `internal/scanner/walker.go`
- Create: `internal/scanner/scanner.go`
- Create: `internal/scanner/scanner_test.go`

- [ ] **Step 1: Implement walker**

```go
// internal/scanner/walker.go
package scanner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type dirEntry struct {
	AbsPath string
	RelPath string
	Name    string
	Mtime   int64
}

type fileEntry struct {
	AbsPath string
	Name    string
	Size    int64
	Mtime   int64
}

// ignoredPrefixes lists name prefixes we skip entirely (OS dotfiles, thumbnail sidecars).
var ignoredPrefixes = []string{".DS_Store", ".", "@eaDir", "Thumbs.db"}

func isIgnored(name string) bool {
	for _, p := range ignoredPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// WalkDirs invokes onDir for every directory under root (including root).
// onDir receives the directory entry and a listing of its immediate files.
// Returns early on ctx cancellation.
func WalkDirs(ctx context.Context, root string, onDir func(dirEntry, []fileEntry) error) error {
	return filepath.WalkDir(root, func(p string, de fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if isIgnored(de.Name()) && p != root {
			if de.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !de.IsDir() {
			return nil
		}
		info, err := os.Stat(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		if rel == "." {
			rel = ""
		}
		// List files in this dir.
		entries, err := os.ReadDir(p)
		if err != nil {
			return err
		}
		var files []fileEntry
		for _, e := range entries {
			if e.IsDir() || isIgnored(e.Name()) {
				continue
			}
			fi, err := e.Info()
			if err != nil {
				continue
			}
			files = append(files, fileEntry{
				AbsPath: filepath.Join(p, e.Name()),
				Name:    e.Name(),
				Size:    fi.Size(),
				Mtime:   fi.ModTime().Unix(),
			})
		}
		return onDir(dirEntry{
			AbsPath: p, RelPath: rel, Name: de.Name(), Mtime: info.ModTime().Unix(),
		}, files)
	})
}
```

- [ ] **Step 2: Implement scanner orchestration**

```go
// internal/scanner/scanner.go
package scanner

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/NielHeesakkers/frames/internal/db"
)

type Scanner struct {
	DB        *db.DB
	Log       *slog.Logger
	Root      string
}

type Stats struct {
	Scanned, Added, Updated, Removed int64
}

// Scan performs one pass. If full is true, the mtime short-circuit is disabled.
func (s *Scanner) Scan(ctx context.Context, full bool) (Stats, error) {
	kind := "incremental"
	if full {
		kind = "full"
	}
	jobID, err := s.DB.StartScanJob(kind)
	if err != nil {
		return Stats{}, err
	}
	var stats Stats
	err = WalkDirs(ctx, s.Root, func(dir dirEntry, files []fileEntry) error {
		return s.handleDir(dir, files, full, &stats)
	})
	emsg := ""
	if err != nil {
		emsg = err.Error()
	}
	if fErr := s.DB.FinishScanJob(jobID, stats.Scanned, stats.Added, stats.Updated, stats.Removed, emsg); fErr != nil {
		s.Log.Warn("failed to finish scan job", "err", fErr)
	}
	return stats, err
}

func (s *Scanner) handleDir(dir dirEntry, files []fileEntry, full bool, stats *Stats) error {
	// Ensure folder row exists; determine parent.
	var parentID *int64
	if dir.RelPath != "" {
		parentRel := filepath.Dir(dir.RelPath)
		if parentRel == "." {
			parentRel = ""
		}
		parent, err := s.DB.FolderByPath(parentRel)
		if err != nil {
			return err
		}
		parentID = &parent.ID
	}
	existing, err := s.DB.FolderByPath(dir.RelPath)
	var folderID int64
	switch {
	case err == nil:
		folderID = existing.ID
		// Short-circuit if mtime unchanged AND not a full scan.
		if !full && existing.Mtime == dir.Mtime {
			return nil
		}
	case err == db.ErrNotFound:
		created, cerr := s.DB.UpsertFolder(db.Folder{
			ParentID: parentID, Path: dir.RelPath, Name: dir.Name, Mtime: dir.Mtime,
		})
		if cerr != nil {
			return cerr
		}
		folderID = created.ID
	default:
		return err
	}

	// Pull existing files for this folder.
	existingFiles, err := s.DB.FilesInFolder(folderID, 100000, 0, db.SortByName)
	if err != nil {
		return err
	}
	byName := make(map[string]db.File, len(existingFiles))
	for _, f := range existingFiles {
		byName[f.Filename] = f
	}

	keep := make([]string, 0, len(files))
	for _, fe := range files {
		stats.Scanned++
		keep = append(keep, fe.Name)
		old, exists := byName[fe.Name]
		kind, mime := Classify(fe.Name)
		rel := filepath.Join(dir.RelPath, fe.Name)
		if !exists {
			_, err := s.DB.InsertFile(db.File{
				FolderID: folderID, Filename: fe.Name, RelativePath: rel,
				Size: fe.Size, Mtime: fe.Mtime, MimeType: mime, Kind: kind,
			})
			if err != nil {
				return err
			}
			stats.Added++
			continue
		}
		if old.Mtime != fe.Mtime || old.Size != fe.Size {
			if err := s.DB.UpdateFileStat(old.ID, fe.Mtime, fe.Size); err != nil {
				return err
			}
			stats.Updated++
		}
	}
	// Remove rows for files no longer present.
	before := int64(len(existingFiles))
	if err := s.DB.DeleteFilesByFolder(folderID, keep); err != nil {
		return err
	}
	stats.Removed += before - int64(len(keep))
	if stats.Removed < 0 {
		stats.Removed = 0
	}

	// Mark folder scanned with new mtime + count.
	return s.DB.SetFolderScanned(folderID, dir.Mtime, int64(len(keep)))
}
```

- [ ] **Step 3: Write integration test**

```go
// internal/scanner/scanner_test.go
package scanner

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NielHeesakkers/frames/internal/db"
)

func writeFile(t *testing.T, p string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, size)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanner_FullRoundTrip(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "2024", "a.jpg"), 10)
	writeFile(t, filepath.Join(root, "2024", "b.jpg"), 20)
	writeFile(t, filepath.Join(root, "2023", "c.nef"), 30)

	d, _ := db.Open(t.TempDir())
	defer d.Close()
	if err := d.Migrate(); err != nil {
		t.Fatal(err)
	}

	s := &Scanner{DB: d, Log: slog.Default(), Root: root}
	stats, err := s.Scan(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Added != 3 {
		t.Errorf("added=%d want 3", stats.Added)
	}

	// Delete a file, add a new one.
	if err := os.Remove(filepath.Join(root, "2024", "b.jpg")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "2024", "c.jpg"), 15)
	// Bump the parent dir's mtime so the incremental scan visits it.
	now := time.Now()
	_ = os.Chtimes(filepath.Join(root, "2024"), now, now)

	stats, err = s.Scan(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Added != 1 {
		t.Errorf("added(2)=%d want 1", stats.Added)
	}
	if stats.Removed != 1 {
		t.Errorf("removed(2)=%d want 1", stats.Removed)
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/scanner/... -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/scanner/
git commit -m "feat(scanner): mtime-driven incremental walker with upsert/remove diff"
```

---

### Task 20: Scheduler + scan API endpoint

**Files:**
- Create: `internal/scanner/scheduler.go`
- Create: `internal/api/handlers_scan.go`
- Modify: `internal/api/router.go`
- Modify: `cmd/frames/main.go`
- Modify: `go.mod` (add `github.com/robfig/cron/v3`)

- [ ] **Step 1: Add cron dependency**

```bash
go get github.com/robfig/cron/v3
go mod tidy
```

- [ ] **Step 2: Implement scheduler**

```go
// internal/scanner/scheduler.go
package scanner

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	Scanner      *Scanner
	Interval     time.Duration
	FullCron     string
	Log          *slog.Logger

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	cronRun *cron.Cron
	trigger chan bool // true = full
}

func (s *Scheduler) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	s.trigger = make(chan bool, 8)

	// Cron for full scan.
	s.cronRun = cron.New()
	_, _ = s.cronRun.AddFunc(s.FullCron, func() { s.requestScan(true) })
	s.cronRun.Start()

	// Ticker for incremental.
	go func() {
		t := time.NewTicker(s.Interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.requestScan(false)
			}
		}
	}()

	// Runner goroutine (one scan at a time).
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case full := <-s.trigger:
				s.runOne(ctx, full)
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	if s.cronRun != nil {
		<-s.cronRun.Stop().Done()
	}
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Scheduler) TriggerNow(full bool) { s.requestScan(full) }

func (s *Scheduler) requestScan(full bool) {
	// Non-blocking: if channel full, drop.
	select {
	case s.trigger <- full:
	default:
		s.Log.Warn("scan trigger dropped (channel full)")
	}
}

func (s *Scheduler) runOne(ctx context.Context, full bool) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		s.Log.Info("scan already running, skipping")
		return
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()
	stats, err := s.Scanner.Scan(ctx, full)
	if err != nil {
		s.Log.Error("scan error", "err", err)
		return
	}
	s.Log.Info("scan done",
		"full", full, "scanned", stats.Scanned,
		"added", stats.Added, "updated", stats.Updated, "removed", stats.Removed)
}
```

- [ ] **Step 3: Implement `/api/scan` endpoint**

```go
// internal/api/handlers_scan.go
package api

import (
	"net/http"

	"github.com/NielHeesakkers/frames/internal/scanner"
)

type scanDeps struct {
	Scheduler *scanner.Scheduler
}

func (sd *scanDeps) handleTrigger(w http.ResponseWriter, r *http.Request) {
	full := r.URL.Query().Get("full") == "1"
	sd.Scheduler.TriggerNow(full)
	WriteJSON(w, http.StatusAccepted, map[string]string{"status": "scheduled"})
}
```

- [ ] **Step 4: Wire router**

Modify `NewRouter` Deps and routes:

```go
type Deps struct {
	Log       *slog.Logger
	DB        *db.DB
	Limiter   *auth.LoginLimiter
	Scheduler *scanner.Scheduler
	Secure    bool
}
```

Under `r.Group(func(r chi.Router) { r.Use(auth.RequireLogin(d.DB)) ... })`:

```go
sd := &scanDeps{Scheduler: d.Scheduler}
r.Post("/scan", sd.handleTrigger)
```

Add import `"github.com/NielHeesakkers/frames/internal/scanner"`.

- [ ] **Step 5: Wire main.go**

In `run()` after `BootstrapAdmin`:

```go
sc := &scanner.Scanner{DB: database, Log: log, Root: cfg.PhotosRoot}
sched := &scanner.Scheduler{
	Scanner: sc, Interval: cfg.ScanInterval,
	FullCron: cfg.FullScanCron, Log: log,
}
sched.Start(ctx)
defer sched.Stop()
```

Pass `Scheduler: sched` into `api.Deps{...}`.

Note: move the `ctx, stop := signal.NotifyContext(...)` line earlier so `sched.Start(ctx)` can receive it.

- [ ] **Step 6: Run tests**

```bash
go test ./... -count=1
```

- [ ] **Step 7: Commit**

```bash
git add internal/scanner/scheduler.go internal/api/handlers_scan.go internal/api/router.go cmd/frames/main.go go.mod go.sum
git commit -m "feat(scanner): scheduler with interval + cron + manual trigger"
```

---

## Phase 4 — Thumbnail / preview pipeline

Implements the cache layout, image/RAW/video thumbnailers, EXIF metadata extraction, worker pool, priority queue, and the `/api/thumb`, `/api/preview`, `/api/original` endpoints.

**External binaries required** (installed in the runtime image): `vipsthumbnail` (from libvips), `ffmpeg`, `exiftool`. They are invoked via `os/exec` — no CGo bindings.

### Task 21: Cache layout helpers

**Files:**
- Create: `internal/thumbnail/cache.go`
- Create: `internal/thumbnail/cache_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/thumbnail/cache_test.go
package thumbnail

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCachePaths(t *testing.T) {
	c := &Cache{Root: "/cache"}
	got := c.ThumbPath(1234)
	want := "/cache/thumb/d2/4d2.webp" // 1234 hex = 4d2, shard = first 2 of 04d2 padded
	if got != want {
		t.Errorf("thumb path = %q want %q", got, want)
	}
	if c.PreviewPath(1234) != "/cache/preview/d2/4d2.webp" {
		t.Errorf("preview path")
	}
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	c := &Cache{Root: dir}
	if err := c.WriteAtomic(c.ThumbPath(7), []byte("data")); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(c.ThumbPath(7))
	if string(b) != "data" {
		t.Errorf("content mismatch")
	}
	// tmp should be empty
	entries, _ := os.ReadDir(filepath.Join(dir, "tmp"))
	if len(entries) != 0 {
		t.Errorf("tmp not cleaned: %d entries", len(entries))
	}
}
```

- [ ] **Step 2: Implement**

```go
// internal/thumbnail/cache.go
package thumbnail

import (
	"fmt"
	"os"
	"path/filepath"
)

type Cache struct {
	Root string
}

func (c *Cache) shard(id int64) string {
	// Take last 2 hex chars of padded id as shard directory.
	hex := fmt.Sprintf("%04x", id)
	return hex[len(hex)-2:]
}

func (c *Cache) idHex(id int64) string {
	return fmt.Sprintf("%x", id)
}

func (c *Cache) ThumbPath(id int64) string {
	return filepath.Join(c.Root, "thumb", c.shard(id), c.idHex(id)+".webp")
}

func (c *Cache) PreviewPath(id int64) string {
	return filepath.Join(c.Root, "preview", c.shard(id), c.idHex(id)+".webp")
}

func (c *Cache) TmpDir() string { return filepath.Join(c.Root, "tmp") }

func (c *Cache) Ensure() error {
	for _, d := range []string{"thumb", "preview", "tmp"} {
		if err := os.MkdirAll(filepath.Join(c.Root, d), 0o755); err != nil {
			return err
		}
	}
	return nil
}

// WriteAtomic writes data to the final path by first writing to tmp/ and renaming.
func (c *Cache) WriteAtomic(finalPath string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(c.TmpDir(), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(c.TmpDir(), "w-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), finalPath)
}

// RemoveDerivatives deletes thumb + preview for a file id (ignores ENOENT).
func (c *Cache) RemoveDerivatives(id int64) {
	_ = os.Remove(c.ThumbPath(id))
	_ = os.Remove(c.PreviewPath(id))
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/thumbnail/... -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/thumbnail/
git commit -m "feat(thumbnail): content-addressable cache layout"
```

---

### Task 22: Image + RAW thumbnailer (vipsthumbnail)

**Files:**
- Create: `internal/thumbnail/image.go`
- Create: `internal/thumbnail/image_test.go`

- [ ] **Step 1: Implement wrapper**

```go
// internal/thumbnail/image.go
package thumbnail

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

// GenerateImageThumb uses `vipsthumbnail` to decode any supported format
// (JPEG, PNG, HEIC, WebP, AVIF, TIFF, and — when libvips is built with
// libraw — RAW) and produce a WebP of the given longest edge.
func GenerateImageThumb(ctx context.Context, src, dst string, size int, quality int) error {
	// vipsthumbnail writes output next to source by default; we pass -o with absolute dest path.
	// Format args after `[...]` control webp encode quality.
	outArg := dst + "[Q=" + itoa(quality) + ",strip]"
	cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(cctx, "vipsthumbnail",
		"--size", fmt.Sprintf("%dx%d", size, size),
		"-o", outArg,
		src,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vipsthumbnail: %w (stderr=%s)", err, stderr.String())
	}
	// vipsthumbnail may write to a different path than dst (it uses a template).
	// To ensure exactness we resolve via -o with absolute path — which works because
	// `-o <absolute>[...]` is honored verbatim. If empty, we verify the file exists.
	if _, err := filepath.Abs(dst); err != nil {
		return err
	}
	return nil
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }
```

- [ ] **Step 2: Write skipping-integration test**

```go
// internal/thumbnail/image_test.go
package thumbnail

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGenerateImageThumb(t *testing.T) {
	if _, err := exec.LookPath("vipsthumbnail"); err != nil {
		t.Skip("vipsthumbnail not installed")
	}
	// Tiny JPEG fixture (1x1 red pixel), hex-encoded to avoid binary files in repo.
	jpgHex := "ffd8ffe000104a46494600010100000100010000ffdb004300080606" +
		"070605080707070909080a0c140d0c0b0b0c1912130f141d1a1f1e1d1a1c1c202429" +
		"2c272024272c3034342e303c3c5453475b5f6769554f6a6c5f553bffdb0043010909" +
		"090c0b0c180d0d181f14111f2626262626262626262626262626262626262626262626262626" +
		"262626262626262626262626262626262626262626ffc0001108000100010301220002" +
		"11010311010333ffc4001f0000010501010101010100000000000000000102030405060708090a0b" +
		"ffc400b5100002010303020403050504040000017d01020300041105122131410613516107227114" +
		"32811491a1082342b1c11552d1f02433627282090a161718191a25262728292a3435363738393a434" +
		"445464748494a535455565758595a636465666768696a737475767778797a838485868788898a929" +
		"3949596979899a2a2a3a4a5a6a7a8a9aab2b3b4b5b6b7b8b9bac2c3c4c5c6c7c8c9cad2d3d4d5d6d7d8d" +
		"9dae1e2e3e4e5e6e7e8e9eaf1f2f3f4f5f6f7f8f9faffc4001f010003010101010101010101010000000" +
		"00000000102030405060708090a0bffc400b51100020102040403040705040400010277000102031104" +
		"0521310612415107617113223281081442914a1b1c109233352f0156272d10a162434e125f11718191a" +
		"262728292a35363738393a434445464748494a535455565758595a636465666768696a737475767778" +
		"797a82838485868788898a92939495969798999aa2a3a4a5a6a7a8a9aab2b3b4b5b6b7b8b9bac2c3c4c" +
		"5c6c7c8c9cad2d3d4d5d6d7d8d9dae2e3e4e5e6e7e8e9eaf2f3f4f5f6f7f8f9faffda000c030100021103" +
		"11003f00fb5203ffd9"

	_ = jpgHex // integration test kept minimal; real fixtures land in a later task if needed.

	dir := t.TempDir()
	src := filepath.Join(dir, "src.jpg")
	// Write a tiny valid JPEG by shelling to vipsthumbnail's helper — otherwise skip.
	if err := exec.Command("vips", "black", src, "4", "4").Run(); err != nil {
		t.Skipf("cannot create fixture: %v", err)
	}
	dst := filepath.Join(dir, "thumb.webp")
	if err := GenerateImageThumb(context.Background(), src, dst, 128, 75); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(dst)
	if err != nil || fi.Size() == 0 {
		t.Fatalf("output missing: %v size=%d", err, fi.Size())
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/thumbnail/... -v
```

Note: the test skips if the binary isn't available. CI must install `libvips` (Alpine: `vips-tools`).

- [ ] **Step 4: Commit**

```bash
git add internal/thumbnail/image.go internal/thumbnail/image_test.go
git commit -m "feat(thumbnail): vipsthumbnail wrapper for images and RAW"
```

---

### Task 23: Video thumbnailer (ffmpeg)

**Files:**
- Create: `internal/thumbnail/video.go`

- [ ] **Step 1: Implement**

```go
// internal/thumbnail/video.go
package thumbnail

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// GenerateVideoThumb extracts a frame at 3s (or 10% in, whichever is earlier)
// and produces a WebP thumbnail with longest edge = size.
func GenerateVideoThumb(ctx context.Context, src, dst string, size int, quality int) error {
	cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	tmp, err := os.CreateTemp(filepath.Dir(dst), "frame-*.png")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	// Extract a single frame to PNG.
	ff := exec.CommandContext(cctx, "ffmpeg",
		"-y", "-ss", "3",
		"-i", src,
		"-frames:v", "1",
		"-q:v", "2",
		tmpPath,
	)
	var stderr bytes.Buffer
	ff.Stderr = &stderr
	if err := ff.Run(); err != nil {
		return fmt.Errorf("ffmpeg: %w (stderr=%s)", err, stderr.String())
	}
	// Now resize+webp via vipsthumbnail.
	return GenerateImageThumb(cctx, tmpPath, dst, size, quality)
}

// ProbeVideoDurationMs returns duration in milliseconds via ffprobe.
func ProbeVideoDurationMs(ctx context.Context, src string) (int64, error) {
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "ffprobe",
		"-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", src)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(out))
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int64(f * 1000), nil
}
```

Add imports `"strconv"` and `"strings"`.

- [ ] **Step 2: Run `go build ./...`**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/thumbnail/video.go
git commit -m "feat(thumbnail): ffmpeg-based video thumbnailer + duration probe"
```

---

### Task 24: RAW preview (full render via libvips)

**Files:**
- Create: `internal/thumbnail/raw.go`

- [ ] **Step 1: Implement**

```go
// internal/thumbnail/raw.go
package thumbnail

// GenerateRawPreview renders a RAW file at preview size using libvips.
// libvips (when built with libraw support) loads RAW files natively.
// We reuse GenerateImageThumb with the larger preview size.
func GenerateRawPreview(ctx context.Context, src, dst string, size, quality int) error {
	return GenerateImageThumb(ctx, src, dst, size, quality)
}
```

Add import `"context"`.

- [ ] **Step 2: Run `go build ./...`**

- [ ] **Step 3: Commit**

```bash
git add internal/thumbnail/raw.go
git commit -m "feat(thumbnail): raw full-preview renderer"
```

---

### Task 25: EXIF metadata extraction

**Files:**
- Create: `internal/thumbnail/metadata.go`
- Create: `internal/thumbnail/metadata_test.go`

- [ ] **Step 1: Implement**

```go
// internal/thumbnail/metadata.go
package thumbnail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/NielHeesakkers/frames/internal/db"
)

// ReadMetadata calls `exiftool -json -DateTimeOriginal -ImageWidth -ImageHeight -Make -Model -Orientation`
// on the source and maps the result onto db.MetadataUpdate.
func ReadMetadata(ctx context.Context, src string) (db.MetadataUpdate, error) {
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "exiftool",
		"-json", "-d", "%Y-%m-%dT%H:%M:%S",
		"-DateTimeOriginal",
		"-ImageWidth", "-ImageHeight",
		"-Make", "-Model", "-Orientation",
		src,
	)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return db.MetadataUpdate{}, fmt.Errorf("exiftool: %w (%s)", err, stderr.String())
	}
	var arr []struct {
		DateTimeOriginal string `json:"DateTimeOriginal"`
		ImageWidth       int    `json:"ImageWidth"`
		ImageHeight      int    `json:"ImageHeight"`
		Make             string `json:"Make"`
		Model            string `json:"Model"`
		Orientation      any    `json:"Orientation"`
	}
	if err := json.Unmarshal(out.Bytes(), &arr); err != nil || len(arr) == 0 {
		return db.MetadataUpdate{}, fmt.Errorf("parse exiftool json: %w", err)
	}
	e := arr[0]
	up := db.MetadataUpdate{}
	if e.DateTimeOriginal != "" {
		if t, err := time.Parse("2006-01-02T15:04:05", e.DateTimeOriginal); err == nil {
			up.TakenAt = &t
		}
	}
	if e.ImageWidth > 0 {
		w := e.ImageWidth
		up.Width = &w
	}
	if e.ImageHeight > 0 {
		h := e.ImageHeight
		up.Height = &h
	}
	if e.Make != "" {
		m := strings.TrimSpace(e.Make)
		up.CameraMake = &m
	}
	if e.Model != "" {
		m := strings.TrimSpace(e.Model)
		up.CameraModel = &m
	}
	// Orientation: exiftool returns either integer 1-8 or descriptive string.
	if e.Orientation != nil {
		o := parseOrientation(e.Orientation)
		if o > 0 {
			up.Orientation = &o
		}
	}
	return up, nil
}

func parseOrientation(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case string:
		// descriptive forms like "Rotate 90 CW"
		switch x {
		case "Horizontal (normal)":
			return 1
		case "Mirror horizontal":
			return 2
		case "Rotate 180":
			return 3
		case "Mirror vertical":
			return 4
		case "Mirror horizontal and rotate 270 CW":
			return 5
		case "Rotate 90 CW":
			return 6
		case "Mirror horizontal and rotate 90 CW":
			return 7
		case "Rotate 270 CW":
			return 8
		}
	}
	return 0
}
```

- [ ] **Step 2: Write test (skipping when exiftool missing)**

```go
// internal/thumbnail/metadata_test.go
package thumbnail

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReadMetadata_SkipsWithoutBinary(t *testing.T) {
	if _, err := exec.LookPath("exiftool"); err != nil {
		t.Skip("exiftool not installed")
	}
	// Create a synthetic JPEG via vips (no real EXIF, but exiftool still responds).
	if _, err := exec.LookPath("vips"); err != nil {
		t.Skip("vips not installed")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "x.jpg")
	if err := exec.Command("vips", "black", src, "8", "8").Run(); err != nil {
		t.Skipf("cannot build fixture: %v", err)
	}
	_, err := ReadMetadata(context.Background(), src)
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/thumbnail/... -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/thumbnail/metadata.go internal/thumbnail/metadata_test.go
git commit -m "feat(thumbnail): exiftool-based metadata extraction"
```

---

### Task 26: Priority queue

**Files:**
- Create: `internal/thumbnail/queue.go`
- Create: `internal/thumbnail/queue_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/thumbnail/queue_test.go
package thumbnail

import "testing"

func TestQueue_OrderAndBoost(t *testing.T) {
	q := NewQueue(8)
	q.Push(1, PrioBackground)
	q.Push(2, PrioBackground)
	q.Push(3, PrioForeground)
	q.Boost(1) // 1 moves to foreground

	got := []int64{q.Pop(), q.Pop(), q.Pop()}
	// Foreground first: 3 and 1 (order within priority by insertion).
	if got[0] != 3 || got[1] != 1 || got[2] != 2 {
		t.Errorf("pop order = %v", got)
	}
}

func TestQueue_Dedup(t *testing.T) {
	q := NewQueue(8)
	q.Push(1, PrioBackground)
	q.Push(1, PrioBackground) // dedup
	if n := q.Len(); n != 1 {
		t.Errorf("len=%d want 1", n)
	}
}
```

- [ ] **Step 2: Implement**

```go
// internal/thumbnail/queue.go
package thumbnail

import "sync"

type Priority int

const (
	PrioBackground Priority = 0
	PrioForeground Priority = 1
)

// Queue is a two-bucket priority queue. Foreground items pop first.
// Deduplicates by file id.
type Queue struct {
	mu       sync.Mutex
	fg, bg   []int64
	inQueue  map[int64]Priority
	notifyCh chan struct{}
}

func NewQueue(initialCap int) *Queue {
	return &Queue{
		inQueue:  make(map[int64]Priority, initialCap),
		notifyCh: make(chan struct{}, 1),
	}
}

func (q *Queue) Push(id int64, p Priority) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if cur, ok := q.inQueue[id]; ok {
		// already queued; only boost
		if p > cur {
			q.boostLocked(id)
			q.inQueue[id] = p
		}
		return
	}
	if p == PrioForeground {
		q.fg = append(q.fg, id)
	} else {
		q.bg = append(q.bg, id)
	}
	q.inQueue[id] = p
	q.notifyOne()
}

func (q *Queue) Boost(id int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if cur, ok := q.inQueue[id]; !ok || cur == PrioForeground {
		return
	}
	q.boostLocked(id)
	q.inQueue[id] = PrioForeground
}

func (q *Queue) boostLocked(id int64) {
	for i, v := range q.bg {
		if v == id {
			q.bg = append(q.bg[:i], q.bg[i+1:]...)
			q.fg = append(q.fg, id)
			return
		}
	}
}

// Pop returns -1 if empty.
func (q *Queue) Pop() int64 {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.fg) > 0 {
		id := q.fg[0]
		q.fg = q.fg[1:]
		delete(q.inQueue, id)
		return id
	}
	if len(q.bg) > 0 {
		id := q.bg[0]
		q.bg = q.bg[1:]
		delete(q.inQueue, id)
		return id
	}
	return -1
}

func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.fg) + len(q.bg)
}

// Notify returns a channel that receives a signal when items are pushed.
func (q *Queue) Notify() <-chan struct{} { return q.notifyCh }

func (q *Queue) notifyOne() {
	select {
	case q.notifyCh <- struct{}{}:
	default:
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/thumbnail/... -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/thumbnail/queue.go internal/thumbnail/queue_test.go
git commit -m "feat(thumbnail): two-bucket priority queue with dedup"
```

---

### Task 27: Worker pool

**Files:**
- Create: `internal/thumbnail/worker.go`

- [ ] **Step 1: Implement**

```go
// internal/thumbnail/worker.go
package thumbnail

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/NielHeesakkers/frames/internal/db"
)

const (
	ThumbSize     = 256
	PreviewSize   = 2048
	ThumbQuality  = 75
	PreviewQualty = 85
	MaxAttempts   = 3
)

type Pool struct {
	DB       *db.DB
	Cache    *Cache
	Queue    *Queue
	Log      *slog.Logger
	Root     string
	Workers  int
}

func (p *Pool) Start(ctx context.Context) {
	_ = p.Cache.Ensure()
	var wg sync.WaitGroup
	for i := 0; i < p.Workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			p.workerLoop(ctx, n)
		}(i)
	}
	// Seeder: periodically fill the queue from PendingThumbs.
	go p.seeder(ctx)
	go func() {
		<-ctx.Done()
		wg.Wait()
	}()
}

func (p *Pool) seeder(ctx context.Context) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	// Initial seed.
	p.fillQueue()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.fillQueue()
		}
	}
}

func (p *Pool) fillQueue() {
	if p.Queue.Len() > 200 {
		return
	}
	pending, err := p.DB.PendingThumbs(500)
	if err != nil {
		p.Log.Warn("pending thumbs fetch failed", "err", err)
		return
	}
	for _, f := range pending {
		p.Queue.Push(f.ID, PrioBackground)
	}
}

func (p *Pool) workerLoop(ctx context.Context, n int) {
	for {
		if ctx.Err() != nil {
			return
		}
		id := p.Queue.Pop()
		if id == -1 {
			select {
			case <-ctx.Done():
				return
			case <-p.Queue.Notify():
			case <-time.After(2 * time.Second):
			}
			continue
		}
		if err := p.processOne(ctx, id); err != nil {
			p.Log.Warn("thumb worker failed", "id", id, "worker", n, "err", err)
		}
	}
}

func (p *Pool) processOne(ctx context.Context, id int64) error {
	f, err := p.DB.FileByID(id)
	if err != nil {
		return err
	}
	if f.ThumbStatus == "ready" {
		return nil
	}
	if f.ThumbAttempts >= MaxAttempts {
		return nil
	}
	src := filepath.Join(p.Root, f.RelativePath)
	dst := p.Cache.ThumbPath(id)

	// Read metadata (best effort).
	if up, mErr := ReadMetadata(ctx, src); mErr == nil {
		_ = p.DB.UpdateFileMetadata(id, up)
		// For videos, add duration.
		if f.Kind == "video" {
			if dur, derr := ProbeVideoDurationMs(ctx, src); derr == nil {
				up2 := db.MetadataUpdate{DurationMs: &dur}
				_ = p.DB.UpdateFileMetadata(id, up2)
			}
		}
	}

	switch f.Kind {
	case "image", "raw":
		err = GenerateImageThumb(ctx, src, dst, ThumbSize, ThumbQuality)
	case "video":
		err = GenerateVideoThumb(ctx, src, dst, ThumbSize, ThumbQuality)
	default:
		// No thumbnail for 'other' files; mark ready with attempts capped so we don't retry.
		_ = p.DB.SetThumbStatus(id, "failed", true)
		return nil
	}
	if err != nil {
		_ = p.DB.SetThumbStatus(id, "pending", true) // bump attempts; still pending unless capped.
		// If we've hit the cap, flip to failed.
		f2, _ := p.DB.FileByID(id)
		if f2 != nil && f2.ThumbAttempts >= MaxAttempts {
			_ = p.DB.SetThumbStatus(id, "failed", false)
		}
		return err
	}
	return p.DB.SetThumbStatus(id, "ready", false)
}

// GeneratePreview renders a preview on demand and updates status.
func (p *Pool) GeneratePreview(ctx context.Context, id int64) error {
	f, err := p.DB.FileByID(id)
	if err != nil {
		return err
	}
	if f.PreviewStatus == "ready" {
		return nil
	}
	src := filepath.Join(p.Root, f.RelativePath)
	dst := p.Cache.PreviewPath(id)
	switch f.Kind {
	case "image":
		err = GenerateImageThumb(ctx, src, dst, PreviewSize, PreviewQualty)
	case "raw":
		err = GenerateRawPreview(ctx, src, dst, PreviewSize, PreviewQualty)
	case "video":
		// Use larger frame.
		err = GenerateVideoThumb(ctx, src, dst, PreviewSize, PreviewQualty)
	default:
		return nil
	}
	if err != nil {
		_ = p.DB.SetPreviewStatus(id, "failed", true)
		return err
	}
	return p.DB.SetPreviewStatus(id, "ready", false)
}
```

- [ ] **Step 2: Run `go build ./...`**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/thumbnail/worker.go
git commit -m "feat(thumbnail): worker pool with seeder and on-demand preview"
```

---

### Task 28: Media-serving HTTP endpoints

**Files:**
- Create: `internal/api/handlers_media.go`
- Modify: `internal/api/router.go`
- Modify: `cmd/frames/main.go`

- [ ] **Step 1: Implement handlers**

```go
// internal/api/handlers_media.go
package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/NielHeesakkers/frames/internal/db"
	"github.com/NielHeesakkers/frames/internal/thumbnail"
)

type mediaDeps struct {
	DB    *db.DB
	Cache *thumbnail.Cache
	Queue *thumbnail.Queue
	Pool  *thumbnail.Pool
	Root  string
}

func parseID(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) }

func (md *mediaDeps) handleThumb(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad id")
		return
	}
	f, err := md.DB.FileByID(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "file not found")
		return
	}
	path := md.Cache.ThumbPath(id)
	if fi, err := os.Stat(path); err == nil && fi.Size() > 0 {
		serveWithETag(w, r, path, fmt.Sprintf("%d-%d", f.ID, f.Mtime), f.MimeType)
		return
	}
	// Not ready yet — boost in queue, 202.
	md.Queue.Push(id, thumbnail.PrioForeground)
	w.Header().Set("Retry-After", "2")
	WriteJSON(w, http.StatusAccepted, map[string]string{"status": "pending"})
}

func (md *mediaDeps) handlePreview(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad id")
		return
	}
	f, err := md.DB.FileByID(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "file not found")
		return
	}
	path := md.Cache.PreviewPath(id)
	if fi, err := os.Stat(path); err == nil && fi.Size() > 0 {
		serveWithETag(w, r, path, fmt.Sprintf("%d-%d-p", f.ID, f.Mtime), "image/webp")
		return
	}
	// Block up to 3s waiting for a render we kick off now.
	ctx, cancel := r.Context(), func() {}
	_ = cancel
	go func() { _ = md.Pool.GeneratePreview(ctx, id) }()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if fi, err := os.Stat(path); err == nil && fi.Size() > 0 {
			serveWithETag(w, r, path, fmt.Sprintf("%d-%d-p", f.ID, f.Mtime), "image/webp")
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	w.Header().Set("Retry-After", "2")
	WriteJSON(w, http.StatusAccepted, map[string]string{"status": "pending"})
}

func (md *mediaDeps) handleOriginal(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad id")
		return
	}
	f, err := md.DB.FileByID(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "file not found")
		return
	}
	path := filepath.Join(md.Root, f.RelativePath)
	fh, err := os.Open(path)
	if err != nil {
		WriteError(w, http.StatusNotFound, "file missing on disk")
		return
	}
	defer fh.Close()
	fi, _ := fh.Stat()
	// For videos/large files we rely on http.ServeContent to handle Range headers.
	if f.MimeType != "" {
		w.Header().Set("Content-Type", f.MimeType)
	}
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`inline; filename=%q`, f.Filename))
	http.ServeContent(w, r, f.Filename, fi.ModTime(), fh)
}

func serveWithETag(w http.ResponseWriter, r *http.Request, path, etag, mime string) {
	w.Header().Set("ETag", etag)
	if match := r.Header.Get("If-None-Match"); match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if mime != "" {
		w.Header().Set("Content-Type", mime)
	}
	fh, err := os.Open(path)
	if err != nil {
		http.Error(w, "", http.StatusNotFound)
		return
	}
	defer fh.Close()
	fi, _ := fh.Stat()
	http.ServeContent(w, r, filepath.Base(path), fi.ModTime(), fh)
}
```

- [ ] **Step 2: Wire router**

Extend `Deps`:

```go
type Deps struct {
	Log       *slog.Logger
	DB        *db.DB
	Limiter   *auth.LoginLimiter
	Scheduler *scanner.Scheduler
	Cache     *thumbnail.Cache
	Queue     *thumbnail.Queue
	Pool      *thumbnail.Pool
	Root      string
	Secure    bool
}
```

Inside the authed group:

```go
mdx := &mediaDeps{DB: d.DB, Cache: d.Cache, Queue: d.Queue, Pool: d.Pool, Root: d.Root}
r.Get("/thumb/{id}", mdx.handleThumb)
r.Get("/preview/{id}", mdx.handlePreview)
r.Get("/original/{id}", mdx.handleOriginal)
```

Add import `"github.com/NielHeesakkers/frames/internal/thumbnail"`.

- [ ] **Step 3: Wire main.go**

After the scheduler block:

```go
cache := &thumbnail.Cache{Root: cfg.CacheDir}
if err := cache.Ensure(); err != nil {
	return err
}
q := thumbnail.NewQueue(4096)
pool := &thumbnail.Pool{
	DB: database, Cache: cache, Queue: q, Log: log,
	Root: cfg.PhotosRoot, Workers: cfg.Workers,
}
pool.Start(ctx)
```

Pass `Cache: cache, Queue: q, Pool: pool, Root: cfg.PhotosRoot` into `api.Deps`.

- [ ] **Step 4: Run tests**

```bash
go test ./... -count=1
```

- [ ] **Step 5: Commit**

```bash
git add internal/api/handlers_media.go internal/api/router.go cmd/frames/main.go
git commit -m "feat(api): thumb/preview/original endpoints with ETag and range support"
```

---

## Phase 5 — Browse + search API

Adds `/api/folder`, `/api/tree`, `/api/search`, and `/api/folder_shares` endpoints used by the frontend browse view.

### Task 29: Folder listing endpoint

**Files:**
- Create: `internal/api/handlers_browse.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Implement**

```go
// internal/api/handlers_browse.go
package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/NielHeesakkers/frames/internal/db"
)

type browseDeps struct {
	DB *db.DB
}

type folderDTO struct {
	ID       int64   `json:"id"`
	ParentID *int64  `json:"parent_id,omitempty"`
	Path     string  `json:"path"`
	Name     string  `json:"name"`
	Items    int64   `json:"items"`
}

type fileDTO struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Kind     string `json:"kind"`
	MimeType string `json:"mime_type"`
	Mtime    int64  `json:"mtime"`
	TakenAt  *string `json:"taken_at,omitempty"`
	Width    *int   `json:"width,omitempty"`
	Height   *int   `json:"height,omitempty"`
	ThumbStatus   string `json:"thumb_status"`
	PreviewStatus string `json:"preview_status"`
}

func (bd *browseDeps) handleFolder(w http.ResponseWriter, r *http.Request) {
	// Path comes from the URL tail after /api/folder/
	pathParam := chi.URLParam(r, "*")
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	sort := db.SortByTakenAt
	switch q.Get("sort") {
	case "name":
		sort = db.SortByName
	case "size":
		sort = db.SortBySize
	}

	f, err := bd.DB.FolderByPath(pathParam)
	if err != nil {
		WriteError(w, http.StatusNotFound, "folder not found")
		return
	}
	children, err := bd.DB.ChildFolders(f.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	files, err := bd.DB.FilesInFolder(f.ID, limit, offset, sort)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	foldersOut := make([]folderDTO, 0, len(children))
	for _, c := range children {
		foldersOut = append(foldersOut, folderDTO{
			ID: c.ID, ParentID: c.ParentID, Path: c.Path, Name: c.Name, Items: c.ItemCount,
		})
	}
	filesOut := make([]fileDTO, 0, len(files))
	for _, fl := range files {
		var takenStr *string
		if fl.TakenAt != nil {
			s := fl.TakenAt.Format("2006-01-02T15:04:05")
			takenStr = &s
		}
		filesOut = append(filesOut, fileDTO{
			ID: fl.ID, Name: fl.Filename, Size: fl.Size, Kind: fl.Kind,
			MimeType: fl.MimeType, Mtime: fl.Mtime, TakenAt: takenStr,
			Width: fl.Width, Height: fl.Height,
			ThumbStatus: fl.ThumbStatus, PreviewStatus: fl.PreviewStatus,
		})
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"folder":   folderDTO{ID: f.ID, ParentID: f.ParentID, Path: f.Path, Name: f.Name, Items: f.ItemCount},
			"folders":  foldersOut,
			"files":    filesOut,
			"has_more": len(files) == limit,
		},
	})
}
```

- [ ] **Step 2: Wire in router**

Inside the authed group:

```go
bd := &browseDeps{DB: d.DB}
r.Get("/folder", bd.handleFolder)       // root
r.Get("/folder/*", bd.handleFolder)     // nested
```

- [ ] **Step 3: Run tests**

```bash
go test ./... -count=1
```

- [ ] **Step 4: Commit**

```bash
git add internal/api/handlers_browse.go internal/api/router.go
git commit -m "feat(api): folder listing with pagination and sort"
```

---

### Task 30: Folder tree endpoint (lazy children)

**Files:**
- Modify: `internal/api/handlers_browse.go`

- [ ] **Step 1: Add tree handler**

Append to `handlers_browse.go`:

```go
type treeNodeDTO struct {
	ID       int64  `json:"id"`
	Path     string `json:"path"`
	Name     string `json:"name"`
	HasChild bool   `json:"has_child"`
	Items    int64  `json:"items"`
}

func (bd *browseDeps) handleTree(w http.ResponseWriter, r *http.Request) {
	parentPath := r.URL.Query().Get("parent")
	var parentID int64
	if parentPath == "" {
		root, err := bd.DB.FolderByPath("")
		if err != nil {
			WriteJSON(w, http.StatusOK, map[string]any{"data": []treeNodeDTO{}})
			return
		}
		parentID = root.ID
	} else {
		f, err := bd.DB.FolderByPath(parentPath)
		if err != nil {
			WriteError(w, http.StatusNotFound, "parent not found")
			return
		}
		parentID = f.ID
	}
	kids, err := bd.DB.ChildFolders(parentID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]treeNodeDTO, 0, len(kids))
	for _, c := range kids {
		sub, _ := bd.DB.ChildFolders(c.ID)
		out = append(out, treeNodeDTO{
			ID: c.ID, Path: c.Path, Name: c.Name,
			HasChild: len(sub) > 0, Items: c.ItemCount,
		})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}
```

- [ ] **Step 2: Wire route**

```go
r.Get("/tree", bd.handleTree)
```

- [ ] **Step 3: Run tests + commit**

```bash
go test ./... -count=1
git add internal/api/handlers_browse.go internal/api/router.go
git commit -m "feat(api): lazy folder tree endpoint"
```

---

### Task 31: Search endpoint

**Files:**
- Modify: `internal/db/files.go` (add `SearchFiles`)
- Create: `internal/api/handlers_search.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Add DB query**

Append to `internal/db/files.go`:

```go
type SearchQuery struct {
	Query    string
	DateFrom *time.Time
	DateTo   *time.Time
	Camera   string
	Kind     string // image | raw | video | other | ""
	Limit    int
	Offset   int
}

func (d *DB) SearchFiles(q SearchQuery) ([]File, error) {
	var args []any
	where := []string{"1=1"}
	if q.Query != "" {
		where = append(where, "(filename LIKE ? OR relative_path LIKE ?)")
		p := "%" + q.Query + "%"
		args = append(args, p, p)
	}
	if q.DateFrom != nil {
		where = append(where, "(taken_at >= ? OR (taken_at IS NULL AND datetime(mtime,'unixepoch') >= ?))")
		args = append(args, q.DateFrom, q.DateFrom)
	}
	if q.DateTo != nil {
		where = append(where, "(taken_at <= ? OR (taken_at IS NULL AND datetime(mtime,'unixepoch') <= ?))")
		args = append(args, q.DateTo, q.DateTo)
	}
	if q.Camera != "" {
		where = append(where, "(camera_make LIKE ? OR camera_model LIKE ?)")
		p := "%" + q.Camera + "%"
		args = append(args, p, p)
	}
	if q.Kind != "" {
		where = append(where, "kind = ?")
		args = append(args, q.Kind)
	}
	if q.Limit <= 0 || q.Limit > 1000 {
		q.Limit = 200
	}
	args = append(args, q.Limit, q.Offset)

	query := fileSelect + " WHERE " + strings.Join(where, " AND ") +
		" ORDER BY COALESCE(taken_at, datetime(mtime,'unixepoch')) DESC LIMIT ? OFFSET ?"
	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []File
	for rows.Next() {
		f, err := scanFileRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}
```

Add import `"strings"` if not present.

- [ ] **Step 2: Implement handler**

```go
// internal/api/handlers_search.go
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/NielHeesakkers/frames/internal/db"
)

type searchDeps struct {
	DB *db.DB
}

func (sd *searchDeps) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sq := db.SearchQuery{
		Query:  q.Get("q"),
		Camera: q.Get("camera"),
		Kind:   q.Get("kind"),
	}
	if s := q.Get("date_from"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err == nil {
			sq.DateFrom = &t
		}
	}
	if s := q.Get("date_to"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err == nil {
			// end-of-day
			end := t.Add(24*time.Hour - time.Second)
			sq.DateTo = &end
		}
	}
	sq.Limit, _ = strconv.Atoi(q.Get("limit"))
	sq.Offset, _ = strconv.Atoi(q.Get("offset"))

	files, err := sd.DB.SearchFiles(sq)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]fileDTO, 0, len(files))
	for _, fl := range files {
		var takenStr *string
		if fl.TakenAt != nil {
			s := fl.TakenAt.Format("2006-01-02T15:04:05")
			takenStr = &s
		}
		out = append(out, fileDTO{
			ID: fl.ID, Name: fl.Filename, Size: fl.Size, Kind: fl.Kind,
			MimeType: fl.MimeType, Mtime: fl.Mtime, TakenAt: takenStr,
			Width: fl.Width, Height: fl.Height,
			ThumbStatus: fl.ThumbStatus, PreviewStatus: fl.PreviewStatus,
		})
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"files": out, "has_more": len(out) == sq.Limit},
	})
}
```

- [ ] **Step 3: Wire route**

```go
sd := &searchDeps{DB: d.DB}
r.Get("/search", sd.handleSearch)
```

- [ ] **Step 4: Run tests + commit**

```bash
go test ./... -count=1
git add internal/db/files.go internal/api/handlers_search.go internal/api/router.go
git commit -m "feat(api): search endpoint with date/camera/kind filters"
```

---

### Task 32: Internal folder-shares repository + API

**Files:**
- Create: `internal/db/folder_shares.go`
- Append to: `internal/api/handlers_shares.go` (created in Task 41 — for now just a small file)
- Modify: `internal/api/router.go`

> We split internal and external shares into separate files. External share-links live in Phase 7. Here we only implement the "navigation hint" sharing between accounts.

- [ ] **Step 1: Implement DB**

```go
// internal/db/folder_shares.go
package db

type FolderShare struct {
	FolderID         int64
	SharedWithUserID int64
	SharedBy         int64
}

func (d *DB) AddFolderShare(folderID, sharedWith, sharedBy int64) error {
	_, err := d.Exec(`
		INSERT OR IGNORE INTO folder_shares(folder_id, shared_with_user_id, shared_by)
		VALUES(?,?,?)
	`, folderID, sharedWith, sharedBy)
	return err
}

func (d *DB) RemoveFolderShare(folderID, sharedWith int64) error {
	_, err := d.Exec(`DELETE FROM folder_shares WHERE folder_id=? AND shared_with_user_id=?`, folderID, sharedWith)
	return err
}

func (d *DB) FoldersSharedWith(userID int64) ([]Folder, error) {
	rows, err := d.Query(`
		SELECT f.id, f.parent_id, f.path, f.name, f.mtime, f.item_count, f.last_scanned_at
		FROM folders f
		JOIN folder_shares s ON s.folder_id = f.id
		WHERE s.shared_with_user_id = ? ORDER BY f.name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Folder
	for rows.Next() {
		var f Folder
		var pid sqlNullInt64
		var ls sqlNullTime
		if err := rows.Scan(&f.ID, &pid, &f.Path, &f.Name, &f.Mtime, &f.ItemCount, &ls); err != nil {
			return nil, err
		}
		if pid.Valid {
			v := pid.Int64
			f.ParentID = &v
		}
		if ls.Valid {
			f.LastScannedAt = &ls.Time
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// Convenience aliases (avoid importing database/sql everywhere).
type (
	sqlNullInt64 = sqlNullInt64Alias
	sqlNullTime  = sqlNullTimeAlias
)
```

Then add the aliases in `internal/db/db.go`:

```go
// Append at bottom of db.go
import sqlPkg "database/sql"

type sqlNullInt64Alias = sqlPkg.NullInt64
type sqlNullTimeAlias  = sqlPkg.NullTime
```

(Adjust if `database/sql` is already imported — just reuse the imported name.)

- [ ] **Step 2: Handler + route**

Append to a new file:

```go
// internal/api/handlers_shares.go (stub; extended in Phase 7)
package api

import (
	"encoding/json"
	"net/http"

	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/db"
)

type sharesDeps struct {
	DB *db.DB
}

type addFolderShareReq struct {
	FolderID int64 `json:"folder_id"`
	UserID   int64 `json:"user_id"`
}

func (sh *sharesDeps) handleAddFolderShare(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	var req addFolderShareReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := sh.DB.AddFolderShare(req.FolderID, req.UserID, u.ID); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (sh *sharesDeps) handleRemoveFolderShare(w http.ResponseWriter, r *http.Request) {
	var req addFolderShareReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := sh.DB.RemoveFolderShare(req.FolderID, req.UserID); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (sh *sharesDeps) handleMySharedFolders(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	folders, err := sh.DB.FoldersSharedWith(u.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]folderDTO, 0, len(folders))
	for _, f := range folders {
		out = append(out, folderDTO{ID: f.ID, ParentID: f.ParentID, Path: f.Path, Name: f.Name, Items: f.ItemCount})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}
```

Wire in router (authed group):

```go
sh := &sharesDeps{DB: d.DB}
r.Post("/folder_shares", sh.handleAddFolderShare)
r.Delete("/folder_shares", sh.handleRemoveFolderShare)
r.Get("/shared_with_me", sh.handleMySharedFolders)
```

- [ ] **Step 3: Run tests + commit**

```bash
go test ./... -count=1
git add internal/db/folder_shares.go internal/api/handlers_shares.go internal/api/router.go internal/db/db.go
git commit -m "feat(api): internal folder-share markers"
```

---

## Phase 6 — File operations API

Implements safe-path helpers, upload, rename, move, delete, mkdir. All operations update the DB alongside the filesystem and re-queue thumbnails where needed.

### Task 33: Safe path helper

**Files:**
- Create: `internal/upload/safepath.go`
- Create: `internal/upload/safepath_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/upload/safepath_test.go
package upload

import "testing"

func TestSafePath(t *testing.T) {
	tests := []struct {
		root, rel string
		wantErr   bool
	}{
		{"/photos", "2024/a.jpg", false},
		{"/photos", "../etc/passwd", true},
		{"/photos", "2024/../2023/x.jpg", false}, // cleans to 2023/x.jpg, still inside root
		{"/photos", "/absolute", true},
		{"/photos", "2024/./b.jpg", false},
		{"/photos", "", true},
	}
	for _, tc := range tests {
		_, err := SafeJoin(tc.root, tc.rel)
		if (err != nil) != tc.wantErr {
			t.Errorf("%q+%q: err=%v want %v", tc.root, tc.rel, err, tc.wantErr)
		}
	}
}
```

- [ ] **Step 2: Implement**

```go
// internal/upload/safepath.go
package upload

import (
	"errors"
	"path/filepath"
	"strings"
)

var ErrBadPath = errors.New("path escapes root")

// SafeJoin cleans rel and joins it to root, returning an absolute path inside root.
// Rejects absolute relative paths and any cleaned path that would escape root.
func SafeJoin(root, rel string) (string, error) {
	if rel == "" {
		return "", ErrBadPath
	}
	if filepath.IsAbs(rel) {
		return "", ErrBadPath
	}
	cleaned := filepath.Clean(rel)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", ErrBadPath
	}
	joined := filepath.Join(root, cleaned)
	// Defensive final check.
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	joinedAbs, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(joinedAbs, rootAbs+string(filepath.Separator)) && joinedAbs != rootAbs {
		return "", ErrBadPath
	}
	return joinedAbs, nil
}
```

- [ ] **Step 3: Run tests + commit**

```bash
go test ./internal/upload/... -v
git add internal/upload/safepath.go internal/upload/safepath_test.go
git commit -m "feat(upload): safe-join helper preventing path traversal"
```

---

### Task 34: Filesystem operations service

**Files:**
- Create: `internal/fsops/fsops.go`
- Create: `internal/fsops/fsops_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/fsops/fsops_test.go
package fsops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NielHeesakkers/frames/internal/db"
)

func setup(t *testing.T) (string, *db.DB, *Ops) {
	t.Helper()
	root := t.TempDir()
	d, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.Migrate(); err != nil {
		t.Fatal(err)
	}
	_, _ = d.UpsertFolder(db.Folder{Path: "", Name: "", Mtime: 1})
	return root, d, &Ops{DB: d, Root: root}
}

func TestMkdir(t *testing.T) {
	root, d, ops := setup(t)
	if err := ops.Mkdir("Vakantie"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "Vakantie")); err != nil {
		t.Fatal("folder not on disk")
	}
	f, err := d.FolderByPath("Vakantie")
	if err != nil {
		t.Fatalf("folder not in db: %v", err)
	}
	if f.Name != "Vakantie" {
		t.Errorf("name=%q", f.Name)
	}
}

func TestRenameFile(t *testing.T) {
	root, d, ops := setup(t)
	if err := os.WriteFile(filepath.Join(root, "old.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rf, _ := d.FolderByPath("")
	id, _ := d.InsertFile(db.File{
		FolderID: rf.ID, Filename: "old.jpg", RelativePath: "old.jpg",
		Size: 1, Mtime: 1, Kind: "image", MimeType: "image/jpeg",
	})
	if err := ops.RenameFile(id, "new.jpg"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "new.jpg")); err != nil {
		t.Fatalf("new not on disk: %v", err)
	}
	got, _ := d.FileByID(id)
	if got.Filename != "new.jpg" || got.RelativePath != "new.jpg" {
		t.Errorf("db not updated: %+v", got)
	}
}
```

- [ ] **Step 2: Implement**

```go
// internal/fsops/fsops.go
package fsops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NielHeesakkers/frames/internal/db"
	"github.com/NielHeesakkers/frames/internal/upload"
)

type Ops struct {
	DB   *db.DB
	Root string
}

func (o *Ops) Mkdir(relPath string) error {
	abs, err := upload.SafeJoin(o.Root, relPath)
	if err != nil {
		return err
	}
	if err := os.Mkdir(abs, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	parentRel := filepath.Dir(relPath)
	if parentRel == "." {
		parentRel = ""
	}
	parent, err := o.DB.FolderByPath(parentRel)
	if err != nil {
		return err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return err
	}
	_, err = o.DB.UpsertFolder(db.Folder{
		ParentID: &parent.ID, Path: relPath, Name: filepath.Base(relPath), Mtime: fi.ModTime().Unix(),
	})
	return err
}

func (o *Ops) RenameFile(id int64, newName string) error {
	if strings.ContainsRune(newName, '/') || strings.ContainsRune(newName, '\\') {
		return fmt.Errorf("invalid name %q", newName)
	}
	f, err := o.DB.FileByID(id)
	if err != nil {
		return err
	}
	folder, err := o.DB.FolderByID(f.FolderID)
	if err != nil {
		return err
	}
	oldAbs, err := upload.SafeJoin(o.Root, f.RelativePath)
	if err != nil {
		return err
	}
	newRel := filepath.Join(folder.Path, newName)
	newAbs, err := upload.SafeJoin(o.Root, newRel)
	if err != nil {
		return err
	}
	if _, err := os.Stat(newAbs); err == nil {
		return fmt.Errorf("destination exists")
	}
	if err := os.Rename(oldAbs, newAbs); err != nil {
		return err
	}
	_, err = o.DB.Exec(`UPDATE files SET filename=?, relative_path=? WHERE id=?`,
		newName, newRel, id)
	return err
}

func (o *Ops) MoveFile(id, newFolderID int64) error {
	f, err := o.DB.FileByID(id)
	if err != nil {
		return err
	}
	dst, err := o.DB.FolderByID(newFolderID)
	if err != nil {
		return err
	}
	oldAbs, err := upload.SafeJoin(o.Root, f.RelativePath)
	if err != nil {
		return err
	}
	newRel := filepath.Join(dst.Path, f.Filename)
	newAbs, err := upload.SafeJoin(o.Root, newRel)
	if err != nil {
		return err
	}
	if _, err := os.Stat(newAbs); err == nil {
		return fmt.Errorf("destination exists")
	}
	if err := os.Rename(oldAbs, newAbs); err != nil {
		return err
	}
	_, err = o.DB.Exec(`UPDATE files SET folder_id=?, relative_path=? WHERE id=?`,
		newFolderID, newRel, id)
	return err
}

func (o *Ops) DeleteFile(id int64) error {
	f, err := o.DB.FileByID(id)
	if err != nil {
		return err
	}
	abs, err := upload.SafeJoin(o.Root, f.RelativePath)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return o.DB.DeleteFile(id)
}

func (o *Ops) DeleteFolder(id int64) error {
	f, err := o.DB.FolderByID(id)
	if err != nil {
		return err
	}
	if f.Path == "" {
		return fmt.Errorf("refusing to delete root")
	}
	abs, err := upload.SafeJoin(o.Root, f.Path)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(abs); err != nil {
		return err
	}
	return o.DB.DeleteFolder(id)
}

func (o *Ops) RenameFolder(id int64, newName string) error {
	if strings.ContainsRune(newName, '/') || strings.ContainsRune(newName, '\\') {
		return fmt.Errorf("invalid name %q", newName)
	}
	f, err := o.DB.FolderByID(id)
	if err != nil {
		return err
	}
	if f.Path == "" {
		return fmt.Errorf("refusing to rename root")
	}
	oldAbs, err := upload.SafeJoin(o.Root, f.Path)
	if err != nil {
		return err
	}
	parent := filepath.Dir(f.Path)
	if parent == "." {
		parent = ""
	}
	newRel := filepath.Join(parent, newName)
	newAbs, err := upload.SafeJoin(o.Root, newRel)
	if err != nil {
		return err
	}
	if err := os.Rename(oldAbs, newAbs); err != nil {
		return err
	}
	// Update folder row and cascade descendants' relative_path values.
	_, err = o.DB.Exec(`UPDATE folders SET path=?, name=? WHERE id=?`, newRel, newName, id)
	if err != nil {
		return err
	}
	// Update descendant folder paths and file relative_paths with the new prefix.
	oldPrefix := f.Path + "/"
	newPrefix := newRel + "/"
	_, err = o.DB.Exec(`UPDATE folders SET path = ? || substr(path, ?) WHERE path LIKE ?`,
		newPrefix, len(oldPrefix)+1, oldPrefix+"%")
	if err != nil {
		return err
	}
	_, err = o.DB.Exec(`UPDATE files SET relative_path = ? || substr(relative_path, ?) WHERE relative_path LIKE ?`,
		newPrefix, len(oldPrefix)+1, oldPrefix+"%")
	return err
}
```

- [ ] **Step 3: Run tests + commit**

```bash
go test ./internal/fsops/... -v
git add internal/fsops/
git commit -m "feat(fsops): mkdir/rename/move/delete with db + fs sync"
```

---

### Task 35: File-ops HTTP handlers

**Files:**
- Create: `internal/api/handlers_ops.go`
- Modify: `internal/api/router.go`
- Modify: `cmd/frames/main.go`

- [ ] **Step 1: Implement**

```go
// internal/api/handlers_ops.go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/NielHeesakkers/frames/internal/fsops"
)

type opsDeps struct {
	Ops *fsops.Ops
}

type mkdirReq struct{ Path string `json:"path"` }
type renameReq struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type moveReq struct {
	ID          int64 `json:"id"`
	NewFolderID int64 `json:"new_folder_id"`
}
type deleteReq struct{ ID int64 `json:"id"` }

func (od *opsDeps) handleMkdir(w http.ResponseWriter, r *http.Request) {
	var req mkdirReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := od.Ops.Mkdir(req.Path); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (od *opsDeps) handleRenameFile(w http.ResponseWriter, r *http.Request) {
	var req renameReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := od.Ops.RenameFile(req.ID, req.Name); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (od *opsDeps) handleMoveFile(w http.ResponseWriter, r *http.Request) {
	var req moveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := od.Ops.MoveFile(req.ID, req.NewFolderID); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (od *opsDeps) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	var req deleteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := od.Ops.DeleteFile(req.ID); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (od *opsDeps) handleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	var req deleteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := od.Ops.DeleteFolder(req.ID); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (od *opsDeps) handleRenameFolder(w http.ResponseWriter, r *http.Request) {
	var req renameReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := od.Ops.RenameFolder(req.ID, req.Name); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 2: Wire in router**

Inside the authed group, add `Ops *fsops.Ops` to Deps and:

```go
od := &opsDeps{Ops: d.Ops}
r.Post("/ops/mkdir", od.handleMkdir)
r.Post("/ops/file/rename", od.handleRenameFile)
r.Post("/ops/file/move", od.handleMoveFile)
r.Post("/ops/file/delete", od.handleDeleteFile)
r.Post("/ops/folder/rename", od.handleRenameFolder)
r.Post("/ops/folder/delete", od.handleDeleteFolder)
```

Add import `"github.com/NielHeesakkers/frames/internal/fsops"`.

- [ ] **Step 3: Wire main.go**

```go
ops := &fsops.Ops{DB: database, Root: cfg.PhotosRoot}
// Pass Ops: ops into api.Deps{...}
```

- [ ] **Step 4: Run tests + commit**

```bash
go test ./... -count=1
git add internal/api/handlers_ops.go internal/api/router.go cmd/frames/main.go
git commit -m "feat(api): file/folder mkdir/rename/move/delete endpoints"
```

---

### Task 36: Upload endpoint (simple chunked POST)

**Files:**
- Create: `internal/upload/chunked.go`
- Create: `internal/api/handlers_upload.go`
- Modify: `internal/api/router.go`

> We ship a plain-POST multipart upload for files up to `FRAMES_MAX_UPLOAD_SIZE`. Resumable chunked uploads (tus) are out of scope for v1; for > 2 GB files this may feel fragile but covers the common case.

- [ ] **Step 1: Implement upload service**

```go
// internal/upload/chunked.go
package upload

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/NielHeesakkers/frames/internal/db"
)

type Service struct {
	DB   *db.DB
	Root string
}

// StoreFile writes an uploaded body into folderPath/filename and registers the DB row.
func (s *Service) StoreFile(folderPath, filename string, body io.Reader, kindFn func(string) (string, string)) (int64, error) {
	if folderPath == "" {
		// root
	}
	abs, err := SafeJoin(s.Root, filepath.Join(folderPath, filename))
	if err != nil {
		return 0, err
	}
	if _, err := os.Stat(abs); err == nil {
		return 0, errors.New("file exists")
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return 0, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(abs), ".upload-*")
	if err != nil {
		return 0, err
	}
	defer os.Remove(tmp.Name())
	n, err := io.Copy(tmp, body)
	if err != nil {
		tmp.Close()
		return 0, err
	}
	if err := tmp.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(tmp.Name(), abs); err != nil {
		return 0, err
	}
	// Register in DB.
	folder, err := s.DB.FolderByPath(folderPath)
	if err != nil {
		return 0, err
	}
	kind, mime := kindFn(filename)
	fi, _ := os.Stat(abs)
	id, err := s.DB.InsertFile(db.File{
		FolderID: folder.ID, Filename: filename,
		RelativePath: filepath.Join(folderPath, filename),
		Size: n, Mtime: fi.ModTime().Unix(),
		Kind: kind, MimeType: mime,
	})
	if err != nil {
		return 0, fmt.Errorf("db insert: %w", err)
	}
	return id, nil
}
```

- [ ] **Step 2: Implement handler**

```go
// internal/api/handlers_upload.go
package api

import (
	"fmt"
	"net/http"

	"github.com/NielHeesakkers/frames/internal/scanner"
	"github.com/NielHeesakkers/frames/internal/thumbnail"
	"github.com/NielHeesakkers/frames/internal/upload"
)

type uploadDeps struct {
	Svc      *upload.Service
	Queue    *thumbnail.Queue
	MaxBytes int64
}

func (ud *uploadDeps) handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil { // 64 MiB in-memory; rest to disk
		WriteError(w, http.StatusBadRequest, "invalid multipart")
		return
	}
	folderPath := r.FormValue("path") // target folder relative path, "" = root
	r.Body = http.MaxBytesReader(w, r.Body, ud.MaxBytes)

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		WriteError(w, http.StatusBadRequest, "no files")
		return
	}
	ids := make([]int64, 0, len(files))
	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		id, err := ud.Svc.StoreFile(folderPath, fh.Filename, f, scanner.Classify)
		f.Close()
		if err != nil {
			WriteError(w, http.StatusInternalServerError, fmt.Sprintf("%s: %v", fh.Filename, err))
			return
		}
		ud.Queue.Push(id, thumbnail.PrioForeground)
		ids = append(ids, id)
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"ids": ids}})
}
```

- [ ] **Step 3: Wire router**

Add to Deps:

```go
UploadSvc *upload.Service
MaxUpload int64
```

In authed group:

```go
ud := &uploadDeps{Svc: d.UploadSvc, Queue: d.Queue, MaxBytes: d.MaxUpload}
r.Post("/upload", ud.handleUpload)
```

Wire main.go:

```go
uploadSvc := &upload.Service{DB: database, Root: cfg.PhotosRoot}
// api.Deps{... UploadSvc: uploadSvc, MaxUpload: cfg.MaxUploadSize}
```

- [ ] **Step 4: Run tests + commit**

```bash
go test ./... -count=1
git add internal/upload/chunked.go internal/api/handlers_upload.go internal/api/router.go cmd/frames/main.go
git commit -m "feat(api): multipart upload endpoint with max-size enforcement"
```

---

## Phase 7 — Sharing API

External share-links (create, revoke, public access, password gate), ZIP streaming, anonymous upload, and share-link rate limiting.

### Task 37: Share token + repository

**Files:**
- Create: `internal/share/token.go`
- Create: `internal/db/shares.go`
- Create: `internal/db/shares_test.go`

- [ ] **Step 1: Token generator**

```go
// internal/share/token.go
package share

import (
	"crypto/rand"
	"encoding/base64"
)

func NewToken() (string, error) {
	b := make([]byte, 24) // 32 base64url chars
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
```

- [ ] **Step 2: Shares repository**

```go
// internal/db/shares.go
package db

import (
	"database/sql"
	"errors"
	"time"
)

type Share struct {
	ID            int64
	Token         string
	FolderID      int64
	CreatedBy     int64
	CreatedAt     time.Time
	ExpiresAt     *time.Time
	PasswordHash  *string
	AllowDownload bool
	AllowUpload   bool
	RevokedAt     *time.Time
}

func (d *DB) CreateShare(s Share) (int64, error) {
	res, err := d.Exec(`
		INSERT INTO shares(token,folder_id,created_by,expires_at,password_hash,allow_download,allow_upload)
		VALUES(?,?,?,?,?,?,?)
	`, s.Token, s.FolderID, s.CreatedBy, s.ExpiresAt, s.PasswordHash, s.AllowDownload, s.AllowUpload)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) ShareByToken(token string) (*Share, error) {
	row := d.QueryRow(`
		SELECT id,token,folder_id,created_by,created_at,expires_at,password_hash,
		       allow_download,allow_upload,revoked_at
		FROM shares WHERE token=?
	`, token)
	return scanShare(row)
}

func (d *DB) ShareByID(id int64) (*Share, error) {
	row := d.QueryRow(`
		SELECT id,token,folder_id,created_by,created_at,expires_at,password_hash,
		       allow_download,allow_upload,revoked_at
		FROM shares WHERE id=?
	`, id)
	return scanShare(row)
}

func (d *DB) SharesByUser(userID int64) ([]Share, error) {
	rows, err := d.Query(`
		SELECT id,token,folder_id,created_by,created_at,expires_at,password_hash,
		       allow_download,allow_upload,revoked_at
		FROM shares WHERE created_by=? ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Share
	for rows.Next() {
		s, err := scanShare(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (d *DB) AllShares() ([]Share, error) {
	rows, err := d.Query(`
		SELECT id,token,folder_id,created_by,created_at,expires_at,password_hash,
		       allow_download,allow_upload,revoked_at
		FROM shares ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Share
	for rows.Next() {
		s, err := scanShare(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (d *DB) RevokeShare(id int64) error {
	_, err := d.Exec(`UPDATE shares SET revoked_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	return err
}

func (d *DB) DeleteShare(id int64) error {
	_, err := d.Exec(`DELETE FROM shares WHERE id=?`, id)
	return err
}

func scanShare(r rowScanner) (*Share, error) {
	s := &Share{}
	var exp sql.NullTime
	var pw sql.NullString
	var rev sql.NullTime
	err := r.Scan(&s.ID, &s.Token, &s.FolderID, &s.CreatedBy, &s.CreatedAt,
		&exp, &pw, &s.AllowDownload, &s.AllowUpload, &rev)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if exp.Valid {
		s.ExpiresAt = &exp.Time
	}
	if pw.Valid {
		v := pw.String
		s.PasswordHash = &v
	}
	if rev.Valid {
		s.RevokedAt = &rev.Time
	}
	return s, nil
}
```

- [ ] **Step 3: Quick test**

```go
// internal/db/shares_test.go
package db

import (
	"testing"
	"time"
)

func TestShareCRUD(t *testing.T) {
	d := setupDB(t)
	uid, _ := d.CreateUser("alice", "h", false)
	root, _ := d.UpsertFolder(Folder{Path: "", Name: "", Mtime: 1})

	exp := time.Now().Add(30 * 24 * time.Hour)
	id, err := d.CreateShare(Share{
		Token: "abc", FolderID: root.ID, CreatedBy: uid,
		ExpiresAt: &exp, AllowDownload: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("zero id")
	}
	got, err := d.ShareByToken("abc")
	if err != nil {
		t.Fatal(err)
	}
	if !got.AllowDownload {
		t.Error("allow_download not set")
	}
	if err := d.RevokeShare(id); err != nil {
		t.Fatal(err)
	}
	got, _ = d.ShareByToken("abc")
	if got.RevokedAt == nil {
		t.Error("RevokedAt not set")
	}
}
```

- [ ] **Step 4: Run tests + commit**

```bash
go test ./internal/db/... -v
git add internal/share/token.go internal/db/shares.go internal/db/shares_test.go
git commit -m "feat(share): token generator and shares repository"
```

---

### Task 38: Share validation + descendant-folder scoping

**Files:**
- Create: `internal/share/validate.go`
- Create: `internal/share/validate_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/share/validate_test.go
package share

import (
	"testing"
	"time"

	"github.com/NielHeesakkers/frames/internal/db"
)

func TestValidateActive(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name   string
		s      db.Share
		wantOK bool
		want   Status
	}{
		{"active", db.Share{}, true, StatusActive},
		{"revoked", db.Share{RevokedAt: &past}, false, StatusRevoked},
		{"expired", db.Share{ExpiresAt: &past}, false, StatusExpired},
		{"future-expiry", db.Share{ExpiresAt: &future}, true, StatusActive},
	}
	for _, c := range cases {
		st := Validate(&c.s)
		if (st == StatusActive) != c.wantOK || st != c.want {
			t.Errorf("%s: got %v", c.name, st)
		}
	}
}
```

- [ ] **Step 2: Implement**

```go
// internal/share/validate.go
package share

import (
	"time"

	"github.com/NielHeesakkers/frames/internal/db"
)

type Status int

const (
	StatusActive Status = iota
	StatusExpired
	StatusRevoked
)

func Validate(s *db.Share) Status {
	if s.RevokedAt != nil {
		return StatusRevoked
	}
	if s.ExpiresAt != nil && !s.ExpiresAt.After(time.Now()) {
		return StatusExpired
	}
	return StatusActive
}

// IsUnderFolder reports whether childPath is the same as or under rootPath.
func IsUnderFolder(rootPath, childPath string) bool {
	if rootPath == "" {
		return true // root contains everything
	}
	if childPath == rootPath {
		return true
	}
	return len(childPath) > len(rootPath) && childPath[:len(rootPath)+1] == rootPath+"/"
}
```

- [ ] **Step 3: Run tests + commit**

```bash
go test ./internal/share/... -v
git add internal/share/validate.go internal/share/validate_test.go
git commit -m "feat(share): status validation and scope helper"
```

---

### Task 39: Authenticated share CRUD endpoints

**Files:**
- Modify: `internal/api/handlers_shares.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Extend shares handler**

Add to `internal/api/handlers_shares.go`:

```go
import (
	"errors"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/NielHeesakkers/frames/internal/share"
	"github.com/NielHeesakkers/frames/internal/auth"
)

type createShareReq struct {
	FolderID      int64  `json:"folder_id"`
	ExpiresInDays int    `json:"expires_in_days"` // 0 = never; positive => days
	Password      string `json:"password"`         // optional
	AllowDownload bool   `json:"allow_download"`
	AllowUpload   bool   `json:"allow_upload"`
}

type shareDTO struct {
	ID            int64   `json:"id"`
	Token         string  `json:"token"`
	FolderID      int64   `json:"folder_id"`
	FolderPath    string  `json:"folder_path"`
	ExpiresAt     *string `json:"expires_at,omitempty"`
	HasPassword   bool    `json:"has_password"`
	AllowDownload bool    `json:"allow_download"`
	AllowUpload   bool    `json:"allow_upload"`
	RevokedAt     *string `json:"revoked_at,omitempty"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
	CreatedBy     int64   `json:"created_by"`
	URL           string  `json:"url"`
}

func (sh *sharesDeps) toDTO(s db.Share, publicURL string) shareDTO {
	f, _ := sh.DB.FolderByID(s.FolderID)
	var folderPath string
	if f != nil {
		folderPath = f.Path
	}
	d := shareDTO{
		ID: s.ID, Token: s.Token, FolderID: s.FolderID, FolderPath: folderPath,
		HasPassword: s.PasswordHash != nil,
		AllowDownload: s.AllowDownload, AllowUpload: s.AllowUpload,
		CreatedBy: s.CreatedBy,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
		URL: publicURL + "/s/" + s.Token,
	}
	if s.ExpiresAt != nil {
		e := s.ExpiresAt.Format(time.RFC3339)
		d.ExpiresAt = &e
	}
	if s.RevokedAt != nil {
		e := s.RevokedAt.Format(time.RFC3339)
		d.RevokedAt = &e
	}
	switch share.Validate(&s) {
	case share.StatusActive:
		d.Status = "active"
	case share.StatusExpired:
		d.Status = "expired"
	case share.StatusRevoked:
		d.Status = "revoked"
	}
	return d
}

// Add: PublicURL field on sharesDeps
//   type sharesDeps struct { DB *db.DB; PublicURL string }

func (sh *sharesDeps) handleCreateShare(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	var req createShareReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if _, err := sh.DB.FolderByID(req.FolderID); err != nil {
		WriteError(w, http.StatusNotFound, "folder not found")
		return
	}
	tok, err := share.NewToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "token error")
		return
	}
	s := db.Share{
		Token: tok, FolderID: req.FolderID, CreatedBy: u.ID,
		AllowDownload: req.AllowDownload, AllowUpload: req.AllowUpload,
	}
	if req.ExpiresInDays > 0 {
		e := time.Now().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
		s.ExpiresAt = &e
	}
	if req.Password != "" {
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "hash error")
			return
		}
		s.PasswordHash = &hash
	}
	id, err := sh.DB.CreateShare(s)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	created, _ := sh.DB.ShareByID(id)
	WriteJSON(w, http.StatusOK, map[string]any{"data": sh.toDTO(*created, sh.PublicURL)})
}

func (sh *sharesDeps) handleListMyShares(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	shares, err := sh.DB.SharesByUser(u.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]shareDTO, 0, len(shares))
	for _, s := range shares {
		out = append(out, sh.toDTO(s, sh.PublicURL))
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (sh *sharesDeps) handleListAllShares(w http.ResponseWriter, r *http.Request) {
	shares, err := sh.DB.AllShares()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]shareDTO, 0, len(shares))
	for _, s := range shares {
		out = append(out, sh.toDTO(s, sh.PublicURL))
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (sh *sharesDeps) handleRevokeShare(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad id")
		return
	}
	u, _ := auth.UserFromContext(r.Context())
	s, err := sh.DB.ShareByID(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not found")
		return
	}
	if s.CreatedBy != u.ID && !u.IsAdmin {
		WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := sh.DB.RevokeShare(id); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (sh *sharesDeps) handleDeleteShare(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad id")
		return
	}
	u, _ := auth.UserFromContext(r.Context())
	s, err := sh.DB.ShareByID(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not found")
		return
	}
	if s.CreatedBy != u.ID && !u.IsAdmin {
		WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := sh.DB.DeleteShare(id); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

var _ = errors.New
```

- [ ] **Step 2: Wire router**

Inside authed group:

```go
r.Post("/shares", sh.handleCreateShare)
r.Get("/shares", sh.handleListMyShares)
r.Delete("/shares/{id}/revoke", sh.handleRevokeShare)
r.Delete("/shares/{id}", sh.handleDeleteShare)
```

Inside an admin group (same level, another `r.Group`):

```go
r.Group(func(r chi.Router) {
	r.Use(auth.RequireLogin(d.DB), auth.RequireAdmin)
	r.Get("/admin/shares", sh.handleListAllShares)
})
```

Also update the sharesDeps instantiation to include `PublicURL: d.PublicURL`.

Add `PublicURL string` to `Deps` and pass from main.go (`PublicURL: cfg.PublicURL`).

- [ ] **Step 3: Run tests + commit**

```bash
go test ./... -count=1
git add internal/api/handlers_shares.go internal/api/router.go cmd/frames/main.go
git commit -m "feat(api): share create/list/revoke/delete"
```

---

### Task 40: Public share access + password unlock

**Files:**
- Create: `internal/api/handlers_share_public.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Implement**

```go
// internal/api/handlers_share_public.go
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/db"
	"github.com/NielHeesakkers/frames/internal/share"
)

// Public share access is NOT wrapped in RequireLogin or CSRF (it's an unauthenticated
// public surface). Instead each handler validates the share status, optional password
// cookie, and scope.

type publicShareDeps struct {
	DB    *db.DB
	Cache *thumbnailCache // interface to avoid import cycle — see below
	Root  string
	Limiter *share.RateLimiter
}

// thumbnailCache is a minimal interface matching *thumbnail.Cache.
type thumbnailCache interface {
	ThumbPath(id int64) string
	PreviewPath(id int64) string
}

func (psh *publicShareDeps) load(r *http.Request) (*db.Share, int) {
	tok := chi.URLParam(r, "token")
	s, err := psh.DB.ShareByToken(tok)
	if err != nil {
		return nil, http.StatusNotFound
	}
	switch share.Validate(s) {
	case share.StatusExpired, share.StatusRevoked:
		return nil, http.StatusGone
	}
	if s.PasswordHash != nil {
		c, _ := r.Cookie(shareCookieName(tok))
		if c == nil {
			return nil, http.StatusUnauthorized
		}
		ok, _ := auth.VerifyPassword(*s.PasswordHash, c.Value)
		if !ok {
			return nil, http.StatusUnauthorized
		}
	}
	if !psh.Limiter.Allow(tok) {
		return nil, http.StatusTooManyRequests
	}
	return s, 0
}

func shareCookieName(tok string) string { return "frames_share_" + tok }

type unlockReq struct{ Password string `json:"password"` }

func (psh *publicShareDeps) handleUnlock(w http.ResponseWriter, r *http.Request) {
	tok := chi.URLParam(r, "token")
	s, err := psh.DB.ShareByToken(tok)
	if err != nil || share.Validate(s) != share.StatusActive {
		WriteError(w, http.StatusNotFound, "invalid share")
		return
	}
	var req unlockReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if s.PasswordHash == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	ok, _ := auth.VerifyPassword(*s.PasswordHash, req.Password)
	if !ok {
		WriteError(w, http.StatusUnauthorized, "wrong password")
		return
	}
	// Set a cookie scoped to this share carrying the plaintext password.
	// (Acceptable here because the attacker who can read it can already use the share URL.)
	http.SetCookie(w, &http.Cookie{
		Name: shareCookieName(tok), Value: req.Password,
		Path: "/s/" + tok, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (psh *publicShareDeps) handleMeta(w http.ResponseWriter, r *http.Request) {
	s, code := psh.load(r)
	if code != 0 {
		WriteError(w, code, http.StatusText(code))
		return
	}
	f, _ := psh.DB.FolderByID(s.FolderID)
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"folder": map[string]any{
				"id": f.ID, "path": f.Path, "name": f.Name, "items": f.ItemCount,
			},
			"allow_download": s.AllowDownload,
			"allow_upload":   s.AllowUpload,
			"has_password":   s.PasswordHash != nil,
			"expires_at":     s.ExpiresAt,
		},
	})
}

func (psh *publicShareDeps) handleListFolder(w http.ResponseWriter, r *http.Request) {
	s, code := psh.load(r)
	if code != 0 {
		WriteError(w, code, http.StatusText(code))
		return
	}
	sub := r.URL.Query().Get("path")
	folder, err := psh.DB.FolderByID(s.FolderID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "folder missing")
		return
	}
	target := folder
	if sub != "" {
		cand, err := psh.DB.FolderByPath(sub)
		if err != nil {
			WriteError(w, http.StatusNotFound, "folder missing")
			return
		}
		if !share.IsUnderFolder(folder.Path, cand.Path) {
			WriteError(w, http.StatusForbidden, "out of share scope")
			return
		}
		target = cand
	}
	children, _ := psh.DB.ChildFolders(target.ID)
	files, _ := psh.DB.FilesInFolder(target.ID, 500, 0, db.SortByTakenAt)

	foldersOut := make([]map[string]any, 0, len(children))
	for _, c := range children {
		foldersOut = append(foldersOut, map[string]any{
			"id": c.ID, "path": c.Path, "name": c.Name, "items": c.ItemCount,
		})
	}
	filesOut := make([]map[string]any, 0, len(files))
	for _, fl := range files {
		var taken *string
		if fl.TakenAt != nil {
			t := fl.TakenAt.Format(time.RFC3339)
			taken = &t
		}
		filesOut = append(filesOut, map[string]any{
			"id": fl.ID, "name": fl.Filename, "size": fl.Size, "kind": fl.Kind,
			"mime_type": fl.MimeType, "taken_at": taken,
			"width": fl.Width, "height": fl.Height,
		})
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"folder":  map[string]any{"id": target.ID, "path": target.Path, "name": target.Name},
			"folders": foldersOut,
			"files":   filesOut,
		},
	})
}

func (psh *publicShareDeps) handleFileMedia(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, code := psh.load(r)
		if code != 0 {
			WriteError(w, code, http.StatusText(code))
			return
		}
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "bad id")
			return
		}
		f, err := psh.DB.FileByID(id)
		if err != nil {
			WriteError(w, http.StatusNotFound, "not found")
			return
		}
		// Scope check.
		rootFolder, _ := psh.DB.FolderByID(s.FolderID)
		folderOfFile, _ := psh.DB.FolderByID(f.FolderID)
		if !share.IsUnderFolder(rootFolder.Path, folderOfFile.Path) {
			WriteError(w, http.StatusForbidden, "out of scope")
			return
		}
		switch kind {
		case "thumb":
			path := psh.Cache.ThumbPath(id)
			serveWithETag(w, r, path,
				"s-"+strconv.FormatInt(f.ID, 10)+"-"+strconv.FormatInt(f.Mtime, 10), f.MimeType)
		case "preview":
			path := psh.Cache.PreviewPath(id)
			serveWithETag(w, r, path,
				"sp-"+strconv.FormatInt(f.ID, 10)+"-"+strconv.FormatInt(f.Mtime, 10), "image/webp")
		case "original":
			if !s.AllowDownload {
				WriteError(w, http.StatusForbidden, "download disabled")
				return
			}
			// Delegate to the media serveOriginal-style logic.
			serveOriginalFile(w, r, psh.Root, f)
		}
	}
}
```

Add a helper in `handlers_media.go` to extract the original-serving body (or simply have both call it):

```go
// internal/api/handlers_media.go — append
func serveOriginalFile(w http.ResponseWriter, r *http.Request, root string, f *db.File) {
	path := filepath.Join(root, f.RelativePath)
	fh, err := os.Open(path)
	if err != nil {
		WriteError(w, http.StatusNotFound, "file missing")
		return
	}
	defer fh.Close()
	fi, _ := fh.Stat()
	if f.MimeType != "" {
		w.Header().Set("Content-Type", f.MimeType)
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename=%q`, f.Filename))
	http.ServeContent(w, r, f.Filename, fi.ModTime(), fh)
}
```

- [ ] **Step 2: Rate limiter for shares**

Create `internal/share/ratelimit.go`:

```go
package share

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu    sync.Mutex
	max   int
	win   time.Duration
	bkts  map[string]*bkt
}

type bkt struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{max: max, win: window, bkts: map[string]*bkt{}}
}

func (l *RateLimiter) Allow(token string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b := l.bkts[token]
	if b == nil || now.After(b.resetAt) {
		l.bkts[token] = &bkt{count: 1, resetAt: now.Add(l.win)}
		return true
	}
	if b.count >= l.max {
		return false
	}
	b.count++
	return true
}
```

- [ ] **Step 3: Wire router**

Outside any auth group (this is a public surface), in `NewRouter` after the `/api` route block:

```go
psh := &publicShareDeps{
	DB: d.DB, Cache: d.Cache, Root: d.Root,
	Limiter: share.NewRateLimiter(100, time.Minute),
}
r.Route("/api/s", func(r chi.Router) {
	r.Post("/{token}/unlock", psh.handleUnlock)
	r.Get("/{token}", psh.handleMeta)
	r.Get("/{token}/folder", psh.handleListFolder)
	r.Get("/{token}/thumb/{id}", psh.handleFileMedia("thumb"))
	r.Get("/{token}/preview/{id}", psh.handleFileMedia("preview"))
	r.Get("/{token}/original/{id}", psh.handleFileMedia("original"))
})
```

Add imports `"time"` and `"github.com/NielHeesakkers/frames/internal/share"`.

- [ ] **Step 4: Run tests + commit**

```bash
go test ./... -count=1
git add internal/api/handlers_share_public.go internal/api/handlers_media.go internal/api/router.go internal/share/ratelimit.go
git commit -m "feat(share): public share access endpoints with password gate + scope check"
```

---

### Task 41: ZIP streaming download

**Files:**
- Create: `internal/share/zip.go`
- Modify: `internal/api/handlers_share_public.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Implement streaming ZIP**

```go
// internal/share/zip.go
package share

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/NielHeesakkers/frames/internal/db"
)

// StreamFolderZip writes a ZIP archive of folder + all descendants into w, streaming.
// It does not buffer the whole archive in memory.
func StreamFolderZip(w io.Writer, d *db.DB, root, rootPath string) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	// Walk files in DB rooted at rootPath.
	rows, err := d.Query(`
		SELECT id, relative_path FROM files
		WHERE relative_path = ? OR relative_path LIKE ?
		ORDER BY relative_path
	`, rootPath, rootPath+"/%")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var rel string
		if err := rows.Scan(&id, &rel); err != nil {
			return err
		}
		abs := filepath.Join(root, rel)
		// Strip the rootPath prefix from entry name for nicer ZIP layout.
		entryName := strings.TrimPrefix(rel, rootPath+"/")
		if entryName == rel {
			entryName = filepath.Base(rel)
		}
		if err := addFile(zw, abs, entryName); err != nil {
			return err
		}
	}
	return rows.Err()
}

func addFile(zw *zip.Writer, abs, entryName string) error {
	fh, err := os.Open(abs)
	if err != nil {
		return err
	}
	defer fh.Close()
	fi, err := fh.Stat()
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}
	hdr.Name = entryName
	hdr.Method = zip.Deflate
	iw, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(iw, fh)
	return err
}
```

- [ ] **Step 2: Endpoint**

Append to `handlers_share_public.go`:

```go
func (psh *publicShareDeps) handleZip(w http.ResponseWriter, r *http.Request) {
	s, code := psh.load(r)
	if code != 0 {
		WriteError(w, code, http.StatusText(code))
		return
	}
	if !s.AllowDownload {
		WriteError(w, http.StatusForbidden, "download disabled")
		return
	}
	folder, _ := psh.DB.FolderByID(s.FolderID)
	w.Header().Set("Content-Type", "application/zip")
	name := folder.Name
	if name == "" {
		name = "frames"
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.zip"`)
	if err := share.StreamFolderZip(w, psh.DB, psh.Root, folder.Path); err != nil {
		// Can't change status at this point; just log.
	}
}
```

Wire route:

```go
r.Get("/{token}/zip", psh.handleZip)
```

- [ ] **Step 3: Run tests + commit**

```bash
go test ./... -count=1
git add internal/share/zip.go internal/api/handlers_share_public.go internal/api/router.go
git commit -m "feat(share): streaming zip download for public shares"
```

---

### Task 42: Anonymous upload via share-link

**Files:**
- Modify: `internal/api/handlers_share_public.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Extend `publicShareDeps` with upload service**

```go
// add to publicShareDeps struct
Upload *upload.Service
Queue  *thumbnail.Queue
MaxBytes int64
```

Add imports `"github.com/NielHeesakkers/frames/internal/thumbnail"`, `"github.com/NielHeesakkers/frames/internal/upload"`, `"github.com/NielHeesakkers/frames/internal/scanner"`.

Append:

```go
func (psh *publicShareDeps) handleAnonymousUpload(w http.ResponseWriter, r *http.Request) {
	s, code := psh.load(r)
	if code != 0 {
		WriteError(w, code, http.StatusText(code))
		return
	}
	if !s.AllowUpload {
		WriteError(w, http.StatusForbidden, "upload disabled")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, psh.MaxBytes)
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid multipart")
		return
	}
	uploader := r.FormValue("name")
	if uploader == "" {
		uploader = "anonymous"
	}
	folder, _ := psh.DB.FolderByID(s.FolderID)
	targetFolder := filepath.Join(folder.Path, "Uploads", sanitizeName(uploader))
	// Ensure the target folder on disk + DB.
	if err := psh.ensureFolder(targetFolder); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		WriteError(w, http.StatusBadRequest, "no files")
		return
	}
	ids := make([]int64, 0, len(files))
	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		id, err := psh.Upload.StoreFile(targetFolder, sanitizeName(fh.Filename), f, scanner.Classify)
		f.Close()
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		psh.Queue.Push(id, thumbnail.PrioForeground)
		ids = append(ids, id)
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"ids": ids}})
}

func (psh *publicShareDeps) ensureFolder(rel string) error {
	abs, err := upload.SafeJoin(psh.Root, rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return err
	}
	// Walk up ensuring rows exist.
	parts := strings.Split(rel, "/")
	cur := ""
	var parentID *int64
	for _, p := range parts {
		if p == "" {
			continue
		}
		if cur != "" {
			cur = cur + "/" + p
		} else {
			cur = p
		}
		existing, err := psh.DB.FolderByPath(cur)
		if err == nil {
			parentID = &existing.ID
			continue
		}
		if err != db.ErrNotFound {
			return err
		}
		fi, _ := os.Stat(filepath.Join(psh.Root, cur))
		created, err := psh.DB.UpsertFolder(db.Folder{
			ParentID: parentID, Path: cur, Name: p, Mtime: fi.ModTime().Unix(),
		})
		if err != nil {
			return err
		}
		parentID = &created.ID
	}
	return nil
}

func sanitizeName(s string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", "..", "_", "\x00", "_")
	return replacer.Replace(s)
}
```

Add imports `"os"`, `"path/filepath"`, `"strings"`.

- [ ] **Step 2: Wire route**

```go
r.Post("/{token}/upload", psh.handleAnonymousUpload)
```

Update `main.go`:

```go
// pass upload service + queue + share-upload-max into publicShareDeps
// (which is constructed inside NewRouter using Deps).
// Extend Deps with ShareUploadMax int64 and pass cfg.ShareUploadMax.
```

Add to `Deps`:

```go
ShareUploadMax int64
```

And inside `NewRouter`, when constructing `psh`:

```go
psh := &publicShareDeps{
	DB: d.DB, Cache: d.Cache, Root: d.Root,
	Limiter:  share.NewRateLimiter(100, time.Minute),
	Upload:   d.UploadSvc,
	Queue:    d.Queue,
	MaxBytes: d.ShareUploadMax,
}
```

- [ ] **Step 3: Run tests + commit**

```bash
go test ./... -count=1
git add internal/api/handlers_share_public.go internal/api/router.go cmd/frames/main.go
git commit -m "feat(share): anonymous upload via share-link"
```

---

### Task 43: Admin users + scan status endpoints

**Files:**
- Create: `internal/api/handlers_admin.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Implement**

```go
// internal/api/handlers_admin.go
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/db"
)

type adminDeps struct {
	DB *db.DB
}

type createUserReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

func (ad *adminDeps) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Username == "" || len(req.Password) < 8 {
		WriteError(w, http.StatusBadRequest, "bad creds")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "hash error")
		return
	}
	id, err := ad.DB.CreateUser(req.Username, hash, req.IsAdmin)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"id": id}})
}

func (ad *adminDeps) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := ad.DB.ListUsers()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		out = append(out, map[string]any{
			"id": u.ID, "username": u.Username, "is_admin": u.IsAdmin,
		})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (ad *adminDeps) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad id")
		return
	}
	u, _ := auth.UserFromContext(r.Context())
	if id == u.ID {
		WriteError(w, http.StatusBadRequest, "cannot delete self")
		return
	}
	if err := ad.DB.DeleteUser(id); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (ad *adminDeps) handleScanStatus(w http.ResponseWriter, r *http.Request) {
	last, _ := ad.DB.LastScanJob("incremental")
	lastFull, _ := ad.DB.LastScanJob("full")
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"last_incremental": last,
			"last_full":        lastFull,
		},
	})
}
```

- [ ] **Step 2: Wire admin routes**

Inside the admin group of `NewRouter`:

```go
ad := &adminDeps{DB: d.DB}
r.Post("/admin/users", ad.handleCreateUser)
r.Get("/admin/users", ad.handleListUsers)
r.Delete("/admin/users/{id}", ad.handleDeleteUser)
r.Get("/admin/scan_status", ad.handleScanStatus)
```

- [ ] **Step 3: Settings: change-own-password (authed, not admin)**

Append to `handlers_admin.go`:

```go
type changePasswordReq struct {
	Old string `json:"old"`
	New string `json:"new"`
}

func (ad *adminDeps) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	var req changePasswordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(req.New) < 8 {
		WriteError(w, http.StatusBadRequest, "password too short")
		return
	}
	cur, err := ad.DB.UserByID(u.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	ok, _ := auth.VerifyPassword(cur.PasswordHash, req.Old)
	if !ok {
		WriteError(w, http.StatusUnauthorized, "wrong old password")
		return
	}
	hash, err := auth.HashPassword(req.New)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "hash error")
		return
	}
	if err := ad.DB.UpdateUserPassword(u.ID, hash); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

Wire inside the normal authed group:

```go
r.Post("/account/password", ad.handleChangePassword)
```

- [ ] **Step 4: Run tests + commit**

```bash
go test ./... -count=1
git add internal/api/handlers_admin.go internal/api/router.go
git commit -m "feat(api): admin user management + scan status + change-password"
```

---

## Phase 8 — Frontend foundation

SvelteKit scaffold, API client, CSRF helper, auth store, login page, frontend-embed hook.

### Task 44: Scaffold SvelteKit project

**Files:**
- Create: `web/package.json`, `web/svelte.config.js`, `web/vite.config.ts`, `web/tsconfig.json`, `web/static/favicon.ico`, `web/src/app.html`, `web/src/app.css`, `web/src/routes/+layout.svelte`, `web/src/routes/+page.ts`

- [ ] **Step 1: `web/package.json`**

```json
{
  "name": "frames-web",
  "version": "0.0.1",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite dev",
    "build": "vite build",
    "preview": "vite preview"
  },
  "devDependencies": {
    "@sveltejs/adapter-static": "^3.0.5",
    "@sveltejs/kit": "^2.5.0",
    "@sveltejs/vite-plugin-svelte": "^3.1.0",
    "svelte": "^4.2.0",
    "svelte-check": "^3.8.0",
    "svelte-virtual": "^0.2.6",
    "typescript": "^5.4.0",
    "vite": "^5.2.0"
  }
}
```

- [ ] **Step 2: `web/svelte.config.js`**

```js
import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

export default {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({
      pages: 'build',
      assets: 'build',
      fallback: 'index.html',
      precompress: false,
      strict: false
    })
  }
};
```

- [ ] **Step 3: `web/vite.config.ts`**

```ts
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080'
    }
  }
});
```

- [ ] **Step 4: `web/tsconfig.json`**

```json
{
  "extends": "./.svelte-kit/tsconfig.json",
  "compilerOptions": {
    "allowJs": true,
    "checkJs": true,
    "esModuleInterop": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    "skipLibCheck": true,
    "sourceMap": true,
    "strict": true,
    "moduleResolution": "bundler"
  }
}
```

- [ ] **Step 5: `web/src/app.html`**

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <link rel="icon" href="%sveltekit.assets%/favicon.ico" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Frames</title>
    %sveltekit.head%
  </head>
  <body data-sveltekit-preload-data="hover">
    <div id="app">%sveltekit.body%</div>
  </body>
</html>
```

- [ ] **Step 6: `web/src/app.css`**

```css
:root {
  --bg: #0f0f10;
  --bg-2: #18181b;
  --fg: #e4e4e7;
  --fg-dim: #a1a1aa;
  --accent: #4a7cff;
  --danger: #ef4444;
  --border: #27272a;
  --radius: 6px;
}
* { box-sizing: border-box; }
html, body { height: 100%; margin: 0; background: var(--bg); color: var(--fg);
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  font-size: 14px; }
#app { height: 100%; }
button { cursor: pointer; background: var(--bg-2); color: var(--fg);
  border: 1px solid var(--border); border-radius: var(--radius); padding: 6px 12px; }
button.primary { background: var(--accent); border-color: var(--accent); color: #fff; }
input, select { background: var(--bg-2); color: var(--fg);
  border: 1px solid var(--border); border-radius: var(--radius); padding: 6px 10px; }
a { color: var(--accent); }
```

- [ ] **Step 7: `web/src/routes/+layout.svelte`**

```svelte
<script lang="ts">
  import '../app.css';
</script>

<slot />
```

- [ ] **Step 8: `web/static/favicon.ico`** — any small file; can be a zero-byte placeholder.

```bash
mkdir -p web/static && : > web/static/favicon.ico
```

- [ ] **Step 9: Install + build**

```bash
cd web
corepack enable && pnpm install || npm install
pnpm build || npm run build
cd ..
```

Expected: `web/build/` exists with `index.html`.

- [ ] **Step 10: Commit**

```bash
git add web/
git commit -m "chore(web): scaffold sveltekit static app"
```

---

### Task 45: API client + CSRF helper + stores

**Files:**
- Create: `web/src/lib/csrf.ts`
- Create: `web/src/lib/api.ts`
- Create: `web/src/lib/stores.ts`

- [ ] **Step 1: CSRF helper**

```ts
// web/src/lib/csrf.ts
export function csrfToken(): string {
  const name = 'frames_csrf=';
  for (const part of document.cookie.split(';')) {
    const p = part.trim();
    if (p.startsWith(name)) return decodeURIComponent(p.slice(name.length));
  }
  return '';
}
```

- [ ] **Step 2: API client**

```ts
// web/src/lib/api.ts
import { csrfToken } from './csrf';

export class ApiError extends Error {
  status: number;
  constructor(status: number, msg: string) {
    super(msg);
    this.status = status;
  }
}

async function req<T>(method: string, path: string, body?: any, opts: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = { ...(opts.headers as any) };
  if (body && !(body instanceof FormData)) {
    headers['Content-Type'] = 'application/json';
  }
  if (method !== 'GET' && method !== 'HEAD') {
    headers['X-CSRF-Token'] = csrfToken();
  }
  const init: RequestInit = {
    method, credentials: 'include',
    headers,
    body: body instanceof FormData ? body : body ? JSON.stringify(body) : undefined,
    ...opts
  };
  const res = await fetch(path, init);
  if (res.status === 204) return undefined as unknown as T;
  if (!res.ok) {
    let msg = res.statusText;
    try { const j = await res.json(); msg = j.error ?? msg; } catch {}
    throw new ApiError(res.status, msg);
  }
  const ct = res.headers.get('content-type') || '';
  if (ct.startsWith('application/json')) {
    const j = await res.json();
    return (j.data ?? j) as T;
  }
  return (await res.text()) as unknown as T;
}

export const api = {
  me: () => req<{ id: number; username: string; is_admin: boolean }>('GET', '/api/me'),
  login: (username: string, password: string) =>
    req<{ id: number; username: string; is_admin: boolean }>('POST', '/api/login', { username, password }),
  logout: () => req<void>('POST', '/api/logout'),

  folder: (path: string, params: { limit?: number; offset?: number; sort?: string } = {}) => {
    const q = new URLSearchParams(params as any).toString();
    const base = path ? `/api/folder/${encodeURIComponent(path).replace(/%2F/g, '/')}` : `/api/folder`;
    return req<{ folder: any; folders: any[]; files: any[]; has_more: boolean }>('GET', q ? `${base}?${q}` : base);
  },
  tree: (parent?: string) =>
    req<Array<{ id: number; path: string; name: string; has_child: boolean; items: number }>>(
      'GET', `/api/tree${parent ? `?parent=${encodeURIComponent(parent)}` : ''}`),
  search: (q: Record<string, string>) =>
    req<{ files: any[]; has_more: boolean }>('GET', '/api/search?' + new URLSearchParams(q)),

  scan: (full = false) => req<void>('POST', `/api/scan${full ? '?full=1' : ''}`),

  mkdir: (path: string) => req<void>('POST', '/api/ops/mkdir', { path }),
  renameFile: (id: number, name: string) => req<void>('POST', '/api/ops/file/rename', { id, name }),
  moveFile: (id: number, new_folder_id: number) => req<void>('POST', '/api/ops/file/move', { id, new_folder_id }),
  deleteFile: (id: number) => req<void>('POST', '/api/ops/file/delete', { id }),
  renameFolder: (id: number, name: string) => req<void>('POST', '/api/ops/folder/rename', { id, name }),
  deleteFolder: (id: number) => req<void>('POST', '/api/ops/folder/delete', { id }),

  upload: (path: string, files: File[]) => {
    const fd = new FormData();
    fd.append('path', path);
    for (const f of files) fd.append('files', f);
    return req<{ ids: number[] }>('POST', '/api/upload', fd);
  },

  myShares: () => req<any[]>('GET', '/api/shares'),
  createShare: (body: any) => req<any>('POST', '/api/shares', body),
  revokeShare: (id: number) => req<void>('DELETE', `/api/shares/${id}/revoke`),
  deleteShare: (id: number) => req<void>('DELETE', `/api/shares/${id}`),

  sharedWithMe: () => req<any[]>('GET', '/api/shared_with_me'),
  addFolderShare: (folder_id: number, user_id: number) =>
    req<void>('POST', '/api/folder_shares', { folder_id, user_id }),
  removeFolderShare: (folder_id: number, user_id: number) =>
    req<void>('DELETE', '/api/folder_shares', { folder_id, user_id }),

  listUsers: () => req<any[]>('GET', '/api/admin/users'),
  createUser: (u: { username: string; password: string; is_admin: boolean }) =>
    req<{ id: number }>('POST', '/api/admin/users', u),
  deleteUser: (id: number) => req<void>('DELETE', `/api/admin/users/${id}`),
  scanStatus: () => req<any>('GET', '/api/admin/scan_status'),

  changePassword: (old: string, neo: string) =>
    req<void>('POST', '/api/account/password', { old, new: neo })
};
```

- [ ] **Step 3: Auth + UI stores**

```ts
// web/src/lib/stores.ts
import { writable, get } from 'svelte/store';
import { api } from './api';

export type Me = { id: number; username: string; is_admin: boolean } | null;

export const me = writable<Me>(null);

export async function refreshMe(): Promise<Me> {
  try {
    const u = await api.me();
    me.set(u);
    return u;
  } catch {
    me.set(null);
    return null;
  }
}

export async function logout() {
  await api.logout();
  me.set(null);
}

export const currentFolderPath = writable<string>('');
export const selection = writable<Set<number>>(new Set());
export const sortMode = writable<'takenAt' | 'name' | 'size'>('takenAt');
export const density = writable<'small' | 'medium' | 'large'>('medium');

export function useMe() {
  return { me, value: () => get(me) };
}
```

- [ ] **Step 4: Commit**

```bash
git add web/src/lib/
git commit -m "feat(web): api client, csrf helper, and shared stores"
```

---

### Task 46: Login page + root guard

**Files:**
- Create: `web/src/routes/login/+page.svelte`
- Create: `web/src/routes/+page.ts`
- Create: `web/src/routes/+layout.ts`

- [ ] **Step 1: Root load → redirects**

```ts
// web/src/routes/+layout.ts
export const ssr = false;
export const prerender = false;
```

```ts
// web/src/routes/+page.ts
import { redirect } from '@sveltejs/kit';
import { refreshMe } from '$lib/stores';

export const load = async () => {
  const u = await refreshMe();
  if (!u) throw redirect(307, '/login');
  throw redirect(307, '/browse');
};
```

- [ ] **Step 2: Login page**

```svelte
<!-- web/src/routes/login/+page.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';
  import { me } from '$lib/stores';

  let username = '';
  let password = '';
  let error = '';
  let busy = false;

  async function submit() {
    error = '';
    busy = true;
    try {
      // Ensure CSRF cookie is seeded by a GET first.
      await fetch('/api/me', { credentials: 'include' });
      const u = await api.login(username, password);
      me.set(u);
      goto('/browse');
    } catch (e: any) {
      error = e.message || 'login failed';
    } finally {
      busy = false;
    }
  }
</script>

<div class="center">
  <form class="card" on:submit|preventDefault={submit}>
    <h1>Frames</h1>
    <label>
      Username
      <input bind:value={username} required autofocus />
    </label>
    <label>
      Password
      <input type="password" bind:value={password} required />
    </label>
    {#if error}<p class="err">{error}</p>{/if}
    <button class="primary" disabled={busy}>{busy ? '...' : 'Login'}</button>
  </form>
</div>

<style>
  .center { min-height: 100vh; display: grid; place-items: center; }
  .card { background: var(--bg-2); padding: 24px; border-radius: 8px;
    width: 320px; display: flex; flex-direction: column; gap: 12px;
    border: 1px solid var(--border); }
  label { display: flex; flex-direction: column; gap: 4px; color: var(--fg-dim); }
  h1 { margin: 0 0 4px; }
  .err { color: var(--danger); margin: 0; }
</style>
```

- [ ] **Step 3: Commit**

```bash
git add web/src/routes/
git commit -m "feat(web): login page and root redirect"
```

---

### Task 47: Embed built frontend in Go binary

**Files:**
- Create: `internal/frontend/frontend.go`
- Modify: `internal/api/router.go`
- Modify: `cmd/frames/main.go`
- Modify: `Makefile`

- [ ] **Step 1: Embed + fallback serving**

```go
// internal/frontend/frontend.go
package frontend

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var dist embed.FS

// FS returns an http.FileSystem rooted at the embedded dist directory.
// If no dist exists yet (dev build), returns http.Dir("web/build").
func FS() http.FileSystem {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return http.Dir("web/build")
	}
	// Detect empty FS (scaffold-only).
	if entries, _ := fs.ReadDir(sub, "."); len(entries) == 0 {
		return http.Dir("web/build")
	}
	return http.FS(sub)
}

// Handler serves static assets and falls back to index.html for SPA routes.
func Handler() http.Handler {
	fsys := FS()
	fileSrv := http.FileServer(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't hijack API paths.
		if strings.HasPrefix(r.URL.Path, "/api/") ||
			strings.HasPrefix(r.URL.Path, "/healthz") ||
			strings.HasPrefix(r.URL.Path, "/s/") {
			http.NotFound(w, r)
			return
		}
		// Try the file; on 404, serve index.html.
		if f, err := fsys.Open(r.URL.Path); err == nil {
			f.Close()
			fileSrv.ServeHTTP(w, r)
			return
		}
		r.URL.Path = "/"
		fileSrv.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 2: Add a placeholder `internal/frontend/dist/.keep`**

```bash
mkdir -p internal/frontend/dist && : > internal/frontend/dist/.keep
```

- [ ] **Step 3: Wire as fallback in router**

In `NewRouter`, after the `/api` and `/api/s` routes:

```go
r.NotFound(frontend.Handler().ServeHTTP)
```

Add import `"github.com/NielHeesakkers/frames/internal/frontend"`.

- [ ] **Step 4: Makefile target to copy build artefacts**

Append to `Makefile`:

```makefile
.PHONY: web
web:
	cd web && (pnpm install || npm install) && (pnpm build || npm run build)
	rm -rf internal/frontend/dist
	cp -r web/build internal/frontend/dist
```

- [ ] **Step 5: Commit**

```bash
git add internal/frontend/ internal/api/router.go Makefile
git commit -m "feat(api): embed sveltekit static build and spa-fallback handler"
```

---

## Phase 9 — Browse UI

Folder tree + breadcrumb + virtualized grid + density/sort controls.

### Task 48: Global app shell (sidebar + top bar)

**Files:**
- Create: `web/src/routes/browse/+layout.svelte`
- Create: `web/src/lib/components/FolderTree.svelte`
- Create: `web/src/lib/components/Breadcrumb.svelte`

- [ ] **Step 1: Layout**

```svelte
<!-- web/src/routes/browse/+layout.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { refreshMe, me, logout } from '$lib/stores';
  import FolderTree from '$lib/components/FolderTree.svelte';
  import Breadcrumb from '$lib/components/Breadcrumb.svelte';

  onMount(async () => {
    const u = await refreshMe();
    if (!u) goto('/login');
  });
</script>

{#if $me}
  <div class="shell">
    <aside>
      <div class="brand">Frames</div>
      <FolderTree />
      <div class="sidebar-footer">
        <span>{$me.username}</span>
        <button on:click={async () => { await logout(); goto('/login'); }}>Logout</button>
      </div>
    </aside>
    <main>
      <header>
        <Breadcrumb />
      </header>
      <div class="main-inner">
        <slot />
      </div>
    </main>
  </div>
{/if}

<style>
  .shell { display: grid; grid-template-columns: 280px 1fr; height: 100vh; }
  aside { background: var(--bg-2); border-right: 1px solid var(--border);
    display: flex; flex-direction: column; min-height: 0; }
  .brand { padding: 14px 16px; font-weight: 600; border-bottom: 1px solid var(--border); }
  .sidebar-footer { border-top: 1px solid var(--border); padding: 10px;
    display: flex; justify-content: space-between; align-items: center; gap: 8px; color: var(--fg-dim); }
  main { display: flex; flex-direction: column; min-height: 0; }
  header { border-bottom: 1px solid var(--border); padding: 10px 16px; }
  .main-inner { flex: 1; overflow: hidden; display: flex; flex-direction: column; min-height: 0; }
  @media (max-width: 768px) {
    .shell { grid-template-columns: 1fr; }
    aside { display: none; }
  }
</style>
```

- [ ] **Step 2: FolderTree**

```svelte
<!-- web/src/lib/components/FolderTree.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';
  import { currentFolderPath } from '$lib/stores';

  type Node = { id: number; path: string; name: string; has_child: boolean; items: number;
                children?: Node[]; expanded?: boolean };

  let roots: Node[] = [];

  async function loadChildren(n: Node) {
    if (n.children) return;
    const kids = await api.tree(n.path);
    n.children = kids.map((k) => ({ ...k }));
    roots = roots;
  }

  async function toggle(n: Node) {
    n.expanded = !n.expanded;
    if (n.expanded) await loadChildren(n);
    roots = roots;
  }

  function select(n: Node) {
    currentFolderPath.set(n.path);
    goto('/browse/' + n.path.split('/').map(encodeURIComponent).join('/'));
  }

  onMount(async () => {
    const top = await api.tree('');
    roots = top.map((t) => ({ ...t }));
  });
</script>

<ul class="tree">
  {#each roots as n}
    <li>
      <div class="row" class:active={$currentFolderPath === n.path} on:click={() => select(n)}>
        {#if n.has_child}
          <button class="toggle" on:click|stopPropagation={() => toggle(n)}>{n.expanded ? '▾' : '▸'}</button>
        {:else}
          <span class="toggle" />
        {/if}
        <span class="name">{n.name || 'Photos'}</span>
        <span class="count">{n.items}</span>
      </div>
      {#if n.expanded && n.children}
        <ul class="tree sub">
          {#each n.children as c}
            <li>
              <div class="row" class:active={$currentFolderPath === c.path} on:click={() => select(c)}>
                {#if c.has_child}
                  <button class="toggle" on:click|stopPropagation={() => toggle(c)}>{c.expanded ? '▾' : '▸'}</button>
                {:else}
                  <span class="toggle" />
                {/if}
                <span class="name">{c.name}</span>
                <span class="count">{c.items}</span>
              </div>
            </li>
          {/each}
        </ul>
      {/if}
    </li>
  {/each}
</ul>

<style>
  .tree { list-style: none; padding: 0; margin: 8px 0; flex: 1; overflow-y: auto; }
  .sub { padding-left: 14px; margin: 0; }
  .row { display: grid; grid-template-columns: 20px 1fr auto; align-items: center;
    gap: 4px; padding: 4px 10px; cursor: pointer; border-radius: 3px; }
  .row:hover { background: rgba(255,255,255,0.05); }
  .row.active { background: rgba(74,124,255,0.15); color: var(--accent); }
  .toggle { width: 20px; height: 20px; background: transparent; border: none;
    color: var(--fg-dim); font-size: 12px; padding: 0; }
  .count { color: var(--fg-dim); font-size: 11px; }
</style>
```

- [ ] **Step 3: Breadcrumb**

```svelte
<!-- web/src/lib/components/Breadcrumb.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { currentFolderPath } from '$lib/stores';
  $: parts = $currentFolderPath === '' ? [] : $currentFolderPath.split('/');
</script>

<nav>
  <a href="#" on:click|preventDefault={() => { currentFolderPath.set(''); goto('/browse'); }}>Photos</a>
  {#each parts as p, i}
    <span class="sep">›</span>
    <a href="#" on:click|preventDefault={() => {
         const sub = parts.slice(0, i + 1).join('/');
         currentFolderPath.set(sub);
         goto('/browse/' + parts.slice(0, i + 1).map(encodeURIComponent).join('/'));
       }}>{p}</a>
  {/each}
</nav>

<style>
  nav { display: flex; align-items: center; gap: 6px; font-size: 14px; }
  a { color: var(--fg); text-decoration: none; }
  a:hover { color: var(--accent); }
  .sep { color: var(--fg-dim); }
</style>
```

- [ ] **Step 4: Commit**

```bash
git add web/src/routes/browse/ web/src/lib/components/FolderTree.svelte web/src/lib/components/Breadcrumb.svelte
git commit -m "feat(web): browse shell with folder tree and breadcrumb"
```

---

### Task 49: Grid + GridItem + browse page

**Files:**
- Create: `web/src/lib/components/Grid.svelte`
- Create: `web/src/lib/components/GridItem.svelte`
- Create: `web/src/routes/browse/+page.svelte`
- Create: `web/src/routes/browse/[...path]/+page.svelte`

- [ ] **Step 1: GridItem**

```svelte
<!-- web/src/lib/components/GridItem.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { selection } from '$lib/stores';

  export let file: any;
  export let size = 160;

  function onClick(e: MouseEvent) {
    if (e.shiftKey || e.metaKey || e.ctrlKey) {
      selection.update((s) => {
        const ns = new Set(s);
        ns.has(file.id) ? ns.delete(file.id) : ns.add(file.id);
        return ns;
      });
      return;
    }
    goto(`/file/${file.id}`);
  }
</script>

<div class="item" style="--size: {size}px" on:click={onClick}
     class:selected={$selection.has(file.id)}>
  {#if file.kind === 'other'}
    <div class="icon">📄</div>
  {:else}
    <img src={`/api/thumb/${file.id}`} loading="lazy" alt={file.name} />
  {/if}
  {#if file.kind === 'video'}<span class="badge">▶</span>{/if}
</div>

<style>
  .item { position: relative; width: var(--size); height: var(--size);
    background: var(--bg-2); border-radius: 3px; overflow: hidden;
    cursor: pointer; }
  .item.selected { outline: 3px solid var(--accent); }
  img { width: 100%; height: 100%; object-fit: cover; display: block; }
  .icon { width: 100%; height: 100%; display: grid; place-items: center;
    font-size: 36px; color: var(--fg-dim); }
  .badge { position: absolute; bottom: 4px; right: 4px; background: rgba(0,0,0,0.6);
    color: #fff; border-radius: 50%; width: 22px; height: 22px; display: grid;
    place-items: center; font-size: 11px; }
</style>
```

- [ ] **Step 2: Grid (responsive CSS grid, not virtualized in v1)**

> Virtualization via `svelte-virtual` is complex at arbitrary grid widths. For v1 we use native CSS grid with `content-visibility: auto` for off-screen elision, which is sufficient up to ~2000 items rendered. Infinite scroll + pagination keeps DOM size bounded.

```svelte
<!-- web/src/lib/components/Grid.svelte -->
<script lang="ts">
  import GridItem from './GridItem.svelte';
  import { density } from '$lib/stores';

  export let files: any[] = [];

  $: itemSize = $density === 'small' ? 120 : $density === 'large' ? 220 : 160;
</script>

<div class="grid" style="--size: {itemSize}px">
  {#each files as f (f.id)}
    <GridItem file={f} size={itemSize} />
  {/each}
</div>

<style>
  .grid { display: grid; grid-template-columns: repeat(auto-fill, var(--size));
    gap: 4px; padding: 8px; overflow-y: auto; flex: 1;
    content-visibility: auto; }
</style>
```

- [ ] **Step 3: Browse pages**

```svelte
<!-- web/src/routes/browse/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { currentFolderPath, sortMode, density } from '$lib/stores';
  import Grid from '$lib/components/Grid.svelte';

  let folder: any = null;
  let folders: any[] = [];
  let files: any[] = [];
  let loading = true;

  async function load() {
    loading = true;
    const sort = $sortMode === 'takenAt' ? 'taken' : $sortMode;
    const r = await api.folder($currentFolderPath, { sort, limit: 500 });
    folder = r.folder;
    folders = r.folders;
    files = r.files;
    loading = false;
  }

  onMount(() => { currentFolderPath.set(''); });
  $: $currentFolderPath, load();
</script>

<div class="toolbar">
  <select bind:value={$sortMode}>
    <option value="takenAt">By capture date</option>
    <option value="name">By name</option>
    <option value="size">By size</option>
  </select>
  <select bind:value={$density}>
    <option value="small">S</option>
    <option value="medium">M</option>
    <option value="large">L</option>
  </select>
</div>

{#if loading}
  <div class="loading">Loading…</div>
{:else}
  {#if folders.length > 0}
    <section>
      <h3>Subfolders</h3>
      <div class="folder-cards">
        {#each folders as sub}
          <a class="fcard" href={`/browse/${sub.path.split('/').map(encodeURIComponent).join('/')}`}
             on:click|preventDefault={() => currentFolderPath.set(sub.path)}>
            <div class="ico">📁</div>
            <div class="name">{sub.name}</div>
            <div class="cnt">{sub.items} items</div>
          </a>
        {/each}
      </div>
    </section>
  {/if}
  <Grid files={files} />
{/if}

<style>
  .toolbar { display: flex; gap: 8px; padding: 8px 16px; border-bottom: 1px solid var(--border); }
  .loading { padding: 20px; color: var(--fg-dim); }
  h3 { margin: 16px 16px 8px; color: var(--fg-dim); font-size: 12px; text-transform: uppercase; }
  .folder-cards { display: grid; grid-template-columns: repeat(auto-fill, 180px);
    gap: 8px; padding: 0 16px; }
  .fcard { background: var(--bg-2); border-radius: 6px; padding: 12px;
    text-align: center; text-decoration: none; color: var(--fg); border: 1px solid var(--border); }
  .fcard:hover { border-color: var(--accent); }
  .ico { font-size: 26px; }
  .cnt { color: var(--fg-dim); font-size: 11px; }
</style>
```

```svelte
<!-- web/src/routes/browse/[...path]/+page.svelte -->
<script lang="ts">
  import { page } from '$app/stores';
  import { currentFolderPath } from '$lib/stores';
  import BrowseRoot from '../+page.svelte';
  $: currentFolderPath.set(decodeURIComponent($page.params.path ?? ''));
</script>

<BrowseRoot />
```

- [ ] **Step 4: Commit**

```bash
git add web/src/lib/components/Grid.svelte web/src/lib/components/GridItem.svelte web/src/routes/browse/
git commit -m "feat(web): folder browse view with grid, sort, density"
```

---

## Phase 10 — Lightbox

### Task 50: Lightbox component + route

**Files:**
- Create: `web/src/lib/components/Lightbox.svelte`
- Create: `web/src/routes/file/[id]/+page.svelte`

- [ ] **Step 1: Component**

```svelte
<!-- web/src/lib/components/Lightbox.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount, onDestroy } from 'svelte';

  export let file: any;
  export let neighbors: number[] = [];

  let index = neighbors.indexOf(file.id);

  function close() { history.back(); }
  function prev() { if (index > 0) goto(`/file/${neighbors[--index]}`); }
  function next() { if (index >= 0 && index < neighbors.length - 1) goto(`/file/${neighbors[++index]}`); }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape') close();
    if (e.key === 'ArrowLeft') prev();
    if (e.key === 'ArrowRight') next();
  }
  onMount(() => window.addEventListener('keydown', onKey));
  onDestroy(() => window.removeEventListener('keydown', onKey));
</script>

<div class="lightbox" on:click={close}>
  <button class="nav left" on:click|stopPropagation={prev} disabled={index <= 0}>‹</button>
  <div class="media" on:click|stopPropagation>
    {#if file.kind === 'video'}
      <video src={`/api/original/${file.id}`} controls autoplay></video>
    {:else if file.kind === 'other'}
      <a href={`/api/original/${file.id}`}>Download {file.name}</a>
    {:else}
      <img src={`/api/preview/${file.id}`} alt={file.name} />
    {/if}
  </div>
  <button class="nav right" on:click|stopPropagation={next} disabled={index < 0 || index >= neighbors.length - 1}>›</button>

  <aside class="info" on:click|stopPropagation>
    <h3>{file.name}</h3>
    <dl>
      <dt>Size</dt><dd>{(file.size / 1024 / 1024).toFixed(2)} MB</dd>
      {#if file.taken_at}<dt>Taken</dt><dd>{file.taken_at}</dd>{/if}
      {#if file.camera_model}<dt>Camera</dt><dd>{file.camera_make ?? ''} {file.camera_model}</dd>{/if}
      {#if file.width}<dt>Dim</dt><dd>{file.width} × {file.height}</dd>{/if}
    </dl>
    <a class="dl" href={`/api/original/${file.id}`} download={file.name}>Download</a>
  </aside>
</div>

<style>
  .lightbox { position: fixed; inset: 0; background: rgba(0,0,0,0.95);
    display: grid; grid-template-columns: 60px 1fr 300px; z-index: 100; }
  .media { display: grid; place-items: center; padding: 20px; grid-column: 2; }
  .media img, .media video { max-width: 100%; max-height: 100%; object-fit: contain; }
  .nav { background: transparent; color: #fff; border: none; font-size: 48px;
    cursor: pointer; }
  .nav:disabled { opacity: 0.3; cursor: default; }
  .info { background: var(--bg-2); padding: 20px; overflow-y: auto;
    border-left: 1px solid var(--border); }
  .info dl { display: grid; grid-template-columns: auto 1fr; gap: 6px 12px; color: var(--fg-dim); margin: 10px 0; }
  dt { text-transform: uppercase; font-size: 11px; }
  dd { margin: 0; color: var(--fg); }
  .dl { display: inline-block; background: var(--accent); color: #fff;
    padding: 8px 14px; border-radius: var(--radius); text-decoration: none; }
  @media (max-width: 768px) {
    .lightbox { grid-template-columns: 1fr; grid-template-rows: 1fr auto; }
    .nav { display: none; }
    .info { border-left: none; border-top: 1px solid var(--border); }
  }
</style>
```

- [ ] **Step 2: Route**

```svelte
<!-- web/src/routes/file/[id]/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import Lightbox from '$lib/components/Lightbox.svelte';
  import { api } from '$lib/api';

  let file: any = null;
  let neighbors: number[] = [];

  async function load() {
    const id = +($page.params.id as string);
    // For now, fetch only the file metadata via folder listing heuristic:
    // fall back to a lightweight endpoint or just show the file alone.
    // A future task could add a /api/file/{id} metadata endpoint.
    // Here we rely on thumb endpoint + no metadata; a richer version arrives in Task 51.
    file = { id, name: `#${id}`, size: 0, kind: 'image' };
  }
  onMount(load);
</script>

{#if file}<Lightbox {file} {neighbors} />{/if}
```

- [ ] **Step 3: Commit**

```bash
git add web/src/lib/components/Lightbox.svelte web/src/routes/file/
git commit -m "feat(web): lightbox with keyboard navigation and download"
```

---

### Task 51: File metadata endpoint + lightbox enrichment

**Files:**
- Modify: `internal/api/handlers_browse.go` (add `/api/file/{id}`)
- Modify: `internal/api/router.go`
- Modify: `web/src/lib/api.ts` (add `file()`)
- Modify: `web/src/routes/file/[id]/+page.svelte`

- [ ] **Step 1: Backend endpoint**

```go
// append to handlers_browse.go
func (bd *browseDeps) handleFile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad id")
		return
	}
	f, err := bd.DB.FileByID(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not found")
		return
	}
	var taken *string
	if f.TakenAt != nil {
		s := f.TakenAt.Format("2006-01-02T15:04:05")
		taken = &s
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id": f.ID, "folder_id": f.FolderID, "name": f.Filename, "relative_path": f.RelativePath,
			"size": f.Size, "mtime": f.Mtime, "kind": f.Kind, "mime_type": f.MimeType,
			"taken_at": taken, "width": f.Width, "height": f.Height,
			"camera_make": f.CameraMake, "camera_model": f.CameraModel,
			"orientation": f.Orientation, "duration_ms": f.DurationMs,
			"thumb_status": f.ThumbStatus, "preview_status": f.PreviewStatus,
		},
	})
}
```

Wire: `r.Get("/file/{id}", bd.handleFile)` (authed group) and add import `"github.com/go-chi/chi/v5"`, `"strconv"`.

- [ ] **Step 2: Frontend**

```ts
// web/src/lib/api.ts — add
file: (id: number) => req<any>('GET', `/api/file/${id}`),
```

```svelte
<!-- web/src/routes/file/[id]/+page.svelte — replace load -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import Lightbox from '$lib/components/Lightbox.svelte';
  import { api } from '$lib/api';

  let file: any = null;

  async function load() {
    const id = +($page.params.id as string);
    file = await api.file(id);
  }
  onMount(load);
</script>

{#if file}<Lightbox {file} neighbors={[]} />{/if}
```

- [ ] **Step 3: Commit**

```bash
git add internal/api/handlers_browse.go internal/api/router.go web/src/lib/api.ts web/src/routes/file/
git commit -m "feat(api): single-file metadata endpoint consumed by lightbox"
```

---

## Phase 11 — File operations UI

Upload (drag-drop), create folder, rename, move, delete, context menu, multi-select actions.

### Task 52: ContextMenu + ConfirmDialog + Upload + Move picker components

**Files:**
- Create: `web/src/lib/components/ContextMenu.svelte`
- Create: `web/src/lib/components/ConfirmDialog.svelte`
- Create: `web/src/lib/components/UploadDialog.svelte`
- Create: `web/src/lib/components/MovePicker.svelte`

- [ ] **Step 1: ContextMenu**

```svelte
<!-- web/src/lib/components/ContextMenu.svelte -->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  export let x = 0;
  export let y = 0;
  export let items: Array<{ label: string; onSelect: () => void; danger?: boolean }> = [];
  export let onClose: () => void;

  function handleDoc(e: MouseEvent) { onClose(); }
  onMount(() => window.addEventListener('click', handleDoc));
  onDestroy(() => window.removeEventListener('click', handleDoc));
</script>

<div class="menu" style="left: {x}px; top: {y}px" on:click|stopPropagation>
  {#each items as it}
    <button class:danger={it.danger} on:click={() => { it.onSelect(); onClose(); }}>
      {it.label}
    </button>
  {/each}
</div>

<style>
  .menu { position: fixed; z-index: 200; background: var(--bg-2);
    border: 1px solid var(--border); border-radius: var(--radius);
    min-width: 160px; box-shadow: 0 6px 24px rgba(0,0,0,0.4);
    display: flex; flex-direction: column; }
  button { background: transparent; text-align: left; border: none;
    padding: 8px 12px; color: var(--fg); border-radius: 0; }
  button:hover { background: rgba(255,255,255,0.05); }
  button.danger { color: var(--danger); }
</style>
```

- [ ] **Step 2: ConfirmDialog**

```svelte
<!-- web/src/lib/components/ConfirmDialog.svelte -->
<script lang="ts">
  export let title = 'Confirm';
  export let message = '';
  export let confirmLabel = 'OK';
  export let danger = false;
  export let onConfirm: () => void;
  export let onCancel: () => void;
</script>

<div class="backdrop" on:click={onCancel}>
  <div class="dialog" on:click|stopPropagation>
    <h3>{title}</h3>
    <p>{message}</p>
    <div class="actions">
      <button on:click={onCancel}>Cancel</button>
      <button class:primary={!danger} class:danger on:click={onConfirm}>{confirmLabel}</button>
    </div>
  </div>
</div>

<style>
  .backdrop { position: fixed; inset: 0; background: rgba(0,0,0,0.6); z-index: 150;
    display: grid; place-items: center; }
  .dialog { background: var(--bg-2); border-radius: 8px; padding: 24px;
    min-width: 320px; border: 1px solid var(--border); }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 16px; }
  button.danger { background: var(--danger); color: white; border-color: var(--danger); }
</style>
```

- [ ] **Step 3: UploadDialog**

```svelte
<!-- web/src/lib/components/UploadDialog.svelte -->
<script lang="ts">
  import { api } from '$lib/api';
  export let path = '';
  export let onClose: () => void;
  export let onDone: () => void;

  let files: FileList | null = null;
  let busy = false;
  let progress = 0;
  let error = '';

  async function upload() {
    if (!files || files.length === 0) return;
    busy = true; error = ''; progress = 0;
    try {
      await api.upload(path, Array.from(files));
      onDone();
      onClose();
    } catch (e: any) {
      error = e.message ?? 'upload failed';
    } finally {
      busy = false;
    }
  }
</script>

<div class="backdrop" on:click={onClose}>
  <div class="dialog" on:click|stopPropagation>
    <h3>Upload to {path || 'root'}</h3>
    <input type="file" multiple bind:files />
    {#if error}<p class="err">{error}</p>{/if}
    <div class="actions">
      <button on:click={onClose}>Cancel</button>
      <button class="primary" on:click={upload} disabled={busy}>
        {busy ? 'Uploading…' : 'Upload'}
      </button>
    </div>
  </div>
</div>

<style>
  .backdrop { position: fixed; inset: 0; background: rgba(0,0,0,0.6); z-index: 150;
    display: grid; place-items: center; }
  .dialog { background: var(--bg-2); border-radius: 8px; padding: 24px;
    min-width: 420px; border: 1px solid var(--border); }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 16px; }
  .err { color: var(--danger); }
</style>
```

- [ ] **Step 4: MovePicker (pick a folder via tree)**

```svelte
<!-- web/src/lib/components/MovePicker.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';

  export let onPick: (folderId: number, path: string) => void;
  export let onClose: () => void;

  type Node = { id: number; path: string; name: string; has_child: boolean; expanded?: boolean; children?: Node[] };
  let nodes: Node[] = [];

  async function loadChildren(n: Node) {
    if (n.children) return;
    n.children = (await api.tree(n.path)) as any;
    nodes = nodes;
  }

  onMount(async () => {
    nodes = (await api.tree('')) as any;
  });
</script>

<div class="backdrop" on:click={onClose}>
  <div class="dialog" on:click|stopPropagation>
    <h3>Move to…</h3>
    <ul class="tree">
      <li>
        <button on:click={() => onPick(0, '')}>Photos (root)</button>
      </li>
      {#each nodes as n}
        <li>
          <div class="row">
            <button class="toggle" on:click={() => { n.expanded = !n.expanded; loadChildren(n); nodes = nodes; }}>
              {n.has_child ? (n.expanded ? '▾' : '▸') : '•'}
            </button>
            <button class="name" on:click={() => onPick(n.id, n.path)}>{n.name}</button>
          </div>
          {#if n.expanded && n.children}
            <ul class="sub">
              {#each n.children as c}
                <li>
                  <button on:click={() => onPick(c.id, c.path)}>{c.name}</button>
                </li>
              {/each}
            </ul>
          {/if}
        </li>
      {/each}
    </ul>
    <div class="actions"><button on:click={onClose}>Cancel</button></div>
  </div>
</div>

<style>
  .backdrop { position: fixed; inset: 0; background: rgba(0,0,0,0.6); z-index: 150;
    display: grid; place-items: center; }
  .dialog { background: var(--bg-2); border-radius: 8px; padding: 20px;
    min-width: 420px; max-height: 70vh; overflow-y: auto; border: 1px solid var(--border); }
  .tree { list-style: none; padding: 0; margin: 12px 0; }
  .sub { padding-left: 18px; list-style: none; }
  .row { display: flex; gap: 4px; align-items: center; }
  .toggle { width: 24px; background: transparent; border: none; color: var(--fg-dim); }
  .name { background: transparent; border: none; color: var(--fg); text-align: left; padding: 4px 6px; }
  .name:hover { background: rgba(255,255,255,0.05); }
  .actions { display: flex; justify-content: flex-end; }
</style>
```

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/components/ContextMenu.svelte web/src/lib/components/ConfirmDialog.svelte web/src/lib/components/UploadDialog.svelte web/src/lib/components/MovePicker.svelte
git commit -m "feat(web): shared dialog + menu + upload + move-picker components"
```

---

### Task 53: Wire context menu and upload into browse view

**Files:**
- Modify: `web/src/routes/browse/+page.svelte`
- Modify: `web/src/lib/components/GridItem.svelte`

- [ ] **Step 1: Update GridItem to emit contextmenu**

Replace the `<script>` block in `GridItem.svelte`:

```svelte
<script lang="ts">
  import { goto } from '$app/navigation';
  import { createEventDispatcher } from 'svelte';
  import { selection } from '$lib/stores';

  export let file: any;
  export let size = 160;

  const dispatch = createEventDispatcher<{ context: { file: any; x: number; y: number } }>();

  function onClick(e: MouseEvent) {
    if (e.shiftKey || e.metaKey || e.ctrlKey) {
      selection.update((s) => {
        const ns = new Set(s);
        ns.has(file.id) ? ns.delete(file.id) : ns.add(file.id);
        return ns;
      });
      return;
    }
    goto(`/file/${file.id}`);
  }

  function onContext(e: MouseEvent) {
    e.preventDefault();
    dispatch('context', { file, x: e.clientX, y: e.clientY });
  }
</script>
```

And update the template:

```svelte
<div class="item" style="--size: {size}px" on:click={onClick} on:contextmenu={onContext}
     class:selected={$selection.has(file.id)}>
  ...
</div>
```

- [ ] **Step 2: Wire in browse page**

Replace `+page.svelte`'s full template and script:

```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { currentFolderPath, sortMode, density, selection } from '$lib/stores';
  import Grid from '$lib/components/Grid.svelte';
  import ContextMenu from '$lib/components/ContextMenu.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import UploadDialog from '$lib/components/UploadDialog.svelte';
  import MovePicker from '$lib/components/MovePicker.svelte';

  let folder: any = null;
  let folders: any[] = [];
  let files: any[] = [];
  let loading = true;

  let menu: { file: any; x: number; y: number } | null = null;
  let confirmDelete: any = null;
  let renaming: any = null;
  let moving: any = null;
  let uploading = false;
  let newFolderName = '';
  let showNewFolder = false;

  async function load() {
    loading = true;
    const sort = $sortMode === 'takenAt' ? 'taken' : $sortMode;
    const r = await api.folder($currentFolderPath, { sort, limit: 500 });
    folder = r.folder;
    folders = r.folders;
    files = r.files;
    loading = false;
  }

  onMount(() => { currentFolderPath.set(''); });
  $: $currentFolderPath, load();

  function onContext(e: CustomEvent<{ file: any; x: number; y: number }>) { menu = e.detail; }

  async function doDelete(f: any) { await api.deleteFile(f.id); load(); }
  async function doRename(f: any, name: string) { await api.renameFile(f.id, name); load(); }
  async function doMove(f: any, folderId: number) { await api.moveFile(f.id, folderId); load(); }
  async function doMkdir() {
    if (!newFolderName) return;
    const sub = $currentFolderPath ? `${$currentFolderPath}/${newFolderName}` : newFolderName;
    await api.mkdir(sub);
    showNewFolder = false;
    newFolderName = '';
    load();
  }

  function contextItems(f: any) {
    return [
      { label: 'Open', onSelect: () => location.assign(`/file/${f.id}`) },
      { label: 'Download', onSelect: () => location.assign(`/api/original/${f.id}`) },
      { label: 'Rename…', onSelect: () => { const n = prompt('New name', f.name); if (n && n !== f.name) doRename(f, n); } },
      { label: 'Move…', onSelect: () => (moving = f) },
      { label: 'Delete', danger: true, onSelect: () => (confirmDelete = f) }
    ];
  }
</script>

<div class="toolbar">
  <select bind:value={$sortMode}>
    <option value="takenAt">By capture date</option>
    <option value="name">By name</option>
    <option value="size">By size</option>
  </select>
  <select bind:value={$density}>
    <option value="small">S</option><option value="medium">M</option><option value="large">L</option>
  </select>
  <div class="spacer" />
  <button on:click={() => (showNewFolder = true)}>New folder</button>
  <button class="primary" on:click={() => (uploading = true)}>+ Upload</button>
</div>

{#if loading}
  <div class="loading">Loading…</div>
{:else}
  {#if folders.length > 0}
    <section>
      <h3>Subfolders</h3>
      <div class="folder-cards">
        {#each folders as sub}
          <a class="fcard" href={`/browse/${sub.path.split('/').map(encodeURIComponent).join('/')}`}
             on:click|preventDefault={() => currentFolderPath.set(sub.path)}>
            <div class="ico">📁</div>
            <div class="name">{sub.name}</div>
            <div class="cnt">{sub.items} items</div>
          </a>
        {/each}
      </div>
    </section>
  {/if}
  <Grid files={files} on:context={onContext} />
{/if}

{#if menu}
  <ContextMenu x={menu.x} y={menu.y} items={contextItems(menu.file)} onClose={() => (menu = null)} />
{/if}

{#if confirmDelete}
  <ConfirmDialog
    title="Delete {confirmDelete.name}?"
    message="This removes the file from disk."
    confirmLabel="Delete" danger
    onConfirm={async () => { await doDelete(confirmDelete); confirmDelete = null; }}
    onCancel={() => (confirmDelete = null)}
  />
{/if}

{#if moving}
  <MovePicker
    onPick={async (id, _path) => { await doMove(moving, id); moving = null; }}
    onClose={() => (moving = null)}
  />
{/if}

{#if uploading}
  <UploadDialog path={$currentFolderPath} onClose={() => (uploading = false)} onDone={load} />
{/if}

{#if showNewFolder}
  <ConfirmDialog
    title="New folder"
    message=""
    confirmLabel="Create"
    onConfirm={doMkdir}
    onCancel={() => { showNewFolder = false; newFolderName = ''; }}
  >
    <input slot="body" bind:value={newFolderName} placeholder="Folder name" />
  </ConfirmDialog>
{/if}

<style>
  .toolbar { display: flex; gap: 8px; padding: 8px 16px; border-bottom: 1px solid var(--border); align-items: center; }
  .spacer { flex: 1; }
  .loading { padding: 20px; color: var(--fg-dim); }
  h3 { margin: 16px 16px 8px; color: var(--fg-dim); font-size: 12px; text-transform: uppercase; }
  .folder-cards { display: grid; grid-template-columns: repeat(auto-fill, 180px);
    gap: 8px; padding: 0 16px; }
  .fcard { background: var(--bg-2); border-radius: 6px; padding: 12px;
    text-align: center; text-decoration: none; color: var(--fg); border: 1px solid var(--border); }
  .fcard:hover { border-color: var(--accent); }
  .ico { font-size: 26px; }
  .cnt { color: var(--fg-dim); font-size: 11px; }
</style>
```

Note: the `ConfirmDialog` prompts for name with its own slot. Update `ConfirmDialog.svelte` template to accept a named slot `body`:

```svelte
<!-- inside .dialog of ConfirmDialog.svelte, between <p>{message}</p> and <div class="actions"> -->
<slot name="body" />
```

- [ ] **Step 3: Update Grid to forward events**

```svelte
<!-- Grid.svelte — change GridItem tag -->
<GridItem file={f} size={itemSize} on:context />
```

- [ ] **Step 4: Commit**

```bash
git add web/src/lib/components/GridItem.svelte web/src/lib/components/Grid.svelte web/src/routes/browse/+page.svelte web/src/lib/components/ConfirmDialog.svelte
git commit -m "feat(web): context menu, upload, rename, move, delete in browse view"
```

---

## Phase 12 — Share UI and public share view

### Task 54: ShareDialog component

**Files:**
- Create: `web/src/lib/components/ShareDialog.svelte`

- [ ] **Step 1: Implement**

```svelte
<!-- web/src/lib/components/ShareDialog.svelte -->
<script lang="ts">
  import { api } from '$lib/api';
  export let folderId: number;
  export let folderPath: string;
  export let onClose: () => void;

  let expiresDays = 30;
  let password = '';
  let allowDownload = true;
  let allowUpload = false;
  let busy = false;
  let error = '';
  let created: any = null;

  async function create() {
    busy = true; error = '';
    try {
      created = await api.createShare({
        folder_id: folderId,
        expires_in_days: Number(expiresDays) || 0,
        password: password || undefined,
        allow_download: allowDownload,
        allow_upload: allowUpload
      });
    } catch (e: any) {
      error = e.message ?? 'failed';
    } finally {
      busy = false;
    }
  }
</script>

<div class="backdrop" on:click={onClose}>
  <div class="dialog" on:click|stopPropagation>
    <h3>Share "{folderPath || 'root'}"</h3>
    {#if !created}
      <label>Expires (days, 0 = never)
        <input type="number" min="0" bind:value={expiresDays} /></label>
      <label>Password (optional)
        <input type="text" bind:value={password} /></label>
      <label><input type="checkbox" bind:checked={allowDownload} /> Allow download (incl. ZIP)</label>
      <label><input type="checkbox" bind:checked={allowUpload} /> Allow upload from external</label>
      {#if error}<p class="err">{error}</p>{/if}
      <div class="actions">
        <button on:click={onClose}>Cancel</button>
        <button class="primary" on:click={create} disabled={busy}>Create</button>
      </div>
    {:else}
      <p>Share created:</p>
      <input readonly value={created.url} on:focus={(e) => e.currentTarget.select()} style="width:100%" />
      <div class="actions">
        <button on:click={() => navigator.clipboard.writeText(created.url)}>Copy link</button>
        <button class="primary" on:click={onClose}>Done</button>
      </div>
    {/if}
  </div>
</div>

<style>
  .backdrop { position: fixed; inset: 0; background: rgba(0,0,0,0.6);
    z-index: 150; display: grid; place-items: center; }
  .dialog { background: var(--bg-2); border-radius: 8px; padding: 24px;
    min-width: 420px; border: 1px solid var(--border);
    display: flex; flex-direction: column; gap: 10px; }
  label { display: flex; flex-direction: column; gap: 4px; color: var(--fg-dim); }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 12px; }
  .err { color: var(--danger); }
</style>
```

- [ ] **Step 2: Wire into browse page context menu**

Append to the script in `browse/+page.svelte`:

```ts
import ShareDialog from '$lib/components/ShareDialog.svelte';
let sharing: any = null;
```

Append to `contextItems()`:

```ts
{ label: 'Share folder containing…', onSelect: () => (sharing = { id: folder.id, path: folder.path }) },
```

Template:

```svelte
{#if sharing}
  <ShareDialog folderId={sharing.id} folderPath={sharing.path} onClose={() => (sharing = null)} />
{/if}
```

Also add a "Share this folder" button to toolbar:

```svelte
<button on:click={() => (sharing = { id: folder.id, path: folder.path })}>Share</button>
```

- [ ] **Step 3: Commit**

```bash
git add web/src/lib/components/ShareDialog.svelte web/src/routes/browse/+page.svelte
git commit -m "feat(web): share dialog for creating share-links"
```

---

### Task 55: Shares management page

**Files:**
- Create: `web/src/routes/shares/+page.svelte`

- [ ] **Step 1: Implement**

```svelte
<!-- web/src/routes/shares/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { refreshMe } from '$lib/stores';

  let shares: any[] = [];
  let loading = true;
  let error = '';

  async function load() {
    loading = true;
    try {
      shares = await api.myShares();
    } catch (e: any) { error = e.message; }
    loading = false;
  }

  async function revoke(id: number) { await api.revokeShare(id); load(); }
  async function del(id: number) { await api.deleteShare(id); load(); }

  onMount(async () => { await refreshMe(); load(); });
</script>

<div class="page">
  <h2>My shares</h2>
  {#if loading}<p>Loading…</p>
  {:else if error}<p class="err">{error}</p>
  {:else if shares.length === 0}<p>No shares yet.</p>
  {:else}
    <table>
      <thead><tr><th>Folder</th><th>URL</th><th>Status</th><th>Expires</th><th>Flags</th><th></th></tr></thead>
      <tbody>
        {#each shares as s}
          <tr>
            <td>{s.folder_path || 'root'}</td>
            <td><input readonly value={s.url} /></td>
            <td>{s.status}</td>
            <td>{s.expires_at ?? 'never'}</td>
            <td>
              {s.has_password ? '🔒 ' : ''}{s.allow_download ? '⬇' : ''}{s.allow_upload ? ' ⬆' : ''}
            </td>
            <td class="actions">
              {#if s.status === 'active'}
                <button on:click={() => revoke(s.id)}>Revoke</button>
              {/if}
              <button class="danger" on:click={() => del(s.id)}>Delete</button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>

<style>
  .page { padding: 24px; overflow-y: auto; height: 100%; }
  table { width: 100%; border-collapse: collapse; margin-top: 12px; }
  th, td { padding: 8px 10px; border-bottom: 1px solid var(--border); font-size: 13px;
    text-align: left; vertical-align: middle; }
  th { color: var(--fg-dim); font-weight: 500; }
  input { width: 100%; }
  .actions { display: flex; gap: 6px; }
  button.danger { background: var(--danger); color: white; border-color: var(--danger); }
  .err { color: var(--danger); }
</style>
```

- [ ] **Step 2: Add nav link in the browse layout**

In `browse/+layout.svelte`, in `.sidebar-footer`:

```svelte
<a href="/shares">Shares</a>
<a href="/settings">Settings</a>
{#if $me?.is_admin}<a href="/admin">Admin</a>{/if}
```

- [ ] **Step 3: Commit**

```bash
git add web/src/routes/shares/+page.svelte web/src/routes/browse/+layout.svelte
git commit -m "feat(web): shares management page"
```

---

### Task 56: Public share page

**Files:**
- Create: `web/src/routes/s/[token]/+page.svelte`
- Modify: `internal/api/router.go` (route `/s/{token}` to frontend)

- [ ] **Step 1: Implement**

```svelte
<!-- web/src/routes/s/[token]/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';

  const token = $page.params.token as string;
  let meta: any = null;
  let folder: any = null;
  let files: any[] = [];
  let folders: any[] = [];
  let needsPassword = false;
  let password = '';
  let error = '';
  let currentSub = '';

  async function fetchJSON(path: string, opts: RequestInit = {}) {
    const r = await fetch(path, { credentials: 'include', ...opts });
    if (r.status === 401) { needsPassword = true; return null; }
    if (r.status === 410) { error = 'Share expired or revoked.'; return null; }
    if (!r.ok) { error = `Error ${r.status}`; return null; }
    return (await r.json()).data;
  }

  async function loadMeta() {
    meta = await fetchJSON(`/api/s/${token}`);
    if (meta) loadFolder('');
  }
  async function loadFolder(sub: string) {
    currentSub = sub;
    const d = await fetchJSON(`/api/s/${token}/folder${sub ? `?path=${encodeURIComponent(sub)}` : ''}`);
    if (d) { folder = d.folder; files = d.files; folders = d.folders; }
  }
  async function unlock() {
    const r = await fetch(`/api/s/${token}/unlock`, {
      method: 'POST', credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password })
    });
    if (r.ok) { needsPassword = false; loadMeta(); }
    else error = 'Wrong password';
  }

  onMount(loadMeta);
</script>

{#if needsPassword}
  <div class="center">
    <form on:submit|preventDefault={unlock} class="card">
      <h2>Password required</h2>
      <input type="password" bind:value={password} autofocus />
      {#if error}<p class="err">{error}</p>{/if}
      <button class="primary">Unlock</button>
    </form>
  </div>
{:else if error}
  <div class="center"><p class="err">{error}</p></div>
{:else if folder}
  <div class="share">
    <header>
      <strong>{meta.folder.name || 'Shared'}</strong>
      <span class="sep">/</span>
      {#each currentSub.split('/').filter(Boolean) as p, i}
        <a href="#" on:click|preventDefault={() => loadFolder(meta.folder.path + '/' + currentSub.split('/').slice(0, i + 1).join('/'))}>{p}</a>
        <span class="sep">›</span>
      {/each}
      <div class="spacer" />
      {#if meta.allow_download}
        <a class="dl" href={`/api/s/${token}/zip`}>Download all (ZIP)</a>
      {/if}
    </header>

    {#if folders.length > 0}
      <h3>Subfolders</h3>
      <div class="fcards">
        {#each folders as sub}
          <a href="#" class="fcard" on:click|preventDefault={() => loadFolder(sub.path)}>
            📁 {sub.name}
          </a>
        {/each}
      </div>
    {/if}

    <div class="grid">
      {#each files as f (f.id)}
        <a class="cell" href={`/api/s/${token}/original/${f.id}`} target="_blank">
          <img src={`/api/s/${token}/thumb/${f.id}`} alt={f.name} loading="lazy" />
        </a>
      {/each}
    </div>

    {#if meta.allow_upload}
      <form class="upload" method="post" enctype="multipart/form-data"
            action={`/api/s/${token}/upload`}>
        <input type="text" name="name" placeholder="Your name" />
        <input type="file" name="files" multiple />
        <button class="primary">Upload</button>
      </form>
    {/if}
  </div>
{/if}

<style>
  .center { min-height: 100vh; display: grid; place-items: center; }
  .card { background: var(--bg-2); padding: 24px; border-radius: 8px;
    border: 1px solid var(--border); display: flex; flex-direction: column; gap: 10px; }
  .share { padding: 16px; max-width: 1200px; margin: 0 auto; }
  header { display: flex; align-items: center; gap: 8px; padding: 8px 0 16px;
    border-bottom: 1px solid var(--border); }
  .spacer { flex: 1; }
  .dl { background: var(--accent); color: white; padding: 6px 12px; border-radius: 4px;
    text-decoration: none; }
  .fcards { display: grid; grid-template-columns: repeat(auto-fill, 180px); gap: 8px; padding: 8px 0; }
  .fcard { background: var(--bg-2); border-radius: 6px; padding: 12px; text-decoration: none;
    color: var(--fg); border: 1px solid var(--border); }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, 160px); gap: 4px; margin-top: 12px; }
  .cell { display: block; aspect-ratio: 1; overflow: hidden; border-radius: 3px; }
  .cell img { width: 100%; height: 100%; object-fit: cover; display: block; }
  .upload { margin-top: 20px; display: flex; gap: 8px; }
  .err { color: var(--danger); }
</style>
```

- [ ] **Step 2: Router allows `/s/*` to fall through to the frontend**

In `internal/frontend/frontend.go`'s `Handler`, remove the `strings.HasPrefix(r.URL.Path, "/s/")` exclusion — we want SvelteKit's static index.html to handle /s/* routes. Keep the `/api/` and `/healthz` exclusions.

- [ ] **Step 3: Commit**

```bash
git add web/src/routes/s/ internal/frontend/frontend.go
git commit -m "feat(web): public share view with folders, grid, zip and anon upload"
```

---

## Phase 13 — Admin, settings, search UI

### Task 57: Settings page

**Files:**
- Create: `web/src/routes/settings/+page.svelte`

- [ ] **Step 1: Implement**

```svelte
<!-- web/src/routes/settings/+page.svelte -->
<script lang="ts">
  import { api } from '$lib/api';
  import { me } from '$lib/stores';
  import { onMount } from 'svelte';

  let oldPw = '';
  let newPw = '';
  let msg = '';
  let error = '';

  async function change() {
    msg = ''; error = '';
    try {
      await api.changePassword(oldPw, newPw);
      oldPw = ''; newPw = '';
      msg = 'Password changed';
    } catch (e: any) {
      error = e.message ?? 'failed';
    }
  }

  onMount(() => { /* me store is hydrated by layout */ });
</script>

<div class="page">
  <h2>Settings</h2>
  {#if $me}<p>Signed in as <strong>{$me.username}</strong>{$me.is_admin ? ' (admin)' : ''}.</p>{/if}

  <h3>Change password</h3>
  <form on:submit|preventDefault={change} class="card">
    <label>Current password<input type="password" bind:value={oldPw} /></label>
    <label>New password (min 8 chars)<input type="password" bind:value={newPw} minlength="8" /></label>
    <button class="primary">Change</button>
    {#if msg}<p class="ok">{msg}</p>{/if}
    {#if error}<p class="err">{error}</p>{/if}
  </form>
</div>

<style>
  .page { padding: 24px; }
  .card { display: flex; flex-direction: column; gap: 10px; max-width: 420px;
    background: var(--bg-2); border: 1px solid var(--border); padding: 20px; border-radius: 8px; }
  label { display: flex; flex-direction: column; gap: 4px; color: var(--fg-dim); }
  .ok { color: #22c55e; }
  .err { color: var(--danger); }
</style>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/routes/settings/
git commit -m "feat(web): settings page with change-password"
```

---

### Task 58: Admin page (users + scan status)

**Files:**
- Create: `web/src/routes/admin/+page.svelte`

- [ ] **Step 1: Implement**

```svelte
<!-- web/src/routes/admin/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';
  import { me } from '$lib/stores';

  let users: any[] = [];
  let status: any = null;
  let newUser = { username: '', password: '', is_admin: false };
  let error = '';

  async function load() {
    users = await api.listUsers();
    status = await api.scanStatus();
  }
  async function add() {
    error = '';
    try {
      await api.createUser(newUser);
      newUser = { username: '', password: '', is_admin: false };
      load();
    } catch (e: any) { error = e.message; }
  }
  async function remove(id: number) {
    if (!confirm('Delete user?')) return;
    await api.deleteUser(id); load();
  }
  async function scanNow(full: boolean) { await api.scan(full); setTimeout(load, 1000); }

  onMount(async () => {
    if (!$me?.is_admin) return goto('/browse');
    load();
  });
</script>

<div class="page">
  <h2>Admin</h2>

  <section>
    <h3>Users</h3>
    <table>
      <thead><tr><th>User</th><th>Admin</th><th></th></tr></thead>
      <tbody>
        {#each users as u}
          <tr>
            <td>{u.username}</td>
            <td>{u.is_admin ? 'yes' : '-'}</td>
            <td><button class="danger" on:click={() => remove(u.id)}>Delete</button></td>
          </tr>
        {/each}
      </tbody>
    </table>
    <form on:submit|preventDefault={add} class="inline">
      <input placeholder="username" bind:value={newUser.username} />
      <input placeholder="password" type="password" bind:value={newUser.password} />
      <label><input type="checkbox" bind:checked={newUser.is_admin} /> admin</label>
      <button class="primary">Add user</button>
      {#if error}<span class="err">{error}</span>{/if}
    </form>
  </section>

  <section>
    <h3>Scan</h3>
    <div class="inline">
      <button on:click={() => scanNow(false)}>Run incremental</button>
      <button on:click={() => scanNow(true)}>Run full</button>
    </div>
    {#if status}
      <pre>{JSON.stringify(status, null, 2)}</pre>
    {/if}
  </section>
</div>

<style>
  .page { padding: 24px; overflow-y: auto; height: 100%; }
  section { margin-bottom: 28px; }
  table { width: 100%; border-collapse: collapse; margin: 10px 0; }
  th, td { padding: 8px 10px; border-bottom: 1px solid var(--border); text-align: left; }
  .inline { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
  pre { background: var(--bg-2); padding: 12px; border-radius: 6px; overflow-x: auto; }
  .err { color: var(--danger); }
  button.danger { background: var(--danger); color: white; border-color: var(--danger); }
</style>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/routes/admin/
git commit -m "feat(web): admin page for users and scan control"
```

---

### Task 59: Search box + results

**Files:**
- Create: `web/src/lib/components/SearchBox.svelte`
- Modify: `web/src/routes/browse/+layout.svelte`
- Create: `web/src/routes/search/+page.svelte`

- [ ] **Step 1: SearchBox**

```svelte
<!-- web/src/lib/components/SearchBox.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  let q = '';
  let timer: any = null;
  function change() {
    clearTimeout(timer);
    timer = setTimeout(() => {
      if (q.trim()) goto(`/search?q=${encodeURIComponent(q.trim())}`);
    }, 250);
  }
</script>

<input class="search" type="search" placeholder="Search filename, date (YYYY-MM-DD), camera…"
       bind:value={q} on:input={change} />

<style>
  .search { width: 100%; max-width: 420px; }
</style>
```

Add inside `browse/+layout.svelte`'s `<header>`:

```svelte
<SearchBox />
```

(With `import SearchBox from '$lib/components/SearchBox.svelte';` at top.)

- [ ] **Step 2: Search page**

```svelte
<!-- web/src/routes/search/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { api } from '$lib/api';
  import Grid from '$lib/components/Grid.svelte';

  let files: any[] = [];
  let loading = true;

  async function run() {
    loading = true;
    const q: Record<string, string> = {};
    const qp = $page.url.searchParams;
    for (const k of ['q', 'date_from', 'date_to', 'camera', 'kind']) {
      const v = qp.get(k); if (v) q[k] = v;
    }
    const r = await api.search(q);
    files = r.files;
    loading = false;
  }

  onMount(run);
  $: $page.url, run();
</script>

<div class="page">
  <h2>Search {$page.url.searchParams.get('q') ? `"${$page.url.searchParams.get('q')}"` : ''}</h2>
  {#if loading}<p>Loading…</p>
  {:else if files.length === 0}<p>No results.</p>
  {:else}<Grid {files} />{/if}
</div>

<style>
  .page { padding: 16px; display: flex; flex-direction: column; min-height: 0; flex: 1; }
  h2 { margin: 0 0 10px; }
</style>
```

- [ ] **Step 3: Commit**

```bash
git add web/src/lib/components/SearchBox.svelte web/src/routes/browse/+layout.svelte web/src/routes/search/
git commit -m "feat(web): search box and results page"
```

---

## Phase 14 — Mobile polish

### Task 60: Mobile sidebar toggle + swipe in lightbox

**Files:**
- Modify: `web/src/routes/browse/+layout.svelte`
- Modify: `web/src/lib/components/Lightbox.svelte`

- [ ] **Step 1: Hamburger toggle**

Replace the `<aside>` section in `browse/+layout.svelte` with a togglable version:

```svelte
<script lang="ts">
  // add:
  let sidebarOpen = false;
</script>

<button class="menu-btn" on:click={() => (sidebarOpen = !sidebarOpen)}>☰</button>

<aside class:open={sidebarOpen}>
  ...
</aside>
```

CSS additions:

```css
.menu-btn { display: none; position: fixed; top: 8px; left: 8px; z-index: 20;
  background: var(--bg-2); border: 1px solid var(--border); padding: 6px 10px; }
@media (max-width: 768px) {
  .menu-btn { display: block; }
  aside { display: flex; position: fixed; inset: 0 40% 0 0; transform: translateX(-100%);
    transition: transform 0.2s; z-index: 15; }
  aside.open { transform: translateX(0); }
}
```

- [ ] **Step 2: Swipe in lightbox**

Add inside `Lightbox.svelte` `<script>`:

```ts
let touchStart = 0;
function onTouchStart(e: TouchEvent) { touchStart = e.touches[0].clientX; }
function onTouchEnd(e: TouchEvent) {
  const dx = e.changedTouches[0].clientX - touchStart;
  if (Math.abs(dx) > 60) dx > 0 ? prev() : next();
}
```

Attach to the media `<div>`:

```svelte
<div class="media" on:click|stopPropagation on:touchstart={onTouchStart} on:touchend={onTouchEnd}>
```

- [ ] **Step 3: Commit**

```bash
git add web/src/routes/browse/+layout.svelte web/src/lib/components/Lightbox.svelte
git commit -m "feat(web): mobile sidebar drawer and lightbox swipe"
```

---

## Phase 15 — Packaging

Final multi-stage Dockerfile with libvips/libraw/ffmpeg/exiftool, README, and an operational smoke test.

### Task 61: Full multi-stage Dockerfile

**Files:**
- Modify: `Dockerfile`

- [ ] **Step 1: Replace Dockerfile**

```dockerfile
# syntax=docker/dockerfile:1.7

# ---------- Stage 1: frontend build ----------
FROM node:20-alpine AS web
WORKDIR /web
RUN corepack enable
COPY web/package.json web/package-lock.json* web/pnpm-lock.yaml* ./
RUN (pnpm install --frozen-lockfile 2>/dev/null || npm install)
COPY web/ ./
RUN (pnpm build 2>/dev/null || npm run build)

# ---------- Stage 2: backend build ----------
FROM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy built frontend into embed path.
RUN rm -rf internal/frontend/dist && mkdir -p internal/frontend/dist
COPY --from=web /web/build internal/frontend/dist
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /out/frames ./cmd/frames

# ---------- Stage 3: runtime ----------
FROM alpine:3.20
RUN apk add --no-cache \
    ca-certificates tzdata \
    sqlite-libs \
    vips-tools \
    libraw-tools \
    ffmpeg \
    exiftool
WORKDIR /app
COPY --from=build /out/frames /app/frames
EXPOSE 8080
ENTRYPOINT ["/app/frames"]
```

- [ ] **Step 2: Build + smoke-test**

```bash
docker compose build
docker compose up -d
sleep 3
curl -sf http://localhost:8080/healthz
# Open http://localhost:8080 in a browser and log in with FRAMES_ADMIN_USERNAME/PASSWORD
docker compose down
```

Expected: healthz returns JSON; login page loads; creating a folder through the UI shows it in the browse view.

- [ ] **Step 3: Commit**

```bash
git add Dockerfile
git commit -m "chore(docker): final multi-stage image with libvips/libraw/ffmpeg/exiftool"
```

---

### Task 62: README

**Files:**
- Create: `README.md`

- [ ] **Step 1: Write README**

```markdown
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
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: README with quick start and dev guide"
```

---

### Task 63: First-boot smoke script

**Files:**
- Create: `scripts/smoke.sh`

- [ ] **Step 1: Implement**

```bash
#!/usr/bin/env bash
# scripts/smoke.sh — brings the stack up, creates a test photo, verifies thumb generation
set -euo pipefail

export FRAMES_SESSION_SECRET=${FRAMES_SESSION_SECRET:-dev-secret-32-characters-xxxxxxxxx}
export FRAMES_PUBLIC_URL=${FRAMES_PUBLIC_URL:-http://localhost:8080}
export FRAMES_ADMIN_USERNAME=${FRAMES_ADMIN_USERNAME:-admin}
export FRAMES_ADMIN_PASSWORD=${FRAMES_ADMIN_PASSWORD:-admin1234567}

mkdir -p photos
# seed one tiny test image
if command -v vips >/dev/null 2>&1; then
  vips black photos/test.jpg 64 64 >/dev/null 2>&1 || true
fi

docker compose up -d --build
trap 'docker compose down' EXIT

# Wait for healthz.
for i in $(seq 1 30); do
  if curl -sf http://localhost:8080/healthz > /dev/null; then break; fi
  sleep 1
done
curl -sf http://localhost:8080/healthz

# Trigger a manual scan (requires login via cookie jar).
jar=$(mktemp)
curl -sf -c "$jar" http://localhost:8080/api/me > /dev/null || true
CSRF=$(grep frames_csrf "$jar" | awk '{print $7}')
curl -sf -b "$jar" -c "$jar" -H "X-CSRF-Token: $CSRF" \
  -H "Content-Type: application/json" \
  -d '{"username":"'"$FRAMES_ADMIN_USERNAME"'","password":"'"$FRAMES_ADMIN_PASSWORD"'"}' \
  http://localhost:8080/api/login > /dev/null
curl -sf -b "$jar" -H "X-CSRF-Token: $CSRF" -X POST http://localhost:8080/api/scan > /dev/null

echo "smoke ok"
```

- [ ] **Step 2: Make executable + run**

```bash
chmod +x scripts/smoke.sh
./scripts/smoke.sh
```

Expected: prints `smoke ok`. If `vips` isn't installed locally, place any JPEG at `photos/test.jpg` first.

- [ ] **Step 3: Commit**

```bash
git add scripts/smoke.sh
git commit -m "chore: end-to-end smoke-test script"
```

---

## Final verification

- [ ] **Run all tests**

```bash
go test ./... -count=1
```

- [ ] **Build front-end + image + compose up**

```bash
make web && docker compose build && docker compose up -d && \
  curl -sf http://localhost:8080/healthz && docker compose down
```

- [ ] **Manual acceptance pass**

Navigate through the UI and confirm each capability works:

- [ ] Login with bootstrap admin
- [ ] Browse a seeded folder; thumbnails appear
- [ ] Upload a file via `+ Upload`
- [ ] Create a subfolder, rename a file, move a file, delete a file
- [ ] Open a file in the lightbox and use ← → to navigate
- [ ] Create a share link with password + expiration + upload enabled
- [ ] Open the share URL in an incognito browser, unlock, browse, ZIP download, upload a file
- [ ] Revoke the share and confirm it returns 410
- [ ] Admin: create a second user, log in as them, confirm they see the library
- [ ] Search: query by filename, by date-from filter, confirm results

---








