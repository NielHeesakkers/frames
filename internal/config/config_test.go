package config

import (
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("FRAMES_SESSION_SECRET", "x")
	t.Setenv("FRAMES_PUBLIC_URL", "http://localhost:8080")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Bind != ":8080" {
		t.Errorf("Bind default = %q; want :8080", cfg.Bind)
	}
	if cfg.PhotosRoot != "/photos" {
		t.Errorf("PhotosRoot default = %q; want /photos", cfg.PhotosRoot)
	}
	if cfg.ScanInterval != 5*time.Minute {
		t.Errorf("ScanInterval default = %v; want 5m", cfg.ScanInterval)
	}
}

func TestLoad_RequiredMissing(t *testing.T) {
	t.Setenv("FRAMES_SESSION_SECRET", "")
	t.Setenv("FRAMES_PUBLIC_URL", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing required vars")
	}
}

func TestLoad_ParseDurationAndCron(t *testing.T) {
	t.Setenv("FRAMES_SESSION_SECRET", "x")
	t.Setenv("FRAMES_PUBLIC_URL", "http://localhost:8080")
	t.Setenv("FRAMES_SCAN_INTERVAL", "2m")
	t.Setenv("FRAMES_FULL_SCAN_CRON", "0 4 * * *")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ScanInterval != 2*time.Minute {
		t.Errorf("ScanInterval = %v; want 2m", cfg.ScanInterval)
	}
	if cfg.FullScanCron != "0 4 * * *" {
		t.Errorf("FullScanCron = %q", cfg.FullScanCron)
	}
}
