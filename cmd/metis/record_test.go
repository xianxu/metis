package main

import (
	"math"
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/record"
)

func sampleSteps() []experiment.StepRun {
	return []experiment.StepRun{
		{Step: experiment.Step{ID: "prep", Uses: "metis/cv-split", With: map[string]any{"k": 5}},
			Result: experiment.StepResult{Artifacts: []string{"folds.json"}}},
		{Step: experiment.Step{ID: "train", Uses: "metis/train", With: map[string]any{"model": "logreg"}},
			Result: experiment.StepResult{Metrics: map[string]float64{"acc": 0.9}}},
	}
}

// stepsOf extracts the full step list from executed StepRuns (the intended config the
// point-address mints from). In these tests every step ran, so allSteps == the runs.
func stepsOf(runs []experiment.StepRun) []experiment.Step {
	out := make([]experiment.Step, len(runs))
	for i, sr := range runs {
		out[i] = sr.Step
	}
	return out
}

func TestBuildRecord_MintsStablePointAddress(t *testing.T) {
	run := experiment.Run{ID: "r1", Experiment: "exp", Seed: 7, Started: "t0", Finished: "t1", Status: "ok"}
	steps := sampleSteps()
	oh := map[string]record.Hash{"prep": "h1", "train": "h2"}

	rec1, err := buildRecord(run, stepsOf(steps), steps, oh, "metis", "sha1", false)
	if err != nil {
		t.Fatal(err)
	}
	rec2, err := buildRecord(run, stepsOf(steps), steps, oh, "metis", "sha1", false)
	if err != nil {
		t.Fatal(err)
	}
	if rec1.PointAddress == "" || rec1.PointAddress != rec2.PointAddress {
		t.Errorf("point-address must be stable across identical runs: %q vs %q", rec1.PointAddress, rec2.PointAddress)
	}
	// Sensitive to the repo SHA (a determinant).
	rec3, _ := buildRecord(run, stepsOf(steps), steps, oh, "metis", "sha2", false)
	if rec3.PointAddress == rec1.PointAddress {
		t.Error("point-address must change with the repo SHA")
	}

	// Per-step records carry resolved config, output hash, code commit, metrics.
	if len(rec1.Steps) != 2 || rec1.Steps[0].StepID != "prep" || rec1.Steps[1].StepID != "train" {
		t.Fatalf("step records wrong/misordered: %+v", rec1.Steps)
	}
	if rec1.Steps[0].OutputHash != "h1" || rec1.Steps[0].Code.Commit != "sha1" {
		t.Errorf("step 0 provenance wrong: %+v", rec1.Steps[0])
	}
	if rec1.Steps[1].Metrics["acc"] != 0.9 {
		t.Errorf("step 1 metrics not carried: %+v", rec1.Steps[1])
	}
	if rec1.RepoSHAs["metis"] != "sha1" || rec1.Dirty {
		t.Errorf("repo provenance wrong: shas=%v dirty=%v", rec1.RepoSHAs, rec1.Dirty)
	}
	// Upstream/D/Deps are metis#2-populated slots — #3 leaves them empty.
	if len(rec1.Steps[0].Upstream) != 0 || len(rec1.Steps[0].Code.D) != 0 {
		t.Errorf("step 0 should leave Upstream/D empty (metis#2's slots): %+v", rec1.Steps[0])
	}
}

// buildRecord populates StepRecord.Upstream (the #3 slot #2 fills): each step's
// needs → the upstream steps' output-hashes, sorted (so K_pre is needs-order
// invariant). This is the DAG-wiring cache.Kpre depends on.
func TestBuildRecord_PopulatesUpstreamFromNeeds(t *testing.T) {
	run := experiment.Run{ID: "r", Experiment: "e", Seed: 1, Started: "t0", Status: "ok"}
	steps := []experiment.StepRun{
		{Step: experiment.Step{ID: "prep", Uses: "metis/cv-split"}},
		{Step: experiment.Step{ID: "train", Uses: "metis/train", Needs: []string{"prep"}}},
	}
	oh := map[string]record.Hash{"prep": "hp", "train": "ht"}
	rec, err := buildRecord(run, stepsOf(steps), steps, oh, "metis", "sha", false)
	if err != nil {
		t.Fatal(err)
	}
	// prep has no needs → empty upstream; train needs prep → [prep's output hash].
	if len(rec.Steps[0].Upstream) != 0 {
		t.Errorf("prep upstream = %v; want empty", rec.Steps[0].Upstream)
	}
	if len(rec.Steps[1].Upstream) != 1 || rec.Steps[1].Upstream[0] != "hp" {
		t.Errorf("train upstream = %v; want [hp] (prep's output hash)", rec.Steps[1].Upstream)
	}
}

func TestBuildRecord_PropagatesConfigError(t *testing.T) {
	run := experiment.Run{ID: "r", Seed: 0}
	steps := []experiment.StepRun{{Step: experiment.Step{ID: "s", With: map[string]any{"lr": math.Inf(1)}}}}
	if _, err := buildRecord(run, stepsOf(steps), steps, nil, "m", "sha", false); err == nil {
		t.Error("buildRecord must propagate the point-address error on non-finite config")
	}
}
