package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
)

// budgetFakeExec routes the metis#18 fake leaf through the REAL shared leaf budget (metis#66)
// so a determinism test actually exercises the prioritySem's acquire/release/heap-transfer
// under the true nested fan-out (with -race), not just the order-independent reduce. It
// acquires at priority 0 (the per-leaf outer-fold priority lives on the production execStep,
// which the injected exec bypasses — grant-ORDER is covered by prioritysem_test; this proves
// the sem is deadlock-free + result-invariant end-to-end).
type budgetFakeExec struct {
	budget leafBudget
	in     foldFakeExec
}

func (b budgetFakeExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	if b.budget != nil {
		b.budget.acquire(0)
		defer b.budget.release()
	}
	return b.in.Execute(step, runDir)
}

// TestLive_ByteIdenticalToDefault is the metis#66/#67 determinism gate: the default prioritySem
// (fold-ordered) MUST produce byte-identical artifacts to the --global-fanout chanSem — the reduce
// is order-independent (metis#18/#31) and sortPointRuns normalizes the
// on-disk order, so changing WHICH leaf runs when cannot change the numbers. Runs the same
// nested shape through both budgets (each fake leaf actually acquiring its budget) in isolated
// workspaces and asserts the ledger bytes, the manifest bytes, and the reported estimate match.
func TestLive_ByteIdenticalToDefault(t *testing.T) {
	shape := foldShapeMD("[a, b, c]") // 3 configs → nested; sweeper cv.k=2 → 2 outer folds

	run := func(t *testing.T, live bool) (ledgerBytes, manifestBytes, estimate string) {
		t.Helper()
		ws := t.TempDir()
		expPath := writeShapeFile(t, ws, shape)
		var budget leafBudget
		if live {
			budget = newPrioritySem(4)
		} else {
			budget = newChanSem(4)
		}
		var out strings.Builder
		_, err := runExperiment(runOpts{
			expPath:     expPath,
			now:         fixedNow(),
			git:         fakeGitProbe{name: "metis", sha: "sha"},
			exec:        budgetFakeExec{budget: budget, in: foldFakeExec{}},
			out:         &out,
			maxParallel: 4,
			leafBudget:  budget, // runExperiment reuses this (non-nil); the fake acquires the SAME one
		})
		if err != nil {
			t.Fatalf("live=%v run failed: %v", live, err)
		}
		led, err := os.ReadFile(ledgerPath(expPath))
		if err != nil {
			t.Fatalf("read ledger: %v", err)
		}
		manFiles, _ := filepath.Glob(filepath.Join(ws, "sweeps", "*", "manifest.json"))
		if len(manFiles) != 1 {
			t.Fatalf("want exactly one manifest, got %d", len(manFiles))
		}
		man, err := os.ReadFile(manFiles[0])
		if err != nil {
			t.Fatalf("read manifest: %v", err)
		}
		return string(led), string(man), estimateLine(out.String())
	}

	dLed, dMan, dEst := run(t, false)
	lLed, lMan, lEst := run(t, true)

	if dLed != lLed {
		t.Errorf("--live ledger differs from default (scheduling must not change the numbers)\ndefault:\n%s\n--live:\n%s", dLed, lLed)
	}
	if dMan != lMan {
		t.Errorf("--live manifest.json differs from default")
	}
	if dEst == "" {
		t.Fatalf("no estimate line captured (default run)")
	}
	if dEst != lEst {
		t.Errorf("--live estimate differs from default:\n default: %q\n --live:  %q", dEst, lEst)
	}
}

// estimateLine extracts the single nested-CV estimate result line from a run's output.
func estimateLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, "nested-CV estimate") {
			return strings.TrimSpace(l)
		}
	}
	return ""
}
