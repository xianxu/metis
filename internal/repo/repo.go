// Package repo locates an ancestor directory by a marker file: the metis module
// root (nearest go.mod) for the runner's step-path fallback + test helpers, and
// the workspace construct-layer root (nearest construct/base.manifest) for
// dependency-graph step discovery (metis#16). One up-walk shared by every
// "nearest ancestor holding X" lookup, replacing near-identical copies (ARCH-DRY).
package repo

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindUp walks up from start to the nearest ancestor directory containing the
// relative marker path (e.g. "go.mod" or "construct/base.manifest"), returning
// that directory. It errors if none is found before the filesystem root.
func FindUp(start, marker string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s found above %s", marker, start)
		}
		dir = parent
	}
}

// Root walks up from start to the nearest ancestor directory containing a go.mod,
// returning that directory. It errors if none is found before the filesystem root.
func Root(start string) (string, error) {
	return FindUp(start, "go.mod")
}
