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
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/healthz") {
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
