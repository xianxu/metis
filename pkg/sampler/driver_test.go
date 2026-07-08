package sampler

import (
	"testing"

	"github.com/xianxu/metis/pkg/shape"
)

func TestSingleDriver_PassesSweeperWinnerThrough(t *testing.T) {
	want := Winner{
		Point:    shape.Point{FreeParams: []shape.FreeParam{{Path: "train.model", Value: "rf"}}},
		Seed:     42,
		FoldKeys: []string{"P#fold0", "P#fold1"},
		Score:    MeanSE{Mean: 0.83},
	}
	calls := 0
	// runPoint = the sweeper; it must run exactly once (driver:single).
	got := Run(Ctx{Seed: 42}, SingleDriver{}, func(SinglePoint) Winner {
		calls++
		return want
	})
	if calls != 1 {
		t.Errorf("sweeper ran %d times, want exactly 1 (driver:single)", calls)
	}
	if got.Point.FreeParams[0].Value != "rf" || got.Seed != 42 || got.Score.Mean != 0.83 {
		t.Errorf("driver Done = %+v, want passthrough of the sweeper winner", got)
	}
}

func TestSingleDriver_AskOnceThenDone(t *testing.T) {
	d := SingleDriver{}
	s := d.Init(Ctx{})
	if batch, done := d.Ask(s); done || len(batch) != 1 {
		t.Fatalf("first Ask = (%d, done=%v), want (1, false)", len(batch), done)
	}
	s = d.Tell(s, SinglePoint{}, Winner{Seed: 1})
	if _, done := d.Ask(s); !done {
		t.Error("Ask after tell: done=false, want true")
	}
}
