package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/ledger"
)

// familyFakeExec scores by MODEL FAMILY (a tagged-sum $any) so an auto-stop e2e has a clear
// winner (rf ~0.90) and a clear loser (logreg ~0.70), with a small per-outer-fold nudge so the
// held-out scores have non-zero spread (the predictive rule's variance path).
type familyFakeExec struct{}

func (familyFakeExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	stepDir := filepath.Join(runDir, step.ID)
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		return experiment.StepResult{}, err
	}
	art := step.ID + "/out.txt"
	if err := os.WriteFile(filepath.Join(runDir, filepath.FromSlash(art)), []byte(step.ID), 0o644); err != nil {
		return experiment.StepResult{}, err
	}
	metrics := map[string]float64{}
	if step.ID == "train" {
		base := map[string]float64{"rf": 0.90, "logreg": 0.70}[familyKeyOf(step.With)]
		metrics["fold_score"] = base + float64(foldIdxOf(step.With))*0.005
		metrics["complexity"] = 0
	}
	return experiment.StepResult{Metrics: metrics, Artifacts: []string{art}}, nil
}

// familyKeyOf reads the chosen tagged-sum branch label from a train step's `with.model`
// (a single-key map `{rf: {...}}`).
func familyKeyOf(with map[string]any) string {
	if m, ok := with["model"].(map[string]any); ok {
		for k := range m {
			return k
		}
	}
	return fmt.Sprint(with["model"])
}

// foldIdxOf reads the engine-injected fold index from a train step's `with._fold` (the OUTER
// fold on a held-out score run).
func foldIdxOf(with map[string]any) int {
	if f, ok := with["_fold"].(map[string]any); ok {
		if i, ok := f["idx"].(int); ok {
			return i
		}
	}
	return 0
}

// TestAutoStop_LoserStoppedWinnerFull is the metis#66 M2 gate (Done-when): with an incumbent
// read from the ledger, a known-loser family stops before full k while a would-be winner runs
// to full k and is never truncated. Incumbent 0.80 sits between rf (0.90) and logreg (0.70), so
// logreg is <95%-likely to reach it and rf clearly beats it.
func TestAutoStop_LoserStoppedWinnerFull(t *testing.T) {
	ws := t.TempDir()
	// 2 families (rf, logreg) via a tagged-sum $any MAP (label → sub-config); k=4 outer folds.
	shape := strings.Replace(foldShapeMD("{rf: {}, logreg: {}}"), "k: 2", "k: 4", 1)
	expPath := writeShapeFile(t, ws, shape)

	// Seed the incumbent as a prior-run OUTER aggregate row at 0.80 (a pass-through row —
	// Fold=nil — so AggregateView keeps its metric verbatim).
	seed := ledger.Ledger{}
	seed.Append(ledger.Row{
		Level: "outer", PointAddr: "prior-incumbent", CodeFingerprint: "prior", Status: "ok",
		FreeParams: map[string]any{"train.model": "prior"},
		Metrics:    map[string]float64{"train.fold_score": 0.80},
	})
	b, err := ledger.Encode(seed)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledgerPath(expPath), b, 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	_, err = runExperiment(runOpts{
		expPath:     expPath,
		now:         fixedNow(),
		git:         fakeGitProbe{name: "metis", sha: "sha"},
		exec:        familyFakeExec{},
		out:         &out,
		autoStop:    true,
		maxParallel: 1,
	})
	if err != nil {
		t.Fatalf("auto-stop run must succeed: %v\n%s", err, out.String())
	}
	s := out.String()

	// Per-family outer-fold coverage + stop marker from the persisted ledger.
	led := loadLedgerOrFatal(t, expPath)
	folds := map[string]map[int]bool{}
	stopped := map[string]bool{}
	for _, r := range led.Rows {
		if r.Level != "outer" || r.PointAddr == "prior-incumbent" {
			continue
		}
		fam := fmt.Sprint(r.FreeParams["train.model"])
		if folds[fam] == nil {
			folds[fam] = map[int]bool{}
		}
		if r.OuterFold != nil {
			folds[fam][*r.OuterFold] = true
		}
		if r.Stopped == "auto" {
			stopped[fam] = true
		}
	}

	if len(folds["rf"]) != 4 {
		t.Errorf("winner rf must run FULL k=4 outer folds (never truncated), got %d: %v", len(folds["rf"]), folds["rf"])
	}
	if n := len(folds["logreg"]); n == 0 || n >= 4 {
		t.Errorf("loser logreg must be auto-stopped before full k, got %d folds: %v", n, folds["logreg"])
	}
	if !stopped["logreg"] {
		t.Errorf("loser logreg's outer rows must be marked stopped:auto")
	}
	if stopped["rf"] {
		t.Errorf("winner rf must NEVER be marked stopped:auto")
	}
	if !strings.Contains(s, "auto-stop") {
		t.Errorf("the run must announce the incumbent + the auto-stop decision; got:\n%s", s)
	}
}
