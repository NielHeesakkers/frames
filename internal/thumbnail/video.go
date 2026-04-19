// internal/thumbnail/video.go
package thumbnail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// VideoInfo summarises the probe data we care about for a video.
type VideoInfo struct {
	DurationMs int64
	Width      int
	Height     int
	// Rotation in degrees (0/90/180/270) as reported by container metadata.
	Rotation int
}

// GenerateVideoThumb extracts the middle frame of the video and produces a
// WebP thumbnail with longest edge = size. ffmpeg auto-applies rotation
// metadata, so the resulting frame is already oriented correctly.
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

	seek := "3"
	if info, derr := ProbeVideoInfo(cctx, src); derr == nil && info.DurationMs > 0 {
		mid := float64(info.DurationMs) / 2000.0
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
// Kept for backwards compatibility; new code should prefer ProbeVideoInfo.
func ProbeVideoDurationMs(ctx context.Context, src string) (int64, error) {
	info, err := ProbeVideoInfo(ctx, src)
	return info.DurationMs, err
}

// ProbeVideoInfo asks ffprobe for the first video stream's width, height,
// and rotation plus the container-level duration. Returns all-zero + error
// when ffprobe fails or the input has no video stream.
func ProbeVideoInfo(ctx context.Context, src string) (VideoInfo, error) {
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height:stream_tags=rotate:stream_side_data=rotation:format=duration",
		"-of", "json",
		src)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return VideoInfo{}, err
	}
	var parsed struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
			Tags   struct {
				Rotate string `json:"rotate"`
			} `json:"tags"`
			SideDataList []struct {
				Rotation any `json:"rotation"`
			} `json:"side_data_list"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		return VideoInfo{}, err
	}
	info := VideoInfo{}
	if f, err := strconv.ParseFloat(strings.TrimSpace(parsed.Format.Duration), 64); err == nil {
		info.DurationMs = int64(f * 1000)
	}
	if len(parsed.Streams) > 0 {
		s := parsed.Streams[0]
		info.Width = s.Width
		info.Height = s.Height
		if r, err := strconv.Atoi(strings.TrimSpace(s.Tags.Rotate)); err == nil {
			info.Rotation = ((r % 360) + 360) % 360
		}
		// side_data_list.rotation is typically -90 for 90° CW.
		if info.Rotation == 0 {
			for _, sd := range s.SideDataList {
				if sd.Rotation == nil {
					continue
				}
				switch v := sd.Rotation.(type) {
				case float64:
					info.Rotation = ((int(v) % 360) + 360) % 360
				case string:
					if n, err := strconv.Atoi(v); err == nil {
						info.Rotation = ((n % 360) + 360) % 360
					}
				}
				if info.Rotation != 0 {
					break
				}
			}
		}
	}
	return info, nil
}
