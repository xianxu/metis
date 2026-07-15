---
id: 000039
status: working
deps: []
github_issue:
created: 2026-07-14
updated: 2026-07-15
estimate_hours: 3
started: 2026-07-15T14:49:03-07:00
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

Durable plan: `workshop/plans/000039-fingerprint-visibility-plan.md` (reviewed; command named
`metis ledger fingerprints`; record.json fields confirmed: Started/Finished RFC3339 +
Steps[].Code.{Commit,CaptureStatus} + Dirty). Single-pass close, no milestones.

- [ ] Task 1: pure core — `cohortSummaries` reducer + `resolveFingerprint` (git-style prefix) +
  `renderCohorts` in `cmd/metis/fingerprints.go`, TDD
- [ ] Task 2: `metis ledger fingerprints <shape.md>` verb (CLI test through real entrypoint)
- [ ] Task 3: `metis run` prints its cohort line (backfill returns fp+dirty; both capture sites;
  nested + flat output asserted)
- [ ] Task 4: prefix resolution wired into `select` + `ledger show`; honest zero-match +
  multi-cohort guard errors (inline cohort table, name the command); delete `distinctFingerprints`
- [ ] Task 5: docs sweep (RUNBOOK/atlas), real-ledger smoke on the 566995b9 cohort, close

## Log

### 2026-07-14
- Filed by operator during the metis#35 honest-beat: the #32 cohort guard fired (correctly — it
  stopped a silent blend of last session's flat rows with today's nested rows) but named hashes
  with no way to inspect them; identification required hand-counting csv rows. Provenance is
  already captured (metis#14's CodeManifest + capture refs) — this is a surfacing issue, not new
  capture. Sibling UX issue from the same run: metis#38 (parallel-run TUI).
- 2026-07-14 (kbench#9 ship, operator hit both live): (1) `select --fingerprint` is an EXACT
  match — an 8-char prefix (`566995b9`) silently matches nothing; accept unique prefixes like
  git does. (2) The zero-match error is a lie: "no scored configs in <ledger> — run `metis run`
  first" when 2,166 rows exist under the full hash — a fingerprint filter that matches nothing
  must say so and LIST the cohorts present (fingerprint + row count + last-run time), which is
  exactly this issue's inspect surface. Until then the operator recipe is
  `tail -1 <ledger>.csv | cut -d, -f1` for the full hash.

### 2026-07-15
- Claimed + start-plan; durable plan authored at `workshop/plans/000039-fingerprint-visibility-plan.md`
  and fresh-eyes plan-reviewed (2 substantive findings fixed: ExtraCommits fold respecified as
  set-cardinality — ledger rows are not time-ordered; printFingerprintLine signature drift between
  concepts table and task sketch reconciled). Lessons persisted to workshop/lessons.md.
- Design decisions: command is `metis ledger fingerprints` (a ledger view, beside `ledger show`;
  discoverability via the guard error naming it verbatim). `ledger.Filter` stays exact (storage
  primitive); prefix resolution is a cmd-layer `resolveFingerprint` shared by select + ledger show
  (ARCH-DRY — ends the --fingerprint/--point matching-semantics split). Record IO (`record.json`
  reads) only on the inspect command + error paths, never the happy select path (ARCH-PURE).
  Behavior change: `ledger show --fingerprint <no-match>` errors (was: `(no rows)`, exit 0) — Log
  defect (b) applied consistently.
