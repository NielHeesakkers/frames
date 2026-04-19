// internal/scanner/scheduler.go
package scanner

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
)

// Progress holds live counters for a currently-running scan. Zero-value when idle.
type Progress struct {
	Type        string `json:"type"`        // "full" | "incremental" | ""
	Running     bool   `json:"running"`
	StartedAt   int64  `json:"started_at"`  // unix seconds, 0 when idle
	FoldersSeen int64  `json:"folders_seen"`
	Scanned     int64  `json:"scanned"`
	Added       int64  `json:"added"`
	Updated     int64  `json:"updated"`
	Removed     int64  `json:"removed"`
}

type Scheduler struct {
	Scanner  *Scanner
	Interval time.Duration
	FullCron string
	Log      *slog.Logger

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	cronRun *cron.Cron
	trigger chan bool // true = full

	// Progress counters — touched with atomic ops so the HTTP handler can read
	// live values while a scan is mid-flight.
	progType      atomic.Value // string
	progStarted   atomic.Int64
	progFolders   atomic.Int64
	progScanned   atomic.Int64
	progAdded     atomic.Int64
	progUpdated   atomic.Int64
	progRemoved   atomic.Int64
}

// ProgressJSON returns the current Progress as an `any` so api handlers can
// render it without importing the scanner package's concrete types.
func (s *Scheduler) ProgressJSON() any { return s.Progress() }

// Progress returns a snapshot of the current scan state. Running=false when idle.
func (s *Scheduler) Progress() Progress {
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	t, _ := s.progType.Load().(string)
	return Progress{
		Type:        t,
		Running:     running,
		StartedAt:   s.progStarted.Load(),
		FoldersSeen: s.progFolders.Load(),
		Scanned:     s.progScanned.Load(),
		Added:       s.progAdded.Load(),
		Updated:     s.progUpdated.Load(),
		Removed:     s.progRemoved.Load(),
	}
}

func (s *Scheduler) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	s.trigger = make(chan bool, 8)

	// Cron for full scan.
	s.cronRun = cron.New()
	_, _ = s.cronRun.AddFunc(s.FullCron, func() { s.requestScan(true) })
	s.cronRun.Start()

	// Ticker for incremental.
	go func() {
		t := time.NewTicker(s.Interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.requestScan(false)
			}
		}
	}()

	// Runner goroutine (one scan at a time).
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case full := <-s.trigger:
				s.runOne(ctx, full)
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	if s.cronRun != nil {
		<-s.cronRun.Stop().Done()
	}
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Scheduler) TriggerNow(full bool) { s.requestScan(full) }

func (s *Scheduler) requestScan(full bool) {
	// Non-blocking: if channel full, drop.
	select {
	case s.trigger <- full:
	default:
		s.Log.Warn("scan trigger dropped (channel full)")
	}
}

func (s *Scheduler) runOne(ctx context.Context, full bool) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		s.Log.Info("scan already running, skipping")
		return
	}
	s.running = true
	s.mu.Unlock()

	kind := "incremental"
	if full {
		kind = "full"
	}
	// Reset progress counters for this run.
	s.progType.Store(kind)
	s.progStarted.Store(time.Now().Unix())
	s.progFolders.Store(0)
	s.progScanned.Store(0)
	s.progAdded.Store(0)
	s.progUpdated.Store(0)
	s.progRemoved.Store(0)

	tracker := &ProgressTracker{
		Folders: &s.progFolders,
		Scanned: &s.progScanned,
		Added:   &s.progAdded,
		Updated: &s.progUpdated,
		Removed: &s.progRemoved,
	}

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()
	stats, err := s.Scanner.ScanWithProgress(ctx, full, tracker)
	if err != nil {
		s.Log.Error("scan error", "err", err)
		return
	}
	s.Log.Info("scan done",
		"full", full, "scanned", stats.Scanned,
		"added", stats.Added, "updated", stats.Updated, "removed", stats.Removed)
}
