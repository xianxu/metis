package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xianxu/metis/pkg/record"
)

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
	writeJSON(t, filepath.Join(stepDir, "reads.json"), readSet{ProjectRoot: root, Reads: []string{"model.py"}})
	writeJSON(t, filepath.Join(root, "runs", runID, "record.json"), record.RunRecord{
		RunID: runID, PointAddress: record.Hash(runID),
		Steps: []record.StepRecord{{StepID: "train"}},
	})

	man := sweepManifest{ShapeRunID: "srun-e2e", Points: []pointRun{{RunID: runID, Status: "ok"}}}
	o := runOpts{expPath: expPath, stepPath: []string{filepath.Join(root, "steps")}}
	if err := captureSweepCode(o, man); err != nil {
		t.Fatalf("captureSweepCode: %v", err)
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
