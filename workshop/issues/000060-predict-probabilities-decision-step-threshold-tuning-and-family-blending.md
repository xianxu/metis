---
id: 000060
status: working
deps: []
github_issue:
created: 2026-07-18
updated: 2026-07-18
estimate_hours:
started: 2026-07-18T22:35:41-07:00
---

# predict probabilities + decision step: threshold tuning and family blending

## Problem

`metis/predict` emits hard labels, which forecloses the two highest-EV M3 moves for arena2
(and any future imbalanced-metric competition): per-class threshold tuning (the balanced-
accuracy-optimal decision rule — argmax is only optimal for accuracy; `class_weight` is the
crude training-time tilt, and tree-ensemble probabilities are miscalibrated anyway, so the
empirical tune beats the divide-by-prior formula) and family blending (`select
--best-per-model-class --promote` already materializes one run per family — metis#22 — but
nothing can combine them). Both need the SAME missing primitive: probability outputs plus a
decision step. Demand #3 from arena2 (operator-approved direction, 2026-07-18 session).


## Spec

- **`metis/predict` gains probability output**: emit `probabilities.csv` (id + one column
  per class, from `predict_proba`) alongside `predictions.csv` (back-compat: label output
  unchanged; downstream steps opt in by reading the new artifact).
- **A `metis/decide` step-type**: reads upstream probabilities + a decision config —
  `{rule: argmax}` (default, today's behavior) | `{rule: per-class-offsets, tune: {metric:
  balanced_accuracy}}` (tune offsets on the upstream OOF probabilities to maximize the
  metric, apply to the target probabilities) | `{rule: blend, sources: [...], weights}`
  (average multiple upstream probability sets, then decide). Pure core; loud on malformed.
- **Honesty seam (the important part):** threshold tuning must run INSIDE the sealed sweep —
  tuned per outer fold on that fold's inner-OOF, scored on the held-out assessment — so the
  honest OUTER estimate covers "fit + tune" as ONE procedure. Tuning offsets on OOF and
  reporting the same OOF score is self-serving; the nested seal is what keeps it honest.
  This likely means the per-fold train path must ALSO emit assessment-fold probabilities
  (the OOF material), not just fold_score.
- OOF assembly: k-fold assessment predictions concatenated = full-length honest
  probabilities, no data sacrificed (not a small holdout).
- sklearn note (verified 1.9.0): both rf and hist_gbm emit predict_proba incl. under
  class_weight + NaN inputs.


## Done when

-

## Plan

- [ ]

## Log

### 2026-07-18
