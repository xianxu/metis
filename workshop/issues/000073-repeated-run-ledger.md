---
id: 000073
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '3186df48535037d6a1b4f3c052a0989d4a6e88c4' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

2026-10-23: Filed unclaimed for the ariadne#577 re-plan. Seams: cmd/metis/run.go, sweep.go, ledger.go and pkg/ledger dedupKey. No implementation changed.
