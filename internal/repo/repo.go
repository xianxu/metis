// Package repo locates the metis module root: the nearest ancestor directory
// holding go.mod. One implementation shared by the runner's step-path resolution
// (cmd/metis) and the test helpers, replacing three near-identical copies of the
// same walk (ARCH-DRY).
package repo

import (
	"fmt"
	"os"
	"path/filepath"
)

// Root walks up from start to the nearest ancestor directory containing a go.mod,
// returning that directory. It errors if none is found before the filesystem root.
func Root(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found above %s", start)
		}
		dir = parent
	}
}
