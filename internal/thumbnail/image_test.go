// internal/thumbnail/image_test.go
package thumbnail

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGenerateImageThumb(t *testing.T) {
	if _, err := exec.LookPath("vipsthumbnail"); err != nil {
		t.Skip("vipsthumbnail not installed")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "src.jpg")
	// Write a tiny valid JPEG by shelling to vipsthumbnail's helper — otherwise skip.
	if err := exec.Command("vips", "black", src, "4", "4").Run(); err != nil {
		t.Skipf("cannot create fixture: %v", err)
	}
	dst := filepath.Join(dir, "thumb.webp")
	if err := GenerateImageThumb(context.Background(), src, dst, 128, 75); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(dst)
	if err != nil || fi.Size() == 0 {
		t.Fatalf("output missing: %v size=%d", err, fi.Size())
	}
}
