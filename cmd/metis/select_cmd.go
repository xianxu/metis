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
//
// `--promote` reconstructs the chosen config from the ledger and ships it on ALL data
// (`runs/best-{family}-{hash}/submission.csv`), printing the run id for `kaggle submit`.
func cmdSelect(args []string) error {
	fs := flag.NewFlagSet("select", flag.ContinueOnError)
	best := fs.Bool("best", false, "print/promote the single ship recommendation (family by honest outer estimate, config by inner CV)")
	perClass := fs.Bool("best-per-model-class", false, "print/promote one winner per model family (the metis#22 ensembling seam)")
	promote := fs.Bool("promote", false, "materialize the selected config(s): reconstruct from the ledger + run on ALL data → runs/best-{family}-{hash}/submission.csv; prints the run id(s)")
	fingerprint := fs.String("fingerprint", "", "restrict to one code-fingerprint (metis#27)")
	point := fs.String("point", "", "metis#41: publish an OPERATOR-CHOSEN config by ledger row — a point_addr (git-style prefix ok); ships as point-{family}-{hash}. Mutually exclusive with --best/--best-per-model-class")
	cohort := fs.Bool("cohort", false, "metis#52: list the ledger's code-fingerprint cohorts and exit (the `metis ledger fingerprints` table, on select's surface)")
	shapePath, flags, err := hoistShapePath(args)
	if err != nil {
		return fmt.Errorf("select: %w (usage: metis select <shape.md> [--best | --best-per-model-class | --point ADDR] [--promote] [--fingerprint HASH])", err)
	}
	if err := fs.Parse(flags); err != nil {
		return err
	}
	if *cohort {
		// metis#52: a listing door where the operator's hands already are — pure
		// delegation to the #39 core (one implementation, two CLI surfaces).
		return showFingerprints(shapePath, os.Stdout)
	}
	if *point == "" && !*best && !*perClass {
		*best = true // default view = the single ship recommendation
	}
	return runSelect(selectOpts{
		shapePath: shapePath, best: *best, perClass: *perClass, promote: *promote, point: *point,
		fingerprint: *fingerprint, stepPath: stepPath(shapePath), out: os.Stdout,
	})
}

type selectOpts struct {
	shapePath   string
	best        bool
	perClass    bool
	promote     bool
	point       string // metis#41: point_addr (prefix) of an operator-chosen config; "" = rule-based
	fingerprint string
	stepPath    []string
	git         gitProbe                // nil → gitCLI (production)
	now         func() time.Time        // nil → time.Now
	exec        experiment.StepExecutor // test seam: injected into the --promote run (nil → production execStep)
	out         io.Writer
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
	led, err = pinFingerprint(o.shapePath, led, o.fingerprint) // metis#39: git-style prefix + honest errors
	if err != nil {
		return fmt.Errorf("select: %w", err)
	}
	// Join soundness (metis#32 §"Join soundness" + workshop/lessons.md): the family↔config reduce pools
	// OUTER rows by family AND INNER rows by config — both MUST be within ONE code-version. A ledger is
	// append-only, so a re-run after a step's code changes adds a SECOND fingerprint cohort; reducing
	// across cohorts silently blends versions into one estimate — the exact silently-wrong-winner class
	// #32 exists to stop. With no explicit --fingerprint, refuse a multi-cohort ledger rather than guess.
	// The count is a cheap pure predicate; the error path renders the full cohort summary (metis#39).
	if o.fingerprint == "" {
		if n := distinctFingerprintCount(led); n > 1 {
			return cohortGuardErr(o.shapePath, led, n)
		}
	}
	metric := sh.Sweeper.Objective.Metric

	// metis#41: --point publishes an operator-chosen config by ledger row — it bypasses the
	// rule-based selection entirely (the cohort guard above still applied).
	if o.point != "" {
		if o.best || o.perClass {
			return fmt.Errorf("select: --point is mutually exclusive with --best/--best-per-model-class — a point-select IS the choice")
		}
		return runPointSelect(o, sh, led, metric)
	}
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
	// metis#52: attach each pick's --point handle (the first cohort-filtered ledger row of
	// that config — any fold row is a valid handle by #41's resolver). Good practice made
	// mechanical: a concrete "best" config is always shown WITH its override handle.
	for i := range picks {
		picks[i].handle = pointHandleFor(led, picks[i].winner.Point)
	}
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
	handle string // metis#52: a representative ledger-row point_addr — the --point override handle
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
		h := ""
		if p.handle != "" {
			h = " · point " + short(p.handle) // metis#52: the --point override handle
		}
		fmt.Fprintf(out, "    %-24s %s%s\n", famLabel(p.family), freeParamStrFromParams(p.winner.Point.FreeParams), h)
		if p.caveat != "" {
			fmt.Fprintf(out, "      caveat: %s\n", p.caveat)
		}
	}
}

// pointHandleFor finds a representative ledger-row point_addr for a config (first match
// in append order) — "" when the config has no rows (then no handle is shown; never lie).
func pointHandleFor(led ledger.Ledger, p shape.Point) string {
	for _, r := range led.Rows {
		if freeParamsEqual(p, r.FreeParams) {
			return r.PointAddr
		}
	}
	return ""
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
		// no leafPins: a promoted ship is a SERIAL single all-data fit — multi-threaded
		// BLAS is wanted here, and one leaf can't oversubscribe (#48's conscious exclusion)
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

// ── metis#41: point-select — publish an operator-chosen config by ledger row ──

// resolvePointRows resolves a point_addr prefix against the (already cohort-filtered) ledger:
// the rows of EXACTLY ONE config (any of its fold rows works as a handle — they share the
// config's FreeParams). 0 configs → not-found; >1 → ambiguous, listing candidates. Pure.
func resolvePointRows(led ledger.Ledger, prefix string) ([]ledger.Row, error) {
	var hit []ledger.Row
	for _, r := range led.Rows {
		if strings.HasPrefix(r.PointAddr, prefix) {
			hit = append(hit, r)
		}
	}
	if len(hit) == 0 {
		return nil, fmt.Errorf("select --point: no ledger row's point_addr starts with %q (if the ledger spans cohorts, the row may be outside the pinned --fingerprint)", prefix)
	}
	// Group to distinct configs; expand hit → all rows of the one config if unique.
	var configs []map[string]any
	var sample []string // parallel to configs: a sample addr per config for the error text
	for _, r := range hit {
		found := false
		for _, c := range configs {
			if freeParamMapsEqual(c, r.FreeParams) {
				found = true
				break
			}
		}
		if !found {
			configs = append(configs, r.FreeParams)
			sample = append(sample, r.PointAddr)
		}
	}
	if len(configs) > 1 {
		var cands []string
		for i, c := range configs {
			cands = append(cands, fmt.Sprintf("%s (%s)", sample[i], freeParamMapStr(c)))
		}
		sort.Strings(cands)
		return nil, fmt.Errorf("select --point: prefix %q is ambiguous across %d configs — disambiguate:\n    %s",
			prefix, len(configs), strings.Join(cands, "\n    "))
	}
	// One config: return ALL its rows (not just the prefix hits) so the board line pools folds.
	var rows []ledger.Row
	for _, r := range led.Rows {
		if freeParamMapsEqual(configs[0], r.FreeParams) {
			rows = append(rows, r)
		}
	}
	return rows, nil
}

// freeParamMapStr renders a free-param MAP as the sorted `k=v` line (the map-side sibling of
// freeParamStrFromParams, which takes the Point's slice form).
func freeParamMapStr(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, m[k]))
	}
	return strings.Join(parts, " ")
}

// freeParamMapsEqual compares two free-param maps by fmt.Sprint value identity (the same
// tolerance freeParamsEqual applies between a Point and a map).
func freeParamMapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		w, ok := b[k]
		if !ok || fmt.Sprint(v) != fmt.Sprint(w) {
			return false
		}
	}
	return true
}

// runPointSelect prints the chosen config's board line (pooled inner mean±SE + outer rows) and,
// with --promote, reconstructs it from the row's FreeParams (promotedExperiment — the SAME path
// --best ships through) into runs/point-{family}-{hash}/ — the `point-` prefix marks the run as
// OPERATOR-chosen (vs the rule-chosen `best-`), so provenance keeps the two apart.
func runPointSelect(o selectOpts, sh experiment.Shape, led ledger.Ledger, metric string) error {
	rows, err := resolvePointRows(led, o.point)
	if err != nil {
		return err
	}
	config := rows[0].FreeParams
	// Family key via the same Expand+match → FamilyOf derivation the rule-based paths use.
	fam := ""
	if points, perr := shape.Expand(sh.Pipeline, 0); perr == nil {
		for _, p := range points {
			if freeParamsEqual(p, config) {
				fam = sampler.FamilyOf(p)
				break
			}
		}
	}
	// Board line via the SAME reduce every sibling selector uses (pkg/ledger.AggregateView —
	// ARCH-DRY, close-review finding 1): the per-fold inner rows collapse to one aggregate row
	// carrying metric/.se/.n and a `failed` Status marker; outer rows (Fold nil) pass through.
	var view ledger.Ledger
	view.Append(rows...)
	agg := ledger.AggregateView(view, metric)
	fmt.Fprintf(o.out, "metis: select --point %s → %s\n", o.point, freeParamMapStr(config))
	hasFailed := false
	for _, r := range agg.Rows {
		if r.Level == "inner" {
			if r.Status == "failed" {
				hasFailed = true
			}
			if mean, ok := r.Metrics[metric]; ok {
				fmt.Fprintf(o.out, "  pooled inner %s: mean %.4f  SE %.4f  (n=%.0f fold rows)\n",
					metric, mean, r.Metrics[metric+".se"], r.Metrics[metric+".n"])
			}
		}
	}
	for _, r := range agg.Rows {
		if r.Level == "outer" && r.OuterFold != nil {
			if s, ok := r.Metrics[metric]; ok {
				fmt.Fprintf(o.out, "  outer fold %d: %.4f\n", *r.OuterFold, s)
			}
		}
	}
	if hasFailed {
		// Loud-error discipline (close-review finding 2): sibling selectors (--best) SKIP a
		// failed config entirely — promoting one is an explicit operator override, so say so.
		fmt.Fprintf(o.out, "  warning: this config has FAILED fold rows (pooled estimate covers the emitted metrics only); --best would skip it — promoting is an explicit operator override\n")
	}
	if !o.promote {
		return nil
	}
	if len(sh.Ship) == 0 {
		return fmt.Errorf("select --point --promote: shape %q has an empty `ship:` — nothing to submit", sh.ID)
	}
	now := o.now
	if now == nil {
		now = time.Now
	}
	exp, err := promotedExperiment(sh, config)
	if err != nil {
		return fmt.Errorf("select --point --promote: %w", err)
	}
	sbh, _ := shapeBlobHash(o.shapePath)
	addr, err := pointAddressOf(exp, sbh)
	if err != nil {
		return fmt.Errorf("select --point --promote: %w", err)
	}
	runID := "point-" + familyTag(fam) + "-" + short(addr)
	// no leafPins: a promoted ship is a SERIAL single all-data fit — multi-threaded
	// BLAS is wanted here, and one leaf can't oversubscribe (#48's conscious exclusion)
	ro := runOpts{expPath: o.shapePath, runID: runID, stepPath: o.stepPath, cache: true, git: o.git, exec: o.exec, out: o.out}
	if _, err := runResolvedExperiment(exp, ro, runID, now, o.out); err != nil {
		return fmt.Errorf("select --point --promote (%s): %w", runID, err)
	}
	fmt.Fprintf(o.out, "  promoted (kaggle submit --run <id>):\n    %s\n", runID)
	return nil
}
