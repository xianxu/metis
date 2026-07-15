package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/metis/pkg/record"
)

// The single-run capture WIRING (not just the captureSingleRun helper): a dirty single
// `metis run` driven through the REAL runExperiment entrypoint must capture — exercising
// the `if !o.inSweep { captureSingleRun }` seam in runResolvedExperiment. Deleting that
// call site leaves the direct-helper tests green but must fail THIS one (the Done-when is
// about `metis run`, not the helper).
func TestRunExperiment_SingleRunCapturesViaWiring(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := gitInit(t)
	// An initial commit so HEAD exists (a real repo always has one; capture parents on HEAD).
	if err := os.WriteFile(filepath.Join(root, "README"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, root, "init")
	// A test/echo experiment at the repo root, untracked (dirty) — so even with an empty
	// Python read-set its spec .md is captured to a side ref (#14's spec hook).
	expPath := filepath.Join(root, "exp.md")
	expMD := "---\ntype: experiment\nid: e\nseed: 1\nstatus: active\nsteps:\n  - id: only\n    uses: test/echo\n    with: {a: 1}\n---\n"
	if err := os.WriteFile(expPath, []byte(expMD), 0o644); err != nil {
		t.Fatal(err)
	}
	runID := "single-wire"
	run, err := runExperiment(runOpts{
		expPath:  expPath,
		runID:    runID,
		stepPath: []string{filepath.Join(repoRoot(t), "testdata", "steps")},
		now:      fixedNow(),
		git:      gitCLI{},
		out:      io.Discard,
	})
	if err != nil {
		t.Fatalf("runExperiment: %v", err)
	}
	if run.Status != "ok" {
		t.Fatalf("status = %q; want ok", run.Status)
	}
	rb, _ := os.ReadFile(filepath.Join(root, "runs", runID, "record.json"))
	var rec record.RunRecord
	if err := json.Unmarshal(rb, &rec); err != nil {
		t.Fatal(err)
	}
	c := rec.Steps[0].Code
	if c.CaptureStatus != "captured" {
		t.Errorf("single run via runExperiment must capture (the wiring); status=%q", c.CaptureStatus)
	}
	if c.Commit == "" || gitRev(t, root, "refs/metis/runs/"+runID) != c.Commit {
		t.Errorf("refs/metis/runs/<id> must point at the captured commit; commit=%q", c.Commit)
	}
}

// captureSweepCode, given a sweep whose points' reads.json name a DIRTY first-party
// file, captures it to a side ref and backfills each point-record's CodeManifest with
// the (path, blob-hash) manifest + the captured commit. End-to-end over the sweep dir
// layout (runs/<id>/<step>/reads.json + runs/<id>/record.json).
func TestCaptureSweepCode_BackfillsCodeManifest(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := gitInit(t)
	// A tracked-then-dirtied first-party code file at the repo root.
	code := filepath.Join(root, "model.py")
	if err := os.WriteFile(code, []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, root, "init")
	dirty := "x = 2  # dirty\n"
	if err := os.WriteFile(code, []byte(dirty), 0o644); err != nil {
		t.Fatal(err)
	}

	// The experiment lives at the repo root; one point ran, producing a step reads.json
	// (naming model.py) and a record.json with one step.
	expPath := filepath.Join(root, "sweep.md")
	runID := "pt-1"
	stepDir := filepath.Join(root, "runs", runID, "train")
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(stepDir, "reads.json"), readSet{Roots: map[string][]string{root: {"model.py"}}})
	writeJSON(t, filepath.Join(root, "runs", runID, "record.json"), record.RunRecord{
		RunID: runID, PointAddress: record.Hash(runID),
		Steps: []record.StepRecord{{StepID: "train"}},
	})

	man := sweepManifest{ShapeRunID: "srun-e2e", Points: []pointRun{{RunID: runID, Status: "ok"}}}
	var out bytes.Buffer
	o := runOpts{expPath: expPath, stepPath: []string{filepath.Join(root, "steps")}, out: &out}
	if err := captureSweepCode(o, man); err != nil {
		t.Fatalf("captureSweepCode: %v", err)
	}

	// metis#39: a sweep (nested/--fast) run states the cohort it records under — the hash
	// the select guard later names must have scrolled by. Exactly once per sweep.
	if got := strings.Count(out.String(), "recording under code_fingerprint "); got != 1 {
		t.Errorf("sweep capture must print the cohort line exactly once, got %d:\n%s", got, out.String())
	}

	// The record's CodeManifest is now populated with D + a real commit.
	rb, _ := os.ReadFile(filepath.Join(root, "runs", runID, "record.json"))
	var rec record.RunRecord
	if err := json.Unmarshal(rb, &rec); err != nil {
		t.Fatal(err)
	}
	code0 := rec.Steps[0].Code
	if code0.Commit == "" {
		t.Error("CodeManifest.Commit must be populated after capture")
	}
	if len(code0.D) != 1 || code0.D[0].Path != "model.py" || code0.D[0].BlobHash == "" {
		t.Errorf("CodeManifest.D not populated with the closure pointer: %+v", code0.D)
	}
	// The captured commit is a real side-ref commit (dirty closure) whose blob is the
	// exact dirty bytes.
	if got := gitCat(t, root, string(code0.D[0].BlobHash)); got != dirty {
		t.Errorf("captured blob = %q; want the dirty bytes %q", got, dirty)
	}
	if gitRev(t, root, "refs/metis/sweeps/srun-e2e") != code0.Commit {
		t.Error("the side ref should point at the captured commit recorded in CodeManifest")
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// metis#14 heart-test 1+2: a SINGLE (non-sweep) run captures its code closure AND the
// run-spec .md to refs/metis/runs/<id>, backfilling the record with D + a recoverable
// commit + status "captured" — a dirty single run is reproducible (code + spec).
func TestCaptureSingleRun_CapturesCodeAndSpec(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := gitInit(t)
	code := filepath.Join(root, "model.py")
	if err := os.WriteFile(code, []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, root, "init")
	dirtyCode := "x = 2  # dirty\n"
	if err := os.WriteFile(code, []byte(dirtyCode), 0o644); err != nil {
		t.Fatal(err)
	}
	// The run-spec .md is a first-party input the Python read-set never sees — the #14
	// second hook. Here it's untracked (a brand-new dirty spec).
	expPath := filepath.Join(root, "exp.md")
	specBytes := "---\ntype: experiment\nid: e\n---\n# dirty spec\n"
	if err := os.WriteFile(expPath, []byte(specBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	runID := "run-1"
	stepDir := filepath.Join(root, "runs", runID, "train")
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(stepDir, "reads.json"), readSet{Roots: map[string][]string{root: {"model.py"}}})
	writeJSON(t, filepath.Join(root, "runs", runID, "record.json"), record.RunRecord{
		RunID: runID, Steps: []record.StepRecord{{StepID: "train"}},
	})

	var out bytes.Buffer
	o := runOpts{expPath: expPath, stepPath: []string{filepath.Join(root, "steps")}, out: &out}
	if err := captureSingleRun(o, runID); err != nil {
		t.Fatalf("captureSingleRun: %v", err)
	}

	// metis#39: a flat single run states its cohort too (same line as the sweep path).
	if !strings.Contains(out.String(), "recording under code_fingerprint ") {
		t.Errorf("single-run capture must print the cohort line:\n%s", out.String())
	}

	rb, _ := os.ReadFile(filepath.Join(root, "runs", runID, "record.json"))
	var rec record.RunRecord
	if err := json.Unmarshal(rb, &rec); err != nil {
		t.Fatal(err)
	}
	c := rec.Steps[0].Code
	if c.CaptureStatus != "captured" {
		t.Errorf("capture_status = %q; want captured", c.CaptureStatus)
	}
	if c.Commit == "" {
		t.Error("CodeManifest.Commit must be populated")
	}
	if gitRev(t, root, "refs/metis/runs/"+runID) != c.Commit {
		t.Error("refs/metis/runs/<id> must point at the captured commit")
	}
	blobs := map[string]record.Hash{}
	for _, ref := range c.D {
		blobs[ref.Path] = ref.BlobHash
	}
	if _, ok := blobs["model.py"]; !ok {
		t.Errorf("D missing the code file model.py: %+v", c.D)
	}
	if _, ok := blobs["exp.md"]; !ok {
		t.Errorf("D missing the run-spec exp.md — the #14 second hook: %+v", c.D)
	}
	if got := gitCat(t, root, string(blobs["model.py"])); got != dirtyCode {
		t.Errorf("model.py blob = %q; want the dirty bytes %q", got, dirtyCode)
	}
	if got := gitCat(t, root, string(blobs["exp.md"])); got != specBytes {
		t.Errorf("run-spec blob = %q; want the dirty spec bytes", got)
	}
}

// metis#14 heart-test 3: a run that CANNOT durably capture (here: a non-git dir) sets
// capture_status != "captured" AND emits a LOUD note — reproducibility is a promise; a
// broken one must be visible, never a silent success.
func TestCaptureSingleRun_LoudWhenUncaptured(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir() // deliberately NOT a git repo
	expPath := filepath.Join(dir, "exp.md")
	if err := os.WriteFile(expPath, []byte("# spec\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "code.py"), []byte("y = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runID := "run-x"
	stepDir := filepath.Join(dir, "runs", runID, "train")
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(stepDir, "reads.json"), readSet{Roots: map[string][]string{dir: {"code.py"}}})
	writeJSON(t, filepath.Join(dir, "runs", runID, "record.json"), record.RunRecord{
		RunID: runID, Steps: []record.StepRecord{{StepID: "train"}},
	})

	var out bytes.Buffer
	o := runOpts{expPath: expPath, stepPath: []string{filepath.Join(dir, "steps")}, out: &out}
	if err := captureSingleRun(o, runID); err != nil {
		t.Fatalf("captureSingleRun: %v", err)
	}
	rb, _ := os.ReadFile(filepath.Join(dir, "runs", runID, "record.json"))
	var rec record.RunRecord
	json.Unmarshal(rb, &rec)
	if rec.Steps[0].Code.CaptureStatus == "captured" {
		t.Errorf("a non-git run must not be 'captured'; got %q", rec.Steps[0].Code.CaptureStatus)
	}
	if !strings.Contains(out.String(), "warning") && !strings.Contains(out.String(), "note") {
		t.Errorf("an uncaptured run must emit a LOUD note; got: %q", out.String())
	}
}
