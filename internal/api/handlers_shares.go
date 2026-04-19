// internal/api/handlers_shares.go
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/db"
	"github.com/NielHeesakkers/frames/internal/share"
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

// ---- External share-link CRUD ----

type createShareReq struct {
	FolderID      int64  `json:"folder_id"`
	ExpiresInDays int    `json:"expires_in_days"` // 0 = never; positive => days
	Password      string `json:"password"`        // optional
	AllowDownload bool   `json:"allow_download"`
	AllowUpload   bool   `json:"allow_upload"`
}

type shareDTO struct {
	ID            int64   `json:"id"`
	Token         string  `json:"token"`
	FolderID      int64   `json:"folder_id"`
	FolderPath    string  `json:"folder_path"`
	ExpiresAt     *string `json:"expires_at,omitempty"`
	HasPassword   bool    `json:"has_password"`
	AllowDownload bool    `json:"allow_download"`
	AllowUpload   bool    `json:"allow_upload"`
	RevokedAt     *string `json:"revoked_at,omitempty"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
	CreatedBy     int64   `json:"created_by"`
	URL           string  `json:"url"`
}

func (sh *sharesDeps) toDTO(s db.Share, publicURL string) shareDTO {
	f, _ := sh.DB.FolderByID(s.FolderID)
	var folderPath string
	if f != nil {
		folderPath = f.Path
	}
	d := shareDTO{
		ID: s.ID, Token: s.Token, FolderID: s.FolderID, FolderPath: folderPath,
		HasPassword:   s.PasswordHash != nil,
		AllowDownload: s.AllowDownload, AllowUpload: s.AllowUpload,
		CreatedBy: s.CreatedBy,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
		URL:       publicURL + "/s/" + s.Token,
	}
	if s.ExpiresAt != nil {
		e := s.ExpiresAt.Format(time.RFC3339)
		d.ExpiresAt = &e
	}
	if s.RevokedAt != nil {
		e := s.RevokedAt.Format(time.RFC3339)
		d.RevokedAt = &e
	}
	switch share.Validate(&s) {
	case share.StatusActive:
		d.Status = "active"
	case share.StatusExpired:
		d.Status = "expired"
	case share.StatusRevoked:
		d.Status = "revoked"
	}
	return d
}

func (sh *sharesDeps) handleCreateShare(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	var req createShareReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if _, err := sh.DB.FolderByID(req.FolderID); err != nil {
		WriteError(w, http.StatusNotFound, "folder not found")
		return
	}
	tok, err := share.NewToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "token error")
		return
	}
	s := db.Share{
		Token: tok, FolderID: req.FolderID, CreatedBy: u.ID,
		AllowDownload: req.AllowDownload, AllowUpload: req.AllowUpload,
	}
	if req.ExpiresInDays > 0 {
		e := time.Now().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
		s.ExpiresAt = &e
	}
	if req.Password != "" {
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "hash error")
			return
		}
		s.PasswordHash = &hash
	}
	id, err := sh.DB.CreateShare(s)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	created, _ := sh.DB.ShareByID(id)
	WriteJSON(w, http.StatusOK, map[string]any{"data": sh.toDTO(*created, sh.PublicURL)})
}

func (sh *sharesDeps) handleListMyShares(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	shares, err := sh.DB.SharesByUser(u.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]shareDTO, 0, len(shares))
	for _, s := range shares {
		out = append(out, sh.toDTO(s, sh.PublicURL))
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (sh *sharesDeps) handleListAllShares(w http.ResponseWriter, r *http.Request) {
	shares, err := sh.DB.AllShares()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]shareDTO, 0, len(shares))
	for _, s := range shares {
		out = append(out, sh.toDTO(s, sh.PublicURL))
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (sh *sharesDeps) handleRevokeShare(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad id")
		return
	}
	u, _ := auth.UserFromContext(r.Context())
	s, err := sh.DB.ShareByID(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not found")
		return
	}
	if s.CreatedBy != u.ID && !u.IsAdmin {
		WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := sh.DB.RevokeShare(id); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (sh *sharesDeps) handleDeleteShare(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad id")
		return
	}
	u, _ := auth.UserFromContext(r.Context())
	s, err := sh.DB.ShareByID(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not found")
		return
	}
	if s.CreatedBy != u.ID && !u.IsAdmin {
		WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := sh.DB.DeleteShare(id); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
