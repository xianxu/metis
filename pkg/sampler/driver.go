package sampler

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

var _ Sampler[driverState, SinglePoint, SweepResult, SweepResult] = SingleDriver{}
