package config

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Bind           string
	PhotosRoot     string
	CacheDir       string
	DataDir        string
	ScanInterval   time.Duration
	FullScanCron   string
	Workers        int
	SessionSecret  string
	PublicURL      string
	MaxUploadSize  int64
	ShareUploadMax int64
	AdminUsername  string
	AdminPassword  string
	TrustProxy     bool
}

func Load() (*Config, error) {
	cfg := &Config{
		Bind:          getenv("FRAMES_BIND", ":8080"),
		PhotosRoot:    getenv("FRAMES_PHOTOS_ROOT", "/photos"),
		CacheDir:      getenv("FRAMES_CACHE_DIR", "/cache"),
		DataDir:       getenv("FRAMES_DATA_DIR", "/data"),
		FullScanCron:  getenv("FRAMES_FULL_SCAN_CRON", "0 3 * * *"),
		SessionSecret: os.Getenv("FRAMES_SESSION_SECRET"),
		PublicURL:     os.Getenv("FRAMES_PUBLIC_URL"),
		AdminUsername: os.Getenv("FRAMES_ADMIN_USERNAME"),
		AdminPassword: os.Getenv("FRAMES_ADMIN_PASSWORD"),
	}

	dur, err := time.ParseDuration(getenv("FRAMES_SCAN_INTERVAL", "5m"))
	if err != nil {
		return nil, fmt.Errorf("FRAMES_SCAN_INTERVAL: %w", err)
	}
	cfg.ScanInterval = dur

	cfg.Workers, err = atoiDefault(os.Getenv("FRAMES_WORKERS"), runtime.NumCPU())
	if err != nil {
		return nil, fmt.Errorf("FRAMES_WORKERS: %w", err)
	}

	cfg.MaxUploadSize, err = parseSize(getenv("FRAMES_MAX_UPLOAD_SIZE", "5GB"))
	if err != nil {
		return nil, fmt.Errorf("FRAMES_MAX_UPLOAD_SIZE: %w", err)
	}
	cfg.ShareUploadMax, err = parseSize(getenv("FRAMES_SHARE_UPLOAD_MAX", "500MB"))
	if err != nil {
		return nil, fmt.Errorf("FRAMES_SHARE_UPLOAD_MAX: %w", err)
	}

	cfg.TrustProxy = parseBool(os.Getenv("FRAMES_TRUST_PROXY"))

	if cfg.SessionSecret == "" {
		return nil, errors.New("FRAMES_SESSION_SECRET is required")
	}
	if cfg.PublicURL == "" {
		return nil, errors.New("FRAMES_PUBLIC_URL is required")
	}
	return cfg, nil
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func atoiDefault(s string, d int) (int, error) {
	if s == "" {
		return d, nil
	}
	return strconv.Atoi(s)
}

func parseSize(s string) (int64, error) {
	if s == "" {
		return 0, errors.New("empty size")
	}
	mult := int64(1)
	switch {
	case hasSuffix(s, "GB"):
		mult = 1 << 30
		s = s[:len(s)-2]
	case hasSuffix(s, "MB"):
		mult = 1 << 20
		s = s[:len(s)-2]
	case hasSuffix(s, "KB"):
		mult = 1 << 10
		s = s[:len(s)-2]
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return n * mult, nil
}

func hasSuffix(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}

func parseBool(s string) bool {
	switch strings.ToLower(s) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
