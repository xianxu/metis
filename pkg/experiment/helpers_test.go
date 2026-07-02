package experiment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/metis/internal/repo"
)

// repoRoot returns the metis module root (nearest ancestor go.mod). Shared by the
// fixture reader and the CUE-conformance drift guards so both address testdata/
// and the sibling ariadne/bin the same way regardless of where `go test` runs.
// Thin wrapper over the shared repo.Root (ARCH-DRY — one walk implementation).
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

// readFixture returns the contents of testdata/experiment/<name>.
func readFixture(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(repoRoot(t), "testdata", "experiment", name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
