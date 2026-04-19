// internal/api/handlers_media.go
package api

import (
	"context"
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
	// Block up to 3s waiting for a render we kick off now. The generation
	// must outlive this request, so use context.Background() for the goroutine.
	go func() { _ = md.Pool.GeneratePreview(context.Background(), id) }()
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
