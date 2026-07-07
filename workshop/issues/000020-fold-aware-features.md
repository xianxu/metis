---
id: 000020
status: open
deps: [metis#18]
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours:
---

# fold-aware (leakage-safe) feature pipeline — features fit per fold

## Problem

`features` runs **once, before `cv-split`, on the whole train** — fine for imputation, but **wrong
for any target-based feature** (e.g. group-survival, kbench#8). Such a feature computed on all-train
leaks: a test-fold passenger's feature encodes labels of group-mates who are also in the test fold →
inflated cv that won't reproduce on the true test set. Leakage-safe features must be fit **per fold**
(train-fold members only).

## Spec

metis-v2 M3 (project + pensive: "the pipeline consequence — fold becomes a first-class value").
Restructure so feature-fitting is fold-aware: either move `features` **after** `cv-split` (fold-id in
its `with`), or have `train` wrap a fit/transform feature pipeline that CV drives per fold (sklearn
`Pipeline` is the reference — it re-fits per fold for free). **Deps metis#18** (fold as a first-class
threaded value). Imputation (already train-frame-fit) should ride the same seam so fit-on-train is the
default everywhere.

## Done when

- A target-based feature computed **per fold** (train-fold only) — a leakage test shows the naive
  whole-train version inflates cv and the fold-aware version doesn't.
- Existing (non-target) features unchanged; the titanic thread still green.

## Plan

- [ ] (spec at claim, after metis#18) decide restructure-vs-wrap; wire fold-id to the feature step; leakage regression test.

## Log

### 2026-07-07
- Filed as metis-v2 M3. The pipeline restructure that unblocks leakage-safe features (kbench#8's ticket
  survival). Same "fold as first-class value" as metis#18. Design in the pensive.
