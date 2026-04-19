// internal/api/handlers_share_public.go
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

// Public share access is NOT wrapped in RequireLogin or CSRF (it's an unauthenticated
// public surface). Instead each handler validates the share status, optional password
// cookie, and scope.

type publicShareDeps struct {
	DB      *db.DB
	Cache   thumbnailCache // interface to avoid import cycle
	Root    string
	Limiter *share.RateLimiter
}

// thumbnailCache is a minimal interface matching *thumbnail.Cache.
type thumbnailCache interface {
	ThumbPath(id int64) string
	PreviewPath(id int64) string
}

func (psh *publicShareDeps) load(r *http.Request) (*db.Share, int) {
	tok := chi.URLParam(r, "token")
	s, err := psh.DB.ShareByToken(tok)
	if err != nil {
		return nil, http.StatusNotFound
	}
	switch share.Validate(s) {
	case share.StatusExpired, share.StatusRevoked:
		return nil, http.StatusGone
	}
	if s.PasswordHash != nil {
		c, _ := r.Cookie(shareCookieName(tok))
		if c == nil {
			return nil, http.StatusUnauthorized
		}
		ok, _ := auth.VerifyPassword(*s.PasswordHash, c.Value)
		if !ok {
			return nil, http.StatusUnauthorized
		}
	}
	if !psh.Limiter.Allow(tok) {
		return nil, http.StatusTooManyRequests
	}
	return s, 0
}

func shareCookieName(tok string) string { return "frames_share_" + tok }

type unlockReq struct {
	Password string `json:"password"`
}

func (psh *publicShareDeps) handleUnlock(w http.ResponseWriter, r *http.Request) {
	tok := chi.URLParam(r, "token")
	s, err := psh.DB.ShareByToken(tok)
	if err != nil || share.Validate(s) != share.StatusActive {
		WriteError(w, http.StatusNotFound, "invalid share")
		return
	}
	var req unlockReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if s.PasswordHash == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	ok, _ := auth.VerifyPassword(*s.PasswordHash, req.Password)
	if !ok {
		WriteError(w, http.StatusUnauthorized, "wrong password")
		return
	}
	// Set a cookie scoped to this share carrying the plaintext password.
	// (Acceptable here because the attacker who can read it can already use the share URL.)
	http.SetCookie(w, &http.Cookie{
		Name: shareCookieName(tok), Value: req.Password,
		Path: "/s/" + tok, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (psh *publicShareDeps) handleMeta(w http.ResponseWriter, r *http.Request) {
	s, code := psh.load(r)
	if code != 0 {
		WriteError(w, code, http.StatusText(code))
		return
	}
	f, _ := psh.DB.FolderByID(s.FolderID)
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"folder": map[string]any{
				"id": f.ID, "path": f.Path, "name": f.Name, "items": f.ItemCount,
			},
			"allow_download": s.AllowDownload,
			"allow_upload":   s.AllowUpload,
			"has_password":   s.PasswordHash != nil,
			"expires_at":     s.ExpiresAt,
		},
	})
}

func (psh *publicShareDeps) handleListFolder(w http.ResponseWriter, r *http.Request) {
	s, code := psh.load(r)
	if code != 0 {
		WriteError(w, code, http.StatusText(code))
		return
	}
	sub := r.URL.Query().Get("path")
	folder, err := psh.DB.FolderByID(s.FolderID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "folder missing")
		return
	}
	target := folder
	if sub != "" {
		cand, err := psh.DB.FolderByPath(sub)
		if err != nil {
			WriteError(w, http.StatusNotFound, "folder missing")
			return
		}
		if !share.IsUnderFolder(folder.Path, cand.Path) {
			WriteError(w, http.StatusForbidden, "out of share scope")
			return
		}
		target = cand
	}
	children, _ := psh.DB.ChildFolders(target.ID)
	files, _ := psh.DB.FilesInFolder(target.ID, 500, 0, db.SortByTakenAt)

	foldersOut := make([]map[string]any, 0, len(children))
	for _, c := range children {
		foldersOut = append(foldersOut, map[string]any{
			"id": c.ID, "path": c.Path, "name": c.Name, "items": c.ItemCount,
		})
	}
	filesOut := make([]map[string]any, 0, len(files))
	for _, fl := range files {
		var taken *string
		if fl.TakenAt != nil {
			t := fl.TakenAt.Format(time.RFC3339)
			taken = &t
		}
		filesOut = append(filesOut, map[string]any{
			"id": fl.ID, "name": fl.Filename, "size": fl.Size, "kind": fl.Kind,
			"mime_type": fl.MimeType, "taken_at": taken,
			"width": fl.Width, "height": fl.Height,
		})
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"folder":  map[string]any{"id": target.ID, "path": target.Path, "name": target.Name},
			"folders": foldersOut,
			"files":   filesOut,
		},
	})
}

func (psh *publicShareDeps) handleFileMedia(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, code := psh.load(r)
		if code != 0 {
			WriteError(w, code, http.StatusText(code))
			return
		}
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "bad id")
			return
		}
		f, err := psh.DB.FileByID(id)
		if err != nil {
			WriteError(w, http.StatusNotFound, "not found")
			return
		}
		// Scope check.
		rootFolder, _ := psh.DB.FolderByID(s.FolderID)
		folderOfFile, _ := psh.DB.FolderByID(f.FolderID)
		if !share.IsUnderFolder(rootFolder.Path, folderOfFile.Path) {
			WriteError(w, http.StatusForbidden, "out of scope")
			return
		}
		switch kind {
		case "thumb":
			path := psh.Cache.ThumbPath(id)
			serveWithETag(w, r, path,
				"s-"+strconv.FormatInt(f.ID, 10)+"-"+strconv.FormatInt(f.Mtime, 10), f.MimeType)
		case "preview":
			path := psh.Cache.PreviewPath(id)
			serveWithETag(w, r, path,
				"sp-"+strconv.FormatInt(f.ID, 10)+"-"+strconv.FormatInt(f.Mtime, 10), "image/webp")
		case "original":
			if !s.AllowDownload {
				WriteError(w, http.StatusForbidden, "download disabled")
				return
			}
			// Delegate to the media serveOriginal-style logic.
			serveOriginalFile(w, r, psh.Root, f)
		}
	}
}
