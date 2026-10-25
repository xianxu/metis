package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
// through boundedLog. The ending goes through classifyOutcome and is recorded in outcome.json
// whatever it was; only an ok outcome returns nil. When this returns, the child is joined (or
// reported cleanup-unconfirmed), the timer is stopped, and the budget slot is released.
func (e execStep) runLegacy(ctx context.Context, step experiment.Step, exe, stepDir string, env []string) error {
	policy, err := experiment.ParseExecPolicy(step)
	if err != nil {
		return err // Validate refuses this before any step runs; kept honest here
	}
	end := stepEnding{logPath: filepath.Join(stepDir, stepLogFile)}
	record := func() error {
		o, cerr := classifyOutcome(end)
		if werr := writeOutcome(stepDir, o); werr != nil && cerr == nil {
			return werr
		}
		return cerr
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
		return err
	}
	defer logFile.Close()
	log := newBoundedLog(logFile, stepLogCap, stepLogTail)

	cmd := exec.Command(exe)
	cmd.Dir = stepDir
	cmd.Env = env
	cmd.Stdout, cmd.Stderr = log, log // one pointer: exec.Cmd serializes the Writes
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = e.joinBoundOr() // bounds the pipe drain after exit (an escaped holder)
	if err := cmd.Start(); err != nil {
		end.exitCode, end.event = -1, endExited
		_ = writeOutcome(stepDir, Outcome{Outcome: outcomeFailed})
		return fmt.Errorf("start: %w", err)
	}
	end.launched, end.started = true, time.Now()
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()

	var deadline <-chan time.Time
	if policy.Timeout > 0 {
		timer := time.NewTimer(policy.Timeout)
		defer timer.Stop()
		deadline = timer.C
	}
	var werr error
	joined := true
	select {
	case werr = <-waited:
		end.event = endExited
	case <-deadline:
		end.event = endDeadline
	case <-ctx.Done():
		end.event = endCancelled
	}
	if end.event != endExited {
		end.signal, werr, joined = e.terminate(cmd.Process.Pid, waited)
	}
	end.ended = time.Now()
	if joined && errors.Is(werr, exec.ErrWaitDelay) {
		joined = false
	}
	end.joined = joined
	end.exitCode = -1
	if cmd.ProcessState != nil {
		end.exitCode = cmd.ProcessState.ExitCode()
	}
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

// terminate ends the child's process group: TERM, then KILL after grace, then a bounded join.
// It returns the last signal sent, the Wait result, and whether the join was confirmed. Only the
// group metis created is signalled; a descendant that left it (setsid) is reported, not chased.
func (e execStep) terminate(pid int, waited <-chan error) (string, error, bool) {
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	select {
	case werr := <-waited:
		return "TERM", werr, true
	case <-time.After(e.graceOr()):
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	select {
	case werr := <-waited:
		return "KILL", werr, true
	case <-time.After(2 * e.joinBoundOr()): // past WaitDelay: the child itself would not die
		return "KILL", nil, false
	}
}

// readPending returns a non-empty pending-resources.json's contents (bounded), else nil.
func readPending(stepDir string) []byte {
	f, err := os.Open(filepath.Join(stepDir, pendingResources))
	if err != nil {
		return nil
	}
	defer f.Close()
	b, _ := io.ReadAll(io.LimitReader(f, pendingMax))
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
