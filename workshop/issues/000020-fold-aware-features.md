---
id: 000020
status: working
deps: [metis#18]
github_issue:
created: 2026-07-07
updated: 2026-07-12
estimate_hours:
started: 2026-07-12T19:10:02-07:00
---

# leakage-safe target features — internal cross-fit (features already per-fold via M1a)

## Problem

A target-based feature (e.g. group-survival, kbench#8) computed on all-train **leaks**: a test-fold
passenger's feature encodes labels of group-mates also in the test fold → inflated cv that won't
reproduce. Even *within* a training fold, using a passenger's own group to score that same passenger
leaks their own label (catastrophic for small groups). So such features need per-fold **and** internal
cross-fitting/shrinkage.

## Spec

metis-v2 M3. Under the converged design (pensive), the restructure is **already done by M1a**: `features`
lives in the `pipeline` phase → runs **per-fold structurally** (cross-fold leakage-safe, no marker, no
cv-split). This issue is then the **target-feature's own within-fold cross-fit**: a feature that uses the
label must cross-fit/shrink *internally* — exactly sklearn `TargetEncoder`'s `fit_transform` (internal CV)
or tidymodels `step_lencode_mixed` (shrinkage) — so a passenger's own label doesn't leak into their own
feature. **No engine `fit_scope` marker** (dropped as error-prone); the step owns it. **Deps metis#18.**
(Future: derive "reads the target" from a column-level data-read trace to *enforce* it — pensive.)

## Done when

- A target-based feature computed **per fold with internal cross-fit** — a leakage test shows the naive
  whole-train version inflates cv and the cross-fit version doesn't.
- Existing (non-target) features unchanged; the Titanic thread still green.

## Plan

- [ ] (spec at claim, after metis#18) target-feature with internal cross-fit/shrinkage; leakage regression test (naive whole-train inflates cv, cross-fit version doesn't).

## Log

### 2026-07-07
- Filed as metis-v2 M3. The pipeline restructure that unblocks leakage-safe features (kbench#8's ticket
  survival). Same "fold as first-class value" as metis#18. Design in the pensive.
### 2026-07-07 (design converged)
- Scope sharpened: M1a already makes features per-fold (they're in the `pipeline` phase), so cross-fold
  safety is free. This issue is now specifically the target-feature's **internal** cross-fit/shrinkage
  (sklearn `TargetEncoder` / tidymodels `step_lencode_mixed`). The `fit_scope` engine marker was dropped.
