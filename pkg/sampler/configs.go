package sampler

import (
	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/shape"
)

// GridConfigs is the static sweeper Sampler: it proposes every config-point from
// shape.Expand at once (grid = the degenerate ask-for-every-point sampler), scores
// each via its inner resample (the runPoint yields the config's (mean, SE, complexity)),
// and Done-selects via the metis#19 rule → a per-family winner map + a cross-family ship
// pick (SweepResult) — a DIFFERENT Done over the same per-config stats, no re-run. Points
// are the pre-expanded config space (shape.Expand over the pipeline phase); Direction is
// the objective direction; Select is the tagged-union rule. The seed comes from Ctx (single
// source — Winner.Seed is sourced there, not duplicated on the sampler).
type GridConfigs struct {
	Points    []shape.Point
	Direction string            // "maximize" | "minimize"
	Select    experiment.Select // the metis#19 select rule (tagged union)
}

type configResult struct {
	point  shape.Point
	meanSE MeanSE
}

type configState struct {
	points  []shape.Point
	results []configResult
	seed    int
}

func (g GridConfigs) Init(ctx Ctx) configState { return configState{points: g.Points, seed: ctx.Seed} }

// Ask proposes the not-yet-told config-points as one batch; done once all told.
func (g GridConfigs) Ask(s configState) ([]shape.Point, bool) {
	if len(s.results) >= len(s.points) {
		return nil, true
	}
	return s.points[len(s.results):], false
}

// Tell records one config's (mean,SE).
func (g GridConfigs) Tell(s configState, p shape.Point, out MeanSE) configState {
	s.results = append(s.results, configResult{point: p, meanSE: out})
	return s
}

// Done selects via the metis#19 rule: build a per-config ConfigStat (tagging each
// point's model family off its With bundling) and hand off to the pure SelectConfigs →
// a per-family winner map + the cross-family ship pick. The rule is a re-reduction over
// the already-cached per-config stats (no re-run); grouping/parsimony live in select.go.
func (g GridConfigs) Done(s configState) SweepResult {
	stats := make([]ConfigStat, len(s.results))
	for i, r := range s.results {
		stats[i] = ConfigStat{Point: r.point, Family: familyOf(r.point), Score: r.meanSE}
	}
	return SelectConfigs(g.Select, g.Direction, s.seed, stats)
}

// betterMean reports whether a is strictly better than b for the objective
// direction ("minimize": lower; default/"maximize": higher). Strictness makes the
// tie-break keep the earlier config.
func betterMean(a, b float64, direction string) bool {
	if direction == "minimize" {
		return a < b
	}
	return a > b
}

var _ Sampler[configState, shape.Point, MeanSE, SweepResult] = GridConfigs{}
