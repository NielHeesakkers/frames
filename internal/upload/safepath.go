// internal/upload/safepath.go
package upload

import (
	"errors"
	"path/filepath"
	"strings"
)

var ErrBadPath = errors.New("path escapes root")

// SafeJoin cleans rel and joins it to root, returning an absolute path inside root.
// Rejects absolute relative paths and any cleaned path that would escape root.
func SafeJoin(root, rel string) (string, error) {
	if rel == "" {
		return "", ErrBadPath
	}
	if filepath.IsAbs(rel) {
		return "", ErrBadPath
	}
	cleaned := filepath.Clean(rel)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", ErrBadPath
	}
	joined := filepath.Join(root, cleaned)
	// Defensive final check.
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	joinedAbs, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(joinedAbs, rootAbs+string(filepath.Separator)) && joinedAbs != rootAbs {
		return "", ErrBadPath
	}
	return joinedAbs, nil
}
