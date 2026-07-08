package sampler

import "github.com/xianxu/metis/pkg/shape"

// GridConfigs is the static sweeper Sampler: it proposes every config-point from
// shape.Expand at once (grid = the degenerate ask-for-every-point sampler), scores
// each via its inner resample (the runPoint yields the config's (mean,SE)), and
// Done-selects the winner by the objective. metis#19's 1-SE robust select is a
// DIFFERENT Done over the same per-config (mean,SE) — no re-run. Points are the
// pre-expanded config space (shape.Expand over the pipeline phase); Direction is
// the objective direction; Select is the rule (M1a: "argmax-mean" only). The seed
// comes from Ctx (single source — Winner.Seed is sourced there, not duplicated on
// the sampler), so M1a-4 wiring can't diverge.
type GridConfigs struct {
	Points    []shape.Point
	Direction string // "maximize" | "minimize"
	Select    string // M1a: "argmax-mean"
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

// Done selects the winner. M1a: argmax-mean (highest mean for "maximize", lowest
// for "minimize"). Deterministic tie-break: the strict comparison keeps the FIRST
// among equals, i.e. the earliest in shape.Expand order (stable/sorted).
func (g GridConfigs) Done(s configState) Winner {
	best := -1
	for i, r := range s.results {
		if best < 0 || betterMean(r.meanSE.Mean, s.results[best].meanSE.Mean, g.Direction) {
			best = i
		}
	}
	if best < 0 {
		return Winner{Seed: s.seed}
	}
	w := s.results[best]
	return Winner{
		Point:    w.point,
		Seed:     s.seed,
		FoldKeys: w.meanSE.ToldSet,
		Score:    w.meanSE,
	}
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

var _ Sampler[configState, shape.Point, MeanSE, Winner] = GridConfigs{}
