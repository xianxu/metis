package sampler

import (
	"math"
	"sort"
)

// FoldScore pairs a resample fold's address (identity for the told-set key) with
// its score. The resample Sampler's Tell builds these from (fold-point, raw score).
type FoldScore struct {
	Addr  string
	Score float64
}

// MeanSE is the resample Sampler's Done — the honest per-config estimate: the mean
// fold score, its standard error, and the sorted set of fold addresses that
// actually ran (the content-addressed, order-independent told-set key; a static
// fixed-k tells all k, an adaptive Sampler a runtime subset — metis#24).
type MeanSE struct {
	Mean    float64
	SE      float64
	ToldSet []string
}

// Aggregate reduces the told fold scores → (mean, SE). SE = sample standard
// deviation / √n (n−1 denominator; 0 when n<2). Order-independent: mean/SE are
// symmetric in the inputs and ToldSet is sorted. Pure. This is the resample
// Sampler's Done — and a DIFFERENT Done (metis#19's 1-SE select) re-reduces the
// same cached FoldScores for free.
func Aggregate(scores []FoldScore) MeanSE {
	n := len(scores)
	if n == 0 {
		return MeanSE{}
	}
	addrs := make([]string, 0, n)
	var sum float64
	for _, s := range scores {
		sum += s.Score
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
	return MeanSE{Mean: mean, SE: se, ToldSet: addrs}
}
