package main

import (
	"os"
	"path/filepath"
	"testing"
)

// repoRoot walks up from the test's cwd to the nearest go.mod (the metis root),
// so the e2e test can address testdata/ regardless of where `go test` runs.
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
