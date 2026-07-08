package sampler

// Run drives one Sampler at any level: seed the accumulator, then loop
// Ask → run each proposed Point via runPoint → Tell, until Ask reports done, and
// return Done's reduce. The loop is identical for every level; nesting is just
// runPoint = a Run of the inner Sampler (the types compose — see the interface
// doc). Pure over (smp, runPoint): all IO lives in the injected runPoint closure
// (ARCH-PURE).
//
// Contract: a well-behaved Sampler makes progress — Ask either returns done, or a
// non-empty batch whose Tell advances the accumulator. (A static Sampler emits its
// whole point-set on the first Ask, then reports done.)
func Run[S, P, O, R any](ctx Ctx, smp Sampler[S, P, O, R], runPoint func(P) O) R {
	s := smp.Init(ctx)
	for {
		batch, done := smp.Ask(s)
		if done {
			break
		}
		for _, p := range batch {
			s = smp.Tell(s, p, runPoint(p))
		}
	}
	return smp.Done(s)
}
