// internal/api/handlers_meta.go
package api

import (
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

// handleLatest returns the most recently added files and folders by insertion order.
// Used by the home view to show "Latest additions" above the folder grid.
func (md *metaDeps) handleLatest(w http.ResponseWriter, r *http.Request) {
	filesLimit := clampInt(r.URL.Query().Get("files"), 10, 1, 50)
	foldersLimit := clampInt(r.URL.Query().Get("folders"), 10, 1, 50)

	filesOut := make([]latestFileDTO, 0, filesLimit)
	rows, err := md.DB.Query(`
		SELECT id, filename, kind, mime_type, folder_id, relative_path, taken_at
		FROM files
		WHERE kind IN ('image','raw','video')
		ORDER BY id DESC
		LIMIT ?
	`, filesLimit)
	if err == nil {
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
