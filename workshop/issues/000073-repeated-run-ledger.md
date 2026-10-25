---
id: 000073
status: codecomplete
deps: [metis#71]
github_issue:
created: 2026-10-23
updated: 2026-10-25
estimate_hours: 1.74
collaboration-mode: collaboration
card_mirror: '28d4ac29c322c2e4401d3b6afa9c0b60109e2217' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T19:48:43-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: metis:1
    worktree: /Users/xianxu/workspace/worktree/metis-slot1/metis
    repository: github.com/xianxu/metis
flow: {kind: full, provenance: operator}
actual_hours: 0.68
---

# Repeat-shaped run records and ledger rows without CV folds

## Problem

couch-bench (ariadne#577) repeats an autonomous issue with fixed TL and worker profiles. Current plain experiments write run.json and record.json but no ledger row; sweep rows deduplicate by point address and code fingerprint, and repeated identical points can reuse a deterministic run directory. A repeat must remain a new observed sample, not a CV fold or a fake configuration choice.

## Spec

Support repeat-shaped experiment results through Metis's existing record and ledger owners. Bind the repeat coordinate, seed, unique execution directory, experiment/config identity and realized tool/code provenance. Emit rows for the single-experiment path as well as repeated execution. Preserve historical ML fold/sweep interpretation and deduplication when no repeat coordinate is present. Exact retries of recording the same execution stay idempotent; different repeats stay distinct. Aggregate numeric metrics with sample count and uncertainty without inventing absent measurements.

## Done when

- Two executions of the same issue/model/config as distinct repeats produce separate directories, immutable records and ledger rows; neither overwrites nor deduplicates the other.
- A single true replay is represented in that same ledger; no couch-bench JSONL ledger is needed.
- Exact duplicate publication is idempotent, conflicting same-execution data refuses, and old ledger rows remain readable with unchanged meaning.
- Means/error views group model/config families across repeats, preserve numeric step metrics and missing-value coverage, and do not invoke nested CV or call a repeat a fold.
- Tests cover identical-config repeats, differing code fingerprints, interrupted record publication, and legacy CV/default behavior.

## Plan

Durable plan: `workshop/plans/000073-repeated-run-ledger-plan.md` (single review boundary, Tasks 1 to 7). Design decisions 1 to 9 there were approved by the TL (see Log).

- [x] Task 1: writeFileAtomic (extracted); run.json and record.json written atomically
- [x] Task 2: pkg/ledger Repeat/Extra, extended key, conflict refusal, unknown columns preserved
- [x] Task 3: repeat aggregation (mean, se and n per metric; total and failed counts)
- [x] Task 4: --repeat label, record field plus CUE, exclusive run-dir claim, sweep refusal
- [x] Task 5: rowFromRecord, locked publishRows, single-run rows, metis ledger publish
- [x] Task 6: e2e repeats, conflicts, interrupted publication, parallel writers, legacy unchanged
- [x] Task 7: atlas

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.05 impl=0.14
item: greenfield-go-module   design=0.1 impl=0.24
item: smaller-go-module      design=0.05 impl=0.16
item: smaller-go-module      design=0.1 impl=0.2
item: smaller-go-module      design=0.05 impl=0.2
item: smaller-go-module      design=0.0 impl=0.16
item: atlas-docs             design=0.05 impl=0.04
item: milestone-review       design=0.0 impl=0.14
design-buffer: 0.15
total: 1.74
```

Items, in order:
1. One atomic-write helper (consolidating pkg/cas and caching.go).
2. Ledger changes: Repeat and Extra, the key, conflict refusal and replace, unknown columns.
3. Repeat aggregation.
4. `--repeat`, the claim, the record field plus CUE, and the in-memory backfill refactor.
5. Locked publishRows and `metis ledger publish`.
6. e2e and concurrency tests.
7. Atlas.
8. One close review.

Design hours are near zero because the TL approved the plan, and it cleared 2 plan-quality rounds. Impl is 40% of the v2 ranges.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Log



- 2026-10-25: closed — Round 2 advisories fixed: the repeat path mints the point address itself (pointAddressOf) and refuses on failure, so no fallback can name the claimed dir (new TestRepeat_RefusesUnaddressableConfig); the atlas states se only from n >= 2 and the untracked/unaddressable refusal. Targeted cmd/metis repeat/publish tests pass. (fixed-then-ship); review verdict: SHIP
- 2026-10-25: closed — Recorded suite sdlc-local/suite passed on 5cfad8db after the close review round 1 fixes. Targeted: pkg/ledger (legacy byte-identical round-trip, unknown columns, Append/Publish kinds, repeat family mean/se/n with no se below n=2, Publish label validation and Extra kept), cmd/metis repeat_e2e_test (distinct repeat samples, label reuse refuses, code edit splits families, interrupted publication recovers, conflict and invalid/misplaced label refuse, untracked experiment refuses --repeat, fingerprint cohorts use their own records (mutation checked), validation failure releases the claim, plain row replace, parallel publishers and claim race).; review verdict: FIX-THEN-SHIP
2026-10-23: Filed unclaimed for the ariadne#577 re-plan. Seams: cmd/metis/run.go, sweep.go, ledger.go and pkg/ledger dedupKey. No implementation changed.
2026-10-23 (metis:1): Concurrent claims in this slot: #71, #75, #76, #72 and #73 are open at once while #75 waits on the operator CI pin (metis#70). Active-time attribution across them is approximate until ariadne#580 lands.
2026-10-23 (metis:1): The TL (ops:0) approved the plan as written, including the old-metis rewrite limit, since couch-bench pins metis. Code starts after #71.
2026-10-23 (metis:1): change-code passed on the pinned full flow. Plan-quality: round 1 raised 3 Important findings (the aggregate-row key collision, plain-row rerun semantics, and the undefined final record); all were fixed in the plan Revisions, and round 2 was clean. Estimate 1.74h; estimate-quality INFO, which suggests the realistic total is 2.0 to 2.3h (items 4 to 6 light). Recorded as accepted. Code waits for #71.
2026-10-25 (metis:1): implementation, Tasks 1 to 7, after #71 and #72 landed (main merged at the start). Concurrent claims note still applies (#73 and #74 open in this slot; ariadne#580 attribution). Notes against the plan:
- Append keeps its first-wins skip for sweep and plain rows; the plain-row replace (PQ-2) is a separate `Ledger.Publish`, so the replace is explicit at the one call site that wants it and AggregateView/Filter/sweeps cannot replace by accident. Append now returns an error (repeat conflicts only).
- `metis ledger publish` takes `<experiment.md> <run-dir>` rather than `<run-dir>` alone: a run dir does not name its experiment file (the record holds only the experiment id), and the ledger sits beside the .md. It refuses a dir that is not under that experiment's runs/. Both refusal hints print the full command.
- rowFromRecord uses the record's point address as the row identity. An untracked experiment (no git) has no shape blob hash, so its run dir is timestamped while its point address is still minted; rowRunID then misses its dir, which the fingerprint view already treats as "no record". The e2e tests use a tracked experiment.
- A failed run's row carries no metrics (decision 5), even when earlier steps measured some; AggregateView ignores failed members' values anyway.
Verification (targeted): pkg/ledger (legacy byte-identical round-trip property, unknown columns, Append/Publish kinds table, repeat family and n-counts property, fold groups unchanged), internal/fsx (rename-crash keeps the old file), pkg/record CUE conformance with repeat, cmd/metis package (new repeat_e2e_test: two repeats, label reuse, code edit splits families, interrupted publication and recovery, conflict refusal, plain row replace and failed row, refusals before effects, replay warning, 16 parallel publishers and a claim race) all pass.
2026-10-25 (metis:1): close review round 1 FIX-THEN-SHIP. All four Important findings were real and are fixed with tests. BR-1: a record's repeat label is validated (Publish, and readFinalRecord checks it names its own dir), so publish can no longer write a row Decode refuses. BR-2: --repeat refuses an untracked experiment, whose timestamped dir made the label claim non-exclusive. BR-3: fingerprint cohorts key records by run id (the new two-fingerprint test fails with the old key, mutation checked). BR-4: a repeat family omits se below n=2. The minors are fixed too: a validation refusal releases the claim, Ledger.Has replaces the hand-written key, and a replace keeps Extra while SameRow ignores it. Plan Revisions updated and Tasks ticked. Lesson added.
2026-10-25 (metis:1): close review round 2 FIX-THEN-SHIP; issue closed with two advisories, both fixed. (1) The repeat-claim family's rule: the repeat path now mints the point address itself (pointAddressOf) and refuses on failure, so no fallback can name the claimed dir; a new test covers a non-finite config. (2) The atlas now states se only from n ≥ 2, the untracked/unaddressable refusal, and the claim release.
2026-10-25 (metis:1): merge-time suite on the integrated head failed TestExecStep_EscapedGrandchildIsCleanupUnconfirmed/exit_0 (from #72): the test asserted step runtime < joinBound 300ms, but the escaper fixture waits for a python3 grandchild to start, which can take longer than 300ms under suite load. Fixed in the test: the exit-0 case uses joinBound 3s, and the failure message prints the value, not a pointer. Targeted run -count=3 passes. Lesson added.
