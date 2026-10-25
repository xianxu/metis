package main

import (
	"fmt"
	"time"
)

// Step outcomes recorded in <stepDir>/outcome.json (metis#72). Only outcomeOK is success;
// every other value is a step error, never retried and never published as success.
const (
	outcomeOK                 = "ok"
	outcomeFailed             = "failed"
	outcomeTimedOut           = "timed-out"
	outcomeCancelled          = "cancelled"
	outcomeCleanupUnconfirmed = "cleanup-unconfirmed"
	outcomeResourcesUnretired = "resources-unretired"
)

// endEvent is the first terminal event a step observed: the child exited on its own, its
// timeout fired, or the run was cancelled. The first one observed wins; a later exit (an
// exit-0 that lands during termination) never overrides a deadline or a cancel.
type endEvent int

const (
	endExited endEvent = iota
	endDeadline
	endCancelled
)

// stepEnding is everything the launch paths observed about one step's end. Pure data.
type stepEnding struct {
	event        endEvent
	launched     bool   // false: cancelled while waiting for a slot, or the launch itself failed
	launchErr    string // why a non-cancelled step never launched (log create, exec start)
	exitKnown    bool   // the child's exit was observed (a reap or a server response)
	exitCode     int    // valid when exitKnown (-1 when killed by a signal)
	signal       string // the last termination signal metis sent ("TERM" | "KILL" | "")
	joined       bool   // the child and its output pipe were confirmed finished
	pending      []byte // contents of a non-empty pending-resources.json (opaque, bounded)
	logTruncated bool
	started      time.Time
	ended        time.Time
	tail         []byte // the log tail, for the error message
	logPath      string
}

// Outcome is the additive outcome.json record. #74 adds wait_seconds beside runtime_seconds.
type Outcome struct {
	Outcome          string     `json:"outcome"`
	ExitCode         *int       `json:"exit_code,omitempty"`
	Signal           string     `json:"signal,omitempty"`
	Started          *time.Time `json:"started,omitempty"`
	Ended            *time.Time `json:"ended,omitempty"`
	RuntimeSeconds   *float64   `json:"runtime_seconds,omitempty"`
	LogTruncated     bool       `json:"log_truncated"`
	PendingResources string     `json:"pending_resources,omitempty"`
}

// classifyOutcome is the single place that decides a step's outcome (ARCH-ORDER: the explicit
// table), so neither launch path can publish success on its own. Precedence:
// cleanup-unconfirmed > resources-unretired > timed-out / cancelled / failed > ok.
func classifyOutcome(e stepEnding) (Outcome, error) {
	o := Outcome{Signal: e.signal, LogTruncated: e.logTruncated, PendingResources: string(e.pending)}
	// Times come only from observed events: a launch gives started; an observed exit gives
	// ended, runtime and the exit code (also after a timeout or cancel: a late exit 0 is
	// recorded, not obeyed). An unobserved exit leaves them out rather than guessing.
	if e.launched {
		started := e.started
		o.Started = &started
		if e.exitKnown {
			ended, code := e.ended, e.exitCode
			rt := ended.Sub(started).Seconds()
			o.Ended, o.RuntimeSeconds, o.ExitCode = &ended, &rt, &code
		}
	}
	tail := ""
	if len(e.tail) > 0 {
		tail = "\n" + string(e.tail)
	}
	switch {
	case e.launched && !e.joined:
		o.Outcome = outcomeCleanupUnconfirmed
		return o, fmt.Errorf("cleanup unconfirmed: the child or a descendant may still hold %s after KILL; find and stop it before rerunning%s", e.logPath, tail)
	case len(e.pending) > 0:
		o.Outcome = outcomeResourcesUnretired
		return o, fmt.Errorf("resources unretired: pending-resources.json still lists %s; retire them (metis does not) before rerunning", e.pending)
	case e.event == endDeadline:
		o.Outcome = outcomeTimedOut
		return o, fmt.Errorf("timed out (with.exec.timeout), terminated with %s%s", e.signal, tail)
	case e.event == endCancelled && !e.launched:
		o.Outcome = outcomeCancelled
		return o, fmt.Errorf("cancelled before launch")
	case e.event == endCancelled:
		o.Outcome = outcomeCancelled
		return o, fmt.Errorf("cancelled, terminated with %s%s", e.signal, tail)
	case !e.launched:
		o.Outcome = outcomeFailed
		return o, fmt.Errorf("start: %s", e.launchErr)
	case e.exitCode != 0:
		o.Outcome = outcomeFailed
		return o, fmt.Errorf("exit status %d%s", e.exitCode, tail)
	default:
		o.Outcome = outcomeOK
		return o, nil
	}
}
