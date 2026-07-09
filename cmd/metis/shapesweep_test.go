package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/ledger"
)

// fixedNow is the deterministic clock every sweep/run test injects (shared helper — also
// used by capture_e2e_test.go). The run-ids key off content-addresses, not the clock, so a
// frozen time keeps records reproducible.
func fixedNow() func() time.Time {
	return func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) }
}

// foldFakeExec is an injected StepExecutor (the runOpts.exec test seam): it drives the
// metis#18 nested-Sampler loop with NO subprocess. It writes one deterministic artifact per
// step (so the cache can store + materialize it) and emits a fold_score for the `train`
// step, scored deterministically from the config knob + the injected fold-context. `calls`
// records the step-ids the INNER exec actually ran — a cache HIT skips it, so calls is the
// MISS trace the cache assertions read.
type foldFakeExec struct {
	calls        *[]string
	noComplexity bool // metis#19: omit the complexity metric (simulate a model class that doesn't report it)
}

func (f foldFakeExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	if f.calls != nil {
		*f.calls = append(*f.calls, step.ID)
	}
	stepDir := filepath.Join(runDir, step.ID)
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		return experiment.StepResult{}, err
	}
	art := step.ID + "/out.txt"
	if err := os.WriteFile(filepath.Join(runDir, filepath.FromSlash(art)), []byte(step.ID), 0o644); err != nil {
		return experiment.StepResult{}, err
	}
	if step.With["model"] == "fail" {
		return experiment.StepResult{}, fmt.Errorf("fake: forced fail in %s (model=fail)", step.ID)
	}
	metrics := map[string]float64{}
	if step.ID == "train" {
		metrics["fold_score"] = fakeTrainScore(step.With)
		if !f.noComplexity {
			metrics["complexity"] = fakeTrainComplexity(step.With)
		}
	}
	return experiment.StepResult{Metrics: metrics, Artifacts: []string{art}}, nil
}

// fakeTrainScore is a deterministic per-(config,fold) score: a per-model base + a per-fold
// nudge, so distinct models have distinct MEANS (winner selection) and distinct folds give
// a non-zero SE (the (mean,SE) reduction). Reads the engine-injected fold idx from `with`.
func fakeTrainScore(with map[string]any) float64 {
	base := map[string]float64{"a": 0.80, "b": 0.90, "c": 0.70}[fmt.Sprint(with["model"])]
	idx := 0
	if fold, ok := with["_fold"].(map[string]any); ok {
		if i, ok := fold["idx"].(int); ok {
			idx = i
		}
	}
	return base + float64(idx)*0.02
}

// fakeTrainComplexity is a deterministic per-model realized-complexity (metis#19): a
// fixed value per model, fold-independent (like a tree's realized leaves), so a config's
// MeanComplexity is stable across folds. Distinct from fakeTrainScore so the tests can tell
// the two metrics apart.
func fakeTrainComplexity(with map[string]any) float64 {
	return map[string]float64{"a": 10, "b": 20, "c": 30}[fmt.Sprint(with["model"])]
}

// foldShapeMD is a valid metis#18 phase shape: data(get-data,adapt) │ pipeline(features,
// train — train sweeps model ∈ {a,b}), a 2-fold stratified-off inner CV, argmax-mean select,
// driver:single. NO ship phase — the sweep-mechanism tests exercise the pure sweep in
// isolation (shipWinner no-ops on an empty ship). 2 configs × 2 folds = 4 per-fold runs.
func foldShapeMD(models string) string { return foldShape(models, "") }

// foldShapeShipMD adds a ship phase (predict → submission) so the driver:single ship path
// runs — the winner is refit on all rows (no _fold) and predicted → a submission artifact.
func foldShapeShipMD(models string) string {
	return foldShape(models, `ship:
  - id: predict
    uses: test/predict
    needs: [train]
    with: {dataset: features, model: train}
  - id: submission
    uses: test/submission
    needs: [predict]
    with: {predictions: predict}
`)
}

func foldShape(models, ship string) string {
	return `---
type: experiment-shape
id: fold-sweep
seed: 7
status: active
data:
  - id: get-data
    uses: test/download
    with: {slug: x}
  - id: adapt
    uses: test/adapt
    needs: [get-data]
    with: {raw: get-data, out: ../data/base}
pipeline:
  - id: features
    uses: test/features
    needs: [adapt]
    with: {dataset: ../data/base}
  - id: train
    uses: test/train
    needs: [features]
    with:
      model: {$any: ` + models + `}
` + ship + `sweeper:
  sampler: grid
  resample: {cv: {k: 2, stratify: false}}
  objective: {metric: train.fold_score, direction: maximize, select: {argmax-mean: {}}}
driver:
  single: {}
---
`
}

// foldShapePctLossMD is foldShapeMD with a pct-loss (parsimony) select rule — used to test
// the metis#19 complexity guard (a parsimony rule needs a measured complexity).
func foldShapePctLossMD(models string) string {
	return strings.Replace(foldShapeMD(models),
		"select: {argmax-mean: {}}", "select: {pct-loss: {tolerance: 0.02}}", 1)
}

func writeShapeFile(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "shape.md")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// runFoldSweep drives runExperiment with the injected fake exec (no subprocess). calls (if
// non-nil) accumulates the inner-exec MISS trace; out captures the leaderboard.
func runFoldSweep(t *testing.T, expPath string, cache bool, calls *[]string, out io.Writer, git gitProbe) error {
	t.Helper()
	if git == nil {
		git = fakeGitProbe{name: "metis", sha: "sha", dirty: false}
	}
	if out == nil {
		out = io.Discard
	}
	_, err := runExperiment(runOpts{
		expPath: expPath,
		now:     fixedNow(),
		git:     git,
		cache:   cache,
		exec:    foldFakeExec{calls: calls},
		out:     out,
	})
	return err
}

// The nested Sampler loop: the sweeper grids over 2 configs, each scored over 2 folds →
// per-config (mean, SE), argmax-mean winner, N×k raw ledger rows. The load-bearing metis#18
// deliverable.
func TestShapeSweep_NestedLoopWinnerAndLedger(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeMD("[a, b]"))
	var out strings.Builder
	if err := runFoldSweep(t, expPath, false, nil, &out, nil); err != nil {
		t.Fatalf("shape sweep should run: %v", err)
	}

	// 2 configs × 2 folds = 4 per-fold runs, each in its own content-addressed run dir.
	runDirs, _ := filepath.Glob(filepath.Join(ws, "runs", "*"))
	if len(runDirs) != 4 {
		t.Errorf("2 configs × 2 folds should produce 4 distinct per-fold run dirs, got %d", len(runDirs))
	}

	// The manifest records all 4 (config, fold) points with their fold coordinate.
	man := readSweepManifest(t, ws)
	if len(man.Points) != 4 {
		t.Fatalf("manifest should list 4 per-fold points, got %d", len(man.Points))
	}
	folds := map[int]int{}
	for _, p := range man.Points {
		folds[p.Fold]++
	}
	if folds[0] != 2 || folds[1] != 2 {
		t.Errorf("each of the 2 folds should appear for both configs; got fold counts %v", folds)
	}

	// The ledger sidecar holds the RAW per-fold rows; AggregateView reduces to per-config
	// (mean, SE): config b (0.90, 0.92 → 0.91) beats config a (0.80, 0.82 → 0.81).
	led := loadLedgerOrFatal(t, expPath)
	if len(led.Rows) != 4 {
		t.Fatalf("ledger should hold 4 raw per-fold rows, got %d", len(led.Rows))
	}
	for _, r := range led.Rows {
		if r.Fold == nil {
			t.Errorf("a swept ledger row must carry a fold coordinate; got %+v", r)
		}
	}
	agg := ledger.AggregateView(led, "train.fold_score")
	if len(agg.Rows) != 2 {
		t.Fatalf("AggregateView should reduce 4 fold rows → 2 per-config rows, got %d", len(agg.Rows))
	}
	best, ok := ledger.Best(agg, "train.fold_score", "maximize")
	if !ok || fmt.Sprint(best.FreeParams["train.model"]) != "b" {
		t.Errorf("winner config by mean should be model=b; got %+v (ok=%v)", best.FreeParams, ok)
	}
	if m := best.Metrics["train.fold_score"]; m < 0.905 || m > 0.915 {
		t.Errorf("winner mean fold_score should be ~0.91, got %v", m)
	}
	if se := best.Metrics["train.fold_score.se"]; se <= 0 {
		t.Errorf("winner SE should be > 0 (folds differ 0.90 vs 0.92); got %v", se)
	}

	// The leaderboard + winner are reported to the user.
	if s := out.String(); !strings.Contains(s, "winner") || !strings.Contains(s, "train.model=b") {
		t.Errorf("sweep should report the winner (model=b); got:\n%s", s)
	}
}

// metis#19 M2: a per-fold `complexity` metric threads fold→config (FoldOutcome.Complexity
// → Aggregate → MeanSE.MeanComplexity) and surfaces on the winner line. Proves runPipelineFold
// reads the metric and the reducer carries it (M1 wired it as 0).
func TestShapeSweep_ComplexityThreadsFoldToConfig(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeMD("[a, b]"))
	var out strings.Builder
	if err := runFoldSweep(t, expPath, false, nil, &out, nil); err != nil {
		t.Fatalf("shape sweep should run: %v", err)
	}
	// The winner (model=b) line reports a non-zero mean complexity (fake emits cx per model).
	s := out.String()
	if !strings.Contains(s, "train.model=b") {
		t.Fatalf("expected winner model=b; got:\n%s", s)
	}
	// The winner line ends with "cx <N>"; N must be > 0 (b's fake complexity = 20).
	if !strings.Contains(s, "cx 20.0") {
		t.Errorf("winner line should report the threaded complexity cx 20.0 (model=b); got:\n%s", s)
	}
	// The raw ledger rows carry the namespaced per-fold complexity.
	led := loadLedgerOrFatal(t, expPath)
	for _, r := range led.Rows {
		if _, ok := r.Metrics["train.complexity"]; !ok {
			t.Errorf("each per-fold ledger row must carry train.complexity; got %+v", r.Metrics)
		}
	}
}

// metis#19 guard: an in-memory sweep with a parsimony rule (pct-loss) whose model step does
// NOT emit complexity → a hard error (raw rows still persisted; only ship/report is gated).
func TestShapeSweep_ParsimonyGuardOnMissingComplexity(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapePctLossMD("[a, b]"))
	_, err := runExperiment(runOpts{
		expPath: expPath, now: fixedNow(),
		git:  fakeGitProbe{name: "metis", sha: "sha", dirty: false},
		exec: foldFakeExec{noComplexity: true}, out: io.Discard,
	})
	if err == nil {
		t.Fatalf("pct-loss with no emitted complexity must error")
	}
	if !strings.Contains(err.Error(), "complexity") {
		t.Errorf("guard error should mention complexity; got %v", err)
	}
	// The raw ledger rows are still persisted (re-selectable after a fix).
	led := loadLedgerOrFatal(t, expPath)
	if len(led.Rows) != 4 {
		t.Errorf("raw fold rows should persist despite the guard; got %d", len(led.Rows))
	}
}

// The same pct-loss shape WITH complexity emitted selects cleanly (the guard passes).
func TestShapeSweep_ParsimonyRuleWithComplexity(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapePctLossMD("[a, b]"))
	var out strings.Builder
	if err := runFoldSweep(t, expPath, false, nil, &out, nil); err != nil {
		t.Fatalf("pct-loss with complexity should run: %v", err)
	}
	// model=b (0.91) is within 2% of itself; parsimony: a (cx 10) is simpler than b (cx 20)
	// but a's mean 0.81 is outside b's 2% band (0.91·0.98=0.8918) → b wins. Just assert it ran.
	if s := out.String(); !strings.Contains(s, "winner") {
		t.Errorf("expected a winner line; got:\n%s", s)
	}
}

// driver:single ships the winner (metis#18 M1a-5): after the sweeper selects the champion,
// the ship phase refits it on ALL rows (no _fold) and runs predict → submission. The ship is
// a SEPARATE content-addressed run, NOT a manifest fold-point — folds run data+cv-split+
// pipeline only, so the ship run is the unique one carrying the ship steps.
func TestShapeSweep_ShipsWinner(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeShipMD("[a, b]"))
	var out strings.Builder
	if err := runFoldSweep(t, expPath, false, nil, &out, nil); err != nil {
		t.Fatalf("sweep+ship should run: %v", err)
	}

	// Exactly one run dir holds a `submission` step — the driver:single ship of the winner.
	shipSteps, _ := filepath.Glob(filepath.Join(ws, "runs", "*", "submission"))
	if len(shipSteps) != 1 {
		t.Fatalf("driver:single must ship exactly one winner (one submission run), got %d", len(shipSteps))
	}
	shipRun := filepath.Dir(shipSteps[0])
	// The ship run is the full winning pipeline refit on all rows + the ship steps — and NO
	// cv-split (the ship needs no CV; shapeConfigToExperiment omits it).
	for _, step := range []string{"get-data", "adapt", "features", "train", "predict", "submission"} {
		if _, err := os.Stat(filepath.Join(shipRun, step)); err != nil {
			t.Errorf("ship run must include the winning pipeline + ship step %q: %v", step, err)
		}
	}
	if _, err := os.Stat(filepath.Join(shipRun, partitionStepID)); err == nil {
		t.Errorf("the ship refit must NOT run cv-split — it fits on all rows, no CV")
	}
	// The ship is NOT a manifest fold-point (the manifest records only the per-fold sweep).
	if man := readSweepManifest(t, ws); len(man.Points) != 4 {
		t.Errorf("manifest should hold only the 4 per-fold points, not the ship; got %d", len(man.Points))
	}
	if !strings.Contains(out.String(), "shipped") {
		t.Errorf("the sweep should report shipping the winner; got:\n%s", out.String())
	}
}

// Fold-distinctness + cache: each (config, fold) of `train` gets a DISTINCT cache entry (the
// B2 collision guard — the _fold overlay makes Kpre fold-distinct), the config+fold-invariant
// data/cv-split steps HIT across points, and a warm re-run HITs everything (0 inner execs).
func TestShapeSweep_CacheFoldDistinctAndReRunHits(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeMD("[a, b]"))

	var cold []string
	if err := runFoldSweep(t, expPath, true, &cold, nil, nil); err != nil {
		t.Fatalf("cold sweep: %v", err)
	}
	// train runs once per (config, fold) — 4 distinct entries (fold overlay ⇒ no collision).
	if n := countCalls(cold, "train"); n != 4 {
		t.Errorf("train should run 4× (2 configs × 2 folds, each cache-distinct), got %d", n)
	}
	// get-data is config+fold-invariant → runs ONCE, then HITs on the on-disk index across
	// the remaining 3 point-runs.
	if n := countCalls(cold, "get-data"); n != 1 {
		t.Errorf("get-data (config+fold-invariant) should run once and HIT after, got %d", n)
	}
	// features is config-invariant but fold-distinct → runs once per fold = 2×.
	if n := countCalls(cold, "features"); n != 2 {
		t.Errorf("features (fold-distinct, config-invariant) should run once per fold = 2×, got %d", n)
	}

	// A warm re-run: every step HITs — no inner exec at all.
	var warm []string
	if err := runFoldSweep(t, expPath, true, &warm, nil, nil); err != nil {
		t.Fatalf("warm re-run: %v", err)
	}
	if len(warm) != 0 {
		t.Errorf("a warm re-run should HIT everything (0 inner execs), got %d: %v", len(warm), warm)
	}
}

// A one-hyperparameter change recomputes ONLY the affected config's folds — the incremental
// property. Warm the cache on {a, b}, then change b→c: config a's folds HIT unchanged; only
// config c's 2 folds recompute (features stays fold-shared, so it HITs too).
func TestShapeSweep_HyperparamChangeRecomputesAffected(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeMD("[a, b]"))
	if err := runFoldSweep(t, expPath, true, nil, nil, nil); err != nil {
		t.Fatalf("warm sweep: %v", err)
	}

	// Change one config knob (b → c) and re-run against the SAME shared cache.
	writeShapeFile(t, ws, foldShapeMD("[a, c]"))
	var calls []string
	if err := runFoldSweep(t, expPath, true, &calls, nil, nil); err != nil {
		t.Fatalf("re-run after hyperparam change: %v", err)
	}
	// Only config c's 2 train folds are new; config a's train + all features + data HIT.
	if n := countCalls(calls, "train"); n != 2 {
		t.Errorf("only the changed config (c) should recompute its 2 folds, got %d train runs: %v", n, calls)
	}
	if n := countCalls(calls, "features"); n != 0 {
		t.Errorf("features is config-invariant → must stay cached across the knob change, got %d", n)
	}
	if n := countCalls(calls, "get-data"); n != 0 {
		t.Errorf("get-data must stay cached across the knob change, got %d", n)
	}
}

// --dry-run lists the swept configs and runs nothing.
func TestShapeSweep_DryRunListsWithoutRunning(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeMD("[a, b]"))
	var out strings.Builder
	_, err := runExperiment(runOpts{
		expPath: expPath, now: fixedNow(), git: fakeGitProbe{name: "metis", sha: "sha"},
		dryRun: true, exec: foldFakeExec{}, out: &out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "dry run") || !strings.Contains(out.String(), "train.model=a") {
		t.Errorf("dry run should list the configs; got:\n%s", out.String())
	}
	if entries, _ := filepath.Glob(filepath.Join(ws, "runs", "*")); len(entries) != 0 {
		t.Errorf("dry run must not create run dirs, found %d", len(entries))
	}
}

// A failing fold is FATAL to the sweep — a partial resample is not an honest (mean, SE)
// estimate, so the sweep surfaces the error rather than recording a half-scored config.
func TestShapeSweep_FailingFoldIsFatal(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeMD("[a, fail]")) // model=fail makes train exit non-zero
	err := runFoldSweep(t, expPath, false, nil, io.Discard, nil)
	if err == nil {
		t.Fatal("a failing fold must abort the sweep (a partial resample isn't an honest estimate)")
	}
	if !strings.Contains(err.Error(), "fold") {
		t.Errorf("the error should name the failing fold; got: %v", err)
	}
}

// Detect-and-abort: a mid-sweep HEAD-sha change breaks the shape-run's one-code identity →
// the sweep aborts with a 'code changed' error.
func TestShapeSweep_DetectAndAbortOnCodeChange(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeMD("[a, b]"))
	// The sha flips after the first per-fold re-check (Probe is called once at freeze, then
	// once per fold) → simulates a mid-sweep code edit.
	probe := &mutatingGitProbe{shas: []string{"sha1", "sha1", "sha2"}}
	err := runFoldSweep(t, expPath, false, nil, io.Discard, probe)
	if err == nil || !strings.Contains(err.Error(), "code changed") {
		t.Errorf("a mid-sweep code change should abort with a 'code changed' error; got: %v", err)
	}
}

// ── test helpers ──────────────────────────────────────────────────────────────

// mutatingGitProbe returns a different sha per call, simulating code drift mid-sweep.
type mutatingGitProbe struct {
	shas []string
	n    int
}

func (m *mutatingGitProbe) Probe(string) (string, string, bool, error) {
	s := m.shas[len(m.shas)-1]
	if m.n < len(m.shas) {
		s = m.shas[m.n]
	}
	m.n++
	return "metis", s, false, nil
}

func countCalls(calls []string, id string) int {
	n := 0
	for _, c := range calls {
		if c == id {
			n++
		}
	}
	return n
}

func readSweepManifest(t *testing.T, ws string) sweepManifest {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(ws, "sweeps", "*", "manifest.json"))
	if len(matches) != 1 {
		t.Fatalf("expected exactly one sweep manifest, found %v", matches)
	}
	b, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	var man sweepManifest
	if err := json.Unmarshal(b, &man); err != nil {
		t.Fatal(err)
	}
	return man
}

func loadLedgerOrFatal(t *testing.T, expPath string) ledger.Ledger {
	t.Helper()
	led, err := loadLedger(expPath)
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	return led
}
