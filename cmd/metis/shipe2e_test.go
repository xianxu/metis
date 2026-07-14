package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
)

// soundFoldExec is the T20 honest-e2e inner exec: it drives the FULL metis#18 phase model
// (fold-aware fold_score like foldFakeExec) AND writes a real reads.json per step pointing at
// editable code files in a temp git repo (like traceFakeExec) — so the metis#24 transitive-D
// closure is genuine and an upstream CODE edit re-runs downstream THROUGH THE SWEEP, end-to-end
// (not just on the linear pipeline caching_soundness_test.go drives). The sweep's per-fold runs
// use the SAME cachingExecutor as that gate, so this ties the whole algebra together in one flow.
type soundFoldExec struct {
	codeRepo string            // a git repo holding the step code files the steps "read"
	reads    map[string]string // step-id → code file (relative to codeRepo) it reads → its D
	calls    *[]string         // MISS trace: the inner exec ran (a HIT skips it)
}

func (f soundFoldExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
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
	// Declare this step's read-set D (the code file it reads) so recordMiss folds it into the
	// transitive-D closure and isHit re-hashes it — the real metis#24 code-invalidation path.
	if rel, ok := f.reads[step.ID]; ok {
		rs := readSet{Roots: map[string][]string{f.codeRepo: {rel}}, UsedSitePackages: false}
		b, err := json.Marshal(rs)
		if err != nil {
			return experiment.StepResult{}, err
		}
		if err := os.WriteFile(filepath.Join(stepDir, "reads.json"), b, 0o644); err != nil {
			return experiment.StepResult{}, err
		}
	}
	metrics := map[string]float64{}
	if step.ID == "train" {
		metrics["fold_score"] = fakeTrainScore(step.With)
	}
	return experiment.StepResult{Metrics: metrics, Artifacts: []string{art}}, nil
}

// TestShapeSweep_HonestE2E is the metis#18 M1a-5 cohesive proof: the whole Sampler algebra in
// ONE flow on the cache — data + partition materialized ONCE above the sweeper, the sweeper ×
// inner folds → an honest per-config (mean, SE) leaderboard, argmax-mean winner, driver:single
// ship → a submission — then the metis#24 soundness gate END-TO-END: an upstream code edit
// re-runs the downstream folds through the sweep while the config/fold-invariant data + partition
// stay cached. The real Titanic 42-config run (Kaggle-creds-gated, sandbox-blocked per RUNBOOK)
// is the operator's honest-numbers check; the mechanism is proven offline here.
func TestShapeSweep_HonestE2E(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH (transitive-D re-hash uses git hash-object)")
	}
	ws := t.TempDir()
	codeRepo := t.TempDir()
	mustRun(t, codeRepo, "git", "init", "-q")
	for _, f := range []string{"features.py", "train.py"} {
		if err := os.WriteFile(filepath.Join(codeRepo, f), []byte("v = 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	expPath := writeShapeFile(t, ws, foldShapeShipMD("[a, b]"))
	reads := map[string]string{"features": "features.py", "train": "train.py"}

	run := func(calls *[]string, out *strings.Builder) {
		t.Helper()
		var w strings.Builder
		if out == nil {
			out = &w
		}
		_, err := runExperiment(runOpts{
			expPath: expPath, now: fixedNow(), git: fakeGitProbe{name: "metis", sha: "sha"},
			cache: true, exec: soundFoldExec{codeRepo: codeRepo, reads: reads, calls: calls}, out: out,
		})
		if err != nil {
			t.Fatalf("sweep+ship run: %v", err)
		}
	}

	// ── Cold: the full sweep + ship ────────────────────────────────────────────────
	var cold []string
	var out strings.Builder
	run(&cold, &out)

	// data + partition are config/fold-invariant → each runs ONCE (then HITs across the rest).
	if n := countCalls(cold, "get-data"); n != 1 {
		t.Errorf("get-data (config/fold-invariant) should run once, got %d", n)
	}
	if n := countCalls(cold, "adapt"); n != 1 {
		t.Errorf("adapt should run once, got %d", n)
	}
	// metis#32 nested: the partition (cv-split) is materialized ABOVE the inner sweeper (not per
	// point) — once per outer fold's sealed inner sweep (2) + the outer-fold held-out eval (1) = 3.
	if n := countCalls(cold, partitionStepID); n != 3 {
		t.Errorf("cv-split should materialize per outer fold + outer eval (3), not per point, got %d runs", n)
	}
	// metis#32: the nested run RECORDS the honest per-config data (inner rows) + per-family outer
	// rows to the ledger, and does NOT ship (shipping is `metis select --promote`).
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
	if nInner == 0 || nOuter == 0 {
		t.Errorf("nested run must record inner AND outer rows; got %d inner, %d outer", nInner, nOuter)
	}
	shipSteps, _ := filepath.Glob(filepath.Join(ws, "runs", "*", "submission", "out.txt"))
	if len(shipSteps) != 0 {
		t.Errorf("`metis run` must NOT ship, got %d submission artifacts", len(shipSteps))
	}
	if strings.Contains(out.String(), "shipped") {
		t.Errorf("`metis run` must not report shipping; got:\n%s", out.String())
	}

	// ── Warm re-run: everything HITs (incl. the ship) — 0 inner execs ──────────────
	var warm []string
	run(&warm, nil)
	if len(warm) != 0 {
		t.Errorf("a warm re-run should HIT everything incl. the ship (0 inner execs), got %d: %v", len(warm), warm)
	}

	// ── The metis#24 soundness gate, end-to-end through the sweep ───────────────────
	// Edit an UPSTREAM step's CODE (features.py). features' input-addressed K_pre is unchanged
	// and train's K_pre (keyed on features' code-invariant K_pre) is unchanged — only the
	// transitive-D closure stored in each downstream entry catches the edit. So the affected
	// downstream (train, every fold) MISSes + re-runs, while the config/fold-invariant data +
	// partition stay cached.
	if err := os.WriteFile(filepath.Join(codeRepo, "features.py"), []byte("v = 2  # edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var after []string
	run(&after, nil)
	if countCalls(after, "features") == 0 {
		t.Error("features (its own code edited) must MISS + re-run")
	}
	if countCalls(after, "train") == 0 {
		t.Error("train MUST MISS on the UPSTREAM features code edit — the transitive-D closure carries " +
			"features.py (the input-addressed K_pre alone would false-HIT)")
	}
	for _, invariant := range []string{"get-data", "adapt", partitionStepID} {
		if n := countCalls(after, invariant); n != 0 {
			t.Errorf("%s is upstream of the edit / config-invariant → must stay cached, got %d re-runs", invariant, n)
		}
	}
}

// metis#32 MIGRATION NOTE: the metis#14 ship-run code-capture invariant (was
// TestShapeSweep_ShipRunIsCodeCaptured) was DELETED here — `metis run` no longer ships, so there is
// no ship run on this path to capture. The invariant did NOT disappear: it moves to the `metis
// select --promote` ship path. **M2 must re-add it** as `TestSelectPromote_ShipRunIsCodeCaptured`
// (assert the promoted all-data ship run captures its OWN code closure + its side-ref resolves).
