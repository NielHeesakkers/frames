// internal/api/handlers_media.go
package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
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

	previewInFlight sync.Map      // key: id (int64), value: struct{}{}
	previewSem      chan struct{} // buffered cap runtime.NumCPU()
}

// newMediaDeps constructs a mediaDeps with the preview concurrency semaphore
// initialized. Use this instead of a bare struct literal.
func newMediaDeps(d *db.DB, c *thumbnail.Cache, q *thumbnail.Queue, p *thumbnail.Pool, root string) *mediaDeps {
	return &mediaDeps{
		DB:         d,
		Cache:      c,
		Queue:      q,
		Pool:       p,
		Root:       root,
		previewSem: make(chan struct{}, runtime.NumCPU()),
	}
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
	// Block up to 3s waiting for a render we kick off now. The generation
	// must outlive this request, so use context.Background() for the goroutine.
	// Dedup in-flight requests and bound concurrent preview renders to NumCPU.
	if _, loaded := md.previewInFlight.LoadOrStore(id, struct{}{}); !loaded {
		go func() {
			defer md.previewInFlight.Delete(id)
			select {
			case md.previewSem <- struct{}{}:
				defer func() { <-md.previewSem }()
				_ = md.Pool.GeneratePreview(context.Background(), id)
			case <-time.After(30 * time.Second):
				// Skip if queue is saturated.
			}
		}()
	}
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
	// Short max-age + must-revalidate: SQLite can recycle file IDs after a
	// library reset, so `immutable` would keep stale thumbs in the browser
	// cache forever. ETag-based revalidation stays cheap (304 on hit).
	w.Header().Set("Cache-Control", "public, max-age=60, must-revalidate")
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

// serveOriginalFile streams the backing original file with proper headers. Shared
// between the authenticated media endpoint and the public share endpoint.
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
