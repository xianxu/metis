package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
)

// TestSweep_ParallelEqualsSerial (metis#31 M3, cmd-level): the SAME sweep run
// serial (maxParallel=1) and parallel (maxParallel=8) via the fake exec must
// produce BYTE-IDENTICAL persisted artifacts — the ledger CSV + the manifest — not
// just an equal winner. Order-preserving ParExec + the order-independent reduce make
// the aggregate deterministic; the sortPointRuns step makes the on-disk bytes match
// too. Runs under -race, which also proves the sweepPass mutex covers every shared
// write the fan-out touches (configs/points/err).
func TestSweep_ParallelEqualsSerial(t *testing.T) {
	body := foldShapeMD("[a, b, c]") // 3 configs × 2 folds = 6 per-fold rows
	run := func(maxPar int) (ledger, manifest []byte) {
		ws := t.TempDir()
		expPath := writeShapeFile(t, ws, body)
		if _, err := runExperiment(runOpts{
			expPath:     expPath,
			now:         fixedNow(),
			git:         fakeGitProbe{name: "metis", sha: "sha"},
			cache:       true,
			exec:        foldFakeExec{},
			out:         io.Discard,
			maxParallel: maxPar,
		}); err != nil {
			t.Fatalf("maxParallel=%d: %v", maxPar, err)
		}
		lb, err := os.ReadFile(filepath.Join(ws, "shape.ledger.csv"))
		if err != nil {
			t.Fatalf("read ledger: %v", err)
		}
		matches, _ := filepath.Glob(filepath.Join(ws, "sweeps", "*", "manifest.json"))
		if len(matches) != 1 {
			t.Fatalf("want exactly 1 manifest, got %d", len(matches))
		}
		mb, err := os.ReadFile(matches[0])
		if err != nil {
			t.Fatalf("read manifest: %v", err)
		}
		return lb, mb
	}
	sL, sM := run(1)
	pL, pM := run(8)
	if !bytes.Equal(sL, pL) {
		t.Errorf("ledger bytes differ serial vs parallel:\n--serial--\n%s\n--parallel--\n%s", sL, pL)
	}
	if !bytes.Equal(sM, pM) {
		t.Errorf("manifest bytes differ serial vs parallel:\n--serial--\n%s\n--parallel--\n%s", sM, pM)
	}
}

// peakExec wraps foldFakeExec, acquiring the SHARED leaf semaphore around each step
// (mimicking the production execStep) and recording peak concurrency, so the peak-≤-n
// test can prove the global cap holds across driver×sweeper×resample nesting. It holds
// the slot briefly (a sleep) so the fan-out actually piles up against the cap. NOTE:
// this proves the BUDGET math via a fake leaf; the real execStep acquire is covered by
// TestExecStep_SemaphoreSerializesRealSubprocess.
type peakExec struct {
	sem  chan struct{}
	mu   *sync.Mutex
	cur  *int
	peak *int
	in   foldFakeExec
}

func (p peakExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	p.sem <- struct{}{}
	p.mu.Lock()
	*p.cur++
	if *p.cur > *p.peak {
		*p.peak = *p.cur
	}
	p.mu.Unlock()
	res, err := p.in.Execute(step, runDir)
	time.Sleep(time.Millisecond) // hold the slot so concurrency builds against the cap
	p.mu.Lock()
	*p.cur--
	p.mu.Unlock()
	<-p.sem
	return res, err
}

// TestNestedCV_PeakConcurrencyWithinCap (metis#31 done-when #4): a driver:cv run
// fans out to outerK×configs×innerK orchestration goroutines, but the ONE global leaf
// semaphore must keep concurrent leaf executions ≤ cap — no matter the nesting — and
// the run must complete (no deadlock). The fake leaf acquires the SAME sem the
// production execStep would.
func TestNestedCV_PeakConcurrencyWithinCap(t *testing.T) {
	const cap = 2 // runControl admits 2n=4 concrete runs; the nested fan-out has >4 children
	ws := t.TempDir()
	// 3 configs → nested (outer folds = sweeper.cv.k = 2) × 2 inner folds → deep nesting, ~many leaf calls.
	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b, c]"))
	sem := make(chan struct{}, cap)
	var mu sync.Mutex
	var cur, peak int
	pe := peakExec{sem: sem, mu: &mu, cur: &cur, peak: &peak, in: foldFakeExec{}}
	result := make(chan error, 1)
	go func() {
		_, err := runExperiment(runOpts{
			expPath:     expPath,
			now:         fixedNow(),
			git:         fakeGitProbe{name: "metis", sha: "sha"},
			cache:       false, // every step runs → maximum fan-out against the cap
			exec:        pe,
			out:         io.Discard,
			maxParallel: cap,
			leafSem:     sem, // runExperiment reuses my sem (maxParallel>1 & non-nil)
		})
		result <- err
	}()
	err := awaitRunControl(t, result, "nested run with more children than admission capacity")
	if err != nil {
		t.Fatalf("driver:cv run must complete (no deadlock), got: %v", err)
	}
	mu.Lock()
	got := peak
	mu.Unlock()
	if got > cap {
		t.Fatalf("peak concurrency %d exceeded the global cap %d — the leaf budget leaked across nesting", got, cap)
	}
	if got < 2 {
		t.Fatalf("peak concurrency %d — the fan-out never overlapped, so the test can't prove the cap actually holds", got)
	}
}

// sleepExec is foldFakeExec with a fixed per-step delay, so a sweep has real
// wall-clock cost — the wall-clock demo runs it serial vs parallel through the REAL
// runExperiment + sampler nesting (only the leaf is a sleeping fake, no subprocess).
type sleepExec struct {
	in foldFakeExec
	d  time.Duration
}

type concurrentBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *concurrentBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *concurrentBuffer) snapshot() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

func (b *concurrentBuffer) len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Len()
}

// failureBarrierExec holds the first four admitted inner folds at their train
// step. Exactly one returns the concrete injected failure; its admitted siblings
// wait for controller publication and then return the cancellation sentinel.
type failureBarrierExec struct {
	in               foldFakeExec
	mu               sync.Mutex
	innerDirs        map[string]struct{}
	innerEntered     chan string
	fourEntered      chan struct{}
	releaseFailure   chan struct{}
	failurePublished chan struct{}
	winner           atomic.Bool
}

func newFailureBarrierExec() *failureBarrierExec {
	return &failureBarrierExec{
		innerDirs:        make(map[string]struct{}),
		innerEntered:     make(chan string, 8),
		fourEntered:      make(chan struct{}),
		releaseFailure:   make(chan struct{}),
		failurePublished: make(chan struct{}),
	}
}

func (f *failureBarrierExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	if step.ID == partitionStepID {
		f.mu.Lock()
		if _, seen := f.innerDirs[runDir]; !seen {
			f.innerDirs[runDir] = struct{}{}
			f.innerEntered <- runDir
			if len(f.innerDirs) == 4 {
				close(f.fourEntered)
			}
		}
		f.mu.Unlock()
		<-f.fourEntered
	}
	if step.ID == "train" {
		if f.winner.CompareAndSwap(false, true) {
			<-f.releaseFailure
			return experiment.StepResult{}, errors.New("injected train failure")
		}
		<-f.failurePublished
		return experiment.StepResult{}, errRunAborted
	}
	return f.in.Execute(step, runDir)
}

func (f *failureBarrierExec) dirCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.innerDirs)
}

func TestNestedCV_FirstFailureStopsAllObservableWork(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b, c]"))
	control := newRunControl(2)
	exec := newFailureBarrierExec()
	out := &concurrentBuffer{}
	publishedOffset := make(chan int, 1)
	control.beforeFailureUnlock = func() {
		publishedOffset <- out.len()
		close(exec.failurePublished)
	}
	result := make(chan error, 1)
	go func() {
		_, err := runExperiment(runOpts{
			expPath: expPath, now: fixedNow(),
			git: fakeGitProbe{name: "metis", sha: "sha"}, exec: exec, out: out,
			maxParallel: 2, runControl: control,
		})
		result <- err
	}()

	for i := 0; i < 4; i++ {
		awaitRunControl(t, exec.innerEntered, "four admitted inner run directories")
	}
	close(exec.releaseFailure)
	offset := awaitRunControl(t, publishedOffset, "first failure publication")
	err := awaitRunControl(t, result, "nested failure to return without parent/child admission deadlock")
	if err == nil || !strings.Contains(err.Error(), "config ") || !strings.Contains(err.Error(), "injected train failure") {
		t.Fatalf("error = %v, want contextual concrete config/fold failure", err)
	}
	if errors.Is(err, errRunAborted) || strings.Contains(err.Error(), errRunAborted.Error()) {
		t.Fatalf("top-level error exposed cancellation sentinel instead of cause: %v", err)
	}
	if got := exec.dirCount(); got != 4 {
		t.Fatalf("inner run dirs = %d, want exactly four admitted dirs and no fifth start", got)
	}
	suffix := out.snapshot()[offset:]
	for _, forbidden := range []string{"metis: progress", "folds/min", "ETA", "score ", "estimate", "mean "} {
		if strings.Contains(suffix, forbidden) {
			t.Errorf("post-failure output contains %q:\n%s", forbidden, suffix)
		}
	}
	if matches, _ := filepath.Glob(filepath.Join(ws, "sweeps", "*", "manifest.json")); len(matches) != 0 {
		t.Fatalf("failure persisted %d manifest(s), want none", len(matches))
	}
	if _, statErr := os.Stat(filepath.Join(ws, "shape.ledger.csv")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failure persisted a ledger: %v", statErr)
	}
}

func (s sleepExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	res, err := s.in.Execute(step, runDir)
	time.Sleep(s.d)
	return res, err
}

// TestSweep_ParallelWallClockDemo (metis#31 done-when: "a wall-clock drop vs
// sequential is demonstrated") measures the real speedup through the whole
// orchestration with a 10ms-per-step leaf. Logs both durations; asserts parallel is
// meaningfully faster (loose bound — the fan-out is 3 configs × 2 folds).
func TestSweep_ParallelWallClockDemo(t *testing.T) {
	body := foldShapeMD("[a, b, c]") // 3 configs × 2 folds; each point runs 4 steps
	timeRun := func(maxPar int) time.Duration {
		ws := t.TempDir()
		expPath := writeShapeFile(t, ws, body)
		start := time.Now()
		if _, err := runExperiment(runOpts{
			expPath:     expPath,
			now:         fixedNow(),
			git:         fakeGitProbe{name: "metis", sha: "sha"},
			cache:       false, // every step sleeps → the fan-out has real cost
			exec:        sleepExec{in: foldFakeExec{}, d: 10 * time.Millisecond},
			out:         io.Discard,
			maxParallel: maxPar,
		}); err != nil {
			t.Fatalf("maxParallel=%d: %v", maxPar, err)
		}
		return time.Since(start)
	}
	serial := timeRun(1)
	par := timeRun(8)
	t.Logf("wall-clock: serial=%v  parallel(8)=%v  speedup=%.1fx", serial, par, float64(serial)/float64(par))
	if par >= serial {
		t.Errorf("parallel (%v) not faster than serial (%v)", par, serial)
	}
}

// flakyGitProbe returns a valid sha on the FIRST call (the sweep-start codeID freeze)
// and an error on every call after — simulating a per-fold `git status` that fails
// under .git/index.lock contention once the fan-out is writing runs/.
type flakyGitProbe struct {
	mu    sync.Mutex
	calls int
}

func (f *flakyGitProbe) Probe(string) (string, string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.calls == 1 {
		return "metis", "sha", false, nil // freezes codeID = "sha"
	}
	return "", "", false, fmt.Errorf("simulated .git/index.lock contention")
}

// TestSweep_ProbeFailureDoesNotFalseAbort (metis#31 C1): a transient per-fold probe
// failure (swallowed by probeRepo to sha="") must NOT be read as "code changed
// mid-sweep". The guard aborts only on a DEFINITE non-empty sha change. Against the
// pre-fix `s != ss.codeID` this fails ("" != "sha" → false abort of the whole run).
func TestSweep_ProbeFailureDoesNotFalseAbort(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeMD("[a, b]"))
	_, err := runExperiment(runOpts{
		expPath: expPath,
		now:     fixedNow(),
		git:     &flakyGitProbe{},
		cache:   false,
		exec:    foldFakeExec{},
		out:     io.Discard,
	})
	if err != nil {
		t.Fatalf("a transient probe failure must not abort the sweep, got: %v", err)
	}
}
