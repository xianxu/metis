package main

import (
	"fmt"
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

// foldShapeCVMD is foldShapeShipMD with driver:cv (nested-CV, outerK outer folds) replacing
// driver:single — a ship phase is present specifically to prove driver:cv ships NOTHING.
func foldShapeCVMD(models string, outerK int) string {
	return strings.Replace(foldShapeShipMD(models),
		"driver:\n  single: {}",
		fmt.Sprintf("driver:\n  cv: {k: %d}", outerK), 1)
}

// TestNestedCV_ProducesHonestEstimateNoShip drives driver:cv over the fake exec and asserts the
// PLUMBING (metis#23): a preamble + k outer SEALED sweeps + k refit-and-score runs execute, the
// driver reports a mean±SE PROCEDURE estimate, and NO winner is shipped even though a ship phase
// exists. The real "estimate < inner cv-max" HONESTY GAP is a real-data property (operator-gated
// Titanic) — a config-deterministic fake can't exhibit it, so this test asserts the mechanism,
// not the gap magnitude.
func TestNestedCV_ProducesHonestEstimateNoShip(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b]", 3))

	var out strings.Builder
	if err := runFoldSweep(t, expPath, false, nil, &out, nil); err != nil {
		t.Fatalf("driver:cv run should succeed: %v", err)
	}
	s := out.String()

	// The honest estimate is reported as a mean±SE over the outer folds (a distinct code path
	// from the flat sweep's inner cv-max winner line).
	if !strings.Contains(s, "nested-CV estimate — mean") {
		t.Errorf("driver:cv should report the honest mean±SE procedure estimate; got:\n%s", s)
	}
	// One held-out score per outer fold (k=3).
	if n := strings.Count(s, "held-out score"); n != 3 {
		t.Errorf("expected 3 outer-fold held-out scores, got %d:\n%s", n, s)
	}
	// NO ship — the inverse of TestShapeSweep_ShipsWinner: driver:cv estimates, never selects a
	// shippable winner, so no submission artifact is produced despite the ship phase.
	shipSteps, _ := filepath.Glob(filepath.Join(ws, "runs", "*", "submission", "out.txt"))
	if len(shipSteps) != 0 {
		t.Errorf("driver:cv must NOT ship a winner, got %d submission artifacts", len(shipSteps))
	}
	if strings.Contains(s, "shipped winner") {
		t.Errorf("driver:cv must not report shipping a winner; got:\n%s", s)
	}
	if !strings.Contains(s, "ships NO winner") {
		t.Errorf("driver:cv should note it ships no winner; got:\n%s", s)
	}
}

// TestNestedCV_DryRunSurfacesOuterCost asserts the ~outerK× cost is surfaced before a run and the
// "no shippable winner" caveat is printed (so the ~5× is opted into knowingly).
func TestNestedCV_DryRunSurfacesOuterCost(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b]", 4))

	var out strings.Builder
	if _, err := runExperiment(runOpts{expPath: expPath, now: fixedNow(),
		git: fakeGitProbe{name: "metis", sha: "sha"}, dryRun: true, out: &out}); err != nil {
		t.Fatalf("dry-run should succeed: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "nested-CV") || !strings.Contains(s, "4 outer folds") {
		t.Errorf("dry-run should surface the outer-fold multiplier; got:\n%s", s)
	}
	if !strings.Contains(s, "NO shippable winner") {
		t.Errorf("dry-run should warn nested-CV ships no winner; got:\n%s", s)
	}
}
