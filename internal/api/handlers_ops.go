// internal/api/handlers_ops.go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/NielHeesakkers/frames/internal/fsops"
)

type opsDeps struct {
	Ops *fsops.Ops
}

type mkdirReq struct {
	Path string `json:"path"`
}
type renameReq struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type moveReq struct {
	ID          int64 `json:"id"`
	NewFolderID int64 `json:"new_folder_id"`
}
type deleteReq struct {
	ID int64 `json:"id"`
}

func (od *opsDeps) handleMkdir(w http.ResponseWriter, r *http.Request) {
	var req mkdirReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := od.Ops.Mkdir(req.Path); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (od *opsDeps) handleRenameFile(w http.ResponseWriter, r *http.Request) {
	var req renameReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := od.Ops.RenameFile(req.ID, req.Name); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (od *opsDeps) handleMoveFile(w http.ResponseWriter, r *http.Request) {
	var req moveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := od.Ops.MoveFile(req.ID, req.NewFolderID); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (od *opsDeps) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	var req deleteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := od.Ops.DeleteFile(req.ID); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (od *opsDeps) handleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	var req deleteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := od.Ops.DeleteFolder(req.ID); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (od *opsDeps) handleRenameFolder(w http.ResponseWriter, r *http.Request) {
	var req renameReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := od.Ops.RenameFolder(req.ID, req.Name); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
