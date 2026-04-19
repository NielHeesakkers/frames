// internal/api/handlers_browse.go
package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/NielHeesakkers/frames/internal/db"
	"github.com/NielHeesakkers/frames/internal/thumbnail"
)

type browseDeps struct {
	DB   *db.DB
	Root string
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
	Rating        int     `json:"rating"`
}

func (bd *browseDeps) handleFolder(w http.ResponseWriter, r *http.Request) {
	// Path comes from the URL tail after /api/folder/
	pathParam := chi.URLParam(r, "*")
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 200
	}
	// Safety cap: even if a client asks for more, stop here. 50k covers
	// essentially any realistic single folder; for bigger folders, paginate.
	if limit > 50000 {
		limit = 50000
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	sort := db.SortByTakenAt
	switch q.Get("sort") {
	case "name":
		sort = db.SortByName
	case "rating":
		sort = db.SortByRating
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

	// Recursive item totals for each child folder so the "Submappen" cards
	// show the same count as the sidebar tree — users expect "Drone" inside
	// "2020 Schotland" to show its entire subtree count, not the direct
	// children only.
	totals := make(map[int64]int64, len(children))
	if rows, err := bd.DB.Query(`
		WITH RECURSIVE subtree(ancestor_id, descendant_id) AS (
			SELECT id, id FROM folders WHERE parent_id = ?
			UNION ALL
			SELECT s.ancestor_id, f.id FROM folders f JOIN subtree s ON f.parent_id = s.descendant_id
		)
		SELECT s.ancestor_id, COUNT(files.id)
		FROM subtree s
		LEFT JOIN files ON files.folder_id = s.descendant_id
		GROUP BY s.ancestor_id
	`, f.ID); err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, count int64
			if err := rows.Scan(&id, &count); err == nil {
				totals[id] = count
			}
		}
	}

	foldersOut := make([]folderDTO, 0, len(children))
	for _, c := range children {
		items := c.ItemCount
		if t, ok := totals[c.ID]; ok {
			items = t
		}
		foldersOut = append(foldersOut, folderDTO{
			ID: c.ID, ParentID: c.ParentID, Path: c.Path, Name: c.Name, Items: items,
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
			Rating: fl.Rating,
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
	// Read a richer EXIF summary on demand. Cheap (single exiftool call) and
	// avoids bloating the DB schema.
	var exif *thumbnail.DetailedEXIF
	if f.Kind == "image" || f.Kind == "raw" {
		abs := filepath.Join(bd.Root, f.RelativePath)
		if d, err := thumbnail.ReadDetailedEXIF(r.Context(), abs); err == nil {
			exif = &d
		}
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id": f.ID, "folder_id": f.FolderID, "name": f.Filename, "relative_path": f.RelativePath,
			"size": f.Size, "mtime": f.Mtime, "kind": f.Kind, "mime_type": f.MimeType,
			"taken_at": taken, "width": f.Width, "height": f.Height,
			"camera_make": f.CameraMake, "camera_model": f.CameraModel,
			"orientation": f.Orientation, "duration_ms": f.DurationMs,
			"thumb_status": f.ThumbStatus, "preview_status": f.PreviewStatus,
			"rating": f.Rating,
			"exif":   exif,
		},
	})
}

type setRatingReq struct{ Rating int `json:"rating"` }

func (bd *browseDeps) handleSetRating(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad id")
		return
	}
	var req setRatingReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := bd.DB.SetRating(id, req.Rating); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleFolderFiles returns every file under a folder's subtree (recursive)
// optionally filtered by a type label ("JPG", "ARW", "MOV"…). Unlike
// /api/folder (which only returns direct children) this one spans all
// descendant folders so clicking the "JPG · 1,110" pill in a container
// folder actually shows all 1,110 JPGs wherever they live.
func (bd *browseDeps) handleFolderFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	typeFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("type")))
	folder, err := bd.DB.FolderByPath(path)
	if err != nil {
		WriteError(w, http.StatusNotFound, "folder not found")
		return
	}
	// Pull the whole subtree once; filter in Go so the type matching matches
	// typeLabel() exactly (same alias handling as the pill counts).
	rows, err := bd.DB.Query(`
		WITH RECURSIVE subtree(id) AS (
			SELECT ? UNION ALL
			SELECT f.id FROM folders f JOIN subtree s ON f.parent_id = s.id
		)
		SELECT id, filename, size, kind, mime_type, mtime, taken_at,
		       width, height, thumb_status, preview_status, rating
		FROM files WHERE folder_id IN (SELECT id FROM subtree)
		ORDER BY COALESCE(taken_at, CAST(mtime AS TEXT)) DESC, id DESC
	`, folder.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := make([]fileDTO, 0, 256)
	for rows.Next() {
		var (
			id                                      int64
			name, kind, mime, thumbStatus, previewStatus string
			size, mtime                             int64
			takenAt                                 *string
			width, height                           *int
			rating                                  int
		)
		if err := rows.Scan(&id, &name, &size, &kind, &mime, &mtime, &takenAt,
			&width, &height, &thumbStatus, &previewStatus, &rating); err != nil {
			continue
		}
		if typeFilter != "" {
			label := typeLabel(name, mime, kind)
			if !strings.EqualFold(label, typeFilter) {
				continue
			}
		}
		out = append(out, fileDTO{
			ID: id, Name: name, Size: size, Kind: kind,
			MimeType: mime, Mtime: mtime, TakenAt: takenAt,
			Width: width, Height: height,
			ThumbStatus: thumbStatus, PreviewStatus: previewStatus,
			Rating: rating,
		})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"files": out}})
}

// handleFolderStats returns file-type counts for a folder subtree — groups
// by a derived label (JPG, HEIC, ARW, MOV, PDF, …) so the frontend can show
// "219 JPG · 5 MOV · 12 ARW" above a folder.
func (bd *browseDeps) handleFolderStats(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	folder, err := bd.DB.FolderByPath(path)
	if err != nil {
		WriteError(w, http.StatusNotFound, "folder not found")
		return
	}
	rows, err := bd.DB.Query(`
		WITH RECURSIVE subtree(id) AS (
			SELECT ? UNION ALL
			SELECT f.id FROM folders f JOIN subtree s ON f.parent_id = s.id
		)
		SELECT filename, mime_type, kind
		FROM files WHERE folder_id IN (SELECT id FROM subtree)
	`, folder.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	counts := map[string]int64{}
	for rows.Next() {
		var fn, mt, kind string
		if err := rows.Scan(&fn, &mt, &kind); err != nil {
			continue
		}
		label := typeLabel(fn, mt, kind)
		if label == "" {
			continue
		}
		counts[label]++
	}
	// Emit sorted by count desc for stable client ordering.
	type entry struct {
		Label string `json:"label"`
		Count int64  `json:"count"`
	}
	out := make([]entry, 0, len(counts))
	for k, v := range counts {
		out = append(out, entry{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Label < out[j].Label
	})
	WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

// typeLabel maps a file to a short, UI-friendly type label. Prefers the
// extension since that's what users actually recognise; falls back to the
// MIME type when the filename has none.
func typeLabel(filename, mime, kind string) string {
	lc := strings.ToLower(filename)
	if i := strings.LastIndex(lc, "."); i >= 0 && i < len(lc)-1 {
		ext := strings.ToUpper(lc[i+1:])
		// Normalise the common aliases.
		switch ext {
		case "JPEG":
			return "JPG"
		case "TIFF":
			return "TIF"
		}
		return ext
	}
	// No extension — use broad kind.
	switch kind {
	case "image":
		return "Afbeelding"
	case "video":
		return "Video"
	case "raw":
		return "RAW"
	}
	return "Overig"
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
