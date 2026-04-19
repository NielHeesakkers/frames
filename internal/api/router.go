// internal/api/router.go
package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/db"
	"github.com/NielHeesakkers/frames/internal/fsops"
	"github.com/NielHeesakkers/frames/internal/scanner"
	"github.com/NielHeesakkers/frames/internal/thumbnail"
)

type Deps struct {
	Log       *slog.Logger
	DB        *db.DB
	Limiter   *auth.LoginLimiter
	Scheduler *scanner.Scheduler
	Cache     *thumbnail.Cache
	Queue     *thumbnail.Queue
	Pool      *thumbnail.Pool
	Ops       *fsops.Ops
	Root      string
	Secure    bool
	PublicURL string
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

			sd := &scanDeps{Scheduler: d.Scheduler}
			r.Post("/scan", sd.handleTrigger)

			mdx := &mediaDeps{DB: d.DB, Cache: d.Cache, Queue: d.Queue, Pool: d.Pool, Root: d.Root}
			r.Get("/thumb/{id}", mdx.handleThumb)
			r.Get("/preview/{id}", mdx.handlePreview)
			r.Get("/original/{id}", mdx.handleOriginal)

			bd := &browseDeps{DB: d.DB}
			r.Get("/folder", bd.handleFolder)
			r.Get("/folder/*", bd.handleFolder)
			r.Get("/tree", bd.handleTree)

			srd := &searchDeps{DB: d.DB}
			r.Get("/search", srd.handleSearch)

			sh := &sharesDeps{DB: d.DB, PublicURL: d.PublicURL}
			r.Post("/folder_shares", sh.handleAddFolderShare)
			r.Delete("/folder_shares", sh.handleRemoveFolderShare)
			r.Get("/shared_with_me", sh.handleMySharedFolders)

			od := &opsDeps{Ops: d.Ops}
			r.Post("/ops/mkdir", od.handleMkdir)
			r.Post("/ops/file/rename", od.handleRenameFile)
			r.Post("/ops/file/move", od.handleMoveFile)
			r.Post("/ops/file/delete", od.handleDeleteFile)
			r.Post("/ops/folder/rename", od.handleRenameFolder)
			r.Post("/ops/folder/delete", od.handleDeleteFolder)
		})
	})

	return r
}
