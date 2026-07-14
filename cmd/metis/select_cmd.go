package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/ledger"
	"github.com/xianxu/metis/pkg/sampler"
	"github.com/xianxu/metis/pkg/shape"
)

// cmdSelect handles `metis select <shape.md> [--best | --best-per-model-class] [--promote]
// [--fingerprint HASH]` — metis#32's honest selector (retires `metis ledger select` +
// `metis promote`). It reads the nested-CV ledger a `metis run` produced and CHOOSES:
//   - the model FAMILY on the honest OUTER estimate (FamilySelect: lowest-SE-within-1-SE) — the
//     signal #23 measured and #32 now ACTUATES (not the optimistic inner-CV cross-family argmax);
//   - the CONFIG within that family on the inner CV (the metis#19 rule).
// `--promote` reconstructs the chosen config from the ledger and ships it on ALL data
// (`runs/best-{family}-{hash}/submission.csv`), printing the run id for `kaggle submit`.
func cmdSelect(args []string) error {
	fs := flag.NewFlagSet("select", flag.ContinueOnError)
	best := fs.Bool("best", false, "print/promote the single ship recommendation (family by honest outer estimate, config by inner CV)")
	perClass := fs.Bool("best-per-model-class", false, "print/promote one winner per model family (the metis#22 ensembling seam)")
	promote := fs.Bool("promote", false, "materialize the selected config(s): reconstruct from the ledger + run on ALL data → runs/best-{family}-{hash}/submission.csv; prints the run id(s)")
	fingerprint := fs.String("fingerprint", "", "restrict to one code-fingerprint (metis#27)")
	shapePath, flags, err := hoistShapePath(args)
	if err != nil {
		return fmt.Errorf("select: %w (usage: metis select <shape.md> [--best | --best-per-model-class] [--promote] [--fingerprint HASH])", err)
	}
	if err := fs.Parse(flags); err != nil {
		return err
	}
	if !*best && !*perClass {
		*best = true // default view = the single ship recommendation
	}
	return runSelect(selectOpts{
		shapePath: shapePath, best: *best, perClass: *perClass, promote: *promote,
		fingerprint: *fingerprint, stepPath: stepPath(shapePath), out: os.Stdout,
	})
}

type selectOpts struct {
	shapePath   string
	best        bool
	perClass    bool
	promote     bool
	fingerprint string
	stepPath    []string
	git         gitProbe                    // nil → gitCLI (production)
	now         func() time.Time            // nil → time.Now
	exec        experiment.StepExecutor     // test seam: injected into the --promote run (nil → production execStep)
	out         io.Writer
}

// distinctFingerprints returns the sorted set of code-fingerprints (8-char) present in the ledger —
// the multi-cohort check for select's join-soundness guard.
func distinctFingerprints(led ledger.Ledger) []string {
	seen := map[string]bool{}
	for _, r := range led.Rows {
		if r.CodeFingerprint == "" {
			continue
		}
		f := r.CodeFingerprint
		if len(f) > 8 {
			f = f[:8]
		}
		seen[f] = true
	}
	out := make([]string, 0, len(seen))
	for f := range seen {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// runSelect is the testable core: load shape + ledger, reduce the OUTER rows to per-family honest
// estimates + the INNER rows to per-family config winners, pick, print, and (--promote) ship.
func runSelect(o selectOpts) error {
	raw, err := os.ReadFile(o.shapePath)
	if err != nil {
		return err
	}
	sh, err := experiment.ParseShape(string(raw))
	if err != nil {
		return err
	}
	led, err := loadLedger(o.shapePath)
	if err != nil {
		return err
	}
	led = ledger.Filter(led, o.fingerprint)
	// Join soundness (metis#32 §"Join soundness" + workshop/lessons.md): the family↔config reduce pools
	// OUTER rows by family AND INNER rows by config — both MUST be within ONE code-version. A ledger is
	// append-only, so a re-run after a step's code changes adds a SECOND fingerprint cohort; reducing
	// across cohorts silently blends versions into one estimate — the exact silently-wrong-winner class
	// #32 exists to stop. With no explicit --fingerprint, refuse a multi-cohort ledger rather than guess.
	if o.fingerprint == "" {
		if fps := distinctFingerprints(led); len(fps) > 1 {
			return fmt.Errorf("select: %s spans %d code-fingerprint cohorts %v — a cross-version reduce would silently blend them; pin one with `--fingerprint <hash>` (or re-run `metis run %s` to refresh)",
				ledgerPath(o.shapePath), len(fps), fps, filepath.Base(o.shapePath))
		}
	}
	metric := sh.Sweeper.Objective.Metric
	direction := sh.Sweeper.Objective.Direction

	// Config side (inner rows): the metis#19 rule over the per-config inner-CV → each family's winner.
	perFamily, err := perFamilyConfigWinners(sh, led, metric)
	if err != nil {
		return err
	}
	if len(perFamily) == 0 {
		return fmt.Errorf("select: no scored configs in %s (objective %q) — run `metis run %s` first", ledgerPath(o.shapePath), metric, filepath.Base(o.shapePath))
	}

	// Family side (OUTER rows): the honest per-family estimate #32 selects on.
	est := familyEstimateFromLedger(sh, led, metric)

	var picks []familyPick // the (family, config, estimate) rows to report/promote
	if len(est) == 0 {
		// No outer rows: a `--fast` run still has them, so this is a flat 1-config/inner-only
		// ledger. A single family degenerates cleanly (nothing to select across); >1 family with
		// no honest estimate is a sharp error, not a silent inner-CV cross-family pick.
		if len(perFamily) > 1 {
			return fmt.Errorf("select: the ledger has %d model families but NO outer-CV rows — the honest per-family estimate is what `metis select` chooses on. Run the full nested `metis run %s` (a `--fast`/1-config ledger has no outer rows)", len(perFamily), filepath.Base(o.shapePath))
		}
		for fam, w := range perFamily { // exactly one family → trivial
			picks = append(picks, familyPick{family: fam, winner: w})
		}
	} else if o.perClass {
		for fam := range est {
			if w, ok := perFamily[fam]; ok {
				picks = append(picks, familyPick{family: fam, winner: w, est: est[fam], hasEst: true})
			}
		}
	} else { // --best: the single honest family pick
		fam, caveat, ok := sampler.FamilySelect(direction, est)
		if !ok {
			return fmt.Errorf("select: could not pick a family from the honest estimates (%d families)", len(est))
		}
		w, ok := perFamily[fam]
		if !ok {
			return fmt.Errorf("select: family %q has an honest outer estimate but no inner config winner (ledger inner/outer rows disagree) — re-run `metis run`", fam)
		}
		picks = append(picks, familyPick{family: fam, winner: w, est: est[fam], hasEst: true, caveat: caveat})
	}

	sort.Slice(picks, func(i, j int) bool { return picks[i].family < picks[j].family })
	printSelect(o.out, sh, est, picks, o.perClass)

	if o.promote {
		return promoteSelected(o, sh, picks)
	}
	return nil
}

// familyPick pairs a chosen family with its config winner + (when available) its honest estimate.
type familyPick struct {
	family string
	winner sampler.Winner
	est    sampler.MeanSE
	hasEst bool
	caveat string
}

// perFamilyConfigWinners runs the shape's metis#19 select rule over the INNER rows (Level != "outer":
// nested inner rows are Level="inner", a flat 1-config run's are Level="") → each family's config winner.
func perFamilyConfigWinners(sh experiment.Shape, led ledger.Ledger, metric string) (map[string]sampler.Winner, error) {
	inner := rowsExcludingLevel(led, "outer")
	agg := ledger.AggregateView(inner, metric)
	stats, err := configStatsFromLedger(sh, agg, metric)
	if err != nil {
		return nil, err
	}
	if len(stats) == 0 {
		return nil, nil
	}
	rule := sh.Sweeper.Objective.Select
	if err := sampler.GuardComplexity(rule, stats); err != nil {
		return nil, err
	}
	return sampler.SelectConfigs(rule, sh.Sweeper.Objective.Direction, sh.Seed, stats).PerFamily, nil
}

// familyEstimateFromLedger reduces the OUTER rows to per-family honest (mean±SE), deriving the family
// from each row's config the SAME way FamilyOf does (Expand + match → FamilyOf) so the family key
// agrees with perFamilyConfigWinners' keys.
func familyEstimateFromLedger(sh experiment.Shape, led ledger.Ledger, metric string) map[string]sampler.MeanSE {
	points, err := shape.Expand(sh.Pipeline, 0)
	if err != nil {
		return nil
	}
	familyOf := func(r ledger.Row) string {
		for _, p := range points {
			if freeParamsEqual(p, r.FreeParams) {
				return sampler.FamilyOf(p)
			}
		}
		return "" // stale/unmatched row → the implicit one-family bucket
	}
	return FamilyEstimate(led, metric, familyOf)
}

// rowsExcludingLevel returns the ledger rows NOT at `level` (used to read the inner/config rows,
// excluding the Level="outer" family-estimate rows).
func rowsExcludingLevel(l ledger.Ledger, level string) ledger.Ledger {
	var out ledger.Ledger
	for _, r := range l.Rows {
		if r.Level != level {
			out.Rows = append(out.Rows, r)
		}
	}
	return out
}

// printSelect reports the per-family honest estimates + the picked config(s) + the honesty caveat.
func printSelect(out io.Writer, sh experiment.Shape, est map[string]sampler.MeanSE, picks []familyPick, perClass bool) {
	fmt.Fprintf(out, "metis: select %s over %s (family by honest outer estimate, config by inner CV)\n", sh.ID, sh.Sweeper.Objective.Metric)
	if len(est) > 0 {
		fams := make([]string, 0, len(est))
		for f := range est {
			fams = append(fams, f)
		}
		sort.Strings(fams)
		fmt.Fprintln(out, "  per-family honest outer estimate (mean ± SE):")
		for _, f := range fams {
			fmt.Fprintf(out, "    %-24s mean %.4f  SE %.4f  (n=%d outer folds)\n", famLabel(f), est[f].Mean, est[f].SE, len(est[f].ToldSet))
		}
	}
	head := "ship recommendation"
	if perClass {
		head = "per-model-class winners"
	}
	fmt.Fprintf(out, "  %s:\n", head)
	for _, p := range picks {
		fmt.Fprintf(out, "    %-24s %s\n", famLabel(p.family), freeParamStrFromParams(p.winner.Point.FreeParams))
		if p.caveat != "" {
			fmt.Fprintf(out, "      caveat: %s\n", p.caveat)
		}
	}
}

func famLabel(f string) string {
	if f == "" {
		return "(one family)"
	}
	return f
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

// promoteSelected materializes each picked config: reconstruct from the ledger (promotedExperiment
// — free-params → experiment, all-data fit since no `_fold`) and run it through the shared runner
// into runs/best-{family}-{hash}/, printing the id for `kaggle submit --run <id>`.
func promoteSelected(o selectOpts, sh experiment.Shape, picks []familyPick) error {
	if len(sh.Ship) == 0 {
		return fmt.Errorf("select --promote: shape %q has an empty `ship:` — nothing to submit (need predict + submission steps to produce submission.csv)", sh.ID)
	}
	now := o.now
	if now == nil {
		now = time.Now
	}
	sbh, _ := shapeBlobHash(o.shapePath)
	fmt.Fprintln(o.out, "  promoted runs (kaggle submit --run <id>):")
	for _, p := range picks {
		exp, err := promotedExperiment(sh, freeParamMap(p.winner.Point))
		if err != nil {
			return fmt.Errorf("select --promote %s: %w", famLabel(p.family), err)
		}
		addr, err := pointAddressOf(exp, sbh)
		if err != nil {
			return fmt.Errorf("select --promote %s: %w", famLabel(p.family), err)
		}
		runID := "best-" + familyTag(p.family) + "-" + short(addr)
		ro := runOpts{expPath: o.shapePath, runID: runID, stepPath: o.stepPath, cache: true, git: o.git, exec: o.exec, out: o.out}
		if _, err := runResolvedExperiment(exp, ro, runID, now, o.out); err != nil {
			return fmt.Errorf("select --promote %s (%s): %w", famLabel(p.family), runID, err)
		}
		fmt.Fprintf(o.out, "    %s\n", runID)
	}
	return nil
}

// familyTag renders a family key (`train.model=rf`) as the short tag (`rf`) for the run-dir name;
// the implicit one-family bucket ("") → "config".
func familyTag(fam string) string {
	if i := strings.LastIndex(fam, "="); i >= 0 && i+1 < len(fam) {
		return fam[i+1:]
	}
	if fam == "" {
		return "config"
	}
	return fam
}
