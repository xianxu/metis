---
id: '000064'
status: done
started: 2026-07-19T00:11:37-07:00
created: 2026-07-19
updated: 2026-07-19
estimate_hours: 0.5
actual_hours: 0.23
---

# select display: family label falls back to (one family); map-valued free-params render as Go maps

## Problem

First sweep with a MAP-VALUED top-level `with` free-param (`decide: {$any: [argmax,
{offsets: {holdout: 0.2}}]}`, kbench M4 cohort a50b6f25) exposed two display defects in
`metis select`:
1. The per-family honest-estimate line printed `(one family)` where `train.model=hist_gbm`
   belonged (rf rendered fine; the underlying outer rows are correct — reading the ledger
   directly shows both families' rows intact). Likely the family-label renderer choking on
   the map-valued free-param somewhere in the label path.
2. Free-param rendering uses Go's map syntax: `train.decide=map[offsets:map[holdout:0.2]]`
   — unreadable in select output and ledger `fp.*` values; should render canonically (e.g.
   `offsets{holdout:0.2}` or compact JSON), stably (map iteration order must not leak).

Display-only (selection itself picked correctly); annoying at exactly the moment the
operator reads results.

