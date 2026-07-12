package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/ledger"
	"github.com/xianxu/metis/pkg/sampler"
	"github.com/xianxu/metis/pkg/shape"
)

// cmdLedgerSelect handles `metis ledger select <shape.md> [--rule R] [--tolerance T]
// [--lambda L] [--fingerprint HASH]` — applies the select rule OFFLINE over the shape's cached
// ledger (no re-run) and prints the per-family robust winners + the cross-family ship pick.
// The SECOND consumer of sampler.SelectConfigs (the in-memory sweep is the first) — the same
// pure rule, so an offline `select` and the live ship agree by construction (metis#19).
func cmdLedgerSelect(args []string) error {
	fs := flag.NewFlagSet("ledger select", flag.ContinueOnError)
	rule := fs.String("rule", "", "select rule: argmax-mean | one-std-err | pct-loss | mean-std (default: the shape's rule)")
	tolerance := fs.Float64("tolerance", 0, "pct-loss band tolerance (default: the shape's, or 0.02)")
	lambda := fs.Float64("lambda", 0, "mean-std penalty λ (default: the shape's, or 1.0)")
	fingerprint := fs.String("fingerprint", "", "restrict to one code-fingerprint (metis#27)")
	shapePath, flags, err := hoistShapePath(args)
	if err != nil {
		return fmt.Errorf("ledger select: %w (usage: metis ledger select <shape.md> [--rule R] [--tolerance T] [--lambda L] [--fingerprint HASH])", err)
	}
	if err := fs.Parse(flags); err != nil {
		return err
	}
	return runLedgerSelect(selectOpts{
		shapePath: shapePath, rule: *rule, tolerance: *tolerance, lambda: *lambda, fingerprint: *fingerprint, out: os.Stdout,
	})
}

type selectOpts struct {
	shapePath   string
	rule        string  // "" → the shape's objective.select
	tolerance   float64 // pct-loss (0 → shape's, then 0.02)
	lambda      float64 // mean-std (0 → shape's, then 1.0)
	fingerprint string
	out         io.Writer
}

// runLedgerSelect is the testable core: load shape + ledger, reduce, group by family, apply
// the pure rule, print. Reuses shape.Expand (to recover each config's Point → family) and
// sampler.SelectConfigs (the identical rule the live sweep runs), so the two surfaces agree.
func runLedgerSelect(o selectOpts) error {
	raw, err := os.ReadFile(o.shapePath)
	if err != nil {
		return err
	}
	sh, err := experiment.ParseShape(string(raw))
	if err != nil {
		return err
	}
	rule, err := resolveSelectRule(sh, o)
	if err != nil {
		return err
	}
	led, err := loadLedger(o.shapePath)
	if err != nil {
		return err
	}
	led = ledger.Filter(led, o.fingerprint)
	metric := sh.Sweeper.Objective.Metric
	agg := ledger.AggregateView(led, metric)

	stats, err := configStatsFromLedger(sh, agg, metric)
	if err != nil {
		return err
	}
	if len(stats) == 0 {
		return fmt.Errorf("ledger select: no scored configs in %s (objective %q) — run the sweep first", ledgerPath(o.shapePath), metric)
	}
	if err := sampler.GuardComplexity(rule, stats); err != nil {
		return err
	}
	res := sampler.SelectConfigs(rule, sh.Sweeper.Objective.Direction, sh.Seed, stats)
	printSelectResult(o.out, sh, rule, res)
	return nil
}

// configStatsFromLedger builds the []sampler.ConfigStat the pure rule consumes from the
// aggregated ledger rows. CRITICAL (metis#19 M1-review): the family key MUST match
// sampler.FamilyOf's format, so we Expand the shape and match each aggregate row to its
// Point (freeParamsEqual, reused from promote), then call sampler.FamilyOf on that Point —
// NOT a bare `fp.train.model`. A row that matches no expanded config is skipped (a stale
// ledger row from an older shape); complexity absence flows through as HasComplexity=false
// (the guard rejects a parsimony rule then).
func configStatsFromLedger(sh experiment.Shape, agg ledger.Ledger, metric string) ([]sampler.ConfigStat, error) {
	points, err := shape.Expand(sh.Pipeline, 0)
	if err != nil {
		return nil, err
	}
	cxMetric := complexityMetricFor(metric)
	var stats []sampler.ConfigStat
	for _, r := range agg.Rows {
		if r.Status == "failed" {
			continue
		}
		mean, ok := r.Metrics[metric]
		if !ok {
			continue // no objective score → not a selectable config
		}
		var pt shape.Point
		matched := false
		for _, p := range points {
			if freeParamsEqual(p, r.FreeParams) {
				pt, matched = p, true
				break
			}
		}
		if !matched {
			continue // a stale row from a different shape revision
		}
		cx, hasCx := r.Metrics[cxMetric]
		n := int(r.Metrics[metric+".n"])
		stats = append(stats, sampler.ConfigStat{
			Point:  pt,
			Family: sampler.FamilyOf(pt),
			Score: sampler.MeanSE{
				Mean:           mean,
				SE:             r.Metrics[metric+".se"],
				MeanComplexity: cx,
				HasComplexity:  hasCx,
				ToldSet:        make([]string, n), // length carries n so mean-std's std = SE·√n
			},
		})
	}
	return stats, nil
}

// complexityMetricFor derives the complexity metric name from the objective metric's step
// prefix — `train.fold_score` → `train.complexity` (the naming convention the train step
// emits; #StepManifest was dropped, so this pairing is convention, metis#19).
func complexityMetricFor(metric string) string {
	if i := strings.Index(metric, "."); i > 0 {
		return metric[:i] + "." + foldComplexityMetric
	}
	return foldComplexityMetric
}

// resolveSelectRule picks the rule: an explicit --rule flag (with its param) overrides the
// shape's objective.select; otherwise the shape's rule is used as-is.
func resolveSelectRule(sh experiment.Shape, o selectOpts) (experiment.Select, error) {
	if o.rule == "" {
		return sh.Sweeper.Objective.Select, nil
	}
	switch o.rule {
	case "argmax-mean":
		return experiment.Select{ArgmaxMean: &experiment.ArgmaxMean{}}, nil
	case "one-std-err":
		return experiment.Select{OneStdErr: &experiment.OneStdErr{}}, nil
	case "pct-loss":
		tol := o.tolerance
		if tol == 0 && sh.Sweeper.Objective.Select.PctLoss != nil {
			tol = sh.Sweeper.Objective.Select.PctLoss.Tolerance
		}
		if tol == 0 {
			tol = 0.02
		}
		return experiment.Select{PctLoss: &experiment.PctLoss{Tolerance: tol}}, nil
	case "mean-std":
		lam := o.lambda
		if lam == 0 && sh.Sweeper.Objective.Select.MeanStd != nil {
			lam = sh.Sweeper.Objective.Select.MeanStd.Lambda
		}
		if lam == 0 {
			lam = 1.0
		}
		return experiment.Select{MeanStd: &experiment.MeanStd{Lambda: lam}}, nil
	default:
		return experiment.Select{}, fmt.Errorf("ledger select: unknown --rule %q; want argmax-mean | one-std-err | pct-loss | mean-std", o.rule)
	}
}

// printSelectResult renders the per-family robust winners (sorted) + the cross-family ship.
func printSelectResult(out io.Writer, sh experiment.Shape, rule experiment.Select, res sampler.SweepResult) {
	kind, _ := rule.Kind()
	fmt.Fprintf(out, "metis: select %s over %s (rule %s)\n", sh.ID, sh.Sweeper.Objective.Metric, kind)
	fams := make([]string, 0, len(res.PerFamily))
	for fam := range res.PerFamily {
		fams = append(fams, fam)
	}
	sort.Strings(fams)
	fmt.Fprintln(out, "  per-family robust winners:")
	for _, fam := range fams {
		w := res.PerFamily[fam]
		label := fam
		if label == "" {
			label = "(one family)"
		}
		fmt.Fprintf(out, "    %-22s %-28s  mean %.4f  cx %.1f\n", label, freeParamStrFromParams(w.Point.FreeParams), w.Score.Mean, w.Score.MeanComplexity)
	}
	w := res.Ship
	fmt.Fprintf(out, "metis: ship %s — mean %.4f (SE %.4f, cx %.1f)\n",
		freeParamStrFromParams(w.Point.FreeParams), w.Score.Mean, w.Score.SE, w.Score.MeanComplexity)
}
