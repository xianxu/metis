---
id: 000033
status: open
deps: []
github_issue:
created: 2026-07-13
updated: 2026-07-13
estimate_hours:
---

# GBM overfits hard on Titanic — bug vs regularization defaults vs effective-complexity measure

## Problem

On the real Titanic honest-beat run (kbench#8), the shipped `hist_gbm [title,family]` (iter=100,
max_leaf_nodes=15) scored inner-CV **0.846 → public 0.749** — a ~0.10 gap, far worse than rf/logreg. Even
the *simplest* GBM in the grid overfits this hard. Three questions to resolve, cheapest first.

## Spec

**(a) Bug or genuine overfit?** Rule out a pipeline defect before treating it as a modeling issue:
- Does the rf-ticket config (same features/pipeline, different model) land ~0.78 public? If yes → the
  pipeline is fine and it's GBM-specific (points to overfit, not a bug). (This is the `promote --family rf`
  submit — a shared diagnostic with the immediate fix.)
- Check for a GBM-specific defect: predict-time feature/dtype drift vs train, a fit/predict mismatch, NaN
  handling in `HistGradientBoosting` differing from rf. Compare a held-out slice's GBM train-vs-CV gap.

**(b) If genuine overfit — regularize GBM for small tabular data.** Titanic is 891 rows; a 100-tree,
15-leaf GBM (1500 leaves) memorizes. Options to sweep/default:
- **Lower the grid floor** — the current `max_iter ∈ {100,300}` has no lightly-boosted option; add
  `max_iter ∈ {20, 50}` (a 20-tree GBM is a different animal).
- **Early stopping** — `n_iter_no_change` + `validation_fraction` so GBM stops before memorizing (and
  `n_iter_` becomes a REAL realized-complexity signal — see (c)).
- **Regularization defaults** — `l2_regularization`, `min_samples_leaf`, smaller `max_leaf_nodes`.

**(c) Fix the effective-complexity measure (the metis#19/#21 weakness the run exposed).** GBM complexity is
currently `total realized leaves` — but with a leaf CAP every tree saturates it, so cx = `max_iter ×
max_leaf_nodes` EXACTLY (1500 for all nine `[100×15]` configs whose means span 0.81–0.85). It's the
hyperparameters echoed back, not a measured property, and it's **shrinkage-blind** (lr=0.1 → effective DoF
≪ 1500). Consequences: intra-family parsimony is a **no-op** for GBM (constant cx → degenerates to
argmax-mean → picks the overfitter). Candidate measures: shrinkage-aware effective DoF (Bühlmann–Hothorn
`df(m)=trace(B_m)`, sub-linear in m — #21's cited literature), realized `n_iter_` under early stopping, or
a family-agnostic **train−CV optimism gap** (which also helps other families). NOTE: metis#32 (outer-CV
family selection) removes the *cross-family* need for a comparable complexity — this (c) is about
**intra-family** GBM parsimony working at all.

## Done when

- Bug ruled out (or found + fixed): a controlled check shows the rf-ticket config generalizes on the same
  pipeline, and no GBM-specific predict-time defect remains.
- If overfit: GBM is regularized (grid floor / early stopping / reg defaults) so its honest estimate is
  competitive, not a memorizer; a re-run shows the GBM family's outer estimate no longer far exceeds public.
- GBM complexity **discriminates within the family** (varies with the fit + reflects shrinkage) so
  intra-family parsimony is no longer a no-op — verified on the ledger (GBM configs get distinct, meaningful
  cx that a parsimony rule can rank).

## Plan

- [ ] (spec at claim) (a) bug-vs-overfit check (rf-ticket diagnostic + GBM predict-path audit); (b) GBM
  regularization (grid floor + early stopping + reg defaults); (c) effective-complexity measure (shrinkage-
  aware / realized-iters / optimism-gap) so intra-family parsimony discriminates; re-run + verify.

## Log

### 2026-07-13
- Filed from the kbench#8 honest-beat run (operator: "debug if GBM has a bug or some improvements so it
  doesn't overfit this much"). The complexity-measure weakness was foreseen in metis#21's log ("leaf-count
  decouples from effective DoF across ν") and deferred; the real run made it bite — cx pinned to
  `max_iter × max_leaf_nodes`, constant across nine configs spanning 0.81–0.85 mean → parsimony no-op. Sibling:
  metis#32 (outer-CV selection — the cross-family half; this is the GBM-intrinsic + intra-family half).
