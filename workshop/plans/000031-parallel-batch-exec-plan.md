# Parallel Batch Executor Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

## Revisions

- **2026-07-13 (post-implementation, from the close-boundary review) —** two Core-concepts cells corrected to match the shipped code (capability unchanged; location/casing only): (1) Pure entity `execFor` → **`ExecFor`** (exported — `cmd/metis` is a different package than `pkg/sampler`); (2) integration point `syncWriter` lives in **`cmd/metis/run.go`** (built inside `runExperiment`, the one-home parallel-invariant per plan-quality judge INFO #1), not `cmd/metis/main.go`. Deferred Minor (review, ARCH-DRY): `runNestedCV`'s `errMu`/`firstErr` set-once latch duplicates `sweepPass.setErr`/`firstError` — a shared `errOnce{mu; err}` type would consolidate (~15 lines); safe follow-up, not done in this pass.

**Goal:** Run a sampler's `Ask` batch concurrently instead of one subprocess at a time, so a 495/2,475-run grid/nested sweep finishes in wall-clock ≈ slowest-leaf × ceil(work/n) instead of Σ(all leaves), under a single global concurrency cap `n`, with byte-identical results.

**Architecture:** One seam, injected the same way `runPoint` is (ARCH-PURE): `Run` gains an `exec(points, runPoint) []O` strategy that runs a batch and returns outputs **in batch order**; `Run` then `Tell`s them in that fixed order, so the order-independent reduce (metis#18) yields an identical `Done`. Two strategies live in `pkg/sampler`: `SeqExec` (today's serial map — the default tests use) and `ParExec` (goroutine fan-out, order-preserving). Orchestration goroutines are cheap and unbounded; the **only** budgeted resource is the real subprocess spawn — a single shared semaphore (capacity `n`) is acquired around `cmd.CombinedOutput()` in `execStep` (the leaf), so nesting (`driver ⊃ sweeper ⊃ resample`) can fan out to hundreds of goroutines yet never exceed `n` concurrent **step** subprocesses, and no orchestration goroutine ever holds a slot while awaiting children (deadlock-free). (The cap is on the *step* subprocess — the uv/python run. Short-lived `git` helpers the record/cache layer spawns per step run OUTSIDE the cap; that's correctness-neutral per the spec but see the C1 hazard + M2 note below.)

**Tech Stack:** Go generics (the existing `Sampler[S,P,O,R]` / `Run` are generic), `chan struct{}` semaphore, `sync.WaitGroup`, the metis#2 content-addressed cache (`pkg/cas` atomic `Put` + `cmd/metis/caching.go`).

---

## Why this is safe (the concurrency audit that shaped the design)

Recon of the run path established the safety envelope — the plan depends on these facts:

1. **Distinct output dirs.** Every (config,fold) point has a content-addressed `runID` → its step outputs land in `runs/<runID>/<stepID>/`, distinct across parallel points. No output collision. (`cmd/metis/run.go:133`, `cmd/metis/exec.go:41`.)
2. **Per-subprocess cwd.** `execStep` sets `cmd.Dir = stepDir`, so the sensor's `reads.json` / `with.json` / `metrics.json` land in the point's own step dir — **no shared-cwd race.** (`cmd/metis/exec.go:60`.)
3. **Atomic CAS blob writes.** `FSStore.Put` writes temp + `os.Rename` (`pkg/cas/fs.go:100`); two points computing the same shared upstream (get-data/adapt) write byte-identical blobs → last-rename-wins is correct.
4. **Per-point executor instances.** `runResolvedExperiment` builds a fresh `cachingExecutor` per call (`run.go:152`); its `kpres`/`transitiveD` maps are per-point — **no shared mutable in-memory state.**
5. **ONE real race to fix:** `cachingExecutor.writeEntry` writes the cache index with a non-atomic `os.WriteFile` to a shared `index/<kpre>.json` (`caching.go:348`). Concurrent points computing the same shared step torn-write/torn-read it. → Task 4 makes it atomic (temp + rename), matching the CAS store.
6. **A cache HIT spawns no subprocess** (`caching.go:107` `materialize`), so it draws no budget — only a MISS reaches `execStep` and acquires the semaphore. This is why the leaf is the correct enforcement point.
7. **SECOND real fix (Critical, found in review):** `runPipelineFold` re-probes `git status --porcelain` per fold as a mid-sweep code-freeze check (`sweep.go:410-412`), and `probeRepo` swallows *any* probe error to `sha=""` (`sweep.go:657-665`). Under fan-out, configs×folds concurrent `git status` on a worktree that the step subprocesses are simultaneously writing (`runs/…`) contends on `.git/index.lock` → a probe exits non-zero → `sha=""` → `"" != codeID` → a **false "code changed mid-sweep" abort that kills the whole honest run.** → Task 5 Step 0 fixes the root cause: only abort on a *definite* sha change (`if s != "" && s != ss.codeID`), so a swallowed probe failure can never masquerade as a code change. (The check stays per-fold; the bounded extra `git` processes are a documented perf note, M2 — not a correctness issue once the false-abort is closed.)

**Documented non-goals (correct, not optimal — write into the flag help + RUNBOOK):**
- **Thundering herd on a COLD cache:** the first batch's ≤`n` points may each recompute the shared upstream (e.g. `n` concurrent Kaggle downloads) before any populates the cache. Bounded by `n`, correct via atomic writes, wasteful. A future singleflight (per-K_pre in-process lock) or a data-phase warm-pass is out of scope. The honest-run data is already cached, so this bites only a fresh checkout.
- **BLAS/`n_jobs` oversubscription:** each leaf is a Python process that may multi-thread (sklearn/OpenBLAS), so `n = NumCPU` processes can oversubscribe cores. Tuning knob (`OMP_NUM_THREADS=1` / set `n` below NumCPU), not correctness.
- **Interleaved progress output:** parallel goroutines write `out` concurrently → lines interleave, AND a concurrent `bytes.Buffer.Write` is itself a data race (the `-race` gate would fail). Clean per-`k/n` progress is the sibling metis#30's scope; this plan only prevents the race + torn lines via a `syncWriter` (Task 3) that wraps `out` when `parallel`. Line *ordering* across goroutines stays best-effort (cosmetic).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `SeqExec` | `pkg/sampler/exec.go` | new |
| `ParExec` | `pkg/sampler/exec.go` | new |
| `execFor` | `pkg/sampler/exec.go` | new |
| `Run` (exec param) | `pkg/sampler/run.go` | modified |

- **`SeqExec[P,O]`** — `func(points []P, runPoint func(P) O) []O` that maps `runPoint` over the batch serially, in order. Byte-for-byte today's loop body; the backward-compat default every sampler test passes.
  - **DRY rationale:** first occurrence of the batch-exec strategy pattern; `ParExec` is the second.
- **`ParExec[P,O]`** — same signature; runs each point in its own goroutine, writes each result to `out[i]` (its fixed batch index — no shared append, no mutex needed on the result slice), `WaitGroup`-joins, returns `out`. Pure combinator: no IO/state of its own; all IO is in the injected `runPoint`, all subprocess bounding is at the leaf semaphore (below). Order-preserving by construction (index-addressed writes), so `Run`'s subsequent in-order `Tell` sees the same sequence `SeqExec` would.
  - **Relationships:** N/A (stateless function).
  - **Future extensions:** a bounded-goroutine variant if orchestration-goroutine count itself ever matters (it doesn't at 495–2,475).
- **`execFor[P,O](parallel bool)`** — returns `ParExec[P,O]` when `parallel`, else `SeqExec[P,O]`. The single branch point; each of the 4 `Run` call sites calls `execFor[...](ss.parallel)` so the Seq/Par choice lives in exactly one place (ARCH-DRY) and stays type-safe per level.
- **`Run` (modified)** — gains a trailing `exec func([]P, func(P) O) []O` parameter. The batch loop becomes `outs := exec(batch, runPoint); for i, p := range batch { s = smp.Tell(s, p, outs[i]) }`. The empty-batch panic and the Ask/done contract are unchanged.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `execStep.sem` | `cmd/metis/exec.go` | modified | subprocess spawn |
| `runOpts.maxParallel` / `.leafSem` | `cmd/metis/run.go` | modified | run config |
| `cachingExecutor.writeEntry` (atomic) | `cmd/metis/caching.go` | modified | cache index file |
| `syncWriter` | `cmd/metis/main.go` | new | concurrent `out` writes |
| `--parallel` flag / `METIS_MAX_PARALLEL` | `cmd/metis/main.go` | modified | user input |
| sweep `Run` call sites | `cmd/metis/sweep.go` | modified | orchestration |

- **`execStep.sem`** — a `chan struct{}` of capacity `n`, shared across every `execStep` invocation of a run. `Execute` does `if e.sem != nil { e.sem <- struct{}{}; defer func(){ <-e.sem }() }` immediately around `cmd.CombinedOutput()`. Nil sem (serial path / cache-only) = no bound.
  - **Injected into:** constructed once per `metis run` (from `runOpts.maxParallel`) and threaded via `runOpts.leafSem` into every `runResolvedExperiment` → `execStep{...sem: o.leafSem}`. `runOpts` is value-copied per point (`pointOpts := ss.o`), but a `chan` is a reference — all copies share the one semaphore (one global budget). ← the load-bearing invariant.
  - **Future extensions:** a weighted semaphore if leaves ever declare heterogeneous cost.
- **`runOpts.maxParallel int` / `runOpts.leafSem chan struct{}`** — `maxParallel` (from the flag) sizes the semaphore; `leafSem` is the shared channel. `parallel := maxParallel > 1` selects `ParExec` in the sweep.
- **`cachingExecutor.writeEntry` (atomic)** — change `os.WriteFile(path, b, 0644)` to a temp-write + `os.Rename` in `indexDir` (same-dir rename is atomic on the same filesystem). Fixes the one concurrent-write race (audit item 5).
- **`syncWriter`** — a `struct { mu sync.Mutex; w io.Writer }` whose `Write` locks around `w.Write`. `cmdRun` wraps `os.Stdout` in it when `parallel > 1` so the fan-out goroutines' `Fprintf(out, …)` don't race/tear. Minimal — does NOT reorder or buffer per-goroutine (that's metis#30). Injected as `runOpts.out`.
- **`--parallel` flag** — `metis run --parallel <n>`; default `runtime.NumCPU()`; `METIS_MAX_PARALLEL` env overrides the default when the flag is unset; `<=1` ⇒ true-serial (`SeqExec`, nil sem — exact current behavior). Help text carries the BLAS caveat.

---

## Chunk 1: the sampler seam + strategies

### Task 1: `exec` seam on `Run` with `SeqExec` default (backward-compat)

**Files:**
- Create: `pkg/sampler/exec.go`
- Create: `pkg/sampler/exec_test.go`
- Modify: `pkg/sampler/run.go:13-31`
- Modify (compile-fix call sites): `cmd/metis/sweep.go` (4 `sampler.Run` calls), any `sampler.Run` in `pkg/sampler/*_test.go`

- [ ] **Step 1: Write the failing test** — `SeqExec` maps in order.

```go
// pkg/sampler/exec_test.go
package sampler

import "testing"

func TestSeqExec_MapsInOrder(t *testing.T) {
	out := SeqExec([]int{1, 2, 3}, func(x int) int { return x * 10 })
	want := []int{10, 20, 30}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("out[%d]=%d want %d", i, out[i], want[i])
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./pkg/sampler/ -run TestSeqExec_MapsInOrder`
Expected: FAIL — `undefined: SeqExec`.

- [ ] **Step 3: Implement `SeqExec` (and stub `ParExec`/`execFor` for later tasks)**

```go
// pkg/sampler/exec.go
package sampler

// SeqExec runs a batch serially, in order — byte-identical to Run's original
// loop body. The backward-compatible default (tests pass this to Run).
func SeqExec[P, O any](points []P, runPoint func(P) O) []O {
	out := make([]O, len(points))
	for i, p := range points {
		out[i] = runPoint(p)
	}
	return out
}
```

- [ ] **Step 4: Modify `Run` to take the exec seam**

```go
// pkg/sampler/run.go — signature + loop body
func Run[S, P, O, R any](ctx Ctx, smp Sampler[S, P, O, R], runPoint func(P) O, exec func([]P, func(P) O) []O) R {
	s := smp.Init(ctx)
	for {
		batch, done := smp.Ask(s)
		if done {
			break
		}
		if len(batch) == 0 {
			panic("sampler: Ask returned an empty batch without done — a Sampler must make progress (emit a non-empty batch or report done)")
		}
		outs := exec(batch, runPoint)
		for i, p := range batch {
			s = smp.Tell(s, p, outs[i])
		}
	}
	return smp.Done(s)
}
```
Update the `Run` doc comment: add a sentence that `exec` runs a batch (whose points are independent by construction) and returns outputs in batch order; `SeqExec` is the default, `ParExec` runs them concurrently under the leaf semaphore.

- [ ] **Step 5: Fix all `sampler.Run(...)` call sites to pass `sampler.SeqExec[P,O]`**

Every existing call gets a 4th arg. In tests and (temporarily) in `sweep.go`, append `sampler.SeqExec[FoldPoint, FoldOutcome]` / `[shape.Point, sampler.MeanSE]` / `[sampler.SinglePoint, sampler.SweepResult]` / `[sampler.OuterFoldPoint, float64]` matching each level. (Task 5 replaces the sweep.go ones with `execFor`.)

- [ ] **Step 6: Run the full sampler + cmd suites — backward-compat green**

Run: `go test ./pkg/sampler/... ./cmd/metis/...`
Expected: PASS (identical behavior; only the signature changed).

- [ ] **Step 7: Commit**

```bash
git add pkg/sampler/exec.go pkg/sampler/exec_test.go pkg/sampler/run.go cmd/metis/sweep.go pkg/sampler/*_test.go
git commit -m "#31 M-none: Run takes an injected batch exec; SeqExec is the default (backward-compat)"
```

### Task 2: `ParExec` + `execFor`, proven order-preserving and equivalent

**Files:**
- Modify: `pkg/sampler/exec.go`
- Modify: `pkg/sampler/exec_test.go`

- [ ] **Step 1: Write the failing test** — `ParExec` ≡ `SeqExec` on results + order, even when work finishes out of order.

```go
func TestParExec_OrderPreservingUnderReordering(t *testing.T) {
	// Force completion order to be the REVERSE of input order: point i blocks
	// until point i+1 has finished. If ParExec appended in completion order the
	// result would be reversed; index-addressed writes must keep INPUT order.
	pts := []int{0, 1, 2, 3, 4}
	done := make([]chan struct{}, len(pts))
	for i := range done {
		done[i] = make(chan struct{})
	}
	rp := func(x int) int {
		if x+1 < len(pts) {
			<-done[x+1] // wait for my successor to finish first
		}
		res := x * 10
		close(done[x]) // now let my predecessor proceed
		return res
	}
	par := ParExec(pts, rp)
	for i, x := range pts {
		if par[i] != x*10 {
			t.Fatalf("ParExec[%d]=%d want %d (completion-order leak?)", i, par[i], x*10)
		}
	}
}

func TestParExec_RunsConcurrently(t *testing.T) {
	// N points that each block on a shared barrier only all-N can release:
	// completes iff they truly run at once (a serial map would deadlock/timeout).
	const n = 8
	var arrived, release = make(chan struct{}, n), make(chan struct{})
	done := make(chan []int, 1)
	go func() {
		done <- ParExec(make([]int, n), func(int) int { arrived <- struct{}{}; <-release; return 1 })
	}()
	for i := 0; i < n; i++ {
		<-arrived // all n goroutines reached the barrier ⇒ genuinely concurrent
	}
	close(release)
	if got := <-done; len(got) != n {
		t.Fatalf("got %d results want %d", len(got), n)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./pkg/sampler/ -run TestParExec`
Expected: FAIL — `undefined: ParExec`.

- [ ] **Step 3: Implement `ParExec` + `execFor`**

```go
// pkg/sampler/exec.go (append)
import "sync"

// ParExec runs a batch concurrently — one goroutine per point — and returns
// outputs in BATCH ORDER (each goroutine writes its own index, so no result
// mutex is needed and the sequence Run.Tell sees is identical to SeqExec's).
// It bounds NOTHING itself: orchestration goroutines are cheap; the only
// budgeted resource is the real subprocess spawn, capped by the leaf semaphore
// inside the injected runPoint's execStep (metis#31). So nesting fans out
// freely while live subprocesses stay ≤ n, and no orchestration goroutine holds
// the budget while awaiting children (deadlock-free).
func ParExec[P, O any](points []P, runPoint func(P) O) []O {
	out := make([]O, len(points))
	var wg sync.WaitGroup
	wg.Add(len(points))
	for i, p := range points {
		go func(i int, p P) { defer wg.Done(); out[i] = runPoint(p) }(i, p)
	}
	wg.Wait()
	return out
}

// execFor selects the batch strategy — the single Seq/Par branch point (ARCH-DRY).
func execFor[P, O any](parallel bool) func([]P, func(P) O) []O {
	if parallel {
		return ParExec[P, O]
	}
	return SeqExec[P, O]
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./pkg/sampler/ -run TestParExec -race`
Expected: PASS, no race (index-addressed writes are race-free).

- [ ] **Step 5: Commit**

```bash
git add pkg/sampler/exec.go pkg/sampler/exec_test.go
git commit -m "#31: ParExec (order-preserving goroutine fan-out) + execFor strategy selector"
```

---

## Chunk 2: the leaf semaphore + the cache-race fix

### Task 3: global leaf semaphore in `execStep`, wired to `--parallel`

**Files:**
- Modify: `cmd/metis/exec.go:28-34,81-85`
- Modify: `cmd/metis/run.go:47-62,142`
- Modify: `cmd/metis/main.go:37-58`
- Test: `cmd/metis/exec_test.go` (new test)

- [ ] **Step 1: Write the failing test** — an instrumented executor never exceeds `n` concurrent subprocess spawns.

```go
// cmd/metis/exec_test.go
// A fake StepExecutor that records peak concurrency, wrapped by the SAME
// acquire/release the production execStep uses, proves the semaphore bounds
// concurrent LEAF executions to n regardless of goroutine fan-out.
func TestLeafSemaphore_BoundsConcurrency(t *testing.T) {
	const n = 3
	sem := make(chan struct{}, n)
	var mu sync.Mutex
	var cur, peak int
	leaf := func() {
		sem <- struct{}{}
		defer func() { <-sem }()
		mu.Lock(); cur++; if cur > peak { peak = cur }; mu.Unlock()
		time.Sleep(2 * time.Millisecond)
		mu.Lock(); cur--; mu.Unlock()
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ { wg.Add(1); go func(){ defer wg.Done(); leaf() }() }
	wg.Wait()
	if peak > n {
		t.Fatalf("peak concurrency %d exceeded cap %d", peak, n)
	}
}
```
(This pins the acquire/release contract directly. Task 5 adds the end-to-end nested-sweep peak-≤-n test through the real `Run`.)

- [ ] **Step 2: Run to verify it fails** (compiles once `sync`/`time` imported; fails only if the pattern is wrong)

Run: `go test ./cmd/metis/ -run TestLeafSemaphore_BoundsConcurrency -race`
Expected: PASS with the correct pattern (this test encodes the contract execStep must implement).

- [ ] **Step 3: Add the `sem` field + acquire/release to `execStep`**

```go
// cmd/metis/exec.go — struct
type execStep struct {
	stepPath []string
	expDir   string
	seed     int
	readRoot string
	out      io.Writer
	sem      chan struct{} // metis#31: global leaf budget; nil = unbounded (serial path)
}

// cmd/metis/exec.go — around the subprocess spawn (replaces the bare CombinedOutput call)
if e.sem != nil {
	e.sem <- struct{}{}
	defer func() { <-e.sem }()
}
combined, err := cmd.CombinedOutput()
if err != nil {
	return experiment.StepResult{}, fmt.Errorf("exec %s: %w\n%s", exe, err, combined)
}
```
(Acquire is AFTER `resolve`/`MkdirAll`/`with.json` write — those are cheap, non-subprocess; the budget is only the process spawn.)

- [ ] **Step 4: Thread `maxParallel`/`leafSem` through `runOpts` and into `execStep`**

```go
// cmd/metis/run.go — runOpts
maxParallel int          // metis#31: >1 ⇒ ParExec; sizes leafSem
leafSem     chan struct{} // metis#31: shared global subprocess budget (nil = serial/cache-only)

// cmd/metis/run.go:142 — pass the shared sem into the production executor
var exec experiment.StepExecutor = execStep{stepPath: o.stepPath, expDir: expDir, seed: exp.Seed, readRoot: o.readRoot, out: out, sem: o.leafSem}
```

- [ ] **Step 5: Add the `--parallel` flag + env default + construct the shared semaphore ONCE**

```go
// cmd/metis/main.go — cmdRun
parallel := fs.Int("parallel", defaultParallel(), "max concurrent step subprocesses across all sweep levels (metis#31); <=1 = serial. Caveat: each leaf is a Python process that may itself multi-thread (BLAS/n_jobs) — n=NumCPU can oversubscribe cores; pin OMP_NUM_THREADS=1 or lower n.")
// ...after parse: cmdRun just passes maxParallel + the raw out. The parallel
// INVARIANT (non-nil leafSem + synchronized out) is established in ONE place —
// runExperiment — so no direct-runOpts caller (the tests) can forget it
// (plan-quality judge INFO #1). This removes a whole class of -race test flake.
_, err := runExperiment(runOpts{
	expPath: rest[0], runID: *runID, stepPath: stepPath(rest[0]),
	cache: *cache, dryRun: *dryRun, out: os.Stdout,
	maxParallel: *parallel,
})
```

Then in `runExperiment` (run.go), normalize the parallel invariant up-front:
```go
// runExperiment: establish the parallel invariant in one home.
if o.maxParallel > 1 {
	if o.leafSem == nil {
		o.leafSem = make(chan struct{}, o.maxParallel)
	}
	o.out = &syncWriter{w: out} // serialize concurrent fan-out writes (I2); `out` = resolved o.out/io.Discard
	out = o.out
}

// syncWriter serializes concurrent Write calls (the parallel fan-out's progress
// output). Minimal — does not reorder/buffer per goroutine (metis#30's scope).
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}

// defaultParallel: METIS_MAX_PARALLEL env overrides NumCPU when set + valid.
func defaultParallel() int {
	if v := os.Getenv("METIS_MAX_PARALLEL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			return n
		}
	}
	return runtime.NumCPU()
}
```

- [ ] **Step 6: Test the REAL `execStep` acquire/release (I5 — not a fake)**

The fake-exec e2e tests bypass `execStep`, so nothing else proves the production acquire is actually wired. Write a test that drives the real `execStep.Execute` with a resolvable trivial step-type (a tiny shell/script fixture under a temp `stepPath` that just sleeps briefly), `sem` cap 1, two concurrent `Execute` calls; assert they serialize (an instrumented step that records overlapping start/end, or measured wall-clock ≈ 2×sleep not ≈1×). Also assert a mis-thread would fail: a variant with `sem == nil` (default) must NOT serialize. This is the one test that catches a forgotten acquire or a mis-threaded `runOpts.leafSem → execStep.sem`.

```go
func TestExecStep_SemaphoreSerializesRealSubprocess(t *testing.T) {
	dir := t.TempDir()
	// write an executable step-type fixture at <dir>/test/sleeper that sleeps ~50ms
	// (resolvable via stepPath=[dir]); run two Execute concurrently with sem cap 1;
	// assert non-overlap (peak==1) vs the nil-sem control (peak==2).
}
```

- [ ] **Step 7: Run cmd suite**

Run: `go test ./cmd/metis/ -race`
Expected: PASS (execStep default path — nil sem for existing single-run tests — unchanged; the new serialization test green).

- [ ] **Step 8: Commit**

```bash
git add cmd/metis/exec.go cmd/metis/run.go cmd/metis/main.go cmd/metis/exec_test.go
git commit -m "#31: global leaf semaphore in execStep + --parallel/METIS_MAX_PARALLEL (default NumCPU); syncWriter"
```

### Task 4: make the cache index write atomic (the one concurrency race)

**Files:**
- Modify: `cmd/metis/caching.go:340-349`
- Test: `cmd/metis/caching_test.go` (new test)

- [ ] **Step 1: Write the failing test** — concurrent `writeEntry` of the same key never yields a torn/parse-failing index file.

```go
// READER-vs-WRITER (I1): a writer-vs-writer test with identical content + a
// post-join read passes even against the buggy os.WriteFile — the torn state
// is only visible to a reader racing a writer. So race a lookup loop against
// repeated writes and assert the reader NEVER sees a torn/parse-failing file.
// (Vary payload length across writes so a truncating O_TRUNC write is mid-way
// observably short.)
func TestWriteEntry_ReaderNeverSeesTornIndex(t *testing.T) {
	dir := t.TempDir()
	c := &cachingExecutor{indexDir: filepath.Join(dir, "index")}
	short := cache.Entry{Kpre: "k", TransitiveD: []record.CodeRef{}, Output: "s"}
	long := cache.Entry{Kpre: "k", TransitiveD: make([]record.CodeRef, 50), Output: "loooooong"}
	if err := c.writeEntry(short); err != nil { t.Fatal(err) }
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { // writer: alternate lengths to widen the torn-read window
		defer wg.Done()
		for i := 0; ; i++ {
			select { case <-stop: return; default: }
			if i%2 == 0 { _ = c.writeEntry(long) } else { _ = c.writeEntry(short) }
		}
	}()
	for i := 0; i < 5000; i++ { // reader: must never parse-fail
		if _, ok, err := c.lookup("k"); err != nil || !ok {
			close(stop); wg.Wait()
			t.Fatalf("reader saw torn index at iter %d: ok=%v err=%v", i, ok, err)
		}
	}
	close(stop); wg.Wait()
}
```

- [ ] **Step 2: Run to verify it fails against the CURRENT (non-atomic) `writeEntry`**

Run: `go test ./cmd/metis/ -run TestWriteEntry_ReaderNeverSeesTornIndex -count=20`
Expected: FAIL (a `lookup` mid-write torn read → `DecodeEntry` parse error). If it passes against the buggy code, the test isn't reproducing the race — strengthen it (more iterations / wider length gap) before proceeding.

- [ ] **Step 3: Make `writeEntry` atomic**

```go
func (c *cachingExecutor) writeEntry(e cache.Entry) error {
	if err := os.MkdirAll(c.indexDir, 0o755); err != nil {
		return err
	}
	b, err := cache.EncodeEntry(e)
	if err != nil {
		return err
	}
	final := filepath.Join(c.indexDir, string(e.Kpre)+".json")
	tmp, err := os.CreateTemp(c.indexDir, "."+string(e.Kpre)+".tmp*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close(); os.Remove(tmp.Name()); return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name()); return err
	}
	return os.Rename(tmp.Name(), final) // same-dir rename is atomic
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./cmd/metis/ -run TestWriteEntry_ReaderNeverSeesTornIndex -count=50`
Expected: PASS every time (the reader never observes a torn index).

- [ ] **Step 5: Commit**

```bash
git add cmd/metis/caching.go cmd/metis/caching_test.go
git commit -m "#31: atomic cache-index write (temp+rename) — the one concurrent-write race under parallel sweeps"
```

---

## Chunk 3: wire parallelism into the sweep + prove end-to-end determinism

### Task 5: thread `execFor` into the 4 sweep `Run` sites + the load-bearing determinism & peak-concurrency tests

**Files:**
- Modify: `cmd/metis/sweep.go` (add `parallel bool` + a `sync.Mutex` to `shapeSweep`/`sweepPass`; replace `SeqExec` args with `execFor[...](ss.parallel)`; the C1 probe fix; sort side-records)
- Modify: `cmd/metis/run.go` (`runShapeSweep` sets `ss.parallel = o.maxParallel > 1`)
- Test: `cmd/metis/sweep_test.go` / the e2e test file (determinism + peak-≤-n + C1 regression) + a sampler-level determinism test in `pkg/sampler/exec_test.go`

- [ ] **Step 0: Fix the C1 false-abort BEFORE enabling fan-out (Critical).** In `runPipelineFold` (`sweep.go:410-412`) the mid-sweep code-freeze check must only fire on a *definite* sha change, never a swallowed probe failure:

```go
// was: if _, s, _ := probeRepo(...); s != ss.codeID { p.err = ... }
if _, s, _ := probeRepo(ss.o.git, filepath.Dir(ss.o.expPath)); s != "" && s != ss.codeID {
	// s == "" means the probe FAILED (probeRepo swallows errors) — under parallel
	// git status contention that is transient, NOT a code change. Only a non-empty
	// sha that differs from the frozen codeID is a real mid-sweep HEAD move.
	p.setErr(fmt.Errorf("code changed mid-sweep (%s → %s) — re-run to sweep the new revision", ss.codeID, s))
	return sampler.FoldOutcome{}
}
```
Add a regression test: a fold whose injected `gitProbe` returns an error (→ `probeRepo` sha="") must NOT abort. (This is a pure-ish unit over `runPipelineFold`'s guard with a fake probe — the fake-probe suite CAN cover it, unlike the real-contention scenario.)

- [ ] **Step 1: Set `ss.parallel` from opts** — `shapeSweep` gains `parallel bool`; `runShapeSweep` sets it: `parallel: o.maxParallel > 1`.

- [ ] **Step 2: Replace the exec arg at each of the 4 `Run` call sites**

```go
// runSweeper — both levels
sampler.Run(ctx, sampler.GridConfigs{...}, func(c shape.Point) sampler.MeanSE {
	ms := sampler.Run(ctx, sampler.FixedKFolds{K: pass.splitK},
		func(f sampler.FoldPoint) sampler.FoldOutcome { return pass.runPipelineFold(c, f) },
		execFor[sampler.FoldPoint, sampler.FoldOutcome](ss.parallel))
	pass.configs = append(pass.configs, configScore{point: c, meanSE: ms}) // ← see Step 3
	return ms
}, execFor[shape.Point, sampler.MeanSE](ss.parallel))

// SingleDriver (runShapeSweep) and CVDriver (runNestedCV): append
//   execFor[sampler.SinglePoint, sampler.SweepResult](ss.parallel)
//   execFor[sampler.OuterFoldPoint, float64](ss.parallel)
```

- [ ] **Step 3: Guard EVERY shared write the fan-out now races (I4 — the plan's first cut missed two)**

Add a `sync.Mutex` to `sweepPass` and guard, via tiny append-only critical sections:
  - `pass.configs` append (sweeper runPoint, `sweep.go:113`);
  - `pass.points` append (`runPipelineFold`, `sweep.go:432`);
  - **`pass.err` — BOTH the set-once write (`sweep.go:404,411,418,429`) AND the early-out READ (`sweep.go:404`)**; wrap as a `setErr`/`err()` pair under the mutex (a concurrent read+write is a race even when the write is "set once"). Use `setErr` in the C1 fix above too.
  - **`firstErr` in `runNestedCV` (`sweep.go:295,300-302`)** — `CVDriver.Ask` emits all outer folds at once, so `ParExec` runs the closure concurrently; guard its read+write with a small mutex (or make `runOuterFold` collect errors and reduce after the join). Outcome is benign but it's a real data race → the `-race` gate fails without this.
These are orchestration bookkeeping, not the reduce — the reduce stays in the sampler's pure `Tell`/`Done`. The `-race` run in Step 7 is the proof they're all covered.

- [ ] **Step 4: Make persisted artifacts deterministic (I3) — sort side-records before writing**

`pass.points` lands in completion order under fan-out → non-deterministic `manifest.json` + `.ledger.csv` bytes (undercuts metis's content-addressing posture even though the winner/estimate are already deterministic). Before `writeManifest`/`writeSweepLedger` consume them, sort `ss.man.Points` by a stable content key (`RunID`, then `Fold`). Add an assertion to the determinism test (Step 5) that the two runs' manifest+ledger bytes are IDENTICAL, not just the aggregate.

- [ ] **Step 5: The load-bearing determinism test — parallel ≡ serial, at BOTH levels (M3)**

Two tests:
```go
// (a) SAMPLER-level (closest to the Done-when's "byte-identical Done(S)"): drive
// Run over a fake runPoint with random per-point delays, ParExec vs SeqExec,
// assert the whole reduced result is ==. No cmd/metis, no subprocess.
func TestRun_ParExecEqualsSeqExec(t *testing.T) { /* fake sampler + delayed runPoint */ }

// (b) CMD-level e2e via the injected fake StepExecutor, twice (maxParallel=1 and
// NumCPU) over the same multi-config × multi-fold shape. Assert (i) the WHOLE
// SweepResult == (per-family winners + ship pick, exact ==, not just mean/SE),
// AND (ii) after Step 4's sort, the manifest.json + .ledger.csv bytes are identical.
func TestSweep_ParallelEqualsSerial(t *testing.T) { /* fake exec; compare full SweepResult + artifact bytes */ }
```

- [ ] **Step 6: The peak-concurrency-≤-n test under nested `driver: cv`**

```go
// A fake StepExecutor that acquires the SAME leafSem and records peak concurrency;
// run a driver:cv shape (outerK × configs × innerK). Assert recorded peak ≤ n
// even though driver×sweeper×resample fan out to hundreds of goroutines, and the
// run completes (no deadlock). NOTE: this uses a fake exec, so it proves the
// BUDGET math, not the production execStep wiring — Task 3 Step 6 covers the wiring.
func TestNestedCV_PeakConcurrencyWithinCap(t *testing.T) { /* leafSem cap n; instrumented fake */ }
```

- [ ] **Step 7: Run the whole suite under the race detector**

Run: `go test ./... -race`
Expected: PASS, no data races (Step 3's mutexes cover `configs`/`points`/`err`/`firstErr`; the syncWriter covers `out`).

- [ ] **Step 8: Wall-clock demo (manual, recorded in the issue Log)**

Run on the fixture sweep, comparing serial vs parallel:
```bash
go build -o bin/metis ./cmd/metis
time bin/metis run --parallel 1 <fixture-sweep>.md   # serial baseline
time bin/metis run --parallel 8 <fixture-sweep>.md   # parallel
```
Expected: parallel wall-clock materially below serial (record both numbers). (The fixture is small; the real 99×5 sweep is the honest demo, run separately.)

- [ ] **Step 9: Commit**

```bash
git add cmd/metis/sweep.go cmd/metis/run.go cmd/metis/*_test.go
git commit -m "#31: run sweeper/resample/driver batches via execFor; determinism + peak-≤-n tests"
```

### Task 6: document the tuning caveats + close

**Files (pinned to the metis repo — keep this a single-repo close; plan-quality judge INFO #2):**
- Modify: metis `atlas/` (the run/sweep flow map gains the exec seam + leaf-semaphore concept + the `--parallel` flag with the BLAS/thundering-herd caveats; update `atlas/index.md` if a new file). The `--parallel` flag help text itself carries the operator-facing caveat, so the kbench RUNBOOK note is optional and deferred (a peer write) rather than blocking this close.

- [ ] **Step 1: Add the `--parallel` + caveats note** (thundering-herd on cold cache, BLAS oversubscription, interleaved output → metis#30).
- [ ] **Step 2: Update the atlas** (the exec seam on `Run`; the single global leaf budget).
- [ ] **Step 3: `sdlc close`** (single-pass atomic work, no `Mx` — one boundary review at close). Verified = the determinism test + the peak-≤-n test + the wall-clock demo numbers. `--actual` measured (`sdlc actual --issue 31`).

---

## Review status

Fresh-eyes reviewed against the actual code before approval (verdict: issues-found, core design sound). All folded in: **C1** (git-probe false-abort → Task 5 Step 0 + regression test), **I1** (reader-vs-writer atomicity test → Task 4), **I2** (`syncWriter` now a real entity + Task 3 wiring), **I3** (sort side-records before persist → Task 5 Step 4), **I4** (`firstErr` + `p.err`-read added to the mutex coverage → Task 5 Step 3), **I5** (real-`execStep` serialization test → Task 3 Step 6), **M1** (barrier reorder test → Task 2), **M2** ("≤ n *step* subprocesses" prose + git-helper note), **M3** (whole-`SweepResult` + sampler-level determinism test → Task 5 Step 5). The five reusable lessons landed in `workshop/lessons.md`.

## Open decision for operator review

- **Default `n = NumCPU` (parallel by default) vs. opt-in (`--parallel` off by default).** The spec says default `NumCPU`. Risk: parallel-by-default changes the out-of-box behavior of every `metis run` sweep and interleaves output. Recommendation: keep the spec's `NumCPU` default (the win is the point), ship the true-serial escape hatch (`--parallel 1`), and defer clean progress to metis#30. Flag if you'd rather it be opt-in.
