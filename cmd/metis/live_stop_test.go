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
// out<n> estimate over the folds that COMPLETED (with a non-zero SE, n≥2), and (3) write a
// partial ledger containing only the completed folds' rows — never the abandoned fold's.
// Serial (maxParallel=1) so outer folds 0,1 fully complete before fold 2 starts; the stop
// fires on fold 2's first leaf → folds 0,1 count (out2, SE>0), fold 2 is abandoned.
func TestLive_QFinalizesHonestPartial(t *testing.T) {
	ws := t.TempDir()
	// k=4 outer folds (2 configs, one family): folds 0,1 complete; fold 2 triggers the stop
	// mid-flight; fold 3 never starts — proving the mid-fold abandon AND the not-yet-started skip.
	k4 := strings.Replace(foldShapeMD("[a, b]"), "k: 2", "k: 4", 1)
	expPath := writeShapeFile(t, ws, k4)

	ctrl := newRunControl(1)
	fire := make(chan struct{})
	exec := &stopOnFoldExec{in: foldFakeExec{}, ctrl: ctrl, fire: fire, marker: "analysis_2"}

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

	// (2) honest partial: STOPPED framing + out2 (folds 0,1 completed → the non-zero-SE path).
	if !strings.Contains(s, "STOPPED by request") {
		t.Errorf("stopped run must report the STOPPED framing; got:\n%s", s)
	}
	if !strings.Contains(s, "over 2 completed outer fold(s)") || !strings.Contains(s, "out2") {
		t.Errorf("stopped run must finalize as an honest out2 (two completed folds); got:\n%s", s)
	}
	// The out2 estimate must carry a NON-ZERO SE (the n≥2 branch of completedOuterEstimate),
	// distinguishing it from the degenerate single-fold case.
	est := estFields(s)
	if est.se <= 0 {
		t.Errorf("out2 partial estimate must have SE>0 (n≥2 path), got SE %.4f from %q", est.se, s)
	}

	// (3) partial ledger: outer rows for folds 0,1 ONLY — fold 2 (abandoned mid-flight) and
	// fold 3 (never started) contribute nothing.
	led := loadLedgerOrFatal(t, expPath)
	outerFolds := map[int]bool{}
	for _, r := range led.Rows {
		if r.Level == "outer" && r.OuterFold != nil {
			outerFolds[*r.OuterFold] = true
		}
	}
	if len(outerFolds) != 2 || !outerFolds[0] || !outerFolds[1] {
		t.Errorf("stopped ledger must carry ONLY the completed folds {0,1} outer rows, got %v", outerFolds)
	}
}

// estFields pulls the mean/SE out of the "STOPPED by request … mean M (SE S)" report line.
func estFields(s string) (r struct{ mean, se float64 }) {
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, "STOPPED by request") {
			fmt.Sscanf(l[strings.Index(l, "mean"):], "mean %f (SE %f)", &r.mean, &r.se)
			return r
		}
	}
	return r
}
