---
id: 000073
status: working
deps: [metis#71]
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours: 1.74
collaboration-mode: collaboration
card_mirror: 'e7fc7b9217ba51c21e80dcbc073ebc4f1c7023e4' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T19:48:43-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: metis:1
    worktree: /Users/xianxu/workspace/worktree/metis-slot1/metis
    repository: github.com/xianxu/metis
flow: {kind: full, provenance: operator}
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

- [ ] Task 1: writeFileAtomic (extracted); run.json and record.json written atomically
- [ ] Task 2: pkg/ledger Repeat/Extra, extended key, conflict refusal, unknown columns preserved
- [ ] Task 3: repeat aggregation (mean, se and n per metric; total and failed counts)
- [ ] Task 4: --repeat label, record field plus CUE, exclusive run-dir claim, sweep refusal
- [ ] Task 5: rowFromRecord, locked publishRows, single-run rows, metis ledger publish
- [ ] Task 6: e2e repeats, conflicts, interrupted publication, parallel writers, legacy unchanged
- [ ] Task 7: atlas

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

2026-10-23: Filed unclaimed for the ariadne#577 re-plan. Seams: cmd/metis/run.go, sweep.go, ledger.go and pkg/ledger dedupKey. No implementation changed.
2026-10-23 (metis:1): Concurrent claims in this slot: #71, #75, #76, #72 and #73 are open at once while #75 waits on the operator CI pin (metis#70). Active-time attribution across them is approximate until ariadne#580 lands.
2026-10-23 (metis:1): The TL (ops:0) approved the plan as written, including the old-metis rewrite limit, since couch-bench pins metis. Code starts after #71.
