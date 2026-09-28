---
id: '000052'
status: done
started: 2026-07-16T08:32:33-07:00
created: 2026-07-16
updated: 2026-07-16
estimate_hours: 0.50
actual_hours: 0.3
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
