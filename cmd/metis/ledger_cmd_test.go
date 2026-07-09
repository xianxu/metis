package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/ledger"
)

// A minimal parse-able experiment-shape (runPromote/cmdLedger only ParseShape it + read the
// objective; they don't run it), plus a per-fold ledger sidecar written beside it.
const foldShapeForLedger = `---
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
    with: {dataset: adapt, model: {$any: [a, b]}}
sweeper:
  sampler: grid
  resample: {cv: {k: 2, stratify: false}}
  objective: {metric: train.fold_score, direction: maximize, select: {argmax-mean: {}}}
driver:
  single: {}
---
`

func writePerFoldLedger(t *testing.T, dir string) string {
	t.Helper()
	shapePath := filepath.Join(dir, "s.md")
	if err := os.WriteFile(shapePath, []byte(foldShapeForLedger), 0o644); err != nil {
		t.Fatal(err)
	}
	f0, f1 := 0, 1
	var led ledger.Ledger
	led.Append(
		ledger.Row{SweepSHA: "sha", PointAddr: "a0", FreeParams: map[string]any{"train.model": "a"}, Fold: &f0, Metrics: map[string]float64{"train.fold_score": 0.80}, Status: "ok"},
		ledger.Row{SweepSHA: "sha", PointAddr: "a1", FreeParams: map[string]any{"train.model": "a"}, Fold: &f1, Metrics: map[string]float64{"train.fold_score": 0.90}, Status: "ok"},
		ledger.Row{SweepSHA: "sha", PointAddr: "b0", FreeParams: map[string]any{"train.model": "b"}, Fold: &f0, Metrics: map[string]float64{"train.fold_score": 0.70}, Status: "ok"},
		ledger.Row{SweepSHA: "sha", PointAddr: "b1", FreeParams: map[string]any{"train.model": "b"}, Fold: &f1, Metrics: map[string]float64{"train.fold_score": 0.72}, Status: "ok"},
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

// promote --best on a per-fold sweep ledger reconstructs a RUNNABLE single experiment
// (metis#18 M1a-5): it reduces the raw fold rows to per-config (mean, SE), picks the champion,
// and writes data ++ pipeline(winner config) ++ ship — NO cv-split (the ship refit needs no
// CV) — with the honest (mean, SE) recorded in the provenance. Winner = config a (0.85 > 0.71).
func TestPromote_PerFoldLedgerReconstructsRunnable(t *testing.T) {
	dir := t.TempDir()
	shapePath := writePerFoldLedger(t, dir)
	var out strings.Builder
	if err := runPromote(promoteOpts{shapePath: shapePath, best: true, name: "winner", out: &out}); err != nil {
		t.Fatalf("promote --best on a per-fold ledger must reconstruct a runnable experiment: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "winner.md"))
	if err != nil {
		t.Fatalf("promote must write winner.md: %v", err)
	}
	// The promoted .md parses + validates as a runnable plain experiment.
	exp, err := experiment.Parse(string(raw))
	if err != nil {
		t.Fatalf("promoted experiment must parse: %v", err)
	}
	if err := experiment.Validate(exp); err != nil {
		t.Fatalf("promoted experiment must validate (runnable): %v", err)
	}
	steps := map[string]experiment.Step{}
	for _, s := range exp.Steps {
		steps[s.ID] = s
	}
	if _, ok := steps[partitionStepID]; ok {
		t.Error("the promoted ship experiment must NOT carry a cv-split step (no CV at ship)")
	}
	if steps["train"].With["model"] != "a" {
		t.Errorf("promoted train should carry the winner's model=a; got %v", steps["train"].With["model"])
	}
	// The honest sweep estimate (mean, SE) is recorded in the promotion provenance (mean 0.85).
	if !strings.Contains(string(raw), "sweep_estimate") || !strings.Contains(string(raw), "0.85") {
		t.Errorf("promoted experiment must record the honest (mean, SE) estimate; got:\n%s", raw)
	}
}

// promote --point on a per-fold ledger selects a CONFIG by its free-params via the SAME
// aggregate-then-select path as --best (not one fold's raw row) — metis#18 M1a-5: `--point
// train.model=b` promotes config b with its honest estimate (b: (0.70+0.72)/2 = 0.71).
func TestPromote_PerFoldLedgerByPoint(t *testing.T) {
	dir := t.TempDir()
	shapePath := writePerFoldLedger(t, dir)
	var out strings.Builder
	if err := runPromote(promoteOpts{shapePath: shapePath, point: "train.model=b", name: "winner", out: &out}); err != nil {
		t.Fatalf("promote --point on a per-fold ledger must reconstruct config b: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "winner.md"))
	if err != nil {
		t.Fatalf("promote --point must write winner.md: %v", err)
	}
	exp, err := experiment.Parse(string(raw))
	if err != nil {
		t.Fatalf("promoted experiment must parse: %v", err)
	}
	for _, s := range exp.Steps {
		if s.ID == "train" && s.With["model"] != "b" {
			t.Errorf("--point train.model=b should promote config b; got %v", s.With["model"])
		}
	}
	// The honest estimate is recorded for the SELECTED config (b: mean 0.71), not the champion.
	if !strings.Contains(string(raw), "sweep_estimate") || !strings.Contains(string(raw), "0.71") {
		t.Errorf("--point promote must record config b's honest estimate (~0.71); got:\n%s", raw)
	}
}

// `ledger show --sort` on a per-fold ledger renders the AggregateView (per-config mean,SE),
// not the raw fold rows — the honest leaderboard.
func TestShowLedger_AggregatesPerConfig(t *testing.T) {
	dir := t.TempDir()
	shapePath := writePerFoldLedger(t, dir)
	var out strings.Builder
	if err := showLedger(shapePath, "", "train.fold_score", "maximize", 0, &out); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	// 2 configs (a, b) — not 4 raw fold rows. Config a (mean 0.85) sorts above b (0.71).
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) != 1+2 { // header + 2 config rows
		t.Fatalf("expected a header + 2 per-config rows (aggregated), got %d lines:\n%s", len(lines), s)
	}
	// argmax-mean: config a (0.85) is best-first.
	if !strings.Contains(lines[1], "model=a") {
		t.Errorf("best-first row should be config a (mean 0.85); got: %s", lines[1])
	}
	if !strings.Contains(s, "train.fold_score.se") {
		t.Errorf("the aggregate view should carry the SE column; got:\n%s", s)
	}
}

// hoistShapePath pulls the single <shape.md> positional out regardless of flag position.
func TestHoistShapePath_ArgOrder(t *testing.T) {
	// flags before AND after the path both work (the stdlib-flag-stops-at-positional fix).
	for _, args := range [][]string{
		{"foo.md", "--sort", "train.fold_score"},
		{"--sort", "train.fold_score", "foo.md"},
	} {
		p, flags, err := hoistShapePath(args)
		if err != nil || p != "foo.md" {
			t.Errorf("hoistShapePath(%v) = (%q, %v); want foo.md", args, p, err)
		}
		if len(flags) != 2 {
			t.Errorf("hoistShapePath(%v) flags = %v; want the 2 flag tokens", args, flags)
		}
	}
	// Missing / duplicate positionals error.
	if _, _, err := hoistShapePath([]string{"--sort", "x"}); err == nil {
		t.Error("missing <shape.md> must error")
	}
	if _, _, err := hoistShapePath([]string{"a.md", "b.md"}); err == nil {
		t.Error("two <shape.md> positionals must error")
	}
}
