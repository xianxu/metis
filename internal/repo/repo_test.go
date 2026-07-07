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

func TestFindUp_FindsMarker(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "construct"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "construct", "base.manifest"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := FindUp(sub, filepath.Join("construct", "base.manifest"))
	if err != nil {
		t.Fatalf("FindUp: %v", err)
	}
	// EvalSymlinks both sides: macOS TempDir is /var→/private/var symlinked.
	wantR, _ := filepath.EvalSymlinks(root)
	gotR, _ := filepath.EvalSymlinks(got)
	if gotR != wantR {
		t.Fatalf("FindUp = %s; want %s", gotR, wantR)
	}
}

func TestFindUp_NoMarker(t *testing.T) {
	if _, err := FindUp(t.TempDir(), "nonesuch.marker"); err == nil {
		t.Fatal("FindUp: want an error when the marker is never found")
	}
}
