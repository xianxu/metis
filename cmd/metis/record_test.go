package main

import (
	"math"
	"strings"
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

func TestBuildRecord_MintsStablePointAddress(t *testing.T) {
	run := experiment.Run{ID: "r1", Experiment: "exp", Seed: 7, Started: "t0", Finished: "t1", Status: "ok"}
	steps := sampleSteps()
	oh := map[string]record.Hash{"prep": "h1", "train": "h2"}

	rec1, err := buildRecord(run, steps, oh, "metis", "sha1", false)
	if err != nil {
		t.Fatal(err)
	}
	rec2, err := buildRecord(run, steps, oh, "metis", "sha1", false)
	if err != nil {
		t.Fatal(err)
	}
	if rec1.PointAddress == "" || rec1.PointAddress != rec2.PointAddress {
		t.Errorf("point-address must be stable across identical runs: %q vs %q", rec1.PointAddress, rec2.PointAddress)
	}
	// Sensitive to the repo SHA (a determinant).
	rec3, _ := buildRecord(run, steps, oh, "metis", "sha2", false)
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

func TestBuildRecord_PropagatesConfigError(t *testing.T) {
	run := experiment.Run{ID: "r", Seed: 0}
	steps := []experiment.StepRun{{Step: experiment.Step{ID: "s", With: map[string]any{"lr": math.Inf(1)}}}}
	if _, err := buildRecord(run, steps, nil, "m", "sha", false); err == nil {
		t.Error("buildRecord must propagate the point-address error on non-finite config")
	}
}

func TestRecordSummary_RendersKnobToScore(t *testing.T) {
	rec := record.RunRecord{
		RunID: "r1", Status: "ok", Finished: "t1",
		Steps: []record.StepRecord{
			{StepID: "prep", With: map[string]any{"k": 5}},
			{StepID: "train", With: map[string]any{"model": "logreg"}, Metrics: map[string]float64{"cv_score": 0.81}},
		},
	}
	s := recordSummary(rec)
	for _, want := range []string{"r1", "ok", "prep.k=5", "train.model=logreg", "cv_score=0.81"} {
		if !strings.Contains(s, want) {
			t.Errorf("recordSummary = %q; missing %q", s, want)
		}
	}
}
