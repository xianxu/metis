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

// FixedKFolds is the static resample Sampler: k folds over the materialized
// partition, all proposed at once (no feedback), reduced by Aggregate → (mean,SE).
// The degenerate static scatter/gather of the resample level; racing/early-stop
// (metis#19+) is a later Sampler over the same FoldPoints. Its runPoint yields the
// raw fold score (float64); Tell pairs it with the fold's authoritative address.
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

// Tell folds one fold-point's raw score in, stamping the fold's authoritative Addr
// (so the told-set key comes from the FoldPoint, not the runPoint).
func (f FixedKFolds) Tell(s foldState, p FoldPoint, out float64) foldState {
	s.scores = append(s.scores, FoldScore{Addr: p.Addr(), Score: out})
	return s
}

// Done reduces the told fold scores → (mean, SE), keyed on the sorted told-set.
func (f FixedKFolds) Done(s foldState) MeanSE { return Aggregate(s.scores) }

var _ Sampler[foldState, FoldPoint, float64, MeanSE] = FixedKFolds{}
