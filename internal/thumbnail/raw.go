// internal/thumbnail/raw.go
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

// extractEmbeddedJPEG extracts the best available embedded JPEG from a RAW
// file using exiftool, writing the bytes to dstPath. It tries, in order:
//   1. -JpgFromRaw (highest quality, typically near-full-resolution)
//   2. -PreviewImage (medium)
//   3. -ThumbnailImage (smallest, last resort)
// Returns an error if no embedded JPEG is found in any of the three.
func extractEmbeddedJPEG(ctx context.Context, src, dstPath string) error {
	tags := []string{"-JpgFromRaw", "-PreviewImage", "-ThumbnailImage"}
	var lastErr error
	for _, tag := range tags {
		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		cmd := exec.CommandContext(cctx, "exiftool", "-b", tag, src)
		var out, stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		cancel()
		if err != nil {
			lastErr = fmt.Errorf("exiftool %s: %w (%s)", tag, err, stderr.String())
			continue
		}
		if out.Len() < 100 {
			// Empty / too small to be a real JPEG.
			lastErr = fmt.Errorf("exiftool %s: empty output", tag)
			continue
		}
		if err := os.WriteFile(dstPath, out.Bytes(), 0o644); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("no embedded JPEG in RAW file: %w", lastErr)
}

// GenerateRawThumb produces a WebP thumbnail of a RAW file by first extracting
// its embedded JPEG (very fast) and then running vipsthumbnail on that JPEG.
// This avoids requiring libvips to be compiled against libraw.
func GenerateRawThumb(ctx context.Context, src, dst string, size, quality int) error {
	// Make sure the destination directory exists before vipsthumbnail writes to it.
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	// Temp JPEG goes to the OS tmpdir — not the cache shard dir (which may not exist yet).
	tmp, err := os.CreateTemp("", "raw-jpeg-*.jpg")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	if err := extractEmbeddedJPEG(ctx, src, tmpPath); err != nil {
		return err
	}
	return GenerateImageThumb(ctx, tmpPath, dst, size, quality)
}

// GenerateRawPreview produces a WebP preview at the larger "preview" size.
// Same strategy: extract embedded JPEG (the biggest one available) and
// resize it to the preview dimensions.
func GenerateRawPreview(ctx context.Context, src, dst string, size, quality int) error {
	return GenerateRawThumb(ctx, src, dst, size, quality)
}
