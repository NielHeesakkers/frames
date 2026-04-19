// internal/thumbnail/raw.go
package thumbnail

import "context"

// GenerateRawPreview renders a RAW file at preview size using libvips.
// libvips (when built with libraw support) loads RAW files natively.
// We reuse GenerateImageThumb with the larger preview size.
func GenerateRawPreview(ctx context.Context, src, dst string, size, quality int) error {
	return GenerateImageThumb(ctx, src, dst, size, quality)
}
