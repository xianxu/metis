---
id: '000056'
status: done
started: 2026-07-18T09:59:55-07:00
created: 2026-07-18
updated: 2026-07-18
estimate_hours: 0.1
actual_hours: 0.25
---

# board status line keeps default color

## Problem

Operator feedback on #55's banding (2026-07-18): the status line ("~slots 0/12 · last
inner-CV run 8s ago · …") should NOT be grayed out — it carries live telemetry (the #49
stall/thrash signals) and dimming de-emphasizes exactly the line that matters mid-run.
