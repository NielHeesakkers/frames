// internal/scanner/scheduler.go
package scanner

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

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
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()
	stats, err := s.Scanner.Scan(ctx, full)
	if err != nil {
		s.Log.Error("scan error", "err", err)
		return
	}
	s.Log.Info("scan done",
		"full", full, "scanned", stats.Scanned,
		"added", stats.Added, "updated", stats.Updated, "removed", stats.Removed)
}
