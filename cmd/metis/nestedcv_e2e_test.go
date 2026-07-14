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
