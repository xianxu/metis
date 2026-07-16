---
id: 000052
status: open
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-16
estimate_hours: 0.50
---

# select surface ergonomics — --cohort listing + point handles on every concrete config line

## Problem

Two operator requests (2026-07-16, post-#50 live session):

1. Listing a shape's cohorts requires switching verbs (`metis ledger fingerprints <shape>`).
   When the operator is composing a `select`, the listing belongs on select's surface:
   `metis select <shape> --cohort`.
2. `select` shows winning configs as free-param tuples with no point handle — good practice:
   **whenever a concrete config is shown as best, show its point value** (the `--point`
   override handle, #41), so promoting a near-winner never requires the raw CSV. (Sibling:
   #51 adds the column to `ledger show`.)

## Spec

1. **`--cohort` on select**: lists the ledger's fingerprint cohorts and exits — a pure
   delegation to the #39 `showFingerprints` core (ARCH-DRY: one implementation, a second CLI
   door where the operator's hands already are). Ignores selection flags.
2. **Point handles**: every pick line (`--best` ship recommendation, `--best-per-model-class`
   winners) carries `· point <short-addr>` — a representative ledger-row address for that
   config (any fold row of the config is a valid `--point` handle by #41's resolver; use the
   first matching row in the cohort-filtered ledger). Round-trip: the printed handle must
   work as `select --point <handle>`.

## Done when

- `metis select <shape> --cohort` (real CLI entrypoint, documented order) prints the same
  per-cohort table as `metis ledger fingerprints`.
- `--best` and `--best-per-model-class` outputs carry `point <addr>` per pick; a fixture
  round-trips the printed handle through `select --point` successfully.
- RUNBOOK §2 mentions both.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.03 impl=0.30
item: atlas-docs          design=0.02 impl=0.08
design-buffer: 0.30
total: 0.45
```

Flag + delegation + handle lookup + two tests; docs row = RUNBOOK/atlas touch.

## Plan

- [ ] TDD: --cohort CLI test + point-handle assertions (incl. round-trip) → implement.

## Log

### 2026-07-16
- Filed from operator requests verbatim. The --cohort door delegates to showFingerprints
  (single mechanism, two triggers — the feedback_minimum_mechanism posture); the handle is a
  ledger-row addr (NOT a fresh mint — #41's resolver accepts any row of the config).
