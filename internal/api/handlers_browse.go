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
	ID       int64  `json:"id"`
	ParentID *int64 `json:"parent_id,omitempty"`
	Path     string `json:"path"`
	Name     string `json:"name"`
	Items    int64  `json:"items"`
}

type fileDTO struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Size          int64   `json:"size"`
	Kind          string  `json:"kind"`
	MimeType      string  `json:"mime_type"`
	Mtime         int64   `json:"mtime"`
	TakenAt       *string `json:"taken_at,omitempty"`
	Width         *int    `json:"width,omitempty"`
	Height        *int    `json:"height,omitempty"`
	ThumbStatus   string  `json:"thumb_status"`
	PreviewStatus string  `json:"preview_status"`
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
