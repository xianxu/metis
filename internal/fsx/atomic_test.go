package fsx

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomic_WritesWithPerm(t *testing.T) {
	p := filepath.Join(t.TempDir(), "run.json")
	if err := WriteFileAtomic(p, []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil || string(b) != "new\n" {
		t.Fatalf("content = %q, %v", b, err)
	}
	fi, _ := os.Stat(p)
	if fi.Mode().Perm() != 0o644 {
		t.Fatalf("perm = %v, want 0644", fi.Mode().Perm())
	}
}

// A crash before the rename (simulated by failing it) leaves the old file intact and no
// temp file behind: an interrupted publication never leaves a truncated record.
func TestWriteFileAtomic_FailureBeforeRenameKeepsOldFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "record.json")
	if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	rename = func(string, string) error { return errors.New("injected crash") }
	t.Cleanup(func() { rename = os.Rename })

	if err := WriteFileAtomic(p, []byte("new-but-interrupted"), 0o644); err == nil {
		t.Fatal("want the injected error")
	}
	if b, _ := os.ReadFile(p); string(b) != "old" {
		t.Fatalf("old file changed: %q", b)
	}
	ents, _ := os.ReadDir(dir)
	if len(ents) != 1 {
		t.Fatalf("temp file left behind: %v", ents)
	}
}
