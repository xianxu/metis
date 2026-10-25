package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
)

// metis#72 legacy-path lifecycle: every way a launched child can end, driven by real shell
// fixtures (testdata/steps/test/*) that mark readiness with a file and block on a FIFO the test
// controls, so no assertion depends on sleep timing to know the child has started.

func lifecycleExec(t *testing.T) execStep {
	t.Helper()
	return execStep{stepPath: []string{filepath.Join(repoRoot(t), "testdata", "steps")}, expDir: t.TempDir(),
		seed: 1, out: io.Discard, grace: 200 * time.Millisecond, joinBound: 300 * time.Millisecond}
}

// gatedRun makes a run dir holding the FIFO test/gated blocks on.
func gatedRun(t *testing.T) string {
	t.Helper()
	runDir := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(runDir, "gate"), 0o600); err != nil {
		t.Fatal(err)
	}
	return runDir
}

func openGate(t *testing.T, runDir string) {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(runDir, "gate"), os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("go\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func waitFor(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", path)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func readOutcome(t *testing.T, stepDir string) Outcome {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(stepDir, "outcome.json"))
	if err != nil {
		t.Fatalf("outcome.json: %v", err)
	}
	var o Outcome
	if err := json.Unmarshal(b, &o); err != nil {
		t.Fatalf("outcome.json: %v\n%s", err, b)
	}
	return o
}

type execResult struct {
	res experiment.StepResult
	err error
}

func startExec(ctx context.Context, e execStep, step experiment.Step, runDir string) <-chan execResult {
	ch := make(chan execResult, 1)
	go func() {
		res, err := e.Execute(ctx, step, runDir)
		ch <- execResult{res, err}
	}()
	return ch
}

func withTimeout(id, uses, timeout string) experiment.Step {
	return experiment.Step{ID: id, Uses: uses, With: map[string]any{"exec": map[string]any{"timeout": timeout}}}
}

// Output is readable in step.log while the child runs; an ordinary success is unchanged and its
// bookkeeping files are not artifacts.
func TestExecStep_LiveLogVisibleWhileRunning(t *testing.T) {
	e, runDir := lifecycleExec(t), gatedRun(t)
	stepDir := filepath.Join(runDir, "s")
	done := startExec(context.Background(), e, experiment.Step{ID: "s", Uses: "test/gated"}, runDir)
	waitFor(t, filepath.Join(stepDir, "ready"))
	live, err := os.ReadFile(filepath.Join(stepDir, "step.log"))
	if err != nil || !strings.Contains(string(live), "before-gate") || strings.Contains(string(live), "after-gate") {
		t.Fatalf("step.log while running = %q, %v; want the first line only", live, err)
	}
	openGate(t, runDir)
	r := <-done
	if r.err != nil {
		t.Fatal(r.err)
	}
	if o := readOutcome(t, stepDir); o.Outcome != outcomeOK || o.ExitCode == nil || *o.ExitCode != 0 || o.RuntimeSeconds == nil {
		t.Errorf("outcome = %+v, want ok with exit 0 and a runtime", o)
	}
	if log, _ := os.ReadFile(filepath.Join(stepDir, "step.log")); !strings.Contains(string(log), "after-gate") {
		t.Errorf("final step.log = %q", log)
	}
	for _, a := range r.res.Artifacts {
		if base := filepath.Base(a); base == "step.log" || base == "outcome.json" {
			t.Errorf("bookkeeping file %s recorded as an artifact", a)
		}
	}
}

func TestExecStep_FailureCarriesLogTail(t *testing.T) {
	e, runDir := lifecycleExec(t), gatedRun(t)
	if err := os.WriteFile(filepath.Join(runDir, "exit-code"), []byte("3"), 0o644); err != nil {
		t.Fatal(err)
	}
	done := startExec(context.Background(), e, experiment.Step{ID: "s", Uses: "test/gated"}, runDir)
	waitFor(t, filepath.Join(runDir, "s", "ready"))
	openGate(t, runDir)
	r := <-done
	if r.err == nil || !strings.Contains(r.err.Error(), "exit status 3") || !strings.Contains(r.err.Error(), "after-gate") {
		t.Fatalf("err = %v; want exit status 3 with the log tail", r.err)
	}
	if o := readOutcome(t, filepath.Join(runDir, "s")); o.Outcome != outcomeFailed || *o.ExitCode != 3 {
		t.Errorf("outcome = %+v, want failed, exit 3", o)
	}
}

func TestExecStep_TimeoutTerminatesWithTERM(t *testing.T) {
	e, runDir := lifecycleExec(t), gatedRun(t)
	start := time.Now()
	_, err := e.Execute(context.Background(), withTimeout("s", "test/gated", "300ms"), runDir)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err = %v, want a timeout", err)
	}
	if took := time.Since(start); took > 300*time.Millisecond+e.grace+2*time.Second {
		t.Errorf("termination took %v, want about timeout + grace", took)
	}
	if o := readOutcome(t, filepath.Join(runDir, "s")); o.Outcome != outcomeTimedOut || o.Signal != "TERM" {
		t.Errorf("outcome = %+v, want timed-out by TERM", o)
	}
}

func TestExecStep_IgnoredTERMEscalatesToKILL(t *testing.T) {
	e, runDir := lifecycleExec(t), t.TempDir()
	_, err := e.Execute(context.Background(), withTimeout("s", "test/term-ignorer", "300ms"), runDir)
	if err == nil || !strings.Contains(err.Error(), "KILL") {
		t.Fatalf("err = %v, want a timeout ended by KILL", err)
	}
	if o := readOutcome(t, filepath.Join(runDir, "s")); o.Outcome != outcomeTimedOut || o.Signal != "KILL" {
		t.Errorf("outcome = %+v, want timed-out by KILL", o)
	}
}

func killOrphan(t *testing.T, runDir string) {
	t.Cleanup(func() {
		if b, err := os.ReadFile(filepath.Join(runDir, "orphan.pid")); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	})
}

// An escaped descendant holding the log pipe means the join cannot be confirmed, whatever else
// happened: a clean exit, a failure, or a timeout (close review BR-1: Go reports ErrWaitDelay
// only after a successful exit, so the drain is tracked on metis's own pipe).
func TestExecStep_EscapedGrandchildIsCleanupUnconfirmed(t *testing.T) {
	for _, c := range []struct {
		name, exitCode string
		gate           bool
		step           experiment.Step
	}{
		{"exit 0", "", false, experiment.Step{ID: "s", Uses: "test/escaper"}},
		{"exit 1", "1", false, experiment.Step{ID: "s", Uses: "test/escaper"}},
		{"timeout", "", true, withTimeout("s", "test/escaper", "300ms")},
	} {
		t.Run(c.name, func(t *testing.T) {
			e, runDir := lifecycleExec(t), t.TempDir()
			killOrphan(t, runDir)
			if c.gate {
				if err := syscall.Mkfifo(filepath.Join(runDir, "gate"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if c.exitCode != "" {
				if err := os.WriteFile(filepath.Join(runDir, "exit-code"), []byte(c.exitCode), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			_, err := e.Execute(context.Background(), c.step, runDir)
			if err == nil || !strings.Contains(err.Error(), "cleanup unconfirmed") {
				t.Fatalf("err = %v, want cleanup unconfirmed (a descendant still holds the log pipe)", err)
			}
			if o := readOutcome(t, filepath.Join(runDir, "s")); o.Outcome != outcomeCleanupUnconfirmed {
				t.Errorf("outcome = %+v, want cleanup-unconfirmed", o)
			}
		})
	}
}

// A step that cannot even start still records its ending through the one classifier.
func TestExecStep_StartFailureRecordsOutcome(t *testing.T) {
	e, runDir := lifecycleExec(t), t.TempDir()
	steps := t.TempDir()
	if err := os.MkdirAll(filepath.Join(steps, "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(steps, "test", "not-executable"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.stepPath = []string{steps}
	_, err := e.Execute(context.Background(), experiment.Step{ID: "s", Uses: "test/not-executable"}, runDir)
	if err == nil || !strings.Contains(err.Error(), "start") {
		t.Fatalf("err = %v, want a start failure", err)
	}
	if o := readOutcome(t, filepath.Join(runDir, "s")); o.Outcome != outcomeFailed || o.Started != nil {
		t.Errorf("outcome = %+v, want failed with no launch time", o)
	}
}

// An unreadable pending-resources.json is not evidence of "none": the step is not a success.
func TestExecStep_UnreadablePendingResourcesRefuseSuccess(t *testing.T) {
	e, runDir := lifecycleExec(t), t.TempDir()
	_, err := e.Execute(context.Background(), experiment.Step{ID: "s", Uses: "test/pending-unreadable"}, runDir)
	if err == nil || !strings.Contains(err.Error(), "resources unretired") {
		t.Fatalf("err = %v, want resources unretired", err)
	}
	if o := readOutcome(t, filepath.Join(runDir, "s")); o.Outcome != outcomeResourcesUnretired {
		t.Errorf("outcome = %+v", o)
	}
}

// runtime_seconds is compute time, launch to exit: termination (grace, join) is not counted.
func TestExecStep_RuntimeExcludesTermination(t *testing.T) {
	e, runDir := lifecycleExec(t), t.TempDir()
	e.grace = 400 * time.Millisecond
	if _, err := e.Execute(context.Background(), withTimeout("s", "test/term-ignorer", "200ms"), runDir); err == nil {
		t.Fatal("want a timeout")
	}
	o := readOutcome(t, filepath.Join(runDir, "s"))
	if o.RuntimeSeconds == nil || *o.RuntimeSeconds < 0.55 {
		t.Fatalf("runtime = %v; the child ran until KILLed, after the timeout plus grace", o.RuntimeSeconds)
	}
	if o.Ended == nil || math.Abs(o.Ended.Sub(*o.Started).Seconds()-*o.RuntimeSeconds) > 0.01 {
		t.Errorf("ended must be the exit time: %+v", o)
	}
}

func TestExecStep_PendingResourcesRefuseSuccess(t *testing.T) {
	e, runDir := lifecycleExec(t), t.TempDir()
	res, err := e.Execute(context.Background(), experiment.Step{ID: "s", Uses: "test/leaves-resources"}, runDir)
	if err == nil || !strings.Contains(err.Error(), "actor-1") {
		t.Fatalf("err = %v, want resources-unretired naming actor-1", err)
	}
	if o := readOutcome(t, filepath.Join(runDir, "s")); o.Outcome != outcomeResourcesUnretired || !strings.Contains(o.PendingResources, "actor-1") {
		t.Errorf("outcome = %+v", o)
	}
	if len(res.Artifacts) != 0 {
		t.Errorf("a refused step publishes no result, got %v", res.Artifacts)
	}
}

func TestExecStep_CancelWhileRunning(t *testing.T) {
	e, runDir := lifecycleExec(t), gatedRun(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := startExec(ctx, e, experiment.Step{ID: "s", Uses: "test/gated"}, runDir)
	waitFor(t, filepath.Join(runDir, "s", "ready"))
	cancel()
	r := <-done
	if r.err == nil || !strings.Contains(r.err.Error(), "cancelled") {
		t.Fatalf("err = %v, want cancelled", r.err)
	}
	if o := readOutcome(t, filepath.Join(runDir, "s")); o.Outcome != outcomeCancelled || o.Signal != "TERM" || o.Started == nil {
		t.Errorf("outcome = %+v, want cancelled by TERM after launch", o)
	}
}

// A step cancelled while it waits for a leaf-budget slot never launches.
func TestExecStep_CancelledWhileQueuedNeverLaunches(t *testing.T) {
	e, runDir := lifecycleExec(t), t.TempDir()
	budget := newPrioritySem(1)
	if err := budget.acquire(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	defer budget.release()
	e.budget = budget
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := startExec(ctx, e, experiment.Step{ID: "s", Uses: "test/launch-marker"}, runDir)
	waitForWaiters(t, budget, 1)
	cancel()
	r := <-done
	if r.err == nil || !errors.Is(r.err, context.Canceled) {
		t.Fatalf("err = %v, want a cancellation", r.err)
	}
	if _, err := os.Stat(filepath.Join(runDir, "launched")); err == nil {
		t.Fatal("the step launched after cancellation")
	}
	if o := readOutcome(t, filepath.Join(runDir, "s")); o.Outcome != outcomeCancelled || o.Started != nil {
		t.Errorf("outcome = %+v, want cancelled with no launch time", o)
	}
}
