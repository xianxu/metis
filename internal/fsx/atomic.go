// Package fsx holds metis's small filesystem primitives: the one atomic file write the CAS
// store, the step cache index, run records and the ledger all use (metis#73, ARCH-DRY).
package fsx

import (
	"os"
	"path/filepath"
)

// TmpPrefix marks an in-flight atomic write's temp file, so a directory scan (the CAS
// eviction sweep) can skip it.
const TmpPrefix = ".tmp-"

// rename is the commit step; a test replaces it to simulate a crash before the rename.
var rename = os.Rename

// WriteFileAtomic writes data to path through a temp file in the same directory, then
// renames it into place: a reader (or a crash) sees the old file or the new one, never a
// truncated one. The temp file is removed on any failure.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), TmpPrefix+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	fail := func(err error) error {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return fail(err)
	}
	if err := tmp.Chmod(perm); err != nil {
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}
