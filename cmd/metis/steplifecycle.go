package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
)

// Termination defaults (metis#72): TERM the child's process group, wait grace, KILL it, then
// confirm the join (exit plus the log pipe drained) within joinBound. Tests shrink both through
// the execStep fields.
const (
	defaultGrace     = 10 * time.Second
	defaultJoinBound = 5 * time.Second
	pendingMax       = 64 << 10 // pending-resources.json is shown verbatim, never interpreted
)

// Step-dir bookkeeping written by metis or by the step contract (metis#72): never artifacts,
// never in the output hash. (with.json, metrics.json, reads.json are the older members.)
const (
	stepLogFile      = "step.log"
	outcomeFile      = "outcome.json"
	pendingResources = "pending-resources.json"
)

func (e execStep) graceOr() time.Duration {
	if e.grace > 0 {
		return e.grace
	}
	return defaultGrace
}

func (e execStep) joinBoundOr() time.Duration {
	if e.joinBound > 0 {
		return e.joinBound
	}
	return defaultJoinBound
}

// runLegacy launches one step as its own process group and drives it to a single terminal
// event: its exit, its timeout, or the run's cancellation, whichever the select observes first
// (a later exit never overrides a deadline or a cancel). Output streams live into step.log
// through boundedLog over a pipe metis owns, so the join is evidence, not inference: confirmed
// only when the process exited and the pipe drained within joinBound (an escaped descendant
// still holding it means cleanup-unconfirmed, however the step ended). The ending goes through
// classifyOutcome and is recorded in outcome.json whatever it was; only ok returns nil. When
// this returns, the copier has stopped, the timer is stopped and the budget slot is released;
// the Wait goroutine outlives it only when the child survived KILL (reported, never awaited).
func (e execStep) runLegacy(ctx context.Context, step experiment.Step, exe, stepDir string, env []string) error {
	policy, err := experiment.ParseExecPolicy(step)
	if err != nil {
		return err // Validate refuses this before any step runs; kept honest here
	}
	end := stepEnding{event: endExited, logPath: filepath.Join(stepDir, stepLogFile)}
	record := func() error {
		o, cerr := classifyOutcome(end)
		if werr := writeOutcome(stepDir, o); werr != nil && cerr == nil {
			return werr
		}
		return cerr
	}
	notLaunched := func(err error) error {
		end.launchErr = err.Error()
		return record()
	}

	// metis#31: the budget is held only while a subprocess runs. metis#72: a cancelled run never
	// launches a queued step.
	if e.budget != nil {
		if err := e.budget.acquire(ctx, e.priority); err != nil {
			end.event = endCancelled
			return errors.Join(record(), err)
		}
		defer e.budget.release()
	}
	if err := ctx.Err(); err != nil {
		end.event = endCancelled
		return errors.Join(record(), err)
	}

	logFile, err := os.Create(end.logPath)
	if err != nil {
		return notLaunched(err)
	}
	defer logFile.Close()
	log := newBoundedLog(logFile, stepLogCap, stepLogTail)
	pr, pw, err := os.Pipe()
	if err != nil {
		return notLaunched(err)
	}
	defer pr.Close()

	cmd := exec.Command(exe)
	cmd.Dir = stepDir
	cmd.Env = env
	cmd.Stdout, cmd.Stderr = pw, pw // a file: the child writes the pipe directly, no exec copier
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err = cmd.Start()
	pw.Close() // the parent's write end; EOF arrives when every holder of it has gone
	if err != nil {
		return notLaunched(err)
	}
	end.launched, end.started = true, time.Now()
	drained := make(chan struct{})
	go func() { _, _ = io.Copy(log, pr); close(drained) }()
	exited := make(chan childExit, 1)
	go func() { err := cmd.Wait(); exited <- childExit{err, time.Now()} }()

	var deadline <-chan time.Time
	if policy.Timeout > 0 {
		timer := time.NewTimer(policy.Timeout)
		defer timer.Stop()
		deadline = timer.C
	}
	var x childExit
	procDone := true
	select {
	case x = <-exited:
	case <-deadline:
		end.event = endDeadline
	case <-ctx.Done():
		end.event = endCancelled
	}
	if end.event != endExited {
		end.signal, x, procDone = e.terminate(cmd.Process.Pid, exited)
	}
	end.ended = time.Now()
	if procDone {
		end.ended = x.at // runtime is launch to exit; termination is not compute
		end.exitKnown, end.exitCode = true, cmd.ProcessState.ExitCode()
		end.joined = e.awaitDrain(cmd.Process.Pid, drained, &end.signal)
	}
	pr.Close() // stop the copier if a holder remains; the log is read only after it stops
	<-drained
	end.pending = readPending(stepDir)
	end.logTruncated, end.tail = log.Truncated(), log.Tail()
	if err := log.Err(); err != nil {
		fmt.Fprintf(e.out, "! step %s: %v\n", step.ID, err)
	}
	cerr := record()
	if end.event == endCancelled && cerr != nil {
		cerr = errors.Join(cerr, ctx.Err())
	}
	return cerr
}

// awaitDrain confirms the join after the process exited: the pipe drains within joinBound, or
// metis stops what is left of the process group it created (TERM, grace, KILL) and waits the
// bound again. A holder that left the group (setsid) survives this, and the join stays
// unconfirmed. sig records any signal sent here.
func (e execStep) awaitDrain(pid int, drained <-chan struct{}, sig *string) bool {
	wait := func(d time.Duration) bool {
		select {
		case <-drained:
			return true
		case <-time.After(d):
			return false
		}
	}
	if wait(e.joinBoundOr()) {
		return true
	}
	if syscall.Kill(-pid, syscall.SIGTERM) == nil {
		*sig = "TERM"
		if wait(e.graceOr()) {
			return true
		}
		if syscall.Kill(-pid, syscall.SIGKILL) == nil {
			*sig = "KILL"
		}
	}
	return wait(e.joinBoundOr())
}

// childExit is the Wait result and when it returned (the exit time runtime is measured to).
type childExit struct {
	err error
	at  time.Time
}

// terminate ends the child's process group: TERM, then KILL after grace. It returns the last
// signal sent, the exit (when it came), and whether the process was confirmed exited. Only the
// group metis created is signalled; a descendant that left it (setsid) is reported, not chased.
func (e execStep) terminate(pid int, exited <-chan childExit) (string, childExit, bool) {
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	select {
	case x := <-exited:
		return "TERM", x, true
	case <-time.After(e.graceOr()):
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	select {
	case x := <-exited:
		return "KILL", x, true
	case <-time.After(e.joinBoundOr()): // the child itself would not die
		return "KILL", childExit{}, false
	}
}

// readPending returns a non-empty pending-resources.json's contents (bounded), else nil. A file
// that exists but cannot be read is not evidence that nothing is pending: it reports as pending,
// naming the error, so the step is never called a success while resources may be live.
func readPending(stepDir string) []byte {
	path := filepath.Join(stepDir, pendingResources)
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return []byte(fmt.Sprintf("<unreadable: %v>", err))
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, pendingMax))
	if err != nil {
		return []byte(fmt.Sprintf("<unreadable: %v>", err))
	}
	return bytes.TrimSpace(b)
}

// writeOutcome records the step's outcome.json (metis#72): written on every executed step.
func writeOutcome(stepDir string, o Outcome) error {
	b, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(stepDir, outcomeFile), append(b, '\n'), 0o644)
}

// runForked is runLegacy's counterpart for a convention-conforming wrapper routed through the
// warm forkserver (metis#44): the same budget, policy, classifier and outcome.json, with the
// child's lifecycle (process group, live log, post-exit join) owned by the server that reaps it.
// fellBack reports that nothing was dispatched and the caller should run the legacy path; once
// a request is dispatched it never falls back (a legacy re-run could double-execute).
func (e execStep) runForked(ctx context.Context, step experiment.Step, spec wrapperSpec, stepDir string, env map[string]string) (err error, fellBack bool) {
	policy, err := experiment.ParseExecPolicy(step)
	if err != nil {
		return err, false
	}
	end := stepEnding{event: endExited, logPath: filepath.Join(stepDir, stepLogFile)}
	record := func() error {
		o, cerr := classifyOutcome(end)
		if werr := writeOutcome(stepDir, o); werr != nil && cerr == nil {
			return werr
		}
		return cerr
	}
	if e.budget != nil {
		if err := e.budget.acquire(ctx, e.priority); err != nil {
			end.event = endCancelled
			return errors.Join(record(), err), false
		}
	}
	run, dispatched, ferr := e.pool.execute(ctx, spec, stepDir, env,
		forkOpts{timeout: policy.Timeout, grace: e.graceOr(), joinBound: e.joinBoundOr(), logPath: end.logPath})
	if e.budget != nil {
		e.budget.release() // before any legacy fallback acquires its own slot
	}
	switch {
	case errors.Is(ferr, errCancelledBeforeDispatch):
		end.event = endCancelled
		return errors.Join(record(), ctx.Err()), false
	case !dispatched:
		return nil, true
	}
	end.launched, end.started, end.event, end.signal = true, run.started, run.event, run.signal
	end.ended = time.Now()
	if run.got {
		end.ended = run.started.Add(time.Duration(run.resp.Runtime * float64(time.Second)))
		end.exitKnown, end.exitCode, end.joined = true, run.resp.Exit, run.resp.Joined
		end.logTruncated, end.tail = run.resp.LogTruncated, []byte(run.resp.Output)
		if _, err := os.Stat(end.logPath); errors.Is(err, fs.ErrNotExist) {
			end.logMissing = true
		}
		if run.event != endExited && run.resp.Exit == 0 {
			fmt.Fprintf(e.out, "· step %s: an exit-0 result arrived after the %s; dropped\n", step.ID, map[endEvent]string{endDeadline: "timeout", endCancelled: "cancel"}[run.event])
		}
	}
	// No response (none within the bound after KILL, or the server died with it in flight):
	// the child may still run in its own group, so the join is unconfirmed.
	end.pending = readPending(stepDir)
	cerr := record()
	if ferr != nil {
		cerr = errors.Join(cerr, ferr)
	}
	if end.event == endCancelled && cerr != nil {
		cerr = errors.Join(cerr, ctx.Err())
	}
	return cerr, false
}
