// internal/thumbnail/image.go
package thumbnail

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// GenerateImageThumb uses `vipsthumbnail` to decode any supported format
// (JPEG, PNG, HEIC, WebP, AVIF, TIFF, and — when libvips is built with
// libraw — RAW) and produce a WebP of the given longest edge.
func GenerateImageThumb(ctx context.Context, src, dst string, size int, quality int) error {
	// Ensure the destination directory exists — cache shard dirs (00..ff) are
	// created on demand, not up front.
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	// vipsthumbnail writes output next to source by default; we pass -o with absolute dest path.
	// Format args after `[...]` control webp encode quality.
	outArg := dst + "[Q=" + itoa(quality) + ",strip]"
	cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(cctx, "vipsthumbnail",
		"--size", fmt.Sprintf("%dx%d", size, size),
		"-o", outArg,
		src,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vipsthumbnail: %w (stderr=%s)", err, stderr.String())
	}
	return nil
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }
