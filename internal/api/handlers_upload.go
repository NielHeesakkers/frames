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
