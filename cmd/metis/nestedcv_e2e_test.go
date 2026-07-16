package main

import (
	"io"
	"path/filepath"
	"strings"
	"testing"
)

// TestNestedCV_ParsimonyGuardOnMissingComplexity: driver:cv must run the SAME GuardComplexity the
// flat path does (metis#23 I1). A parsimony select rule + a model that emits no complexity would
// otherwise SILENTLY mis-select in EACH outer fold (all Complexity=0 → tie-break to mean) and report
// an "honest" procedure estimate over quietly-wrong winners — the exact silent-wrongness the guard
// exists to prevent. The flat path loudly rejects this shape; driver:cv must too.
func TestNestedCV_ParsimonyGuardOnMissingComplexity(t *testing.T) {
	ws := t.TempDir()
	cvPctLoss := strings.Replace(foldShapePctLossMD("[a, b]"),
		"driver:\n  single: {}", "driver:\n  cv: {k: 2}", 1)
	expPath := writeShapeFile(t, ws, cvPctLoss)
	_, err := runExperiment(runOpts{
		expPath: expPath, now: fixedNow(),
		git:  fakeGitProbe{name: "metis", sha: "sha", dirty: false},
		exec: foldFakeExec{noComplexity: true}, out: io.Discard,
	})
	if err == nil {
		t.Fatal("driver:cv + pct-loss with no emitted complexity must error, not silently mis-select per outer fold")
	}
	if !strings.Contains(err.Error(), "complexity") {
		t.Errorf("the guard error should mention complexity; got %v", err)
	}
}

// foldShapeCVMD is foldShapeShipMD (a shape WITH a ship phase). metis#32: the run mode is derived
// by config-count (>1 config → nested), so no `driver:` field is injected — the ship phase is
// present specifically to prove `metis run` records the honest estimate and ships NOTHING.
func foldShapeCVMD(models string) string {
	return foldShapeShipMD(models)
}

// TestNestedCV_ProducesHonestEstimateNoShip drives a multi-config (→ nested, metis#32) sweep over
// the fake exec and asserts the metis#23/#32 contract: k outer SEALED sweeps + per-family held-out
// scoring execute, the run reports a mean±SE PROCEDURE estimate, RECORDS inner+outer rows to the
// ledger, and ships NOTHING even though a ship phase exists (shipping moved to `metis select
// --promote`). The real "estimate < inner cv-max" honesty gap is operator-gated (real Titanic) — a
// config-deterministic fake can't exhibit it, so this asserts the mechanism, not the gap magnitude.
func TestNestedCV_ProducesHonestEstimateNoShip(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b]"))

	var out strings.Builder
	if err := runFoldSweep(t, expPath, false, nil, &out, nil); err != nil {
		t.Fatalf("nested run should succeed: %v", err)
	}
	s := out.String()

	// The honest estimate is reported as a mean±SE over the outer folds.
	if !strings.Contains(s, "nested-CV estimate — mean") {
		t.Errorf("a nested run should report the honest mean±SE procedure estimate; got:\n%s", s)
	}
	// metis#30: live progress lines. The fixture clock is FROZEN (fixedNow), so the 1s
	// throttle never elapses — only the always-emit lines appear (outer completions +
	// finish); the throttle itself is pinned by the scripted-clock unit test. The FINAL
	// line carries the complete outer count and a numeric est.
	if !strings.Contains(s, "metis: progress") {
		t.Errorf("a nested run must print live progress lines; got:\n%s", s)
	}
	finalProg := s[strings.LastIndex(s, "metis: progress"):]
	finalProg = finalProg[:strings.IndexByte(finalProg, '\n')]
	if !strings.Contains(finalProg, "outer 2/2") || !strings.Contains(finalProg, "est 0.") {
		t.Errorf("the final progress line must carry the completed outer count + a numeric est; got: %q", finalProg)
	}
	// One held-out score per (outer fold × family): outerK = sweeper.cv.k = 2, and a,b are one
	// family → 2 held-out lines.
	if n := strings.Count(s, "→ held-out "); n != 2 {
		t.Errorf("expected 2 outer-fold held-out scores (2 outer folds × 1 family), got %d:\n%s", n, s)
	}
	// metis#32: the nested run RECORDS both inner and outer rows (it no longer records nothing):
	// inner rows (Level=inner) per (outer-fold, config, inner-fold); one outer row (Level=outer)
	// per (outer-fold, family).
	led := loadLedgerOrFatal(t, expPath)
	var nInner, nOuter int
	for _, r := range led.Rows {
		switch r.Level {
		case "inner":
			nInner++
		case "outer":
			nOuter++
		}
	}
	if nInner == 0 {
		t.Errorf("nested run must record inner rows (Level=inner); got none in %d rows", len(led.Rows))
	}
	if nOuter != 2 {
		t.Errorf("nested run must record one outer row per (outer-fold, family) = 2; got %d", nOuter)
	}
	// NO ship — shipping moved to `metis select --promote`; no submission artifact despite the ship phase.
	shipSteps, _ := filepath.Glob(filepath.Join(ws, "runs", "*", "submission", "out.txt"))
	if len(shipSteps) != 0 {
		t.Errorf("`metis run` must NOT ship, got %d submission artifacts", len(shipSteps))
	}
	if strings.Contains(s, "shipped winner") {
		t.Errorf("`metis run` must not report shipping a winner; got:\n%s", s)
	}
	if !strings.Contains(s, "metis select --best --promote") {
		t.Errorf("the caveat should point to `metis select` for shipping; got:\n%s", s)
	}
}

// TestNestedCV_DryRunSurfacesOuterCost asserts the ~outerK× cost is surfaced before a run and the
// caveat (records inner/outer rows; ship via `metis select --promote`) is printed.
func TestNestedCV_DryRunSurfacesOuterCost(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b]"))

	var out strings.Builder
	if _, err := runExperiment(runOpts{expPath: expPath, now: fixedNow(),
		git: fakeGitProbe{name: "metis", sha: "sha"}, dryRun: true, out: &out}); err != nil {
		t.Fatalf("dry-run should succeed: %v", err)
	}
	s := out.String()
	// 2 configs → nested; outer folds = sweeper.resample.cv.k = 2.
	if !strings.Contains(s, "nested-CV") || !strings.Contains(s, "2 outer fold(s)") {
		t.Errorf("dry-run should surface the outer-fold multiplier; got:\n%s", s)
	}
	if !strings.Contains(s, "metis select --promote") {
		t.Errorf("dry-run should note shipping moved to `metis select --promote`; got:\n%s", s)
	}
}

// TestNestedCV_SampleRunsMOfKFolds (metis#42): `--sample m` runs exactly m of the k outer folds of
// the ALWAYS-k-way partition — the m-of-k generalization of `--fast` (k stays the estimand: each
// fold trains on (k-1)/k of the rows; m only sets how many unbiased samples of that estimand run).
// Asserts: m held-out scores, m outer ledger rows (folds 0..m-1 of the k-partition), and the
// estimate reported over m fold(s).
func TestNestedCV_SampleRunsMOfKFolds(t *testing.T) {
	ws := t.TempDir()
	k3 := strings.Replace(foldShapeMD("[a, b]"), "k: 2", "k: 3", 1)
	expPath := writeShapeFile(t, ws, k3)

	var out strings.Builder
	_, err := runExperiment(runOpts{
		expPath: expPath, now: fixedNow(),
		git:  fakeGitProbe{name: "metis", sha: "sha", dirty: false},
		exec: foldFakeExec{}, out: &out,
		sample: 2,
	})
	if err != nil {
		t.Fatalf("--sample 2 of k=3 should succeed: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "2 outer fold(s)") {
		t.Errorf("banner should show the SAMPLED outer-fold count (2), got:\n%s", s)
	}
	// 2 sampled outer folds × 1 family (a,b share the scalar `model` knob) = 2 held-out lines.
	if n := strings.Count(s, "→ held-out "); n != 2 {
		t.Errorf("expected 2 outer-fold held-out scores (--sample 2 × 1 family), got %d:\n%s", n, s)
	}
	if !strings.Contains(s, "over 2 outer fold(s)") {
		t.Errorf("estimate should aggregate over the 2 SAMPLED folds, got:\n%s", s)
	}
	led := loadLedgerOrFatal(t, expPath)
	outerFolds := map[int]bool{}
	for _, r := range led.Rows {
		if r.Level == "outer" && r.OuterFold != nil {
			outerFolds[*r.OuterFold] = true
		}
	}
	// Folds 0..m-1 of the k-way partition (the seeded partition makes a prefix a valid
	// random m-subset); fold 2 must NOT have run.
	if len(outerFolds) != 2 || !outerFolds[0] || !outerFolds[1] || outerFolds[2] {
		t.Errorf("outer rows should cover exactly sampled folds {0,1} of k=3, got %v", outerFolds)
	}
}

// TestNestedCV_SampleGuards (metis#42): the misuse edges fail LOUDLY — m>k (the partition has only
// k folds), --sample on a single-config shape (the flat path has no outer folds to sample), and
// --sample combined with --fast (--fast is shorthand for --sample 1; two knobs for one thing is
// an ambiguity, not a convenience).
func TestNestedCV_SampleGuards(t *testing.T) {
	newShape := func(t *testing.T, models string) string {
		return writeShapeFile(t, t.TempDir(), foldShapeMD(models))
	}
	base := func(expPath string) runOpts {
		return runOpts{expPath: expPath, now: fixedNow(),
			git:  fakeGitProbe{name: "metis", sha: "sha", dirty: false},
			exec: foldFakeExec{}, out: io.Discard}
	}

	t.Run("m exceeds k", func(t *testing.T) {
		o := base(newShape(t, "[a, b]")) // k=2
		o.sample = 3
		if _, err := runExperiment(o); err == nil || !strings.Contains(err.Error(), "sample") {
			t.Errorf("--sample 3 with k=2 must error mentioning sample, got %v", err)
		}
	})
	t.Run("negative m", func(t *testing.T) {
		o := base(newShape(t, "[a, b]"))
		o.sample = -1
		if _, err := runExperiment(o); err == nil || !strings.Contains(err.Error(), "sample") {
			t.Errorf("--sample -1 must error mentioning sample, got %v", err)
		}
	})
	t.Run("flat single-config shape", func(t *testing.T) {
		o := base(newShape(t, "[a]")) // 1 config → flat CV, no outer folds
		o.sample = 1
		if _, err := runExperiment(o); err == nil || !strings.Contains(err.Error(), "sample") {
			t.Errorf("--sample on a flat (single-config) run must error mentioning sample, got %v", err)
		}
	})
	t.Run("sample plus fast", func(t *testing.T) {
		o := base(newShape(t, "[a, b]"))
		o.sample, o.fast = 2, true
		if _, err := runExperiment(o); err == nil || !strings.Contains(err.Error(), "sample") {
			t.Errorf("--sample + --fast must error (ambiguous), got %v", err)
		}
	})
}
