// internal/api/router.go
package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/db"
	"github.com/NielHeesakkers/frames/internal/frontend"
	"github.com/NielHeesakkers/frames/internal/fsops"
	"github.com/NielHeesakkers/frames/internal/scanner"
	"github.com/NielHeesakkers/frames/internal/share"
	"github.com/NielHeesakkers/frames/internal/thumbnail"
	"github.com/NielHeesakkers/frames/internal/upload"
)

type Deps struct {
	Log            *slog.Logger
	DB             *db.DB
	Limiter        *auth.LoginLimiter
	Scheduler      *scanner.Scheduler
	Cache          *thumbnail.Cache
	Queue          *thumbnail.Queue
	Pool           *thumbnail.Pool
	Ops            *fsops.Ops
	UploadSvc      *upload.Service
	MaxUpload      int64
	ShareUploadMax int64
	Root           string
	Secure         bool
	TrustProxy     bool
	PublicURL      string
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	ad := &AuthDeps{DB: d.DB, Limiter: d.Limiter, Secure: d.Secure, TrustProxy: d.TrustProxy}

	// CSRF applies to all /api routes. Login itself is unsafe but is only reachable
	// after a GET seeded the cookie; the frontend fetches /api/me (GET) first.
	r.Route("/api", func(r chi.Router) {
		r.Use(auth.CSRF)
		r.Post("/login", ad.handleLogin)
		r.Post("/logout", ad.handleLogout)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireLogin(d.DB))
			r.Get("/me", handleMe)

			sd := &scanDeps{Scheduler: d.Scheduler}
			r.Post("/scan", sd.handleTrigger)

			mdx := newMediaDeps(d.DB, d.Cache, d.Queue, d.Pool, d.Root)
			r.Get("/thumb/{id}", mdx.handleThumb)
			r.Get("/preview/{id}", mdx.handlePreview)
			r.Get("/original/{id}", mdx.handleOriginal)

			bd := &browseDeps{DB: d.DB, Root: d.Root}
			r.Get("/folder", bd.handleFolder)
			r.Get("/folder/*", bd.handleFolder)
			r.Get("/tree", bd.handleTree)
			r.Get("/file/{id}", bd.handleFile)

			srd := &searchDeps{DB: d.DB}
			r.Get("/search", srd.handleSearch)

			sh := &sharesDeps{DB: d.DB, PublicURL: d.PublicURL}
			r.Post("/folder_shares", sh.handleAddFolderShare)
			r.Delete("/folder_shares", sh.handleRemoveFolderShare)
			r.Get("/shared_with_me", sh.handleMySharedFolders)

			// External share-link CRUD.
			r.Post("/shares", sh.handleCreateShare)
			r.Get("/shares", sh.handleListMyShares)
			r.Delete("/shares/{id}/revoke", sh.handleRevokeShare)
			r.Delete("/shares/{id}", sh.handleDeleteShare)

			od := &opsDeps{Ops: d.Ops}
			r.Post("/ops/mkdir", od.handleMkdir)
			r.Post("/ops/file/rename", od.handleRenameFile)
			r.Post("/ops/file/move", od.handleMoveFile)
			r.Post("/ops/file/delete", od.handleDeleteFile)
			r.Post("/ops/folder/rename", od.handleRenameFolder)
			r.Post("/ops/folder/delete", od.handleDeleteFolder)

			ud := &uploadDeps{Svc: d.UploadSvc, Queue: d.Queue, MaxBytes: d.MaxUpload}
			r.Post("/upload", ud.handleUpload)

			ad := &adminDeps{DB: d.DB}
			r.Post("/account/password", ad.handleChangePassword)

			md := &metaDeps{DB: d.DB}
			r.Get("/version", md.handleVersion)
			r.Get("/latest", md.handleLatest)
		})

		// Admin-only routes (require login + admin).
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireLogin(d.DB), auth.RequireAdmin)
			sh := &sharesDeps{DB: d.DB, PublicURL: d.PublicURL}
			r.Get("/admin/shares", sh.handleListAllShares)

			ad := &adminDeps{DB: d.DB, Cache: d.Cache}
			r.Post("/admin/users", ad.handleCreateUser)
			r.Get("/admin/users", ad.handleListUsers)
			r.Delete("/admin/users/{id}", ad.handleDeleteUser)
			r.Get("/admin/scan_status", ad.handleScanStatus)
			r.Post("/admin/cache/clear", ad.handleClearCache)
			r.Post("/admin/index/reset", ad.handleResetIndex)
		})
	})

	// Public share-link surface (unauthenticated, no CSRF).
	psh := &publicShareDeps{
		DB:       d.DB,
		Cache:    d.Cache,
		Root:     d.Root,
		Limiter:  share.NewRateLimiter(100, time.Minute),
		Upload:   d.UploadSvc,
		Queue:    d.Queue,
		MaxBytes: d.ShareUploadMax,
		Log:      d.Log,
		Secure:   d.Secure,
	}
	r.Route("/api/s", func(r chi.Router) {
		r.Post("/{token}/unlock", psh.handleUnlock)
		r.Get("/{token}", psh.handleMeta)
		r.Get("/{token}/folder", psh.handleListFolder)
		r.Get("/{token}/thumb/{id}", psh.handleFileMedia("thumb"))
		r.Get("/{token}/preview/{id}", psh.handleFileMedia("preview"))
		r.Get("/{token}/original/{id}", psh.handleFileMedia("original"))
		r.Get("/{token}/zip", psh.handleZip)
		r.Post("/{token}/upload", psh.handleAnonymousUpload)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			WriteError(w, http.StatusNotFound, "not found")
			return
		}
		frontend.Handler().ServeHTTP(w, r)
	})

	return r
}
