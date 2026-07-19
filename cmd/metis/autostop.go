package main

import (
	"math"

	"github.com/xianxu/metis/pkg/ledger"
)

// metis#66 M2: incumbent-referenced early stop of LOSING families. The incumbent is read ONCE
// at run start from the shape's EXISTING ledger (prior runs — no --baseline flag); after each
// completed outer fold (n≥2) a family whose full-k mean is <95%-likely to REACH the incumbent
// stops its remaining outer folds. Losers ONLY — a family that could still win runs to full k
// (a truncated optimistic estimate must never be shippable). The rule is a PURE function
// (shouldStop/tCrit, unit-tested directly, ARCH-PURE); the sweep driver is the thin seam that
// consults it and filters the next fold's config set.

// incumbentRef is the score a family must be able to reach — snapshotted BEFORE the current run
// contributes any rows (writeSweepLedger runs only at finalize), so a family is never compared
// against its own partial estimate. present=false ⇒ no prior run ⇒ auto-stop is a loud no-op.
type incumbentRef struct {
	score     float64
	direction string
	present   bool
}

// readIncumbent snapshots the best per-family OUTER estimate already in the shape's ledger:
// AggregateView reduces the raw outer rows to per-(family-winning-config) means, and the best by
// direction is the score to beat. The OUTER estimate (not the optimistic inner CV) is the honest
// reference. Empty / prior-less ledger ⇒ present=false.
func readIncumbent(shapePath, metric, direction string) incumbentRef {
	ref := incumbentRef{direction: direction}
	led, err := loadLedger(shapePath)
	if err != nil {
		return ref
	}
	for _, r := range ledger.AggregateView(led, metric).Rows {
		if r.Level != "outer" || r.Status == "failed" {
			continue
		}
		v, ok := r.Metrics[metric]
		if !ok {
			continue
		}
		if !ref.present || betterMeanSE(v, ref.score, direction) {
			ref.score, ref.present = v, true
		}
	}
	return ref
}

// shouldStop reports whether a family with n≥2 observed outer-fold scores is <95%-likely to
// reach `incumbent` by full k folds — the metis#66 loser-stop rule.
//
// DERIVATION (documented, not silent — the Spec authorizes a liberal-but-documented rule): the n
// done folds fix their contribution; model the r=k−n remaining folds as iid draws sharing the
// observed mean m and sample sd s. The full-k mean M_k = (n·m + Σ r future)/k has predictive mean
// m and predictive variance from BOTH the spread of the r future folds AND the error in
// estimating their mean from n samples:
//
//	SEpred² = (s²·r / k²) · (1 + r/n)
//
// A one-sided t_{n-1} 95% bound on M_k is m ± t·SEpred. For MAXIMIZE, stop iff even the highest
// plausible full-k mean m + t·SEpred is still below the incumbent (the family cannot plausibly
// reach it). MINIMIZE is symmetric: stop iff the lowest plausible m − t·SEpred is still above it.
// n<2 (no spread estimate) and r≤0 (already full k) never stop — so a would-be winner, whose
// bound straddles the incumbent, always runs to full k.
func shouldStop(scores []float64, k int, incumbent float64, direction string) bool {
	n := len(scores)
	r := k - n
	if n < 2 || r <= 0 {
		return false
	}
	var sum float64
	for _, x := range scores {
		sum += x
	}
	mean := sum / float64(n)
	var ss float64
	for _, x := range scores {
		d := x - mean
		ss += d * d
	}
	s := math.Sqrt(ss / float64(n-1)) // sample standard deviation
	sePred := s * math.Sqrt(float64(r)/(float64(k)*float64(k))*(1+float64(r)/float64(n)))
	t := tCrit(n - 1)
	if direction == "minimize" {
		return mean-t*sePred > incumbent // even the best (lowest) plausible mean can't get low enough
	}
	return mean+t*sePred < incumbent // even the best (highest) plausible mean can't reach it
}

// tCrit is the one-sided 95% Student-t critical value for df degrees of freedom — a small
// hardcoded table (df 1..10); df≥11 uses the normal z=1.645 (negligible at the loser-stop
// decision). The wide small-df values (t_1=6.31) deliberately protect a would-be winner from a
// premature stop when only 2–3 folds are in.
func tCrit(df int) float64 {
	table := []float64{6.314, 2.920, 2.353, 2.132, 2.015, 1.943, 1.895, 1.860, 1.833, 1.812}
	if df >= 1 && df <= len(table) {
		return table[df-1]
	}
	return 1.645
}
