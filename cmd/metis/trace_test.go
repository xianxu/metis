package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xianxu/metis/pkg/record"
)

func TestBuildD_MapsReadsToCodeRefs(t *testing.T) {
	reads := []string{"metis/io.py", "metis/model.py"}
	hasher := func(p string) (record.Hash, error) {
		return record.Hash("blob-" + p), nil
	}
	d, err := buildD(reads, hasher)
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 2 || d[0].Path != "metis/io.py" || d[0].BlobHash != "blob-metis/io.py" {
		t.Fatalf("buildD = %+v", d)
	}
}

func TestBuildD_PropagatesHasherError(t *testing.T) {
	hasher := func(string) (record.Hash, error) { return "", fmt.Errorf("boom") }
	if _, err := buildD([]string{"x.py"}, hasher); err == nil {
		t.Error("buildD must propagate a hasher error (a D file that can't be hashed)")
	}
}

func TestLoadReadSet(t *testing.T) {
	dir := t.TempDir()
	body := `{"project_root":"/abs/metis","reads":["metis/io.py"],"used_site_packages":true}`
	if err := os.WriteFile(filepath.Join(dir, "reads.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	rs, err := loadReadSet(dir)
	if err != nil {
		t.Fatal(err)
	}
	if rs.ProjectRoot != "/abs/metis" || len(rs.Reads) != 1 || rs.Reads[0] != "metis/io.py" || !rs.UsedSitePackages {
		t.Errorf("loadReadSet = %+v", rs)
	}
}

func TestLoadReadSet_AbsentIsEmpty(t *testing.T) {
	// A step that ran without the sensor (or wrote nothing) → an empty read-set, not
	// an error (the caller treats an empty D as a K_pre-only cache entry).
	rs, err := loadReadSet(t.TempDir())
	if err != nil {
		t.Fatalf("absent reads.json should not error: %v", err)
	}
	if len(rs.Reads) != 0 {
		t.Errorf("absent reads.json should yield an empty read-set, got %+v", rs)
	}
}

// gitBlobHashes must equal `git hash-object` for real files (git's blob-hash IS the
// content-hash the cache re-hashes to validate). Batches all paths in one call.
func TestGitBlobHashes_MatchesGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := repoRoot(t)
	paths := []string{"go.mod", "metis/io.py"}
	got, err := gitBlobHashes(root, paths)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		want, err := exec.Command("git", "-C", root, "hash-object", p).Output()
		if err != nil {
			t.Fatal(err)
		}
		if string(got[p]) != trimNL(string(want)) {
			t.Errorf("gitBlobHashes[%s] = %q; want %q", p, got[p], trimNL(string(want)))
		}
	}
}

func TestGitBlobHashes_EmptyIsNoop(t *testing.T) {
	m, err := gitBlobHashes(t.TempDir(), nil)
	if err != nil || len(m) != 0 {
		t.Errorf("gitBlobHashes(nil) = (%v, %v); want (empty, nil)", m, err)
	}
}

func trimNL(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

// The Python read-sensor records the first-party code closure (uv-gated). Runs the
// train step through `metis.trace`; even though it fails early (no step context),
// the sensor's finally-block writes reads.json with the metis code files.
func TestSensor_RecordsFirstPartyCodeReads(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not on PATH; skipping the Python read-sensor test")
	}
	root := repoRoot(t)
	stepDir := t.TempDir()
	cmd := exec.Command("uv", "run", "--project", root, "python", "-m", "metis.trace", "metis.steps.train")
	cmd.Env = append(os.Environ(), "METIS_STEP_DIR="+stepDir, "METIS_RUN_DIR="+filepath.Join(stepDir, "runs"))
	_ = cmd.Run() // expected to fail (no METIS_STEP_ID) — the sensor still writes reads.json

	rs, err := loadReadSet(stepDir)
	if err != nil {
		t.Fatalf("sensor did not write a loadable reads.json: %v", err)
	}
	has := func(p string) bool {
		for _, r := range rs.Reads {
			if r == p {
				return true
			}
		}
		return false
	}
	if !has("metis/io.py") || !has("metis/steps/train.py") {
		t.Errorf("sensor missed first-party code; reads = %v", rs.Reads)
	}
	if !rs.UsedSitePackages {
		t.Errorf("train imports pandas/sklearn → used_site_packages should be true; got %+v", rs)
	}
	// No class-1 data / stdlib should leak into D.
	for _, r := range rs.Reads {
		if filepath.Ext(r) != ".py" && filepath.Base(r) != "uv.lock" {
			t.Errorf("unexpected non-code path in D: %q", r)
		}
	}
}
