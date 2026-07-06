package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/cache"
	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/record"
)

// The cheap-sweeps payoff (metis#2): a second identical run HITs every step (no
// subprocess), and changing one downstream knob HITs the shared upstream while
// re-running only the changed step. Uses the no-uv test/echo steps.
func TestCache_CheapSweeps(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH (cache re-hash uses git hash-object)")
	}
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := filepath.Join(ws, "exp.md")
	write := func(trainModel string) {
		md := `---
type: experiment
id: sweep
seed: 5
status: active
steps:
  - id: prep
    uses: test/echo
    with: {k: 5}
  - id: train
    uses: test/echo
    needs: [prep]
    with: {model: ` + trainModel + `}
---
`
		if err := os.WriteFile(expPath, []byte(md), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	run := func(runID string) string {
		var out strings.Builder
		_, err := runExperiment(runOpts{
			expPath:  expPath,
			runID:    runID,
			stepPath: []string{filepath.Join(root, "testdata", "steps")},
			now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
			git:      fakeGitProbe{name: "metis", sha: "sha", dirty: false},
			cache:    true,
			out:      &out,
		})
		if err != nil {
			t.Fatalf("run %s: %v", runID, err)
		}
		return out.String()
	}

	// Run 1 (cold): both steps MISS (no cache hit markers).
	write("logreg")
	out1 := run("r1")
	if strings.Contains(out1, "cache hit") {
		t.Errorf("cold run should have no cache hits; got:\n%s", out1)
	}

	// Run 2 (identical): both steps HIT — no subprocess.
	out2 := run("r2")
	if !hitFor(out2, "prep") || !hitFor(out2, "train") {
		t.Errorf("identical re-run should HIT every step; got:\n%s", out2)
	}

	// Run 3 (change the downstream train knob): prep (unchanged config + upstream)
	// still HITs; train MISSes (its resolved-with changed → different K_pre).
	write("rf")
	out3 := run("r3")
	if !hitFor(out3, "prep") {
		t.Errorf("prep (unchanged) should still HIT after a downstream knob change; got:\n%s", out3)
	}
	if hitFor(out3, "train") {
		t.Errorf("train (knob changed) must MISS and re-run; got:\n%s", out3)
	}
}

func hitFor(out, step string) bool {
	return strings.Contains(out, "step "+step+" (cache hit)")
}

// TestCache_ToyPipelineHitsOnRerun drives the REAL uv/Python toy pipeline twice with
// caching (uv-gated): the second run must HIT every step (the sensor's real reads.json
// → D re-hashes clean), materialize the parquet outputs from the CAS, and reproduce the
// same cv_score — proving the cache end-to-end with real artifacts, not just test/echo.
func TestCache_ToyPipelineHitsOnRerun(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not on PATH; skipping the real-pipeline cache e2e")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := repoRoot(t)
	ws := t.TempDir()
	expDir := filepath.Join(ws, "experiment")
	if err := os.MkdirAll(expDir, 0o755); err != nil {
		t.Fatal(err)
	}
	copyFile(t, filepath.Join(root, "testdata", "experiment", "toy-pipeline.md"),
		filepath.Join(expDir, "toy-pipeline.md"))
	copyDir(t, filepath.Join(root, "testdata", "dataset", "toy"), filepath.Join(ws, "dataset", "toy"))

	run := func(runID string) (string, float64) {
		var out strings.Builder
		r, err := runExperiment(runOpts{
			expPath:  filepath.Join(expDir, "toy-pipeline.md"),
			runID:    runID,
			stepPath: []string{filepath.Join(root, "steps")},
			now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
			git:      fakeGitProbe{name: "metis", sha: "sha", dirty: false},
			cache:    true,
			out:      &out,
		})
		if err != nil {
			t.Fatalf("run %s: %v", runID, err)
		}
		return out.String(), r.Metrics["cv_score"]
	}

	out1, cv1 := run("c1")
	if strings.Contains(out1, "cache hit") {
		t.Errorf("cold run should not HIT; got:\n%s", out1)
	}
	if cv1 <= 0.5 {
		t.Fatalf("cold run cv_score = %v; want a real score", cv1)
	}

	out2, cv2 := run("c2")
	for _, step := range []string{"split", "train", "predict"} {
		if !hitFor(out2, step) {
			t.Errorf("re-run should HIT step %s; got:\n%s", step, out2)
		}
	}
	if cv2 != cv1 {
		t.Errorf("cached re-run cv_score = %v; want %v (reproduced from cache)", cv2, cv1)
	}
	// The materialized submission output exists (downstream consumers can read it).
	if _, err := os.Stat(filepath.Join(expDir, "runs", "c2", "predict", "predictions.csv")); err != nil {
		t.Errorf("cached run should materialize predictions.csv: %v", err)
	}
}

// An immutable-leaf step (a pinned external fetch) HITs on the K_pre match alone —
// the v1 leaf policy. isImmutableLeaf recognizes the `with: {cache: {leaf: immutable}}`
// marker.
func TestCache_ImmutableLeafMarker(t *testing.T) {
	leaf := experiment.Step{ID: "get", Uses: "kaggle/get-data",
		With: map[string]any{"cache": map[string]any{"leaf": "immutable"}}}
	if !isImmutableLeaf(leaf) {
		t.Error("a step marked cache.leaf=immutable should be an immutable leaf")
	}
	plain := experiment.Step{ID: "train", Uses: "metis/train", With: map[string]any{"model": "rf"}}
	if isImmutableLeaf(plain) {
		t.Error("an unmarked step must not be an immutable leaf")
	}
	if isImmutableLeaf(experiment.Step{ID: "x"}) {
		t.Error("a step with no with must not be an immutable leaf")
	}
}

// The wipeable-cache contract (pkg/cas): wiping the CAS output blobs while keeping the
// index must NOT fail a run — a step whose index-hit output was wiped/evicted/corrupted
// recomputes. The design promises `rm -rf cas/` is safe; this pins it end-to-end.
func TestCache_WipedCASRecomputesNotFails(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := filepath.Join(ws, "exp.md")
	md := `---
type: experiment
id: wipe
seed: 5
status: active
steps:
  - id: prep
    uses: test/echo
    with: {k: 5}
---
`
	if err := os.WriteFile(expPath, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := runOpts{
		expPath:  expPath,
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "sha", dirty: false},
		cache:    true,
	}
	opts.runID, opts.out = "w1", io.Discard
	if _, err := runExperiment(opts); err != nil {
		t.Fatalf("cold run: %v", err)
	}
	// Wipe ONLY the CAS blobs, keeping the index (the exact "rm -rf cas/ is safe" case).
	if err := os.RemoveAll(filepath.Join(ws, ".metis-cache", "cas")); err != nil {
		t.Fatal(err)
	}
	// Re-run: the index still hits, but the output bytes are gone → must recompute,
	// not hard-fail.
	var out strings.Builder
	opts.runID, opts.out = "w2", &out
	if _, err := runExperiment(opts); err != nil {
		t.Fatalf("run after CAS wipe must recompute, not fail: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "recomputing") {
		t.Errorf("expected a recompute notice after CAS wipe; got:\n%s", out.String())
	}
}

// The cache's CORE soundness claim, locked through the real git-hash path: a stored
// non-empty D HITs while its files are unchanged and MISSes the moment a file in D is
// edited (byte-changed). The e2es can't cover this — CheapSweeps' test/echo steps
// write no reads.json (empty D → vacuous HIT) and the toy-pipeline test only re-runs
// identically — so a regression silently emptying D would false-HIT with green CI.
func TestCachingExecutor_RealDMissesOnSourceEdit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH (D re-hash uses git hash-object)")
	}
	root := t.TempDir()
	mustRun(t, root, "git", "init", "-q")
	src := filepath.Join(root, "src.py")
	if err := os.WriteFile(src, []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hashes, err := gitBlobHashes(root, []string{"src.py"})
	if err != nil {
		t.Fatal(err)
	}
	c := &cachingExecutor{}
	// metis#11: D is repo-qualified — the ref carries its repo root so isHit re-hashes it there.
	entry := cache.Entry{Kpre: "k", D: []record.CodeRef{{Repo: root, Path: "src.py", BlobHash: hashes["src.py"]}}}
	step := experiment.Step{ID: "s"} // NOT a leaf → D revalidation applies

	if !c.isHit(step, entry) {
		t.Fatal("unchanged D must HIT")
	}
	// Edit the source file in D → its blob-hash moves → the step must MISS.
	if err := os.WriteFile(src, []byte("x = 2  # edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if c.isHit(step, entry) {
		t.Error("a byte-changed file in D must MISS (D revalidation is the cache's core soundness)")
	}
}

// metis#11 — THE guarantee this issue exists for: a D spanning TWO repos (metis + a
// consumer like kbench) HITs while both are unchanged, and MISSes when the CONSUMER repo's
// file is edited. Before metis#11 the consumer's code never entered D, so editing it was a
// silent false-HIT (a sweep serving output computed by old consumer code).
func TestCachingExecutor_MultiRepoDMissesOnConsumerEdit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH (D re-hash uses git hash-object)")
	}
	metisRepo := t.TempDir()
	consumerRepo := t.TempDir()
	mustRun(t, metisRepo, "git", "init", "-q")
	mustRun(t, consumerRepo, "git", "init", "-q")
	if err := os.MkdirAll(filepath.Join(consumerRepo, "titanic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metisRepo, "io.py"), []byte("m = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	feat := filepath.Join(consumerRepo, "titanic", "features.py")
	if err := os.WriteFile(feat, []byte("def group_title(): return 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mh, _ := gitBlobHashes(metisRepo, []string{"io.py"})
	ch, _ := gitBlobHashes(consumerRepo, []string{"titanic/features.py"})

	c := &cachingExecutor{}
	entry := cache.Entry{Kpre: "k", D: []record.CodeRef{
		{Repo: metisRepo, Path: "io.py", BlobHash: mh["io.py"]},
		{Repo: consumerRepo, Path: "titanic/features.py", BlobHash: ch["titanic/features.py"]},
	}}
	step := experiment.Step{ID: "s"}

	if !c.isHit(step, entry) {
		t.Fatal("a two-repo D with both files unchanged must HIT")
	}
	// Edit ONLY the consumer repo's file → its blob-hash moves → the step must MISS.
	if err := os.WriteFile(feat, []byte("def group_title(): return 2  # edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if c.isHit(step, entry) {
		t.Error("editing the CONSUMER repo's code must MISS — the metis#11 cross-repo guarantee")
	}
}

func mustRun(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

// The leaf policy end-to-end from a YAML-parsed marker: isImmutableLeaf must see
// `with: {cache: {leaf: immutable}}` through the real experiment.Parse path (yaml.v3
// → map[string]any), not just a hand-built Step. A YAML-lib swap or a With type change
// would silently kill the leaf policy (a Done-when item) otherwise.
func TestCache_LeafPolicyFromParsedYAML(t *testing.T) {
	md := `---
type: experiment
id: leafy
seed: 1
status: active
steps:
  - id: get
    uses: kaggle/get-data
    with: {cache: {leaf: immutable}}
  - id: train
    uses: metis/train
    with: {model: rf}
---
`
	exp, err := experiment.Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]experiment.Step{}
	for _, s := range exp.Steps {
		byID[s.ID] = s
	}
	if !isImmutableLeaf(byID["get"]) {
		t.Errorf("parsed cache.leaf=immutable step should be an immutable leaf; with=%v", byID["get"].With)
	}
	if isImmutableLeaf(byID["train"]) {
		t.Error("an unmarked parsed step must not be an immutable leaf")
	}
}

// The immutable-leaf HIT path bypasses D re-validation entirely (HIT on K_pre alone):
// an entry whose D would MISS (a file that won't re-hash) still HITs for a leaf, but
// MISSes for a normal step. Pins the runner-level bypass, not just the marker predicate.
func TestCachingExecutor_ImmutableLeafBypassesDValidation(t *testing.T) {
	c := &cachingExecutor{}
	tmp := t.TempDir() // no git repo / no D file here → the ref can't re-hash → MISS (for a normal step)
	badEntry := cache.Entry{Kpre: "k", D: []record.CodeRef{{Repo: tmp, Path: "nope.py", BlobHash: "stale"}}}

	leaf := experiment.Step{ID: "get", With: map[string]any{"cache": map[string]any{"leaf": "immutable"}}}
	if !c.isHit(leaf, badEntry) {
		t.Error("an immutable leaf must HIT on K_pre alone, bypassing D re-validation")
	}
	plain := experiment.Step{ID: "train"}
	if c.isHit(plain, badEntry) {
		t.Error("a normal step must MISS when its D cannot re-hash clean")
	}
}

// A legacy (pre-#11) D ref with an empty repo root must be rejected (→ MISS), NOT hashed:
// `git -C "" hash-object` is a no-op that resolves against cwd (returns a hash), so relying
// on "git fails" would make HIT/MISS cwd-dependent. The explicit guard keeps it sound (#11).
func TestHashDByRepo_RejectsLegacyEmptyRepo(t *testing.T) {
	if _, err := hashDByRepo([]record.CodeRef{{Repo: "", Path: "metis/io.py", BlobHash: "h"}}); err == nil {
		t.Error("a D ref with an empty repo root must error (→ cwd-independent MISS), not hash against cwd")
	}
}
