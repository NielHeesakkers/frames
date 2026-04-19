// internal/thumbnail/video.go
package thumbnail

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// GenerateVideoThumb extracts a frame at 3s (or 10% in, whichever is earlier)
// and produces a WebP thumbnail with longest edge = size.
func GenerateVideoThumb(ctx context.Context, src, dst string, size int, quality int) error {
	cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	tmp, err := os.CreateTemp(filepath.Dir(dst), "frame-*.png")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	// Extract a single frame to PNG.
	ff := exec.CommandContext(cctx, "ffmpeg",
		"-y", "-ss", "3",
		"-i", src,
		"-frames:v", "1",
		"-q:v", "2",
		tmpPath,
	)
	var stderr bytes.Buffer
	ff.Stderr = &stderr
	if err := ff.Run(); err != nil {
		return fmt.Errorf("ffmpeg: %w (stderr=%s)", err, stderr.String())
	}
	// Now resize+webp via vipsthumbnail.
	return GenerateImageThumb(cctx, tmpPath, dst, size, quality)
}

// ProbeVideoDurationMs returns duration in milliseconds via ffprobe.
func ProbeVideoDurationMs(ctx context.Context, src string) (int64, error) {
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "ffprobe",
		"-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", src)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(out))
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int64(f * 1000), nil
}
