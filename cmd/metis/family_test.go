package main

import (
	"math"
	"testing"

	"github.com/xianxu/metis/pkg/ledger"
)

// FamilyEstimate groups OUTER rows by family (not exact config) and reduces the metric over
// the outer folds — the reduction AggregateView cannot do, because a family's winner differs
// across outer folds (rf md=4 in fold 0, rf md=8 in fold 1) → distinct free-params, same family.
func TestFamilyEstimate_GroupsByFamilyAcrossOuterFolds(t *testing.T) {
	o0, o1 := 0, 1
	outer := func(model string, md, of int, score float64) ledger.Row {
		return ledger.Row{
			FreeParams:      map[string]any{"train.model": model, "train.model." + model + ".max_depth": md},
			CodeFingerprint: "abc", PointAddr: model + "-md" + itoa(md) + "-o" + itoa(of),
			Level: "outer", OuterFold: ofPtr(of), Metrics: map[string]float64{"train.fold_score": score}, Status: "ok",
		}
	}
	l := ledger.Ledger{}
	// rf winner differs across outer folds (md4 then md8) — same family, must pool.
	l.Append(
		outer("rf", 4, o0, 0.80),
		outer("rf", 8, o1, 0.78),
		outer("gbm", 15, o0, 0.75),
		outer("gbm", 31, o1, 0.73),
	)
	// also drop an INNER row for rf — it must be IGNORED by FamilyEstimate (outer-only).
	f0 := 0
	l.Append(ledger.Row{FreeParams: map[string]any{"train.model": "rf"}, CodeFingerprint: "abc",
		PointAddr: "rf-inner", Level: "inner", Fold: &f0, OuterFold: &o0,
		Metrics: map[string]float64{"train.fold_score": 0.99}, Status: "ok"})

	familyOf := func(r ledger.Row) string { return "train.model=" + r.FreeParams["train.model"].(string) }
	est := FamilyEstimate(l, "train.fold_score", familyOf)

	if len(est) != 2 {
		t.Fatalf("want 2 families (rf, gbm), got %d: %v", len(est), est)
	}
	rf := est["train.model=rf"]
	if math.Abs(rf.Mean-0.79) > 1e-9 { // (0.80+0.78)/2, NOT contaminated by the 0.99 inner row
		t.Errorf("rf family mean=%v want 0.79 (pooled two outer folds, inner row ignored)", rf.Mean)
	}
	if rf.SE == 0 {
		t.Errorf("rf family SE should be non-zero over 2 outer folds")
	}
	if gbm := est["train.model=gbm"]; math.Abs(gbm.Mean-0.74) > 1e-9 {
		t.Errorf("gbm family mean=%v want 0.74", gbm.Mean)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

func ofPtr(i int) *int { return &i }
