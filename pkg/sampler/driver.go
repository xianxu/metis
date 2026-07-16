package sampler

import "fmt"

// SinglePoint is the driver:single degenerate outer point — "the whole training
// set, once". Exported so the cmd/metis driver loop names it as the sweeper's outer
// point type (metis#18 M1a-5: the driver level is now real, not inlined).
type SinglePoint struct{}

// SingleDriver is the degenerate outer Sampler for driver:single: no honest outer
// resample — it runs the sweeper once on all data and passes its SweepResult through
// (no estimate). The seam metis#23's k-outer-fold nested-CV driver replaces (an
// adaptive-nesting Sampler that scores the winner on each sealed outer fold).
type SingleDriver struct{}

type driverState struct {
	result SweepResult
	told   bool
}

func (SingleDriver) Init(ctx Ctx) driverState { return driverState{} }

// Ask proposes one all-data point, then is done.
func (SingleDriver) Ask(s driverState) ([]SinglePoint, bool) {
	if s.told {
		return nil, true
	}
	return []SinglePoint{{}}, false
}

// Tell captures the sweeper's SweepResult (the runPoint's output).
func (SingleDriver) Tell(s driverState, _ SinglePoint, r SweepResult) driverState {
	s.result = r
	s.told = true
	return s
}

// Done passes the sweeper result through — driver:single ships its cross-family pick
// (no honest estimate; that's driver:cv, metis#23).
func (SingleDriver) Done(s driverState) SweepResult { return s.result }

// SizeHint: the single driver proposes exactly one all-data point (metis#30).
func (SingleDriver) SizeHint(driverState) (int, SizeKind) { return 1, SizeExact }

var _ Sampler[driverState, SinglePoint, SweepResult, SweepResult] = SingleDriver{}

// OuterFoldPoint is one outer resample fold (metis#23 nested-CV): the driver hands the
// sweeper this fold's sealed outer-analysis data and scores the winner on the held
// outer-assessment. Mirrors FoldPoint at the outer level (just the index; the analysis
// subset is materialized by the IO shell, not carried in the pure point).
type OuterFoldPoint struct{ Idx int }

// CVDriver is the nested-CV outer Sampler (metis#23) — the honest counterpart to
// SingleDriver's pass-through. It proposes k outer folds; the runPoint runs the black-box
// sweeper on each fold's SEALED outer-analysis → a winner, then refits + scores that winner
// on the held outer-assessment (the runPoint's float64). Done aggregates the k outer scores
// → MeanSE, the HONEST procedure estimate. Result-dependent (folds may select different
// winners) and produces NO shippable winner — estimation ≠ selection; the ship stays on
// driver:single. A new Sampler impl over the UNCHANGED Run loop (no engine change).
type CVDriver struct {
	K        int
	Stratify bool
}

type cvDriverState struct {
	points []OuterFoldPoint
	scores []FoldScore
}

// Init enumerates the k outer fold-points (mirror FixedKFolds.Init).
func (d CVDriver) Init(ctx Ctx) cvDriverState {
	pts := make([]OuterFoldPoint, d.K)
	for i := range pts {
		pts[i] = OuterFoldPoint{Idx: i}
	}
	return cvDriverState{points: pts}
}

// Ask proposes the not-yet-told outer folds as one batch; done once all k are told —
// derived from the told count (mirror FixedKFolds; no separate flag, so k=0 is done
// immediately rather than emitting an empty non-done batch).
func (d CVDriver) Ask(s cvDriverState) ([]OuterFoldPoint, bool) {
	if len(s.scores) >= len(s.points) {
		return nil, true
	}
	return s.points[len(s.scores):], false
}

// Tell folds one outer fold's held-out score in. FoldScore.Addr is the outer fold's
// identity (keeps MeanSE.ToldSet meaningful); complexity is not measured at the outer
// level (the honest estimate is over scores, not parsimony).
func (d CVDriver) Tell(s cvDriverState, p OuterFoldPoint, out float64) cvDriverState {
	s.scores = append(s.scores, FoldScore{Addr: fmt.Sprintf("outer#%d", p.Idx), Score: out})
	return s
}

// Done aggregates the k outer scores → the honest procedure estimate (mean ± SE),
// reusing the SAME reducer as the resample level (sampler.Aggregate).
func (d CVDriver) Done(s cvDriverState) MeanSE { return Aggregate(s.scores) }

// SizeHint is the outer fold count (metis#30).
func (d CVDriver) SizeHint(s cvDriverState) (int, SizeKind) { return len(s.points), SizeExact }

var _ Sampler[cvDriverState, OuterFoldPoint, float64, MeanSE] = CVDriver{}
