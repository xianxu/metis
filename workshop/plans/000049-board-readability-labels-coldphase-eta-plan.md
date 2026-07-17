# Board Readability: Typed Activity, Cold-Phase Truth, and Stable ETA — Implementation Plan

> **For the implementer:** Execute this plan with `superpowers-executing-plans`; use TDD for every
> behavior change. The issue has one delivery boundary, so do not add milestone tags or run
> `sdlc milestone-close`; cross the mandatory fresh-context review once with `sdlc close`.

**Goal:** Make the sweep board explain cold work truthfully, use unambiguous flat/nested vocabulary,
and withhold rate/ETA until time-based evidence is stable without weakening stall visibility.

**Architecture:** A typed activity emitter connects two concrete facts to `sweepProgress`: successful
final-executor steps (outside cache) and successfully executed, durably persisted concrete runs.
`sweepProgress` is the sole synchronized reducer; pure bounded windows derive smoothed occupancy,
event-time rate readiness, decay, and last-run age. `progressCore` remains the shared semantic source
for plain and TUI output, while `renderBoard` only formats snapshots. Activity publication is gated by
`runControl` before taking the progress mutex, preserving the established controller → progress →
writer lock order and preventing post-failure repaint (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

**Tech stack:** Go, standard library, existing metis run/sweep/progress/board abstractions, Go tests,
and the kbench Markdown RUNBOOK.

## Core concepts

### PURE entities and transforms

| Concept | Responsibility | Invariants |
|---|---|---|
| `activityEvent` | Immutable successful activity fact with kind, typed run role/identity, and injected-clock timestamp. | Failed work creates no event; timestamps describe completion, not callback delivery. |
| `runRole` | Distinguish nested inner-CV, flat CV, preamble, outer score, and ineligible/no-role runs. | Only inner-CV and flat CV are rate/counter eligible. |
| `occupancyWindow` | Retain the last four 500ms occupancy samples and return their rounded mean. | Event count cannot affect the result; capacity is four. |
| `movingRate` | Retain the latest 64 eligible completion times in event-time order and derive readiness/rate from `now`. | Ready only at n≥16 and span≥15s; rate is `(n-1)/(now-oldest)`; reversed delivery is deterministic. |
| `activitySnapshot` | Read-only facts consumed by formatting: steps, max step time, eligible runs, max run time, smoothed slots, and optional rate. | Last times are maxima; startup ends on the first eligible run. |

### INTEGRATION boundaries

| Boundary | Responsibility | Failure semantics |
|---|---|---|
| `activityExecutor` | Decorate the final cache-aware executor and emit one step event after a successful real execution or cache hit. | Inner error is returned unchanged and emits nothing. |
| `runResolvedExperiment` activity publication | Emit the typed run event only after execution and required `runs/<id>/{run,record}.json` persistence succeed. | Execution failure, `run.json`/`record.json` failure, or provenance assembly failure emits no successful-run event; best-effort capture is not a success gate. |
| `runControl`-gated emitter | Linearize all step and run activity against fatal failure before calling `sweepProgress`. | Rejected after failure; never acquire controller state while holding progress state. |
| `sweepProgress` | Synchronize activity/tick reduction and publish immutable board snapshots. | Short callbacks; non-sweep callers receive a no-op emitter. |
| `renderBoard` / `progressCore` | Apply shared vocabulary and factual startup/mature wording to snapshots. | No diagnosis such as “not hung”; width, cadence, failure flush, and terminal cleanup remain intact. |
| kbench RUNBOOK | Document the shipped board contract using the exact operator-facing nouns. | Full peer commit SHA is recorded in issue #49 before close. |

## Chunk 1: Typed activity at concrete success seams

### Task 1: Define activity facts and the final-executor decorator

**Files:**
- Modify: `cmd/metis/run.go`
- Create or modify: `cmd/metis/activity_test.go`
- Modify: `cmd/metis/caching_test.go`

1. Write failing table tests proving the decorator emits exactly one timestamped step-success event
   after a successful inner executor, emits none on error, and preserves the exact result/error.
2. Add a cache wiring regression: one cold execution and its warm cache hit each produce one step
   event. Assert the decorator is outside the cache wrapper, not merely that the cold path works.
3. Introduce the smallest typed `activityEvent`/kind/role vocabulary and a no-op-capable emitter.
   Implement `activityExecutor` around the final executor built in `runResolvedExperimentAdmitted`.
   It must publish through the same `runControl`-gated emitter used by successful-run events, so a
   late successful step callback after sibling failure is rejected before progress repaint.
4. Run the focused tests and keep identities/timestamps injected; do not parse step output or inspect
   cache implementation details.

### Task 2: Publish successful concrete-run events at the persistence boundary

**Files:**
- Modify: `cmd/metis/run.go`
- Modify: `cmd/metis/run_test.go`

1. Extend the successful-run test with an activity callback that observes both required
   `runs/<id>/run.json` and `runs/<id>/record.json` artifacts already persisted when the event arrives.
2. Pin negative paths: a failed execution that successfully writes its failure record emits no run
   event, and a forced required-persistence failure (for example, a directory at the record path)
   emits no run event.
3. Add the run role to `runOpts` and publish only after `runErr == nil` plus required persistence.
   Preserve best-effort capture behavior: capture failure must not retroactively make a successful
   run ineligible unless the current contract already treats that artifact as required.
4. Route successful-run publication through the shared `runControl`-gated emitter before the progress
   callback. Add barrier regressions showing a sibling fatal failure prevents both a later step event
   and a later run event from repainting without introducing controller↔progress lock inversion.

### Task 3: Assign roles at every sweep call site

**Files:**
- Modify: `cmd/metis/sweep.go`
- Modify: `cmd/metis/run.go`
- Modify: `cmd/metis/run_test.go`

1. Write a call-site trace test that distinguishes flat CV, nested inner-CV, nested preamble, and
   outer-score runs and proves only the first two are eligibility candidates.
2. Add `sweepPass.runRole`; set it at flat pass construction and nested pass construction, then copy
   it into `pointOpts.runRole` immediately before `runPipelineFold` calls `runResolvedExperiment`.
   Assign preamble and outer-score roles at their direct launch sites. Keep plain `metis run` and
   `metis select --promote` explicitly no-op/ineligible; document the bypass rather than silently
   relying on a zero value.
3. Assert emitted run-event roles from the concrete call paths, not just enum eligibility. Run focused
   tests plus `go test ./cmd/metis -run 'Activity|Cache|RunResolved|RunControl' -race`.

## Chunk 2: Deterministic telemetry reduction and board semantics

### Task 4: Replace callback-count rate sampling with event-time reduction

**Files:**
- Modify: `cmd/metis/progress.go`
- Modify: `cmd/metis/progress_test.go`

1. Write failing pure tests for eligible/ineligible roles, max last-step/run timestamps, and reversed
   callback delivery. Feed 65 shuffled completions and prove the latest 64 by event time survive.
2. Pin readiness boundaries: 15 events are unready; 16 spanning under 15s are unready; 16 spanning
   exactly 15s are ready. Assert `(n-1)/(now-oldest)`, including a `now` later than the newest event.
3. Add a mature trace followed by five 1s ticks: last-run age advances five seconds, numeric rate is
   non-increasing, and ETA is non-decreasing. Then add completions and prove gradual 64-event-window
   recovery rather than a one-interval snap.
4. Refactor `movingRate` into a sorted, bounded event-time window. Reduce typed eligible run events
   under the existing progress mutex and remove fold-callback-time rate mutation.
5. Move the aggregate displayed `inner-CV runs` / `CV runs` counter to typed eligible run-completion
   events. Add a reversed-delivery regression proving typed events alone advance the aggregate
   counter/rate/ETA, while sampler fold callbacks retain only score and per-row duties and cannot
   double-count or lag the board counter.

### Task 5: Make occupancy tick-driven and event-density independent

**Files:**
- Modify: `cmd/metis/progress.go`
- Modify: `cmd/metis/progress_test.go`

1. Add a pure four-sample test: occupancies `[1,2,3,4]` at capacity 12 render as rounded mean 3, and
   a fifth sample evicts the first.
2. Drive equal timestamped occupancy samples through traces with sparse versus dense activity events;
   assert identical snapshots.
3. Sample `leafGauge` only from the existing 500ms tick, retain four values, and expose the rounded
   mean plus capacity. Do not update the window from activity callbacks or repaint flushes.

### Task 6: Render shared vocabulary, factual startup, and confidence states

**Files:**
- Modify: `cmd/metis/progress.go`
- Modify: `cmd/metis/board.go`
- Modify: `cmd/metis/progress_test.go`
- Modify: `cmd/metis/board_test.go`

1. Replace existing expected strings with the exact shared nouns: `outer folds`, `configs scored`,
   nested `inner-CV runs`, flat `CV runs`, and row prefix `outer fold N`.
2. Add nested and flat startup golden tests for: no occupancy/activity, occupied but silent, successful
   steps with last-step age, and the first eligible run removing startup. Positive text must derive
   only from typed successful events; never print “not hung” or infer a phase.
3. Add pre-confidence tests showing `— inner-CV runs/min` or `— CV runs/min` and no ETA. Add mature
   tests showing `~ETA`, the matching rate noun, remaining eligible-run scope, and tick-driven
   `last … run Ns ago`.
4. Update `progressCore` once so plain and TUI output share counter semantics; keep board-only temporal
   observations in the board snapshot/renderer. Preserve width clamping with narrow-width tests.

## Chunk 3: End-to-end wiring, terminal proof, and peer documentation

### Task 7: Wire tick/activity flow through flat and nested sweeps

**Files:**
- Modify: `cmd/metis/sweep.go`
- Modify: `cmd/metis/progress.go`
- Modify: `cmd/metis/progress_test.go`
- Modify: `cmd/metis/board_test.go`

1. Add end-to-end scripted flat and nested tests that execute successful steps/runs, advance the fake
   clock across ticks, and assert startup → confidence → mature/stall transitions.
2. Extend the TUI fatal-failure test so activity is visible before failure, then prove the final error
   frame is stable, no post-failure activity repaints, the ticker joins, and terminal cleanup remains
   correct.
3. Connect the activity emitter after board-writer replacement so callbacks use the compositor's
   temporal writer identity. Keep the 500ms ticker and existing health gates; do not introduce a
   second clock loop.
4. Run `go test ./cmd/metis -run 'Progress|Board|Sweep|Activity|RunControl' -race` and fix the cause of
   any flake, race, or lock-order timeout before proceeding.

### Task 8: Update the operator contract in kbench

**Files:**
- Modify in peer repo: `competition/titanic/pipelines/RUNBOOK-sweep.md`
- Modify: `workshop/issues/000049-board-readability-labels-coldphase-eta.md`

1. In `/Users/xianxu/workspace/kbench`, update the board example/description to the exact flat/nested
   vocabulary, factual startup line, `~slots` smoothing, 16-completion/15-second confidence gate,
   last-run age, and mature `~ETA`. Preserve the documented plain-output distinction unless behavior
   actually changed.
2. Search the RUNBOOK for stale `fold`, `leaves`, and `folds/min` board terminology; inspect each hit
   rather than globally replacing legitimate domain language.
3. Run Markdown/diff checks available in kbench, commit that documentation-only peer change, and add
   its full 40-character SHA to issue #49's Log.

### Task 9: Full verification and close-boundary preparation

**Files:**
- Modify if architecture changed: `atlas/` and `atlas/index.md`
- Modify: `workshop/issues/000049-board-readability-labels-coldphase-eta.md`

1. Run `gofmt` on changed Go files, `go test ./cmd/metis -race`, `go test ./... -race`, and
   `git diff --check` in metis. Run the relevant kbench checks and `git diff --check` there.
2. Grep Go, tests, atlas, and operator docs for obsolete board strings; classify remaining domain
   uses. Confirm flat/nested output, width, repaint cadence, failure cleanup, and no-op non-sweep paths.
3. Update atlas only if the implementation adds a durable architectural seam; otherwise record why
   `--no-atlas` is appropriate in close evidence. Check all issue/plan boxes and log exact commands.
4. Run `sdlc close --issue 49 --verified '<evidence>'` once. Let the binary dispatch the mandatory
   fresh-context boundary review; fix every Critical/Important finding and rerun the gate as directed.
