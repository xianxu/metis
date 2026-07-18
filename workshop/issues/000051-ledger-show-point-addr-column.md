---
id: 000051
status: working
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-17
estimate_hours: 0.17
started: 2026-07-17T22:50:40-07:00
---

# ledger show — add a point_addr column (the --point handle has no surface)

## Problem

`metis select --point <addr>` (metis#41) publishes an operator-chosen config by ledger row —
but no command SHOWS point addresses: `renderLedger` prints code/status/free-params/metrics
only, so the --point handle can only be scraped from the raw CSV. Operator hit it 2026-07-16
("select didn't show the point value to use for promotion").


## Spec

Add a short (8-char, git-style) `point` column to `metis ledger show`, placed after `code`
in `renderLedger`'s header and rows — the value is `short(r.PointAddr)`, resolvable back
through the existing `resolvePointRows` prefix matcher (select_cmd.go:386, the #41 path the
flag help already documents as "git-style prefix ok"). Round-trip pinned by a fixture test:
a rendered short handle fed to the resolver returns exactly its source row. The Spec's
second half ("winner's point_addr in select's board line") SHIPPED separately via metis#52
(`· point <addr>` handles on every pick line) — out of scope here; this issue closes the
one remaining discovery gap (`ledger show`).


## Done when

- `metis ledger show <shape> --sort <metric>` rows carry a short point_addr usable directly
  as `select --point <prefix>` (round-trip asserted in a fixture test).

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.02 impl=0.15
design-buffer: 0.15
total: 0.17
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

One row: the renderLedger column + the round-trip fixture test (short handle → the existing
--point prefix resolver → same row). The select-side half of the original Spec ("winner's
point_addr in select's board line") already SHIPPED via metis#52 (`· point <addr>` handles) —
this issue closes the remaining `ledger show` gap only.

## Plan

- [ ] `renderLedger`: `point` column (short 8-char) after `code`; header updated
- [ ] round-trip test: rendered short handle resolves via the --point prefix path to the same row
- [ ] Log evidence; atlas untouched (`--no-atlas` justified: one column on an existing documented surface)

## Log

### 2026-07-16
- Filed from the operator's promotion attempt: the --point flow exists (#41) and resolves
  prefixes (#39) but has no discovery surface. Related: metis#40 (select-as-conversation)
  would supersede parts of this; the column is the cheap immediate fix.
