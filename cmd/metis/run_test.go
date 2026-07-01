package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
)

// TestRunExperiment_EndToEnd exercises the REAL subprocess executor: it runs the
// run-echo fixture (two test/echo steps) through cmd/metis, which spawns the
// process-level fake step (testdata/steps/test/echo) via os/exec, and asserts the
// ledger is written (runs/<id>/run.json) and the `## Runs` log is appended. The
// fixture is copied into a temp dir first so the run artifacts and the ## Runs
// append never touch the committed testdata/.
func TestRunExperiment_EndToEnd(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "testdata", "experiment", "run-echo.md")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	expPath := filepath.Join(dir, "run-echo.md")
	if err := os.WriteFile(expPath, b, 0o644); err != nil {
		t.Fatal(err)
	}

	run, err := runExperiment(runOpts{
		expPath:  expPath,
		runID:    "run-001",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		out:      io.Discard,
	})
	if err != nil {
		t.Fatalf("runExperiment: %v", err)
	}
	if run.Status != "ok" {
		t.Fatalf("status = %q; want ok", run.Status)
	}

	// runs/run-001/run.json written, parseable, and matching the #Run shape.
	rb, err := os.ReadFile(filepath.Join(dir, "runs", "run-001", "run.json"))
	if err != nil {
		t.Fatalf("read run.json: %v", err)
	}
	var got experiment.Run
	if err := json.Unmarshal(rb, &got); err != nil {
		t.Fatalf("parse run.json: %v", err)
	}
	if got.ID != "run-001" || got.Experiment != "run-echo" || got.Seed != 7 || got.Status != "ok" {
		t.Errorf("run.json header wrong: %+v", got)
	}
	if got.Metrics["echoed"] != 1 {
		t.Errorf("metrics = %v; want echoed=1", got.Metrics)
	}
	if len(got.Artifacts) == 0 {
		t.Errorf("no artifacts recorded in run.json")
	}

	// `## Runs` log appended to the experiment.
	updated, err := os.ReadFile(expPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "- run-001 — ok") {
		t.Errorf("`## Runs` line not appended:\n%s", updated)
	}
}

// TestRunExperiment_RejectsInvalidAtRunTime is the execution-time enforcement
// test: a semantically-invalid experiment (a cycle — shape-valid, so CUE accepts
// it) is rejected by `metis run` BEFORE any step runs, closing the SHAPE-only gap
// M1 deferred. No ledger and no `## Runs` line are written for a rejected run.
func TestRunExperiment_RejectsInvalidAtRunTime(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "testdata", "experiment", "invalid-cycle.md")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	expPath := filepath.Join(dir, "invalid-cycle.md")
	if err := os.WriteFile(expPath, b, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = runExperiment(runOpts{
		expPath:  expPath,
		runID:    "run-001",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		out:      io.Discard,
	})
	if err == nil {
		t.Fatal("runExperiment accepted a cyclic experiment; want a validation error")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error = %q; want it to mention the cycle", err)
	}
	// No ledger written and the source untouched (no ## Runs bullet appended).
	if _, statErr := os.Stat(filepath.Join(dir, "runs")); statErr == nil {
		t.Error("a runs/ dir was created for a rejected experiment; want none")
	}
	after, err := os.ReadFile(expPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(after), "- run-001") {
		t.Errorf("a ## Runs line was appended for a rejected experiment:\n%s", after)
	}
}
