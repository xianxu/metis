package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
)

// TestRunExperiment_EndToEnd exercises the REAL subprocess executor: it runs the
// run-echo fixture (two test/echo steps) through cmd/metis, which spawns the
// process-level fake step (testdata/steps/test/echo) via os/exec, and asserts the
// run record is written (runs/<id>/run.json) and the experiment .md is left immutable
// (#13). The fixture is copied into a temp dir first so the run artifacts never touch
// the committed testdata/.
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
		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
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
	// Exact artifact set: each step's echoed.json, step-qualified under runs/<id>/,
	// and NOTHING else — metrics.json (the metrics channel) and with.json (the
	// input) must not leak into the artifact list.
	wantArtifacts := []string{"first/echoed.json", "second/echoed.json"}
	if !reflect.DeepEqual(got.Artifacts, wantArtifacts) {
		t.Errorf("artifacts = %v; want exactly %v", got.Artifacts, wantArtifacts)
	}

	// The experiment file is IMMUTABLE input — a run must leave it byte-for-byte unchanged
	// (#13). Run output lives in runs/<id>/{run,record}.json, never in the config.
	updated, err := os.ReadFile(expPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(updated) != string(b) {
		t.Errorf("run mutated the experiment .md (must be immutable input); after:\n%s", updated)
	}
}

// TestRunExperiment_RelativePath is the regression test for the relative-path
// bug: invoked as a user actually would — cd into the workspace, pass a BARE
// relative filename — the run must still execute end-to-end. The absolute
// t.TempDir() path in TestRunExperiment_EndToEnd masked this: unless runDir is
// absolutized, the injected METIS_STEP_DIR is relative and the child (whose cwd
// IS the step dir) resolves $METIS_STEP_DIR/with.json under itself and fails. We
// assert the step's declared output artifact (echoed.json) actually exists.
func TestRunExperiment_RelativePath(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "testdata", "experiment", "run-echo.md")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "run-echo.md"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir) // run from the workspace, like a real invocation

	run, err := runExperiment(runOpts{
		expPath:  "run-echo.md", // RELATIVE — the normal invocation
		runID:    "run-rel",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
		out:      io.Discard,
	})
	if err != nil {
		t.Fatalf("runExperiment with a relative path: %v", err)
	}
	if run.Status != "ok" {
		t.Fatalf("status = %q; want ok", run.Status)
	}
	echoed := filepath.Join(dir, "runs", "run-rel", "first", "echoed.json")
	if _, err := os.Stat(echoed); err != nil {
		t.Fatalf("step artifact %s not written (relative-path resolution broken): %v", echoed, err)
	}
}

// TestRunExperiment_FailedStepStillWritesLedger exercises the ledger-on-failure
// branch: a step that exits non-zero (via `with: {fail: true}` in the run-fail
// fixture) must still produce runs/<id>/run.json with status "failed" and a
// `## Runs` bullet — every attempt that began execution is recorded — while
// runExperiment surfaces the error. Fixture copied into t.TempDir() first.
func TestRunExperiment_FailedStepStillWritesLedger(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "testdata", "experiment", "run-fail.md")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	expPath := filepath.Join(dir, "run-fail.md")
	if err := os.WriteFile(expPath, b, 0o644); err != nil {
		t.Fatal(err)
	}

	run, err := runExperiment(runOpts{
		expPath:  expPath,
		runID:    "run-001",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
		out:      io.Discard,
	})
	if err == nil {
		t.Fatal("runExperiment: want an error from the failing step, got nil")
	}
	if run.Status != "failed" {
		t.Errorf("returned run status = %q; want failed", run.Status)
	}

	// runs/run-001/run.json written with status=failed.
	rb, err := os.ReadFile(filepath.Join(dir, "runs", "run-001", "run.json"))
	if err != nil {
		t.Fatalf("read run.json (failed run should still be recorded): %v", err)
	}
	var got experiment.Run
	if err := json.Unmarshal(rb, &got); err != nil {
		t.Fatalf("parse run.json: %v", err)
	}
	if got.ID != "run-001" || got.Experiment != "run-fail" || got.Status != "failed" {
		t.Errorf("run.json wrong: %+v", got)
	}

	// #13: a failed run is recorded in run.json/record.json (asserted above) — the config .md
	// stays byte-for-byte immutable input (the fixture already carries a `## Runs` heading, so
	// this asserts equality, not absence).
	updated, err := os.ReadFile(expPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(updated) != string(b) {
		t.Errorf("failed run mutated the config .md (must be immutable input):\n%s", updated)
	}
}

func TestRunExperimentActivityRunSuccessPublishesAfterRequiredArtifacts(t *testing.T) {
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

	var runEvents []activityEvent
	var publishErrs []error
	_, err = runExperiment(runOpts{
		expPath:  expPath,
		runID:    "run-activity",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
		out:      io.Discard,
		activity: func(ev activityEvent) {
			if ev.Kind != activityRunSuccess {
				return
			}
			runEvents = append(runEvents, ev)
			for _, name := range []string{"run.json", "record.json"} {
				if _, statErr := os.Stat(filepath.Join(dir, "runs", ev.RunID, name)); statErr != nil {
					publishErrs = append(publishErrs, statErr)
				}
			}
		},
	})
	if err != nil {
		t.Fatalf("runExperiment: %v", err)
	}
	if len(publishErrs) > 0 {
		t.Fatalf("run-success event arrived before required artifacts existed: %v", publishErrs)
	}
	if len(runEvents) != 1 {
		t.Fatalf("run-success events = %d; want 1 (%+v)", len(runEvents), runEvents)
	}
	if runEvents[0].RunID != "run-activity" {
		t.Fatalf("run event = %+v; want runID run-activity", runEvents[0])
	}
	if runEvents[0].Role != runRoleNone {
		t.Fatalf("plain run event role = %q; want no role/ineligible", runEvents[0].Role)
	}
}

func TestRunExperimentActivityRunSuccessNotPublishedForFailedExecution(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "testdata", "experiment", "run-fail.md")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	expPath := filepath.Join(dir, "run-fail.md")
	if err := os.WriteFile(expPath, b, 0o644); err != nil {
		t.Fatal(err)
	}

	var runEvents []activityEvent
	_, err = runExperiment(runOpts{
		expPath:  expPath,
		runID:    "run-failed-activity",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
		out:      io.Discard,
		activity: func(ev activityEvent) {
			if ev.Kind == activityRunSuccess {
				runEvents = append(runEvents, ev)
			}
		},
	})
	if err == nil {
		t.Fatal("runExperiment: want failing step error")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "runs", "run-failed-activity", "run.json")); statErr != nil {
		t.Fatalf("failed run should still write run.json: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "runs", "run-failed-activity", "record.json")); statErr != nil {
		t.Fatalf("failed run should still write record.json: %v", statErr)
	}
	if len(runEvents) != 0 {
		t.Fatalf("failed execution emitted run-success events: %+v", runEvents)
	}
}

func TestRunExperimentActivityRunSuccessNotPublishedWhenRecordPersistenceFails(t *testing.T) {
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
	if err := os.MkdirAll(filepath.Join(dir, "runs", "run-record-fails", "record.json"), 0o755); err != nil {
		t.Fatal(err)
	}

	var runEvents []activityEvent
	_, err = runExperiment(runOpts{
		expPath:  expPath,
		runID:    "run-record-fails",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
		out:      io.Discard,
		activity: func(ev activityEvent) {
			if ev.Kind == activityRunSuccess {
				runEvents = append(runEvents, ev)
			}
		},
	})
	if err == nil {
		t.Fatal("runExperiment: want record persistence error")
	}
	if len(runEvents) != 0 {
		t.Fatalf("record persistence failure emitted run-success events: %+v", runEvents)
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
		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
		out:      io.Discard,
	})
	if err == nil {
		t.Fatal("runExperiment accepted a cyclic experiment; want a validation error")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error = %q; want it to mention the cycle", err)
	}
	// A rejected run writes no runs/ dir and leaves the config .md untouched (#13 — the
	// config is immutable input in every case, rejected or not).
	if _, statErr := os.Stat(filepath.Join(dir, "runs")); statErr == nil {
		t.Error("a runs/ dir was created for a rejected experiment; want none")
	}
	after, err := os.ReadFile(expPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(b) {
		t.Errorf("a rejected experiment's config .md was mutated (must be immutable input):\n%s", after)
	}
}
