---
id: 000067
status: codecomplete
deps: []
github_issue:
created: 2026-07-19
updated: 2026-07-19
estimate_hours: 0.61
started: 2026-07-19T11:44:40-07:00
actual_hours: 0.24
---

# Default fold-ordered scheduling (graduate --live); --global-fanout escape hatch

## Problem

metis#66 shipped fold-ordered scheduling (`prioritySem`: freed leaf-budget slots go to the
lowest outer-fold index → fold 0 finishes first, the live mean±SE tightens fold-by-fold, backfill
keeps every core busy) as the OPT-IN `--live` flag, with priority-blind global fan-out (`chanSem`)
as the default. But the scheduler is proven **byte-identical** (scheduling-only; the reduce is
order-independent, locked by the determinism test), and `prioritySem` backfills so it's never
slower — there is **no reason** for the better-observability scheduler to be opt-in. Operators
running a normal `metis run` see all folds fan out at once (no fold-0-first tightening) unless they
happen to know to pass `--live`. Graduate it to the default.

## Spec

- **Default = `prioritySem`** for every nested parallel run (`maxParallel>1`). Fold-ordered
  scheduling with backfill becomes the out-of-the-box behavior.
- **`--global-fanout`** — a loud escape hatch back to the pre-#66 priority-blind `chanSem`
  (escape-hatch-loud: keep the proven fallback reachable, even though it's never known to be
  better).
- **Remove `--live`** — its behavior is now unconditional; no runbook/shape references it (only
  code comments + the archived #66 close-review), so removal is clean (minimum-mechanism; no dead
  no-op alias). The `o.live` field goes too.
- **`q`-finalize gated on `o.tui` alone** (was `o.live && o.tui`) — any interactive TTY sweep can
  press q to finalize an honest partial `out<n>` estimate early; it's an interactive affordance,
  orthogonal to the (now-default) scheduler.
- **`--auto-stop` unchanged** — stays explicit opt-in. It *drops* losing folds (changes the
  estimate → a real behavioral change, correctly opt-in) and keeps its stronger sequential-outer
  mode. It already selects `prioritySem`; with the new default that's a no-op simplification.
- Scheduler stays byte-identical: the determinism test must still pass under the new default.

## Done when

- `metis run` (no flag) on a nested parallel shape uses `prioritySem` (fold 0 finishes first);
  `--global-fanout` restores `chanSem`; both byte-identical to each other and to pre-#67.
- `--live` is gone (unknown-flag on use); `--auto-stop` still works, still opt-in, still sequential.
- `q`-finalize works on any TTY sweep (no `--live` needed).
- Tests green (`go test ./... -race` incl. prioritysem + run scheduler-selection + determinism);
  help text + code comments + atlas updated.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.2 impl=0.25
item: atlas-docs          design=0.0 impl=0.1
design-buffer: 0.30
total: 0.61
```

Small extend of an existing Go module (scheduler default + flag rename + one field threaded +
scheduler-selection tests; the design is fully specced above and the code was read this session),
plus comment/atlas reconciliation. No new abstraction (the `leafBudget` interface already exists).

## Plan

- [x] **Extract the pure selector seam (ARCH-PURE; the plan-quality finding).** New
  `selectLeafBudget(maxParallel int, globalFanout bool) leafBudget` — the pure decision:
  `maxParallel<=1 → nil`; else `globalFanout → newChanSem` else `newPrioritySem`. `run.go:151-157`
  shrinks to call it. Drop the `o.live` field. (The selection is byte-identical in run output by
  design, so it's untestable from artifacts — the pure function is the seam that makes the
  default-uses-prioritySem assertion writeable.)
- [x] `main.go`: replace the `--live` flag with `--global-fanout` (the loud escape hatch); thread
  `o.globalFanout`; drop the `live: *live || *autoStop` line (auto-stop threads `autoStop`
  directly). Gate `q`-finalize on `o.tui` alone (was `o.live && o.tui`). Update the `--auto-stop`
  help (drop "implies --live"; note the flat-shape no-op the #66 close-review flagged).
- [x] Tests: (a) NEW `selectLeafBudget` unit test — default→`*prioritySem`, `globalFanout`→
  `*chanSem`, `maxParallel<=1`→nil (assert concrete type). (b) KEEP `live_test.go` byte-identity
  determinism test — it injects both budgets directly, so it's a *prioritySem-vs-chanSem
  determinism* check (NOT a selection test); just drop the vestigial `live: live` line. (c) NEW:
  `q`-finalize on a NON-sweep TTY run (`maxParallel<=1`) is a safe no-op — the broadened `o.tui`
  gate must not hang a plain experiment (`stopSignal` ignored off the sweep path). (d) auto-stop
  tests still green — swap `live: true` → `autoStop: true` where the removed field was set
  (`autostop_e2e_test.go:93`, `live_stop_test.go:62`).
- [x] Comments/atlas — reconcile the exact stale `--live` anchors: `run.go:89,91,149-150`;
  `exec.go:38,39-40,147`; `prioritysem.go:5,25`; `main.go:53` (`--auto-stop` help); atlas
  `experiment.md` scheduler note. `go test ./... -race` + `go vet` green.

## Log

### 2026-07-19
- 2026-07-19: closed — go vet clean; go test ./cmd/metis ./pkg/sampler ./pkg/ledger -race green (cmd/metis 34.8s incl. TestLive_ByteIdenticalToDefault determinism + new TestSelectLeafBudget + TestQFinalize_NonSweepIgnoresStopSignal); CLI verified: run --help shows --global-fanout, run --live errors (removed); no stale --live in non-test code or atlas; review verdict: FIX-THEN-SHIP
- Filed from the arena2 session: operator observed "all folds at once" on a normal `metis run`,
  diagnosed --live as opt-in, and directed graduating the priority queue to default while keeping
  `--auto-stop` explicit. Scope confirmed: `o.live` is used in exactly 2 places (scheduler select
  in run.go:152, q-finalize gate in main.go:85); no runbook uses `--live`.
- change-code plan-quality judge (FAILURE, correctly): the promised "default-uses-prioritySem"
  test was unwriteable — the selection is byte-identical AND buried in `runExperiment`'s IO glue.
  Folded in the fix: extract a pure `selectLeafBudget(maxParallel, globalFanout)` seam and unit-test
  that. Also folded the 3 minors (determinism-vs-selection test split; a q-finalize-no-op-on-
  non-sweep test for the broadened `o.tui` gate; the exact comment anchors).
- **Implemented + verified.** Pure `selectLeafBudget` in `prioritysem.go` (default `*prioritySem`,
  `--global-fanout`→`*chanSem`, serial→nil); `run.go` calls it, `o.live` field dropped; `main.go`
  `--live`→`--global-fanout`, `q`-finalize regated on `o.tui`; `--auto-stop` help updated. Tests:
  `TestSelectLeafBudget` (concrete-type), determinism test kept (`live: live` line dropped),
  `TestQFinalize_NonSweepIgnoresStopSignal` (plain run ignores a fired stopSignal), auto-stop tests
  green (vestigial `live: true` removed). `go vet` clean; `go test ./cmd/metis ./pkg/sampler
  ./pkg/ledger -race` green (cmd/metis 34.8s incl. the determinism test). CLI verified: `run --help`
  shows `--global-fanout`, `run --live` errors (flag removed). Atlas (`experiment.md` + `index.md`)
  reconciled. **Co-existence note (judge minor):** `--auto-stop --global-fanout` → chanSem, a
  harmless no-op — auto-stop enforces sequential-outer in `sweep.go` independent of the budget type,
  and the two sems are byte-identical (determinism test).
