package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
)

// stopOnFoldExec fires the metis#66 graceful-stop signal (as if the operator pressed Q) the
// first time a leaf reads a given outer fold's sealed data (`analysis_<n>`), then spins until
// the stop has propagated through the bridge — so the test is deterministic (no sleep race):
// by the time the NEXT leaf checks the stop gate it is definitely set. It exercises the REAL
// path (stopSignal channel → runControl.requestStop), not a direct latch.
type stopOnFoldExec struct {
	in     foldFakeExec
	ctrl   *runControl
	fire   chan struct{} // closed to trigger the stopSignal bridge
	marker string        // e.g. "analysis_1" — the outer fold whose start triggers the stop
	once   sync.Once
}

func (e *stopOnFoldExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	if strings.Contains(fmt.Sprint(step.With["dataset"]), e.marker) {
		e.once.Do(func() {
			close(e.fire) // fire Q
			for !e.ctrl.stopRequested() {
				runtime.Gosched() // deterministically wait for the bridge to latch the stop
			}
		})
	}
	return e.in.Execute(step, runDir)
}

// TestLive_QFinalizesHonestPartial is the metis#66 M1 graceful-stop gate: a Q while a nested
// run is in flight must (1) finalize cleanly (no error), (2) report an honest partial
// out<n> estimate over the folds that COMPLETED, and (3) write a partial ledger containing
// only the completed folds' rows — never the abandoned fold's. Serial (maxParallel=1) so
// outer fold 0 fully completes before fold 1 starts; the stop fires on fold 1's first leaf
// → fold 0 counts (out1), fold 1 is abandoned.
func TestLive_QFinalizesHonestPartial(t *testing.T) {
	ws := t.TempDir()
	// k=3 outer folds (2 configs, one family): fold 0 completes, fold 1 triggers the stop,
	// fold 2 never starts — proving both the mid-fold abandon and the not-yet-started skip.
	k3 := strings.Replace(foldShapeMD("[a, b]"), "k: 2", "k: 3", 1)
	expPath := writeShapeFile(t, ws, k3)

	ctrl := newRunControl(1)
	fire := make(chan struct{})
	exec := &stopOnFoldExec{in: foldFakeExec{}, ctrl: ctrl, fire: fire, marker: "analysis_1"}

	var out strings.Builder
	_, err := runExperiment(runOpts{
		expPath:    expPath,
		now:        fixedNow(),
		git:        fakeGitProbe{name: "metis", sha: "sha"},
		exec:       exec,
		out:        &out,
		live:       true,
		stopSignal: fire,
		runControl: ctrl,
	})
	if err != nil {
		t.Fatalf("a graceful Q-stop must finalize cleanly (no error), got: %v\n%s", err, out.String())
	}
	s := out.String()

	// (2) honest partial: STOPPED framing + out1 (only fold 0 completed).
	if !strings.Contains(s, "STOPPED by request") {
		t.Errorf("stopped run must report the STOPPED framing; got:\n%s", s)
	}
	if !strings.Contains(s, "over 1 completed outer fold(s)") || !strings.Contains(s, "out1") {
		t.Errorf("stopped run must finalize as an honest out1 (one completed fold); got:\n%s", s)
	}

	// (3) partial ledger: outer rows for fold 0 ONLY — fold 1 (abandoned mid-flight) and
	// fold 2 (never started) contribute nothing.
	led := loadLedgerOrFatal(t, expPath)
	outerFolds := map[int]bool{}
	for _, r := range led.Rows {
		if r.Level == "outer" && r.OuterFold != nil {
			outerFolds[*r.OuterFold] = true
		}
	}
	if len(outerFolds) != 1 || !outerFolds[0] {
		t.Errorf("stopped ledger must carry ONLY the completed fold 0's outer rows, got folds %v", outerFolds)
	}
}
