---
id: 000056
status: working
deps: []
github_issue:
created: 2026-07-18
updated: 2026-07-18
estimate_hours: 0.1
started: 2026-07-18T09:59:55-07:00
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

### 2026-07-18
- One case dropped in redraw (status line → default color); atlas reconciled; full suite
  green (no test pinned the dim — the #55 e2e asserts separator + bold aggregate only).
