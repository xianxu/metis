---
id: '000051'
status: done
started: 2026-07-17T22:50:40-07:00
created: 2026-07-16
updated: 2026-07-17
estimate_hours: 0.17
actual_hours: 0.35
---

# ledger show — add a point_addr column (the --point handle has no surface)

## Problem

`metis select --point <addr>` (metis#41) publishes an operator-chosen config by ledger row —
but no command SHOWS point addresses: `renderLedger` prints code/status/free-params/metrics
only, so the --point handle can only be scraped from the raw CSV. Operator hit it 2026-07-16
("select didn't show the point value to use for promotion").

