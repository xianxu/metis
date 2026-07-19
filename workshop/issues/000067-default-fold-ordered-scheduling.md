---
id: 000067
status: working
deps: []
github_issue:
created: 2026-07-19
updated: 2026-07-19
estimate_hours:
started: 2026-07-19T11:44:40-07:00
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

## Plan

- [ ] `run.go`: default scheduler `prioritySem`, `chanSem` only under `o.globalFanout`; drop
  `o.live`. `main.go`: replace `--live` with `--global-fanout`, thread `o.globalFanout`, gate
  `q`-finalize on `o.tui`. Update the `--auto-stop` help (drop "implies --live").
- [ ] Tests: default-uses-prioritySem + `--global-fanout`-uses-chanSem + auto-stop-unchanged +
  determinism still byte-identical. Update/rename any `--live` scheduler-selection test.
- [ ] Sweep comments (`run.go`/`exec.go`/`prioritysem.go`) + atlas (`experiment.md` scheduler
  note) reconciled to the new default.

## Log

### 2026-07-19
- Filed from the arena2 session: operator observed "all folds at once" on a normal `metis run`,
  diagnosed --live as opt-in, and directed graduating the priority queue to default while keeping
  `--auto-stop` explicit. Scope confirmed: `o.live` is used in exactly 2 places (scheduler select
  in run.go:152, q-finalize gate in main.go:85); no runbook uses `--live`.
