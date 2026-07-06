package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitInit(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@t.co"}, {"config", "user.name", "t"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func gitCommitAll(t *testing.T, dir, msg string) {
	t.Helper()
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-qm", msg}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// captureClosure commits a DIRTY closure file to a side ref and returns a real commit
// SHA + the (path, blob-hash) manifest; git cat-file of the manifest hash returns the
// EXACT dirty bytes, and git checkout of the SHA restores the dirty version — the
// durability the design promises (a dirty run still has a recoverable committed SHA).
func TestCaptureClosure_DirtyFile(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := gitInit(t)
	src := filepath.Join(root, "model.py")
	if err := os.WriteFile(src, []byte("x = 1  # committed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, root, "init")
	// Now DIRTY the file (uncommitted edit).
	dirty := "x = 2  # dirty edit\n"
	if err := os.WriteFile(src, []byte(dirty), 0o644); err != nil {
		t.Fatal(err)
	}

	commit, refs, err := captureClosure(root, []string{"model.py"}, "srun-abc")
	if err != nil {
		t.Fatal(err)
	}
	head := gitRev(t, root, "HEAD")
	if commit == head {
		t.Errorf("a dirty closure must capture a NEW commit, got HEAD %s", head)
	}
	// The side ref exists and points at the capture commit.
	if got := gitRev(t, root, "refs/metis/sweeps/srun-abc"); got != commit {
		t.Errorf("side ref = %s; want the capture commit %s", got, commit)
	}
	// The manifest's blob-hash resolves to the EXACT dirty bytes.
	if len(refs) != 1 || refs[0].Path != "model.py" {
		t.Fatalf("manifest = %+v; want one model.py ref", refs)
	}
	if bytes := gitCat(t, root, string(refs[0].BlobHash)); bytes != dirty {
		t.Errorf("cat-file of the captured blob = %q; want the dirty bytes %q", bytes, dirty)
	}
	// Recovery: git checkout the capture commit restores the dirty version.
	// (checkout the file from the commit into a clean state.)
	restore := exec.Command("git", "-C", root, "checkout", commit, "--", "model.py")
	if out, err := restore.CombinedOutput(); err != nil {
		t.Fatalf("recovery checkout: %v\n%s", err, out)
	}
	got, _ := os.ReadFile(src)
	if string(got) != dirty {
		t.Errorf("recovery restored %q; want the captured dirty bytes %q", got, dirty)
	}
}

// A CLEAN closure captures nothing — the commit is HEAD, no side ref written.
func TestCaptureClosure_CleanIsHead(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := gitInit(t)
	if err := os.WriteFile(filepath.Join(root, "model.py"), []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, root, "init")

	commit, refs, err := captureClosure(root, []string{"model.py"}, "srun-clean")
	if err != nil {
		t.Fatal(err)
	}
	if commit != gitRev(t, root, "HEAD") {
		t.Errorf("clean closure commit = %s; want HEAD", commit)
	}
	// The manifest still records the (path, blob-hash) pointer even when clean.
	if len(refs) != 1 || refs[0].Path != "model.py" {
		t.Errorf("clean manifest should still carry the pointer, got %+v", refs)
	}
	// No side ref for a clean capture.
	cmd := exec.Command("git", "-C", root, "rev-parse", "--verify", "-q", "refs/metis/sweeps/srun-clean")
	if err := cmd.Run(); err == nil {
		t.Error("a clean closure must not create a side ref")
	}
}

func gitRev(t *testing.T, dir, ref string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-parse", ref).Output()
	if err != nil {
		t.Fatalf("rev-parse %s: %v", ref, err)
	}
	return strings.TrimSpace(string(out))
}

func gitCat(t *testing.T, dir, hash string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "cat-file", "blob", hash).Output()
	if err != nil {
		t.Fatalf("cat-file %s: %v", hash, err)
	}
	return string(out)
}
