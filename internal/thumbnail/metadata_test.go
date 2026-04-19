// internal/thumbnail/metadata_test.go
package thumbnail

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReadMetadata_SkipsWithoutBinary(t *testing.T) {
	if _, err := exec.LookPath("exiftool"); err != nil {
		t.Skip("exiftool not installed")
	}
	// Create a synthetic JPEG via vips (no real EXIF, but exiftool still responds).
	if _, err := exec.LookPath("vips"); err != nil {
		t.Skip("vips not installed")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "x.jpg")
	if err := exec.Command("vips", "black", src, "8", "8").Run(); err != nil {
		t.Skipf("cannot build fixture: %v", err)
	}
	_, err := ReadMetadata(context.Background(), src)
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
}
