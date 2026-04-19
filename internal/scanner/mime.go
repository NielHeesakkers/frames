// internal/scanner/mime.go
package scanner

import (
	"path/filepath"
	"strings"
)

var rawExts = map[string]string{
	".arw": "image/x-sony-arw",
	".cr2": "image/x-canon-cr2",
	".cr3": "image/x-canon-cr3",
	".nef": "image/x-nikon-nef",
	".dng": "image/x-adobe-dng",
	".raf": "image/x-fuji-raf",
	".rw2": "image/x-panasonic-rw2",
	".orf": "image/x-olympus-orf",
	".srw": "image/x-samsung-srw",
	".pef": "image/x-pentax-pef",
}

var imageExts = map[string]string{
	".jpg": "image/jpeg", ".jpeg": "image/jpeg",
	".png": "image/png", ".gif": "image/gif",
	".webp": "image/webp", ".avif": "image/avif",
	".heic": "image/heic", ".heif": "image/heif",
	".tif": "image/tiff", ".tiff": "image/tiff",
	".bmp": "image/bmp",
}

var videoExts = map[string]string{
	".mp4": "video/mp4", ".mov": "video/quicktime",
	".mkv": "video/x-matroska", ".avi": "video/x-msvideo",
	".webm": "video/webm", ".m4v": "video/x-m4v",
}

var otherExts = map[string]string{
	".pdf": "application/pdf",
	".mp3": "audio/mpeg", ".flac": "audio/flac", ".wav": "audio/wav",
	".txt": "text/plain", ".md": "text/markdown",
}

// Classify returns (kind, mime) for a filename. kind ∈ image|raw|video|other.
func Classify(name string) (string, string) {
	ext := strings.ToLower(filepath.Ext(name))
	if m, ok := rawExts[ext]; ok {
		return "raw", m
	}
	if m, ok := imageExts[ext]; ok {
		return "image", m
	}
	if m, ok := videoExts[ext]; ok {
		return "video", m
	}
	if m, ok := otherExts[ext]; ok {
		return "other", m
	}
	return "other", "application/octet-stream"
}
