// internal/api/handlers_search.go
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/NielHeesakkers/frames/internal/db"
)

type searchDeps struct {
	DB *db.DB
}

func (sd *searchDeps) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sq := db.SearchQuery{
		Query:  q.Get("q"),
		Camera: q.Get("camera"),
		Kind:   q.Get("kind"),
	}
	if s := q.Get("date_from"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err == nil {
			sq.DateFrom = &t
		}
	}
	if s := q.Get("date_to"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err == nil {
			// end-of-day
			end := t.Add(24*time.Hour - time.Second)
			sq.DateTo = &end
		}
	}
	sq.Limit, _ = strconv.Atoi(q.Get("limit"))
	sq.Offset, _ = strconv.Atoi(q.Get("offset"))

	files, err := sd.DB.SearchFiles(sq)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]fileDTO, 0, len(files))
	for _, fl := range files {
		var takenStr *string
		if fl.TakenAt != nil {
			s := fl.TakenAt.Format("2006-01-02T15:04:05")
			takenStr = &s
		}
		out = append(out, fileDTO{
			ID: fl.ID, Name: fl.Filename, Size: fl.Size, Kind: fl.Kind,
			MimeType: fl.MimeType, Mtime: fl.Mtime, TakenAt: takenStr,
			Width: fl.Width, Height: fl.Height,
			ThumbStatus: fl.ThumbStatus, PreviewStatus: fl.PreviewStatus,
			Rating: fl.Rating,
		})
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"files": out, "has_more": len(out) == sq.Limit},
	})
}
