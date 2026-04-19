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
	// Swap width/height when EXIF orientation indicates a 90° rotation
	// (5,6,7,8 are the "sideways" orientations). We want stored dimensions
	// to match the *displayed* image, which is what layout code and the
	// lightbox need.
	w := e.ImageWidth
	h := e.ImageHeight
	if e.Orientation != nil {
		o := parseOrientation(e.Orientation)
		if o >= 5 && o <= 8 {
			w, h = h, w
		}
	}
	if w > 0 {
		up.Width = &w
	}
	if h > 0 {
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
	// Orientation stored as an integer 1-8 (EXIF orientation).
	if e.Orientation != nil {
		o := parseOrientation(e.Orientation)
		if o > 0 {
			up.Orientation = &o
		}
	}
	return up, nil
}

// Also swap Width/Height on the detailed EXIF view when orientation is rotated.

// DetailedEXIF holds a richer EXIF summary pulled on demand (e.g. when the
// lightbox is opened). These fields are not persisted to the DB — they are
// cheap enough to read with a single exiftool invocation per request.
type DetailedEXIF struct {
	TakenAt     string `json:"taken_at,omitempty"`
	Camera      string `json:"camera,omitempty"`
	Lens        string `json:"lens,omitempty"`
	Aperture    string `json:"aperture,omitempty"`       // f/2.8
	Shutter     string `json:"shutter_speed,omitempty"`  // 1/250 s
	ISO         string `json:"iso,omitempty"`
	FocalLength string `json:"focal_length,omitempty"`   // 35mm
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	GPSLat      string `json:"gps_lat,omitempty"`
	GPSLon      string `json:"gps_lon,omitempty"`
	Software    string `json:"software,omitempty"`
}

// ReadDetailedEXIF returns a user-friendly EXIF summary for the Lightbox.
// Uses exiftool's human-readable defaults so the strings ("1/250", "f/2.8",
// "35 mm") don't need re-formatting in the frontend.
func ReadDetailedEXIF(ctx context.Context, src string) (DetailedEXIF, error) {
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "exiftool",
		"-json", "-d", "%Y-%m-%d %H:%M:%S",
		"-DateTimeOriginal",
		"-Make", "-Model",
		"-LensModel", "-Lens", "-LensID",
		"-FNumber", "-Aperture",
		"-ExposureTime", "-ShutterSpeed",
		"-ISO",
		"-FocalLength", "-FocalLengthIn35mmFormat",
		"-ImageWidth", "-ImageHeight",
		"-GPSLatitude", "-GPSLongitude",
		"-Software",
		src,
	)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return DetailedEXIF{}, fmt.Errorf("exiftool: %w (%s)", err, stderr.String())
	}
	var arr []map[string]any
	if err := json.Unmarshal(out.Bytes(), &arr); err != nil || len(arr) == 0 {
		return DetailedEXIF{}, fmt.Errorf("parse exiftool json: %w", err)
	}
	e := arr[0]
	d := DetailedEXIF{}
	d.TakenAt = strFrom(e, "DateTimeOriginal")
	make_ := strings.TrimSpace(strFrom(e, "Make"))
	model := strings.TrimSpace(strFrom(e, "Model"))
	if make_ != "" && model != "" {
		d.Camera = strings.TrimSpace(make_ + " " + model)
	} else if model != "" {
		d.Camera = model
	} else if make_ != "" {
		d.Camera = make_
	}
	d.Lens = firstNonEmpty(strFrom(e, "LensModel"), strFrom(e, "Lens"), strFrom(e, "LensID"))
	if ap := firstNonEmpty(strFrom(e, "FNumber"), strFrom(e, "Aperture")); ap != "" {
		d.Aperture = "f/" + ap
	}
	if s := firstNonEmpty(strFrom(e, "ExposureTime"), strFrom(e, "ShutterSpeed")); s != "" {
		d.Shutter = s + " s"
	}
	d.ISO = strFrom(e, "ISO")
	if fl := firstNonEmpty(strFrom(e, "FocalLength"), strFrom(e, "FocalLengthIn35mmFormat")); fl != "" {
		d.FocalLength = fl
	}
	d.Width = intFrom(e, "ImageWidth")
	d.Height = intFrom(e, "ImageHeight")
	// Swap when the photo is side-oriented (EXIF orientation 5..8).
	if rawO, ok := e["Orientation"]; ok && rawO != nil {
		o := parseOrientation(rawO)
		if o >= 5 && o <= 8 {
			d.Width, d.Height = d.Height, d.Width
		}
	}
	d.GPSLat = strFrom(e, "GPSLatitude")
	d.GPSLon = strFrom(e, "GPSLongitude")
	d.Software = strFrom(e, "Software")
	return d, nil
}

func strFrom(m map[string]any, k string) string {
	v, ok := m[k]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%g", x)
	case int:
		return fmt.Sprintf("%d", x)
	}
	return fmt.Sprintf("%v", v)
}

func intFrom(m map[string]any, k string) int {
	v, ok := m[k]
	if !ok || v == nil {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case string:
		var n int
		fmt.Sscanf(x, "%d", &n)
		return n
	}
	return 0
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
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
