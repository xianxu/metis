package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// foldMetric is the per-fold score the resample folds over — the metric the train step
// emits per (config, fold) run. The ledger keeps the raw per-fold rows under its
// namespaced form (`<train-step>.fold_score`); AggregateView reduces them to per-config
// (mean, SE). (Kept as the bare name here; run.Metrics is the flat merge.)
const foldMetric = "fold_score"

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
	o        runOpts
	sh       experiment.Shape
	now      func() time.Time
	out      io.Writer
	repoSHAs map[string]string
	codeID   string // the frozen HEAD sha; a mid-sweep change detect-and-aborts
	partRef  sampler.PartitionRef
	man      sweepManifest
	configs  []configScore
	err      error
}

// runShapeSweep drives the metis#18 nested Sampler loop: the sweeper (GridConfigs over the
// expanded pipeline configs) wraps the inner resample (FixedKFolds over the materialized
// partition); each (config, fold) runs {data + cv-split + pipeline} once through the shared
// cached runner (runResolvedExperiment), emitting one fold_score. The sweeper's Done selects
// the winner by the objective; driver:single ships it (M1a-5). Produces per-config (mean,SE)
// + the manifest + the raw per-fold ledger. Per-fold failure is fatal to the sweep (surfaced,
// not swallowed — a partial resample is not an honest estimate).
func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.Writer) error {
	repoName, sha, _ := probeRepo(o.git, filepath.Dir(o.expPath))
	repoSHAs := repoSHAsOf(repoName, sha)

	configPts, err := shape.Expand(sh.Pipeline, 0)
	if err != nil {
		return fmt.Errorf("%s: %w", o.expPath, err)
	}
	k := sh.Sweeper.Resample.CV.K
	if o.dryRun {
		fmt.Fprintf(out, "metis: sweep %s — %d configs × %d folds (dry run):\n", sh.ID, len(configPts), k)
		for i, p := range configPts {
			fmt.Fprintf(out, "  [%d] %s\n", i, freeParamStr(p))
		}
		return nil
	}

	shapeRunID, err := shapeRunIdentity(sh, repoSHAs)
	if err != nil {
		return err
	}
	ss := &shapeSweep{
		o: o, sh: sh, now: now, out: out, repoSHAs: repoSHAs, codeID: sha,
		partRef: partitionRef(sh),
		man:     sweepManifest{ShapeRunID: shapeRunID, Shape: sh.ID, Sampler: sh.Sweeper.Sampler, Seed: sh.Seed},
	}
	ctx := sampler.Ctx{Seed: sh.Seed, Partition: ss.partRef}
	fmt.Fprintf(out, "metis: sweep %s (%s) — %d configs × %d folds\n", sh.ID, shapeRunID[:12], len(configPts), k)

	// The nested fold: driver:single runs the sweeper once on all data (the degenerate
	// outer Sampler — sampler.SingleDriver documents the seam metis#23's driver:cv replaces).
	// The sweeper grids over configs; the inner FixedKFolds scores each over k folds.
	winner := sampler.Run(ctx,
		sampler.GridConfigs{Points: configPts, Direction: sh.Sweeper.Objective.Direction, Select: sh.Sweeper.Objective.Select},
		func(c shape.Point) sampler.MeanSE {
			ms := sampler.Run(ctx, sampler.FixedKFolds{K: k},
				func(f sampler.FoldPoint) float64 { return ss.runPipelineFold(c, f) })
			ss.configs = append(ss.configs, configScore{point: c, meanSE: ms})
			return ms
		})
	if ss.err != nil {
		return ss.err
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
	if err := writeSweepLedger(o.expPath, ss.man, sh.Sweeper.Objective); err != nil {
		return err
	}
	ss.reportWinner(winner)
	return nil
}

// runPipelineFold runs ONE (config, fold) point: build its per-fold experiment (data +
// synthesized cv-split + pipeline, with the config + fold-context overlaid), run it through
// the shared cached runner, record the manifest row, and return the fold_score the inner
// resample Sampler folds. A fatal outcome sets ss.err and returns 0 (the pure Run keeps
// going; runShapeSweep checks ss.err before using the winner).
func (ss *shapeSweep) runPipelineFold(c shape.Point, f sampler.FoldPoint) float64 {
	if ss.err != nil {
		return 0
	}
	// Detect-and-abort: a mid-sweep HEAD-sha change breaks the shape-run's one-code
	// identity (per-fold records stay correct). Compares the HEAD sha only, not the dirty
	// flag — the sweep's own writes (runs/, manifest) dirty the tree (see codeID freeze).
	if _, s, _ := probeRepo(ss.o.git, filepath.Dir(ss.o.expPath)); s != ss.codeID {
		ss.err = fmt.Errorf("code changed mid-sweep (%s → %s) — re-run to sweep the new revision", ss.codeID, s)
		return 0
	}

	exp := ss.buildFoldExperiment(c, f)
	runID, err := pointAddressOf(exp, ss.repoSHAs)
	if err != nil {
		ss.err = fmt.Errorf("config %s fold %d: %w", freeParamStr(c), f.Idx, err)
		return 0
	}
	pointOpts := ss.o
	pointOpts.inSweep = true // metis#14: the sweep captures once (captureSweepCode), not per point
	run, runErr := runResolvedExperiment(exp, pointOpts, runID, ss.now, ss.out)
	// A failing fold is FATAL to the sweep, unlike a v1 flat point: a config scored over a
	// PARTIAL fold set is not an honest (mean, SE) estimate. Any error (a step failure, a
	// validation never-start, a persistence error) aborts — surfaced, never a half-scored config.
	if runErr != nil {
		ss.err = fmt.Errorf("config %s fold %d (%s): %w", freeParamStr(c), f.Idx, runID, runErr)
		return 0
	}
	ss.man.Points = append(ss.man.Points, pointRun{
		RunID:      runID,
		FreeParams: freeParamMap(c),
		Fold:       f.Idx,
		Status:     run.Status,
		Metrics:    run.Metrics,
	})
	return run.Metrics[foldMetric]
}

// buildFoldExperiment reconstructs the runnable per-fold experiment for one (config, fold):
// the data steps (as declared — cache-shared, config+fold-invariant) + the engine-synthesized
// cv-split partition step + the pipeline steps with the config's resolved `with` overlaid AND
// the fold-context injected. The fold-context ({_fold:{partition,idx}, folds:<cv-split>}) enters
// each pipeline step's `with` so its Kpre is fold-distinct (the B2 collision guard) and the step
// can read the fold assignment. Ship is NOT included (winner-only, M1a-5).
func (ss *shapeSweep) buildFoldExperiment(c shape.Point, f sampler.FoldPoint) experiment.Experiment {
	sh := ss.sh
	steps := make([]experiment.Step, 0, len(sh.Data)+1+len(sh.Pipeline))
	steps = append(steps, sh.Data...)
	steps = append(steps, partitionStep(sh))
	for _, ps := range sh.Pipeline {
		s := ps
		s.With = foldWith(c.With[ps.ID], ss.partRef, f.Idx)
		s.Needs = appendUnique(ps.Needs, partitionStepID)
		steps = append(steps, s)
	}
	exp := experiment.Experiment{Header: sh.Header, Steps: steps}
	exp.Type = "experiment"
	return exp
}

// partitionStep synthesizes the cv-split step from sweeper.resample.cv (single-source — no
// cv-split in the shape): it splits the base dataset the last data step produced (its `out`
// path, by convention) into k folds, writing folds.json the per-fold pipeline reads.
func partitionStep(sh experiment.Shape) experiment.Step {
	dataOut, dataID := baseDatasetRef(sh)
	with := map[string]any{
		"dataset":  dataOut,
		"k":        sh.Sweeper.Resample.CV.K,
		"stratify": sh.Sweeper.Resample.CV.Stratify,
	}
	var needs []string
	if dataID != "" {
		needs = []string{dataID}
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
// (k, stratify, seed) so every fold-point's address is reproducible. metis#18 M1a-4 will
// thread the cv-split's content-hash here; a deterministic id suffices for the told-set key.
func partitionRef(sh experiment.Shape) sampler.PartitionRef {
	cv := sh.Sweeper.Resample.CV
	return sampler.PartitionRef(fmt.Sprintf("cv-k%d-strat%t-seed%d", cv.K, cv.Stratify, sh.Seed))
}

// pointAddressOf pre-computes a (config, fold) run's content-address (== its run-dir id),
// minted from its FULL resolved config the SAME way buildRecord mints the record's address —
// so the manifest run_id and the record.json point_address can't desync (metis#8's handoff).
func pointAddressOf(exp experiment.Experiment, repoSHAs map[string]string) (string, error) {
	resolved := make(map[string]map[string]any, len(exp.Steps))
	for _, s := range exp.Steps {
		resolved[s.ID] = s.With
	}
	h, err := record.PointAddress(resolved, repoSHAs, exp.Seed)
	return string(h), err
}

// reportWinner prints the honest per-config (mean, SE) leaderboard (best-first by the
// objective) + the selected winner — the metis#18 deliverable. Ship (refit + submission) is
// metis#18 M1a-5; here we report the selection.
func (ss *shapeSweep) reportWinner(w sampler.Winner) {
	fmt.Fprintf(ss.out, "metis: sweep %s done — %d configs scored (manifest %s)\n", ss.sh.ID, len(ss.configs), ss.man.ShapeRunID[:12])
	best := betterFirst(ss.configs, ss.sh.Sweeper.Objective.Direction)
	fmt.Fprintln(ss.out, "  config                          mean      SE")
	for _, cs := range best {
		fmt.Fprintf(ss.out, "  %-30s  %.4f  %.4f\n", freeParamStr(cs.point), cs.meanSE.Mean, cs.meanSE.SE)
	}
	fmt.Fprintf(ss.out, "metis: winner %s — mean %.4f (SE %.4f) over %d folds\n",
		freeParamStrFromParams(w.FreeParams), w.Score.Mean, w.Score.SE, len(w.FoldKeys))
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
func shapeRunIdentity(sh experiment.Shape, repoSHAs map[string]string) (string, error) {
	h, err := record.CanonicalHash(struct {
		Shape    string             `json:"shape"`
		Data     []experiment.Step  `json:"data"`
		Pipeline []experiment.Step  `json:"pipeline"`
		Ship     []experiment.Step  `json:"ship"`
		Sweeper  experiment.Sweeper `json:"sweeper"`
		RepoSHAs map[string]string  `json:"repo_shas"`
		Seed     int                `json:"seed"`
	}{sh.ID, sh.Data, sh.Pipeline, sh.Ship, sh.Sweeper, repoSHAs, sh.Seed})
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

// repoSHAsOf builds the {repoName: sha} map buildRecord uses — same construction, so a
// pre-computed PointAddress matches the record's internal one (incl. the no-git case).
func repoSHAsOf(repoName, sha string) map[string]string {
	m := map[string]string{}
	if repoName != "" {
		m[repoName] = sha
	}
	return m
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
