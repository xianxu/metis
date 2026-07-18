---
id: 000056
status: codecomplete
deps: []
github_issue:
created: 2026-07-18
updated: 2026-07-18
estimate_hours: 0.1
started: 2026-07-18T09:59:55-07:00
actual_hours: 0.25
---

# board status line keeps default color

## Problem

Operator feedback on #55's banding (2026-07-18): the status line ("~slots 0/12 · last
inner-CV run 8s ago · …") should NOT be grayed out — it carries live telemetry (the #49
stall/thrash signals) and dimming de-emphasizes exactly the line that matters mid-run.

## Spec

Drop the status-line dim case in `boardWriter.redraw` — the frame's last line renders in
default color like the fold rows. The dim SEPARATOR rule and bold aggregate stay (the
banding). Reconcile the #55 wording in atlas + the archived issue's claims where they say
"dims the status line".

## Done when

-

## Plan

- [x] drop the last-line dim case; adjust tests/docs; suite; direct-main push

## Log

### 2026-07-18
- 2026-07-18: closed — full -race green; e2e pins: status line never dimmed, closing rule present between restore and estimate; all stale restatements reconciled incl. archived #55; actual 0.25h labeled judgment; review verdict: FIX-THEN-SHIP
- 2026-07-18: closed — one-case removal; full cmd/metis suite green; atlas reconciled; actual 0.1h labeled judgment; review verdict: FIX-THEN-SHIP

### 2026-07-18
- One case dropped in redraw (status line → default color); atlas reconciled; full suite
  green (no test pinned the dim — the #55 e2e asserts separator + bold aggregate only).

## Revisions

### 2026-07-18 — scope addition (operator, same feedback thread) + review fold
- ADDED: a closing dim rule between the footer's status line and the epilogue result (the
  operator's paste: status → ─── → estimate). Painted at close, only when an epilogue exists.
- Close review (FIX-THEN-SHIP) folded: stale "dim status" restatements reconciled in
  board.go's redraw comment, the #55 e2e comment, and the ARCHIVED #55 issue; regression pin
  added (sgrDim must not precede the status line) + closing-rule ordering assert.
