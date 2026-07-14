package main

import (
	"github.com/xianxu/metis/pkg/ledger"
	"github.com/xianxu/metis/pkg/sampler"
)

// FamilyEstimate reduces a nested-CV run's OUTER rows into the per-family honest estimate
// (metis#32): group the outer rows by MODEL FAMILY (derived via the injected familyOf) and
// reduce the objective metric over the outer folds via the shared sampler.Aggregate →
// per-family (mean ± SE).
//
// Why a DEDICATED reducer, not ledger.AggregateView: a family's winning config DIFFERS across
// outer folds (outer-fold 0 → rf md=4, outer-fold 1 → rf md=8), so those rows share the family
// but carry DISTINCT free-params — AggregateView (which groups by exact free-params) would put
// them in separate groups and never compute the per-family mean over the outer folds. Only
// Level=="outer" rows participate; the inner rows are the config-selection signal (read via
// AggregateView, which pools a config's inner folds across outer folds), not this.
//
// familyOf is injected so pkg/ledger stays free of a pkg/sampler dependency (sampler.FamilyOf
// needs the full shape.Point, which the caller reconstructs; a test can pass a trivial one).
func FamilyEstimate(l ledger.Ledger, metric string, familyOf func(ledger.Row) string) map[string]sampler.MeanSE {
	byFamily := map[string][]sampler.FoldScore{}
	for _, r := range l.Rows {
		if r.Level != "outer" || r.Status == "failed" {
			continue
		}
		v, ok := r.Metrics[metric]
		if !ok {
			continue
		}
		fam := familyOf(r)
		byFamily[fam] = append(byFamily[fam], sampler.FoldScore{Addr: r.PointAddr, Score: v})
	}
	out := make(map[string]sampler.MeanSE, len(byFamily))
	for fam, scores := range byFamily {
		out[fam] = sampler.Aggregate(scores)
	}
	return out
}
