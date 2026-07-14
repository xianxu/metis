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

// metis#32: a multi-config shape runs NESTED (config-count dispatch). `metis run` RECORDS the whole
// nested CV to the ledger — inner rows (Level=inner) per (outer-fold, config, inner-fold) tagged
// with their outer fold, + one outer row (Level=outer) per (outer-fold, family) — and reports the
// honest mean±SE estimate. It does NOT ship (that's `metis select --promote`). 2 configs (one
// family) × 2 inner × 2 outer folds → 8 inner + 2 outer rows.
func TestShapeSweep_NestedLoopWinnerAndLedger(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeMD("[a, b]"))
	var out strings.Builder
	if err := runFoldSweep(t, expPath, false, nil, &out, nil); err != nil {
		t.Fatalf("nested sweep should run: %v", err)
	}

	led := loadLedgerOrFatal(t, expPath)
	var nInner, nOuter int
	for _, r := range led.Rows {
		// metis#27: the code_fingerprint must reach the PERSISTED row (capture-before-ledger
		// ordering) — a reorder would silently yield empty-fingerprint rows.
		if r.CodeFingerprint == "" {
			t.Errorf("a swept ledger row must carry a non-empty code_fingerprint; got %+v", r)
		}
		switch r.Level {
		case "inner":
			nInner++
			// metis#32: an inner row is tagged with its outer fold (so select can pool per config).
			if r.OuterFold == nil {
				t.Errorf("an inner row must carry its outer-fold coordinate; got %+v", r)
			}
		case "outer":
			nOuter++
		default:
			t.Errorf("a nested-run ledger row must be Level inner|outer; got %q in %+v", r.Level, r)
		}
	}
	if nInner != 8 {
		t.Errorf("2 configs × 2 inner × 2 outer folds → 8 inner rows, got %d", nInner)
	}
	if nOuter != 2 {
		t.Errorf("1 family × 2 outer folds → 2 outer rows, got %d", nOuter)
	}

	// The honest procedure estimate (mean±SE over outer folds) is reported — NOT a single winner
	// line (selection + ship moved to `metis select`).
	if s := out.String(); !strings.Contains(s, "nested-CV estimate — mean") {
		t.Errorf("nested run should report the honest mean±SE estimate; got:\n%s", s)
	}
}

// metis#19 M2 / metis#32: the per-fold `complexity` metric threads fold→config
// (FoldOutcome.Complexity → the recorded inner rows). Proves runPipelineFold reads the metric and
// the nested recording carries it. (Nested: no single winner line — the threading essence is that
// the metric reaches the recorded inner rows.)
func TestShapeSweep_ComplexityThreadsFoldToConfig(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeMD("[a, b]"))
	if err := runFoldSweep(t, expPath, false, nil, nil, nil); err != nil {
		t.Fatalf("nested sweep should run: %v", err)
	}
	// Every recorded INNER row carries the namespaced per-fold complexity (fake emits cx per model).
	led := loadLedgerOrFatal(t, expPath)
	var innerRows int
	for _, r := range led.Rows {
		if r.Level != "inner" {
			continue
		}
		innerRows++
		if _, ok := r.Metrics["train.complexity"]; !ok {
			t.Errorf("each inner ledger row must carry train.complexity; got %+v", r.Metrics)
		}
	}
	if innerRows == 0 {
		t.Fatal("expected recorded inner rows carrying complexity")
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
	// metis#32: the load-bearing invariant is that the guard ERRORS (rather than silently
	// mis-selecting per outer fold). NB the nested path's ledger write is end-of-run, AFTER the
	// outer loop, so a guard error aborts before any rows persist here — unlike the old flat path,
	// raw rows are NOT re-selectable after a nested guard error (a re-run recomputes cheaply from cache).
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

// metis#32: `metis run` MEASURES ONLY — it never auto-ships, even when a ship phase is present.
// Shipping moved to `metis select --promote` (which reconstructs the chosen config, refits on ALL
// rows, and runs predict → submission). A multi-config shape runs nested; NO submission artifact is
// produced by the run. (The ship-assembly correctness — all-data refit, no cv-split — is verified
// on the `metis select --promote` path in M2.)
func TestShapeSweep_DoesNotShip(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeShipMD("[a, b]"))
	var out strings.Builder
	if err := runFoldSweep(t, expPath, false, nil, &out, nil); err != nil {
		t.Fatalf("nested run should succeed: %v", err)
	}

	// NO run dir holds a `submission` step — `metis run` measures, it does not ship.
	shipSteps, _ := filepath.Glob(filepath.Join(ws, "runs", "*", "submission"))
	if len(shipSteps) != 0 {
		t.Errorf("`metis run` must NOT ship (shipping is `metis select --promote`); got %d submission runs", len(shipSteps))
	}
	if strings.Contains(out.String(), "shipped") {
		t.Errorf("`metis run` must not report shipping a winner; got:\n%s", out.String())
	}
}

// Fold-distinctness + cache under NESTED (metis#32): each (outer-fold, config, inner-fold) of
// `train` gets a DISTINCT cache entry (the _fold overlay makes Kpre fold-distinct), the
// config/fold-invariant data steps HIT, and a warm re-run HITs everything (0 inner execs).
func TestShapeSweep_CacheFoldDistinctAndReRunHits(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeMD("[a, b]"))

	var cold []string
	if err := runFoldSweep(t, expPath, true, &cold, nil, nil); err != nil {
		t.Fatalf("cold sweep: %v", err)
	}
	// train runs per distinct (outer-fold, config, inner-fold): the sealed inner sweeps
	// (2 outer × 2 configs × 2 inner = 8) + the per-family outer-fold held-out scorings (2) = 10.
	if n := countCalls(cold, "train"); n != 10 {
		t.Errorf("train should run 10× (8 sealed inner + 2 outer-scoring, each cache-distinct), got %d", n)
	}
	// get-data is fully invariant → the outer-split preamble runs it ONCE, then everything HITs.
	if n := countCalls(cold, "get-data"); n != 1 {
		t.Errorf("get-data (fully invariant) should run once and HIT after, got %d", n)
	}
	// features is config-invariant but fold-distinct → per (outer, inner-fold) once (4) + the 2
	// outer-fold scorings = 6.
	if n := countCalls(cold, "features"); n != 6 {
		t.Errorf("features (fold-distinct, config-invariant) should run 6× (4 inner + 2 outer-scoring), got %d", n)
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

// A one-hyperparameter change recomputes ONLY the affected config — the incremental property,
// under NESTED (metis#32). Warm on {a, b}, then change b→c: config a HITs unchanged; only config
// c's runs recompute (features/data stay shared, so they HIT).
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
	// Only config c's runs are new: its sealed inner runs (2 outer × 2 inner = 4) + the outer-fold
	// scorings it wins (2) = 6; config a's train + all features + data HIT. (10 would mean BOTH
	// configs recomputed — the count pins "only the changed config recomputed".)
	if n := countCalls(calls, "train"); n != 6 {
		t.Errorf("only the changed config (c) should recompute (6 nested runs); got %d train runs: %v", n, calls)
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
