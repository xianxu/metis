package sampler

import (
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/shape"
)

// countSampler is a trivial Sampler (Init=plan[1,2,3], Ask emits once, Tell=+out,
// Done=sum) that also counts its method calls — the T6 driver-loop proof.
type countSampler struct{ asks, tells, dones int }

type countState struct {
	pts       []int
	sum, told int
}

func (c *countSampler) Init(Ctx) countState { return countState{pts: []int{1, 2, 3}} }
func (c *countSampler) Ask(s countState) ([]int, bool) {
	c.asks++
	if s.told >= len(s.pts) {
		return nil, true
	}
	return s.pts[s.told:], false
}
func (c *countSampler) Tell(s countState, _ int, out int) countState {
	c.tells++
	s.sum += out
	s.told++
	return s
}
func (c *countSampler) Done(s countState) int { c.dones++; return s.sum }

// stuckSampler violates the progress contract: Ask never reports done and never
// proposes a point — Run must fail loud, not hang.
type stuckSampler struct{}

func (stuckSampler) Init(Ctx) int             { return 0 }
func (stuckSampler) Ask(int) ([]int, bool)    { return nil, false } // empty batch, not done
func (stuckSampler) Tell(s int, _, o int) int { return s + o }
func (stuckSampler) Done(s int) int           { return s }

func TestRun_PanicsOnNonProgressingAsk(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Run did not panic on an empty-batch/not-done Ask (would hang forever)")
		}
	}()
	Run(Ctx{}, stuckSampler{}, func(p int) int { return p })
}

func TestRun_DrivesAskTellDone(t *testing.T) {
	c := &countSampler{}
	got := Run(Ctx{}, c, func(p int) int { return p }) // runPoint = identity
	if got != 6 {
		t.Fatalf("Run sum = %d, want 6", got)
	}
	if c.tells != 3 {
		t.Errorf("Tell called %d times, want 3", c.tells)
	}
	if c.dones != 1 {
		t.Errorf("Done called %d times, want 1", c.dones)
	}
	if c.asks != 2 { // one emitting the batch, one reporting done
		t.Errorf("Ask called %d times, want 2 (emit + done)", c.asks)
	}
}

// configPoint builds a fake config-point identified by its model free-param — a TAGGED
// sum bundle `{model: {}}` in With (what shape.Expand emits for `model: {$any: {...}}`),
// so familyOf reads it as the "train.model=<model>" family.
func configPoint(model string) shape.Point {
	return shape.Point{
		With:       map[string]map[string]any{"train": {"model": map[string]any{model: map[string]any{}}}},
		FreeParams: []shape.FreeParam{{Path: "train.model", Value: model}},
	}
}

// TestRun_NestedComposition is the load-bearing type-composition proof: the same
// generic Run drives all three levels nested — driver ⊃ sweeper ⊃ resample — with
// the types composing (resample R=MeanSE is the sweeper's O; sweeper R=Winner is
// the driver's O). rf scores 0.85/fold, logreg 0.75/fold → argmax-mean picks rf.
func TestRun_NestedComposition(t *testing.T) {
	ctx := Ctx{Seed: 42, Partition: "part-abc"}
	cfgRF, cfgLR := configPoint("rf"), configPoint("logreg")
	sweeper := GridConfigs{Points: []shape.Point{cfgRF, cfgLR}, Direction: "maximize", Select: experiment.Select{ArgmaxMean: &experiment.ArgmaxMean{}}}

	scoreOf := func(p shape.Point) float64 {
		if p.FreeParams[0].Value == "rf" {
			return 0.85
		}
		return 0.75
	}
	// sweeper's runPoint = run the inner resample for this config.
	sweeperRun := func(p shape.Point) MeanSE {
		return Run(ctx, FixedKFolds{K: 3}, func(FoldPoint) FoldOutcome { return FoldOutcome{Score: scoreOf(p)} })
	}
	// driver's runPoint = run the whole sweeper (R = SweepResult).
	driverRun := func(SinglePoint) SweepResult { return Run(ctx, sweeper, sweeperRun) }

	res := Run(ctx, SingleDriver{}, driverRun)
	winner := res.Ship // the cross-family ship pick: rf (0.85) beats logreg (0.75)

	if got := winner.Point.FreeParams[0].Value; got != "rf" {
		t.Fatalf("winner model = %v, want rf", got)
	}
	if winner.Score.Mean != 0.85 {
		t.Errorf("winner mean = %v, want 0.85", winner.Score.Mean)
	}
	if winner.Score.SE != 0 {
		t.Errorf("winner SE = %v, want 0 (constant score)", winner.Score.SE)
	}
	if winner.Seed != 42 {
		t.Errorf("winner seed = %d, want 42", winner.Seed)
	}
	if len(res.PerFamily) != 2 {
		t.Errorf("per-family count = %d, want 2 (rf|logreg)", len(res.PerFamily))
	}
	wantKeys := []string{"part-abc#fold0", "part-abc#fold1", "part-abc#fold2"}
	if len(winner.FoldKeys) != 3 {
		t.Fatalf("winner FoldKeys = %v, want %v", winner.FoldKeys, wantKeys)
	}
	for i, k := range wantKeys {
		if winner.FoldKeys[i] != k {
			t.Errorf("FoldKeys[%d] = %q, want %q", i, winner.FoldKeys[i], k)
		}
	}
}
