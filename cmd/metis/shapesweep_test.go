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
	calls *[]string
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

// foldShapeMD is a valid metis#18 phase shape: data(get-data,adapt) │ pipeline(features,
// train — train sweeps model ∈ {a,b}) │ ship(predict), a 2-fold stratified-off inner CV,
// argmax-mean select, driver:single. 2 configs × 2 folds = 4 per-fold runs.
func foldShapeMD(models string) string {
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
ship:
  - id: predict
    uses: test/predict
    needs: [train]
    with: {model: train}
sweeper:
  sampler: grid
  resample: {cv: {k: 2, stratify: false}}
  objective: {metric: train.fold_score, direction: maximize, select: argmax-mean}
driver:
  single: {}
---
`
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
