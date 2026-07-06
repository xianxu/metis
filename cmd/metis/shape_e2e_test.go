package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A singleton experiment-shape (every knob pinned) runs exactly like a v0 experiment
// through `metis run` — the #Experiment = #ExperimentShape & all-singleton collapse,
// made runnable. Uses the no-uv test/echo steps.
func TestRunShape_SingletonRunsLikeExperiment(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := filepath.Join(ws, "shape.md")
	md := `---
type: experiment-shape
id: singleton
seed: 5
status: active
sweep:
  sampler: grid
  objective: {metric: echoed, direction: maximize}
steps:
  - id: prep
    uses: test/echo
    with: {k: 5}
  - id: train
    uses: test/echo
    needs: [prep]
    with: {model: logreg}
---
`
	if err := os.WriteFile(expPath, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	run, err := runExperiment(runOpts{
		expPath:  expPath,
		runID:    "s1",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "sha", dirty: false},
		out:      io.Discard,
	})
	if err != nil {
		t.Fatalf("singleton shape should run like an experiment: %v", err)
	}
	if run.Status != "ok" {
		t.Errorf("run status = %q; want ok", run.Status)
	}
	// The resolved record shows the pinned singleton config (knob→score).
	body, _ := os.ReadFile(expPath)
	if !strings.Contains(string(body), "prep.k=5") {
		t.Errorf("## Runs should show the resolved singleton config; got:\n%s", body)
	}
}

// A multi-point shape is a sweep — refused here with a pointer to metis#7 (the sweep
// driver isn't this issue), NOT run inline.
func TestRunShape_MultiPointPointsToSweeper(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := filepath.Join(ws, "sweep.md")
	md := `---
type: experiment-shape
id: multi
seed: 5
status: active
sweep:
  sampler: grid
  objective: {metric: echoed, direction: maximize}
steps:
  - id: train
    uses: test/echo
    with: {model: {$any: [logreg, rf]}}
---
`
	if err := os.WriteFile(expPath, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := runExperiment(runOpts{
		expPath:  expPath,
		runID:    "m1",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "sha", dirty: false},
		out:      io.Discard,
	})
	if err == nil {
		t.Fatal("a multi-point shape must be refused (the sweep driver is metis#7), not run inline")
	}
	if !strings.Contains(err.Error(), "metis#7") || !strings.Contains(err.Error(), "2 points") {
		t.Errorf("error should name the point count + point to metis#7; got: %v", err)
	}
}
