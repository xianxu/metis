package main

import (
	"time"

	"github.com/xianxu/metis/pkg/experiment"
)

type activityKind string

const (
	activityStepSuccess activityKind = "step-success"
	activityRunSuccess  activityKind = "run-success"
)

type runRole string

const (
	runRoleNone           runRole = ""
	runRoleNestedInnerCV  runRole = "nested-inner-cv"
	runRoleFlatCV         runRole = "flat-cv"
	runRoleNestedPreamble runRole = "nested-preamble"
	runRoleOuterScore     runRole = "outer-score"
)

type activityEvent struct {
	Kind   activityKind
	At     time.Time
	StepID string
	RunID  string
	Role   runRole
}

type activityEmitter func(activityEvent)

func (e activityEmitter) emit(ev activityEvent) {
	if e != nil {
		e(ev)
	}
}

func teeActivityEmitter(a, b activityEmitter) activityEmitter {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return func(ev activityEvent) {
		a.emit(ev)
		b.emit(ev)
	}
}

func runControlActivityEmitter(control *runControl, emit activityEmitter) activityEmitter {
	if control == nil {
		return emit
	}
	return func(ev activityEvent) {
		control.whileHealthy(func() { emit.emit(ev) })
	}
}

type activityExecutor struct {
	inner experiment.StepExecutor
	now   func() time.Time
	emit  activityEmitter
}

func (e activityExecutor) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	res, err := e.inner.Execute(step, runDir)
	if err != nil {
		return res, err
	}
	now := e.now
	if now == nil {
		now = time.Now
	}
	e.emit.emit(activityEvent{Kind: activityStepSuccess, At: now(), StepID: step.ID})
	return res, nil
}
