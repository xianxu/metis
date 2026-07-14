package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
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
	const cap = 3
	ws := t.TempDir()
	// 3 configs → nested (outer folds = sweeper.cv.k = 2) × 2 inner folds → deep nesting, ~many leaf calls.
	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b, c]"))
	sem := make(chan struct{}, cap)
	var mu sync.Mutex
	var cur, peak int
	pe := peakExec{sem: sem, mu: &mu, cur: &cur, peak: &peak, in: foldFakeExec{}}
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
