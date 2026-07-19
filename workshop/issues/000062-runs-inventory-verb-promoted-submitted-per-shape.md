---
id: 000062
status: open
deps: [kaggle#7]
github_issue:
created: 2026-07-18
updated: 2026-07-18
estimate_hours:
---

# runs inventory verb: promoted + submitted per shape

## Problem

After a sweep there is no inventory: which points were PROMOTED (materialized as
`runs/best-*`/`point-*`) and which of those were SUBMITTED (and what they scored) is
archaeology — `ls runs/` + per-dir `record.json` + memory, with public scores living only in
hand-written Logs. Operator-requested (2026-07-19, arena2 M3 wrap: two promoted families,
two submissions, zero machine-readable trace of either fact at the shape level).


## Spec

- **Verb:** operator proposal `metis select --promoted <shape.md>`; open naming question —
  as a READ-ONLY inspection it may belong in the metis#61 `debug` family
  (`metis debug runs <shape.md>`), keeping `select` purely a chooser. Decide at plan time;
  one home only.
- **Output:** one row per materialized run of the shape: run id · point (family + free-param
  tuple, the ledger's `fp.*` rendering) · cohort fingerprint (prefix) · created · submission
  status — from **kaggle#7's receipt sidecar** (`runs/<id>/submission/receipt*.json`):
  `public 0.94966 @ 2026-07-19` or `—` (never submitted). Multiple receipts (re-submits) all
  shown.
- **Data sources (all local, no network):** `runs/*/record.json` (run_id, point_address,
  code_fingerprint, started; filter to ship-materialized runs) + receipt sidecars. Depends on
  kaggle#7 for NEW submissions; historical ones (0.76794/0.79186 titanic, 0.94903/0.94966/
  0.94906 s6e7) predate receipts — display `—` honestly, do NOT backfill fabricated receipts.
- Sort newest-first; `--json` for agent consumption (optional, cheap).


## Done when

-

## Plan

- [ ]

## Log

### 2026-07-18
