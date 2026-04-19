// internal/api/handlers_share_public.go
package api

import (
	"encoding/json"
	"html"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/db"
	"github.com/NielHeesakkers/frames/internal/frontend"
	"github.com/NielHeesakkers/frames/internal/scanner"
	"github.com/NielHeesakkers/frames/internal/share"
	"github.com/NielHeesakkers/frames/internal/thumbnail"
	"github.com/NielHeesakkers/frames/internal/upload"
)

// Public share access is NOT wrapped in RequireLogin or CSRF (it's an unauthenticated
// public surface). Instead each handler validates the share status, optional password
// cookie, and scope.

type publicShareDeps struct {
	DB        *db.DB
	Cache     thumbnailCache // interface to avoid import cycle
	Root      string
	Limiter   *share.RateLimiter
	Upload    *upload.Service
	Queue     *thumbnail.Queue
	MaxBytes  int64
	Log       *slog.Logger
	Secure    bool
	PublicURL string
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
	if !psh.Limiter.Allow("unlock:" + tok) {
		WriteError(w, http.StatusTooManyRequests, "too many attempts")
		return
	}
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
		Path: "/s/" + tok, HttpOnly: true, Secure: psh.Secure, SameSite: http.SameSiteLaxMode,
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
	var children []db.Folder
	var files []db.File
	if len(s.FileIDs) > 0 {
		// File-scoped share: no subfolder navigation; only the named files.
		for _, id := range s.FileIDs {
			f, err := psh.DB.FileByID(id)
			if err == nil && f != nil {
				files = append(files, *f)
			}
		}
	} else {
		children, _ = psh.DB.ChildFolders(target.ID)
		files, _ = psh.DB.FilesInFolder(target.ID, 50000, 0, db.SortByTakenAt)
	}

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
		// Scope check: when the share is file-scoped, only those explicit
		// file IDs are reachable; otherwise fall back to "under the folder
		// subtree".
		if len(s.FileIDs) > 0 {
			allowed := false
			for _, aid := range s.FileIDs {
				if aid == id {
					allowed = true
					break
				}
			}
			if !allowed {
				WriteError(w, http.StatusForbidden, "out of scope")
				return
			}
		} else {
			rootFolder, _ := psh.DB.FolderByID(s.FolderID)
			folderOfFile, _ := psh.DB.FolderByID(f.FolderID)
			if !share.IsUnderFolder(rootFolder.Path, folderOfFile.Path) {
				WriteError(w, http.StatusForbidden, "out of scope")
				return
			}
		}
		switch kind {
		case "thumb":
			path := psh.Cache.ThumbPath(id)
			// Thumbs in the cache are WebP — send image/webp, not the source
			// file's mime (would be video/quicktime for .MOV, which makes the
			// browser reject the response).
			serveWithETag(w, r, path,
				"s-"+strconv.FormatInt(f.ID, 10)+"-"+strconv.FormatInt(f.Mtime, 10), "image/webp")
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

func (psh *publicShareDeps) handleZip(w http.ResponseWriter, r *http.Request) {
	s, code := psh.load(r)
	if code != 0 {
		WriteError(w, code, http.StatusText(code))
		return
	}
	if !s.AllowDownload {
		WriteError(w, http.StatusForbidden, "download disabled")
		return
	}
	folder, _ := psh.DB.FolderByID(s.FolderID)
	w.Header().Set("Content-Type", "application/zip")
	name := folder.Name
	if name == "" {
		name = "frames"
	}
	if len(s.FileIDs) > 0 {
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.zip"`)
		if err := share.StreamFilesZip(w, psh.DB, psh.Root, s.FileIDs); err != nil {
			psh.Log.Warn("zip stream error", "err", err)
		}
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.zip"`)
	if err := share.StreamFolderZip(w, psh.DB, psh.Root, folder.Path); err != nil {
		psh.Log.Warn("zip stream error", "token", chi.URLParam(r, "token"), "err", err)
	}
}

func (psh *publicShareDeps) handleAnonymousUpload(w http.ResponseWriter, r *http.Request) {
	s, code := psh.load(r)
	if code != 0 {
		WriteError(w, code, http.StatusText(code))
		return
	}
	if !s.AllowUpload {
		WriteError(w, http.StatusForbidden, "upload disabled")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, psh.MaxBytes)
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid multipart")
		return
	}
	uploader := r.FormValue("name")
	if uploader == "" {
		uploader = "anonymous"
	}
	folder, _ := psh.DB.FolderByID(s.FolderID)
	targetFolder := filepath.Join(folder.Path, "Uploads", sanitizeName(uploader))
	// Ensure the target folder on disk + DB.
	if err := psh.ensureFolder(targetFolder); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		WriteError(w, http.StatusBadRequest, "no files")
		return
	}
	ids := make([]int64, 0, len(files))
	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		id, err := psh.Upload.StoreFile(targetFolder, sanitizeName(fh.Filename), f, scanner.Classify)
		f.Close()
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		psh.Queue.Push(id, thumbnail.PrioForeground)
		ids = append(ids, id)
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"ids": ids}})
}

func (psh *publicShareDeps) ensureFolder(rel string) error {
	abs, err := upload.SafeJoin(psh.Root, rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return err
	}
	// Walk up ensuring rows exist.
	parts := strings.Split(rel, "/")
	cur := ""
	var parentID *int64
	for _, p := range parts {
		if p == "" {
			continue
		}
		if cur != "" {
			cur = cur + "/" + p
		} else {
			cur = p
		}
		existing, err := psh.DB.FolderByPath(cur)
		if err == nil {
			parentID = &existing.ID
			continue
		}
		if err != db.ErrNotFound {
			return err
		}
		fi, _ := os.Stat(filepath.Join(psh.Root, cur))
		var mtime int64
		if fi != nil {
			mtime = fi.ModTime().Unix()
		}
		created, err := psh.DB.UpsertFolder(db.Folder{
			ParentID: parentID, Path: cur, Name: p, Mtime: mtime,
		})
		if err != nil {
			return err
		}
		parentID = &created.ID
	}
	return nil
}

// handleShareLanding serves the SvelteKit index.html with OpenGraph meta tags
// injected so Slack, WhatsApp, iMessage, Twitter, etc. show a rich preview
// (title + first photo) when the share URL is pasted. The SPA still boots
// client-side and the user sees the normal share page. For password-protected
// shares we deliberately DO NOT leak the first photo — the preview shows only
// a generic title.
func (psh *publicShareDeps) handleShareLanding(w http.ResponseWriter, r *http.Request) {
	idx, err := frontend.IndexHTML()
	if err != nil {
		// Fall back to the plain frontend handler; SPA will still work.
		frontend.Handler().ServeHTTP(w, r)
		return
	}

	tok := chi.URLParam(r, "token")
	s, sErr := psh.DB.ShareByToken(tok)
	valid := sErr == nil && share.Validate(s) == share.StatusActive

	title := "Gedeeld album"
	description := "Bekijk dit gedeelde album op Frames."
	var ogImage string

	if valid {
		if folder, err := psh.DB.FolderByID(s.FolderID); err == nil && folder != nil && folder.Name != "" {
			title = folder.Name
		}
		if s.PasswordHash != nil {
			description = "Dit album is met een wachtwoord beveiligd."
		} else if firstID := psh.pickCoverFileID(s); firstID > 0 && psh.PublicURL != "" {
			ogImage = strings.TrimRight(psh.PublicURL, "/") + "/api/s/" + tok + "/preview/" + strconv.FormatInt(firstID, 10)
		}
	}

	shareURL := ""
	if psh.PublicURL != "" {
		shareURL = strings.TrimRight(psh.PublicURL, "/") + "/s/" + tok
	}

	var og strings.Builder
	// Basic meta.
	og.WriteString(`<meta name="description" content="` + html.EscapeString(description) + `" />` + "\n")
	// OpenGraph.
	og.WriteString(`<meta property="og:type" content="website" />` + "\n")
	og.WriteString(`<meta property="og:site_name" content="Frames" />` + "\n")
	og.WriteString(`<meta property="og:title" content="` + html.EscapeString(title) + `" />` + "\n")
	og.WriteString(`<meta property="og:description" content="` + html.EscapeString(description) + `" />` + "\n")
	if shareURL != "" {
		og.WriteString(`<meta property="og:url" content="` + html.EscapeString(shareURL) + `" />` + "\n")
	}
	if ogImage != "" {
		og.WriteString(`<meta property="og:image" content="` + html.EscapeString(ogImage) + `" />` + "\n")
		og.WriteString(`<meta property="og:image:alt" content="` + html.EscapeString(title) + `" />` + "\n")
	}
	// Twitter.
	og.WriteString(`<meta name="twitter:card" content="` + func() string {
		if ogImage != "" {
			return "summary_large_image"
		}
		return "summary"
	}() + `" />` + "\n")
	og.WriteString(`<meta name="twitter:title" content="` + html.EscapeString(title) + `" />` + "\n")
	og.WriteString(`<meta name="twitter:description" content="` + html.EscapeString(description) + `" />` + "\n")
	if ogImage != "" {
		og.WriteString(`<meta name="twitter:image" content="` + html.EscapeString(ogImage) + `" />` + "\n")
	}

	// Inject just before </head>. Also override the default <title> so the
	// browser tab shows the album name before the SPA mounts.
	doc := strings.Replace(idx,
		"<title>Frames</title>",
		"<title>"+html.EscapeString(title)+" · Frames</title>",
		1)
	doc = strings.Replace(doc, "</head>", og.String()+"</head>", 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Keep caches short — share metadata can change (rename, revoke).
	w.Header().Set("Cache-Control", "no-store, must-revalidate")
	_, _ = w.Write([]byte(doc))
}

// pickCoverFileID returns the ID of the photo to use as OpenGraph image —
// the first file when the share is file-scoped, otherwise the earliest
// "taken_at" photo from the folder subtree. Returns 0 when no suitable
// image file exists.
func (psh *publicShareDeps) pickCoverFileID(s *db.Share) int64 {
	if len(s.FileIDs) > 0 {
		for _, id := range s.FileIDs {
			f, err := psh.DB.FileByID(id)
			if err == nil && f != nil && (f.Kind == "image" || f.Kind == "raw") {
				return f.ID
			}
		}
		// No image file — fall back to the first entry regardless.
		return s.FileIDs[0]
	}
	// Folder-scoped: pick earliest-taken image from subtree.
	row := psh.DB.QueryRow(`
		WITH RECURSIVE subtree(id) AS (
			SELECT ? UNION ALL
			SELECT f.id FROM folders f JOIN subtree s ON f.parent_id = s.id
		)
		SELECT files.id FROM files
		WHERE files.folder_id IN (SELECT id FROM subtree)
		  AND files.kind IN ('image','raw')
		ORDER BY COALESCE(files.taken_at, files.mtime) ASC, files.id ASC
		LIMIT 1
	`, s.FolderID)
	var id int64
	if err := row.Scan(&id); err != nil {
		return 0
	}
	return id
}

func sanitizeName(s string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", "..", "_", "\x00", "_")
	s = replacer.Replace(s)
	if len(s) > 64 {
		s = s[:64]
	}
	if s == "" || s == "." || s == ".." {
		s = "anonymous"
	}
	return s
}
