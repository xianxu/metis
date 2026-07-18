package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/metis/pkg/record"
)

// --- promoteDrift unit tests (pure core, fake hasher) ---

// fakeHasher mirrors gitBlobHashes' BATCH semantics faithfully (close-review finding): any
// unknown path fails the WHOLE call — per-path recovery is promoteDrift's job, not the hasher's.
func fakeHasher(m map[string]record.Hash) func(string, []string) (map[string]record.Hash, error) {
	return func(repo string, paths []string) (map[string]record.Hash, error) {
		out := map[string]record.Hash{}
		for _, p := range paths {
			h, ok := m[repo+":"+p]
			if !ok {
				return nil, fmt.Errorf("fatal: could not hash %s", p)
			}
			out[p] = h
		}
		return out, nil
	}
}

func recWith(fp string, refs ...record.CodeRef) record.RunRecord {
	return record.RunRecord{
		CodeFingerprint: record.Hash(fp),
		Steps:           []record.StepRecord{{Code: record.CodeManifest{Commit: "cafe1234", D: refs}}},
	}
}

func TestPromoteDrift_CleanTree(t *testing.T) {
	recs := map[string]record.RunRecord{"a": recWith("FP",
		record.CodeRef{Repo: "/r", Path: "code.py", BlobHash: "h1"})}
	drifted, commit, checked := promoteDrift(recs, "FP", fakeHasher(map[string]record.Hash{"/r:code.py": "h1"}))
	if !checked || len(drifted) != 0 || commit != "cafe1234" {
		t.Errorf("clean tree: drifted=%v commit=%s checked=%v", drifted, commit, checked)
	}
}

func TestPromoteDrift_EditAndMissingDetected(t *testing.T) {
	recs := map[string]record.RunRecord{"a": recWith("FP",
		record.CodeRef{Repo: "/r", Path: "code.py", BlobHash: "h1"},
		record.CodeRef{Repo: "/r", Path: "gone.py", BlobHash: "h2"})}
	drifted, _, checked := promoteDrift(recs, "FP", fakeHasher(map[string]record.Hash{"/r:code.py": "hX"}))
	if !checked || len(drifted) != 2 {
		t.Fatalf("want 2 drifted, got %v (checked=%v)", drifted, checked)
	}
	// batch fails (gone.py) → per-path retry: code.py still gets its REAL new hash (the
	// message must not lie about unchanged-but-batched siblings), gone.py carries the error.
	if drifted[0].Path != "code.py" || drifted[0].New != "hX" || drifted[0].Err != "" {
		t.Errorf("edited sibling must verify via per-path retry: %+v", drifted[0])
	}
	if drifted[1].Path != "gone.py" || drifted[1].New != "" || drifted[1].Err == "" {
		t.Errorf("missing path must carry the hasher error, not swallow it: %+v", drifted[1])
	}
}

func TestPromoteDrift_LegacyNoDIsUnchecked(t *testing.T) {
	recs := map[string]record.RunRecord{"a": recWith("FP")} // no D refs
	_, _, checked := promoteDrift(recs, "FP", fakeHasher(nil))
	if checked {
		t.Error("no D closure must report checked=false (warn-and-proceed, never block)")
	}
	// wrong-cohort records are also "nothing to compare"
	recs2 := map[string]record.RunRecord{"a": recWith("OTHER",
		record.CodeRef{Repo: "/r", Path: "x", BlobHash: "h"})}
	if _, _, checked := promoteDrift(recs2, "FP", fakeHasher(nil)); checked {
		t.Error("records of a different cohort must not be compared")
	}
}

// --- the guard through the REAL promote path (fixture e2e) ---

// writeGuardRecord plants runs/<addr>/record.json for the harness cohort "cf" whose D
// closure is ONE real file in a temp "repo" dir; returns the repo dir + file path.
func writeGuardRecord(t *testing.T, dir, addr string) (repoDir, codePath string) {
	t.Helper()
	repoDir = filepath.Join(dir, "coderepo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	codePath = filepath.Join(repoDir, "train.py")
	if err := os.WriteFile(codePath, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "util.py"), []byte("helpers\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := gitBlobHashes(repoDir, []string{"train.py", "util.py"})
	if err != nil {
		t.Fatalf("gitBlobHashes: %v", err)
	}
	rec := recWith("cf",
		record.CodeRef{Repo: repoDir, Path: "train.py", BlobHash: h["train.py"]},
		record.CodeRef{Repo: repoDir, Path: "util.py", BlobHash: h["util.py"]})
	b, _ := json.Marshal(rec)
	rdir := filepath.Join(dir, "runs", addr)
	if err := os.MkdirAll(rdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rdir, "record.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	return repoDir, codePath
}

// TestPromoteGuard_RefusesDriftAndRoundTrips (metis#53 Done-when): sweep fixture → edit a
// closure file → promote REFUSES naming the path; restoring the captured content (the
// checkout hint's effect) re-promotes clean; unchanged tree = no false positive.
func TestPromoteGuard_RefusesDriftAndRoundTrips(t *testing.T) {
	dir := t.TempDir()
	shapePath := writeSelectLedger(t, dir, taggedShipShapeForSelect, true)
	_, codePath := writeGuardRecord(t, dir, "i-lr1-0")

	promote := func() (string, error) {
		var out strings.Builder
		err := runSelect(selectOpts{shapePath: shapePath, best: true, promote: true,
			exec: foldFakeExec{}, git: fakeGitProbe{name: "metis", sha: "sha"}, now: fixedNow(), out: &out})
		return out.String(), err
	}

	// (a) unchanged tree: clean promote, no refusal
	if out, err := promote(); err != nil {
		t.Fatalf("unchanged tree must promote clean: %v\n%s", err, out)
	}

	// (b) drift: edit the closure file → REFUSE naming the path + the restore hint
	if err := os.WriteFile(codePath, []byte("EDITED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := promote()
	if err == nil {
		t.Fatal("drifted tree must REFUSE the promote")
	}
	for _, want := range []string{"train.py", "cafe1234"[:8], "--no-fingerprint-check", "NOT the selected cohort"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal must mention %q; got:\n%v", want, err)
		}
	}

	// (c) override proceeds LOUDLY
	var out strings.Builder
	if err := runSelect(selectOpts{shapePath: shapePath, best: true, promote: true, noFPCheck: true,
		exec: foldFakeExec{}, git: fakeGitProbe{name: "metis", sha: "sha"}, now: fixedNow(), out: &out}); err != nil {
		t.Fatalf("--no-fingerprint-check must proceed: %v", err)
	}
	if !strings.Contains(out.String(), "OVERRIDDEN") {
		t.Errorf("override must be loud; got:\n%s", out.String())
	}

	// (d) the checkout hint's round trip: restore captured content → clean again
	if err := os.WriteFile(codePath, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out2, err := promote(); err != nil {
		t.Fatalf("restored tree must promote clean again: %v\n%s", err, out2)
	}
}

// TestPromoteGuard_PointPathGuardedToo: the --point promote path runs the same guard.
func TestPromoteGuard_PointPathGuardedToo(t *testing.T) {
	dir := t.TempDir()
	shapePath := writeSelectLedger(t, dir, taggedShipShapeForSelect, true)
	_, codePath := writeGuardRecord(t, dir, "i-lr1-0")
	if err := os.WriteFile(codePath, []byte("EDITED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	err := runSelect(selectOpts{shapePath: shapePath, point: "i-lr1", promote: true,
		exec: foldFakeExec{}, git: fakeGitProbe{name: "metis", sha: "sha"}, now: fixedNow(), out: &out})
	if err == nil || !strings.Contains(err.Error(), "train.py") {
		t.Errorf("--point promote must refuse on drift naming the path; got %v", err)
	}
}

// TestPromoteGuard_LegacyCohortWarnsAndProceeds: no records with D → warn line, promote runs.
func TestPromoteGuard_LegacyCohortWarnsAndProceeds(t *testing.T) {
	dir := t.TempDir()
	shapePath := writeSelectLedger(t, dir, taggedShipShapeForSelect, true) // rows carry cf, no records on disk
	var out strings.Builder
	if err := runSelect(selectOpts{shapePath: shapePath, best: true, promote: true,
		exec: foldFakeExec{}, git: fakeGitProbe{name: "metis", sha: "sha"}, now: fixedNow(), out: &out}); err != nil {
		t.Fatalf("legacy cohort must warn-and-proceed: %v", err)
	}
	if !strings.Contains(out.String(), "nothing to compare") {
		t.Errorf("legacy cohort needs the loud nothing-to-compare line; got:\n%s", out.String())
	}
}

// TestPromoteGuard_DeletedFileDoesNotPoisonSiblings (close-review Important): deleting ONE
// closure file must name ONLY that file — the batch hash-object failure is retried per-path,
// so the unchanged sibling still verifies and the refusal doesn't lie.
func TestPromoteGuard_DeletedFileDoesNotPoisonSiblings(t *testing.T) {
	dir := t.TempDir()
	shapePath := writeSelectLedger(t, dir, taggedShipShapeForSelect, true)
	repoDir, _ := writeGuardRecord(t, dir, "i-lr1-0")
	if err := os.Remove(filepath.Join(repoDir, "util.py")); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	err := runSelect(selectOpts{shapePath: shapePath, best: true, promote: true,
		exec: foldFakeExec{}, git: fakeGitProbe{name: "metis", sha: "sha"}, now: fixedNow(), out: &out})
	if err == nil {
		t.Fatal("deleted closure file must refuse the promote")
	}
	msg := err.Error()
	if !strings.Contains(msg, "util.py") {
		t.Errorf("refusal must name the deleted file:\n%s", msg)
	}
	if strings.Contains(msg, "train.py") {
		t.Errorf("unchanged sibling must NOT read as drifted (batch poisoning):\n%s", msg)
	}
}
