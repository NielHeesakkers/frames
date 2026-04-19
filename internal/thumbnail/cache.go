// internal/thumbnail/cache.go
package thumbnail

import (
	"fmt"
	"os"
	"path/filepath"
)

type Cache struct {
	Root string
}

func (c *Cache) shard(id int64) string {
	// Take last 2 hex chars of padded id as shard directory.
	hex := fmt.Sprintf("%04x", id)
	return hex[len(hex)-2:]
}

func (c *Cache) idHex(id int64) string {
	return fmt.Sprintf("%x", id)
}

func (c *Cache) ThumbPath(id int64) string {
	return filepath.Join(c.Root, "thumb", c.shard(id), c.idHex(id)+".webp")
}

func (c *Cache) PreviewPath(id int64) string {
	return filepath.Join(c.Root, "preview", c.shard(id), c.idHex(id)+".webp")
}

func (c *Cache) TmpDir() string { return filepath.Join(c.Root, "tmp") }

func (c *Cache) Ensure() error {
	for _, d := range []string{"thumb", "preview", "tmp"} {
		if err := os.MkdirAll(filepath.Join(c.Root, d), 0o755); err != nil {
			return err
		}
	}
	return nil
}

// WriteAtomic writes data to the final path by first writing to tmp/ and renaming.
func (c *Cache) WriteAtomic(finalPath string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(c.TmpDir(), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(c.TmpDir(), "w-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), finalPath)
}

// RemoveDerivatives deletes thumb + preview for a file id (ignores ENOENT).
func (c *Cache) RemoveDerivatives(id int64) {
	_ = os.Remove(c.ThumbPath(id))
	_ = os.Remove(c.PreviewPath(id))
}
