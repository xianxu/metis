package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
)

func TestRunResolvedExperiment_AbortedBeforeSideEffects(t *testing.T) {
	ws := t.TempDir()
	control := newRunControl(2)
	control.fail("earlier fold", errors.New("failed"))
	var out bytes.Buffer
	exp := experiment.Experiment{Header: experiment.Header{Type: "experiment", ID: "queued"}}

	_, err := runResolvedExperiment(exp, runOpts{
		expPath:    filepath.Join(ws, "shape.md"),
		runControl: control,
		runLabel:   "queued fold",
		cache:      true,
	}, "queued", fixedNow(), &out)
	if !errors.Is(err, errRunAborted) {
		t.Fatalf("error = %v, want errRunAborted", err)
	}
	if out.Len() != 0 {
		t.Fatalf("aborted run wrote output: %q", out.String())
	}
	if _, statErr := os.Stat(filepath.Join(ws, "runs", "queued")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("queued run created state: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(ws, ".metis-cache")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("queued run initialized cache state: %v", statErr)
	}
}

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
	var acquired atomic.Int32
	var released atomic.Int32
	control.afterAcquire = func(string) { acquired.Add(1) }
	control.beforeRelease = func(string) { released.Add(1) }

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
	if got := acquired.Load(); got != 12 {
		t.Fatalf("acquire hook calls = %d, want 12 attempted runs", got)
	}
	if got := released.Load(); got != 12 {
		t.Fatalf("release hook calls = %d, want 12 attempted runs", got)
	}
}

func TestRunControlHookPanicsStillReleaseAdmission(t *testing.T) {
	panicValue := errors.New("observation hook panic")
	for _, tc := range []struct {
		name string
		set  func(*runControl)
	}{
		{
			name: "after acquire",
			set: func(control *runControl) {
				control.afterAcquire = func(string) { panic(panicValue) }
			},
		},
		{
			name: "before release",
			set: func(control *runControl) {
				control.beforeRelease = func(string) { panic(panicValue) }
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			control := &runControl{slots: make(chan struct{}, 1)}
			tc.set(control)
			got := recoverRunControlPanic(func() {
				_, _ = control.run("observed", func() (experiment.Run, error) {
					return experiment.Run{ID: "ok"}, nil
				})
			})
			if got != panicValue {
				t.Fatalf("recovered panic = %v, want exact hook panic %v", got, panicValue)
			}
			if got := len(control.slots); got != 0 {
				t.Fatalf("admission slots after recovered hook panic = %d, want 0", got)
			}
		})
	}
}

func recoverRunControlPanic(fn func()) (recovered any) {
	defer func() { recovered = recover() }()
	fn()
	return nil
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

func TestRunControlAcquiresAdmissionBeforeCheckingFailure(t *testing.T) {
	control := &runControl{slots: make(chan struct{}, 1)}
	control.slots <- struct{}{}
	prior := errors.New("prior failure")
	var callbackCalled atomic.Bool
	result := make(chan runControlResult, 1)

	control.mu.Lock()
	go func() {
		run, err := control.run("later", func() (experiment.Run, error) {
			callbackCalled.Store(true)
			return experiment.Run{ID: "must-not-run"}, nil
		})
		result <- runControlResult{run: run, err: err}
	}()

	// Make one admission slot available while firstError remains blocked on mu.
	// A correctly ordered run refills the slot before attempting the error check.
	<-control.slots
	timer := time.NewTimer(runControlTestTimeout)
	defer timer.Stop()
	for len(control.slots) != 1 {
		select {
		case <-timer.C:
			control.err = prior
			control.mu.Unlock()
			t.Fatal("run did not acquire admission before attempting the failure check")
		default:
			runtime.Gosched()
		}
	}
	control.err = prior
	control.mu.Unlock()

	got := awaitRunControl(t, result, "admitted run to observe prior failure")
	if !isZeroRun(got.run) || !errors.Is(got.err, errRunAborted) {
		t.Fatalf("run result = (%+v, %v), want zero Run and errRunAborted", got.run, got.err)
	}
	if callbackCalled.Load() {
		t.Fatal("callback executed despite failure installed before the post-admission check")
	}
	if got := len(control.slots); got != 0 {
		t.Fatalf("slots after aborted run = %d, want released", got)
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

func TestRunControlFailureWithoutLabelPreservesError(t *testing.T) {
	control := newRunControl(1)
	cause := errors.New("unlabeled failure")

	got := control.fail("", cause)
	if got != cause {
		t.Fatalf("unlabeled failure = %v (%p), want original error %v (%p)", got, got, cause, cause)
	}
	if stored := control.firstError(); stored != cause {
		t.Fatalf("stored unlabeled failure = %v (%p), want original error %v (%p)", stored, stored, cause, cause)
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

func TestRunControlWinnerHookRunsOnceInsideFailureMutex(t *testing.T) {
	control := newRunControl(2)
	callbacksEntered := make(chan struct{}, 2)
	releaseLeft := make(chan struct{})
	releaseRight := make(chan struct{})
	hookEntered := make(chan struct{}, 1)
	releaseWinner := make(chan struct{})
	results := make(chan runControlResult, 2)
	var hookCalls atomic.Int32
	control.beforeFailureUnlock = func() {
		hookCalls.Add(1)
		hookEntered <- struct{}{}
		<-releaseWinner
	}

	for _, failure := range []struct {
		label   string
		release <-chan struct{}
	}{{label: "left", release: releaseLeft}, {label: "right", release: releaseRight}} {
		failure := failure
		go func() {
			run, err := control.run(failure.label, func() (experiment.Run, error) {
				callbacksEntered <- struct{}{}
				<-failure.release
				return experiment.Run{}, errors.New("failed")
			})
			results <- runControlResult{run: run, err: err}
		}()
	}
	awaitRunControl(t, callbacksEntered, "first failing callback")
	awaitRunControl(t, callbacksEntered, "second failing callback")
	close(releaseLeft)
	awaitRunControl(t, hookEntered, "winner failure hook")

	hookHeldMutex := !control.mu.TryLock()
	if !hookHeldMutex {
		control.mu.Unlock()
	}
	lookupStarted := make(chan struct{})
	lookupResult := make(chan error, 1)
	go func() {
		close(lookupStarted)
		lookupResult <- control.firstError()
	}()
	awaitRunControl(t, lookupStarted, "firstError lookup to start")
	runtime.Gosched()
	lookupReturnedEarly := false
	var stored error
	select {
	case stored = <-lookupResult:
		lookupReturnedEarly = true
	default:
	}

	close(releaseRight)
	close(releaseWinner)
	first := awaitRunControl(t, results, "first concurrent failure result")
	second := awaitRunControl(t, results, "second concurrent failure result")
	if !lookupReturnedEarly {
		stored = awaitRunControl(t, lookupResult, "blocked firstError lookup")
	}

	if !hookHeldMutex {
		t.Fatal("winner hook ran outside the failure mutex")
	}
	if lookupReturnedEarly {
		t.Fatal("firstError returned while winner hook was blocked")
	}
	if got := hookCalls.Load(); got != 1 {
		t.Fatalf("winner hook calls = %d, want exactly 1", got)
	}
	if first.err == nil || second.err == nil || first.err.Error() != second.err.Error() {
		t.Fatalf("concurrent failures = (%v, %v), want one authoritative error", first.err, second.err)
	}
	if stored == nil || stored.Error() != first.err.Error() {
		t.Fatalf("stored first error = %v, want %v", stored, first.err)
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

func TestRunControlObservationLinearizesBeforeFailure(t *testing.T) {
	control := newRunControl(2)
	observationEntered := make(chan struct{})
	releaseObservation := make(chan struct{})
	failureReachedLock := make(chan struct{})
	failureReturned := make(chan error, 1)

	control.beforeFailureLock = func() { close(failureReachedLock) }
	observationReturned := make(chan bool, 1)
	go func() {
		observationReturned <- control.whileHealthy(func() {
			close(observationEntered)
			<-releaseObservation
		})
	}()
	awaitRunControl(t, observationEntered, "observation callback to hold the controller")

	go func() { failureReturned <- control.fail("fold", errors.New("boom")) }()
	awaitRunControl(t, failureReachedLock, "failure to reach the controller mutex")
	select {
	case err := <-failureReturned:
		t.Fatalf("failure returned while an earlier observation held the controller: %v", err)
	default:
	}

	close(releaseObservation)
	if ok := awaitRunControl(t, observationReturned, "observation to finish"); !ok {
		t.Fatal("observation admitted before failure was unexpectedly rejected")
	}
	if err := awaitRunControl(t, failureReturned, "failure to publish"); err == nil || err.Error() != "fold: boom" {
		t.Fatalf("failure = %v, want fold: boom", err)
	}

	called := false
	if ok := control.whileHealthy(func() { called = true }); ok {
		t.Fatal("observation after failure publication was admitted")
	}
	if called {
		t.Fatal("rejected observation callback ran")
	}
}

func TestRunControlActivityEmitterDropsLateStepAndRunEventsAfterFailure(t *testing.T) {
	control := newRunControl(2)
	var events []activityEvent
	emit := runControlActivityEmitter(control, func(ev activityEvent) {
		events = append(events, ev)
	})

	emit(activityEvent{Kind: activityStepSuccess, StepID: "prep"})
	if len(events) != 1 {
		t.Fatalf("healthy activity events = %d; want 1", len(events))
	}

	control.fail("first", errors.New("boom"))
	emit(activityEvent{Kind: activityStepSuccess, StepID: "late-step"})
	emit(activityEvent{Kind: activityRunSuccess, RunID: "late-run"})
	if len(events) != 1 {
		t.Fatalf("late activity after failure was published: %+v", events)
	}
}
