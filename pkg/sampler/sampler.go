// Package sampler is the metis-v2 M1a Sampler fold node — the first-class
// resample/sweep construct. A Sampler is an ask/tell fold: Init → (Ask → run each
// proposed Point → Tell)* → Done, instantiated nested (driver ⊃ sweeper ⊃
// resample). Static scatter/gather is the degenerate Sampler whose Tell ignores
// feedback and whose Ask emits its whole point-set at once (grid over configs,
// fixed-k over folds); adaptive Samplers (1-SE select, nested driver, racing,
// Bayesian — metis#19/#23/future) use the feedback edge. Pure: no IO here — the
// subprocess/FS execution is injected as the runPoint closure (ARCH-PURE).
package sampler

// PartitionRef identifies an already-materialized fold partition (its
// content-hash / which-rows). Materialization is IO (a cv-split step, metis#18
// M1a-4); a Sampler only consumes the ref via Ctx, so it stays pure.
type PartitionRef string

// Ctx carries the run-scoped inputs a Sampler's Init may need: the experiment
// seed and the materialized partition ref (empty when the level doesn't resample).
type Ctx struct {
	Seed      int
	Partition PartitionRef
}

// SizeKind classifies a Sampler's SizeHint total (metis#30): exact (a static
// point-set), budget (an upper bound, e.g. maxEvals), or unknown (open-ended
// adaptive). The renderer shows k/n, k/≤n, k/? respectively.
type SizeKind int

const (
	SizeExact SizeKind = iota
	SizeBudget
	SizeUnknown
)

// Sampler is the ask/tell fold node, generic over its accumulator (S), the Point
// it proposes (P), the Output a run yields (O), and its terminal Result (R):
//
//	Init(ctx)   → S           initial accumulator (incumbent / running stats)
//	Ask(S)      → ([]P, done) propose the next scatter (may be empty); done = stop
//	Tell(S,P,O) → S           fold one completed Point's output into the accumulator
//	Done(S)     → R           terminal reduce (the gather): (mean,SE) | winner
//	SizeHint(S) → (n, kind)   the level's total point count (metis#30) — the ONLY
//	                          per-sampler progress bit; Run reads it once on the
//	                          initial accumulator and stamps it on every event.
//
// The levels compose by type: the sweeper's runPoint is Run(resample,…) whose
// R=(mean,SE) is the sweeper's O; the driver's runPoint is Run(sweeper,…) whose
// R=Winner is the driver's O.
type Sampler[S, P, O, R any] interface {
	Init(ctx Ctx) S
	Ask(s S) (batch []P, done bool)
	Tell(s S, p P, out O) S
	Done(s S) R
	SizeHint(s S) (total int, kind SizeKind)
}
