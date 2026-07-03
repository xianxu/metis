---
id: 000002
status: open
deps: []
github_issue:
created: 2026-07-02
updated: 2026-07-02
estimate_hours:
---

# Uniform DAG step caching: content-address step inputs, skip unchanged, recompute only what changed

**Stage: DESIGN — do not build yet.** Captured from the kbench#1 post-live-run
discussion; the design must come first. This ticket holds the intent + constraints.
Part of the **metis v1** project (`brain/data/project/metis-v1.md`); the caching
design is worked out in `brain/workshop/pensive/2026-07-03-01-pensive-experiment-shape-workbench-design.md`
(§Caching) — content-addressed `cas/<hash>` pool, working-tree paths become pointers,
key folds in step-code-version, orthogonal to git so branch-switch never invalidates.

## Problem

`metis run` re-executes the *entire* DAG every time (`cmd/metis/run.go` →
`Runner.Run` → TopoSort → execute-each; no skip/cache logic). So every run
re-downloads from Kaggle (`get-data`, network) and re-trains (compute) even when
nothing about those steps changed. For a **learning bench** whose loop is "tinker
one knob, re-run," that is needlessly slow and re-hits external services each run.

## Spec (intent, not yet a plan)

A **uniform, consistent** cache — the same mechanism whether the step downloads
data, engineers features, splits folds, or trains. The runner resolves **what
actually needs recomputing** from the DAG and skips the rest. Design goals:

- **Content-addressed, not timestamp-based.** A step's cache key = a hash of
  everything determining its output: `uses` (+ the step-type's code version), the
  **resolved** `with`, the **seed**, and the **cache keys of its upstream inputs**
  — so a change propagates downstream but not sideways. Edit adapt's FEATURES →
  adapt + everything downstream recompute; `get-data` is reused.
- **Uniform across layers.** A runner-level concern keyed off the step contract,
  not per-step code — identical for kaggle/kbench/metis steps.
- **Honest invalidation.** A step-type code change must bust the key (hash the
  executable / a declared version) or the cache silently serves stale output — the
  classic caching bug. External-fetch steps (`get-data`) need a policy knob
  (cache the download vs. always re-pull).
- **Cluster with metis#3 (provenance) + fork-per-experiment.** A sibling
  experiment sharing `get-data`/`adapt` config should hit a prior run's cache.
  This is the "tinker a small portion, the system resolves what to compute"
  vision — design the three together.

## Done when

- (design-stage) A design note settles: cache-key composition, where cached
  outputs live + are keyed, code-version invalidation, the external-fetch policy,
  and the interaction with run provenance. Only then split into build milestones.

## Plan

- [ ] Design note: cache-key composition + invalidation + storage + external-fetch policy + interaction with #3.
- [ ] (deferred, post-design) implementation milestones.

## Log

### 2026-07-02
- Filed design-stage from the kbench#1 discussion. Operator: "we shouldn't hurry; caching needs to be uniform and consistent… a well-designed cache greatly improves iteration efficiency by tinkering a small portion and letting the system resolve what to recompute." Cluster with metis#3 + the fork-per-experiment convention.
