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

// GenerateVideoThumb extracts the middle frame of the video (based on duration
// from ffprobe) and produces a WebP thumbnail with longest edge = size.
// Falls back to "3 seconds in" when the duration can't be read or the clip is
// shorter than ~1 second.
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

	// Pick the middle of the clip. For very short / unknown duration, fall
	// back to 1 second (or 3 s for anything 6 s+).
	seek := "3"
	if durMs, derr := ProbeVideoDurationMs(cctx, src); derr == nil && durMs > 0 {
		mid := float64(durMs) / 2000.0 // seconds
		if mid < 0.5 {
			mid = 0
		}
		seek = strconv.FormatFloat(mid, 'f', 2, 64)
	}

	ff := exec.CommandContext(cctx, "ffmpeg",
		"-y", "-ss", seek,
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
