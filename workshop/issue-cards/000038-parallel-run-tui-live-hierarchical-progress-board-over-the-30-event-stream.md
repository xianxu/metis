---
id: '000038'
status: done
started: 2026-07-15T17:24:43-07:00
created: 2026-07-14
updated: 2026-07-15
estimate_hours: 2.19
actual_hours: 1.50
---

# parallel-run TUI — live hierarchical progress board over the #30 event stream

## Problem

With metis#31, a nested sweep fans out across NumCPU leaves — 5 outer folds × 99 configs × 5 inner
folds = 2,475 fold runs executing concurrently — and the terminal shows **nothing** until the pipe
flushes at exit (felt acutely on the metis#35 honest-beat run: minutes of silence, no way to tell
"downloading" from "hung" from "3/5 outer folds done"). metis#30's aggregated single line fixes
blindness for a SERIAL mental model, but under parallelism one line can't render what's actually
happening: several outer folds in flight at once, each with its own inner progress, plus a shared
leaf-semaphore occupancy. The operator (2026-07-14) wants a TUI/curses implementation so parallel
progress is comprehensible.
