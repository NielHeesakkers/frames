// internal/api/handlers_shares.go (stub; extended in Phase 7)
package api

import (
	"encoding/json"
	"net/http"

	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/db"
)

type sharesDeps struct {
	DB        *db.DB
	PublicURL string
}

type addFolderShareReq struct {
	FolderID int64 `json:"folder_id"`
	UserID   int64 `json:"user_id"`
}

func (sh *sharesDeps) handleAddFolderShare(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	var req addFolderShareReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := sh.DB.AddFolderShare(req.FolderID, req.UserID, u.ID); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (sh *sharesDeps) handleRemoveFolderShare(w http.ResponseWriter, r *http.Request) {
	var req addFolderShareReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := sh.DB.RemoveFolderShare(req.FolderID, req.UserID); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (sh *sharesDeps) handleMySharedFolders(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	folders, err := sh.DB.FoldersSharedWith(u.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]folderDTO, 0, len(folders))
	for _, f := range folders {
		out = append(out, folderDTO{ID: f.ID, ParentID: f.ParentID, Path: f.Path, Name: f.Name, Items: f.ItemCount})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}
