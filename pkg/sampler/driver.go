package sampler

// singlePoint is the driver:single degenerate outer point — "the whole training
// set, once".
type singlePoint struct{}

// SingleDriver is the degenerate outer Sampler for driver:single: no honest outer
// resample — it runs the sweeper once on all data and passes its Winner through
// (no estimate). The seam metis#23's k-outer-fold nested-CV driver replaces (an
// adaptive-nesting Sampler that scores the winner on each sealed outer fold).
type SingleDriver struct{}

type driverState struct {
	winner Winner
	told   bool
}

func (SingleDriver) Init(ctx Ctx) driverState { return driverState{} }

// Ask proposes one all-data point, then is done.
func (SingleDriver) Ask(s driverState) ([]singlePoint, bool) {
	if s.told {
		return nil, true
	}
	return []singlePoint{{}}, false
}

// Tell captures the sweeper's Winner (the runPoint's output).
func (SingleDriver) Tell(s driverState, _ singlePoint, w Winner) driverState {
	s.winner = w
	s.told = true
	return s
}

// Done passes the winner through — driver:single ships it (no honest estimate).
func (SingleDriver) Done(s driverState) Winner { return s.winner }

var _ Sampler[driverState, singlePoint, Winner, Winner] = SingleDriver{}
