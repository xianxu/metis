// Package sweep is the metis#7 sweep sampler: the pure ask/tell seam that drives a
// config-space exploration, plus the grid sampler and the stop predicates. The driver
// (cmd/metis) loops `Ask → run the point → Tell`; the sampler owns which point comes
// next and when to stop. Grid enumerates #6's pre-expanded points in order; adaptive
// samplers (post-v1) hold a model updated via Tell and slot in with no loop change.
// Pure + seeded-deterministic — no IO here; the run/record/cache IO is the cmd/metis
// shell (ARCH-PURE).
package sweep

import "github.com/xianxu/metis/pkg/shape"

// Result is one point's outcome, fed back via Tell — the thin signal a sampler needs
// (adaptive samplers read Metrics; grid ignores it, but a stop predicate may use it).
type Result struct {
	Metrics map[string]float64
	Status  string // "ok" | "failed"
}

// TellRecord is one entry of a sampler's tell-history: the point that ran + its result.
type TellRecord struct {
	Point  shape.Point
	Result Result
}

// Sampler is the ask/tell seam. Ask returns the next point to run and whether the
// sweep should continue (false = done); Tell reports a point's result back so a
// stateful sampler can adapt (grid: no-op) and stop predicates can observe the history.
type Sampler interface {
	Ask() (shape.Point, bool)
	Tell(shape.Point, Result)
}

// StopPredicate decides, from the accumulated tell-history, whether to stop early
// (before the sampler's natural exhaustion). Pure over the history.
type StopPredicate func(history []TellRecord) bool

// Grid is the v1 sampler: it enumerates a fixed, pre-expanded point set (from
// shape.Expand) in order. Tell is a no-op (grid holds no model). It is done on
// exhaustion, or early when its optional StopPredicate fires over the tell-history.
// Deterministic; the sampler_seed slot (unused by grid) documents the seam an adaptive
// impl uses for reproducibility.
type Grid struct {
	points  []shape.Point
	next    int
	stop    StopPredicate
	history []TellRecord
}

// NewGrid returns a grid sampler over points, stopping early if stop fires (nil = run
// to exhaustion).
func NewGrid(points []shape.Point, stop StopPredicate) *Grid {
	return &Grid{points: points, stop: stop}
}

func (g *Grid) Ask() (shape.Point, bool) {
	if g.next >= len(g.points) {
		return shape.Point{}, false
	}
	if g.stop != nil && g.stop(g.history) {
		return shape.Point{}, false
	}
	p := g.points[g.next]
	g.next++
	return p, true
}

func (g *Grid) Tell(p shape.Point, r Result) {
	g.history = append(g.history, TellRecord{Point: p, Result: r})
}

var _ Sampler = (*Grid)(nil)

// MaxPoints stops the sweep once n points have been told (a budget cap).
func MaxPoints(n int) StopPredicate {
	return func(history []TellRecord) bool { return len(history) >= n }
}

// TargetReached stops once a told result's metric crosses the threshold in the
// objective direction ("maximize": metric >= threshold; "minimize": metric <=
// threshold). A point missing the metric (e.g. a failed run) never trips it.
func TargetReached(metric, direction string, threshold float64) StopPredicate {
	return func(history []TellRecord) bool {
		for _, h := range history {
			v, ok := h.Result.Metrics[metric]
			if !ok {
				continue
			}
			if direction == "minimize" {
				if v <= threshold {
					return true
				}
			} else if v >= threshold { // default/maximize
				return true
			}
		}
		return false
	}
}

// AnyStop composes predicates: it fires when any of them fires.
func AnyStop(preds ...StopPredicate) StopPredicate {
	return func(history []TellRecord) bool {
		for _, p := range preds {
			if p != nil && p(history) {
				return true
			}
		}
		return false
	}
}
