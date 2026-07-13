package sampler

// Run drives one Sampler at any level: seed the accumulator, then loop
// Ask → run the proposed batch via the injected exec → Tell each in batch order,
// until Ask reports done, and return Done's reduce. The loop is identical for every
// level; nesting is just runPoint = a Run of the inner Sampler (the types compose —
// see the interface doc). Pure over (smp, runPoint, exec): all IO lives in the
// injected runPoint closure (ARCH-PURE).
//
// exec (metis#31) runs one Ask batch and returns outputs IN BATCH ORDER — every
// point in a batch was proposed without any other's result, so a batch is
// independent by construction and safe to run concurrently; exec = SeqExec is the
// serial default, ParExec runs the batch concurrently under the leaf semaphore. Run
// Tells the outputs in the fixed batch order, so the order-independent reduce
// (metis#18) yields an identical Done regardless of the exec strategy.
//
// Contract: a well-behaved Sampler makes progress — Ask either returns done, or a
// non-empty batch whose Tell advances the accumulator. (A static Sampler emits its
// whole point-set on the first Ask, then reports done.)
func Run[S, P, O, R any](ctx Ctx, smp Sampler[S, P, O, R], runPoint func(P) O, exec func([]P, func(P) O) []O) R {
	s := smp.Init(ctx)
	for {
		batch, done := smp.Ask(s)
		if done {
			break
		}
		if len(batch) == 0 {
			// Contract violation: a non-done Ask that proposes nothing can't make
			// progress → the loop would spin forever. Fail LOUD + diagnosable rather
			// than hang (guards a future adaptive Sampler's bug).
			panic("sampler: Ask returned an empty batch without done — a Sampler must make progress (emit a non-empty batch or report done)")
		}
		outs := exec(batch, runPoint)
		for i, p := range batch {
			s = smp.Tell(s, p, outs[i])
		}
	}
	return smp.Done(s)
}
