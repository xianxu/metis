---
id: 000060
status: working
deps: []
github_issue:
created: 2026-07-18
updated: 2026-07-18
estimate_hours: 1.6
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


## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.2 impl=0.25
item: smaller-go-module   design=0.1 impl=0.2
item: atlas-docs          design=0.0 impl=0.05
item: smaller-go-module   design=0.15 impl=0.15
item: milestone-review    design=0.0 impl=0.2
item: milestone-review    design=0.0 impl=0.2
design-buffer: 0.15
total: 1.57
```

(Items: decision core+unit tests · train/predict wiring+step tests · docs · M2 blend verb ·
2 review boundaries. Buffer 0.15: reviewed plan doc.

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.)

## Done when

- M1: `decide: {offsets: {holdout}}` on `metis/train` — leaf-local tuned per-class
  log-offsets (aux-holdout inside the fold's training rows; assessment never enters tuning;
  the sealed sweep measures fit+tune as ONE procedure), eager-loud parse on every path;
  `fold_score`/`cv_score` are after-decide scores; ship refit persists `offsets.json`
  (+classes); `predict` always emits `probabilities.csv` (class-labeled columns) and applies
  validated offsets; argmax-absent behavior byte-identical (no re-key, compat anchor test).
  Unit + step tests on a dedicated decide frame (legal internal split, informative feature);
  full python + Go suites green; atlas carries the two honest costs (SE inflation, aux/main
  mismatch) and the 2-fits price.
- M2: `metis blend` — weights-only averaging of promoted runs' probabilities + the shape's
  submission step on the result; leaderboard-measured (no in-sweep OOF for blend — accepted).

## Revisions

### 2026-07-19 — leaf-local pivot (plan review; supersedes the Spec's step-type sketch)

The Spec's `metis/decide` STEP + full-OOF tuning ("not a small holdout") + per-fold
probability emission are superseded by the LEAF-LOCAL design: offsets are a fitted
parameter learned inside each leaf's training rows (aux 80/20 holdout → tune on held-out
probabilities → main model fitted on all training rows), scored on the untouched assessment
fold. Why: the step design needs per-config OOF aggregation ABOVE the leaf — engine surgery
plus a new honesty seam; leaf-local gets the same honesty from the EXISTING seal (the
impute-median fitted-parameter precedent) at 2 fits/leaf. Accepted losses, recorded: offsets
tune on ~20% of leaf training rows (higher per-leaf variance → honest SE inflation; the
1-SE band widens for offsets configs); aux/main probability mismatch (tuned on the 80%-fit
model, applied to the 100%-fit — no leakage, standard CV-style pessimism); blend (M2) has
no in-sweep OOF material — leaderboard-measured only.

**Naming anchor (operator asked):** this is the classic statistical-decision-theory split —
INFERENCE (estimate p(y|x); metric-independent) vs DECISION (choose labels to maximize a
declared objective). The mechanism is the **cost-sensitive plug-in decision rule** ("plug-in"
= threshold/reweight estimated probabilities; "threshold moving"/"prior correction" in
applied ML). Balanced accuracy = accuracy under a uniform prior = the DIAGONAL cost matrix
(1/π_k); the offsets rule is exactly that diagonal case, and a future full cost-matrix rule
(`decide: {costs: ...}`) generalizes it without touching the sweeper — the sweeper only ever
sees a scalar per-fold score of the declared procedure.

## Plan

Durable plan: `workshop/plans/000060-decide-step-plan.md` (fresh-eyes reviewed; 8 findings
folded; the honesty deviation confirmed sound by the reviewer).

- [ ] M1 — decision core (proba/tune/apply/parse, fold_fit 3-tuple) + train/predict wiring +
  dedicated decide frame tests + docs (two honest costs) + milestone-close
- [ ] M2 — `metis blend` (weights-only) + close

## Log

### 2026-07-18
