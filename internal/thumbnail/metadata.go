// internal/thumbnail/metadata.go
package thumbnail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/NielHeesakkers/frames/internal/db"
)

// ReadMetadata calls `exiftool -json -DateTimeOriginal -ImageWidth -ImageHeight -Make -Model -Orientation`
// on the source and maps the result onto db.MetadataUpdate.
func ReadMetadata(ctx context.Context, src string) (db.MetadataUpdate, error) {
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "exiftool",
		"-json", "-d", "%Y-%m-%dT%H:%M:%S",
		"-DateTimeOriginal",
		"-ImageWidth", "-ImageHeight",
		"-Make", "-Model", "-Orientation",
		src,
	)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return db.MetadataUpdate{}, fmt.Errorf("exiftool: %w (%s)", err, stderr.String())
	}
	var arr []struct {
		DateTimeOriginal string `json:"DateTimeOriginal"`
		ImageWidth       int    `json:"ImageWidth"`
		ImageHeight      int    `json:"ImageHeight"`
		Make             string `json:"Make"`
		Model            string `json:"Model"`
		Orientation      any    `json:"Orientation"`
	}
	if err := json.Unmarshal(out.Bytes(), &arr); err != nil || len(arr) == 0 {
		return db.MetadataUpdate{}, fmt.Errorf("parse exiftool json: %w", err)
	}
	e := arr[0]
	up := db.MetadataUpdate{}
	if e.DateTimeOriginal != "" {
		if t, err := time.Parse("2006-01-02T15:04:05", e.DateTimeOriginal); err == nil {
			up.TakenAt = &t
		}
	}
	if e.ImageWidth > 0 {
		w := e.ImageWidth
		up.Width = &w
	}
	if e.ImageHeight > 0 {
		h := e.ImageHeight
		up.Height = &h
	}
	if e.Make != "" {
		m := strings.TrimSpace(e.Make)
		up.CameraMake = &m
	}
	if e.Model != "" {
		m := strings.TrimSpace(e.Model)
		up.CameraModel = &m
	}
	// Orientation: exiftool returns either integer 1-8 or descriptive string.
	if e.Orientation != nil {
		o := parseOrientation(e.Orientation)
		if o > 0 {
			up.Orientation = &o
		}
	}
	return up, nil
}

func parseOrientation(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case string:
		// descriptive forms like "Rotate 90 CW"
		switch x {
		case "Horizontal (normal)":
			return 1
		case "Mirror horizontal":
			return 2
		case "Rotate 180":
			return 3
		case "Mirror vertical":
			return 4
		case "Mirror horizontal and rotate 270 CW":
			return 5
		case "Rotate 90 CW":
			return 6
		case "Mirror horizontal and rotate 90 CW":
			return 7
		case "Rotate 270 CW":
			return 8
		}
	}
	return 0
}
