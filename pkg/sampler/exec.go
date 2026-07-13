package sampler

import "sync"

// Exec is the batch-execution strategy injected into Run (the metis#31 seam): it
// runs an Ask batch's points through runPoint and returns the outputs IN BATCH
// ORDER, so Run can Tell them in the fixed order the sampler proposed. Every point
// in an Ask batch was proposed without any other's result (that is what makes it a
// batch), so a batch is independent by construction → safe to run concurrently.
// SeqExec (the default) runs serially; ParExec runs concurrently. The strategy
// bounds nothing itself — the only budgeted resource is the real subprocess spawn,
// capped at the leaf (cmd/metis execStep's semaphore).
type Exec[P, O any] func(points []P, runPoint func(P) O) []O

// SeqExec runs a batch serially, in order — byte-identical to Run's original loop
// body. The backward-compatible default (tests and the serial path pass this).
func SeqExec[P, O any](points []P, runPoint func(P) O) []O {
	out := make([]O, len(points))
	for i, p := range points {
		out[i] = runPoint(p)
	}
	return out
}

// ParExec runs a batch concurrently — one goroutine per point — and returns outputs
// in BATCH ORDER (each goroutine writes its own index, so no result mutex is needed
// and the sequence Run.Tell sees is identical to SeqExec's). It bounds NOTHING
// itself: orchestration goroutines are cheap; the only budgeted resource is the real
// subprocess spawn, capped by the leaf semaphore inside the injected runPoint's
// execStep (metis#31). So nesting (driver ⊃ sweeper ⊃ resample) fans out freely
// while live step-subprocesses stay ≤ n, and no orchestration goroutine holds the
// budget while awaiting children (deadlock-free).
func ParExec[P, O any](points []P, runPoint func(P) O) []O {
	out := make([]O, len(points))
	var wg sync.WaitGroup
	wg.Add(len(points))
	for i, p := range points {
		go func(i int, p P) { defer wg.Done(); out[i] = runPoint(p) }(i, p)
	}
	wg.Wait()
	return out
}

// execFor selects the batch strategy — the single Seq/Par branch point (ARCH-DRY).
func execFor[P, O any](parallel bool) func([]P, func(P) O) []O {
	if parallel {
		return ParExec[P, O]
	}
	return SeqExec[P, O]
}
