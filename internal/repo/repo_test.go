package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoot_FindsGoMod(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := Root(wd) // this package lives under the metis module
	if err != nil {
		t.Fatalf("Root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("Root returned %s but it has no go.mod: %v", root, err)
	}
}

func TestRoot_NoGoModAbove(t *testing.T) {
	if _, err := Root(t.TempDir()); err == nil {
		t.Fatal("Root: want an error for a directory with no go.mod ancestor")
	}
}
