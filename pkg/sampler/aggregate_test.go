package sampler

import (
	"math"
	"reflect"
	"testing"
)

// TestAggregate_Complexity: Aggregate also means the per-fold complexity (metis#19),
// and reports HasComplexity only when every fold carried a measured value — so the M2
// guard can tell "measured 0" from "not measured".
func TestAggregate_Complexity(t *testing.T) {
	got := Aggregate([]FoldScore{
		{Addr: "a", Score: 0.8, Complexity: 16, HasComplexity: true},
		{Addr: "b", Score: 0.9, Complexity: 14, HasComplexity: true},
	})
	if math.Abs(got.Mean-0.85) > 1e-12 {
		t.Errorf("mean = %v, want 0.85", got.Mean)
	}
	if !got.HasComplexity || math.Abs(got.MeanComplexity-15) > 1e-12 {
		t.Errorf("complexity = %v (has=%v), want 15,true", got.MeanComplexity, got.HasComplexity)
	}
	// Folds without a measured complexity (M1 wires 0 e2e) → HasComplexity false.
	z := Aggregate([]FoldScore{{Addr: "a", Score: 0.8}, {Addr: "b", Score: 0.9}})
	if z.HasComplexity {
		t.Errorf("HasComplexity = true for unmeasured folds, want false")
	}
}

func TestAggregate_MeanSE(t *testing.T) {
	// scores 0.80, 0.82, 0.78, 0.84, 0.76 → mean 0.80; sample sd 0.0316227766…;
	// SE = sd/√5.
	scores := []FoldScore{
		{Addr: "p#fold0", Score: 0.80},
		{Addr: "p#fold1", Score: 0.82},
		{Addr: "p#fold2", Score: 0.78},
		{Addr: "p#fold3", Score: 0.84},
		{Addr: "p#fold4", Score: 0.76},
	}
	got := Aggregate(scores)
	if math.Abs(got.Mean-0.80) > 1e-12 {
		t.Errorf("mean = %v, want 0.80", got.Mean)
	}
	// sample variance = Σ(x-mean)²/(n-1) = (0+0.0004+0.0004+0.0016+0.0016)/4 = 0.001
	wantSD := math.Sqrt(0.001)
	wantSE := wantSD / math.Sqrt(5)
	if math.Abs(got.SE-wantSE) > 1e-12 {
		t.Errorf("SE = %v, want %v", got.SE, wantSE)
	}
	wantSet := []string{"p#fold0", "p#fold1", "p#fold2", "p#fold3", "p#fold4"}
	if !reflect.DeepEqual(got.ToldSet, wantSet) {
		t.Errorf("ToldSet = %v, want %v", got.ToldSet, wantSet)
	}
}

func TestAggregate_OrderIndependent(t *testing.T) {
	a := Aggregate([]FoldScore{{Addr: "b", Score: 1}, {Addr: "a", Score: 3}, {Addr: "c", Score: 2}})
	b := Aggregate([]FoldScore{{Addr: "c", Score: 2}, {Addr: "b", Score: 1}, {Addr: "a", Score: 3}})
	if a.Mean != b.Mean || a.SE != b.SE {
		t.Errorf("permutation changed (mean,SE): %v vs %v", a, b)
	}
	if !reflect.DeepEqual(a.ToldSet, b.ToldSet) {
		t.Errorf("ToldSet not order-independent: %v vs %v", a.ToldSet, b.ToldSet)
	}
	if !reflect.DeepEqual(a.ToldSet, []string{"a", "b", "c"}) {
		t.Errorf("ToldSet = %v, want sorted [a b c]", a.ToldSet)
	}
}

func TestAggregate_SingletonAndEmpty(t *testing.T) {
	one := Aggregate([]FoldScore{{Addr: "x", Score: 0.9}})
	if one.Mean != 0.9 || one.SE != 0 {
		t.Errorf("singleton = %+v, want mean 0.9 SE 0", one)
	}
	empty := Aggregate(nil)
	if empty.Mean != 0 || empty.SE != 0 || empty.ToldSet != nil {
		t.Errorf("empty = %+v, want zero", empty)
	}
}
