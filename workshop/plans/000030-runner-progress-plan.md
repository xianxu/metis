# Runner Progress Reporting Implementation Plan (metis#30)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A sweep stops running blind: `SizeHint` on the `Sampler` interface (n per level), a progress callback fired by `Run` once per completed point (live k + the point's typed output), and a throttled aggregated line in `cmd/metis` (`outer 1/3 · configs 84/216 · folds 421/1080 · est 0.8283 ± 0.0140`).

**Architecture:** One event mechanism through the existing `Run` loop (no new runner): `Run` wraps the injected `runPoint` so the callback fires at **point completion** (live under `ParExec`), carrying `{K, Total, Kind, Point, Out}` — a *generic typed* event, so each level's closure in `cmd/metis` receives its own `P`/`O` without type switches. A mutex'd `sweepProgress` sink in `cmd/metis` folds per-level events into one throttled plain line (injected clock). `pkg/sampler` stays pure: the callback is injected like `runPoint`/`exec` (ARCH-PURE); totals come only from `SizeHint` (ARCH-DRY — no shape-math duplication in the renderer).

**Tech Stack:** Go. `pkg/sampler` (interface + Run), `cmd/metis` (sink + wiring). Tests: `go test ./pkg/sampler/ ./cmd/metis/`.

---

## Spec revision this plan encodes (→ issue `## Revisions` at change-code)

The issue's premise — *"the Run loop fires a `Tell` per completed point (that's k)"* — was true under serial execution when filed (2026-07-13), but #31 (landed later the same day) made `exec` batch-scoped: `Run` calls `exec(batch, runPoint)` and `Tell`s **after the whole batch returns**, and every production sampler is one-batch static (`GridConfigs`/`FixedKFolds`/`CVDriver`/`SingleDriver` each `Ask` their whole point-set once). A per-`Tell` callback would therefore fire *all* its events at batch end — dead as live progress, exactly the blindness this issue exists to fix.

**Revision:** `Run` fires `progress` at **point completion** (it wraps `runPoint` before handing it to `exec`), mutex-serialized with a monotonically increasing `k`. The *count* contract is unchanged — exactly one event per point, same as one `Tell` per point — only the *timing* moves from batch-end to live. The event carries the completed `(Point, Out)` pair instead of accumulator state (`S` is untouched until `Tell`, and it's opaque to `Run` anyway); the running estimate is derived by the `cmd/metis` sink from the typed outputs its own closures already handle (it records `configScore`s/outer scores today).

**#38 seam (designed for, not built):** the TUI consumes this same callback; the sink receives every fold completion and holds the injected clock (`ss.now`), which is what the moving-average runs/sec + ETA line needs; leaf occupancy will come from the #31 `leafSem` gauge in `cmd/metis`. **Outer-fold identity comes from CLOSURE BINDING, not the payload** (plan-review finding): the event payload has no outer-fold coordinate and must not grow one — `runOuterFold` knows `i` when it creates each pass's Run closures, so the sink hands out per-pass hooks (`prog.forPass(outerIdx)`) whose closures carry the identity. #38 builds its per-in-flight-fold board rows by extending the sink behind those same hooks — zero `pkg/sampler` change, honoring its "no new instrumentation" constraint. (Beware the lookalike: `FoldPoint.Partition` is byte-identical across outer folds — it is NOT a discriminator.) Nothing unrendered is built now — #38 adds its own state when it renders it.

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `SizeKind` + `SizeHint` (interface method) | `pkg/sampler/sampler.go` | modified (interface) |
| `SizeHint` impls ×6 | `pkg/sampler/{configs,folds,driver}.go`, `run_test.go` | modified |
| `ProgressEvent[P,O]` | `pkg/sampler/run.go` | new |
| `Run` (progress param) | `pkg/sampler/run.go` | modified |
| `progressState` + `progressLine` | `cmd/metis/progress.go` | new |

- **`SizeKind`** — `const (SizeExact SizeKind = iota; SizeBudget; SizeUnknown)`. `SizeHint(s S) (total int, kind SizeKind)` joins the `Sampler[S,P,O,R]` interface — the ONLY per-sampler bit (the varying n pushed into the interface, not a runner branch — the issue's design). All four production samplers return exact: `GridConfigs → (len(g.Points), SizeExact)`, `FixedKFolds → (f.K, SizeExact)`, `CVDriver → (d.K, SizeExact)`, `SingleDriver → (1, SizeExact)`. The two test samplers in `run_test.go` (`countSampler`, `stuckSampler`) also gain trivial impls (interface compile break — enumerated, not discovered). `Run` captures `SizeHint(s₀)` once after `Init` and stamps it into every event; a future adaptive sampler whose budget moves mid-run can revisit then (YAGNI now, seam documented).
  - **DRY rationale:** `cmd/metis` could compute `∏grid` from the shape, but that duplicates sampler knowledge and breaks the moment a non-grid sampler exists; `SizeHint` is the single source the renderer trusts.
- **`ProgressEvent[P,O]`** — `struct { K, Total int; Kind SizeKind; Point P; Out O }`. K is 1-based and monotone (Run's internal mutex serializes increment + callback — the callback need not be reentrant, must be fast). Generic, so each Run call site's closure receives TYPED payloads (`shape.Point`+`MeanSE` at the sweeper level, `OuterFoldPoint`+`float64` at the driver level) — no `any`, no type switches in the sink (ARCH-PURE-adjacent: the seam carries data, the consumer owns interpretation).
- **`Run`** — signature gains the callback: `Run[S,P,O,R](ctx Ctx, smp Sampler[S,P,O,R], runPoint func(P) O, exec Exec[P,O], progress func(ProgressEvent[P,O])) R`. `nil` → no wrapping, zero overhead (the exact current loop). Non-nil → `runPoint` is wrapped: run, lock, k++, fire, unlock, return. **Single API, no sugar wrapper** — all **19** call sites (grep-verified: 4 production in `cmd/metis/sweep.go`; tests: `run_test.go` ×5, `configs_test.go` ×4, `driver_test.go` ×3, `exec_test.go` ×2, `folds_test.go` ×1) updated mechanically (tests pass `nil` except the new progress tests).
- **`countSampler` refactor (required, stated up front):** the test fake hardcodes `pts: []int{1,2,3}` in `Init` with no size field — the new ParExec progress test needs 32 points. Parameterize it: add an `n int` field, `Init` builds `1..n` (default 3 when `n==0` so the existing `&countSampler{}` constructions keep their sum-6 behavior unchanged).
- **`progressState` / `progressLine`** — the pure half of the sink: a plain struct (per-level `k/total/kind`, completed outer scores, per-config display-best) plus `progressLine(st progressState) string` rendering the aggregated line. Pure, table-tested without IO or clock. Formats: nested `outer 1/3 · configs 84/216 · folds 421/1080 · est 0.8283 ± 0.0140` (est `—` until ≥1 outer fold lands; SE omitted until n≥2); flat — **which since metis#32 runs iff exactly 1 config** (plan-review correction; the spec's `47/99` example is the pre-#32 world) — degenerates to `folds 3/5 · score 0.8340` (the one config's running fold mean; `configs 1/1` adds nothing — drop it; label `score`, not `best`, since there is nothing to be best OF). Kinds render `k/n` (exact), `k/≤n` (budget), `k/?` (unknown). **Totals are SEEDED AT WIRING TIME** (plan-review finding): learned-from-the-stream totals arrive only with a level's first completion — for the driver level that is the first *completed outer fold*, near the END of a parallel run, starving the display (and #38's ETA) of `n` for most of the run. `cmd/metis` constructs the samplers, so it calls `SizeHint` directly at sink construction (`CVDriver{K: runFolds}`, `GridConfigs{Points: configPts}`, `FixedKFolds{K: k}` on their `Init(ctx)` states) — SizeHint stays the single source (ARCH-DRY), no shape math duplicated, and the first line already shows full denominators. Event-carried totals then serve adaptive samplers whose n moves (rendered if they differ). **Aggregate counters are SINK-OWNED increments** — never `ev.K` (each concurrent resample Run instance counts its own 1..k; explicit so the implementer doesn't reach for it).

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `sweepProgress` (sink) | `cmd/metis/progress.go` | new | stdout via `ss.out` + injected clock |
| Run call sites ×4 | `cmd/metis/sweep.go` | modified | the progress seam |

- **`sweepProgress`** — `newSweepProgress(out io.Writer, now func() time.Time, direction string, totals progressTotals)` (totals seeded from direct `SizeHint` calls, above). The event surface is **per-pass hooks** (the #38 identity fix): `forPass(outer int) passHooks` returns `passHooks{config func(ProgressEvent[shape.Point, MeanSE]), fold func(ProgressEvent[FoldPoint, FoldOutcome])}` — closures bound to their outer-fold index (`-1` = the flat path's single pass). Driver level: `driverEvent(ev ProgressEvent[OuterFoldPoint, float64])`. One mutex (events arrive concurrently from ParExec goroutines across sibling passes); folds into `progressState`; **throttled emit**: write `progressLine` when `now() - lastEmit ≥ 1s`, ALWAYS on a driver-level (outer fold) completion, and once at `finish()` (the terminal state line). After the nested path's `firstErr` latches, remaining outer closures return 0 — `runNestedCV`'s wiring skips `driverEvent` once errored (don't fold sentinel zeros into the displayed est). Writes go to `ss.out`, which `runExperiment` already wraps in `syncWriter` under parallelism (#31) — no interleaving corruption; lock order is strictly Run-mu → sink-mu → syncWriter-mu (review-verified, no cycle). Plain appended lines, no `\r`/escape codes (non-TTY-safe by construction; the TTY board is #38).
  - **Injected into:** `shapeSweep` as a `prog *sweepProgress` field (nil ⇒ all methods no-op via nil-receiver guard — the single-run/non-sweep path stays silent). `sweepPass` gains a `hooks passHooks` field, set where the pass is created (`runOuterFold` with its `i`; the flat path with `-1`).
  - **Clock:** `ss.now` (already injected through `runOpts` — controllable-time posture; tests drive a fake clock, never sleep).
- **Run call sites** (all in `sweep.go`): `runSweeper` passes `pass.hooks.config` to the GridConfigs Run and `pass.hooks.fold` to the FixedKFolds Run; `runNestedCV` passes the error-gated `driverEvent` wrapper to the CVDriver Run; the flat path's SingleDriver Run passes `nil` (a 1/1 level with nothing to report — its sweeper/resample hooks carry the signal). #30's sink AGGREGATES across passes (the single-line mental model); the per-pass hook binding exists so #38 can add per-fold rows behind the same API without rewiring.
- **No new external surface**: no fake needed; e2e rides the existing toy-pipeline workspace.

**Test surface:** `pkg/sampler`: SizeHint table test (6 impls); Run+progress — recording callback under `SeqExec` (k strictly 1..n in order, Total/Kind correct, Point/Out are the completed pair) and under `ParExec` (exactly n events, k values = {1..n} each seen once, monotone in arrival order — the mutex guarantee), `nil` callback exercises the unwrapped path (existing tests, now passing nil, ARE that regression). `cmd/metis`: `progressLine` table test (nested/flat/est-thresholds/kinds); `sweepProgress` fake-clock test (throttle: events 200ms apart → emits at ≥1s boundaries + always on outer completion + finish); e2e: the real toy nested + flat runs assert a `metis: progress` line with the correct final `k/n` appears in captured output (done-when 3: verified on a real run, not just asserted).

---

## Tasks

Single-pass close for #30, plain checkboxes (§3). (#38 is a separate issue/plan consuming this seam.)

### Task 1: `SizeHint` on the Sampler interface (pkg/sampler)

**Files:** Modify `pkg/sampler/sampler.go`, `configs.go`, `folds.go`, `driver.go`, `run_test.go` (test fakes). Test: `pkg/sampler/sizehint_test.go` (new).

- [ ] **Step 1: failing test** — `sizehint_test.go`:

```go
func TestSizeHints(t *testing.T) {
	grid := GridConfigs{Points: make([]shape.Point, 7)}
	if n, k := grid.SizeHint(grid.Init(Ctx{})); n != 7 || k != SizeExact {
		t.Errorf("grid: (%d,%v)", n, k)
	}
	folds := FixedKFolds{K: 5}
	if n, k := folds.SizeHint(folds.Init(Ctx{})); n != 5 || k != SizeExact {
		t.Errorf("folds: (%d,%v)", n, k)
	}
	cv := CVDriver{K: 3}
	if n, k := cv.SizeHint(cv.Init(Ctx{})); n != 3 || k != SizeExact {
		t.Errorf("cv: (%d,%v)", n, k)
	}
	var sd SingleDriver
	if n, k := sd.SizeHint(sd.Init(Ctx{})); n != 1 || k != SizeExact {
		t.Errorf("single: (%d,%v)", n, k)
	}
}
```

- [ ] **Step 2: verify FAIL** (`undefined: SizeExact` / missing method).
- [ ] **Step 3: implement** — in `sampler.go`:

```go
// SizeKind classifies a Sampler's SizeHint total: exact (a static point-set),
// budget (an upper bound, e.g. maxEvals), or unknown (open-ended adaptive).
// The renderer shows k/n, k/≤n, k/? respectively (metis#30).
type SizeKind int

const (
	SizeExact SizeKind = iota
	SizeBudget
	SizeUnknown
)
```

Add to the `Sampler` interface: `SizeHint(s S) (total int, kind SizeKind)` (doc: the per-sampler "n" — the only progress bit pushed into the interface; called once by Run on the initial accumulator). Impls: grid `return len(g.Points), SizeExact`; folds `return f.K, SizeExact`; CVDriver `return d.K, SizeExact`; SingleDriver `return 1, SizeExact`; `run_test.go` fakes: `countSampler → (c.n, SizeExact)`, `stuckSampler → (0, SizeUnknown)`.

- [ ] **Step 4: verify PASS** — `go test ./pkg/sampler/`.
- [ ] **Step 5: commit** — `#30: SizeHint(total, kind) on the Sampler interface — the per-sampler n`.

### Task 2: `Run` fires progress at point completion

**Files:** Modify `pkg/sampler/run.go` (+ the 19 call sites: `cmd/metis/sweep.go` ×4, sampler tests ×15). Test: extend `pkg/sampler/run_test.go`.

- [ ] **Step 1: failing tests** — in `run_test.go` (uses the existing `countSampler`, whose points are ints):

```go
// Run fires progress ONCE PER COMPLETED POINT (live — not per Tell, which under
// ParExec + one-batch static samplers happens only at batch end; metis#30 revision),
// with monotone 1-based k and the SizeHint total/kind stamped on every event.
func TestRunProgress_SeqOrdered(t *testing.T) {
	var evs []ProgressEvent[int, int]
	smp := &countSampler{n: 3}
	Run[countState, int, int, int](Ctx{}, smp, func(p int) int { return p * 10 },
		SeqExec[int, int], func(ev ProgressEvent[int, int]) { evs = append(evs, ev) })
	if len(evs) != 3 {
		t.Fatalf("want 3 events, got %d", len(evs))
	}
	for i, ev := range evs {
		if ev.K != i+1 || ev.Total != 3 || ev.Kind != SizeExact {
			t.Errorf("event %d: %+v", i, ev)
		}
		if ev.Out != ev.Point*10 {
			t.Errorf("event %d: (Point,Out) must be the completed pair: %+v", i, ev)
		}
	}
}

func TestRunProgress_ParallelMonotoneComplete(t *testing.T) {
	var mu sync.Mutex
	var ks []int
	smp := &countSampler{n: 32}
	Run[countState, int, int, int](Ctx{}, smp, func(p int) int { return p },
		ParExec[int, int], func(ev ProgressEvent[int, int]) { mu.Lock(); ks = append(ks, ev.K); mu.Unlock() })
	if len(ks) != 32 {
		t.Fatalf("want 32 events, got %d", len(ks))
	}
	for i, k := range ks { // Run's mutex serializes increment+fire → arrival order IS k order
		if k != i+1 {
			t.Fatalf("k must arrive monotone 1..n, got %v", ks)
		}
	}
}
```

(The nil-callback no-op is pinned by every EXISTING Run test passing `nil` — the unwrapped path.)

- [ ] **Step 2: verify FAIL** (signature mismatch).
- [ ] **Step 3: implement** — `run.go`:

```go
// ProgressEvent is metis#30's per-completion progress payload: the 1-based monotone
// completion count K against the level's SizeHint (Total, Kind), plus the completed
// (Point, Out) pair — typed per Run instantiation, so a consumer's closure gets its
// own level's payloads without type switches. Fired at POINT COMPLETION (not Tell:
// under ParExec + one-batch static samplers every Tell lands at batch end — useless
// live). Exactly one event per point (the same count as one Tell per point).
type ProgressEvent[P, O any] struct {
	K, Total int
	Kind     SizeKind
	Point    P
	Out      O
}
```

In `Run`: after `s := smp.Init(ctx)`, capture `total, kind := smp.SizeHint(s)`; if `progress != nil`, wrap:

```go
	if progress != nil {
		var mu sync.Mutex
		k := 0
		inner := runPoint
		runPoint = func(p P) O {
			o := inner(p)
			mu.Lock()
			k++
			progress(ProgressEvent[P, O]{K: k, Total: total, Kind: kind, Point: p, Out: o})
			mu.Unlock()
			return o
		}
	}
```

(Wrapping BEFORE the loop; the callback runs on exec goroutines, serialized by the mutex — document: keep it fast, it holds completions.) Update all 19 call sites (`, nil` in tests; the 4 `cmd/metis/sweep.go` sites take real closures in Task 3 — pass `nil` in this commit to keep it compiling).

- [ ] **Step 4: verify PASS** — `go test ./pkg/sampler/ ./cmd/metis/` (whole module compiles; behavior unchanged with nil).
- [ ] **Step 5: commit** — `#30: Run fires ProgressEvent[P,O] at point completion (live under ParExec)`.

### Task 3: the `cmd/metis` sink + aggregated line

**Files:** Create `cmd/metis/progress.go` + `cmd/metis/progress_test.go`. Modify `cmd/metis/sweep.go` (shapeSweep field + 4 closures).

- [ ] **Step 1: failing tests** — `progress_test.go`, three groups:
  1. `progressLine` table test (pure): nested pre-outer (`est —`), nested 1 outer (est, no SE), nested ≥2 outer (`± SE`), flat (1-config: `folds 3/5 · score 0.8340`), unknown kind (`k/?`), budget kind (`k/≤n`).
  2. `sweepProgress` fake-clock throttle: a `now` returning scripted times; feed 10 fold events 200ms apart → exactly the ≥1s-boundary lines emitted; an outer (driver) event emits immediately regardless; `finish()` emits the final line once.
  3. Concurrency smoke: `-race`-clean under parallel event fire (a `t.Parallel`-style goroutine fan-in).
- [ ] **Step 2: verify FAIL.**
- [ ] **Step 3: implement** `progress.go` — `progressState` (sink-owned counters per level + `outerScores []float64` + the flat config's running fold scores + direction), `progressLine(st) string` (pure), `sweepProgress` (mutex + out + now + lastEmit + seeded `progressTotals`; nil-receiver-safe `forPass`/`driverEvent`/`finish`; `est` = mean±SE over `outerScores` — computed locally in the sink with a comment pointing at `sampler.MeanSE` (display-only; do NOT export new sampler surface just for display)). Wire `sweep.go`: `ss.prog = newSweepProgress(out, now, direction, seededTotals(ctx, runFolds, configPts, k))` in `runShapeSweep` (both paths); `pass.hooks = ss.prog.forPass(i / -1)` at pass creation; the error-gated `driverEvent` at the CVDriver site; `ss.prog.finish()` before the terminal report lines.
- [ ] **Step 4: verify** — `go test ./cmd/metis/ -race -run 'TestProgress'` then the full package.
- [ ] **Step 5: commit** — `#30: throttled aggregated progress line over the completion events`.

### Task 4: fixture-sweep output pins (+ real-run evidence at close)

**Plan-review correction — the original premises were FALSE:** `TestToyPipeline_EndToEnd` runs a plain `type: experiment` single run (never enters `runShapeSweep` — the nil-`prog` silent path) AND passes `out: io.Discard`; the nested e2es drive `foldFakeExec` (no uv). There is NO real-uv sweep e2e in the repo. Done-when 3 explicitly allows "real (or fixture)" — fixture pins here, the REAL-run evidence lands in Task 5's `--verified` (a live kbench sweep).

**Files:** extend `cmd/metis/nestedcv_e2e_test.go` (`TestNestedCV_ProducesHonestEstimateNoShip` — already captures `&out`) and the flat-sweep test in `cmd/metis/shapesweep_test.go`.

- [ ] **Step 1:** Assert in the nested fixture test: ≥1 `metis: progress` line; the final one carries `outer <m>/<m>` and a numeric `est`. In the flat fixture test: final line carries `folds <k>/<k>` and `score`. NOTE: these fixtures run with `fixedNow()` — a frozen clock never crosses the 1s throttle, so ONLY the always-emit lines (driver-level completions + `finish()`) appear; assert those, and the throttle behavior stays pinned by Task 3's scripted-clock unit test.
- [ ] **Step 2:** run both — PASS (fake-exec, fast).
- [ ] **Step 3: commit** — `#30: fixture sweeps pin the progress lines (nested + flat)`.

### Task 5: docs + close

- [ ] atlas/index.md: the progress seam (SizeHint + completion-fired ProgressEvent + the cmd/metis sink; #38 consumes the same seam for the TTY board).
- [ ] Issue `## Revisions`: BOTH spec deviations — the Tell→completion timing revision (rationale above) AND the flat-format correction (the spec's `47/99 · best` example is pre-#32; flat now runs iff 1 config → `folds k/n · score`), so the issue file doesn't contradict the shipped renderer.
- [ ] kbench RUNBOOK: one line — sweeps now print `metis: progress …` (babysitting recipe supersession note pointing at #38 for throughput/ETA).
- [ ] **Real-run evidence**: build the binary and run a real kbench sweep (the smoke shape `titanic-sweep-smoke.md` with BLAS pins — cheap, real uv/Python leaves) redirecting to a file; confirm live progress lines with correct totals + est appear and the output is escape-code-free. This is done-when 3's "real" half.
- [ ] `sdlc close --issue 30 --verified '<evidence incl. the real smoke-sweep output>'`.

## Verification (Done-when → checks)

| Done-when | Check |
|---|---|
| `SizeHint` grid returns exact product (unit) | Task 1 test (grid = configs; the config×fold product is the composed display total — the sweeper×resample levels each report their own exact n) |
| progress once per Tell-equivalent, monotone k, correct total/kind; nil = no-op | Task 2 tests (Seq ordered + Par monotone-complete; existing nil-passing tests pin the no-op) |
| `metis run` prints live k/n + running best (flat) and `outer j/k · est mean±SE` (nested), verified on a real (or fixture) sweep | Task 3 renderer/throttle tests + Task 4 fixture-sweep output pins + Task 5's REAL live-kbench-sweep output in `--verified` |
