package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xianxu/metis/pkg/record"
)

func TestBuildD_MapsReadsToCodeRefs(t *testing.T) {
	// metis#11: buildD is multi-root — reads grouped by repo → repo-qualified, sorted D.
	roots := map[string][]string{
		"/abs/metis":  {"metis/model.py", "metis/io.py"},
		"/abs/kbench": {"titanic/features.py"},
	}
	hasher := func(repo, p string) (record.Hash, error) {
		return record.Hash("blob:" + repo + ":" + p), nil
	}
	d, err := buildD(roots, hasher)
	if err != nil {
		t.Fatal(err)
	}
	// Sorted by (repo, path): kbench/features, metis/io, metis/model.
	if len(d) != 3 {
		t.Fatalf("buildD = %+v", d)
	}
	if d[0].Repo != "/abs/kbench" || d[0].Path != "titanic/features.py" {
		t.Errorf("first ref should be the kbench file (repo-qualified): %+v", d[0])
	}
	if d[1].Repo != "/abs/metis" || d[1].Path != "metis/io.py" || d[1].BlobHash != "blob:/abs/metis:metis/io.py" {
		t.Errorf("metis ref wrong: %+v", d[1])
	}
}

func TestBuildD_PropagatesHasherError(t *testing.T) {
	hasher := func(string, string) (record.Hash, error) { return "", fmt.Errorf("boom") }
	if _, err := buildD(map[string][]string{"/r": {"x.py"}}, hasher); err == nil {
		t.Error("buildD must propagate a hasher error (a D file that can't be hashed)")
	}
}

func TestLoadReadSet(t *testing.T) {
	dir := t.TempDir()
	body := `{"roots":{"/abs/metis":["metis/io.py"],"/abs/kbench":["titanic/features.py"]},"used_site_packages":true}`
	if err := os.WriteFile(filepath.Join(dir, "reads.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	rs, err := loadReadSet(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs.Roots) != 2 || rs.Roots["/abs/metis"][0] != "metis/io.py" ||
		rs.Roots["/abs/kbench"][0] != "titanic/features.py" || !rs.UsedSitePackages {
		t.Errorf("loadReadSet = %+v", rs)
	}
}

// A legacy v1 reads.json (project_root/reads) must FAIL LOUD — never silently unmarshal to
// an empty Roots → empty D → a vacuous K_pre-only false HIT (metis#11's lockstep guard).
func TestLoadReadSet_RejectsLegacyV1(t *testing.T) {
	dir := t.TempDir()
	body := `{"project_root":"/abs/metis","reads":["metis/io.py"],"used_site_packages":true}`
	if err := os.WriteFile(filepath.Join(dir, "reads.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadReadSet(dir); err == nil {
		t.Error("a legacy v1 reads.json must error, not yield an empty (false-HIT) read-set")
	}
}

func TestLoadReadSet_AbsentIsEmpty(t *testing.T) {
	// A step that ran without the sensor (or wrote nothing) → an empty read-set, not
	// an error (the caller treats an empty D as a K_pre-only cache entry).
	rs, err := loadReadSet(t.TempDir())
	if err != nil {
		t.Fatalf("absent reads.json should not error: %v", err)
	}
	if len(rs.Roots) != 0 {
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
		for _, paths := range rs.Roots { // metis#11: reads grouped by repo root
			for _, r := range paths {
				if r == p {
					return true
				}
			}
		}
		return false
	}
	if !has("metis/io.py") || !has("metis/steps/train.py") {
		t.Errorf("sensor missed first-party code; roots = %v", rs.Roots)
	}
	if !rs.UsedSitePackages {
		t.Errorf("train imports pandas/sklearn → used_site_packages should be true; got %+v", rs)
	}
	// No class-1 data / stdlib should leak into D.
	for _, paths := range rs.Roots {
		for _, r := range paths {
			if filepath.Ext(r) != ".py" && filepath.Base(r) != "uv.lock" {
				t.Errorf("unexpected non-code path in D: %q", r)
			}
		}
	}
}

// The sensor's EXCLUSION filters are load-bearing: a read under METIS_RUN_DIR (a
// step's own outputs + the upstream artifacts it reads — which change every run) or
// a stdlib/site-packages read must NOT enter D, else at M3 every step MISSes forever.
// This drives metis.trace._classify directly to pin the filter contract.
func TestSensor_ExcludesRunDirAndStdlib(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not on PATH")
	}
	root := repoRoot(t)
	script := `
import os, json, metis.trace as t
r = os.path.dirname(os.path.dirname(os.path.abspath(t.__file__)))  # the metis repo root
os.environ["METIS_RUN_DIR"] = os.path.join(r, "runs")
t._classify(os.path.join(r, "metis", "io.py"))            # first-party source  -> KEEP
t._classify(os.path.join(r, "runs", "run1", "out.bin"))   # run-dir output      -> DROP
t._classify(os.path.join(r, "runs", "run1", "folds.json"))# upstream artifact    -> DROP
t._classify("/usr/lib/python3.12/json/__init__.py")       # stdlib              -> DROP
t._classify(os.path.join(r, ".venv", "x", "pandas.py"))   # venv/site-packages  -> DROP (flag)
print(json.dumps(sorted(t._roots.get(r, []))))            # only the metis root's kept reads
`
	cmd := exec.Command("uv", "run", "--project", root, "python", "-c", script)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("driving metis.trace._classify failed: %v", err)
	}
	var kept []string
	if err := json.Unmarshal([]byte(trimNL(string(out))), &kept); err != nil {
		t.Fatalf("parse classify output %q: %v", out, err)
	}
	if len(kept) != 1 || kept[0] != "metis/io.py" {
		t.Errorf("filters wrong: D = %v; want only [metis/io.py] (run-dir/stdlib/venv excluded)", kept)
	}
}
