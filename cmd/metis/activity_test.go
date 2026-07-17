package main

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
)

type activityFakeExec struct {
	result experiment.StepResult
	err    error
	calls  int
}

func (f *activityFakeExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	f.calls++
	return f.result, f.err
}

func TestActivityExecutorEmitsOneStepSuccessAfterSuccessfulInnerExecution(t *testing.T) {
	at := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	wantResult := experiment.StepResult{
		Metrics:   map[string]float64{"score": 0.91},
		Artifacts: []string{"train/model.bin"},
	}
	inner := &activityFakeExec{result: wantResult}
	var events []activityEvent

	got, err := activityExecutor{
		inner: inner,
		now:   func() time.Time { return at },
		emit:  func(ev activityEvent) { events = append(events, ev) },
	}.Execute(experiment.Step{ID: "train"}, "/tmp/run")

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !reflect.DeepEqual(got, wantResult) {
		t.Fatalf("Execute result = %+v; want %+v", got, wantResult)
	}
	if inner.calls != 1 {
		t.Fatalf("inner calls = %d; want 1", inner.calls)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d; want 1 (%+v)", len(events), events)
	}
	if events[0].Kind != activityStepSuccess || events[0].StepID != "train" || !events[0].At.Equal(at) {
		t.Fatalf("event = %+v; want one step-success event for train at %s", events[0], at.Format(time.RFC3339))
	}
}

func TestActivityExecutorEmitsNothingOnInnerErrorAndPreservesFailure(t *testing.T) {
	wantErr := errors.New("boom")
	inner := &activityFakeExec{err: wantErr}
	var events []activityEvent

	got, err := activityExecutor{
		inner: inner,
		now:   func() time.Time { return time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC) },
		emit:  func(ev activityEvent) { events = append(events, ev) },
	}.Execute(experiment.Step{ID: "train"}, "/tmp/run")

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute error = %v; want %v", err, wantErr)
	}
	if !reflect.DeepEqual(got, experiment.StepResult{}) {
		t.Fatalf("Execute result = %+v; want zero result from inner failure", got)
	}
	if inner.calls != 1 {
		t.Fatalf("inner calls = %d; want 1", inner.calls)
	}
	if len(events) != 0 {
		t.Fatalf("events = %+v; want none on error", events)
	}
}
