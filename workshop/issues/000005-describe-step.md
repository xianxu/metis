---
id: 000005
status: open
deps: []
github_issue:
created: 2026-07-02
updated: 2026-07-02
estimate_hours:
---

# metis/describe step: per-feature distribution + class balance for human inspection

## Problem

The pipeline goes raw → adapt → model with no place for a human to *look at the
data*. For a learning bench, seeing each feature's distribution and the target's
class balance is a core early step (EDA) — it's how you decide what to engineer and
whether the adaptation looks sane.

## Spec

A read-only `metis/describe` step-type: load a `Dataset` (`with.dataset`,
experiment-relative) and emit per-feature summary stats + target class balance.

- **Outputs:** `metrics.json` (flat numeric — e.g. `n`, per-feature `missing_frac`,
  `mean`, `std`, `min`, `max`; target `positive_frac`) so it lands in the run
  ledger, **plus** a human-readable artifact (a small markdown/text table, and
  optionally text histograms — keep it dependency-light and deterministic).
- **Shape:** thin IO entrypoint over a **pure** core in `metis/` (stats computed on
  in-memory frames, unit-tested with no IO — same ARCH-PURE split as split/model).
- **Placement:** off the critical path — `needs: [adapt]`, nothing downstream needs
  it. Competition-agnostic → metis layer.

## Done when

- `metis/describe` exists (`steps/metis/describe` wrapper + `metis/steps/describe.py`
  + a pure `metis/describe.py` core); pure core pytested on in-memory frames.
- Emits metrics.json + a human-readable artifact; process-level exercised (a run or
  a step test) against a small fixture Dataset.
- atlas/experiment.md step catalog + (if landed) the step manifest updated.

## Plan

- [ ] Pure core: `describe(dataset) → stats` (per-feature + target balance); unit test.
- [ ] Thin entrypoint `metis/steps/describe.py`; wrapper `steps/metis/describe`.
- [ ] Human-readable artifact rendering (table; optional text histogram).
- [ ] atlas step-catalog entry (+ manifest if metis#4 landed).

## Log

### 2026-07-02
- Filed at operator request from the kbench#1 walkthrough — "seems worth a step to print out distribution of features for human inspection." Read-only EDA step, competition-agnostic (metis).
