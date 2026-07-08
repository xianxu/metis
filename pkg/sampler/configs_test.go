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
	if w.FreeParams[0].Value != "rf" {
		t.Fatalf("winner = %v, want rf (argmax-mean)", w.FreeParams[0].Value)
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

func TestGridConfigs_MinimizeDirection(t *testing.T) {
	cfgA, cfgB := configPoint("a"), configPoint("b")
	g := GridConfigs{Points: []shape.Point{cfgA, cfgB}, Direction: "minimize", Select: "argmax-mean"}
	scores := map[string]MeanSE{"a": meanSE(0.30), "b": meanSE(0.20)}
	w := Run(Ctx{}, g, func(p shape.Point) MeanSE { return scores[p.FreeParams[0].Value.(string)] })
	if w.FreeParams[0].Value != "b" {
		t.Errorf("minimize winner = %v, want b (lowest)", w.FreeParams[0].Value)
	}
}

func TestGridConfigs_DeterministicTieBreak(t *testing.T) {
	// Two configs with identical means → the FIRST in expansion order wins.
	cfgA, cfgB := configPoint("first"), configPoint("second")
	g := GridConfigs{Points: []shape.Point{cfgA, cfgB}, Direction: "maximize"}
	w := Run(Ctx{}, g, func(shape.Point) MeanSE { return meanSE(0.80) })
	if w.FreeParams[0].Value != "first" {
		t.Errorf("tie winner = %v, want first (earliest)", w.FreeParams[0].Value)
	}
}
