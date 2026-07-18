package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
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
	// metis#32 nested-CV recording: Level = "" (flat single-level CV row) | "inner" (a sealed
	// inner-CV fold row) | "outer" (a per-(outer-fold, family) held-out score). OuterFold is the
	// outer-fold coordinate (nil on the flat path). Both propagate into the ledger.Row.
	Level     string `json:"level,omitempty"`
	OuterFold *int   `json:"outer_fold,omitempty"`
}

// configScore pairs an expanded config-point with its honest inner-resample estimate —
// captured during the sweep so the leaderboard prints every config's (mean, SE), not just
// the winner (the Sampler's Done returns only the winner).
type configScore struct {
	point  shape.Point
	meanSE sampler.MeanSE
}

// shapeSweep is the mutable accumulator the pure nested-Sampler loop folds through the IO
// shell. Its runControl is the one failure authority across every nested pass.
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
	parallel      bool           // metis#31: >1 max-parallel ⇒ the sweeper/resample/driver batches run via ParExec
	manMu         sync.Mutex     // metis#32: guards man.Points — concurrent outer folds (ParExec) each record rows
	prog          *sweepProgress // metis#30: the live-progress sink (nil = silent)
	start         time.Time      // metis#50: sweep wall-clock start (injected clock)
}

// addManPoints appends a batch of manifest rows under the manifest lock (metis#32: the
// nested run's outer folds run concurrently under ParExec, each recording its inner+outer rows).
func (ss *shapeSweep) fail(label string, err error) error {
	return ss.o.runControl.fail(label, err)
}

func (ss *shapeSweep) firstError() error {
	return ss.o.runControl.firstError()
}

func (ss *shapeSweep) whileHealthy(fn func()) bool {
	return ss.o.runControl.whileHealthy(fn)
}

func (ss *shapeSweep) addManPoints(pts []pointRun) bool {
	return ss.whileHealthy(func() {
		ss.manMu.Lock()
		defer ss.manMu.Unlock()
		ss.man.Points = append(ss.man.Points, pts...)
	})
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
	// splitK vs runK (metis#58) — the two roles the fold count used to conflate:
	// splitK is the PARTITION count (cvSplitStep, partitionRef → each leaf's content-address);
	// runK is the folds ENUMERATED this pass (FixedKFolds{K: runK} — a 0..runK-1 prefix of the
	// SAME partition). Equal everywhere except an inner-sampled run (--sample inN), which is
	// what keeps subset runs address-compatible with full runs (cache escalation).
	// partitionRef/buildFoldExperiment must NEVER see runK.
	splitK   int
	runK     int
	stratify bool // the cv-split stratify flag for this pass
	partRef  sampler.PartitionRef // this pass's partition identity (fed into each point's address)
	runRole  runRole              // concrete-run role for every pipeline fold in this pass
	hooks    passHooks            // metis#30: this pass's progress hooks, closure-bound to its outer fold
	// metis#31: under ParExec the sweeper fans out over configs and each config's
	// resample fans out over folds — all appending to this ONE pass. `mu` guards the
	// orchestration bookkeeping (configs/points); the honest reduce stays in the
	// sampler's pure Tell/Done, not here.
	mu      sync.Mutex
	configs []configScore
	points  []pointRun
}

func (p *sweepPass) setErr(label string, err error) error {
	return p.ss.fail(label, err)
}

func (p *sweepPass) firstError() error {
	return p.ss.firstError()
}

// addConfigScore / addPoint append the per-config estimate / per-fold row under the
// pass lock (concurrent under ParExec).
func (p *sweepPass) addConfigScore(cs configScore) bool {
	return p.ss.whileHealthy(func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.configs = append(p.configs, cs)
	})
}

func (p *sweepPass) addPoint(pr pointRun) bool {
	return p.ss.whileHealthy(func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.points = append(p.points, pr)
	})
}

// runSweeper runs the black-box sweeper (GridConfigs ⊃ FixedKFolds) over configPts, folding each
// (config, fold) through the shared cached runner into `pass` — the sweeper's Done selects the
// winner by the metis#19 rule. Reused by BOTH driver:single (one flat pass) and driver:cv (one
// sealed pass per outer fold), so the select/reduce logic lives in ONE place (ARCH-DRY).
func (ss *shapeSweep) runSweeper(ctx sampler.Ctx, configPts []shape.Point, pass *sweepPass) sampler.SweepResult {
	return sampler.Run(ctx,
		sampler.GridConfigs{Points: configPts, Direction: ss.sh.Sweeper.Objective.Direction, Select: ss.sh.Sweeper.Objective.Select},
		func(c shape.Point) sampler.MeanSE {
			ms := sampler.Run(ctx, sampler.FixedKFolds{K: pass.runK},
				func(f sampler.FoldPoint) sampler.FoldOutcome { return pass.runPipelineFold(c, f) },
				sampler.ExecFor[sampler.FoldPoint, sampler.FoldOutcome](ss.parallel), func(ev sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]) {
					ss.whileHealthy(func() { pass.hooks.fold(ev) })
				})
			pass.addConfigScore(configScore{point: c, meanSE: ms})
			return ms
		},
		sampler.ExecFor[shape.Point, sampler.MeanSE](ss.parallel), func(ev sampler.ProgressEvent[shape.Point, sampler.MeanSE]) {
			ss.whileHealthy(func() { pass.hooks.config(ev) })
		})
}

// runShapeSweep drives the metis#18 nested Sampler loop: the sweeper (GridConfigs over the
// expanded pipeline configs) wraps the inner resample (FixedKFolds over the materialized
// partition); each (config, fold) runs {data + cv-split + pipeline} once through the shared
// cached runner (runResolvedExperiment), emitting one fold_score. The sweeper's Done selects
// the winner by the objective; driver:single ships it (M1a-5). Produces per-config (mean,SE)
// + the manifest + the raw per-fold ledger. Per-fold failure is fatal to the sweep (surfaced,
// not swallowed — a partial resample is not an honest estimate).
func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.Writer) (result error) {
	sweepStart := now() // metis#50: the run-end summary reports wall-clock elapsed
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
	// metis#32: the run mode is DERIVED from the config count, not a declared `driver:` field.
	// >1 config → nested CV (the honest per-family measure); ==1 config → a flat single-level CV
	// (the nested outer loop has one candidate, but the flat path is the cheaper distinct code —
	// a plain k-fold of the one config on ALL data, not the nested subset-sweep). The outer folds
	// reuse the sweeper's inner cv.k; `--fast` runs one outer fold (a 1/k holdout, ~1/k the cost)
	// for iteration. Neither path SHIPS — `metis run` only measures; shipping is `metis select --promote`.
	k := sh.Sweeper.Resample.CV.K
	stratify := sh.Sweeper.Resample.CV.Stratify
	nested := len(configPts) > 1
	// metis#45: k is the ESTIMAND knob (outer fold count + inner default); inner_k overrides
	// the INNER per-config CV only. A flat run's CV IS the reported estimate, so inner_k is a
	// NESTED-only knob — flat ignores it, loudly (once), and runs at k.
	innerK := sh.Sweeper.Resample.CV.InnerFolds()
	splitFolds := k // the per-config CV fold count THIS run actually uses
	if nested {
		splitFolds = innerK
	} else if innerK != k {
		fmt.Fprintf(out, "metis: inner_k ignored — a flat (single-config) run has no inner level; the %d-fold CV is the estimand\n", k)
	}
	runFolds := k
	// runInnerK inits from splitFolds, NOT innerK: on a FLAT run the per-config CV runs at k
	// (inner_k ignored-loudly above) and this value feeds the board denominators — innerK here
	// would display inner_k folds for a k-fold flat run. Identical to innerK on the nested path.
	runInnerK := splitFolds
	if o.sample.Out != 0 || o.sample.In != 0 {
		// metis#42 (outer) + metis#58 (inner): prefix fold sampling at BOTH levels. The
		// partitions are ALWAYS split at the declared counts (k / inner_k are the estimand —
		// the train fraction each fold simulates); out<M>/in<N> just run prefix subsets of
		// them (each fold an unbiased sample of the estimand; the seeded partition makes the
		// 0..M-1 prefix a valid random subset, and prefix subsets share leaf content-addresses
		// with full runs — iteration cache-escalates into decision runs). Misuse fails loudly.
		// The < 1 guards STAY despite the CLI parser rejecting 0: runOpts is a
		// direct-construction seam (every e2e builds it without the parser), and a negative
		// count would reach make([]…, -1) and panic in CVDriver.Init/FixedKFolds.Init.
		if o.fast {
			return fmt.Errorf("run: --sample and --fast are mutually exclusive (--fast is shorthand for --sample out1)")
		}
		if !nested {
			return fmt.Errorf("run: --sample only applies to a nested (multi-config) run — this shape has 1 config, a flat CV with no outer/inner levels to sample")
		}
		if o.sample.Out != 0 && (o.sample.Out < 1 || o.sample.Out > k) {
			return fmt.Errorf("run: --sample out%d out of range — want 1 ≤ M ≤ k=%d (the outer partition has exactly k folds)", o.sample.Out, k)
		}
		if o.sample.In != 0 && (o.sample.In < 1 || o.sample.In > innerK) {
			return fmt.Errorf("run: --sample in%d out of range — want 1 ≤ N ≤ inner_k=%d (the inner partition has exactly inner_k folds)", o.sample.In, innerK)
		}
		if o.sample.Out != 0 {
			runFolds = o.sample.Out
		}
		if o.sample.In != 0 {
			runInnerK = o.sample.In
		}
	} else if o.fast {
		runFolds = 1
	}
	if o.dryRun {
		if nested {
			fmt.Fprintf(out, "metis: nested-CV %s — %s outer fold(s) × (%d configs × %s inner folds) (dry run):\n",
				sh.ID, fmtLevel(runFolds, k), len(configPts), fmtLevel(runInnerK, innerK))
			fmt.Fprintf(out, "  (measures the honest per-family estimate + records inner/outer rows; ship via `metis select --promote`)\n")
		} else {
			fmt.Fprintf(out, "metis: single-level CV %s — %d config × %d folds (dry run):\n", sh.ID, len(configPts), k)
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
		o: o, sh: sh, now: now, out: out, shapeBlobHash: sbh, codeID: sha, start: sweepStart,
		partRef:  partitionRef(sh, splitFolds),
		man:      sweepManifest{ShapeRunID: shapeRunID, Shape: sh.ID, Sampler: sh.Sweeper.Sampler, Seed: sh.Seed},
		parallel: o.maxParallel > 1, // metis#31: fan out the sweeper/resample/driver batches
	}
	ctx := sampler.Ctx{Seed: sh.Seed, Partition: ss.partRef}
	// metis#30: seed the sink's denominators AT WIRING TIME from the same SizeHint the
	// levels report (stream-learned totals would arrive only with each level's first
	// completion — for the driver level that's the first COMPLETED outer fold, too late).
	// metis#58: the fold-level denominator is runInnerK (what this run will actually do),
	// not splitFolds (the partition count) — equal unless inner-sampled.
	ss.prog = newSweepProgress(out, now, sh.Sweeper.Objective.Direction, seededTotals(ctx, nested, runFolds, configPts, runInnerK))
	ss.o.activity = teeActivityEmitter(ss.o.activity, ss.prog.activity)
	// metis#38: board mode — the sink paints the pinned board instead of plain lines,
	// and a 500ms ticker keeps the rate decay + ETA live between events (sink-first:
	// tick() locks sp.mu then hands the frame to bw — the one global lock order).
	if o.board != nil {
		ss.prog.bw = o.board
		ss.prog.width = boardWidth()
		ss.prog.gauge = o.leafGauge
		tickC := o.boardTick
		var ticker *time.Ticker
		if tickC == nil {
			ticker = time.NewTicker(500 * time.Millisecond)
			tickC = ticker.C
		}
		tickDone := make(chan struct{})
		tickStopped := make(chan struct{})
		go func() {
			defer close(tickStopped)
			for {
				select {
				case <-tickC:
					if o.beforeBoardTick != nil {
						o.beforeBoardTick()
					}
					ss.whileHealthy(func() { ss.prog.tick() })
					if o.afterBoardTick != nil {
						o.afterBoardTick()
					}
				case <-tickDone:
					return
				}
			}
		}()
		defer func() {
			if ticker != nil {
				ticker.Stop()
			}
			close(tickDone)
			<-tickStopped
			if result != nil {
				ss.prog.abort()
			}
		}()
	}

	// metis#32: >1 config → nested CV (records inner + per-family outer rows; the honest measure).
	if nested {
		return ss.runNestedCV(ctx, configPts, k, innerK, runFolds, runInnerK, stratify, shapeRunID)
	}

	fmt.Fprintf(out, "metis: single-level CV %s (%s) — %d config × %d folds\n", sh.ID, shapeRunID[:12], len(configPts), k)

	// The flat single-level CV path (1 config): the SingleDriver (a runtime sampler node, NOT the
	// deleted shape `driver:`) runs the sweeper once on all data → the sweeper's inner k-fold CV
	// scores the one config → (mean, SE, complexity) recorded to the ledger. metis#32: it MEASURES
	// ONLY — no `shipWinner` (shipping is `metis select --promote`).
	pass := &sweepPass{ss: ss, splitK: k, runK: k, stratify: stratify, partRef: ss.partRef, runRole: runRoleFlatCV,
		hooks: ss.prog.forPass(-1)} // metis#30: the flat path's single pass
	res := sampler.Run(ctx, sampler.SingleDriver{}, func(sampler.SinglePoint) sampler.SweepResult {
		return ss.runSweeper(ctx, configPts, pass)
	}, sampler.ExecFor[sampler.SinglePoint, sampler.SweepResult](ss.parallel), nil)
	if err := ss.firstError(); err != nil {
		return err
	}
	ss.whileHealthy(ss.prog.finish) // metis#30: the terminal progress line, before the report
	// metis#31: sort the fan-out's completion-order bookkeeping to a stable content key
	// BEFORE persisting, so manifest.json + the ledger are byte-deterministic across
	// serial/parallel runs (the winner/estimate are already deterministic; this makes
	// the on-disk artifacts match metis's content-addressing posture).
	sortPointRuns(pass.points)
	ss.man.Points = pass.points
	ss.configs = pass.configs
	if err := writeManifest(o.expPath, ss.man); err != nil {
		return ss.fail("write sweep manifest", err)
	}
	// Capture the sweep's code closure to a git side ref (metis#8/#14) — BEST-EFFORT: the
	// records + manifest are already valid, so a capture hiccup warns, never aborts.
	cohort, err := captureSweepCode(o, ss.man)
	if err != nil {
		ss.whileHealthy(func() {
			fmt.Fprintf(out, "metis: warning: code capture failed (%v) — the sweep's records are valid but not committed to a side ref\n", err)
		})
	}
	// Persist the raw per-fold rows to the shape's append-only ledger sidecar (metis#8/#18):
	// AggregateView reduces them read-time to per-config (mean, SE) — so metis#19's 1-SE
	// select re-reduces the same rows without a re-run.
	if err := writeSweepLedger(o.expPath, ss.man); err != nil {
		return ss.fail("write sweep ledger", err)
	}
	// Guard (metis#19): a parsimony rule (one-std-err/pct-loss) needs a measured complexity
	// for every swept family — else the parsimony axis is silently dropped and the winner is
	// quietly wrong. The raw fold rows are already persisted (re-selectable after a fix); only
	// the ship/report is gated. Checked here (post-fold) because HasComplexity is only known
	// after the folds run.
	if err := sampler.GuardComplexity(sh.Sweeper.Objective.Select, configStatsOf(ss.configs)); err != nil {
		return ss.fail("sweep complexity guard", err)
	}
	ss.whileHealthy(func() {
		ss.reportWinner(res)
		printRunSummary(summaryWriter(out), o.expPath, now().Sub(sweepStart), len(ss.man.Points), cohort)
	})
	return ss.firstError()
}

// fmtLevel renders a fold-level count for banners: the plain declared count when the run
// covers the whole level, "run/declared" when sampled (metis#58). --fast renders as 1/k —
// it IS a sampled run (--sample out1 equivalent).
func fmtLevel(run, total int) string {
	if run == total {
		return fmt.Sprintf("%d", total)
	}
	return fmt.Sprintf("%d/%d", run, total)
}

// configStatsOf builds the per-config stats (with each config's family) from a completed
// sweep pass — the GuardComplexity input, matching what GridConfigs.Done reduces internally.
// Free over a []configScore so BOTH the flat path (ss.configs) and each driver:cv sealed
// outer fold (pass.configs) guard the same way (ARCH-DRY, metis#23 I1).
func configStatsOf(configs []configScore) []sampler.ConfigStat {
	stats := make([]sampler.ConfigStat, len(configs))
	for i, c := range configs {
		stats[i] = sampler.ConfigStat{Point: c.point, Family: sampler.FamilyOf(c.point), Score: c.meanSE}
	}
	return stats
}

// (metis#32: `shipWinner` was deleted — `metis run` no longer ships; the ship path moved to
// `metis select --promote`, which reconstructs the honest winner via `promotedExperiment` and runs
// it on all data. `shapeConfigToExperiment` (the all-data assembly) is now called from there.)

// runNestedCV drives driver:cv (metis#23): the OUTER resample around the black-box sweeper → the
// honest procedure estimate. A preamble materializes the k outer-analysis subset dirs ONCE; then
// the CVDriver, per outer fold, (a) runs the sweeper SEALED on analysis_i (confined via exp_path so
// its inner-CV cannot see outer-assessment) → a winner, then (b) refits+scores that winner on the
// held outer-assessment — a plain full-data fold run at OUTER k, held=i (post-selection, so
// unconfined and leakage-free; cv_folds's determinism reproduces the exact analysis_i partition).
// Aggregate(outer scores) → mean±SE: the estimate. It ships NO winner (estimation ≠ selection).
//
// PROVENANCE (deliberate, metis#23): the nested path writes NO grouped sweepManifest / ledger and
// does NO captureSweepCode. Each inner run's record.json still exists (via runResolvedExperiment),
// but a driver:cv run is estimation-only — it produces no shippable/reproducible winner — so the
// flat path's manifest+ledger+code-side-ref provenance (which exists to re-select/ship a winner
// without a re-run) has no consumer here. If a durable procedure-estimate provenance is later
// wanted (e.g. to compare estimates across code revisions), wire a thin nested manifest then.
func (ss *shapeSweep) runNestedCV(ctx sampler.Ctx, configPts []shape.Point, k, innerK, runFolds, runInnerK int, stratify bool, shapeRunID string) error {
	fmt.Fprintf(ss.out, "metis: nested-CV %s (%s) — %s outer fold(s) × (%d configs × %s inner folds)\n",
		ss.sh.ID, shapeRunID[:12], fmtLevel(runFolds, k), len(configPts), fmtLevel(runInnerK, innerK))

	// Preamble: materialize the k outer-analysis subset dirs ONCE (unconfined — outer-split reads
	// the full dataset to split it). Always split into k dirs (a stable partition); --fast just runs
	// fewer of them (runFolds ≤ k). Deterministic run id → the analysis_i refs are locatable.
	analysisRefs, err := ss.materializeOuterAnalysis(k, stratify)
	if err != nil {
		if first := ss.firstError(); first != nil {
			return first
		}
		return ss.fail("nested-CV preamble", err)
	}
	outerPart := sampler.PartitionRef(fmt.Sprintf("outer-cv-k%d-strat%t-seed%d", k, stratify, ss.sh.Seed))

	est := sampler.Run(ctx, sampler.CVDriver{K: runFolds, Stratify: stratify},
		func(p sampler.OuterFoldPoint) float64 {
			if ss.firstError() != nil {
				return 0
			}
			score, ferr := ss.runOuterFold(ctx, configPts, k, innerK, runInnerK, stratify, analysisRefs[p.Idx], outerPart, p.Idx)
			if ferr != nil {
				if ss.firstError() == nil {
					ss.fail(fmt.Sprintf("outer fold %d", p.Idx), ferr)
				}
				return 0
			}
			return score
		},
		sampler.ExecFor[sampler.OuterFoldPoint, float64](ss.parallel),
		// metis#30: outer-fold completions always emit. Error-gated: once runControl
		// latches, remaining closures return sentinel zeros — don't fold those into
		// the displayed est (the run is aborting; a fake 0 would tank the line).
		func(ev sampler.ProgressEvent[sampler.OuterFoldPoint, float64]) {
			ss.whileHealthy(func() { ss.prog.driverEvent(ev) })
		})
	if err := ss.firstError(); err != nil {
		return err
	}
	ss.whileHealthy(ss.prog.finish) // metis#30: the terminal progress line, before the estimate report

	// metis#32: the nested run now RECORDS (unlike metis#23's estimation-only path) — persist the
	// inner + per-family outer rows accumulated in ss.man.Points so `metis select` can reduce them
	// (family from the outer rows, config from the inner rows). Sort to a stable content key first
	// (the outer folds appended concurrently under ParExec) for byte-deterministic artifacts.
	sortPointRuns(ss.man.Points)
	if err := writeManifest(ss.o.expPath, ss.man); err != nil {
		return ss.fail("write nested sweep manifest", err)
	}
	cohort, cerr := captureSweepCode(ss.o, ss.man)
	if cerr != nil {
		ss.whileHealthy(func() {
			fmt.Fprintf(ss.out, "metis: warning: code capture failed (%v) — the nested run's records are valid but not committed to a side ref\n", cerr)
		})
	}
	if err := writeSweepLedger(ss.o.expPath, ss.man); err != nil {
		return ss.fail("write nested sweep ledger", err)
	}
	ss.whileHealthy(func() {
		ss.reportEstimate(est, runFolds)
		printRunSummary(summaryWriter(ss.out), ss.o.expPath, ss.now().Sub(ss.start), len(ss.man.Points), cohort)
	})
	return ss.firstError()
}

// materializeOuterAnalysis runs the nested-CV preamble ({data phase + outer-split(k=outerK)}) ONCE
// and returns the k analysis_i refs (experiment-relative, so a sealed sweep reading one routes
// through exp_path → confined). Unconfined (outer-split reads the full dataset to split it).
func (ss *shapeSweep) materializeOuterAnalysis(outerK int, stratify bool) ([]string, error) {
	baseOut, baseID := baseDatasetRef(ss.sh)
	var needs []string
	if baseID != "" {
		needs = []string{baseID}
	}
	osStep := experiment.Step{ID: outerSplitStepID, Uses: "metis/outer-split", Needs: needs,
		With: map[string]any{"dataset": baseOut, "k": outerK, "stratify": stratify}}
	steps := append(append([]experiment.Step{}, ss.sh.Data...), osStep)
	exp := experiment.Experiment{Header: ss.sh.Header, Steps: steps}
	exp.Type = "experiment"
	preID, err := pointAddressOf(exp, ss.shapeBlobHash)
	if err != nil {
		return nil, ss.fail("nested-CV preamble address", err)
	}
	preOpts := ss.o
	preOpts.inSweep = true // one preamble run; skip the per-run capture noise
	preOpts.readRoot = ""  // outer-split legitimately reads the full dataset
	preOpts.runLabel = fmt.Sprintf("outer-analysis preamble (%s)", preID)
	preOpts.runRole = runRoleNestedPreamble
	if _, err := runResolvedExperiment(exp, preOpts, preID, ss.now, ss.out); err != nil {
		return nil, err
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
// metis#45: k = OUTER (held-out scoring partition, cv_folds determinism — the #23 invariant);
// innerK = the sealed sweep's per-config CV fold count.
func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, k, innerK, runInnerK int, stratify bool, analysisRef string, outerPart sampler.PartitionRef, i int) (float64, error) {
	analysisAbs, err := filepath.Abs(filepath.Join(filepath.Dir(ss.o.expPath), analysisRef))
	if err != nil {
		return 0, ss.fail(fmt.Sprintf("outer fold %d analysis path", i), err)
	}
	// (a) sealed selection: the sweeper's inner-CV runs entirely within analysis_i (inner k/stratify).
	pass := &sweepPass{ss: ss, baseRef: analysisRef, readRoot: analysisAbs, splitK: innerK, runK: runInnerK,
		stratify: stratify, partRef: ss.partRef,
		runRole: runRoleNestedInnerCV,
		hooks:   ss.prog.forPass(i)} // metis#30/#38: outer-fold identity via closure binding
	sres := ss.runSweeper(ctx, configPts, pass)
	if err := pass.firstError(); err != nil {
		return 0, err
	}
	// Guard (metis#19/#23 I1): the parsimony select rule needs a measured complexity for every
	// swept family — same guard the flat path runs before trusting its winner. Without it, a
	// parsimony-select + non-reporting-model shape would SILENTLY mis-select in each outer fold.
	if err := sampler.GuardComplexity(ss.sh.Sweeper.Objective.Select, configStatsOf(pass.configs)); err != nil {
		return 0, ss.fail(fmt.Sprintf("outer fold %d complexity guard", i), err)
	}

	// metis#32: record the sealed sweep's INNER rows (Level=inner, tagged with this outer fold).
	of := i
	rows := make([]pointRun, 0, len(pass.points)+len(sres.PerFamily))
	if !ss.whileHealthy(func() {
		for _, pr := range pass.points {
			pr.Level = "inner"
			pr.OuterFold = &of
			rows = append(rows, pr)
		}
	}) {
		return 0, errRunAborted
	}

	// (b) score EACH family's inner-winner on the held outer-assessment — post-selection, so
	// unconfined and leakage-free (each winner was selected SEALED within analysis_i; scoring on
	// the held-out fold never influenced that selection). One OUTER row per family → the honest
	// per-family measure `metis select` reduces (metis#32). The metis#23 estimate the CVDriver
	// aggregates stays the SHIP-family's outer score (the argmax-mean procedure's honest number).
	// The cv-split uses the OUTER k + stratify so cv_folds's determinism reproduces the exact
	// partition outer-split materialized (else the held fold ≠ analysis_i's assessment rows).
	shipFamily := sres.Ship.Family
	var shipScore float64
	for _, fam := range sortedFamilies(sres.PerFamily) {
		w := sres.PerFamily[fam]
		score, scoreID, status, ferr := ss.scoreOnOuterFold(w.Point, i, k, stratify, outerPart, fam)
		if ferr != nil {
			return 0, ferr
		}
		if !ss.whileHealthy(func() {
			rows = append(rows, pointRun{
				RunID:      scoreID,
				FreeParams: freeParamMap(w.Point),
				Fold:       of, // the outer fold this held-out score is on
				Level:      "outer",
				OuterFold:  &of,
				Status:     status,
				// Metrics filled read-time from the run's record.json (namespaced), like inner rows.
			})
			if fam == shipFamily {
				shipScore = score
			}
			fmt.Fprintf(ss.out, "  outer fold %d: %s winner %s → held-out %.4f\n",
				i, fam, freeParamStrFromParams(w.Point.FreeParams), score)
		}) {
			return 0, errRunAborted
		}
	}
	if !ss.addManPoints(rows) {
		return 0, errRunAborted
	}
	return shipScore, nil
}

// scoreOnOuterFold refit-and-scores one config's winner on the held outer-assessment fold i (a
// full-data fold run at outer-k; post-selection, so unconfined). Returns the held-out fold_score,
// the run id (→ its record.json carries the namespaced metric the ledger reads), and its status.
func (ss *shapeSweep) scoreOnOuterFold(point shape.Point, i, k int, stratify bool, outerPart sampler.PartitionRef, fam string) (float64, string, string, error) {
	scoreExp := ss.buildFoldExperiment(point, sampler.FoldPoint{Idx: i}, nil, k, stratify, outerPart)
	scoreID, err := pointAddressOf(scoreExp, ss.shapeBlobHash)
	if err != nil {
		return 0, "", "", ss.fail(fmt.Sprintf("outer fold %d family %s score address", i, fam), err)
	}
	scoreOpts := ss.o
	scoreOpts.inSweep = true
	scoreOpts.readRoot = "" // the outer-assessment eval reads full data legitimately
	scoreOpts.runLabel = fmt.Sprintf("outer fold %d family %s score (%s)", i, fam, scoreID)
	scoreOpts.runRole = runRoleOuterScore
	run, err := runResolvedExperiment(scoreExp, scoreOpts, scoreID, ss.now, ss.out)
	if err != nil {
		return 0, "", "", err
	}
	return run.Metrics[foldMetric], scoreID, run.Status, nil
}

// sortedFamilies returns the family keys of a per-family winner map in deterministic order
// (the recording + the returned ship-score must not depend on Go's random map iteration).
func sortedFamilies(perFamily map[string]sampler.Winner) []string {
	fams := make([]string, 0, len(perFamily))
	for fam := range perFamily {
		fams = append(fams, fam)
	}
	sort.Strings(fams)
	return fams
}

// reportEstimate prints the honest procedure estimate — mean±SE over the outer folds — and the
// standing reminder that driver:cv produces NO shippable winner (estimation ≠ selection).
func (ss *shapeSweep) reportEstimate(est sampler.MeanSE, outerK int) {
	out := summaryWriter(ss.out) // metis#55: the RESULT lands after the footer in board mode
	fmt.Fprintf(out, "metis: nested-CV estimate — mean %.4f (SE %.4f) over %d outer fold(s) — the HONEST procedure estimate (argmax-mean family)\n",
		est.Mean, est.SE, outerK)
	fmt.Fprintf(out, "  (per-family honest estimates recorded to the ledger; choose + ship via `metis select --best --promote`)\n")
}

// summaryWriter routes run-RESULT prints (metis#55): in board mode they land in the
// epilogue (flushed after the final frame at close — the terminal ends on the result);
// plain/redirected mode already prints last, so the writer passes through unchanged.
func summaryWriter(out io.Writer) io.Writer {
	if bw, ok := out.(*boardWriter); ok {
		return bw.epilogueWriter()
	}
	return out
}

// runPipelineFold runs ONE (config, fold) point: build its per-fold experiment (data +
// synthesized cv-split + pipeline, with the config + fold-context overlaid), run it through
// the shared cached runner, record the manifest row, and return the fold_score the inner
// resample Sampler folds. A fatal outcome publishes through the experiment-wide
// runControl and returns 0; every sampler callback/sink rejects placeholders after
// publication, and the top level returns the stored concrete cause.
func (p *sweepPass) runPipelineFold(c shape.Point, f sampler.FoldPoint) sampler.FoldOutcome {
	ss := p.ss
	if p.firstError() != nil {
		return sampler.FoldOutcome{}
	}
	// Detect-and-abort: a mid-sweep HEAD-sha change breaks the shape-run's one-code
	// identity (per-fold records stay correct). Compares the HEAD sha only, not the dirty
	// flag — the sweep's own writes (runs/, manifest) dirty the tree (see codeID freeze).
	// metis#31: only a DEFINITE sha change aborts — `s != ""`. probeRepo swallows any
	// probe error to "", and under parallel fan-out concurrent `git status` contends on
	// .git/index.lock so a transient probe failure is expected; treating "" as a change
	// would false-abort the whole honest run.
	if _, s, _ := probeRepo(ss.o.git, filepath.Dir(ss.o.expPath)); s != "" && s != ss.codeID {
		p.setErr(fmt.Sprintf("config %s fold %d", freeParamStr(c), f.Idx),
			fmt.Errorf("code changed mid-sweep (%s → %s) — re-run to sweep the new revision", ss.codeID, s))
		return sampler.FoldOutcome{}
	}

	exp := ss.buildFoldExperiment(c, f, p.baseRef, p.splitK, p.stratify, p.partRef)
	runID, err := pointAddressOf(exp, ss.shapeBlobHash)
	if err != nil {
		p.setErr(fmt.Sprintf("config %s fold %d", freeParamStr(c), f.Idx), err)
		return sampler.FoldOutcome{}
	}
	pointOpts := ss.o
	pointOpts.inSweep = true        // metis#14: the sweep captures once (captureSweepCode), not per point
	pointOpts.readRoot = p.readRoot // metis#23: confine a sealed outer-fold pass to its analysis root
	pointOpts.runLabel = fmt.Sprintf("config %s fold %d (%s)", freeParamStr(c), f.Idx, runID)
	pointOpts.runRole = p.runRole
	run, runErr := runResolvedExperiment(exp, pointOpts, runID, ss.now, ss.out)
	// A failing fold is FATAL to the sweep, unlike a v1 flat point: a config scored over a
	// PARTIAL fold set is not an honest (mean, SE) estimate. Any error (a step failure, a
	// validation never-start, a persistence error) aborts — surfaced, never a half-scored config.
	if runErr != nil {
		// runControl already published a concrete admitted-run failure. A queued or
		// late sibling returns errRunAborted; neither path may republish the sentinel.
		return sampler.FoldOutcome{}
	}
	if !p.addPoint(pointRun{
		RunID:      runID,
		FreeParams: freeParamMap(c),
		Fold:       f.Idx,
		Status:     run.Status,
		Metrics:    run.Metrics,
	}) {
		return sampler.FoldOutcome{}
	}
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
func (ss *shapeSweep) buildFoldExperiment(c shape.Point, f sampler.FoldPoint, baseRef any, splitK int, stratify bool, partRef sampler.PartitionRef) experiment.Experiment {
	sh := ss.sh
	steps := make([]experiment.Step, 0, len(sh.Data)+1+len(sh.Pipeline))
	baseOut, baseID := baseDatasetRef(sh)
	origOut := baseOut // the declared base, captured before the sealed branch reassigns baseOut
	var partNeeds []string
	if baseRef == nil {
		steps = append(steps, sh.Data...)
		if baseID != "" {
			partNeeds = []string{baseID}
		}
	} else {
		baseOut = baseRef // sealed: cv-split + pipeline read analysis_i, no data phase
	}
	steps = append(steps, cvSplitStep(baseOut, partNeeds, splitK, stratify))
	dataIDs := dataStepIDs(sh)
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
func cvSplitStep(dataset any, needs []string, k int, stratify bool) experiment.Step {
	with := map[string]any{
		"dataset":  dataset,
		"k":        k,
		"stratify": stratify,
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

// partitionRef is the stable identity of the materialized per-config CV partition —
// deterministic per (fold count, stratify, seed) so every fold-point's address is
// reproducible. metis#45: the fold count is the RESOLVED one this run uses (inner_k on a
// nested run, k on a flat run) — an absent inner_k mints the same string as ever. A later
// boundary can thread the cv-split's content-hash here.
func partitionRef(sh experiment.Shape, splitFolds int) sampler.PartitionRef {
	cv := sh.Sweeper.Resample.CV
	return sampler.PartitionRef(fmt.Sprintf("cv-k%d-strat%t-seed%d", splitFolds, cv.Stratify, sh.Seed))
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
	out := summaryWriter(ss.out) // metis#55: the flat run's RESULT is the winner board — lands after the footer

	fmt.Fprintf(out, "metis: sweep %s done — %d configs scored (manifest %s)\n", ss.sh.ID, len(ss.configs), ss.man.ShapeRunID[:12])
	best := betterFirst(ss.configs, ss.sh.Sweeper.Objective.Direction)
	fmt.Fprintln(out, "  config                          mean      SE       cx")
	for _, cs := range best {
		fmt.Fprintf(out, "  %-30s  %.4f  %.4f  %6.1f\n", freeParamStr(cs.point), cs.meanSE.Mean, cs.meanSE.SE, cs.meanSE.MeanComplexity)
	}
	if len(res.PerFamily) > 1 {
		fams := make([]string, 0, len(res.PerFamily))
		for fam := range res.PerFamily {
			fams = append(fams, fam)
		}
		sort.Strings(fams)
		fmt.Fprintln(out, "  per-family winners (metis#19):")
		for _, fam := range fams {
			w := res.PerFamily[fam]
			fmt.Fprintf(out, "    %-22s %-24s  mean %.4f  cx %.1f\n", fam, freeParamStrFromParams(w.Point.FreeParams), w.Score.Mean, w.Score.MeanComplexity)
		}
	}
	w := res.Ship
	fmt.Fprintf(out, "metis: winner %s — mean %.4f (SE %.4f, cx %.1f) over %d folds\n",
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

// sortPointRuns orders the per-fold rows by a stable content key (RunID, then Fold)
// so a parallel sweep's completion-order appends persist byte-identically to a serial
// run (metis#31). The winner/estimate are already order-independent (the sampler
// reduce); this makes the manifest + ledger artifacts match metis's content-addressing.
func sortPointRuns(pts []pointRun) {
	sort.SliceStable(pts, func(i, j int) bool {
		if pts[i].RunID != pts[j].RunID {
			return pts[i].RunID < pts[j].RunID
		}
		return pts[i].Fold < pts[j].Fold
	})
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

// printRunSummary is metis#50's run-end handoff: elapsed wall-clock, what landed where,
// and the paste-ready follow-up commands with the cohort fingerprint pre-filled — the
// operator should never scrape the scrollback to assemble a `metis select`. A degraded
// capture (no fingerprint) degrades honestly: `cohort ?` and un-pinned hints (a
// single-cohort ledger needs no pin).
func printRunSummary(out io.Writer, expPath string, elapsed time.Duration, rows int, cohort record.Hash) {
	base := filepath.Base(ledgerPath(expPath))
	if cohort == "" {
		fmt.Fprintf(out, "metis: done in %s — %d rows → %s (cohort ?)\n", fmtETA(elapsed), rows, base)
		fmt.Fprintf(out, "  next: metis select %s\n", expPath)
		fmt.Fprintf(out, "        metis select %s --best --promote\n", expPath)
		fmt.Fprintf(out, "        metis ledger fingerprints %s\n", expPath)
		return
	}
	fp := short(string(cohort))
	fmt.Fprintf(out, "metis: done in %s — %d rows → %s (cohort %s)\n", fmtETA(elapsed), rows, base, fp)
	fmt.Fprintf(out, "  next: metis select %s --fingerprint %s               # the honest pick\n", expPath, fp)
	fmt.Fprintf(out, "        metis select %s --fingerprint %s --best --promote   # materialize it\n", expPath, fp)
	fmt.Fprintf(out, "        metis ledger fingerprints %s                   # cohorts\n", expPath)
}
