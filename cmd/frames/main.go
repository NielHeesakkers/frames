package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NielHeesakkers/frames/internal/api"
	"github.com/NielHeesakkers/frames/internal/config"
	"github.com/NielHeesakkers/frames/internal/db"
	"github.com/NielHeesakkers/frames/internal/logger"
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

	h := api.NewRouter(api.Deps{Log: log})
	srv := &http.Server{
		Addr:              cfg.Bind,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
