package main

import (
	"os"
	"testing"

	"github.com/xianxu/metis/internal/repo"
)

// repoRoot returns the metis module root (nearest ancestor go.mod), so tests can
// address testdata/ regardless of where `go test` runs. Thin wrapper over the
// shared repo.Root (ARCH-DRY — one walk implementation).
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := repo.Root(wd)
	if err != nil {
		t.Fatal(err)
	}
	return root
}
