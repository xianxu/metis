package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/record"
	"github.com/xianxu/metis/pkg/sampler"
	"github.com/xianxu/metis/pkg/shape"
)

// partitionStepID is the id of the engine-synthesized cv-split step. The resample is
// declared ONCE in sweeper.resample.cv (no cv-split in the shape); the engine
// materializes the partition from it and threads its identity into each per-fold run.
const partitionStepID = "cv-split"

// outerSplitStepID is the id of the metis#23 nested-CV preamble step that materializes the k
// outer-analysis subset dirs (analysis_0/ … analysis_{k-1}/) the sealed sweeps read.
const outerSplitStepID = "outer-split"

// foldMetric is the per-fold score the resample folds over — the metric the train step
// emits per (config, fold) run. The ledger keeps the raw per-fold rows under its
// namespaced form (`<train-step>.fold_score`); AggregateView reduces them to per-config
// (mean, SE). (Kept as the bare name here; run.Metrics is the flat merge.)
const foldMetric = "fold_score"

// foldComplexityMetric is the bare per-fold metric the train step emits alongside
// fold_score (metis#19): the fitted model's realized complexity (rf mean leaves / logreg
// coef count). runPipelineFold reads it into FoldOutcome; absent → HasComplexity stays
// false (the guard for a parsimony rule keys off this). Ledger-namespaced to
// `<train-step>.complexity`.
const foldComplexityMetric = "complexity"

// sweepManifest groups the point-runs an experiment-shape invocation produced. Its
// identity (ShapeRunID) filters the accumulating ledger (metis#8) by invocation /
// code-version; each PointRun row is a raw (config, fold) run. metis#18: a "point" is now
// one (config × fold) run of the nested Sampler loop, not a flat sweep point.
type sweepManifest struct {
	ShapeRunID string     `json:"shape_run_id"`
	Shape      string     `json:"shape"`
	Sampler    string     `json:"sampler"`
	Seed       int        `json:"seed"`
	Points     []pointRun `json:"points"`
}

type pointRun struct {
	RunID      string             `json:"run_id"` // = the (config,fold) point's PointAddress
	FreeParams map[string]any     `json:"free_params"`
	Fold       int                `json:"fold"` // metis#18: the resample-fold index
	Status     string             `json:"status"`
	Metrics    map[string]float64 `json:"metrics,omitempty"`
}

// configScore pairs an expanded config-point with its honest inner-resample estimate —
// captured during the sweep so the leaderboard prints every config's (mean, SE), not just
// the winner (the Sampler's Done returns only the winner).
type configScore struct {
	point  shape.Point
	meanSE sampler.MeanSE
}

// shapeSweep is the mutable accumulator the pure nested-Sampler loop folds through the IO
// shell: it drives each (config, fold) point-run through the shared cached runner, records
// the manifest + per-config estimates, and captures the first fatal error (the pure Run
// has no error channel, so a fatal fold sets ss.err and short-circuits the rest).
type shapeSweep struct {
	o             runOpts
	sh            experiment.Shape
	now           func() time.Time
	out           io.Writer
	shapeBlobHash string // metis#27: the shape .md's blob-hash — the intent-address term
	codeID        string // the frozen HEAD sha; a mid-sweep change detect-and-aborts
	partRef       sampler.PartitionRef
	man           sweepManifest
	configs       []configScore
}

// sweepPass accumulates ONE black-box sweeper run (the sweeper ⊃ resample loop): its per-config
// scores, manifest points, and first fatal error — PER-CALL, so a driver:cv outer fold's pass
// never bleeds its manifest/leaderboard into another's (metis#23). `baseRef` repoints the base
// dataset the pipeline reads: nil = the shape's declared base (the flat driver:single path, data
// phase + baseDatasetRef); non-nil = a sealed nested outer fold's `analysis_i` dir (the data phase
// is dropped — analysis_i is already adapted — and cv-split + pipeline read it). `readRoot` (abs)
// confines that pass's base-dataset reads via the exp_path chokepoint (empty = unconfined).
type sweepPass struct {
	ss       *shapeSweep
	baseRef  any
	readRoot string
	splitK   int                  // the cv-split / FixedKFolds fold count for this pass
	partRef  sampler.PartitionRef // this pass's partition identity (fed into each point's address)
	configs  []configScore
	points   []pointRun
	err      error
}

// runSweeper runs the black-box sweeper (GridConfigs ⊃ FixedKFolds) over configPts, folding each
// (config, fold) through the shared cached runner into `pass` — the sweeper's Done selects the
// winner by the metis#19 rule. Reused by BOTH driver:single (one flat pass) and driver:cv (one
// sealed pass per outer fold), so the select/reduce logic lives in ONE place (ARCH-DRY).
func (ss *shapeSweep) runSweeper(ctx sampler.Ctx, configPts []shape.Point, pass *sweepPass) sampler.SweepResult {
	return sampler.Run(ctx,
		sampler.GridConfigs{Points: configPts, Direction: ss.sh.Sweeper.Objective.Direction, Select: ss.sh.Sweeper.Objective.Select},
		func(c shape.Point) sampler.MeanSE {
			ms := sampler.Run(ctx, sampler.FixedKFolds{K: pass.splitK},
				func(f sampler.FoldPoint) sampler.FoldOutcome { return pass.runPipelineFold(c, f) })
			pass.configs = append(pass.configs, configScore{point: c, meanSE: ms})
			return ms
		})
}

// runShapeSweep drives the metis#18 nested Sampler loop: the sweeper (GridConfigs over the
// expanded pipeline configs) wraps the inner resample (FixedKFolds over the materialized
// partition); each (config, fold) runs {data + cv-split + pipeline} once through the shared
// cached runner (runResolvedExperiment), emitting one fold_score. The sweeper's Done selects
// the winner by the objective; driver:single ships it (M1a-5). Produces per-config (mean,SE)
// + the manifest + the raw per-fold ledger. Per-fold failure is fatal to the sweep (surfaced,
// not swallowed — a partial resample is not an honest estimate).
func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.Writer) error {
	// probeRepo's HEAD sha still drives the mid-sweep code-freeze guard (codeID) — NOT the
	// identity (metis#27 dropped repo_shas). The shape's blob-hash content-addresses the intent.
	_, sha, _ := probeRepo(o.git, filepath.Dir(o.expPath))
	sbh, _ := shapeBlobHash(o.expPath)

	configPts, err := shape.Expand(sh.Pipeline, 0)
	if err != nil {
		return fmt.Errorf("%s: %w", o.expPath, err)
	}
	// An empty config-space (e.g. a pipeline `$any: []`) has no winner — reject early with a
	// sharp diagnostic rather than let the sweeper's Done return a zero Winner whose nil Point
	// crashes the driver:single ship late (a pipeline step with a nil `with`).
	if len(configPts) == 0 {
		return fmt.Errorf("%s: shape %q expands to 0 configs — an empty sweep has no winner (check the pipeline's $any choices)", o.expPath, sh.ID)
	}
	k := sh.Sweeper.Resample.CV.K
	outerK := 0
	if sh.Driver.CV != nil {
		outerK = sh.Driver.CV.K
	}
	if o.dryRun {
		if outerK > 0 {
			// metis#23 nested-CV: ~outerK× the flat cost (outerK independent sealed sweeps +
			// outerK refit-and-score runs); the estimate is honest, but it ships NO winner.
			fmt.Fprintf(out, "metis: nested-CV %s — %d outer folds × (%d configs × %d inner folds) + %d refits ≈ %d×%d runs (dry run):\n",
				sh.ID, outerK, len(configPts), k, outerK, outerK, len(configPts)*k)
			fmt.Fprintf(out, "  (driver:cv estimates the tune-then-fit procedure — NO shippable winner; ship via driver:single)\n")
		} else {
			fmt.Fprintf(out, "metis: sweep %s — %d configs × %d folds (dry run):\n", sh.ID, len(configPts), k)
		}
		for i, p := range configPts {
			fmt.Fprintf(out, "  [%d] %s\n", i, freeParamStr(p))
		}
		return nil
	}

	shapeRunID, err := shapeRunIdentity(sh, sbh)
	if err != nil {
		return err
	}
	ss := &shapeSweep{
		o: o, sh: sh, now: now, out: out, shapeBlobHash: sbh, codeID: sha,
		partRef: partitionRef(sh),
		man:     sweepManifest{ShapeRunID: shapeRunID, Shape: sh.ID, Sampler: sh.Sweeper.Sampler, Seed: sh.Seed},
	}
	ctx := sampler.Ctx{Seed: sh.Seed, Partition: ss.partRef}

	// metis#23: driver:cv wraps the sweeper in an OUTER resample → the honest procedure estimate
	// (produces no winner). driver:single runs the sweeper once on all data and ships its winner.
	if outerK > 0 {
		return ss.runNestedCV(ctx, configPts, k, outerK, shapeRunID)
	}

	fmt.Fprintf(out, "metis: sweep %s (%s) — %d configs × %d folds\n", sh.ID, shapeRunID[:12], len(configPts), k)

	// The three-level nested fold — driver ⊃ sweeper ⊃ resample, each the SAME Sampler node
	// (metis#18). driver:single (SingleDriver) runs the sweeper once on all data and passes
	// its SweepResult through. The sweeper (GridConfigs) grids over configs, Done-selecting via
	// the metis#19 rule → a per-family winner map + the cross-family ship pick; the inner
	// FixedKFolds scores each config over k folds → (mean, SE, complexity).
	pass := &sweepPass{ss: ss, splitK: k, partRef: ss.partRef}
	res := sampler.Run(ctx, sampler.SingleDriver{}, func(sampler.SinglePoint) sampler.SweepResult {
		return ss.runSweeper(ctx, configPts, pass)
	})
	ss.man.Points = pass.points
	ss.configs = pass.configs
	if pass.err != nil {
		return pass.err
	}

	if err := writeManifest(o.expPath, ss.man); err != nil {
		return err
	}
	// Capture the sweep's code closure to a git side ref (metis#8/#14) — BEST-EFFORT: the
	// records + manifest are already valid, so a capture hiccup warns, never aborts.
	if err := captureSweepCode(o, ss.man); err != nil {
		fmt.Fprintf(out, "metis: warning: code capture failed (%v) — the sweep's records are valid but not committed to a side ref\n", err)
	}
	// Persist the raw per-fold rows to the shape's append-only ledger sidecar (metis#8/#18):
	// AggregateView reduces them read-time to per-config (mean, SE) — so metis#19's 1-SE
	// select re-reduces the same rows without a re-run.
	if err := writeSweepLedger(o.expPath, ss.man); err != nil {
		return err
	}
	// Guard (metis#19): a parsimony rule (one-std-err/pct-loss) needs a measured complexity
	// for every swept family — else the parsimony axis is silently dropped and the winner is
	// quietly wrong. The raw fold rows are already persisted (re-selectable after a fix); only
	// the ship/report is gated. Checked here (post-fold) because HasComplexity is only known
	// after the folds run.
	if err := sampler.GuardComplexity(sh.Sweeper.Objective.Select, ss.configStats()); err != nil {
		return err
	}
	ss.reportWinner(res)
	return ss.shipWinner(res.Ship)
}

// configStats builds the per-config stats (with each config's family) from the completed
// sweep — the guard's input, matching the shape GridConfigs.Done reduces internally.
func (ss *shapeSweep) configStats() []sampler.ConfigStat {
	stats := make([]sampler.ConfigStat, len(ss.configs))
	for i, c := range ss.configs {
		stats[i] = sampler.ConfigStat{Point: c.point, Family: sampler.FamilyOf(c.point), Score: c.meanSE}
	}
	return stats
}

// shipWinner runs the driver:single ship: reconstruct the winning config's runnable
// experiment (data ++ pipeline all-rows ++ ship — NO cv-split, the refit needs no CV) DIRECTLY
// from the Winner's resolved Point (metis#18 M1a-5 T19 — not by re-expanding the grid), refit
// it on ALL training rows, predict, and write the submission. A no-ship shape (leaderboard-only)
// is a clean no-op. Its run dir is content-addressed on the no-_fold config, so it's distinct
// from the per-fold runs and re-runs cache-HIT.
//
// The ship captures its OWN code closure (inSweep stays false → runResolvedExperiment calls
// captureSingleRun → refs/metis/runs/<shipRunID> + backfills its record). It must NOT ride the
// sweep's single capture: captureSweepCode ran BEFORE the ship existed and its closure is the
// UNION of the fold runs only (features/train/cv-split) — the ship-only steps (predict,
// submission) aren't in it, so a dirty ship would silently lose its durable SHA (metis#14). The
// ship is ONE run, so capturing it directly is correct + non-redundant (the inSweep optimization
// only exists to avoid N×k redundant per-FOLD captures).
func (ss *shapeSweep) shipWinner(w sampler.Winner) error {
	if len(ss.sh.Ship) == 0 {
		return nil
	}
	shipExp := shapeConfigToExperiment(ss.sh, w.Point)
	shipRunID, err := pointAddressOf(shipExp, ss.shapeBlobHash)
	if err != nil {
		return fmt.Errorf("ship winner %s: %w", freeParamStrFromParams(w.Point.FreeParams), err)
	}
	run, err := runResolvedExperiment(shipExp, ss.o, shipRunID, ss.now, ss.out)
	if err != nil {
		return fmt.Errorf("ship winner %s (%s): %w", freeParamStrFromParams(w.Point.FreeParams), shipRunID, err)
	}
	fmt.Fprintf(ss.out, "metis: shipped winner %s → runs/%s/ (%s)\n",
		freeParamStrFromParams(w.Point.FreeParams), shipRunID, run.Status)
	return nil
}

// runNestedCV drives driver:cv (metis#23): the OUTER resample around the black-box sweeper → the
// honest procedure estimate. A preamble materializes the k outer-analysis subset dirs ONCE; then
// the CVDriver, per outer fold, (a) runs the sweeper SEALED on analysis_i (confined via exp_path so
// its inner-CV cannot see outer-assessment) → a winner, then (b) refits+scores that winner on the
// held outer-assessment — a plain full-data fold run at OUTER k, held=i (post-selection, so
// unconfined and leakage-free; cv_folds's determinism reproduces the exact analysis_i partition).
// Aggregate(outer scores) → mean±SE: the estimate. It ships NO winner (estimation ≠ selection).
func (ss *shapeSweep) runNestedCV(ctx sampler.Ctx, configPts []shape.Point, innerK, outerK int, shapeRunID string) error {
	fmt.Fprintf(ss.out, "metis: nested-CV %s (%s) — %d outer folds × (%d configs × %d inner folds)\n",
		ss.sh.ID, shapeRunID[:12], outerK, len(configPts), innerK)

	// Preamble: materialize the outer-analysis subset dirs ONCE (unconfined — outer-split reads
	// the full dataset to split it). Deterministic run id → the analysis_i refs are locatable.
	analysisRefs, err := ss.materializeOuterAnalysis(outerK)
	if err != nil {
		return err
	}
	outerPart := sampler.PartitionRef(fmt.Sprintf("outer-cv-k%d-strat%t-seed%d", outerK, ss.sh.Driver.CV.Stratify, ss.sh.Seed))

	var firstErr error
	est := sampler.Run(ctx, sampler.CVDriver{K: outerK, Stratify: ss.sh.Driver.CV.Stratify},
		func(p sampler.OuterFoldPoint) float64 {
			if firstErr != nil {
				return 0
			}
			score, ferr := ss.runOuterFold(ctx, configPts, innerK, outerK, analysisRefs[p.Idx], outerPart, p.Idx)
			if ferr != nil {
				firstErr = ferr
				return 0
			}
			return score
		})
	if firstErr != nil {
		return firstErr
	}
	ss.reportEstimate(est, outerK)
	return nil
}

// materializeOuterAnalysis runs the nested-CV preamble ({data phase + outer-split(k=outerK)}) ONCE
// and returns the k analysis_i refs (experiment-relative, so a sealed sweep reading one routes
// through exp_path → confined). Unconfined (outer-split reads the full dataset to split it).
func (ss *shapeSweep) materializeOuterAnalysis(outerK int) ([]string, error) {
	baseOut, baseID := baseDatasetRef(ss.sh)
	var needs []string
	if baseID != "" {
		needs = []string{baseID}
	}
	osStep := experiment.Step{ID: outerSplitStepID, Uses: "metis/outer-split", Needs: needs,
		With: map[string]any{"dataset": baseOut, "k": outerK, "stratify": ss.sh.Driver.CV.Stratify}}
	steps := append(append([]experiment.Step{}, ss.sh.Data...), osStep)
	exp := experiment.Experiment{Header: ss.sh.Header, Steps: steps}
	exp.Type = "experiment"
	preID, err := pointAddressOf(exp, ss.shapeBlobHash)
	if err != nil {
		return nil, fmt.Errorf("nested-CV preamble: %w", err)
	}
	preOpts := ss.o
	preOpts.inSweep = true  // one preamble run; skip the per-run capture noise
	preOpts.readRoot = ""   // outer-split legitimately reads the full dataset
	if _, err := runResolvedExperiment(exp, preOpts, preID, ss.now, ss.out); err != nil {
		return nil, fmt.Errorf("nested-CV preamble (%s): %w", preID, err)
	}
	refs := make([]string, outerK)
	for i := 0; i < outerK; i++ {
		refs[i] = filepath.ToSlash(filepath.Join("runs", preID, outerSplitStepID, fmt.Sprintf("analysis_%d", i)))
	}
	return refs, nil
}

// runOuterFold runs one outer fold: (a) the SEALED sweeper on analysis_i → a winner (confined via
// the exp_path chokepoint — readRoot = analysis_i abs), then (b) the refit-and-score of that winner
// on the held outer-assessment (a full-data fold run at outer-k, held=i; unconfined). Returns the
// honest outer-fold score.
func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, innerK, outerK int, analysisRef string, outerPart sampler.PartitionRef, i int) (float64, error) {
	analysisAbs, err := filepath.Abs(filepath.Join(filepath.Dir(ss.o.expPath), analysisRef))
	if err != nil {
		return 0, err
	}
	// (a) sealed selection: the sweeper's inner-CV runs entirely within analysis_i.
	pass := &sweepPass{ss: ss, baseRef: analysisRef, readRoot: analysisAbs, splitK: innerK, partRef: ss.partRef}
	sres := ss.runSweeper(ctx, configPts, pass)
	if pass.err != nil {
		return 0, fmt.Errorf("outer fold %d sealed sweep: %w", i, pass.err)
	}
	winner := sres.Ship

	// (b) refit-and-score on the held outer-assessment — post-selection, so unconfined and honest.
	scoreExp := ss.buildFoldExperiment(winner.Point, sampler.FoldPoint{Idx: i}, nil, outerK, outerPart)
	scoreID, err := pointAddressOf(scoreExp, ss.shapeBlobHash)
	if err != nil {
		return 0, fmt.Errorf("outer fold %d score: %w", i, err)
	}
	scoreOpts := ss.o
	scoreOpts.inSweep = true
	scoreOpts.readRoot = "" // the outer-assessment eval reads full data legitimately
	run, err := runResolvedExperiment(scoreExp, scoreOpts, scoreID, ss.now, ss.out)
	if err != nil {
		return 0, fmt.Errorf("outer fold %d score (%s): %w", i, scoreID, err)
	}
	fmt.Fprintf(ss.out, "  outer fold %d: winner %s → held-out score %.4f\n",
		i, freeParamStrFromParams(winner.Point.FreeParams), run.Metrics[foldMetric])
	return run.Metrics[foldMetric], nil
}

// reportEstimate prints the honest procedure estimate — mean±SE over the outer folds — and the
// standing reminder that driver:cv produces NO shippable winner (estimation ≠ selection).
func (ss *shapeSweep) reportEstimate(est sampler.MeanSE, outerK int) {
	fmt.Fprintf(ss.out, "metis: nested-CV estimate — mean %.4f (SE %.4f) over %d outer folds — the HONEST procedure estimate\n",
		est.Mean, est.SE, outerK)
	fmt.Fprintf(ss.out, "  (driver:cv estimates the tune-then-fit procedure and ships NO winner; ship the config via driver:single)\n")
}

// runPipelineFold runs ONE (config, fold) point: build its per-fold experiment (data +
// synthesized cv-split + pipeline, with the config + fold-context overlaid), run it through
// the shared cached runner, record the manifest row, and return the fold_score the inner
// resample Sampler folds. A fatal outcome sets ss.err and returns 0 (the pure Run keeps
// going; runShapeSweep checks ss.err before using the winner).
func (p *sweepPass) runPipelineFold(c shape.Point, f sampler.FoldPoint) sampler.FoldOutcome {
	ss := p.ss
	if p.err != nil {
		return sampler.FoldOutcome{}
	}
	// Detect-and-abort: a mid-sweep HEAD-sha change breaks the shape-run's one-code
	// identity (per-fold records stay correct). Compares the HEAD sha only, not the dirty
	// flag — the sweep's own writes (runs/, manifest) dirty the tree (see codeID freeze).
	if _, s, _ := probeRepo(ss.o.git, filepath.Dir(ss.o.expPath)); s != ss.codeID {
		p.err = fmt.Errorf("code changed mid-sweep (%s → %s) — re-run to sweep the new revision", ss.codeID, s)
		return sampler.FoldOutcome{}
	}

	exp := ss.buildFoldExperiment(c, f, p.baseRef, p.splitK, p.partRef)
	runID, err := pointAddressOf(exp, ss.shapeBlobHash)
	if err != nil {
		p.err = fmt.Errorf("config %s fold %d: %w", freeParamStr(c), f.Idx, err)
		return sampler.FoldOutcome{}
	}
	pointOpts := ss.o
	pointOpts.inSweep = true      // metis#14: the sweep captures once (captureSweepCode), not per point
	pointOpts.readRoot = p.readRoot // metis#23: confine a sealed outer-fold pass to its analysis root
	run, runErr := runResolvedExperiment(exp, pointOpts, runID, ss.now, ss.out)
	// A failing fold is FATAL to the sweep, unlike a v1 flat point: a config scored over a
	// PARTIAL fold set is not an honest (mean, SE) estimate. Any error (a step failure, a
	// validation never-start, a persistence error) aborts — surfaced, never a half-scored config.
	if runErr != nil {
		p.err = fmt.Errorf("config %s fold %d (%s): %w", freeParamStr(c), f.Idx, runID, runErr)
		return sampler.FoldOutcome{}
	}
	p.points = append(p.points, pointRun{
		RunID:      runID,
		FreeParams: freeParamMap(c),
		Fold:       f.Idx,
		Status:     run.Status,
		Metrics:    run.Metrics,
	})
	// metis#19 M2: read the train step's realized-complexity metric. Present → the parsimony
	// rules consume it; absent (HasComplexity false) → the guard rejects a parsimony rule.
	cx, hasCx := run.Metrics[foldComplexityMetric]
	return sampler.FoldOutcome{Score: run.Metrics[foldMetric], Complexity: cx, HasComplexity: hasCx}
}

// buildFoldExperiment reconstructs the runnable per-fold experiment for one (config, fold):
// the data steps (as declared — cache-shared, config+fold-invariant) + the engine-synthesized
// cv-split partition step + the pipeline steps with the config's resolved `with` overlaid AND
// the fold-context injected. The fold-context ({_fold:{partition,idx}, folds:<cv-split>}) enters
// each pipeline step's `with` so its Kpre is fold-distinct (the B2 collision guard) and the step
// can read the fold assignment. Ship is NOT included (winner-only, M1a-5).
// baseRef nil = the flat driver:single path (data phase + cv-split over the declared base).
// baseRef non-nil = a sealed nested outer fold (metis#23): the data phase is DROPPED (analysis_i
// is already the adapted base) and cv-split + every pipeline step that read the declared base are
// repointed to baseRef (analysis_i), so their reads route through exp_path → confined to the
// outer-analysis root and the sweeper's inner-CV structurally cannot see outer-assessment.
func (ss *shapeSweep) buildFoldExperiment(c shape.Point, f sampler.FoldPoint, baseRef any, splitK int, partRef sampler.PartitionRef) experiment.Experiment {
	sh := ss.sh
	steps := make([]experiment.Step, 0, len(sh.Data)+1+len(sh.Pipeline))
	baseOut, baseID := baseDatasetRef(sh)
	var partNeeds []string
	if baseRef == nil {
		steps = append(steps, sh.Data...)
		if baseID != "" {
			partNeeds = []string{baseID}
		}
	} else {
		baseOut = baseRef // sealed: cv-split + pipeline read analysis_i, no data phase
	}
	steps = append(steps, cvSplitStep(sh, baseOut, partNeeds, splitK))
	dataIDs := dataStepIDs(sh)
	origOut, _ := baseDatasetRef(sh)
	for _, ps := range sh.Pipeline {
		s := ps
		s.With = foldWith(c.With[ps.ID], partRef, f.Idx)
		if baseRef != nil {
			// repoint a pipeline step that read the declared base → the sealed analysis_i,
			// and drop its now-absent data-step needs.
			if origOut != nil && fmt.Sprint(s.With["dataset"]) == fmt.Sprint(origOut) {
				s.With["dataset"] = baseRef
			}
			s.Needs = appendUnique(dropNeeds(ps.Needs, dataIDs), partitionStepID)
		} else {
			s.Needs = appendUnique(ps.Needs, partitionStepID)
		}
		steps = append(steps, s)
	}
	exp := experiment.Experiment{Header: sh.Header, Steps: steps}
	exp.Type = "experiment"
	return exp
}

// dataStepIDs is the set of the shape's data-phase step ids (dropped from a sealed pass).
func dataStepIDs(sh experiment.Shape) map[string]bool {
	ids := make(map[string]bool, len(sh.Data))
	for _, d := range sh.Data {
		ids[d.ID] = true
	}
	return ids
}

// dropNeeds returns needs with any id in `drop` removed (a sealed pass has no data steps).
func dropNeeds(needs []string, drop map[string]bool) []string {
	out := make([]string, 0, len(needs))
	for _, n := range needs {
		if !drop[n] {
			out = append(out, n)
		}
	}
	return out
}

// cvSplitStep synthesizes the cv-split step from sweeper.resample.cv (single-source — no
// cv-split in the shape): it splits `dataset` into k folds, writing folds.json the per-fold
// pipeline reads. `dataset`/`needs`/`k` are passed so BOTH the flat path (declared base, inner k)
// and metis#23's sealed sweep (analysis_i, inner k) / outer-score run (full base, OUTER k) reuse it.
func cvSplitStep(sh experiment.Shape, dataset any, needs []string, k int) experiment.Step {
	with := map[string]any{
		"dataset":  dataset,
		"k":        k,
		"stratify": sh.Sweeper.Resample.CV.Stratify,
	}
	return experiment.Step{ID: partitionStepID, Uses: "metis/cv-split", Needs: needs, With: with}
}

// baseDatasetRef returns the base-dataset the cv-split partitions: by convention the LAST
// data step produces it, and its `with.out` is the (experiment-relative) dataset path. Its
// id anchors cv-split's `needs`.
func baseDatasetRef(sh experiment.Shape) (out any, id string) {
	if len(sh.Data) == 0 {
		return nil, ""
	}
	last := sh.Data[len(sh.Data)-1]
	return last.With["out"], last.ID
}

// foldWith overlays the fold-context onto a config point's per-step `with`: the partition
// ref + fold index (so Kpre is fold-distinct and the step scores the one assessment fold)
// and the cv-split id (so the step reads folds.json via the upstream-artifact convention).
func foldWith(base map[string]any, partRef sampler.PartitionRef, idx int) map[string]any {
	w := make(map[string]any, len(base)+2)
	for k, v := range base {
		w[k] = v
	}
	w["folds"] = partitionStepID
	w["_fold"] = map[string]any{"partition": string(partRef), "idx": idx}
	return w
}

// appendUnique returns needs + extra (extra added only if absent), never mutating needs.
func appendUnique(needs []string, extra string) []string {
	out := make([]string, 0, len(needs)+1)
	out = append(out, needs...)
	for _, n := range needs {
		if n == extra {
			return out
		}
	}
	return append(out, extra)
}

// partitionRef is the stable identity of the materialized partition — deterministic per
// (k, stratify, seed) so every fold-point's address is reproducible. A later boundary can
// thread the cv-split's content-hash here; a deterministic id suffices for the told-set key.
func partitionRef(sh experiment.Shape) sampler.PartitionRef {
	cv := sh.Sweeper.Resample.CV
	return sampler.PartitionRef(fmt.Sprintf("cv-k%d-strat%t-seed%d", cv.K, cv.Stratify, sh.Seed))
}

// pointAddressOf pre-computes a (config, fold) run's content-address (== its run-dir id),
// minted from its FULL resolved config the SAME way buildRecord mints the record's address —
// so the manifest run_id and the record.json point_address can't desync (metis#8's handoff).
func pointAddressOf(exp experiment.Experiment, shapeBlobHash string) (string, error) {
	resolved := make(map[string]map[string]any, len(exp.Steps))
	for _, s := range exp.Steps {
		resolved[s.ID] = s.With
	}
	h, err := record.PointAddress(resolved, shapeBlobHash, exp.Seed)
	return string(h), err
}

// reportWinner prints the honest per-config (mean, SE, complexity) leaderboard (best-first
// by the objective), the per-family robust winners (metis#19), and the cross-family ship
// pick. Ship (refit + submission) is metis#18 M1a-5; here we report the selection.
func (ss *shapeSweep) reportWinner(res sampler.SweepResult) {
	fmt.Fprintf(ss.out, "metis: sweep %s done — %d configs scored (manifest %s)\n", ss.sh.ID, len(ss.configs), ss.man.ShapeRunID[:12])
	best := betterFirst(ss.configs, ss.sh.Sweeper.Objective.Direction)
	fmt.Fprintln(ss.out, "  config                          mean      SE       cx")
	for _, cs := range best {
		fmt.Fprintf(ss.out, "  %-30s  %.4f  %.4f  %6.1f\n", freeParamStr(cs.point), cs.meanSE.Mean, cs.meanSE.SE, cs.meanSE.MeanComplexity)
	}
	if len(res.PerFamily) > 1 {
		fams := make([]string, 0, len(res.PerFamily))
		for fam := range res.PerFamily {
			fams = append(fams, fam)
		}
		sort.Strings(fams)
		fmt.Fprintln(ss.out, "  per-family winners (metis#19):")
		for _, fam := range fams {
			w := res.PerFamily[fam]
			fmt.Fprintf(ss.out, "    %-22s %-24s  mean %.4f  cx %.1f\n", fam, freeParamStrFromParams(w.Point.FreeParams), w.Score.Mean, w.Score.MeanComplexity)
		}
	}
	w := res.Ship
	fmt.Fprintf(ss.out, "metis: winner %s — mean %.4f (SE %.4f, cx %.1f) over %d folds\n",
		freeParamStrFromParams(w.Point.FreeParams), w.Score.Mean, w.Score.SE, w.Score.MeanComplexity, len(w.FoldKeys))
}

// betterFirst returns the configs sorted best-first by the objective direction (a stable
// view; the live leaderboard, not a stored order).
func betterFirst(cs []configScore, direction string) []configScore {
	out := append([]configScore(nil), cs...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && betterMeanSE(out[j].meanSE.Mean, out[j-1].meanSE.Mean, direction); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func betterMeanSE(a, b float64, direction string) bool {
	if direction == "minimize" {
		return a < b
	}
	return a > b
}

// shapeRunIdentity mints the invocation identity that groups the sweep's point-runs:
// hash(shape id + phases + sweeper + repo SHAs + seed). The config × fold set is derivable
// from the shape, so the manifest stays thin.
func shapeRunIdentity(sh experiment.Shape, shapeBlobHash string) (string, error) {
	h, err := record.CanonicalHash(struct {
		Shape         string             `json:"shape"`
		Data          []experiment.Step  `json:"data"`
		Pipeline      []experiment.Step  `json:"pipeline"`
		Ship          []experiment.Step  `json:"ship"`
		Sweeper       experiment.Sweeper `json:"sweeper"`
		ShapeBlobHash string             `json:"shape_blob_hash"`
		Seed          int                `json:"seed"`
	}{sh.ID, sh.Data, sh.Pipeline, sh.Ship, sh.Sweeper, shapeBlobHash, sh.Seed})
	return string(h), err
}

func writeManifest(expPath string, man sweepManifest) error {
	dir := filepath.Join(filepath.Dir(expPath), "sweeps", man.ShapeRunID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "manifest.json"), append(b, '\n'), 0o644)
}

// probeRepo runs the injected gitProbe, degrading to empty provenance (like assembleRecord)
// when there's no repo — so a sweep outside git still runs.
func probeRepo(git gitProbe, dir string) (name, sha string, dirty bool) {
	if git == nil {
		git = gitCLI{}
	}
	n, s, d, err := git.Probe(dir)
	if err != nil {
		return "", "", false
	}
	return n, s, d
}

// freeParamMap renders a config point's free-param path as a {path: value} map (for the
// manifest + ledger); freeParamStr renders the same as a compact human string (for logs).
func freeParamMap(p shape.Point) map[string]any {
	m := make(map[string]any, len(p.FreeParams))
	for _, fp := range p.FreeParams {
		m[fp.Path] = fp.Value
	}
	return m
}

func freeParamStr(p shape.Point) string {
	return freeParamStrFromParams(p.FreeParams)
}

func freeParamStrFromParams(fps []shape.FreeParam) string {
	s := ""
	for i, fp := range fps {
		if i > 0 {
			s += " "
		}
		s += fmt.Sprintf("%s=%v", fp.Path, fp.Value)
	}
	if s == "" {
		return "(no free params)"
	}
	return s
}
