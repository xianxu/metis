package sampler

import (
	"math"
	"testing"
)

func TestFixedKFolds_InitEnumeratesPartitionFolds(t *testing.T) {
	s := FixedKFolds{K: 5}.Init(Ctx{Partition: "P", Seed: 7})
	if len(s.points) != 5 {
		t.Fatalf("Init made %d fold-points, want 5", len(s.points))
	}
	for i, p := range s.points {
		if p.Partition != "P" || p.Idx != i {
			t.Errorf("point %d = %+v, want {P %d}", i, p, i)
		}
	}
	// Ask emits all 5 once, then done.
	batch, done := FixedKFolds{K: 5}.Ask(s)
	if done || len(batch) != 5 {
		t.Fatalf("first Ask = (%d points, done=%v), want (5, false)", len(batch), done)
	}
}

func TestFixedKFolds_AskDoneAfterAllTold(t *testing.T) {
	f := FixedKFolds{K: 3}
	s := f.Init(Ctx{Partition: "P"})
	batch, _ := f.Ask(s)
	for _, p := range batch {
		s = f.Tell(s, p, 0.5)
	}
	if _, done := f.Ask(s); !done {
		t.Error("Ask after all told: done=false, want true")
	}
}

func TestFixedKFolds_DoneReducesToMeanSE(t *testing.T) {
	// run through the generic loop; runPoint scores fold i as 0.80 + 0.01*i.
	got := Run(Ctx{Partition: "P"}, FixedKFolds{K: 5}, func(fp FoldPoint) float64 {
		return 0.80 + 0.01*float64(fp.Idx)
	})
	// scores 0.80..0.84 → mean 0.82.
	if math.Abs(got.Mean-0.82) > 1e-12 {
		t.Errorf("mean = %v, want 0.82", got.Mean)
	}
	if len(got.ToldSet) != 5 || got.ToldSet[0] != "P#fold0" || got.ToldSet[4] != "P#fold4" {
		t.Errorf("told-set = %v, want the 5 sorted fold addrs", got.ToldSet)
	}
}

func TestFixedKFolds_TellOrderIndependent(t *testing.T) {
	f := FixedKFolds{K: 3}
	fwd := f.Init(Ctx{Partition: "P"})
	for i := 0; i < 3; i++ {
		fwd = f.Tell(fwd, FoldPoint{Partition: "P", Idx: i}, float64(i))
	}
	rev := f.Init(Ctx{Partition: "P"})
	for i := 2; i >= 0; i-- {
		rev = f.Tell(rev, FoldPoint{Partition: "P", Idx: i}, float64(i))
	}
	if a, b := f.Done(fwd), f.Done(rev); a.Mean != b.Mean || a.SE != b.SE {
		t.Errorf("Done depends on tell order: %+v vs %+v", a, b)
	}
}
