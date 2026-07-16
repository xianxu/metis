package sampler

import (
	"sort"
	"testing"

	"github.com/xianxu/metis/pkg/shape"
)

func TestSingleDriver_PassesSweeperResultThrough(t *testing.T) {
	want := SweepResult{
		Ship: Winner{
			Point:    shape.Point{FreeParams: []shape.FreeParam{{Path: "train.model", Value: "rf"}}},
			Seed:     42,
			FoldKeys: []string{"P#fold0", "P#fold1"},
			Score:    MeanSE{Mean: 0.83},
			Family:   "train.model=rf",
		},
		PerFamily: map[string]Winner{"train.model=rf": {Score: MeanSE{Mean: 0.83}}},
	}
	calls := 0
	// runPoint = the sweeper; it must run exactly once (driver:single).
	got := Run(Ctx{Seed: 42}, SingleDriver{}, func(SinglePoint) SweepResult {
		calls++
		return want
	}, SeqExec[SinglePoint, SweepResult], nil)
	if calls != 1 {
		t.Errorf("sweeper ran %d times, want exactly 1 (driver:single)", calls)
	}
	if got.Ship.Point.FreeParams[0].Value != "rf" || got.Ship.Seed != 42 || got.Ship.Score.Mean != 0.83 {
		t.Errorf("driver Done = %+v, want passthrough of the sweeper result", got)
	}
	if len(got.PerFamily) != 1 {
		t.Errorf("per-family passthrough dropped: %+v", got.PerFamily)
	}
}

func TestSingleDriver_AskOnceThenDone(t *testing.T) {
	d := SingleDriver{}
	s := d.Init(Ctx{})
	if batch, done := d.Ask(s); done || len(batch) != 1 {
		t.Fatalf("first Ask = (%d, done=%v), want (1, false)", len(batch), done)
	}
	s = d.Tell(s, SinglePoint{}, SweepResult{Ship: Winner{Seed: 1}})
	if _, done := d.Ask(s); !done {
		t.Error("Ask after tell: done=false, want true")
	}
}

func TestCVDriver_RunsSweeperPerOuterFoldAndAggregates(t *testing.T) {
	// metis#23: the runPoint = the sealed sweeper + refit-and-score, returning one outer
	// fold's honest held-out score. The driver must run it once per outer fold and
	// aggregate the k scores → the honest procedure estimate (mean ± SE).
	var seen []int
	got := Run(Ctx{Seed: 1}, CVDriver{K: 3}, func(p OuterFoldPoint) float64 {
		seen = append(seen, p.Idx)
		return float64(p.Idx) // outer scores 0,1,2 → mean 1.0
	}, SeqExec[OuterFoldPoint, float64], nil)
	if len(seen) != 3 {
		t.Fatalf("sweeper ran %d times, want 3 (one per outer fold)", len(seen))
	}
	sort.Ints(seen)
	for i, v := range seen {
		if v != i {
			t.Errorf("outer fold idx[%d] = %d, want %d", i, v, i)
		}
	}
	if got.Mean != 1.0 {
		t.Errorf("Done mean = %v, want 1.0 (Aggregate of 0,1,2 — the honest estimate)", got.Mean)
	}
	if len(got.ToldSet) != 3 {
		t.Errorf("ToldSet has %d addrs, want 3 (distinct outer folds)", len(got.ToldSet))
	}
}

func TestCVDriver_AsksAllOuterFoldsOnceThenDone(t *testing.T) {
	d := CVDriver{K: 3}
	s := d.Init(Ctx{})
	batch, done := d.Ask(s)
	if done || len(batch) != 3 {
		t.Fatalf("first Ask = (%d, done=%v), want (3, false)", len(batch), done)
	}
	for _, p := range batch {
		s = d.Tell(s, p, float64(p.Idx))
	}
	if _, done := d.Ask(s); !done {
		t.Error("Ask after all outer folds told: done=false, want true")
	}
}

func TestCVDriver_ZeroKIsDoneImmediately(t *testing.T) {
	// k=0 must not spin the Run loop (empty non-done batch → panic guard). Done → empty MeanSE.
	got := Run(Ctx{}, CVDriver{K: 0}, func(OuterFoldPoint) float64 { return 1 }, SeqExec[OuterFoldPoint, float64], nil)
	if got.Mean != 0 || len(got.ToldSet) != 0 {
		t.Errorf("k=0 Done = %+v, want zero MeanSE", got)
	}
}
