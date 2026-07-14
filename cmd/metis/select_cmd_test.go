package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/metis/pkg/ledger"
)

// A tagged-sum shape (model: $any-MAP → families logreg/rf), so the offline select path
// exercises family grouping. Each branch sweeps one hyperparam.
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

// writeTaggedSelectLedger writes the tagged shape + a per-fold ledger carrying BOTH
// train.fold_score and train.complexity for the 4 configs (logreg C∈{0.1,1}, rf md∈{4,8}).
// rf md=8 is the argmax-mean champion (0.844) but the deep overfitter (cx 40); rf md=4
// (0.834, cx 16) is the shallower config pct-loss should recover.
func writeTaggedSelectLedger(t *testing.T, dir string) string {
	t.Helper()
	shapePath := filepath.Join(dir, "s.md")
	if err := os.WriteFile(shapePath, []byte(taggedShapeForSelect), 0o644); err != nil {
		t.Fatal(err)
	}
	row := func(addr string, fp map[string]any, fold int, score, cx float64) ledger.Row {
		ff := fold
		return ledger.Row{CodeFingerprint: "cf", PointAddr: addr, FreeParams: fp, Fold: &ff,
			Metrics: map[string]float64{"train.fold_score": score, "train.complexity": cx}, Status: "ok"}
	}
	lr01 := map[string]any{"train.model": "logreg", "train.model.logreg.C": 0.1}
	lr1 := map[string]any{"train.model": "logreg", "train.model.logreg.C": 1.0}
	rf4 := map[string]any{"train.model": "rf", "train.model.rf.max_depth": 4}
	rf8 := map[string]any{"train.model": "rf", "train.model.rf.max_depth": 8}
	var led ledger.Ledger
	led.Append(
		row("lr01_0", lr01, 0, 0.82, 6), row("lr01_1", lr01, 1, 0.82, 6),
		row("lr1_0", lr1, 0, 0.80, 6), row("lr1_1", lr1, 1, 0.80, 6),
		row("rf4_0", rf4, 0, 0.834, 16), row("rf4_1", rf4, 1, 0.834, 16),
		row("rf8_0", rf8, 0, 0.844, 40), row("rf8_1", rf8, 1, 0.844, 40),
	)
	b, err := ledger.Encode(led)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledgerPath(shapePath), b, 0o644); err != nil {
		t.Fatal(err)
	}
	return shapePath
}

// `metis ledger select --rule pct-loss` groups by family, applies the pure SelectConfigs
// rule offline, and prints per-family winners + the ship. The family key MUST be the
// path-qualified `train.model=rf` (matching sampler.FamilyOf) — a bare `rf` would prove the
// two selection surfaces diverged (the M1-review DRY finding).
func TestLedgerSelect_PctLoss_PerFamilyAndShip(t *testing.T) {
	dir := t.TempDir()
	shapePath := writeTaggedSelectLedger(t, dir)
	var out strings.Builder
	if err := runLedgerSelect(selectOpts{shapePath: shapePath, out: &out}); err != nil {
		t.Fatalf("ledger select: %v", err)
	}
	s := out.String()
	// DRY format: the family key is path-qualified (sampler.FamilyOf), not bare "rf".
	if !strings.Contains(s, "train.model=rf") || !strings.Contains(s, "train.model=logreg") {
		t.Errorf("expected path-qualified family keys train.model=rf / train.model=logreg; got:\n%s", s)
	}
	// pct-loss recovers the shallower rf md=4 (cx 16) over the argmax-mean champion rf md=8
	// (cx 40); the ship is argmax-mean over the family winners → rf md=4 (0.834 > logreg 0.82).
	if !strings.Contains(s, "max_depth=4") {
		t.Errorf("pct-loss should recover rf max_depth=4 (shallower than md=8); got:\n%s", s)
	}
	if strings.Contains(s, "max_depth=8") {
		t.Errorf("pct-loss should NOT select the deep rf md=8; got:\n%s", s)
	}
}

// The rule flag drives selection: argmax-mean picks the global-max config (rf md=8), unlike
// pct-loss. Proves the offline path honors the --rule flag, not just the shape default.
func TestLedgerSelect_ArgmaxMean_PicksGlobalMax(t *testing.T) {
	dir := t.TempDir()
	shapePath := writeTaggedSelectLedger(t, dir)
	var out strings.Builder
	if err := runLedgerSelect(selectOpts{shapePath: shapePath, rule: "argmax-mean", out: &out}); err != nil {
		t.Fatalf("ledger select: %v", err)
	}
	if s := out.String(); !strings.Contains(s, "max_depth=8") {
		t.Errorf("argmax-mean should ship the global-max rf md=8; got:\n%s", s)
	}
}

// The guard (metis#19): a parsimony rule over a ledger that carries NO complexity → a hard
// error (a silently-dropped parsimony axis would give a quietly-wrong winner). argmax-mean
// over the same ledger is fine (never reads complexity).
func TestLedgerSelect_ParsimonyWithoutComplexity_Errors(t *testing.T) {
	dir := t.TempDir()
	shapePath := writeTaggedSelectLedger(t, dir)
	// Rewrite the ledger WITHOUT the complexity column (simulate a pre-metis#19 sweep).
	stripComplexityFromLedger(t, shapePath)

	var out strings.Builder
	err := runLedgerSelect(selectOpts{shapePath: shapePath, rule: "pct-loss", out: &out})
	if err == nil {
		t.Fatalf("pct-loss over a complexity-less ledger must error; got success:\n%s", out.String())
	}
	if !strings.Contains(err.Error(), "complexity") {
		t.Errorf("guard error should mention complexity; got %v", err)
	}
	// argmax-mean over the same ledger is fine.
	out.Reset()
	if err := runLedgerSelect(selectOpts{shapePath: shapePath, rule: "argmax-mean", out: &out}); err != nil {
		t.Errorf("argmax-mean must not need complexity; got %v", err)
	}
}

// stripComplexityFromLedger rewrites the shape's ledger keeping only train.fold_score.
func stripComplexityFromLedger(t *testing.T, shapePath string) {
	t.Helper()
	led, err := loadLedger(shapePath)
	if err != nil {
		t.Fatal(err)
	}
	var stripped ledger.Ledger
	for _, r := range led.Rows {
		m := map[string]float64{}
		if v, ok := r.Metrics["train.fold_score"]; ok {
			m["train.fold_score"] = v
		}
		nr := r
		nr.Metrics = m
		stripped.Append(nr)
	}
	b, err := ledger.Encode(stripped)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledgerPath(shapePath), b, 0o644); err != nil {
		t.Fatal(err)
	}
}
