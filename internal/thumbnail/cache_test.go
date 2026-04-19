// internal/thumbnail/cache_test.go
package thumbnail

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCachePaths(t *testing.T) {
	c := &Cache{Root: "/cache"}
	got := c.ThumbPath(1234)
	want := "/cache/thumb/d2/4d2.webp" // 1234 hex = 4d2, shard = first 2 of 04d2 padded
	if got != want {
		t.Errorf("thumb path = %q want %q", got, want)
	}
	if c.PreviewPath(1234) != "/cache/preview/d2/4d2.webp" {
		t.Errorf("preview path")
	}
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	c := &Cache{Root: dir}
	if err := c.WriteAtomic(c.ThumbPath(7), []byte("data")); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(c.ThumbPath(7))
	if string(b) != "data" {
		t.Errorf("content mismatch")
	}
	// tmp should be empty
	entries, _ := os.ReadDir(filepath.Join(dir, "tmp"))
	if len(entries) != 0 {
		t.Errorf("tmp not cleaned: %d entries", len(entries))
	}
}
