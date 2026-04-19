// internal/scanner/scanner_test.go
package scanner

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NielHeesakkers/frames/internal/db"
)

func writeFile(t *testing.T, p string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, size)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanner_FullRoundTrip(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "2024", "a.jpg"), 10)
	writeFile(t, filepath.Join(root, "2024", "b.jpg"), 20)
	writeFile(t, filepath.Join(root, "2023", "c.nef"), 30)

	d, _ := db.Open(t.TempDir())
	defer d.Close()
	if err := d.Migrate(); err != nil {
		t.Fatal(err)
	}

	s := &Scanner{DB: d, Log: slog.Default(), Root: root}
	stats, err := s.Scan(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Added != 3 {
		t.Errorf("added=%d want 3", stats.Added)
	}

	// Delete a file, add a new one.
	if err := os.Remove(filepath.Join(root, "2024", "b.jpg")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "2024", "c.jpg"), 15)
	// Bump the parent dir's mtime so the incremental scan visits it.
	now := time.Now()
	_ = os.Chtimes(filepath.Join(root, "2024"), now, now)

	stats, err = s.Scan(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Added != 1 {
		t.Errorf("added(2)=%d want 1", stats.Added)
	}
	if stats.Removed != 1 {
		t.Errorf("removed(2)=%d want 1", stats.Removed)
	}
}
