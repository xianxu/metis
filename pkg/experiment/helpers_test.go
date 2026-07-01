package experiment

import (
	"os"
	"path/filepath"
	"testing"
)

// repoRoot walks up from the test's working directory (the package dir) to the
// nearest ancestor holding go.mod — the metis repo root. Shared by the fixture
// reader and the CUE-conformance drift guard so both address testdata/ and the
// sibling ariadne/bin the same way regardless of where `go test` is invoked.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repoRoot: no go.mod found above cwd")
		}
		dir = parent
	}
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
