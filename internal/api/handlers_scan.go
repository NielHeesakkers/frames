// internal/api/handlers_scan.go
package api

import (
	"net/http"

	"github.com/NielHeesakkers/frames/internal/scanner"
)

type scanDeps struct {
	Scheduler *scanner.Scheduler
}

func (sd *scanDeps) handleTrigger(w http.ResponseWriter, r *http.Request) {
	full := r.URL.Query().Get("full") == "1"
	sd.Scheduler.TriggerNow(full)
	WriteJSON(w, http.StatusAccepted, map[string]string{"status": "scheduled"})
}
