---
id: '000067'
status: done
started: 2026-07-19T11:44:40-07:00
created: 2026-07-19
updated: 2026-07-19
estimate_hours: 0.61
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
