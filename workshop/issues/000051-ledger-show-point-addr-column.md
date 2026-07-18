---
id: 000051
status: working
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-17
estimate_hours:
started: 2026-07-17T22:50:40-07:00
---

# ledger show — add a point_addr column (the --point handle has no surface)

## Problem

`metis select --point <addr>` (metis#41) publishes an operator-chosen config by ledger row —
but no command SHOWS point addresses: `renderLedger` prints code/status/free-params/metrics
only, so the --point handle can only be scraped from the raw CSV. Operator hit it 2026-07-16
("select didn't show the point value to use for promotion").


## Spec

Add a short (8-char, git-style — #39 prefixes already resolve) `point` column to
`metis ledger show`; consider also printing the winner's point_addr in `metis select`'s
board line (the natural place to grab a handle for a near-winner --point override).


## Done when

- `metis ledger show <shape> --sort <metric>` rows carry a short point_addr usable directly
  as `select --point <prefix>` (round-trip asserted in a fixture test).

## Plan

- [ ]

## Log

### 2026-07-16
- Filed from the operator's promotion attempt: the --point flow exists (#41) and resolves
  prefixes (#39) but has no discovery surface. Related: metis#40 (select-as-conversation)
  would supersede parts of this; the column is the cheap immediate fix.
