package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// physicalPath returns p absolute with every symlink resolved (metis#76): /tmp → /private/tmp,
// a macOS TMPDIR under /var, a symlinked experiment or runs/ dir. A step may hand these paths to
// a backend that refuses non-physical ones, and the child compares METIS_EXP_DIR-relative reads
// against METIS_READ_ROOT, so both must share one form. p need not exist yet: the deepest
// existing ancestor is resolved and the rest re-joined.
func physicalPath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	rest := ""
	for dir := abs; ; dir = filepath.Dir(dir) {
		if _, err := os.Lstat(dir); err == nil {
			r, err := filepath.EvalSymlinks(dir)
			if err != nil {
				return "", err
			}
			return filepath.Join(r, rest), nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		if parent := filepath.Dir(dir); parent == dir {
			return abs, nil
		}
		rest = filepath.Join(filepath.Base(dir), rest)
	}
}
