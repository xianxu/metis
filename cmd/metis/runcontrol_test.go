package main

import (
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
)

const runControlTestTimeout = 2 * time.Second

type runControlResult struct {
	run experiment.Run
	err error
}

func isZeroRun(run experiment.Run) bool {
	return reflect.DeepEqual(run, experiment.Run{})
}

func awaitRunControl[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(runControlTestTimeout):
		t.Fatalf("timed out waiting for %s", what)
		var zero T
		return zero
	}
}

func TestRunControlBoundsAdmissionAtTwiceParallelism(t *testing.T) {
	control := newRunControl(3)
	entered := make(chan struct{}, 12)
	release := make(chan struct{})
	results := make(chan runControlResult, 12)
	var active atomic.Int32
	var peak atomic.Int32

	for range 12 {
		go func() {
			run, err := control.run("point", func() (experiment.Run, error) {
				current := active.Add(1)
				for old := peak.Load(); current > old && !peak.CompareAndSwap(old, current); old = peak.Load() {
				}
				entered <- struct{}{}
				<-release
				active.Add(-1)
				return experiment.Run{ID: "ok"}, nil
			})
			results <- runControlResult{run: run, err: err}
		}()
	}

	for i := 0; i < 6; i++ {
		awaitRunControl(t, entered, "six admitted callbacks")
	}
	if got := len(control.slots); got != 6 {
		t.Fatalf("admitted slots = %d, want 6", got)
	}
	select {
	case <-entered:
		t.Fatal("more than six callbacks entered before an admission slot was released")
	default:
	}
	close(release)

	for i := 0; i < 12; i++ {
		got := awaitRunControl(t, results, "all bounded runs to drain")
		if got.err != nil || got.run.ID != "ok" {
			t.Fatalf("run result = (%+v, %v), want successful run", got.run, got.err)
		}
	}
	if got := peak.Load(); got != 6 {
		t.Fatalf("peak callbacks = %d, want exactly 6", got)
	}
}

func TestRunControlPublishesFailureBeforeAdmissionRelease(t *testing.T) {
	control := &runControl{slots: make(chan struct{}, 1)}
	published := make(chan struct{})
	letTokenRelease := make(chan struct{})
	control.beforeFailureUnlock = func() {
		close(published)
		<-letTokenRelease
	}

	firstResult := make(chan runControlResult, 1)
	go func() {
		run, err := control.run("first", func() (experiment.Run, error) {
			return experiment.Run{ID: "must-be-discarded"}, errors.New("boom")
		})
		firstResult <- runControlResult{run: run, err: err}
	}()
	awaitRunControl(t, published, "failure publication hook")

	var secondCalled atomic.Bool
	secondResult := make(chan runControlResult, 1)
	go func() {
		run, err := control.run("second", func() (experiment.Run, error) {
			secondCalled.Store(true)
			return experiment.Run{ID: "must-not-run"}, nil
		})
		secondResult <- runControlResult{run: run, err: err}
	}()

	if got := len(control.slots); got != 1 {
		t.Fatalf("slots while failure publisher holds the mutex = %d, want 1", got)
	}
	close(letTokenRelease)

	first := awaitRunControl(t, firstResult, "first failed run")
	second := awaitRunControl(t, secondResult, "second aborted run")
	if !isZeroRun(first.run) {
		t.Fatalf("failed run = %+v, want zero Run", first.run)
	}
	if first.err == nil || first.err.Error() != "first: boom" {
		t.Fatalf("first error = %v, want contextual first failure", first.err)
	}
	if !isZeroRun(second.run) || !errors.Is(second.err, errRunAborted) {
		t.Fatalf("second result = (%+v, %v), want zero Run and errRunAborted", second.run, second.err)
	}
	if secondCalled.Load() {
		t.Fatal("second callback executed after failure publication")
	}
	if got := control.firstError(); got == nil || got.Error() != "first: boom" {
		t.Fatalf("stored first error = %v, want first: boom", got)
	}
}

func TestRunControlSerialStillLatchesFailure(t *testing.T) {
	control := newRunControl(1)
	if control.slots != nil {
		t.Fatal("serial controller unexpectedly allocated admission slots")
	}
	if got := control.fail("ignored", nil); got != nil || control.firstError() != nil {
		t.Fatalf("nil failure = %v with stored error %v, want neither", got, control.firstError())
	}

	failed, err := control.run("serial", func() (experiment.Run, error) {
		return experiment.Run{ID: "must-be-discarded"}, errors.New("broken")
	})
	if !isZeroRun(failed) || err == nil || err.Error() != "serial: broken" {
		t.Fatalf("failed result = (%+v, %v), want zero Run and contextual error", failed, err)
	}

	called := false
	aborted, err := control.run("later", func() (experiment.Run, error) {
		called = true
		return experiment.Run{ID: "must-not-run"}, nil
	})
	if !isZeroRun(aborted) || !errors.Is(err, errRunAborted) {
		t.Fatalf("later result = (%+v, %v), want zero Run and errRunAborted", aborted, err)
	}
	if called {
		t.Fatal("later serial callback executed after failure")
	}
	if got := control.firstError(); got == nil || got.Error() != "serial: broken" {
		t.Fatalf("stored first error = %v, want serial: broken", got)
	}
}

func TestRunControlConcurrentFailuresKeepOneContextualCause(t *testing.T) {
	control := newRunControl(2)
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	results := make(chan runControlResult, 2)

	for _, tc := range []struct {
		label string
		err   string
	}{{label: "left", err: "left failed"}, {label: "right", err: "right failed"}} {
		tc := tc
		go func() {
			run, err := control.run(tc.label, func() (experiment.Run, error) {
				entered <- struct{}{}
				<-release
				return experiment.Run{ID: tc.label}, errors.New(tc.err)
			})
			results <- runControlResult{run: run, err: err}
		}()
	}
	awaitRunControl(t, entered, "left failure callback")
	awaitRunControl(t, entered, "right failure callback")
	close(release)

	first := awaitRunControl(t, results, "first concurrent failure")
	second := awaitRunControl(t, results, "second concurrent failure")
	if !isZeroRun(first.run) || !isZeroRun(second.run) {
		t.Fatalf("failed runs = (%+v, %+v), want zero Runs", first.run, second.run)
	}
	if first.err == nil || second.err == nil || first.err.Error() != second.err.Error() {
		t.Fatalf("concurrent errors = (%v, %v), want one authoritative error", first.err, second.err)
	}
	if got := first.err.Error(); got != "left: left failed" && got != "right: right failed" {
		t.Fatalf("authoritative error = %q, want one contextual cause", got)
	}
	if got := control.firstError(); got == nil || got.Error() != first.err.Error() {
		t.Fatalf("stored first error = %v, want %v", got, first.err)
	}
}

func TestRunControlDiscardsLateSuccessAfterSiblingFailure(t *testing.T) {
	control := newRunControl(2)
	successEntered := make(chan struct{})
	failurePublished := make(chan struct{})
	control.beforeFailureUnlock = func() { close(failurePublished) }

	successResult := make(chan runControlResult, 1)
	go func() {
		run, err := control.run("slow success", func() (experiment.Run, error) {
			close(successEntered)
			<-failurePublished
			return experiment.Run{ID: "late"}, nil
		})
		successResult <- runControlResult{run: run, err: err}
	}()
	awaitRunControl(t, successEntered, "successful callback to enter")

	failureResult := make(chan runControlResult, 1)
	go func() {
		run, err := control.run("sibling", func() (experiment.Run, error) {
			return experiment.Run{}, errors.New("failed")
		})
		failureResult <- runControlResult{run: run, err: err}
	}()

	failure := awaitRunControl(t, failureResult, "sibling failure")
	success := awaitRunControl(t, successResult, "late success")
	if failure.err == nil || failure.err.Error() != "sibling: failed" {
		t.Fatalf("failure error = %v, want sibling: failed", failure.err)
	}
	if !isZeroRun(success.run) || !errors.Is(success.err, errRunAborted) {
		t.Fatalf("late success = (%+v, %v), want discarded Run and errRunAborted", success.run, success.err)
	}
	if strings.Contains(success.err.Error(), "sibling") {
		t.Fatalf("late success exposed sibling cause instead of abort sentinel: %v", success.err)
	}
}
