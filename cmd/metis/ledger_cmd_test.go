package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		ledger.Row{CodeFingerprint: "cf", PointAddr: "a0", FreeParams: map[string]any{"train.model": "a"}, Fold: &f0, Metrics: map[string]float64{"train.fold_score": 0.80}, Status: "ok"},
		ledger.Row{CodeFingerprint: "cf", PointAddr: "a1", FreeParams: map[string]any{"train.model": "a"}, Fold: &f1, Metrics: map[string]float64{"train.fold_score": 0.90}, Status: "ok"},
		ledger.Row{CodeFingerprint: "cf", PointAddr: "b0", FreeParams: map[string]any{"train.model": "b"}, Fold: &f0, Metrics: map[string]float64{"train.fold_score": 0.70}, Status: "ok"},
		ledger.Row{CodeFingerprint: "cf", PointAddr: "b1", FreeParams: map[string]any{"train.model": "b"}, Fold: &f1, Metrics: map[string]float64{"train.fold_score": 0.72}, Status: "ok"},
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
