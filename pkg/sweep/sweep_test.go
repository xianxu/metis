package sweep

import (
	"testing"

	"github.com/xianxu/metis/pkg/shape"
)

func pts(n int) []shape.Point {
	out := make([]shape.Point, n)
	for i := range out {
		out[i] = shape.Point{With: map[string]map[string]any{"s": {"i": i}}}
	}
	return out
}

// A grid sampler asks each point in order, then reports done; Tell is a no-op.
func TestGrid_EnumeratesAllThenDone(t *testing.T) {
	g := NewGrid(pts(3), nil)
	var seen []int
	for {
		p, ok := g.Ask()
		if !ok {
			break
		}
		seen = append(seen, p.With["s"]["i"].(int))
		g.Tell(p, Result{Status: "ok"})
	}
	if len(seen) != 3 || seen[0] != 0 || seen[1] != 1 || seen[2] != 2 {
		t.Errorf("grid should enumerate [0 1 2] in order, got %v", seen)
	}
	// A further Ask after exhaustion stays done.
	if _, ok := g.Ask(); ok {
		t.Error("grid should stay done after exhaustion")
	}
}

func TestGrid_EmptyIsImmediatelyDone(t *testing.T) {
	g := NewGrid(nil, nil)
	if _, ok := g.Ask(); ok {
		t.Error("an empty grid must be immediately done")
	}
}

// MaxPoints stops the sweep after k asks, before exhaustion.
func TestGrid_MaxPointsStopsEarly(t *testing.T) {
	g := NewGrid(pts(10), MaxPoints(3))
	n := 0
	for {
		p, ok := g.Ask()
		if !ok {
			break
		}
		n++
		g.Tell(p, Result{Status: "ok"})
	}
	if n != 3 {
		t.Errorf("MaxPoints(3) should yield 3 points from a 10-point grid, got %d", n)
	}
}

// TargetReached stops once a told result crosses the threshold — both directions.
func TestGrid_TargetReachedStops(t *testing.T) {
	// maximize: stop once cv_score >= 0.9. Points tell increasing scores.
	scores := []float64{0.70, 0.85, 0.92, 0.95}
	g := NewGrid(pts(len(scores)), TargetReached("cv_score", "maximize", 0.9))
	n := 0
	for {
		p, ok := g.Ask()
		if !ok {
			break
		}
		g.Tell(p, Result{Status: "ok", Metrics: map[string]float64{"cv_score": scores[n]}})
		n++
	}
	// asks 0.70, 0.85, 0.92 (crosses) → stops before the 4th.
	if n != 3 {
		t.Errorf("maximize target 0.9 should stop after the 0.92 tell (3 points), got %d", n)
	}

	// minimize: stop once loss <= 0.1.
	losses := []float64{0.5, 0.3, 0.08, 0.02}
	g2 := NewGrid(pts(len(losses)), TargetReached("loss", "minimize", 0.1))
	m := 0
	for {
		p, ok := g2.Ask()
		if !ok {
			break
		}
		g2.Tell(p, Result{Status: "ok", Metrics: map[string]float64{"loss": losses[m]}})
		m++
	}
	if m != 3 {
		t.Errorf("minimize target 0.1 should stop after the 0.08 tell (3 points), got %d", m)
	}
}

// A failed point (no target metric) doesn't trip TargetReached.
func TestTargetReached_IgnoresMissingMetric(t *testing.T) {
	g := NewGrid(pts(2), TargetReached("cv_score", "maximize", 0.5))
	p, _ := g.Ask()
	g.Tell(p, Result{Status: "failed"}) // no metrics
	if _, ok := g.Ask(); !ok {
		t.Error("a failed point (no metric) must not trip the target — sweep should continue")
	}
}

// AnyStop fires if any of its predicates fires.
func TestAnyStop_Composes(t *testing.T) {
	g := NewGrid(pts(10), AnyStop(MaxPoints(5), TargetReached("x", "maximize", 100)))
	n := 0
	for {
		p, ok := g.Ask()
		if !ok {
			break
		}
		n++
		g.Tell(p, Result{Status: "ok"})
	}
	if n != 5 {
		t.Errorf("AnyStop(MaxPoints(5), …) should stop at 5, got %d", n)
	}
}
