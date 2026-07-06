package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/shape"
)

// A sweep writes the shape's ledger sidecar (one row per point, namespaced metrics) and
// regenerates the body top-N summary.
func TestLedger_SweepWritesSidecarAndSummary(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := writeShape(t, ws, `---
type: experiment-shape
id: led
seed: 5
status: active
sweep: {sampler: grid, objective: {metric: train.echoed, direction: maximize}}
steps:
  - id: train
    uses: test/echo
    with: {model: {$any: [logreg, rf]}}
---

body here
`)
	if err := runSweepViaRun(t, expPath, root, runOpts{cache: false}); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	// The ledger sidecar exists with 2 rows + the union columns.
	csv, err := os.ReadFile(ledgerPath(expPath))
	if err != nil {
		t.Fatalf("ledger sidecar not written: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(csv)), "\n")
	if len(lines) != 3 { // header + 2 rows
		t.Errorf("ledger should have header + 2 rows, got %d lines:\n%s", len(lines), csv)
	}
	if !strings.Contains(lines[0], "fp.train.model") || !strings.Contains(lines[0], "metric.train.echoed") {
		t.Errorf("ledger header missing expected columns: %s", lines[0])
	}
	// The body top-N summary regenerated between the markers.
	body, _ := os.ReadFile(expPath)
	if !strings.Contains(string(body), "metis:ledger:begin") || !strings.Contains(string(body), "## Top runs") {
		t.Errorf("body summary not regenerated:\n%s", body)
	}
	// Idempotent: a second identical sweep dedups (still 2 rows).
	if err := runSweepViaRun(t, expPath, root, runOpts{cache: false}); err != nil {
		t.Fatal(err)
	}
	csv2, _ := os.ReadFile(ledgerPath(expPath))
	if n := len(strings.Split(strings.TrimSpace(string(csv2)), "\n")); n != 3 {
		t.Errorf("re-sweep must dedup (still 2 rows), got %d lines", n)
	}
}

// promote --best reconstructs the winning point as an all-singleton experiment that
// round-trips: it re-runs and reproduces the winner's run.
func TestLedger_PromoteBestRoundTrips(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	// A 2-point sweep; test/echo emits echoed=1 for both, so "best" is deterministic
	// (the first by objective order). We assert the promoted experiment re-runs.
	expPath := writeShape(t, ws, `---
type: experiment-shape
id: promo
seed: 5
status: active
sweep: {sampler: grid, objective: {metric: train.echoed, direction: maximize}}
steps:
  - id: train
    uses: test/echo
    with: {model: {$any: [logreg, rf]}}
---
`)
	if err := runSweepViaRun(t, expPath, root, runOpts{cache: false}); err != nil {
		t.Fatal(err)
	}
	if err := runPromote(promoteOpts{
		shapePath: expPath, best: true, name: "winner",
		out: os.Stdout, git: fakeGitProbe{name: "metis", sha: "sha", dirty: false},
	}); err != nil {
		t.Fatalf("promote --best: %v", err)
	}
	winnerPath := filepath.Join(ws, "winner.md")
	wb, err := os.ReadFile(winnerPath)
	if err != nil {
		t.Fatalf("winner.md not written: %v", err)
	}
	if !strings.Contains(string(wb), "type: experiment") || !strings.Contains(string(wb), "promoted_from: promo") {
		t.Errorf("promoted experiment malformed:\n%s", wb)
	}
	// The promoted experiment's id must match its <name>.md filename (the experiment
	// convention), NOT the shape's id.
	if !strings.Contains(string(wb), "id: winner") {
		t.Errorf("promoted experiment id must be the --name (winner), not the shape id:\n%s", wb)
	}
	// Round-trip: the promoted all-singleton experiment re-runs (like a v0 experiment).
	run, err := runExperiment(runOpts{
		expPath:  winnerPath,
		runID:    "rt",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      fixedNow(),
		git:      fakeGitProbe{name: "metis", sha: "sha", dirty: false},
		out:      io.Discard,
	})
	if err != nil {
		t.Fatalf("promoted experiment must re-run: %v", err)
	}
	if run.Status != "ok" {
		t.Errorf("promoted round-trip status = %q; want ok", run.Status)
	}
}

// Immutability by per-row snapshot: after a shape's SPACE is edited, prior rows still
// reproduce — because each row snapshots its full point (free-params) + sweep-SHA, and
// promote reconstructs from the row, not the edited space.
func TestLedger_PriorRowsReproduceAfterSpaceEdit(t *testing.T) {
	// A row captured under the ORIGINAL space (model ∈ {logreg, rf}).
	origShape := experiment.Shape{
		Experiment: experiment.Experiment{
			Type: "experiment-shape", ID: "s", Seed: 5, Status: "active",
			Steps: []experiment.Step{{ID: "train", Uses: "test/echo",
				With: map[string]any{"model": map[string]any{"$any": []any{"logreg", "rf"}}}}},
		},
		Sweep: experiment.Sweep{Sampler: "grid"},
	}
	// The row snapshots the free-param point model=rf.
	rowFP := map[string]any{"train.model": "rf"}

	// Now the space is EDITED (model ∈ {gbm, svm} — rf removed).
	editedShape := origShape
	editedShape.Steps = []experiment.Step{{ID: "train", Uses: "test/echo",
		With: map[string]any{"model": map[string]any{"$any": []any{"gbm", "svm"}}}}}

	// The prior row still reconstructs against the ORIGINAL space (its snapshot) — the
	// per-row snapshot is self-contained, so promotion of an old row is stable.
	exp, err := promotedExperiment(origShape, rowFP)
	if err != nil {
		t.Fatalf("prior row must reproduce against its snapshot space: %v", err)
	}
	if exp.Steps[0].With["model"] != "rf" {
		t.Errorf("prior row reconstructed to model=%v; want rf", exp.Steps[0].With["model"])
	}
	// Against the EDITED space the point no longer exists — which is exactly why the row
	// carries its own snapshot rather than depending on the mutable space.
	if _, err := promotedExperiment(editedShape, rowFP); err == nil {
		t.Error("the edited space no longer contains model=rf — the row's snapshot is what keeps it reproducible")
	}
	_ = shape.Point{} // (shape imported for the reconstruction path)
}
