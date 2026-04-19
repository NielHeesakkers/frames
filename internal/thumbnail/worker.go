// internal/thumbnail/worker.go
package thumbnail

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/NielHeesakkers/frames/internal/db"
)

const (
	ThumbSize     = 256
	PreviewSize   = 2048
	ThumbQuality  = 75
	PreviewQuality = 85
	MaxAttempts   = 3
)

type Pool struct {
	DB      *db.DB
	Cache   *Cache
	Queue   *Queue
	Log     *slog.Logger
	Root    string
	Workers int
}

func (p *Pool) Start(ctx context.Context) {
	_ = p.Cache.Ensure()
	var wg sync.WaitGroup
	for i := 0; i < p.Workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			p.workerLoop(ctx, n)
		}(i)
	}
	// Seeder: periodically fill the queue from PendingThumbs.
	go p.seeder(ctx)
	go func() {
		<-ctx.Done()
		wg.Wait()
	}()
}

func (p *Pool) seeder(ctx context.Context) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	// Initial seed.
	p.fillQueue()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.fillQueue()
		}
	}
}

func (p *Pool) fillQueue() {
	if p.Queue.Len() > 200 {
		return
	}
	pending, err := p.DB.PendingThumbs(500)
	if err != nil {
		p.Log.Warn("pending thumbs fetch failed", "err", err)
		return
	}
	for _, f := range pending {
		p.Queue.Push(f.ID, PrioBackground)
	}
}

func (p *Pool) workerLoop(ctx context.Context, n int) {
	for {
		if ctx.Err() != nil {
			return
		}
		id := p.Queue.Pop()
		if id == -1 {
			select {
			case <-ctx.Done():
				return
			case <-p.Queue.Notify():
			case <-time.After(2 * time.Second):
			}
			continue
		}
		if err := p.processOne(ctx, id); err != nil {
			p.Log.Warn("thumb worker failed", "id", id, "worker", n, "err", err)
		}
	}
}

func (p *Pool) processOne(ctx context.Context, id int64) error {
	f, err := p.DB.FileByID(id)
	if err != nil {
		return err
	}
	if f.ThumbStatus == "ready" {
		return nil
	}
	if f.ThumbAttempts >= MaxAttempts {
		return nil
	}
	src := filepath.Join(p.Root, f.RelativePath)
	dst := p.Cache.ThumbPath(id)

	// Read metadata (best effort).
	if f.Kind == "video" {
		// ffprobe-driven metadata: width/height/rotation/duration.
		if info, derr := ProbeVideoInfo(ctx, src); derr == nil {
			w, h := info.Width, info.Height
			if info.Rotation == 90 || info.Rotation == 270 {
				w, h = h, w
			}
			up := db.MetadataUpdate{
				DurationMs: &info.DurationMs,
			}
			if w > 0 {
				up.Width = &w
			}
			if h > 0 {
				up.Height = &h
			}
			_ = p.DB.UpdateFileMetadata(id, up)
		}
	} else {
		if up, mErr := ReadMetadata(ctx, src); mErr == nil {
			_ = p.DB.UpdateFileMetadata(id, up)
		}
	}

	switch f.Kind {
	case "image":
		err = GenerateImageThumb(ctx, src, dst, ThumbSize, ThumbQuality)
	case "raw":
		// libvips on Alpine isn't built with libraw support, so we can't decode
		// RAW pixels directly. Every modern RAW format embeds a JPEG preview —
		// we extract that with exiftool and thumbnail the extracted JPEG.
		err = GenerateRawThumb(ctx, src, dst, ThumbSize, ThumbQuality)
	case "video":
		err = GenerateVideoThumb(ctx, src, dst, ThumbSize, ThumbQuality)
	default:
		// No thumbnail for 'other' files; mark ready with attempts capped so we don't retry.
		_ = p.DB.SetThumbStatus(id, "failed", true)
		return nil
	}
	if err != nil {
		_ = p.DB.SetThumbStatus(id, "pending", true) // bump attempts; still pending unless capped.
		// If we've hit the cap, flip to failed.
		f2, _ := p.DB.FileByID(id)
		if f2 != nil && f2.ThumbAttempts >= MaxAttempts {
			_ = p.DB.SetThumbStatus(id, "failed", false)
		}
		return err
	}
	return p.DB.SetThumbStatus(id, "ready", false)
}

// GeneratePreview renders a preview on demand and updates status.
func (p *Pool) GeneratePreview(ctx context.Context, id int64) error {
	f, err := p.DB.FileByID(id)
	if err != nil {
		return err
	}
	if f.PreviewStatus == "ready" {
		return nil
	}
	src := filepath.Join(p.Root, f.RelativePath)
	dst := p.Cache.PreviewPath(id)
	switch f.Kind {
	case "image":
		err = GenerateImageThumb(ctx, src, dst, PreviewSize, PreviewQuality)
	case "raw":
		err = GenerateRawPreview(ctx, src, dst, PreviewSize, PreviewQuality)
	case "video":
		// Use larger frame.
		err = GenerateVideoThumb(ctx, src, dst, PreviewSize, PreviewQuality)
	default:
		return nil
	}
	if err != nil {
		_ = p.DB.SetPreviewStatus(id, "failed", true)
		return err
	}
	return p.DB.SetPreviewStatus(id, "ready", false)
}
