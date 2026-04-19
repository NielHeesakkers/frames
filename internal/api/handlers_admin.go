// internal/api/handlers_admin.go
package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/db"
	"github.com/NielHeesakkers/frames/internal/thumbnail"
)

type adminDeps struct {
	DB    *db.DB
	Cache *thumbnail.Cache
}

type createUserReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

func (ad *adminDeps) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Username == "" || len(req.Password) < 8 {
		WriteError(w, http.StatusBadRequest, "bad creds")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "hash error")
		return
	}
	id, err := ad.DB.CreateUser(req.Username, hash, req.IsAdmin)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"id": id}})
}

func (ad *adminDeps) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := ad.DB.ListUsers()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		out = append(out, map[string]any{
			"id": u.ID, "username": u.Username, "is_admin": u.IsAdmin,
		})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (ad *adminDeps) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad id")
		return
	}
	u, _ := auth.UserFromContext(r.Context())
	if id == u.ID {
		WriteError(w, http.StatusBadRequest, "cannot delete self")
		return
	}
	if err := ad.DB.DeleteUser(id); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (ad *adminDeps) handleScanStatus(w http.ResponseWriter, r *http.Request) {
	last, _ := ad.DB.LastScanJob("incremental")
	lastFull, _ := ad.DB.LastScanJob("full")
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"last_incremental": last,
			"last_full":        lastFull,
		},
	})
}

type changePasswordReq struct {
	Old string `json:"old"`
	New string `json:"new"`
}

func (ad *adminDeps) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	var req changePasswordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(req.New) < 8 {
		WriteError(w, http.StatusBadRequest, "password too short")
		return
	}
	cur, err := ad.DB.UserByID(u.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	ok, _ := auth.VerifyPassword(cur.PasswordHash, req.Old)
	if !ok {
		WriteError(w, http.StatusUnauthorized, "wrong old password")
		return
	}
	hash, err := auth.HashPassword(req.New)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "hash error")
		return
	}
	if err := ad.DB.UpdateUserPassword(u.ID, hash); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleClearCache removes all generated thumbnails and previews from disk
// and resets the per-file cache status so the worker regenerates them on
// the next scan. Does not touch the folder/file index.
func (ad *adminDeps) handleClearCache(w http.ResponseWriter, r *http.Request) {
	removed := 0
	for _, sub := range []string{"thumb", "preview", "tmp"} {
		dir := filepath.Join(ad.Cache.Root, sub)
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				p := filepath.Join(dir, e.Name())
				if err := os.RemoveAll(p); err == nil {
					removed++
				}
			}
		}
	}
	// Reset DB status so the worker regenerates.
	if _, err := ad.DB.Exec(
		`UPDATE files SET thumb_status='pending', thumb_attempts=0,
		                  preview_status='pending', preview_attempts=0`); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"removed_entries": removed},
	})
}

// handleResetIndex wipes the folder/file index, cache files, and scan history
// so the next full scan rebuilds everything from scratch. Use after changing
// the photos root mount. Users, sessions, and account settings are kept.
func (ad *adminDeps) handleResetIndex(w http.ResponseWriter, r *http.Request) {
	// 1. Wipe derivatives on disk.
	for _, sub := range []string{"thumb", "preview", "tmp"} {
		dir := filepath.Join(ad.Cache.Root, sub)
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				_ = os.RemoveAll(filepath.Join(dir, e.Name()))
			}
		}
	}
	// 2. Wipe DB rows that reference filesystem state. Order matters — children
	// before parents so foreign keys stay consistent (cascades handle most, but
	// we're explicit here for clarity).
	stmts := []string{
		`DELETE FROM shares`,
		`DELETE FROM folder_shares`,
		`DELETE FROM files`,
		`DELETE FROM folders`,
		`DELETE FROM scan_jobs`,
	}
	for _, s := range stmts {
		if _, err := ad.DB.Exec(s); err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	// 3. VACUUM to reclaim disk space after the big delete.
	_, _ = ad.DB.Exec(`VACUUM`)
	w.WriteHeader(http.StatusNoContent)
}
