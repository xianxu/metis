package sampler

import (
	"testing"

	"github.com/xianxu/metis/pkg/shape"
)

// meanSE builds a MeanSE with a given mean + told-set (SE irrelevant to selection).
func meanSE(mean float64, told ...string) MeanSE {
	return MeanSE{Mean: mean, ToldSet: told}
}

func TestGridConfigs_SelectsArgmaxMean(t *testing.T) {
	cfgA, cfgB, cfgC := configPoint("logreg"), configPoint("rf"), configPoint("gbm")
	g := GridConfigs{
		Points:    []shape.Point{cfgA, cfgB, cfgC},
		Direction: "maximize",
		Select:    "argmax-mean",
	}
	scores := map[string]MeanSE{
		"logreg": meanSE(0.79, "k0", "k1"),
		"rf":     meanSE(0.83, "r0", "r1"), // best
		"gbm":    meanSE(0.81, "g0", "g1"),
	}
	w := Run(Ctx{Seed: 42}, g, func(p shape.Point) MeanSE {
		return scores[p.FreeParams[0].Value.(string)]
	})
	if w.Point.FreeParams[0].Value != "rf" {
		t.Fatalf("winner = %v, want rf (argmax-mean)", w.Point.FreeParams[0].Value)
	}
	if w.Seed != 42 {
		t.Errorf("winner seed = %d, want 42", w.Seed)
	}
	if w.Score.Mean != 0.83 {
		t.Errorf("winner score mean = %v, want 0.83", w.Score.Mean)
	}
	// Winner carries the winning config's fold addresses (its resample told-set).
	if len(w.FoldKeys) != 2 || w.FoldKeys[0] != "r0" || w.FoldKeys[1] != "r1" {
		t.Errorf("winner FoldKeys = %v, want [r0 r1]", w.FoldKeys)
	}
}

// The Winner carries the WHOLE resolved config point (its per-step `With` + free-params) —
// so ship/promote rebuild the exact experiment from the run-keys DIRECTLY, not by re-expanding
// the grid and matching free-params (metis#18 M1a-5 T19: reconstructable winner run-keys).
func TestGridConfigs_WinnerCarriesResolvedPoint(t *testing.T) {
	cfg := shape.Point{
		With: map[string]map[string]any{
			"features": {"features": []any{"title"}},
			"train":    {"model": "rf"},
		},
		FreeParams: []shape.FreeParam{{Path: "train.model", Value: "rf"}},
	}
	g := GridConfigs{Points: []shape.Point{cfg}, Direction: "maximize", Select: "argmax-mean"}
	w := Run(Ctx{Seed: 7}, g, func(shape.Point) MeanSE { return meanSE(0.9, "k0") })
	if w.Point.With["train"]["model"] != "rf" || w.Point.With["features"] == nil {
		t.Errorf("winner Point.With must carry the FULL resolved config for direct rebuild; got %v", w.Point.With)
	}
	if len(w.Point.FreeParams) != 1 || w.Point.FreeParams[0].Value != "rf" {
		t.Errorf("winner Point.FreeParams = %v, want [train.model=rf]", w.Point.FreeParams)
	}
}

func TestGridConfigs_MinimizeDirection(t *testing.T) {
	cfgA, cfgB := configPoint("a"), configPoint("b")
	g := GridConfigs{Points: []shape.Point{cfgA, cfgB}, Direction: "minimize", Select: "argmax-mean"}
	scores := map[string]MeanSE{"a": meanSE(0.30), "b": meanSE(0.20)}
	w := Run(Ctx{}, g, func(p shape.Point) MeanSE { return scores[p.FreeParams[0].Value.(string)] })
	if w.Point.FreeParams[0].Value != "b" {
		t.Errorf("minimize winner = %v, want b (lowest)", w.Point.FreeParams[0].Value)
	}
}

func TestGridConfigs_DeterministicTieBreak(t *testing.T) {
	// Two configs with identical means → the FIRST in expansion order wins.
	cfgA, cfgB := configPoint("first"), configPoint("second")
	g := GridConfigs{Points: []shape.Point{cfgA, cfgB}, Direction: "maximize"}
	w := Run(Ctx{}, g, func(shape.Point) MeanSE { return meanSE(0.80) })
	if w.Point.FreeParams[0].Value != "first" {
		t.Errorf("tie winner = %v, want first (earliest)", w.Point.FreeParams[0].Value)
	}
}
