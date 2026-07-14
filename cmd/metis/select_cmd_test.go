package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/metis/pkg/ledger"
)

// A tagged-sum shape (model: $any-MAP → families logreg/rf) so `metis select` exercises real
// cross-family selection. Each branch sweeps one hyperparam.
const taggedShapeForSelect = `---
type: experiment-shape
id: s
seed: 1
status: active
data:
  - id: adapt
    uses: titanic/adapt
    with: {out: ../data/x}
pipeline:
  - id: train
    uses: metis/train
    needs: [adapt]
    with:
      dataset: adapt
      model:
        $any:
          logreg: {C: {$any: [0.1, 1.0]}}
          rf: {max_depth: {$any: [4, 8]}}
sweeper:
  sampler: grid
  resample: {cv: {k: 2, stratify: false}}
  objective: {metric: train.fold_score, direction: maximize, select: {pct-loss: {tolerance: 0.02}}}
---
`

var (
	lr01 = map[string]any{"train.model": "logreg", "train.model.logreg.C": 0.1}
	lr1  = map[string]any{"train.model": "logreg", "train.model.logreg.C": 1.0}
	rf4  = map[string]any{"train.model": "rf", "train.model.rf.max_depth": 4}
	rf8  = map[string]any{"train.model": "rf", "train.model.rf.max_depth": 8}
)

// writeSelectLedger writes the tagged shape + a nested-CV ledger encoding the metis#32 story:
// on the INNER CV the rf deep tree (md=8) is the flashy champion (0.86, cx 40) — the cross-family
// inner-argmax would ship it. But on the honest OUTER estimate rf overfit and DROPS to a wide 0.78,
// while logreg holds a tight 0.81. So the honest selector must ship LOGREG (the generalizer), not rf.
func writeSelectLedger(t *testing.T, dir string, withOuter bool) string {
	t.Helper()
	shapePath := filepath.Join(dir, "s.md")
	if err := os.WriteFile(shapePath, []byte(taggedShapeForSelect), 0o644); err != nil {
		t.Fatal(err)
	}
	inner := func(addr string, fp map[string]any, ofold, ifold int, score, cx float64) ledger.Row {
		of, ff := ofold, ifold
		return ledger.Row{CodeFingerprint: "cf", PointAddr: addr, FreeParams: fp, Level: "inner", OuterFold: &of, Fold: &ff,
			Metrics: map[string]float64{"train.fold_score": score, "train.complexity": cx}, Status: "ok"}
	}
	outer := func(addr string, fp map[string]any, ofold int, score float64) ledger.Row {
		of := ofold
		return ledger.Row{CodeFingerprint: "cf", PointAddr: addr, FreeParams: fp, Level: "outer", OuterFold: &of,
			Metrics: map[string]float64{"train.fold_score": score}, Status: "ok"}
	}
	var led ledger.Ledger
	// INNER rows (config side): rf md=8 is the inner champion; logreg C=1 is logreg's best.
	led.Append(
		inner("i-lr01-0", lr01, 0, 0, 0.78, 6), inner("i-lr01-1", lr01, 0, 1, 0.78, 6),
		inner("i-lr1-0", lr1, 0, 0, 0.80, 6), inner("i-lr1-1", lr1, 0, 1, 0.80, 6),
		inner("i-rf4-0", rf4, 0, 0, 0.83, 16), inner("i-rf4-1", rf4, 0, 1, 0.83, 16),
		inner("i-rf8-0", rf8, 0, 0, 0.86, 40), inner("i-rf8-1", rf8, 0, 1, 0.86, 40),
	)
	if withOuter {
		// OUTER rows (family side): logreg holds tight (mean 0.81, SE ~0.01); rf overfit → wide
		// (mean 0.78, SE ~0.04). Honest family pick = logreg (higher mean AND lower SE).
		led.Append(
			outer("o-lr-0", lr1, 0, 0.80), outer("o-lr-1", lr1, 1, 0.82),
			outer("o-rf-0", rf8, 0, 0.74), outer("o-rf-1", rf8, 1, 0.82),
		)
	}
	b, err := ledger.Encode(led)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledgerPath(shapePath), b, 0o644); err != nil {
		t.Fatal(err)
	}
	return shapePath
}

// THE ACCEPTANCE GATE (metis#32): `metis select --best` chooses the family on the honest OUTER
// estimate, so it ships LOGREG (the generalizer) — NOT the rf deep tree the inner-CV argmax favors.
// This is the whole point: the honest estimate ACTUATES selection instead of just reporting.
func TestSelect_PicksGeneralizerNotInnerOverfitter(t *testing.T) {
	dir := t.TempDir()
	shapePath := writeSelectLedger(t, dir, true)
	var out strings.Builder
	if err := runSelect(selectOpts{shapePath: shapePath, best: true, out: &out}); err != nil {
		t.Fatalf("select: %v", err)
	}
	s := out.String()
	// The ship recommendation must be logreg (family key path-qualified via sampler.FamilyOf).
	if !strings.Contains(s, "train.model=logreg") {
		t.Errorf("select --best must ship the honest generalizer (logreg family); got:\n%s", s)
	}
	// It must NOT ship rf (the inner-CV cross-family champion) — the #32 flip.
	shipIdx := strings.Index(s, "ship recommendation")
	if shipIdx >= 0 && strings.Contains(s[shipIdx:], "train.model=rf") {
		t.Errorf("select --best must NOT ship the rf inner-overfitter; got:\n%s", s)
	}
	// Both families' honest estimates are reported (transparency).
	if !strings.Contains(s, "per-family honest outer estimate") {
		t.Errorf("select should report the per-family honest estimates; got:\n%s", s)
	}
}

// A multi-family ledger with NO outer rows (a flat/`--fast`-less inner-only ledger) is a SHARP
// error — `metis select` chooses on the honest outer estimate, which isn't there. Never a silent
// inner-CV cross-family argmax (that's the overfitting #32 exists to stop).
func TestSelect_MultiFamilyNoOuterRowsErrors(t *testing.T) {
	dir := t.TempDir()
	shapePath := writeSelectLedger(t, dir, false) // inner rows only
	err := runSelect(selectOpts{shapePath: shapePath, best: true, out: &strings.Builder{}})
	if err == nil {
		t.Fatal("select over a multi-family inner-only ledger must error (no honest outer estimate)")
	}
	if !strings.Contains(err.Error(), "outer") {
		t.Errorf("the error should point at the missing outer-CV rows; got %v", err)
	}
}

// --best-per-model-class reports one winner per family (the metis#22 ensembling seam).
func TestSelect_PerModelClass_ReportsEachFamily(t *testing.T) {
	dir := t.TempDir()
	shapePath := writeSelectLedger(t, dir, true)
	var out strings.Builder
	if err := runSelect(selectOpts{shapePath: shapePath, perClass: true, out: &out}); err != nil {
		t.Fatalf("select --best-per-model-class: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "train.model=logreg") || !strings.Contains(s, "train.model=rf") {
		t.Errorf("--best-per-model-class should report both families; got:\n%s", s)
	}
}
