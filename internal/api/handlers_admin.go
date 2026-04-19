// internal/api/handlers_admin.go
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/db"
)

type adminDeps struct {
	DB *db.DB
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
