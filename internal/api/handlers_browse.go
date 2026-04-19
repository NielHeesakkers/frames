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

type treeNodeDTO struct {
	ID       int64  `json:"id"`
	Path     string `json:"path"`
	Name     string `json:"name"`
	HasChild bool   `json:"has_child"`
	Items    int64  `json:"items"`
}

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

	// One recursive CTE returns (ancestor_id, total_items) for every child of the parent.
	// ancestor_id is the direct child of parentID; descendant_id enumerates its entire
	// subtree (inclusive). We then join files by folder_id = descendant_id and count.
	totals := make(map[int64]int64, len(kids))
	hasSub := make(map[int64]bool, len(kids))
	rows, err := bd.DB.Query(`
		WITH RECURSIVE subtree(ancestor_id, descendant_id) AS (
			SELECT id, id FROM folders WHERE parent_id = ?
			UNION ALL
			SELECT s.ancestor_id, f.id FROM folders f JOIN subtree s ON f.parent_id = s.descendant_id
		)
		SELECT s.ancestor_id, COUNT(files.id) AS total, MAX(s.descendant_id != s.ancestor_id) AS has_sub
		FROM subtree s
		LEFT JOIN files ON files.folder_id = s.descendant_id
		GROUP BY s.ancestor_id
	`, parentID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, count int64
			var hs int
			if err := rows.Scan(&id, &count, &hs); err == nil {
				totals[id] = count
				hasSub[id] = hs == 1
			}
		}
	}

	out := make([]treeNodeDTO, 0, len(kids))
	for _, c := range kids {
		t, ok := totals[c.ID]
		if !ok {
			t = c.ItemCount
		}
		out = append(out, treeNodeDTO{
			ID: c.ID, Path: c.Path, Name: c.Name,
			HasChild: hasSub[c.ID], Items: t,
		})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}
