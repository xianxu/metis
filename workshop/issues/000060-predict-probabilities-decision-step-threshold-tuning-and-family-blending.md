---
id: 000060
status: codecomplete
deps: []
github_issue:
created: 2026-07-18
updated: 2026-07-19
estimate_hours: 1.6
started: 2026-07-18T22:35:41-07:00
actual_hours: 1.98
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

- [x] M1 — decision core (proba/tune/apply/parse, fold_fit 3-tuple) + train/predict wiring +
  dedicated decide frame tests + docs (two honest costs) + milestone-close
- [x] M2 — `metis blend` (weights-only) + close

## Log

### 2026-07-18
- 2026-07-18: closed M1 — 103 python tests + go test ./cmd/metis green (fork + independent re-run); decision core on dedicated 40-row frame; per-fold tuned-vs-argmax bound, offsets.json presence/absence, eager foldless refusal, probabilities + compat anchor (byte-identical argmax) all pass; atlas carries the two honest costs; review verdict: FIX-THEN-SHIP

### 2026-07-19 (M1 implementation + close)
- 2026-07-19: closed — M1: decision layer shipped (104 py + Go green; leaf-local tuning; retroactive M4 sweep proved it live). M2: metis blend 8/8 tests + full suites; execStep.Execute seam; runref slug + literal-path asserted; provenance guard; single-member identity pins clip+offsets. Both milestones fresh-eyes reviewed; deviations recorded in Revisions; review verdict: FIX-THEN-SHIP

- Fork commits dbb881d (decision core; ±4 grid, no-op-anchored strict-improvement tie-break,
  fold_fit 3-tuple both unpack sites, cv_score threads decide) + f53f446 (train/predict
  wiring; eager parse; offsets.json+classes; class-labeled probabilities.csv; compat
  anchor). 103→104 python tests + Go green (independent re-run).
- M1 review FIX-THEN-SHIP; fixes bundled: parse_decide closed key-set (typo'd inner key
  loud + test), anti-vacuity assert on the predict offsets test (the dropped flip-check —
  the vacuously-green lesson closing its own loop), NEW class-mismatch loud test (the
  honesty guarantee path), priors comment corrected, int-normalization comment.
- Deltas logged by fork: docs folded into Task 2 commit; apply_offsets returns indices
  (callers map via classes_); no-op tie-break implemented as init-at-zeros.
- EARLY MERGE (loud, deliberate): M1 publishes ahead of issue close because kbench#15 (the
  M4 decision-grid sweep) depends on it cross-repo — the operator sweep's cohort fingerprint
  should pin a published main commit, not a mutable branch checkout. M2 (blend) continues on
  a FRESH branch name per the #148 no-reuse rule.

### 2026-07-19 (M2 — metis blend)

- Fork commit 4a31b87 on 000060-m2-blend (manual fresh branch — change-code ran at M1; #148
  no-reuse; loud continuation as planned): blend.go + 8 tests + main.go registration.
  execStep.Execute fit exactly as the symbol-level review specified. Blend record embeds all
  shape steps (runref first-slug-wins asserted in-test). Suites: blend 8/8, full Go, 104
  python — independently re-verified. Provenance guard + --allow-mixed live; single-member
  identity + slug-resolution + literal-path tests in.
- Close-review FIX-THEN-SHIP fixes bundled: atlas blend-verb bullet (the docs gate),
  realignColumns order-invariance + column-mismatch tests (the untested permutation path),
  normalizeWeights refuses NaN/±Inf loudly, plan-doc checkboxes ticked (housekeeping).
