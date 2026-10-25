package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
)

// metis#72 M2: the forked launch path's lifecycle against the REAL uv forkserver. Toy modules
// gate on a FIFO at $METIS_RUN_DIR/gate and mark readiness with a file, as the legacy fixtures do.

const toyGated = `import os
print("early", flush=True)
open("ready", "w").close()
open(os.path.join(os.environ["METIS_RUN_DIR"], "gate")).read()
print("late", flush=True)
`

const toyStubborn = `import os, signal, time
def seen(*_):
    open("term-seen", "w").close()
signal.signal(signal.SIGTERM, seen)
open("pid", "w").write(str(os.getpid()))
open("ready", "w").close()
while True:
    time.sleep(0.05)
`

// toyLateOK exits 0 on TERM: a successful result that lands after the deadline fired.
const toyLateOK = `import os, signal, sys
signal.signal(signal.SIGTERM, lambda *_: sys.exit(0))
open("ready", "w").close()
open(os.path.join(os.environ["METIS_RUN_DIR"], "gate")).read()
`

type forkFixture struct {
	e      execStep
	root   string
	runDir string
}

func newForkFixture(t *testing.T, modules map[string]string) forkFixture {
	t.Helper()
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not on PATH")
	}
	mods := t.TempDir()
	for name, body := range modules {
		if err := os.WriteFile(filepath.Join(mods, name+".py"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PYTHONPATH", mods)
	t.Setenv("METIS_FORKSERVER_PRELOAD", "")
	pool := newServerPool(io.Discard, nil)
	t.Cleanup(pool.shutdown)
	runDir := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(runDir, "gate"), 0o600); err != nil {
		t.Fatal(err)
	}
	return forkFixture{
		e:    execStep{expDir: t.TempDir(), seed: 1, out: io.Discard, pool: pool, grace: 300 * time.Millisecond, joinBound: 300 * time.Millisecond},
		root: repoRoot(t), runDir: runDir,
	}
}

// run drives one forked step through runForked, as Execute does after parsing a wrapper.
func (f forkFixture) run(t *testing.T, ctx context.Context, step experiment.Step, module string) error {
	t.Helper()
	stepDir := filepath.Join(f.runDir, step.ID)
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	env, err := f.e.stepEnv(step, stepDir, f.runDir)
	if err != nil {
		t.Fatal(err)
	}
	err, fellBack := f.e.runForked(ctx, step, wrapperSpec{root: f.root, module: module}, stepDir, env)
	if fellBack {
		t.Fatalf("fell back to legacy (err %v); a real server must serve", err)
	}
	return err
}

func (f forkFixture) start(t *testing.T, ctx context.Context, step experiment.Step, module string) <-chan error {
	done := make(chan error, 1)
	go func() { done <- f.run(t, ctx, step, module) }()
	return done
}

func TestForkedStep_OKWritesLogAndOutcome(t *testing.T) {
	f := newForkFixture(t, map[string]string{"toy_gated": toyGated})
	done := f.start(t, context.Background(), experiment.Step{ID: "s", Uses: "t/x"}, "toy_gated")
	stepDir := filepath.Join(f.runDir, "s")
	waitFor(t, filepath.Join(stepDir, "ready"))
	if b, _ := os.ReadFile(filepath.Join(stepDir, "step.log")); string(b) != "early\n" {
		t.Errorf("step.log while running = %q", b)
	}
	openGate(t, f.runDir)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	o := readOutcome(t, stepDir)
	if o.Outcome != outcomeOK || o.RuntimeSeconds == nil || *o.ExitCode != 0 {
		t.Errorf("outcome = %+v", o)
	}
}

func TestForkedStep_TimeoutTerminatesWithTERM(t *testing.T) {
	f := newForkFixture(t, map[string]string{"toy_gated": toyGated})
	err := f.run(t, context.Background(), withTimeout("s", "t/x", "500ms"), "toy_gated")
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err = %v, want a timeout", err)
	}
	if o := readOutcome(t, filepath.Join(f.runDir, "s")); o.Outcome != outcomeTimedOut || o.Signal != "TERM" {
		t.Errorf("outcome = %+v, want timed-out by TERM", o)
	}
}

func TestForkedStep_IgnoredTERMEscalatesToKILL(t *testing.T) {
	f := newForkFixture(t, map[string]string{"toy_stubborn": toyStubborn})
	err := f.run(t, context.Background(), withTimeout("s", "t/x", "500ms"), "toy_stubborn")
	if err == nil || !strings.Contains(err.Error(), "KILL") {
		t.Fatalf("err = %v, want a timeout ended by KILL", err)
	}
	if o := readOutcome(t, filepath.Join(f.runDir, "s")); o.Outcome != outcomeTimedOut || o.Signal != "KILL" {
		t.Errorf("outcome = %+v", o)
	}
}

// The first terminal event wins: an exit 0 caused by the timeout's TERM stays timed-out.
func TestForkedStep_LateExitZeroStaysTimedOut(t *testing.T) {
	f := newForkFixture(t, map[string]string{"toy_late_ok": toyLateOK})
	err := f.run(t, context.Background(), withTimeout("s", "t/x", "500ms"), "toy_late_ok")
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err = %v, want timed out despite the exit 0", err)
	}
	o := readOutcome(t, filepath.Join(f.runDir, "s"))
	if o.Outcome != outcomeTimedOut || o.ExitCode == nil || *o.ExitCode != 0 {
		t.Errorf("outcome = %+v, want timed-out with the late exit 0 recorded", o)
	}
}

// The server dies while a cancel is in progress: the child may still run in its own group, so
// the step is cleanup-unconfirmed, the root is broken, and nothing re-runs it on the legacy path.
func TestForkedStep_ServerDiesDuringCancel(t *testing.T) {
	f := newForkFixture(t, map[string]string{"toy_stubborn": toyStubborn})
	stepDir := filepath.Join(f.runDir, "s")
	t.Cleanup(func() {
		if b, err := os.ReadFile(filepath.Join(stepDir, "pid")); err == nil {
			if pid, err := strconv.Atoi(string(b)); err == nil {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	})
	done := f.start(t, context.Background(), withTimeout("s", "t/x", "500ms"), "toy_stubborn")
	waitFor(t, filepath.Join(stepDir, "term-seen")) // the timeout fired and the TERM was delivered
	f.e.pool.mu.Lock()
	srv := f.e.pool.servers[f.root]
	f.e.pool.mu.Unlock()
	if err := syscall.Kill(-srv.cmd.Process.Pid, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	err := <-done
	if err == nil || !strings.Contains(err.Error(), "cleanup unconfirmed") {
		t.Fatalf("err = %v, want cleanup unconfirmed", err)
	}
	if o := readOutcome(t, stepDir); o.Outcome != outcomeCleanupUnconfirmed {
		t.Errorf("outcome = %+v", o)
	}
	f.e.pool.mu.Lock()
	broken := f.e.pool.broken[f.root]
	f.e.pool.mu.Unlock()
	if !broken {
		t.Error("the root must be marked broken after its server died mid-cancel")
	}
}

// A run cancelled while the step waits for the server's ready line never dispatches it, and
// never falls back to the legacy path either.
func TestForkedStep_CancelDuringReadyWaitNeverLaunches(t *testing.T) {
	f := newForkFixture(t, map[string]string{
		"toy_slow_import": "import time\ntime.sleep(3)\n",
		"toy_marker":      "import os\nopen(os.path.join(os.environ['METIS_RUN_DIR'], 'launched'), 'w').close()\n",
	})
	t.Setenv("METIS_FORKSERVER_PRELOAD", "toy_slow_import") // the server takes ~3s to be ready
	ctx, cancel := context.WithCancel(context.Background())
	done := f.start(t, ctx, experiment.Step{ID: "s", Uses: "t/x"}, "toy_marker")
	time.Sleep(300 * time.Millisecond)
	cancel()
	err := <-done
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want a cancellation", err)
	}
	if o := readOutcome(t, filepath.Join(f.runDir, "s")); o.Outcome != outcomeCancelled || o.Started != nil {
		t.Errorf("outcome = %+v, want cancelled before launch", o)
	}
	if _, err := os.Stat(filepath.Join(f.runDir, "launched")); err == nil {
		t.Error("the step launched after cancellation")
	}
}

func TestForkedStep_CancelWhileRunning(t *testing.T) {
	f := newForkFixture(t, map[string]string{"toy_gated": toyGated})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := f.start(t, ctx, experiment.Step{ID: "s", Uses: "t/x"}, "toy_gated")
	waitFor(t, filepath.Join(f.runDir, "s", "ready"))
	cancel()
	err := <-done
	if err == nil || !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("err = %v, want cancelled", err)
	}
	if o := readOutcome(t, filepath.Join(f.runDir, "s")); o.Outcome != outcomeCancelled || o.Signal != "TERM" || o.Started == nil {
		t.Errorf("outcome = %+v, want cancelled by TERM after launch", o)
	}
}

// The server answers joined=false (a descendant outside the group holds the pipe): unconfirmed.
func TestForkedStep_UnjoinedResponseIsCleanupUnconfirmed(t *testing.T) {
	f := newForkFixture(t, map[string]string{"toy_escaper": `import os, subprocess, sys, time
subprocess.Popen([sys.executable, "-c",
    "import os, sys, time; os.setsid(); open(sys.argv[1], 'w').write(str(os.getpid())); time.sleep(30)",
    "orphan.pid"])
while not os.path.exists("orphan.pid") or not open("orphan.pid").read():
    time.sleep(0.01)
`})
	stepDir := filepath.Join(f.runDir, "s")
	t.Cleanup(func() {
		if b, err := os.ReadFile(filepath.Join(stepDir, "orphan.pid")); err == nil {
			if pid, err := strconv.Atoi(string(b)); err == nil {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	})
	err := f.run(t, context.Background(), experiment.Step{ID: "s", Uses: "t/x"}, "toy_escaper")
	if err == nil || !strings.Contains(err.Error(), "cleanup unconfirmed") {
		t.Fatalf("err = %v, want cleanup unconfirmed", err)
	}
	o := readOutcome(t, stepDir)
	if o.Outcome != outcomeCleanupUnconfirmed || o.ExitCode == nil || *o.ExitCode != 0 {
		t.Errorf("outcome = %+v, want cleanup-unconfirmed with the observed exit 0", o)
	}
}

func TestForkedStep_PendingResourcesRefuseSuccess(t *testing.T) {
	f := newForkFixture(t, map[string]string{"toy_resources": `import os
open(os.environ["METIS_PENDING_RESOURCES"], "w").write('["actor-1"]')
`})
	err := f.run(t, context.Background(), experiment.Step{ID: "s", Uses: "t/x"}, "toy_resources")
	if err == nil || !strings.Contains(err.Error(), "actor-1") {
		t.Fatalf("err = %v, want resources unretired naming actor-1", err)
	}
	if o := readOutcome(t, filepath.Join(f.runDir, "s")); o.Outcome != outcomeResourcesUnretired {
		t.Errorf("outcome = %+v", o)
	}
}

// The server refuses a request whose step.log it cannot open, before fork: not launched.
func TestForkedStep_ServerRefusalBeforeForkNeverLaunches(t *testing.T) {
	f := newForkFixture(t, map[string]string{"toy_marker": "import os\nopen(os.path.join(os.environ['METIS_RUN_DIR'], 'launched'), 'w').close()\n"})
	if err := os.MkdirAll(filepath.Join(f.runDir, "s", stepLogFile), 0o755); err != nil { // a directory
		t.Fatal(err)
	}
	err := f.run(t, context.Background(), experiment.Step{ID: "s", Uses: "t/x"}, "toy_marker")
	if err == nil || !strings.Contains(err.Error(), "start") {
		t.Fatalf("err = %v, want a refused start", err)
	}
	if o := readOutcome(t, filepath.Join(f.runDir, "s")); o.Outcome != outcomeFailed || o.Started != nil {
		t.Errorf("outcome = %+v, want failed, never launched", o)
	}
	if _, err := os.Stat(filepath.Join(f.runDir, "launched")); err == nil {
		t.Error("the step launched despite the refusal")
	}
}

func TestCheckForkProtocol(t *testing.T) {
	if err := checkForkProtocol(forkProtocol); err != nil {
		t.Errorf("current protocol refused: %v", err)
	}
	for _, v := range []int{0, 1} { // 0: a pre-metis#72 server sends no protocol at all
		if err := checkForkProtocol(v); !errors.Is(err, errServerUnavailable) {
			t.Errorf("protocol %d: %v, want errServerUnavailable (fall back before dispatch)", v, err)
		}
	}
}
