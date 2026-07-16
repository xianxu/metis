package sampler

import "fmt"

// FoldPoint is one proposed resample point: fold idx of the materialized
// partition. Its Addr is the deterministic identity that enters the told-set key
// (and, at M1a-4, the per-fold cache Kpre via the with-overlay).
type FoldPoint struct {
	Partition PartitionRef
	Idx       int
}

// Addr is the fold-point's stable identity string.
func (fp FoldPoint) Addr() string { return fmt.Sprintf("%s#fold%d", fp.Partition, fp.Idx) }

// FoldOutcome is the resample runPoint's output (metis#19): a fold's raw score plus
// the model's MEASURED complexity for that fold (HasComplexity=false when the step
// emitted none — M1 wires it as an absent 0). Widened from a bare float64 so the
// per-config reduction carries complexity for the select rule.
type FoldOutcome struct {
	Score         float64
	Complexity    float64
	HasComplexity bool
}

// FixedKFolds is the static resample Sampler: k folds over the materialized
// partition, all proposed at once (no feedback), reduced by Aggregate → (mean, SE,
// complexity). The degenerate static scatter/gather of the resample level;
// racing/early-stop is a later Sampler over the same FoldPoints. Its runPoint yields a
// FoldOutcome; Tell pairs it with the fold's authoritative address.
type FixedKFolds struct {
	K int
}

type foldState struct {
	points []FoldPoint
	scores []FoldScore
}

// Init reads the already-materialized partition ref from ctx (no IO — that's
// M1a-4) and enumerates the k fold-points.
func (f FixedKFolds) Init(ctx Ctx) foldState {
	pts := make([]FoldPoint, f.K)
	for i := range pts {
		pts[i] = FoldPoint{Partition: ctx.Partition, Idx: i}
	}
	return foldState{points: pts}
}

// Ask proposes the not-yet-told fold-points as one batch; done once all k are told.
func (f FixedKFolds) Ask(s foldState) ([]FoldPoint, bool) {
	if len(s.scores) >= len(s.points) {
		return nil, true
	}
	return s.points[len(s.scores):], false
}

// Tell folds one fold-point's outcome in, stamping the fold's authoritative Addr
// (so the told-set key comes from the FoldPoint, not the runPoint).
func (f FixedKFolds) Tell(s foldState, p FoldPoint, out FoldOutcome) foldState {
	s.scores = append(s.scores, FoldScore{Addr: p.Addr(), Score: out.Score, Complexity: out.Complexity, HasComplexity: out.HasComplexity})
	return s
}

// Done reduces the told fold scores → (mean, SE, complexity), keyed on the sorted told-set.
func (f FixedKFolds) Done(s foldState) MeanSE { return Aggregate(s.scores) }

// SizeHint is the fixed fold count k (metis#30).
func (f FixedKFolds) SizeHint(s foldState) (int, SizeKind) { return len(s.points), SizeExact }

var _ Sampler[foldState, FoldPoint, FoldOutcome, MeanSE] = FixedKFolds{}
