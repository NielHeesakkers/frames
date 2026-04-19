package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/NielHeesakkers/frames/internal/api"
	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/config"
	"github.com/NielHeesakkers/frames/internal/db"
	"github.com/NielHeesakkers/frames/internal/fsops"
	"github.com/NielHeesakkers/frames/internal/logger"
	"github.com/NielHeesakkers/frames/internal/scanner"
	"github.com/NielHeesakkers/frames/internal/thumbnail"
	"github.com/NielHeesakkers/frames/internal/upload"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	log := logger.New(os.Getenv("FRAMES_LOG_LEVEL"))
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log.Info("loaded config", "bind", cfg.Bind, "photos", cfg.PhotosRoot)

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return err
	}
	database, err := db.Open(cfg.DataDir)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := database.Migrate(); err != nil {
		return err
	}

	created, err := auth.BootstrapAdmin(database, cfg.AdminUsername, cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("bootstrap admin: %w", err)
	}
	if created {
		log.Info("admin user created", "username", cfg.AdminUsername)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	sc := &scanner.Scanner{DB: database, Log: log, Root: cfg.PhotosRoot}
	sched := &scanner.Scheduler{
		Scanner: sc, Interval: cfg.ScanInterval,
		FullCron: cfg.FullScanCron, Log: log,
	}
	sched.Start(ctx)
	defer sched.Stop()

	go func() {
		t := time.NewTicker(1 * time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if n, err := database.CleanupExpiredSessions(); err == nil && n > 0 {
					log.Info("cleaned expired sessions", "count", n)
				}
			}
		}
	}()

	cache := &thumbnail.Cache{Root: cfg.CacheDir}
	if err := cache.Ensure(); err != nil {
		return err
	}
	q := thumbnail.NewQueue(4096)
	uploadSvc := &upload.Service{DB: database, Root: cfg.PhotosRoot}
	pool := &thumbnail.Pool{
		DB: database, Cache: cache, Queue: q, Log: log,
		Root: cfg.PhotosRoot, Workers: cfg.Workers,
	}
	pool.Start(ctx)

	ops := &fsops.Ops{DB: database, Root: cfg.PhotosRoot, Cache: cache}

	lim := auth.NewLoginLimiter(5, 15*time.Minute)
	h := api.NewRouter(api.Deps{
		Log: log, DB: database, Limiter: lim, Scheduler: sched,
		Cache: cache, Queue: q, Pool: pool, Ops: ops, Root: cfg.PhotosRoot,
		UploadSvc:      uploadSvc,
		MaxUpload:      cfg.MaxUploadSize,
		ShareUploadMax: cfg.ShareUploadMax,
		Secure:         strings.HasPrefix(cfg.PublicURL, "https://"),
		TrustProxy:     cfg.TrustProxy,
		PublicURL:      cfg.PublicURL,
	})
	srv := &http.Server{
		Addr:              cfg.Bind,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http listening", "addr", cfg.Bind)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down")
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
