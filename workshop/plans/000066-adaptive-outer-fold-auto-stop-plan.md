---
issue: 000066
title: adaptive outer-fold scheduling + --auto-stop
status: active
created: 2026-07-19
---

# Plan — adaptive outer-fold scheduling (`--live`) + `--auto-stop`

## Context (what exists)

- `runNestedCV` (cmd/metis/sweep.go) drives `sampler.CVDriver` over k outer folds
  via `ParExec`. Each outer fold's `runOuterFold(i)` runs the SEALED sweeper
  (`GridConfigs` ⊃ `FixedKFolds`, both `ParExec`) → per-family winners, then scores
  each family's winner on the held-out outer-assessment. All leaves across all folds
  fan out as goroutines and contend on ONE global leaf semaphore (`execStep.sem
  chan struct{}`, cap = maxParallel) — the only budgeted resource (a cache HIT never
  reaches it).
- The reduce is order-independent (`sampler.Aggregate`), and `sortPointRuns` sorts
  rows to a content key before persisting → the manifest/ledger/estimate are already
  byte-deterministic across serial/parallel. **This is why changing the SCHEDULE is
  pure and cannot change the numbers** — the determinism test locks it.
- The board (metis#38/#55) is paint-only over `sweepProgress` (metis#30 sink). Outer
  completions fire `driverEvent` → append the held-out score to `outerScores` +
  collapse the fold's row; the aggregate line already renders `est mean ± SE`. So the
  "live incremental estimate" is ALREADY emitted per fold — priority scheduling is
  what makes fold 0 finish first so it tightens meaningfully.
- No keyboard/raw-terminal code today (metis is stdlib-only by charter).

## ARCH notes
- **ARCH-PURE:** the priority semaphore is a concurrency primitive (thin, injected,
  colocated with the existing `leafSem` in cmd/metis) — NOT domain logic. The
  predictive stop rule (M2) is a PURE function (partial stats + incumbent → stop?),
  unit-tested directly; the scheduler is the thin seam that consults it.
- **ARCH-DRY:** reuse the existing leaf-sem seam (one budget), the existing
  `driverEvent`/`outerScores` sink, `sampler.Aggregate`/`ledger.AggregateView`, and
  `FamilyOf`/`FamilySelect`. Do NOT add a second budget or a second estimate reduce.
- **ARCH-PURPOSE:** the purpose is (1) an honest live estimate that tightens
  fold-by-fold with backfill, (2) auto-stop that reclaims real budget from losing
  families. Skipping only the per-family SCORE leaf would reclaim ~nothing (the inner
  sweep is the cost) — so auto-stop must skip a stopped family's INNER SWEEP on
  remaining folds. That forces sequential-outer scheduling for `--auto-stop`
  (decision after fold n gates fold n+1's config set) — documented, not deferred.

## Determinism guarantee (the load-bearing invariant)
`--live` MUST be byte-identical to the default full run. Proof strategy: a Go test
runs the SAME nested shape through `runExperiment` twice — default (chanSem, global
fan-out) and `--live` (prioritySem) — and asserts the persisted `.ledger.csv` bytes,
`manifest.json` bytes, and the reported estimate string are identical. Holds because
the reduce is order-independent and `sortPointRuns` normalizes on-disk order.

---

## M1 — priority scheduling + `--live` + live estimate + board `Q`

### M1.1 leaf-budget interface + priority semaphore (new file `prioritysem.go`)
- Introduce `type leafBudget interface { acquire(priority int); release(); gauge() (busy, capacity int) }`.
- `chanSem` — wraps the existing `chan struct{}`; `acquire` ignores priority (today's
  global fan-out). `gauge()` = `(len, cap)`. This is the DEFAULT.
- `prioritySem` — a counting semaphore that grants a free slot to the lowest-priority
  (= lowest outer-fold index) waiter first; FIFO tiebreak by arrival seq. Invariant:
  `waiters non-empty ⟹ inflight == capacity` (a free slot is never held while a
  waiter exists → no idle cores → emergent backfill). Pure-ish, fully unit-testable
  (grant order under contention; capacity honored; concurrency race via `-race`).
- `execStep.sem chan struct{}` → `execStep.budget leafBudget`; `Execute` calls
  `budget.acquire(e.priority)` / `budget.release()` around BOTH spawn seams (legacy +
  fork-server). Thread `priority int` onto `execStep` and `runOpts`.
- `runExperiment` builds `chanSem` (default) or `prioritySem` (when `live &&
  maxParallel>1`); `leafGauge` derives from `budget.gauge`.

### M1.2 thread the outer-fold index as priority
- `sweepPass.priority int` (the outer fold idx). `runOuterFold(i)` sets it;
  `runPipelineFold` sets `pointOpts.priority = p.priority`. `scoreOnOuterFold(i)` sets
  `scoreOpts.priority = i`. Flat path / preamble → priority 0 (irrelevant; single
  contention-free work or the chanSem default).

### M1.3 `--live` flag
- `cmdRun`: `--live` (bool) → `runOpts.live`. Doc: "outer folds finish fold-ordered
  (priority queue, backfill preserved) so the mean±SE tightens live; DEFAULT keeps
  global fan-out for unattended runs." Serial runs are already fold-ordered (SeqExec)
  so `--live` there is a no-op beyond intent.

### M1.4 board `Q` → graceful finalize
- Stop seam: `runOpts.stopSignal <-chan struct{}` (test injects; production spawns a
  stdin reader goroutine in board mode that fires on a `q`/`Q` line — documented as
  q+Enter, an intentional clean Ctrl-C; stdlib-only, no raw mode).
- `runControl` gains a clean soft-latch: `requestStop()` sets `stopped=true` (distinct
  from `err` — a stop is NOT a failure). `run()` returns a clean `errRunStopped`
  sentinel BEFORE executing a leaf when stopped (does NOT call `fail`). A goroutine
  bridges `stopSignal` → `runControl.requestStop()`.
- `runOuterFold`: after the sealed sweep, if `stopRequested` → mark the fold ABANDONED
  (`ss.abandoned[i]=true`, guarded) and return a stopped sentinel WITHOUT recording
  rows or scoring. In-flight folds fast-drain (their remaining leaves skip via the
  clean sentinel); completed folds (finished before Q) already recorded + fired
  driverEvent.
- `driverEvent`: skip abandoned fold indices (so `outerScores` holds only honest
  completions). On stop, `runNestedCV` finalizes: estimate = `Aggregate` over the
  sink's completed `outerScores` → honest `out<n>` (n = completed folds); write the
  partial manifest+ledger (only completed folds added rows — the #58 heal path);
  print an honest stopped-summary. NO pkg/sampler type change.

### M1.5 tests
- **Determinism** (`live_determinism_test.go`): default ≡ `--live` byte-identical
  ledger+manifest+estimate (fake exec, real prioritySem).
- **prioritySem unit** (`prioritysem_test.go`): grant order = priority order under
  contention; capacity honored; `-race` clean; backfill invariant (a lower-priority
  waiter never blocks a free slot).
- **Q finalize** (`live_stop_test.go`): inject `stopSignal` after N fold completions
  (scripted via the board-tick + a gate); assert the run finalizes with `out<n>`, the
  ledger has only completed folds' rows, no error, exit 0.

---

## M2 — `--auto-stop` (incumbent-referenced early stop of losing families)

### M2.1 `--auto-stop` flag + incumbent read
- `cmdRun`: `--auto-stop` → `runOpts.autoStop` (implies `live`; sequential-outer, see
  M2.2). Reads the incumbent from the shape's EXISTING ledger (prior runs) — no
  `--baseline`. Incumbent = the best objective mean among existing OUTER-level
  aggregate rows (`ledger.AggregateView` → `ledger.Best` over the objective metric,
  by direction). Empty ledger → no incumbent → auto-stop is a loud no-op ("run once to
  establish a baseline"); the run proceeds as a normal `--live` run.

### M2.2 sequential-outer + per-fold family filtering
- When `autoStop`, run the OUTER driver via `SeqExec` (inner levels stay `ParExec` —
  cores stay busy within a fold). Fold n fully completes (records per-family outer
  scores) BEFORE fold n+1 starts → the stop decision is clean and race-free.
- `shapeSweep.stopped map[string]bool` (family → stopped). Before `runOuterFold(i)`
  runs its sealed sweep, filter `configPts` to drop stopped families' configs (real
  budget reclaimed — the inner sweep for those families never runs on later folds).
- After each outer fold, accumulate per-family held-out scores (`ss.familyScores
  map[string][]float64`), then evaluate the predictive rule (n≥2) → add losers to
  `stopped`.

### M2.3 predictive stop rule (pure — `autostop.go`)
- `func shouldStop(scores []float64, k int, incumbent float64, direction string) bool`.
- Rule (DOCUMENTED): observe n≥2 fold scores (mean mₙ, sample sd sₙ), r = k−n remaining.
  Predict the full-k mean Mₖ = (n·mₙ + Σ r future)/k. The n done folds are fixed; the
  r future contribute predictive variance ≈ (sₙ²·r/k²)·(1 + r/n) (spread of r future
  iid + estimation error in μ from n samples). One-sided 95% bound via t_{n−1}
  (small-n honest; protects a would-be winner). Maximize: stop iff
  `mₙ + t·SEpred < incumbent`. Minimize: stop iff `mₙ − t·SEpred > incumbent`.
  **Losers only** — a config that could still reach the incumbent runs to full k.
- `tCrit(df)` — hardcoded one-sided 95% table (df 1..10) then z=1.645. Pure, tested.

### M2.4 `stopped: auto` ledger marker
- `ledger.Row.Stopped string` ("" | "auto"), ragged CSV column (present only when set,
  mirroring fold/level/outer_fold). Set on the stopped family's outer rows. Encode/Decode
  round-trip; AggregateView/Best treat it as passthrough metadata (NOT failed).

### M2.5 e2e (`autostop_e2e_test.go`)
- A 2-family fake shape (winner family ~0.90, loser ~0.70), incumbent 0.80 seeded in
  the ledger, k=4, `--auto-stop`. Assert: loser has < k outer rows + `stopped=auto`;
  winner has k outer rows; the estimate ships the winner family; no would-be-winner
  truncated.

---

## Verification
- `go build -o bin/metis ./cmd/metis`; `go test ./...`; `uv run pytest -q` (no regress).
- Determinism test is the hard gate for M1; the loser-stopped/winner-full e2e for M2.
- Atlas: update `atlas/experiment.md` (nested-CV + parallel-executor sections) +
  `atlas/index.md` for `--live`/`--auto-stop` + the priority-scheduling / sequential-
  outer / auto-stop-rule surface.

## Revisions

### 2026-07-19 — M1 boundary review (FIX-THEN-SHIP → fixed in the close commit)
- The `--auto-stop` CLI flag was registered at M1 (not just M2), because its `runOpts.autoStop`
  plumbing threads through the same budget-selection seam as `--live`. The M1 boundary review
  (ARCH-PURPOSE) flagged its help as over-promising M2 behavior; **decision:** the flag's help is
  reworded to be M1-honest ("NOT YET ACTIVE — currently only enables --live; incumbent-read +
  loser-stop land in M2"). M2 restores the full promise once the logic lands.
- **Q during the preamble is now handled, not a no-op:** the stop-bridge is armed BEFORE
  `materializeOuterAnalysis`, so a Q during the (single, long) outer-split short-circuits its
  leaves and finalizes as `out0` — resolving the review's "unreachable preamble-stop branch."
- `--live` help softened: a flat (single-config) run is unaffected (all leaves one priority),
  not an error (unlike `--sample`), matching the actual behavior.
- Q-finalize test strengthened to `out2` (n≥2) so the non-zero-SE branch of
  `completedOuterEstimate` is covered.

### 2026-07-19 — M2 boundary review (FIX-THEN-SHIP → fixed in the close commit)
- **M2.1 incumbent source correction (the substantive fix).** M2.1 above specified
  `ledger.AggregateView → best`, which contradicts the Spec's "`metis select`'s best-per-family"
  and `family.go`'s documented reason that AggregateView is the WRONG per-family reducer (a family's
  winning config varies across outer folds, so it splits one family into per-config subgroups and
  takes an optimistic MAX subgroup mean → an inflated bar that over-stops would-be winners).
  **Corrected:** `readIncumbent` now reads `familyEstimateFromLedger(sh, led, metric)` → the honest
  pooled per-family outer estimate → `sampler.FamilySelect` (the exact reduce `metis select` ships),
  with `ss.sh` threaded in. Regression test `TestReadIncumbent_PoolsPerFamilyNotPerConfig` (rf's
  winner varies across folds → incumbent = 0.85 pooled, not 0.90 per-config max).
- `--auto-stop` now rejects `--sample`/`--fast` loudly (it runs the full-k estimand and stops
  losers itself — the `shouldStop` `k−n` model would mis-count under a sampled subset).
- Added the missing `pkg/ledger` unit test for the `stopped` ragged column (round-trip + the
  no-stopped-rows→no-header byte-identity guard).
- Doc/lock-domain minors corrected: atlas + `autostop.go` wording ("best-per-family via
  FamilyEstimate/FamilySelect", not AggregateView); `markStoppedRows` lock-domain comment; a
  documented cohort limitation (the incumbent pools across fingerprint cohorts, unlike `metis select`).

## Estimate
See the issue's `## Estimate` block (authoritative).
