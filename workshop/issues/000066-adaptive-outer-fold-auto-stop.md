---
id: 000066
status: working
deps: []
github_issue:
created: 2026-07-19
updated: 2026-07-19
estimate_hours: 7.6
started: 2026-07-19T08:32:12-07:00
---

# adaptive outer-fold scheduling + --auto-stop (incumbent-referenced early stop of losing configs)

## Problem

A full nested-CV run (`--sample out10` on a real competition = ~100 min) commits the whole
budget before showing any honest number — the mean±SE only lands at the very end. Today the
metis#31 parallel executor fans leaves out GLOBALLY (all outer×config×inner leaves scheduled
together, bounded by the semaphore), so no fold "finishes first" and there's no live signal
to act on. Two things the operator wants (arena2 M6 design session, 2026-07-19):

1. **Early partial estimates.** Finish outer fold 0 first, then fold 1, … so a 1-fold →
   2-fold → 3-fold estimate appears live (SE tightening) — the operator can eyeball an
   obvious loser at fold 3 instead of waiting for fold 10.
2. **Auto-stop losers.** After a few folds, if a config is statistically unlikely to beat the
   known incumbent, stop scheduling its remaining folds and reclaim the budget for the rest.

This is the OUTER, incumbent-referenced cousin of metis#54 (racing successive-halving INNER
sampler) — they must not collide (see Non-goals). It also finally delivers the clean per-fold
progress deferred to metis#30.

## Spec

**Correctness is free.** The nested-CV reduce is order-independent (metis#18/#31) — the honest
mean±SE is byte-identical regardless of leaf completion order. So changing the SCHEDULE (which
fold's leaves run when) is a pure scheduling change, in the same class as `SeqExec` vs
`ParExec`: same numbers, different arrival order. No estimand change; a partial run is honestly
an `out<n-done>` estimate.

**M1 — priority scheduling + live incremental estimate (+ human Q-to-stop).** Replace the flat
global fan-out with a **fold-numbered priority queue**: the leaf semaphore dispatches the
lowest-numbered incomplete outer fold's leaves first. That's the whole rule — "backfill" is
emergent, not separate logic: once fold 0's leaves are all in-flight, the next-highest-priority
ready leaves are fold 1's, so a free slot never idles and fold 0 still finishes first for the
early checkpoint (operator: "really just a priority queue based on fold number, don't create
bottlenecks"). After each outer fold completes, emit the running
`Aggregate` (mean±SE over completed folds — SE withheld until n≥2) to the board (metis#55) /
`--sample` progress. A `--live` (or `--sequential-outer`) mode gates it; the default keeps
global fan-out for unattended runs. `Q` on the board = a CLEAN abort: stop scheduling new
leaves, finalize the partial mean±SE, write the ledger (the #58 heal path already exists),
print the honest `out<n>` estimate — an intentional Ctrl-C.

**M2 — `--auto-stop` (remove the human).** Reads the incumbent from the ledger itself — the
promoted run / `metis select`'s best-per-family — so NO `--baseline` flag needed (metis already
records every run). After each fold (n≥2..3), for each config/family, apply a one-sided
predictive stopping rule: if the config's full-k mean is unlikely (≥95% confidence) to reach
the incumbent given the partial (mean, SE, folds remaining), STOP scheduling its remaining
outer folds. **Losers only** — a config that could still win runs to full k (never truncate a
would-be winner; a truncated optimistic estimate must never be shipped). Stopped configs record
their partial estimate + a `stopped: auto` marker in the ledger. Flag: `metis run <shape>
--auto-stop` (composes with `--live`; implies priority scheduling).

**Statistical rule (M2 design note, to settle at plan time):** the honest frame is a sequential
test with repeated looks — naive "partial mean + 1.96·SE < incumbent" inflates the false-stop
rate across k looks. Candidate: a predictive bound on the full-k mean (the n done folds fix
their contribution; bound the k−n remaining by the observed fold spread), one-sided at 95%.
Because the action is STOPPING a loser (not a ship claim), the cost of a wrong stop is low (the
operator can re-run), so a slightly liberal rule is acceptable — but it must be documented, not
silent.

## Non-goals / relationships
- NOT metis#54 (racing successive-halving INNER sampler — reallocates inner-fold budget across
  configs automatically). This issue is OUTER-level and incumbent-referenced. Build so the two
  compose (inner racing within a fold, outer auto-stop across folds) rather than fight.
- Delivers the per-fold progress deferred in metis#30 (fold-ordered completion = natural
  checkpoints).

## Done when

- M1: `--live` runs outer folds fold-ordered with backfill (no idle cores — a throughput test
  vs global fan-out shows ≤ small fold-tail overhead); the board shows a tightening mean±SE
  after each fold (SE withheld at n=1); `Q` finalizes an honest partial `out<n>` estimate +
  ledger. Determinism: `--live` result byte-identical to the default full run.
- M2: `--auto-stop` reads the incumbent from the ledger, stops a config whose partial is
  <95%-likely to reach it (n≥2), never stops a would-be winner, marks stopped configs in the
  ledger; a documented stopping rule; an e2e where a known-loser config is stopped and a winner
  runs full.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.3   impl=0.8
item: smaller-go-module      design=0.2   impl=0.5
item: tui-screen             design=0.4   impl=0.9
item: smaller-go-module      design=0.1   impl=0.5
item: milestone-review       design=0.0   impl=0.2
item: smaller-go-module      design=0.2   impl=0.4
item: smaller-go-module      design=0.3   impl=0.6
item: method-b-decisions     design=0.3   impl=0.5
item: smaller-go-module      design=0.15  impl=0.5
item: milestone-review       design=0.0   impl=0.2
item: atlas-docs             design=0.05  impl=0.2
design-buffer: 0.15
total: 7.6
```

Σdesign 2.0 × 1.15 = 2.30; Σimpl 5.30 × 1.00 = 5.30; total **7.6** (= `estimate_hours`).
Item map: `greenfield-go-module` = the priority-semaphore + leaf-budget interface;
`smaller-go-module`×5 = thread-priority/`--live`, the determinism test harness,
`--auto-stop`+incumbent-read, sequential-outer+family-filtering, ledger-marker+e2e;
`tui-screen` = board `Q`→graceful-finalize (stdin seam + stop-latch + abandon +
partial finalize); `method-b-decisions` = the predictive stop rule (2 decisions ×
0.15 = the predictive-variance model + the t-critical choice); `milestone-review`×2
= the M1/M2 boundary reviews; `atlas-docs` = the atlas sweep. Two review boundaries.

## Plan

- [ ] **M1** — fold-ordered priority+backfill scheduler in the metis#31 executor (a
  `leafBudget` interface: `chanSem` default / `prioritySem` under `--live`); incremental
  `Aggregate` emission per fold; `--live` gate; board `Q`→graceful-finalize (stdin seam →
  clean stop-latch → abandon in-flight folds → partial ledger + honest `out<n>`).
  Determinism test (`--live` ≡ default, byte-identical), prioritySem unit test (grant
  order + backfill invariant + `-race`), Q-finalize test.
- [ ] **M2** — `--auto-stop`: sequential-outer scheduling + per-fold family filtering; read
  incumbent from the shape's existing ledger; documented predictive stopping rule (pure
  `shouldStop`, losers only, t_{n-1} one-sided 95%); `stopped: auto` ledger marker; e2e
  (loser stopped, winner runs full k).

## Log

### 2026-07-19 — M1 implemented
- **Priority scheduling as a pure budget swap.** The metis#31 leaf `chan struct{}` became a
  `leafBudget` interface (`cmd/metis/prioritysem.go`): `chanSem` (default global fan-out) /
  `prioritySem` (min-heap, grants a freed slot to the lowest outer-fold index; backfill
  invariant `len(waiters)>0 ⟹ inflight==capacity`). No ParExec change — all leaves still fan
  out as goroutines; only the budget's grant policy orders them, so fold 0 finishes first. The
  incremental mean±SE emission already existed (`driverEvent`/`outerScores`) — priority
  scheduling makes it tighten meaningfully. Priority = outer-fold idx, threaded
  `runOpts.priority`→`execStep.priority` (set in `runOuterFold`/`scoreOnOuterFold`).
- **`--live`** (implied by `--auto-stop`) builds the prioritySem when parallel; serial is
  already fold-ordered.
- **Board Q** = clean graceful finalize: `stdinStopSignal` (q/Q line, stdlib-only) →
  `runControl.requestStop` (soft-latch ≠ failure) → admitted leaves short-circuit
  (`errRunStopped`) → in-flight folds ABANDONED (`ss.abandoned`, excluded from the estimate) →
  `finalizeStopped` reports honest partial `out<n>` + partial ledger. Full & stopped tails share
  `persistNestedAndReport` (ARCH-DRY). No pkg/sampler change (avoided a type change by gating
  abandoned folds at the driverEvent call site + aggregating the sink's completed scores).
- **Tests:** `TestLive_ByteIdenticalToDefault` (the determinism gate — default ≡ `--live`
  byte-identical ledger+manifest+estimate, budget exercised e2e under `-race`), `prioritysem_test.go`
  (grant order + FIFO + capacity + backfill peak, `-race`), `TestLive_QFinalizesHonestPartial`
  (fold 0 completes → out1, folds 1/2 abandoned). Full suite green (Go + 124 pytest); vet clean.

### 2026-07-19
- Filed from the arena2 M6 design session (operator-designed). Correctness is a non-issue
  (order-independent reduce → scheduling-only change); the work is the executor scheduler + the
  board TUI (M1) and the predictive stopping rule + ledger-incumbent read (M2). NOT claimed —
  awaiting operator priority call (may sequence after the M6 cascade diagnostic + feature probes).
