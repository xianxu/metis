---
id: 000043
status: working
deps: []
github_issue:
created: 2026-07-14
updated: 2026-07-16
estimate_hours: 4.74
started: 2026-07-16T12:57:07-07:00
---

# leaf scheduler: depth-first run priority so cold-cache sweeps reach trains early

## Problem

On a COLD cache, a nested sweep's ~2k run closures all start concurrently and the shared leaf
semaphore serves steps roughly FIFO — so a run's `features` queues behind every other run's
`cv-split`, and execution degenerates into phase WAVES: all cv-splits, then all features, then
all trains (observed on the kbench#9 decision run 2026-07-14: 2,160/2,160 cv-splits done,
1,315 features, 0 trains after ~10 min — looked like a hang; zero records complete until wave 3,
so run-level progress reads as frozen). On a warm cache the waves collapse and trains appear
early, which is why this wasn't seen before. Depth-first (finish admitted runs before admitting
new ones) reaches complete records ~immediately, makes progress meaningful, and bounds
in-flight run state; it also pairs with #38's moving-average runs/sec board (wave 1-2 have zero
"runs/sec" by construction).

## Spec

(at claim) Options: bound the number of ADMITTED run closures (e.g. 2×parallel) with a run-level
semaphore above the leaf semaphore (simplest — preserves leaf fairness within admitted runs);
or a priority leaf queue keyed by run progress. Keep byte-determinism of artifacts/ledger
(ordering already normalized by sortPointRuns).

### Approved design — bounded whole-run admission

Use a distinct, shared **run-admission gate** in addition to the existing leaf-subprocess
semaphore:

- `runExperiment` creates the gate once when `maxParallel > 1`, with capacity
  `2 × maxParallel`, and threads it through copied `runOpts`. Serial runs have no gate. There is
  no second CLI/config knob: `--parallel` remains the one operator-facing budget.
- Every concrete `runResolvedExperiment` acquires admission before printing the run-start line,
  creating run directories/records, or executing a step, and releases with `defer` on every return.
  This is the single choke point shared by inner fold runs, outer scoring, preamble work, and future
  concrete runs (ARCH-DRY, ARCH-PURPOSE).
- The gate shares one **runExperiment-scoped abort latch** across every concrete run in the flat or
  nested sweep; it is not scoped to an individual outer-fold pass. Each call supplies its contextual
  run label (config/fold, outer score, or preamble). The admitted wrapper checks the latch before any
  run side effect, executes the concrete run, and publishes its final contextual error to the
  set-once latch **before** releasing admission. Thus a queued sibling cannot acquire in the gap
  between failure and publication; the first contextual error remains authoritative globally.
- A run that acquires after failure returns a typed `errRunAborted` without logs, files, records, or
  activity events. `sampler.Run` may still require a shape-preserving zero value for that batch slot,
  but it is explicitly non-authoritative: fold/config/driver progress hooks and config/point
  accumulation check the shared latch; each sampler level checks it before further scoring,
  persistence, or reporting; and the top-level result is discarded in favor of the original error.
  Tests pin that aborted slots affect no persisted rows, displayed score/estimate, completion count,
  or throughput signal.
- The run gate and leaf semaphore stay independent. No outer/config orchestration closure holds
  admission while synchronously awaiting a child that also needs admission. A concrete admitted run
  may wait for leaf capacity, but a leaf holder never waits for run admission, so the lock order is
  acyclic and nested fan-out cannot semaphore-deadlock (ARCH-PURE).
- Admission order is deliberately not a stable artifact/API. The guarantee is bounded whole-run
  breadth and early completions, not a byte-defined goroutine schedule. `ParExec` continues returning
  outputs by input index; `Run.Tell` stays proposal-ordered; completion-order side records remain
  sorted before persistence. Serial and parallel manifests/ledgers must remain byte-identical.
- Capacity `2n` is the utilization hedge: admitted runs can be between steps or serving cache hits
  while the `n` leaf slots remain available to useful subprocess work. A priority leaf scheduler is
  explicitly out: it leaves all run state admitted and adds starvation/cancellation policy without
  better serving the observed cold-wave problem.

## Done when

- With parallelism `n`, no more than `2n` concrete runs are admitted globally across nested outer
  folds/configs/inner folds, while subprocess occupancy remains capped at `n` by the existing leaf
  semaphore.
- A cold scripted sweep reaches a completed train/inner-CV run before the first step has started for
  every run (the old phase-wave behavior fails this test).
- A first run failure prevents queued-not-admitted runs anywhere in the experiment from creating
  state or executing steps, publishes the contextual error before admission release, and surfaces
  that original error without bogus rows, counters, throughput, score, or estimate.
- In TUI mode, failure stops ticker-driven progress observations and atomically erases the pinned
  board without retaining a frame that deferred close could redraw; only pending ordinary output
  and terminal cleanup escapes after publication.
- Nested execution completes without deadlock. Serial versus admitted-parallel runs persist
  byte-identical deterministic manifests and ledger rows; timing-bearing run records are
  semantically equal apart from their expected timestamps/durations.
- Full `go test ./... -race` and a disposable cold real-sweep smoke show early inner-run progress and
  sane throughput.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.5
item: issue-spec design=0.80 impl=0.12
item: greenfield-go-module design=0.30 impl=0.32
item: cross-cutting-refactor design=0.20 impl=0.20
item: cross-cutting-refactor design=0.20 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: tui-screen design=0.10 impl=0.20
item: atlas-docs design=0.06 impl=0.08
item: milestone-review design=0.10 impl=0.20
design-buffer: 0.15
total: 4.74
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. The `issue-spec` row includes the paired #43/#49
design and fresh-eyes convergence already performed after claim. The controller is a greenfield
bounded-concurrency concern; the two refactor rows separately cover the concrete-run boundary and
the nested reduction/progress shell; the two smaller-module rows cover deterministic concurrency
regressions and the isolated real-process smoke. Implementation values use v3.1's 40%-of-v2
AI-paired scale, multiplied by 1.5 for novel-but-bounded concurrency. The thorough reviewed plan
earns the 15% design buffer.
The calibration source is currently marked stale by `sdlc estimate-source`, so this derivation
is structurally current but provisionally calibrated.

## Plan

Durable implementation detail: [workshop/plans/000043-leaf-sched-depth-first-plan.md](../plans/000043-leaf-sched-depth-first-plan.md).

- [x] Chunk 1 — build the race-safe `runControl` admission/failure primitive with deterministic handoff tests.
- [x] Chunk 2 — integrate the controller at the concrete-run boundary and linearize every sampler observation/reduction against global failure.
- [x] Chunk 3 — prove early completion, independent nested caps, deterministic records, full race safety, atlas accuracy, and an isolated real-process cold smoke.

## Log

### 2026-07-14

### 2026-07-16 — paired #43/#49 design approved
- Chose bounded whole-run admission at the shared concrete-run choke point over a local fold-only
  gate or priority leaf queue. Co-designed with #49 so the board reports the earlier completions
  truthfully; implementation/review boundaries remain separate (#43 merges first).

### 2026-07-16 — implementation plan approved
- Wrote and independently reviewed all three chunks of the durable #43 plan: controller primitive,
  concrete-run/sampler integration, and scheduling/determinism/real-process verification. Estimate
  derives from the provisional v3.1 calibration; delivery remains one atomic close boundary.

### 2026-07-16 — change-code estimate correction
- The gate's plan-quality judge found the original 1.94h derivation materially under-decomposed the
  concurrency integration and verification surface. Re-derived 4.33h from eight explicit Method A
  work units with a 1.5 novel-but-bounded familiarity multiplier; no scope or architecture changed.

### 2026-07-16 — change-code TUI consumer correction
- The second implementation-entry review found two post-failure display consumers outside the
  sampler reductions: the board ticker and deferred `boardWriter.close`. Added an explicit board
  abort path, controlled-tick race proof, and a ninth Method A unit; the estimate is now 4.74h.

### 2026-07-16 — Chunk 1 controller complete
- Added the optional `2n` admission gate and set-once first-error latch with deterministic tests for
  publication-before-release, serial cancellation, concurrent authority, and late-success discard.
- Focused `-race -count=20`, spec review, and code-quality review passed. Review strengthened the
  suite with mutation-sensitive proofs that admission precedes the first error check and that the
  winner-only publication hook remains inside the controller mutex.

### 2026-07-16 — Task 2 concrete-run boundary complete
- Wrapped the shared concrete runner before path, cache, directory, output, executor, or record side
  effects; all fold, outer-score, and preamble calls now attach complete controller-owned context.
- Focused race tests plus spec and code-quality review passed. A review-driven cache-enabled mutation
  test proves pre-admission cancellation cannot even initialize `.metis-cache`.

### 2026-07-16 — Chunk 2 global abort integration complete
- Replaced pass-local and nested error ownership with the sweep controller, linearized every progress,
  accumulation, scoring, persistence, and report consumer, and preserved the first contextual cause.
- Extended the boundary through the live board: ticker refreshes are gated, joined before cleanup,
  and an error erases and forgets the frame before deferred close. Controlled pre/post-failure tick
  tests and the focused `-race -count=20` suite passed, followed by full spec and quality approval.

### 2026-07-16 — Task 4 scheduling acceptance complete
- Proved a flat cold sweep completes a train before its fifth admission, nested run/leaf peaks remain
  independently capped, every token releases, and serial/parallel manifests, ledgers, runs, and
  records remain byte- or semantically identical as appropriate.
- Review hardened the proof against zero-hook false passes, hook panics, and payload/path identity
  swaps. The scheduling/cancellation subset passed `-race -count=10` and full race verification.

### 2026-07-16 — Task 5 cold real-process acceptance
- Added the credential-free three-config, two-fold `toy-sweep-smoke` fixture; its dry run reported
  nested CV with `2 × (3 configs × 2 inner folds)` and the expected `C=0.5,1,2` grid.
- In a no-hardlink temporary clone with all writable caches redirected below the temporary root, the
  first completed train appeared on log line 25 before the fifth `cv-split` start on line 32. The
  nested estimate appeared on line 60 and the seven-row completion summary on line 62: seven real
  trains finished in 44s (~9.5 trains/min). Source status and `refs/metis/*` were byte-identical
  before and after, and the temporary tree was removed. The isolated build also cloned the declared
  `../ariadne` replacement beside Metis; the first setup attempt had exposed that missing dependency
  before the sweep began.
- Focused scheduling/cancellation tests passed `go test ./cmd/metis -race -run
  'Test(Sweep_ColdAdmissionCompletesTrainBeforeFifthAcquire|NestedCV_PeakConcurrencyWithinCap|NestedCV_FirstFailureStopsAllObservableWork|RunControl)'
  -count=10`; `go test ./cmd/metis -race -count=1`, `go test ./... -race -count=1`, and
  `git diff --check` also passed with no race or whitespace report. `sdlc issue validate --issue 43`
  passed. Task 4's deterministic acceptance covers byte-identical serial/parallel manifests and
  ledgers plus semantically equal run records.
- Quality-review rerun pinned the no-hardlink Ariadne clone at
  `a24643e566eb67ebbc69376126000e469761f09a` before making the no-hardlink Metis clone. The durable
  recipe passed unchanged: first train line 25 preceded fifth `cv-split` line 32; the estimate and
  seven-row summary were on lines 60 and 62; seven trains completed in 45s (~9.3/min). Source status
  and `refs/metis/*` remained byte-identical, and the temporary clone/cache root was removed.

## Revisions

### 2026-07-16 — fresh-eyes spec review
- Made cancellation experiment-wide, required failure publication before admission release, and
  defined how typed aborted slots are excluded from every observable/reduction seam. Narrowed the
  determinism claim for timing-bearing run records and added a barrier-testable failure handoff.

### 2026-07-16 — durable plan review
- Replaced the generic two-line implementation sketch with the reviewed three-chunk plan and added
  the reconciled estimate. Review tightened controller hook semantics, atomically linearized
  observations against failure, and made cold-order/throughput/isolation proof executable.

### 2026-07-16 — estimate-gate revision
- Raised the estimate from 1.94h to 4.33h after the implementation-entry judge identified missing
  primitive decomposition for the controller, two cross-cutting integrations, deterministic race
  suite, and real-process smoke. The approved behavior and one-boundary delivery plan are unchanged.

### 2026-07-16 — TUI failure-path revision
- Extended the plan's global-abort boundary through the live progress board: ticker observations
  linearize through `runControl`, an error return discards and erases the stored frame before the
  top-level close, and a publication-barrier test drives a post-failure tick deterministically.
- Raised the estimate from 4.33h to 4.74h for the added TUI integration and race-test surface.
