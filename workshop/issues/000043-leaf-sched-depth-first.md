---
id: 000043
status: working
deps: []
github_issue:
created: 2026-07-14
updated: 2026-07-16
estimate_hours:
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
- Nested execution completes without deadlock. Serial versus admitted-parallel runs persist
  byte-identical deterministic manifests and ledger rows; timing-bearing run records are
  semantically equal apart from their expected timestamps/durations.
- Full `go test ./... -race` and a disposable cold real-sweep smoke show early inner-run progress and
  sane throughput.

## Plan

- [ ] Implement the approved run-admission design through a durable TDD plan.
- [ ] Verify nested cap, early completion, abort behavior, determinism, race safety, and real cold smoke.

## Log

### 2026-07-14

### 2026-07-16 — paired #43/#49 design approved
- Chose bounded whole-run admission at the shared concrete-run choke point over a local fold-only
  gate or priority leaf queue. Co-designed with #49 so the board reports the earlier completions
  truthfully; implementation/review boundaries remain separate (#43 merges first).

## Revisions

### 2026-07-16 — fresh-eyes spec review
- Made cancellation experiment-wide, required failure publication before admission release, and
  defined how typed aborted slots are excluded from every observable/reduction seam. Narrowed the
  determinism claim for timing-bearing run records and added a barrier-testable failure handoff.
