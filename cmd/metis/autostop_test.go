package main

import (
	"math"
	"testing"
)

// TestShouldStop_LosersOnly is the metis#66 M2 safety table: the predictive rule must stop a
// clear loser, NEVER a would-be winner, and hold its fire while the estimate is too green — the
// "never truncate a would-be winner" invariant the issue is most exposed on.
func TestShouldStop_LosersOnly(t *testing.T) {
	cases := []struct {
		name      string
		scores    []float64
		k         int
		incumbent float64
		dir       string
		want      bool
	}{
		// maximize (higher is better), incumbent 0.80
		{"clear loser stops", []float64{0.70, 0.705}, 4, 0.80, "maximize", true},
		{"clear winner never stops", []float64{0.90, 0.905}, 4, 0.80, "maximize", false},
		{"borderline (near incumbent) does not stop", []float64{0.79, 0.80}, 4, 0.80, "maximize", false},
		{"n=1 never stops (no spread)", []float64{0.10}, 4, 0.80, "maximize", false},
		{"already full k never stops", []float64{0.70, 0.70, 0.70, 0.70}, 4, 0.80, "maximize", false},
		{"loser with a bit more data stays stopped", []float64{0.68, 0.71, 0.69}, 5, 0.80, "maximize", true},
		{"high-variance loser near incumbent is spared (conservative)", []float64{0.60, 0.95}, 4, 0.80, "maximize", false},
		// minimize (lower is better), incumbent 0.20 (e.g. an error/loss metric)
		{"minimize: high loser stops", []float64{0.40, 0.405}, 4, 0.20, "minimize", true},
		{"minimize: low winner never stops", []float64{0.10, 0.105}, 4, 0.20, "minimize", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldStop(c.scores, c.k, c.incumbent, c.dir); got != c.want {
				t.Errorf("shouldStop(%v, k=%d, inc=%.2f, %s) = %v, want %v", c.scores, c.k, c.incumbent, c.dir, got, c.want)
			}
		})
	}
}

// TestShouldStop_MonotoneInIncumbent: a higher incumbent (for maximize) can only make stopping
// MORE likely — the rule must never flip from stop→run as the bar rises.
func TestShouldStop_MonotoneInIncumbent(t *testing.T) {
	scores := []float64{0.75, 0.76}
	stoppedAtLow := shouldStop(scores, 5, 0.70, "maximize")
	stoppedAtHigh := shouldStop(scores, 5, 0.95, "maximize")
	if stoppedAtLow && !stoppedAtHigh {
		t.Errorf("raising the incumbent 0.70→0.95 flipped stop→run — non-monotone")
	}
	if !stoppedAtHigh {
		t.Errorf("a family well below a high incumbent must stop")
	}
}

// TestTCrit pins the one-sided 95% t table + the z fallback (the bound's critical values are
// load-bearing for the safety property, so they're asserted numerically).
func TestTCrit(t *testing.T) {
	want := map[int]float64{1: 6.314, 2: 2.920, 3: 2.353, 5: 2.015, 10: 1.812}
	for df, w := range want {
		if got := tCrit(df); math.Abs(got-w) > 1e-3 {
			t.Errorf("tCrit(%d) = %.4f, want %.4f", df, got, w)
		}
	}
	if got := tCrit(50); math.Abs(got-1.645) > 1e-3 {
		t.Errorf("tCrit(50) = %.4f, want the normal z=1.645", got)
	}
	if got := tCrit(0); math.Abs(got-1.645) > 1e-3 {
		t.Errorf("tCrit(0) = %.4f, want the z fallback", got)
	}
}
