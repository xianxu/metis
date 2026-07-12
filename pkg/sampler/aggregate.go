package sampler

import (
	"math"
	"sort"
)

// FoldScore pairs a resample fold's address (identity for the told-set key) with
// its score and the model's MEASURED complexity for that fold (metis#19). The
// resample Sampler's Tell builds these from (fold-point, FoldOutcome). HasComplexity
// distinguishes "measured 0" from "not measured" (M1 wires complexity as 0 e2e).
type FoldScore struct {
	Addr          string
	Score         float64
	Complexity    float64
	HasComplexity bool
}

// MeanSE is the resample Sampler's Done — the honest per-config estimate: the mean
// fold score, its standard error, the mean measured complexity (metis#19), and the
// sorted set of fold addresses that actually ran (the content-addressed,
// order-independent told-set key; a static fixed-k tells all k, an adaptive Sampler a
// runtime subset — metis#24).
type MeanSE struct {
	Mean           float64
	SE             float64
	MeanComplexity float64 // mean of the per-fold measured complexity (metis#19)
	HasComplexity  bool    // true iff every fold reported a measured complexity
	ToldSet        []string
}

// Aggregate reduces the told fold scores → (mean, SE, meanComplexity). SE = sample
// standard deviation / √n (n−1 denominator; 0 when n<2). HasComplexity is true only
// when every fold carried a measured complexity, so a partial/absent report reads as
// "not measured" (the M2 parsimony guard keys off it). Order-independent: the
// reductions are symmetric and ToldSet is sorted. Pure. This is the resample
// Sampler's Done — and a DIFFERENT Done (metis#19's select rule) re-reduces the same
// cached FoldScores for free.
func Aggregate(scores []FoldScore) MeanSE {
	n := len(scores)
	if n == 0 {
		return MeanSE{}
	}
	addrs := make([]string, 0, n)
	var sum, cxSum float64
	hasComplexity := true
	for _, s := range scores {
		sum += s.Score
		cxSum += s.Complexity
		hasComplexity = hasComplexity && s.HasComplexity
		addrs = append(addrs, s.Addr)
	}
	mean := sum / float64(n)
	var se float64
	if n > 1 {
		var ss float64
		for _, s := range scores {
			d := s.Score - mean
			ss += d * d
		}
		se = math.Sqrt(ss/float64(n-1)) / math.Sqrt(float64(n))
	}
	sort.Strings(addrs)
	return MeanSE{Mean: mean, SE: se, MeanComplexity: cxSum / float64(n), HasComplexity: hasComplexity, ToldSet: addrs}
}
