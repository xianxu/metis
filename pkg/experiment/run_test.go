package experiment

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

// fakeExecutor records the order steps are executed and returns canned results —
// so Runner.Run is exercised with NO subprocess (the ARCH-PURE line: orchestration
// is pure/thin, real execution lives behind the StepExecutor seam in cmd/metis).
type fakeExecutor struct {
	calls   []string
	results map[string]StepResult
	failOn  string
}

func (f *fakeExecutor) Execute(step Step, runDir string) (StepResult, error) {
	f.calls = append(f.calls, step.ID)
	if step.ID == f.failOn {
		return StepResult{}, fmt.Errorf("boom in %s", step.ID)
	}
	return f.results[step.ID], nil
}

func fixedClock(t time.Time) Clock { return func() time.Time { return t } }

// TestRunner_Run_OrderAndAssembly: a 2-step experiment (declared out of order)
// executes in dependency order, no subprocess, and assembles a Run with merged
// metrics, ordered artifacts, and injected-clock timestamps.
func TestRunner_Run_OrderAndAssembly(t *testing.T) {
	exp := Experiment{ID: "exp1", Seed: 7, Steps: []Step{
		{ID: "train", Uses: "metis/train", Needs: []string{"prep"}},
		{ID: "prep", Uses: "metis/cv-split"},
	}}
	fake := &fakeExecutor{results: map[string]StepResult{
		"prep":  {Artifacts: []string{"folds.json"}},
		"train": {Metrics: map[string]float64{"acc": 0.9}, Artifacts: []string{"model.pkl"}},
	}}
	clock := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	r := Runner{Exec: fake, Now: fixedClock(clock)}

	run, _, err := r.Run(exp, "run-001", "/runs/run-001")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !reflect.DeepEqual(fake.calls, []string{"prep", "train"}) {
		t.Fatalf("execution order = %v; want [prep train]", fake.calls)
	}
	if run.ID != "run-001" || run.Experiment != "exp1" || run.Seed != 7 {
		t.Errorf("run header wrong: %+v", run)
	}
	if run.Status != "ok" {
		t.Errorf("status = %q; want ok", run.Status)
	}
	if run.Metrics["acc"] != 0.9 {
		t.Errorf("metrics = %v; want acc=0.9", run.Metrics)
	}
	if !reflect.DeepEqual(run.Artifacts, []string{"folds.json", "model.pkl"}) {
		t.Errorf("artifacts = %v; want [folds.json model.pkl]", run.Artifacts)
	}
	if run.Started != "2026-07-01T12:00:00Z" || run.Finished != "2026-07-01T12:00:00Z" {
		t.Errorf("timestamps: started=%q finished=%q", run.Started, run.Finished)
	}
}

// TestRunner_Run_ReturnsPerStepResults: the runner retains per-step results in
// execution (topo) order — the breakdown a provenance record needs, which the flat
// Run merge discards. Each StepRun pairs the executed Step with its Result.
func TestRunner_Run_ReturnsPerStepResults(t *testing.T) {
	exp := Experiment{ID: "exp1", Seed: 7, Steps: []Step{
		{ID: "train", Uses: "metis/train", Needs: []string{"prep"}},
		{ID: "prep", Uses: "metis/cv-split", With: map[string]any{"k": 5}},
	}}
	fake := &fakeExecutor{results: map[string]StepResult{
		"prep":  {Artifacts: []string{"folds.json"}},
		"train": {Metrics: map[string]float64{"acc": 0.9}, Artifacts: []string{"model.pkl"}},
	}}
	r := Runner{Exec: fake, Now: fixedClock(time.Unix(0, 0).UTC())}

	_, steps, err := r.Run(exp, "run-001", "/runs/run-001")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("want 2 StepRuns, got %d", len(steps))
	}
	// Topo order: prep before train.
	if steps[0].Step.ID != "prep" || steps[1].Step.ID != "train" {
		t.Fatalf("StepRun order = [%s %s]; want [prep train]", steps[0].Step.ID, steps[1].Step.ID)
	}
	if steps[0].Step.With["k"] != 5 {
		t.Errorf("prep resolved With not retained: %v", steps[0].Step.With)
	}
	if steps[1].Result.Metrics["acc"] != 0.9 {
		t.Errorf("train per-step metric not retained: %v", steps[1].Result.Metrics)
	}
	if !reflect.DeepEqual(steps[0].Result.Artifacts, []string{"folds.json"}) {
		t.Errorf("prep artifacts = %v; want [folds.json]", steps[0].Result.Artifacts)
	}
}

// TestRunner_Run_StepFailure: a failing step stops the pipeline and records a
// "failed" Run; later steps never execute.
func TestRunner_Run_StepFailure(t *testing.T) {
	exp := Experiment{ID: "exp1", Steps: []Step{
		{ID: "a", Uses: "metis/a"},
		{ID: "b", Uses: "metis/b", Needs: []string{"a"}},
	}}
	fake := &fakeExecutor{failOn: "a", results: map[string]StepResult{}}
	r := Runner{Exec: fake, Now: fixedClock(time.Unix(0, 0).UTC())}

	run, _, err := r.Run(exp, "run-x", "d")
	if err == nil {
		t.Fatal("Run: want error from failing step, got nil")
	}
	if run.Status != "failed" {
		t.Errorf("status = %q; want failed", run.Status)
	}
	if !reflect.DeepEqual(fake.calls, []string{"a"}) {
		t.Errorf("calls = %v; pipeline should stop after the failing step", fake.calls)
	}
}
