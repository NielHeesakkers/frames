// internal/api/handlers_meta.go
package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/NielHeesakkers/frames/internal/db"
	"github.com/NielHeesakkers/frames/internal/version"
)

type metaDeps struct {
	DB *db.DB
}

// handleVersion returns the current version string + full changelog markdown.
func (md *metaDeps) handleVersion(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"version":   version.Current,
			"changelog": version.Changelog(),
		},
	})
}

type latestFileDTO struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Kind         string  `json:"kind"`
	MimeType     string  `json:"mime_type"`
	FolderID     int64   `json:"folder_id"`
	RelativePath string  `json:"relative_path"`
	TakenAt      *string `json:"taken_at,omitempty"`
}

type latestFolderDTO struct {
	ID    int64  `json:"id"`
	Path  string `json:"path"`
	Name  string `json:"name"`
	Items int64  `json:"items"`
}

// handleLatest returns the most recently added files and folders by insertion
// order. Optional `path` query parameter scopes results to that folder's entire
// subtree — useful when showing "Latest photos in this folder" for a container
// folder that has no direct files of its own.
func (md *metaDeps) handleLatest(w http.ResponseWriter, r *http.Request) {
	filesLimit := clampInt(r.URL.Query().Get("files"), 10, 1, 50)
	foldersLimit := clampInt(r.URL.Query().Get("folders"), 10, 0, 50)
	scopePath := r.URL.Query().Get("path")

	filesOut := make([]latestFileDTO, 0, filesLimit)

	var rows *sql.Rows
	var err error
	if scopePath == "" {
		rows, err = md.DB.Query(`
			SELECT id, filename, kind, mime_type, folder_id, relative_path, taken_at
			FROM files
			WHERE kind IN ('image','raw','video')
			ORDER BY id DESC
			LIMIT ?
		`, filesLimit)
	} else {
		folder, ferr := md.DB.FolderByPath(scopePath)
		if ferr != nil {
			WriteJSON(w, http.StatusOK, map[string]any{
				"data": map[string]any{"files": filesOut, "folders": []latestFolderDTO{}},
			})
			return
		}
		rows, err = md.DB.Query(`
			WITH RECURSIVE subtree(id) AS (
				SELECT ? UNION ALL
				SELECT f.id FROM folders f JOIN subtree s ON f.parent_id = s.id
			)
			SELECT id, filename, kind, mime_type, folder_id, relative_path, taken_at
			FROM files
			WHERE folder_id IN (SELECT id FROM subtree)
			  AND kind IN ('image','raw','video')
			ORDER BY id DESC
			LIMIT ?
		`, folder.ID, filesLimit)
	}
	if err == nil && rows != nil {
		defer rows.Close()
		for rows.Next() {
			var f latestFileDTO
			var taken *string
			if err := rows.Scan(&f.ID, &f.Name, &f.Kind, &f.MimeType, &f.FolderID, &f.RelativePath, &taken); err != nil {
				continue
			}
			f.TakenAt = taken
			filesOut = append(filesOut, f)
		}
	}

	foldersOut := make([]latestFolderDTO, 0, foldersLimit)
	if foldersLimit > 0 && scopePath == "" {
		frows, err := md.DB.Query(`
			SELECT id, path, name, item_count
			FROM folders
			WHERE path != ''
			ORDER BY id DESC
			LIMIT ?
		`, foldersLimit)
		if err == nil {
			defer frows.Close()
			for frows.Next() {
				var fl latestFolderDTO
				if err := frows.Scan(&fl.ID, &fl.Path, &fl.Name, &fl.Items); err != nil {
					continue
				}
				foldersOut = append(foldersOut, fl)
			}
		}
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"files":   filesOut,
			"folders": foldersOut,
		},
	})
}

func clampInt(s string, def, min, max int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
