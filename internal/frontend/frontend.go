// internal/frontend/frontend.go
package frontend

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

//go:embed all:dist
var dist embed.FS

// FS returns an http.FileSystem for the frontend assets. Precedence:
//  1. FRAMES_FRONTEND_DIR env var — if set and the directory exists, serve
//     from disk. This is the hot-reload path for development: just run
//     `npm run build` (or bind-mount the build dir) and changes show up
//     on the next request without rebuilding the Go binary.
//  2. The embedded dist FS (production default).
//  3. web/build on disk (local dev before first embed).
func FS() http.FileSystem {
	if dir := os.Getenv("FRAMES_FRONTEND_DIR"); dir != "" {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return http.Dir(dir)
		}
	}
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return http.Dir("web/build")
	}
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
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/healthz") {
			http.NotFound(w, r)
			return
		}
		if f, err := fsys.Open(r.URL.Path); err == nil {
			f.Close()
			fileSrv.ServeHTTP(w, r)
			return
		}
		r.URL.Path = "/"
		fileSrv.ServeHTTP(w, r)
	})
}

// IndexHTML returns the raw index.html used by the SPA. Useful for handlers
// that need to serve a modified copy (e.g. injecting OpenGraph meta tags for
// shared links) while still booting the same SvelteKit client bundle.
func IndexHTML() (string, error) {
	fsys := FS()
	f, err := fsys.Open("/index.html")
	if err != nil {
		return "", err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
