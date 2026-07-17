# Bounded Whole-Run Admission Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bound a parallel sweep to `2 × --parallel` admitted concrete runs so cold sweeps finish useful runs early, while globally aborting queued work after the first failure and preserving deterministic persisted results.

**Architecture:** Add one sweep-scoped `runControl`: its optional token channel bounds whole-run admission, while its mutex-protected error slot is the experiment-wide authoritative failure latch in serial and parallel modes. Wrap the existing `runResolvedExperiment` side-effect body at its single shared boundary, then make the nested sampler shell consult the same control before emitting progress, accumulating results, scoring, or persisting. Keep the existing leaf semaphore independent and unchanged.

**Tech Stack:** Go 1.x, generics-based `pkg/sampler`, channel/mutex coordination, injected `experiment.StepExecutor` fakes, real `uv`/Python process smoke, standard `go test -race`.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `errRunAborted` | `cmd/metis/runcontrol.go` | new |

- **`errRunAborted`** — immutable sentinel returned when a concrete run is skipped, or its otherwise-successful result becomes non-authoritative, because another run already published the experiment's first failure.
  - **Relationships:** N:1 from concrete run attempts to the sweep's one authoritative first error; callers use `errors.Is` to distinguish cancellation from the original failure.
  - **DRY rationale:** One typed signal prevents each sampler level from inventing a zero-value/error convention.
  - **Future extensions:** It can later wrap a cancellation reason without changing the admission contract.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `runControl` | `cmd/metis/runcontrol.go` | new | goroutine admission and sweep failure state |
| `runResolvedExperiment` admission wrapper | `cmd/metis/run.go` | modified | concrete-run filesystem, cache, executor, record, and capture effects |
| `shapeSweep` abort-aware reduction | `cmd/metis/sweep.go` | modified | nested sampler progress, accumulation, scoring, and persistence |
| `sweepProgress` / `boardWriter` abort path | `cmd/metis/progress.go`, `cmd/metis/board.go` | modified | ticker observations, pinned frame erasure, and deferred board close |

- **`runControl`** — one controller per shape run; its token channel is nil in serial mode and has capacity `2 × maxParallel` in parallel mode, while its first-error latch exists in both modes.
  - **Injected into:** copied `runOpts`, so inner folds, outer scoring, and the outer-analysis preamble share one controller without global state.
  - **Future extensions:** #49 may add a separate typed activity sink beside it; activity is not part of this controller.
- **`runResolvedExperiment` admission wrapper** — acquires before the first concrete-run side effect, checks cancellation, executes the existing body, publishes a contextual failure before token release, and rejects a late success if a sibling failed meanwhile.
  - **Injected into:** every current and future concrete-run call automatically through `runOpts`; individual callers supply only a contextual label.
  - **Future extensions:** a typed run role may replace the label when #49 adds completion telemetry.
- **`shapeSweep` abort-aware reduction** — preserves `sampler.Run`'s fixed output shape while ensuring its cancellation zero values never escape into progress, estimates, manifests, ledgers, or outer scoring.
  - **Injected into:** flat and nested sweep closures through the existing `shapeSweep` / `sweepPass` ownership graph.
  - **Future extensions:** adaptive samplers can consult the same authoritative stop state without changing the concrete-run gate.
- **`sweepProgress` / `boardWriter` abort path** — linearizes ticker repaints through the same
  failure latch, then erases and forgets the pinned frame on any post-wiring error so the top-level
  deferred close can flush ordinary output and restore the cursor without redrawing stale progress.
  - **Injected into:** `runShapeSweep`'s board wiring and named error return; plain output remains
    unchanged and retains no board-specific branch.
  - **Future extensions:** any new live display refresh must enter through the progress sink and the
    controller's healthy-observation operation.

## Chunk 1: Admission and failure primitive

### Task 1: Build `runControl` with race-safe handoff semantics

**Files:**
- Create: `cmd/metis/runcontrol.go`
- Create: `cmd/metis/runcontrol_test.go`

- [x] **Step 1: Write failing tests for capacity, first-error authority, and publication-before-release**

Define bounded helpers in the test file so a broken controller fails locally instead of waiting for
Go's global test timeout:

```go
type controlResult struct { run experiment.Run; err error }

func recvWithin[T any](t *testing.T, ch <-chan T) T {
    t.Helper()
    select {
    case v := <-ch:
        return v
    case <-time.After(2 * time.Second):
        t.Fatal("timed out waiting for run-control event")
        var zero T
        return zero
    }
}

func waitUntil(t *testing.T, pred func() bool) {
    t.Helper()
    deadline := time.Now().Add(2 * time.Second)
    for !pred() {
        if time.Now().After(deadline) { t.Fatal("timed out waiting for condition") }
        runtime.Gosched()
    }
}
```

Then create focused tests with no filesystem or executor mocks:

```go
func TestRunControlBoundsAdmission(t *testing.T) {
    const parallel = 3
    c := newRunControl(parallel)
    release := make(chan struct{})
    var mu sync.Mutex
    active, peak := 0, 0
    var wg sync.WaitGroup
    for i := 0; i < 12; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, _ = c.run("fold", func() (experiment.Run, error) {
                mu.Lock()
                active++
                if active > peak { peak = active }
                mu.Unlock()
                <-release
                mu.Lock(); active--; mu.Unlock()
                return experiment.Run{}, nil
            })
        }()
    }
    waitUntil(t, func() bool { mu.Lock(); defer mu.Unlock(); return peak == 2*parallel })
    close(release)
    done := make(chan struct{})
    go func() { wg.Wait(); close(done) }()
    recvWithin(t, done)
    if peak != 2*parallel { t.Fatalf("peak = %d, want %d", peak, 2*parallel) }
}

func TestRunControlPublishesFailureBeforeAdmissionRelease(t *testing.T) {
    c := &runControl{slots: make(chan struct{}, 1)}
    failingEntered := make(chan struct{})
    letFailureReturn := make(chan struct{})
    published := make(chan struct{})
    letTokenRelease := make(chan struct{})
    secondExecuted := make(chan struct{}, 1)
    c.beforeFailureUnlock = func() {
        close(published)
        <-letTokenRelease
    }

    firstDone := make(chan controlResult, 1)
    go func() {
        run, err := c.run("config a fold 0", func() (experiment.Run, error) {
            close(failingEntered)
            <-letFailureReturn
            return experiment.Run{ID: "must-not-escape"}, errors.New("train failed")
        })
        firstDone <- controlResult{run, err}
    }()
    recvWithin(t, failingEntered)

    secondDone := make(chan controlResult, 1)
    go func() {
        run, err := c.run("config b fold 0", func() (experiment.Run, error) {
            secondExecuted <- struct{}{}
            return experiment.Run{ID: "queued"}, nil
        })
        secondDone <- controlResult{run, err}
    }()
    close(letFailureReturn)

    recvWithin(t, published)
    if got := len(c.slots); got != 1 { t.Fatalf("slot released before publication hook: len=%d", got) }
    close(letTokenRelease)

    first := recvWithin(t, firstDone)
    second := recvWithin(t, secondDone)
    if got := first.err.Error(); !strings.Contains(got, "config a fold 0: train failed") {
        t.Fatalf("first error = %q", got)
    }
    if got := c.firstError(); got == nil || !strings.Contains(got.Error(), "config a fold 0") {
        t.Fatalf("stored failure after hook release = %v", got)
    }
    if !reflect.DeepEqual(first.run, experiment.Run{}) { t.Fatalf("failed run escaped: %+v", first.run) }
    if !errors.Is(second.err, errRunAborted) { t.Fatalf("second error = %v", second.err) }
    if !reflect.DeepEqual(second.run, experiment.Run{}) { t.Fatalf("aborted run escaped: %+v", second.run) }
    select {
    case <-secondExecuted: t.Fatal("queued run executed after failure")
    default:
    }
}
```

Add these complete companion cases:

```go
func TestRunControlSerialStillLatchesFailure(t *testing.T) {
    c := newRunControl(1)
    if c.slots != nil { t.Fatal("serial control unexpectedly has admission slots") }
    run, err := c.run("serial fold", func() (experiment.Run, error) {
        return experiment.Run{ID: "bad"}, errors.New("boom")
    })
    if !reflect.DeepEqual(run, experiment.Run{}) || err == nil || c.firstError() == nil {
        t.Fatalf("run=%+v err=%v first=%v", run, err, c.firstError())
    }
}

func TestRunControlConcurrentFailuresReturnFirstContext(t *testing.T) {
    c := &runControl{slots: make(chan struct{}, 2)}
    enteredA, enteredB := make(chan struct{}), make(chan struct{})
    releaseA, releaseB := make(chan struct{}), make(chan struct{})
    doneA, doneB := make(chan controlResult, 1), make(chan controlResult, 1)
    go func() {
        run, err := c.run("A", func() (experiment.Run, error) {
            close(enteredA); <-releaseA
            return experiment.Run{ID: "A"}, errors.New("failed A")
        })
        doneA <- controlResult{run, err}
    }()
    go func() {
        run, err := c.run("B", func() (experiment.Run, error) {
            close(enteredB); <-releaseB
            return experiment.Run{ID: "B"}, errors.New("failed B")
        })
        doneB <- controlResult{run, err}
    }()
    recvWithin(t, enteredA); recvWithin(t, enteredB)
    close(releaseA)
    a := recvWithin(t, doneA)
    close(releaseB)
    b := recvWithin(t, doneB)
    if !reflect.DeepEqual(a.run, experiment.Run{}) || !reflect.DeepEqual(b.run, experiment.Run{}) { t.Fatal("error result escaped") }
    if a.err.Error() != b.err.Error() || !strings.Contains(a.err.Error(), "A: failed A") {
        t.Fatalf("A=%v B=%v", a.err, b.err)
    }
}

func TestRunControlLateSuccessBecomesAborted(t *testing.T) {
    c := &runControl{slots: make(chan struct{}, 2)}
    successEntered, releaseSuccess := make(chan struct{}), make(chan struct{})
    successDone := make(chan controlResult, 1)
    go func() {
        run, err := c.run("slow success", func() (experiment.Run, error) {
            close(successEntered); <-releaseSuccess
            return experiment.Run{ID: "must-not-escape"}, nil
        })
        successDone <- controlResult{run, err}
    }()
    recvWithin(t, successEntered)
    _, failure := c.run("winner", func() (experiment.Run, error) {
        return experiment.Run{}, errors.New("first failure")
    })
    if failure == nil { t.Fatal("failure was not published") }
    close(releaseSuccess)
    got := recvWithin(t, successDone)
    if !reflect.DeepEqual(got.run, experiment.Run{}) || !errors.Is(got.err, errRunAborted) {
        t.Fatalf("late success = %+v, %v", got.run, got.err)
    }
}
```

The shared `controlResult` type and bounded helpers keep every blocking assertion finite.

- [x] **Step 2: Run the focused tests and verify RED**

Run: `go test ./cmd/metis -run '^TestRunControl' -count=1`

Expected: FAIL to compile because `runControl`, `newRunControl`, and `errRunAborted` do not exist.

- [x] **Step 3: Implement the minimal controller**

Create `cmd/metis/runcontrol.go` with this contract:

```go
var errRunAborted = errors.New("run aborted after earlier sweep failure")

type runControl struct {
    slots                 chan struct{}
    mu                    sync.Mutex
    err                   error
    beforeFailureLock     func() // deterministic lock-attempt seam; nil outside focused tests
    beforeFailureUnlock   func() // winner-only, after c.err store and before c.mu unlock; tests only
}

func newRunControl(maxParallel int) *runControl {
    c := &runControl{}
    if maxParallel > 1 {
        c.slots = make(chan struct{}, 2*maxParallel)
    }
    return c
}

func (c *runControl) firstError() error {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.err
}

func (c *runControl) fail(label string, err error) error {
    if err == nil { return nil }
    if label != "" { err = fmt.Errorf("%s: %w", label, err) }
    if c.beforeFailureLock != nil { c.beforeFailureLock() }
    c.mu.Lock()
    won := c.err == nil
    if won { c.err = err }
    first := c.err
    if won && c.beforeFailureUnlock != nil { c.beforeFailureUnlock() }
    c.mu.Unlock()
    return first
}

func (c *runControl) run(label string, fn func() (experiment.Run, error)) (experiment.Run, error) {
    if c.slots != nil {
        c.slots <- struct{}{}
        defer func() { <-c.slots }()
    }
    if c.firstError() != nil { return experiment.Run{}, errRunAborted }
    run, err := fn()
    if err != nil {
        first := c.fail(label, err)
        return experiment.Run{}, first
    }
    if c.firstError() != nil { return experiment.Run{}, errRunAborted }
    return run, nil
}
```

Keep `fail` set-once and return the stored error so concurrent failures converge on one authoritative
value. `beforeFailureLock` runs for every attempted failure (focused lock-order tests only);
`beforeFailureUnlock` runs only for the winning publisher after `c.err` is stored and while `c.mu`
is still held, so losing concurrent failures cannot fire it twice. It is a test-only ordering seam:
the callback must not call controller, progress, or board methods. Document `run` as
**non-reentrant**: its callback is exactly one concrete run and must never synchronously call `run`
on the same controller. Orchestration parents remain outside admission, so they never consume a
slot while waiting for child runs. Do not use `context.Context`: there is no interruptible
subprocess contract in scope, and admitted work is allowed to drain.

- [x] **Step 4: Run unit tests under the race detector and verify GREEN**

Run: `go test ./cmd/metis -run '^TestRunControl' -race -count=20`

Expected: PASS on all 20 repetitions with no race report or hang.

- [x] **Step 5: Commit the primitive**

```bash
git add cmd/metis/runcontrol.go cmd/metis/runcontrol_test.go
git commit -m "#43: add whole-run admission control" -m "Bound admitted concrete runs independently from leaf subprocesses and make first-failure publication atomic with admission release." -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

## Chunk 2: Concrete-run and sampler integration

### Task 2: Put the controller around the one concrete-run side-effect boundary

**Files:**
- Modify: `cmd/metis/run.go:62-96`
- Modify: `cmd/metis/run.go:172-182`
- Modify: `cmd/metis/run.go:202-290`
- Modify: `cmd/metis/sweep.go:440-570`
- Test: `cmd/metis/runcontrol_test.go`

- [x] **Step 1: Add a failing boundary test**

Install a controller that already holds an error, call `runResolvedExperiment` with a buffer and a temporary experiment directory, and assert no side effect precedes its cancellation check:

```go
func TestRunResolvedExperiment_AbortedBeforeSideEffects(t *testing.T) {
    ws := t.TempDir()
    c := newRunControl(2)
    c.fail("earlier fold", errors.New("failed"))
    var out bytes.Buffer
    exp := experiment.Experiment{Header: experiment.Header{Type: "experiment", ID: "queued"}}
    _, err := runResolvedExperiment(exp, runOpts{
        expPath: filepath.Join(ws, "shape.md"),
        runControl: c,
        runLabel: "queued fold",
    }, "queued", fixedNow(), &out)
    if !errors.Is(err, errRunAborted) { t.Fatalf("error = %v", err) }
    if out.Len() != 0 { t.Fatalf("aborted run wrote output: %q", out.String()) }
    if _, statErr := os.Stat(filepath.Join(ws, "runs", "queued")); !errors.Is(statErr, os.ErrNotExist) {
        t.Fatalf("queued run created state: %v", statErr)
    }
}
```

Run: `go test ./cmd/metis -run '^TestRunResolvedExperiment_AbortedBeforeSideEffects$' -count=1`

Expected: FAIL because `runOpts` and `runResolvedExperiment` do not consult the controller.

- [x] **Step 2: Thread control and contextual labels through `runOpts`**

Add:

```go
runControl *runControl // one per shape run: global abort + optional 2n admission slots
runLabel   string      // config/fold/preamble context captured with the first error
```

After strict shape validation and before `runShapeSweep`, assign `o.runControl = newRunControl(o.maxParallel)` only when it is nil. The non-nil path is an injected integration-test seam; production creates exactly one controller here. Leave plain one-point experiments unchanged: they have no fan-out and no controller.

- [x] **Step 3: Split the wrapper from the admitted body**

Keep the call-site name and move the current implementation byte-for-byte into `runResolvedExperimentAdmitted`:

```go
func runResolvedExperiment(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
    if o.runControl == nil {
        return runResolvedExperimentAdmitted(exp, o, runID, now, out)
    }
    return o.runControl.run(o.runLabel, func() (experiment.Run, error) {
        return runResolvedExperimentAdmitted(exp, o, runID, now, out)
    })
}
```

The wrapper executes before `filepath.Abs`, cache-directory creation, the run-start line, or executor construction. Its callback contains one concrete run only and never calls another admitted run, satisfying Chunk 1's non-reentrant contract.

- [x] **Step 4: Give every concrete sweep call a contextual label**

Set the label on each copied options value immediately before the call:

```go
pointOpts.runLabel = fmt.Sprintf("config %s fold %d (%s)", freeParamStr(c), f.Idx, runID)
scoreOpts.runLabel = fmt.Sprintf("outer fold %d family %s score (%s)", i, fam, scoreID)
preambleOpts.runLabel = fmt.Sprintf("outer-analysis preamble (%s)", preID)
```

Pass `fam` into `scoreOnOuterFold` so the label is complete at the boundary. Do not wrap a controller-returned error again at a caller; the stored first error is already contextual.

- [x] **Step 5: Run boundary and existing run tests**

Run: `go test ./cmd/metis -run 'TestRunResolvedExperiment|TestRunExperiment' -race -count=1`

Expected: PASS; the new abort test proves no output or directory creation before admission.

- [x] **Step 6: Commit the concrete-run boundary**

```bash
git add cmd/metis/run.go cmd/metis/runcontrol_test.go cmd/metis/sweep.go
git commit -m "#43: admit sweep runs at the concrete boundary" -m "Acquire before any run side effect and attach fold, score, or preamble context where the first failure becomes authoritative." -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

### Task 3: Make all sampler reductions observe the global failure latch

**Files:**
- Modify: `cmd/metis/runcontrol.go`
- Modify: `cmd/metis/sweep.go:74-174`
- Modify: `cmd/metis/sweep.go:275-335`
- Modify: `cmd/metis/sweep.go:366-440`
- Modify: `cmd/metis/sweep.go:470-630`
- Modify: `cmd/metis/progress.go`
- Modify: `cmd/metis/board.go`
- Test: `cmd/metis/board_test.go`
- Test: `cmd/metis/runcontrol_test.go`
- Test: `cmd/metis/parallel_test.go`

- [x] **Step 1: Write a failing nested-sibling cancellation test**

Add a `failureBarrierExec` for `foldShapeCVMD("[a, b, c]")` with `maxParallel=2` (admission capacity four). The fake records each distinct inner `runDir` on its first step. Once four inner runs have entered, it lets exactly one `train` return `errors.New("injected train failure")`; the other admitted trains wait on `failurePublished`, which the controller's winning-publication hook closes while admission is still held. Every wait is bounded, and the top-level `runExperiment` executes in a goroutine observed through `recvWithin`.

```go
type failureBarrierExec struct {
    in               foldFakeExec
    mu               sync.Mutex
    innerRuns        map[string]bool
    fourEntered      chan struct{}
    failurePublished <-chan struct{}
    failOnce         sync.Once
}

func (e *failureBarrierExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
    if step.ID == partitionStepID {
        e.mu.Lock()
        e.innerRuns[runDir] = true
        if len(e.innerRuns) == 4 {
            select { case <-e.fourEntered: default: close(e.fourEntered) }
        }
        e.mu.Unlock()
    }
    if step.ID == "train" {
        failed := false
        e.failOnce.Do(func() { failed = true })
        if failed {
            select {
            case <-e.fourEntered:
            case <-time.After(2 * time.Second):
                return experiment.StepResult{}, errors.New("timed out waiting for four admitted runs")
            }
            return experiment.StepResult{}, errors.New("injected train failure")
        }
        select {
        case <-e.failurePublished:
            return experiment.StepResult{}, errRunAborted
        case <-time.After(2 * time.Second):
            return experiment.StepResult{}, errors.New("timed out waiting for failure publication")
        }
    }
    return e.in.Execute(step, runDir)
}
```

Pre-create the controller, set its ordering hook to close `failurePublished`, and inject it through `runOpts` so `runExperiment` reuses it:

```go
control := newRunControl(2)
failurePublished := make(chan struct{})
control.beforeFailureUnlock = func() { close(failurePublished) }
exec := &failureBarrierExec{
    in: foldFakeExec{}, innerRuns: map[string]bool{},
    fourEntered: make(chan struct{}), failurePublished: failurePublished,
}
done := make(chan error, 1)
go func() {
    _, err := runExperiment(runOpts{
        expPath: expPath, now: fixedNow(), git: fakeGitProbe{name: "metis", sha: "sha"},
        exec: exec, out: &out, maxParallel: 2, runControl: control,
    })
    done <- err
}()
err := recvWithin(t, done)
```

Because the non-failing admitted trains wait until the controller's post-publication hook fires, none can release a token early and accidentally admit a fifth run. Assert:

- returned error contains the first concrete label plus `injected train failure`, not `errRunAborted`;
- exactly four inner run directories started and no fifth run-start line/directory appeared;
- no `metis: progress` line, fold/config/driver completion count, `folds/min`, ETA, score, or estimate appears after publication (the fake holds all eligible trains until the failure, so the strongest expected result is no such output anywhere);
- no sweep manifest or ledger exists.

Run: `go test ./cmd/metis -run '^TestNestedCV_FirstFailureStopsQueuedSiblingPasses$' -race -count=1`

Expected: FAIL because the current per-pass and outer error latches do not stop sibling passes.

- [x] **Step 2: Replace pass-local and outer-local error ownership with `runControl`**

Give `shapeSweep` helpers:

```go
func (ss *shapeSweep) fail(label string, err error) error {
    return ss.o.runControl.fail(label, err)
}
func (ss *shapeSweep) firstError() error { return ss.o.runControl.firstError() }
func (ss *shapeSweep) whileHealthy(fn func()) bool { return ss.o.runControl.whileHealthy(fn) }
```

Make `sweepPass.setErr` delegate to `ss.fail`; make `sweepPass.firstError` delegate to `ss.firstError`; remove `sweepPass.err`; retain its mutex only for `configs` and `points`. Delete `runNestedCV`'s separate `errMu` / `firstErr` closures and use the shape-wide helpers for orchestration errors.

- [x] **Step 3: Add an atomic healthy-observation operation**

Add the following controller method and a unit test that holds the observation callback open, starts `fail` concurrently, proves `fail` cannot return until the observation exits, then proves a later observation is rejected:

```go
// whileHealthy linearizes observable sweep state before or after first-failure publication.
// fn must not call runControl methods. Lock order is control.mu -> progress/pass/manifest mutex.
func (c *runControl) whileHealthy(fn func()) bool {
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.err != nil { return false }
    fn()
    return true
}
```

```go
func TestRunControlObservationLinearizesWithFailure(t *testing.T) {
    c := newRunControl(2)
    observationEntered := make(chan struct{})
    releaseObservation := make(chan struct{})
    failureAtLock := make(chan struct{})
    c.beforeFailureLock = func() { close(failureAtLock) }
    observed := make(chan bool, 1)
    go func() {
        observed <- c.whileHealthy(func() {
            close(observationEntered)
            <-releaseObservation
        })
    }()
    recvWithin(t, observationEntered)
    failed := make(chan error, 1)
    go func() { failed <- c.fail("fold", errors.New("boom")) }()
    recvWithin(t, failureAtLock) // fail has reached the statement immediately before c.mu.Lock
    select {
    case err := <-failed: t.Fatalf("failure published inside observation: %v", err)
    default:
    }
    close(releaseObservation)
    if !recvWithin(t, observed) { t.Fatal("pre-failure observation was rejected") }
    recvWithin(t, failed)
    if c.whileHealthy(func() { t.Fatal("post-failure observation executed") }) {
        t.Fatal("post-failure observation reported healthy")
    }
}
```

- [x] **Step 4: Gate sampler callbacks and accumulators without changing batch shape**

In `runSweeper`, wrap hooks explicitly:

```go
foldHook := func(ev sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]) {
    ss.whileHealthy(func() { pass.hooks.fold(ev) })
}
configHook := func(ev sampler.ProgressEvent[shape.Point, sampler.MeanSE]) {
    ss.whileHealthy(func() { pass.hooks.config(ev) })
}
```

Implement `addConfigScore`, `addPoint`, and `addManPoints` as `whileHealthy` callbacks that acquire their existing mutex only after `control.mu`. Never acquire a pass/progress/manifest mutex and then call the controller. This fixed order makes each append linearize entirely before failure or not occur at all. Zero values may still occupy cancelled batch indices, but no observable sink consumes them.

- [x] **Step 5: Gate every transition to further work or persistence**

Add first-error checks:

- after admission returns in `runPipelineFold`, treating `errRunAborted` as cancellation rather than a newly wrapped error;
- after each inner sweeper, before complexity guard and outer scoring;
- before each family outer-score run and before outer-row assembly;
- around driver progress callbacks: `ss.whileHealthy(func() { ss.prog.driverEvent(ev) })`;
- around each direct held-out-score line: `ss.whileHealthy(func() { fmt.Fprintf(ss.out, ...) })`;
- around every board ticker callback: `ss.whileHealthy(func() { ss.prog.tick() })`;
- immediately after flat or nested top-level `sampler.Run`, before `finish`, sorting, manifest/ledger/capture, estimate, or summary.

Make `runPipelineFold` distinguish the two controller outcomes explicitly:

```go
run, runErr := runResolvedExperiment(exp, pointOpts, runID, ss.now, ss.out)
if errors.Is(runErr, errRunAborted) { return sampler.FoldOutcome{} }
if runErr != nil {
    // The controller already stored the contextual authoritative error; do not wrap/publish it again.
    return sampler.FoldOutcome{}
}
if !p.addPoint(pointRun{/* successful run fields */}) { return sampler.FoldOutcome{} }
```

Change `addPoint` (and the other guarded accumulators) to return the boolean from `whileHealthy`. When an orchestration-only error occurs (point-address failure, code-freeze change, guard failure), publish it through `ss.fail` with its contextual label so queued concrete runs see it. The top-level flat/nested return must always use `ss.firstError()` so a cancellation sentinel never replaces the authoritative cause.

Give `runShapeSweep` a named error result and make the board ticker controllable without changing
production timing. Add nil-by-default `boardTick <-chan time.Time` and `afterBoardTick func()` test
seams to `runOpts`, plus `beforeBoardTick func()` immediately after the ticker goroutine selects a
tick; production creates and stops its existing 500ms ticker when `boardTick` is nil, while tests
inject a controlled channel (unbuffered for the publication handshake). Each received tick invokes `beforeBoardTick`, attempts the guarded
callback above, and then invokes `afterBoardTick`, allowing a test to prove a post-publication tick
was selected, blocked behind publication, and rejected without sleeps. The callbacks are
observation-only and must not call controller methods.

Add `(*boardWriter).discardFrame`: under `bw.mu`, set `frame = nil` and force the existing atomic
flush so the painted region is erased and complete pending ordinary lines are emitted without a
redraw. Add `(*sweepProgress).abort`: under `sp.mu`, call `bw.discardFrame`, preserving the sole
`sp.mu -> bw.mu` order. At board wiring install one deferred cleanup that stops/signals the ticker
and, when the named result is non-nil, calls `ss.prog.abort()` before returning to
`runExperiment`'s already-deferred `boardWriter.close`. The ticker goroutine closes a `tickStopped`
channel on exit; cleanup closes `tickDone`, joins `<-tickStopped`, and only then aborts the frame, so
no refresh can race behind cleanup or escape the function. Thus close may newline-flush a pending
tail and restore the cursor, but its nil stored frame cannot resurrect rate, ETA, counters, or scores.
Never call the controller while holding `sp.mu` or `bw.mu`; ticker gating acquires
`control.mu -> sp.mu -> bw.mu`, while error cleanup starts at `sp.mu` only after the result has been
chosen. This cleanup applies to every error return after board wiring, including non-sampler
orchestration failures.

- [x] **Step 6: Prove the TUI cannot repaint stale progress after publication**

First add a direct `boardWriter` unit test: paint a frame containing `folds/min` and `ETA`, call
`discardFrame`, then call `close`. Assert the suffix beginning at discard contains the erase and
cursor restoration sequences but neither board token; call close twice to retain idempotence proof.

Then run the nested failure barrier in board mode with a concurrency-safe recording writer,
pre-created controller, injected **unbuffered** `boardTick`, before/after tick hooks, and a
`releaseFailure` barrier in `failureBarrierExec`. Once four runs enter but before the failing train
returns, synchronously send one pre-failure tick and await `afterBoardTick`. Assert the recording
writer now contains a recognizable pinned frame with a progress/queued row and `folds/min`; only
then close `releaseFailure`.

In the controller's winner-only `beforeFailureUnlock` hook, snapshot and publish the writer offset,
then synchronously send a second tick on the unbuffered channel while failure still owns
`control.mu`. The ticker receives it and `beforeBoardTick` records selection, but its
`whileHealthy` call must wait for the hook to return and the mutex to unlock. Await the second
`afterBoardTick` before accepting the top-level result, proving the controlled post-publication tick
was rejected rather than painted. After `runExperiment` returns and its deferred board close has executed, assert
the output suffix from the publication offset contains no progress counters, `folds/min`, `ETA`,
score, estimate, or stored board row. ANSI erase/synchronized-output/cursor-restore sequences and
pending ordinary error output are permitted. Keep all waits bounded with `recvWithin`; run under
`-race` so writer snapshots, abort, tick rejection, and close are checked concurrently.

- [x] **Step 7: Prove orchestration never holds admission while awaiting a child**

Extend the nested test with a two-second top-level timeout and `maxParallel=2`. A nested shape has more than four children, so completion proves outer/config sampler closures remain outside `runControl.run`; if any parent acquired before synchronously awaiting a child, the test would exhaust all four slots and time out.

- [x] **Step 8: Run cancellation, board-abort, and nested tests repeatedly under race**

Run: `go test ./cmd/metis -run 'TestRunControlObservation|TestBoardWriter_DiscardFrame|TestNestedCV_FirstFailure|TestNestedCV_PeakConcurrency|TestSweep_ProbeFailure' -race -count=20`

Expected: PASS on all repetitions; no hang, race, manifest, ledger, post-failure plain or TUI
completion/rate output, deferred-close frame resurrection, or loss of the original contextual error.
This is Chunk 2's focused integration proof; Chunk 3 adds the cold early-completion regression,
serial/parallel artifact and run-record comparisons, the full `go test ./... -race`, and the
disposable real-process smoke before close.

- [x] **Step 9: Commit global abort integration**

```bash
git add cmd/metis/run.go cmd/metis/runcontrol.go cmd/metis/runcontrol_test.go cmd/metis/sweep.go cmd/metis/progress.go cmd/metis/board.go cmd/metis/board_test.go cmd/metis/parallel_test.go
git commit -m "#43: abort queued runs across nested sweep passes" -m "Use one experiment-wide failure latch and keep cancellation placeholders out of progress, reduction side records, TUI frames, scoring, and persistence." -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

## Chunk 3: Scheduling acceptance, documentation, and full proof

### Task 4: Pin early completion, nested caps, and deterministic records

**Files:**
- Modify: `cmd/metis/runcontrol.go`
- Modify: `cmd/metis/runcontrol_test.go`
- Modify: `cmd/metis/parallel_test.go:16-175`

- [x] **Step 1: Add acquire/release observation seams for integration tests**

Add two nil-by-default hooks to `runControl`:

```go
afterAcquire  func(label string) // called with a token held, before cancellation check
beforeRelease func(label string) // called with a token held, immediately before receive/release
```

Invoke them only on the channel-backed parallel path:

```go
if c.slots != nil {
    c.slots <- struct{}{}
    if c.afterAcquire != nil { c.afterAcquire(label) }
    defer func() {
        if c.beforeRelease != nil { c.beforeRelease(label) }
        <-c.slots
    }()
}
```

These are observation-only test seams: callbacks must not call controller methods or block production. Extend `TestRunControlBoundsAdmission` to count hook acquisitions/releases and assert both equal the attempted run count.

- [x] **Step 2: Write the deterministic cold-wave regression**

Add a shared trace recorder and executor:

```go
type scheduleTrace struct {
    mu     sync.Mutex
    events []string
}
func (s *scheduleTrace) add(event string) {
    s.mu.Lock(); defer s.mu.Unlock()
    s.events = append(s.events, event)
}
func (s *scheduleTrace) snapshot() []string {
    s.mu.Lock(); defer s.mu.Unlock()
    return append([]string(nil), s.events...)
}

type scheduleTraceExec struct {
    in    foldFakeExec
    trace *scheduleTrace
}
func (e scheduleTraceExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
    result, err := e.in.Execute(step, runDir)
    if err == nil && step.ID == "train" { e.trace.add("train-complete:" + runDir) }
    return result, err
}
```

Run a genuinely flat six-run shape (one config, `k=6`) with a controller of capacity four:

```go
func TestSweep_ColdAdmissionCompletesTrainBeforeFifthRunStarts(t *testing.T) {
    ws := t.TempDir()
    body := strings.Replace(foldShapeMD("[a]"), "resample: {cv: {k: 2", "resample: {cv: {k: 6", 1)
    expPath := writeShapeFile(t, ws, body)
    trace := &scheduleTrace{}
    control := newRunControl(2)
    control.afterAcquire = func(label string) { trace.add("acquire:" + label) }
    _, err := runExperiment(runOpts{
        expPath: expPath, now: fixedNow(), git: fakeGitProbe{name: "metis", sha: "sha"},
        cache: false, exec: scheduleTraceExec{in: foldFakeExec{}, trace: trace},
        out: io.Discard, maxParallel: 2, runControl: control,
    })
    if err != nil { t.Fatal(err) }
    events := trace.snapshot()
    firstTrain, fifthAcquire, acquisitions := -1, -1, 0
    for i, event := range events {
        if strings.HasPrefix(event, "train-complete:") && firstTrain < 0 { firstTrain = i }
        if strings.HasPrefix(event, "acquire:") {
            acquisitions++
            if acquisitions == 5 { fifthAcquire = i }
        }
    }
    if firstTrain < 0 || fifthAcquire < 0 || firstTrain > fifthAcquire {
        t.Fatalf("events = %v; want a completed train before run 5 acquires", events)
    }
}
```

Because the fifth acquire hook cannot fire until a token is released, and token release occurs only after the admitted concrete run returns, this test deterministically pins the desired ordering without sleeps.

- [x] **Step 3: Extend the nested test to inspect both concurrency budgets**

Inject a controller into `TestNestedCV_PeakConcurrencyWithinCap`. Its acquire/release hooks update `activeRuns`, `peakRuns`, `acquiredRuns`, and `releasedRuns` under the test mutex while the existing `peakExec` tracks leaves. Run `runExperiment` in a goroutine and use `recvWithin` for a two-second deadlock bound. Assert:

```go
if peakRuns > 2*cap { t.Fatalf("peak admitted runs %d > %d", peakRuns, 2*cap) }
if peakLeaves > cap { t.Fatalf("peak leaves %d > %d", peakLeaves, cap) }
if activeRuns != 0 { t.Fatalf("admission leak: %d runs still active", activeRuns) }
if acquiredRuns != releasedRuns { t.Fatalf("acquired=%d released=%d", acquiredRuns, releasedRuns) }
```

Keep the unit controller test's exact `peak == 2n` assertion; the nested integration assertion is an upper bound because fast fake runs need not saturate all slots.

- [x] **Step 4: Strengthen serial/parallel determinism to include run semantics**

Extend `TestSweep_ParallelEqualsSerial` so its helper loads every `runs/*/run.json` and `runs/*/record.json` into maps keyed by run ID. Normalize only `Started` and `Finished`, which are timing-bearing:

```go
func semanticRuns(t *testing.T, ws string) map[string]experiment.Run {
    t.Helper()
    paths, err := filepath.Glob(filepath.Join(ws, "runs", "*", "run.json"))
    if err != nil { t.Fatal(err) }
    got := make(map[string]experiment.Run, len(paths))
    for _, path := range paths {
        b, err := os.ReadFile(path); if err != nil { t.Fatal(err) }
        var run experiment.Run
        if err := json.Unmarshal(b, &run); err != nil { t.Fatal(err) }
        run.Started, run.Finished = "", ""
        got[run.ID] = run
    }
    return got
}

func semanticRecords(t *testing.T, ws string) map[string]record.RunRecord {
    t.Helper()
    paths, err := filepath.Glob(filepath.Join(ws, "runs", "*", "record.json"))
    if err != nil { t.Fatal(err) }
    got := make(map[string]record.RunRecord, len(paths))
    for _, path := range paths {
        b, err := os.ReadFile(path); if err != nil { t.Fatal(err) }
        var rec record.RunRecord
        if err := json.Unmarshal(b, &rec); err != nil { t.Fatal(err) }
        rec.Started, rec.Finished = "", ""
        got[rec.RunID] = rec
    }
    return got
}
```

Return both maps beside ledger/manifest bytes and assert `reflect.DeepEqual` for serial versus parallel runs and records. Keep the existing byte comparisons unchanged.

- [x] **Step 5: Run the complete scheduling acceptance subset**

Run: `go test ./cmd/metis -run 'TestRunControl|TestSweep_ColdAdmission|TestSweep_ParallelEqualsSerial|TestNestedCV_PeakConcurrency|TestNestedCV_FirstFailure' -race -count=10`

Expected: PASS ten times with no race, timeout, admission leak, post-failure output, or deterministic-output difference.

- [x] **Step 6: Commit the scheduling regressions**

```bash
git add cmd/metis/runcontrol.go cmd/metis/runcontrol_test.go cmd/metis/parallel_test.go
git commit -m "#43: pin bounded depth-first sweep scheduling" -m "Prove early train completion, independent run and leaf caps, nested liveness, cancellation isolation, and deterministic persisted output." -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

### Task 5: Add a disposable real-process smoke and update the atlas

**Files:**
- Create: `testdata/experiment/toy-sweep-smoke.md`
- Modify: `atlas/index.md:84-113`
- Modify: `workshop/issues/000043-leaf-sched-depth-first.md`

- [x] **Step 1: Add the smallest real subprocess sweep fixture**

Create a three-config, two-fold shape. `test/echo` is a process-level data-phase adapter whose `out` points at the copied toy dataset; the pipeline itself uses the real `metis/train` Python step:

```yaml
---
type: experiment-shape
id: toy-sweep-smoke
seed: 42
status: active
data:
  - id: data
    uses: test/echo
    with: {out: ../dataset/toy}
pipeline:
  - id: train
    uses: metis/train
    needs: [data]
    with:
      dataset: ../dataset/toy
      model:
        $any:
          logreg: {C: {$any: [0.5, 1.0, 2.0]}}
sweeper:
  sampler: grid
  resample: {cv: {k: 2, stratify: true}}
  objective: {metric: train.fold_score, direction: maximize, select: {argmax-mean: {}}}
---

# toy-sweep-smoke

A disposable real-process nested sweep used to verify whole-run admission without Kaggle credentials.
```

Run `go run ./cmd/metis run --dry-run testdata/experiment/toy-sweep-smoke.md`; expected output reports three configs and nested-CV mode.

- [x] **Step 2: Document both concurrency budgets in the atlas**

In the `metis#31` execution paragraph, add: parallel sampler fan-out remains order-preserving, but every concrete run crosses one sweep-scoped `2n` admission controller before side effects; the existing leaf semaphore remains `n`; the controller also owns the experiment-wide first failure so queued runs stop without producing observable state. Cite `cmd/metis/runcontrol.go`, `runResolvedExperiment`, and the cancellation regressions.

- [x] **Step 3: Run the disposable cold real-process smoke**

Run entirely inside a temporary no-hardlink clone so code capture can update only the clone's Metis refs. Snapshot the source checkout and refs, redirect writable build/runtime caches, and clean up on both success and failure:

```bash
(
set -euo pipefail
source_repo=$(pwd -P)
source_status=$(git status --porcelain=v1 --untracked-files=all)
source_refs=$(git for-each-ref --format='%(refname) %(objectname)' refs/metis)
ariadne_repo=$(cd "$source_repo/../ariadne" && pwd -P)
ariadne_sha=$(git -C "$ariadne_repo" rev-parse HEAD)
tmpdir=$(mktemp -d /tmp/metis-43-smoke.XXXXXX)
smoke_log="$tmpdir/smoke.log"
cleanup() {
  rc=$?
  if [ "$rc" -ne 0 ] && [ -f "$smoke_log" ]; then
    tail -120 "$smoke_log"
  fi
  chmod -R u+w "$tmpdir" 2>/dev/null || true
  rm -rf "$tmpdir"
  exit "$rc"
}
trap cleanup EXIT
git clone --local --no-hardlinks "$ariadne_repo" "$tmpdir/ariadne"
git -C "$tmpdir/ariadne" checkout --detach "$ariadne_sha"
test "$ariadne_sha" = "$(git -C "$tmpdir/ariadne" rev-parse HEAD)"
git clone --local --no-hardlinks "$source_repo" "$tmpdir/metis-src"
cd "$tmpdir/metis-src"
export GOCACHE="$tmpdir/go-cache"
export GOMODCACHE="$tmpdir/go-mod-cache"
export GOTELEMETRYDIR="$tmpdir/go-telemetry"
export UV_CACHE_DIR="$tmpdir/uv-cache"
export XDG_CACHE_HOME="$tmpdir/xdg-cache"
export PYTHONPYCACHEPREFIX="$tmpdir/pycache"
go build -o "$tmpdir/metis" ./cmd/metis
started=$(date +%s)
METIS_STEP_PATH="$PWD/testdata/steps:$PWD/steps" "$tmpdir/metis" run --fast --parallel 2 --forkserver=false --cache=false testdata/experiment/toy-sweep-smoke.md >"$tmpdir/smoke.log" 2>&1
elapsed=$(($(date +%s) - started))
if [ "$elapsed" -lt 1 ]; then elapsed=1; fi
rg -n 'metis: run|→ step cv-split|✓ step train|nested-CV estimate|rows →' "$tmpdir/smoke.log"
awk '
  /✓ step train/ && !first_train { first_train=NR }
  /→ step cv-split/ { inner_starts++; if (inner_starts==5) fifth_inner=NR }
  END {
    if (!first_train || !fifth_inner || first_train >= fifth_inner) {
      print "cold ordering failed: first_train=" first_train ", fifth_inner=" fifth_inner > "/dev/stderr"
      exit 1
    }
    printf "cold ordering passed: first_train_line=%d fifth_cv_split_line=%d\n", first_train, fifth_inner
  }
' "$tmpdir/smoke.log"
completed_trains=$(rg -c '✓ step train' "$tmpdir/smoke.log")
awk -v completed="$completed_trains" -v seconds="$elapsed" 'BEGIN {
  rate=60*completed/seconds
  if (completed < 7 || seconds > 600 || rate <= 0 || rate != rate) {
    print "throughput check failed: completed=" completed ", seconds=" seconds ", trains/min=" rate > "/dev/stderr"
    exit 1
  }
  printf "smoke throughput: %d trains in %ds (~%.1f trains/min)\n", completed, seconds, rate
}'
test "$source_status" = "$(git -C "$source_repo" status --porcelain=v1 --untracked-files=all)"
test "$source_refs" = "$(git -C "$source_repo" for-each-ref --format='%(refname) %(objectname)' refs/metis)"
cd "$source_repo"
chmod -R u+w "$tmpdir" 2>/dev/null || true
rm -rf "$tmpdir"
trap - EXIT
)
```

Expected: exit 0; the first `awk` proves the first completed train precedes the fifth inner run's first `cv-split` step, followed by the nested estimate and row summary. The second requires all six inner trains plus the one outer-score train, completion within ten minutes, and a finite positive measured trains/min rate. The source worktree and `refs/metis/*` snapshots are byte-identical before/after; clone refs, experiment outputs, Go build/module/telemetry caches, uv cache, XDG cache, and Python bytecode all live under the removed temporary directory. Record the relevant ordering/result lines in the issue Log before cleanup.

- [x] **Step 4: Run the complete automated verification**

Run:

```bash
gofmt -w cmd/metis/runcontrol.go cmd/metis/runcontrol_test.go cmd/metis/run.go cmd/metis/sweep.go cmd/metis/parallel_test.go
go test ./cmd/metis -race -count=1
go test ./... -race -count=1
git diff --check
```

Expected: every command exits 0; no race report; no whitespace errors.

- [x] **Step 5: Update the issue record with evidence**

Tick the plain issue-plan checkboxes only after their commands pass. Append a dated Log entry naming the focused race repetitions, full race suite, serial/parallel byte comparison, semantic run-record comparison, and temporary real-process smoke result. Do not add an `M1` marker: this issue has one close-review boundary.

- [x] **Step 6: Commit docs and final verification artifacts**

```bash
git add testdata/experiment/toy-sweep-smoke.md atlas/index.md workshop/issues/000043-leaf-sched-depth-first.md
git commit -m "#43: document bounded sweep admission" -m "Record the independent run and leaf budgets and keep a credential-free real-process sweep for cold scheduling smoke checks." -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

- [x] **Step 7: Enter the single SDLC close boundary**

Run `sdlc close --issue 43 --agent codex --verified '<focused race repetitions; full go test ./... -race; byte/semantic determinism; disposable real-process smoke>'` and address every Critical/Important finding from the gate-owned fresh review before proceeding to PR/merge.

## Revisions

### 2026-07-16 — implementation-entry TUI failure-path review

- Added the live board as an explicit global-abort consumer after the `change-code` judge identified
  ticker repaint and deferred close as paths that could resurrect stale progress after failure.
- Defined the fixed `control.mu -> progress.mu -> board.mu` ticker order, an error-only frame discard
  before top-level close, and controlled-tick publication-barrier tests for both direct compositor
  behavior and the nested failure integration.
- Moved the winner-only controller seam between the first-error store and mutex unlock, and made the
  integration paint and assert a real pre-failure frame before driving an unbuffered post-store tick.
- Kept the under-lock controller test non-reentrant and limited the integration's pre-failure rate
  assertion to available samples; the direct compositor test retains the explicit stale-ETA proof.
- Expanded Chunk 2's files, focused race command, and commit boundary to cover `run.go`,
  `progress.go`, `board.go`, and `board_test.go`.

### 2026-07-16 — cold-smoke peer pin correction

- Made the disposable smoke resolve the declared `../ariadne` replacement from the source checkout,
  snapshot its exact HEAD, and no-hardlink clone Ariadne first into the sibling path expected by the
  cloned Metis module. The recipe checks out and verifies that detached peer commit before building.
- Removed the uncommitted-fixture copy assumption now that the fixture is durable, and made cleanup
  writable-cache-safe while preserving source status/ref, ordering, throughput, and elapsed-time
  assertions.
