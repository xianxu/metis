---
id: 000050
status: working
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-16
estimate_hours: 0.54
started: 2026-07-16T08:14:43-07:00
---

# run-end summary — elapsed time, fingerprint, rows, and paste-ready next commands

## Problem

A sweep ends with the estimate line and a generic "ship via `metis select --promote`" hint.
The operator (2026-07-16) then has to scrape the cohort fingerprint out of the scrollback
(the #39 `recording under` line), remember the shape path, and assemble the follow-up
commands by hand. The run KNOWS all of it: wall-clock elapsed, the fingerprint it recorded
under, how many rows landed in which ledger, and what the sensible next commands are.

## Spec

Both sweep paths (nested + flat) end with a summary block, printed LAST (after the
estimate/report lines; in board mode it scrolls above the final frame):

```
metis: done in 42m10s — 2,160 rows → titanic-sweep.ledger.csv (cohort 18e6e02a)
  next: metis select titanic-sweep.md --fingerprint 18e6e02a              # the honest pick
        metis select titanic-sweep.md --fingerprint 18e6e02a --best --promote   # materialize it
        metis ledger fingerprints titanic-sweep.md                        # cohorts
```

- Elapsed via the injected `now` captured at sweep start (controllable-time; tests assert a
  scripted duration). Reuse the #38 compact-duration formatter (rename fmtETA → fmtDuration
  or share).
- Fingerprint: `captureSweepCode` already mints + prints it (#39) — return it to the caller
  (signature change, 2 call sites) rather than re-deriving (ARCH-DRY: one mint site).
- Rows = len(man.Points); ledger path via `ledgerPath`. Degraded capture (no fp) → the
  cohort segment reads `(cohort ?)` and the `--fingerprint` argument is omitted from the
  hints (they still work — single-cohort ledgers need no pin).
- The shape path in hints is the operator-typed `o.expPath` (relative, paste-ready), not an
  absolute resolution.

## Done when

- Nested + flat fixture sweeps end with the block: `done in <dur>`, the row count, the
  ledger filename, the short fingerprint, and a `metis select <shape> --fingerprint <fp>`
  line (asserted on captured output; scripted clock gives a deterministic duration).
- Degraded/absent fingerprint degrades the block gracefully (no lying pin).
- A real smoke run shows the block with a sane elapsed.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.05 impl=0.35
item: atlas-docs          design=0.02 impl=0.10
design-buffer: 0.30
total: 0.54
```

One helper + start-time capture + captureSweepCode return-value change (2 call sites) +
fixture assertions; docs row = RUNBOOK/atlas touch + smoke evidence.

## Plan

- [ ] TDD: summary assertions in the nested + flat fixture tests → `printRunSummary` helper
  + start-time capture + captureSweepCode returns fp.

## Log

### 2026-07-16
- Filed from operator request ("print at the end: actual time took, fingerprint, and any
  other information needed for further commands"). Completes the #39 fingerprint-visibility
  loop: run prints its cohort at record time AND hands the operator the paste-ready select
  commands at exit.
