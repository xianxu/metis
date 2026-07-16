package sampler

import "sync"

// ProgressEvent is metis#30's per-completion progress payload: the 1-based monotone
// completion count K against the level's SizeHint (Total, Kind), plus the completed
// (Point, Out) pair — typed per Run instantiation, so a consumer's closure gets its
// own level's payloads without type switches. Fired at POINT COMPLETION, not Tell:
// under ParExec + one-batch static samplers every Tell lands at batch end — useless
// as live progress. Exactly one event per point (the same count as one Tell per
// point); Run's internal mutex serializes increment+fire, so events arrive in k
// order even under ParExec.
type ProgressEvent[P, O any] struct {
	K, Total int
	Kind     SizeKind
	Point    P
	Out      O
}

// Run drives one Sampler at any level: seed the accumulator, then loop
// Ask → run the proposed batch via the injected exec → Tell each in batch order,
// until Ask reports done, and return Done's reduce. The loop is identical for every
// level; nesting is just runPoint = a Run of the inner Sampler (the types compose —
// see the interface doc). Pure over (smp, runPoint, exec, progress): all IO lives in
// the injected closures (ARCH-PURE).
//
// exec (metis#31) runs one Ask batch and returns outputs IN BATCH ORDER — every
// point in a batch was proposed without any other's result, so a batch is
// independent by construction and safe to run concurrently; exec = SeqExec is the
// serial default, ParExec runs the batch concurrently under the leaf semaphore. Run
// Tells the outputs in the fixed batch order, so the order-independent reduce
// (metis#18) yields an identical Done regardless of the exec strategy.
//
// progress (metis#30) fires once per completed point with the level's live k against
// SizeHint's (total, kind) — nil = no-op (zero overhead, the unwrapped loop). The
// callback runs ON exec's goroutines while holding Run's event mutex: keep it fast
// (it serializes completions), and never call back into this Run from it.
//
// Contract: a well-behaved Sampler makes progress — Ask either returns done, or a
// non-empty batch whose Tell advances the accumulator. (A static Sampler emits its
// whole point-set on the first Ask, then reports done.)
func Run[S, P, O, R any](ctx Ctx, smp Sampler[S, P, O, R], runPoint func(P) O, exec func([]P, func(P) O) []O, progress func(ProgressEvent[P, O])) R {
	s := smp.Init(ctx)
	if progress != nil {
		total, kind := smp.SizeHint(s)
		var mu sync.Mutex
		k := 0
		inner := runPoint
		runPoint = func(p P) O {
			o := inner(p)
			mu.Lock()
			k++
			progress(ProgressEvent[P, O]{K: k, Total: total, Kind: kind, Point: p, Out: o})
			mu.Unlock()
			return o
		}
	}
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
