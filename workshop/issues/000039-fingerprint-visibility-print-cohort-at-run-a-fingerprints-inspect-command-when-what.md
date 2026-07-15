---
id: 000039
status: open
deps: []
github_issue:
created: 2026-07-14
updated: 2026-07-14
estimate_hours:
---

# fingerprint visibility — print cohort at run + a fingerprints inspect command (when/what)

## Problem

`code_fingerprint` is load-bearing but invisible. The metis#35 honest-beat run (2026-07-14):
`metis select` refused with *"ledger spans 2 code-fingerprint cohorts [4cc9b742 b7aee3de] — pin one
with `--fingerprint <hash>`"* — correct guard, but the operator has **never seen either hash**:
`metis run` doesn't print the fingerprint it records under, and no command answers "which of these
is which — when did each run, from what code?" Resolving it took reverse-engineering row counts
from the csv (495 = last session's flat run; 2,490 = today's nested). The provenance already
exists on disk — per-run `record.json` CodeManifest (commit, dirty D, capture status, timestamps)
and the capture refs (`refs/metis/sweeps/<shapeRunID>`) — it just has no surface.

## Spec

Two additions, both presentation over existing capture data (no new instrumentation):

1. **`metis run` prints its cohort**: one line at record time, e.g.
   `metis: recording under code_fingerprint b7aee3de (commit 9cea652, clean)` — so the hash the
   guard later names is one the operator has already seen scroll by. Same line on `--fast`/flat.
2. **`metis fingerprints <shape.md>`** (name at design; maybe `metis ledger fingerprints`): list
   the ledger's cohorts with the attributes that let an operator pick one —
   `fingerprint · rows (inner/outer) · first-run … last-run (from record.json timestamps) ·
   commit + dirty-status (from CodeManifest / the capture ref) · capture status`. Rows whose
   fingerprint is empty (pre-fingerprint legacy) group as `(legacy)`.
3. **The guard message upgrades for free**: when select refuses, it can render the same per-cohort
   summary inline (or point at the new command) instead of bare hashes.

## Done when

- A nested + a flat `metis run` each print the fingerprint they record under (asserted in the
  e2e/unit output checks).
- `metis fingerprints` on a multi-cohort ledger (fixture: two cohorts + legacy blank) shows per
  cohort: row counts by level, first/last run timestamps, commit + dirty status. Deterministic
  ordering (newest last).
- The select cohort-guard message names the command (or inlines the summary) — an operator hitting
  it can resolve the pin without opening the csv.

## Plan

- [ ] Spec at claim: pick the command name/placement, confirm record.json timestamp fields, then
  TDD (pure cohort-summary reducer over ledger rows + records; thin cmd rendering).

## Log

### 2026-07-14
- Filed by operator during the metis#35 honest-beat: the #32 cohort guard fired (correctly — it
  stopped a silent blend of last session's flat rows with today's nested rows) but named hashes
  with no way to inspect them; identification required hand-counting csv rows. Provenance is
  already captured (metis#14's CodeManifest + capture refs) — this is a surfacing issue, not new
  capture. Sibling UX issue from the same run: metis#38 (parallel-run TUI).
