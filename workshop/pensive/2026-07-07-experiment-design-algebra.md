---
status: active
type: pensive
created: 2026-07-07
---

# The experiment-design algebra — resampling & selection as first-class axes

Design substrate for the next workbench chapter (project `metis-v2-experiment-algebra`).
Came out of a design conversation with the operator (2026-07-07) after we shipped the `$any`
unification (metis#17). The trigger: the metis-v1 Titanic winner scored **~0.81 cv → 0.77990
public** — a classic overfit gap. Chasing more features/models while selecting by raw cv-max just
overfits harder. The fix isn't "more knobs"; it's making the workbench **find honest, non-overfit
performance** — and that turns out to be an *algebra* problem, not a pile of loops.

## The core realization: expansion operators have an AXIS KIND + a REDUCER

`$any` already expands a config-space into points. The insight: **data-splitting (CV) is just
another expansion axis** — but a *different kind*, distinguished by how it **reduces**:

- **Config axes** — `$any` (list=untagged / map=tagged, metis#17), `$linear-range`/`$log-range`.
  Expand the *config* space. Reduced by **SELECTION**: you produce candidates and *pick one*
  (argmax of an objective). N points → 1 winner.
- **Resample axes** — `$fold`/`$cv(k)`, `$repeat(n)`, `$bootstrap`. Replicate **one** config across
  data slices. Reduced by **AGGREGATION**: you *average* (mean/std) into one score per config. You
  never "pick a fold" — folds are summed, not selected.
- **Nesting** — an **outer resample axis wrapping a (config-selection over an inner resample axis)**.
  That *is* nested cross-validation, expressed declaratively instead of as three hand-written loops.

So the design is not "add more `$any`". It's: **each expansion operator declares an axis kind, and
each axis carries a reducer** (config→argmax-objective, resample→mean/std). The engine expands the
step-list into a flat run-set `(config × outer-fold × inner-fold)`; metis's content-addressed cache
dedups the shared work; the **ledger reduces axis-by-axis** back to (a) a decision and (b) an honest
estimate. The linear step-list (`adapt → features → cv-split → train → predict`) is **untouched** —
the algebra lives in the sweep/`with` block; the fold-id threads through as a value.

```
current (2 loops):                        target (3 loops, but DECLARATIVE):
  for config in $any-space:                 outer resample  ($fold, aggregate)
    for fold in cv:                           for config in $any-space  (select)
      fit/score                                 inner resample ($fold, aggregate)
  pick argmax  ← selection on ONE split           fit/score
                                              → honest estimate of the whole procedure
```

## Two SEPARATE knobs: estimation vs selection (they are not the same)

The operator drove out the key distinction: **nested CV is a measurement instrument, not a
selection policy.** Its inner loop still picks argmax-inner-cv — the *same* biased criterion. It
faithfully *reports* that the selected config generalizes to 0.78; it does **not** favor a
less-overfitting config. Structural reason: the outer fold must stay **sealed** from selection — the
instant you select on it, it's contaminated and you're back to selection bias one level up. So
estimation and selection are forced onto different data views. They are orthogonal:

- **Estimation** — *how good is my procedure?* Knob: flat-cv vs **nested-cv**. Attacks *selection
  optimism*. (Repeated-cv is a third, orthogonal knob that attacks *variance/noise*, not bias.)
- **Selection** — *which config do I ship?* Knob: the **objective** that `promote` argmaxes.
  - `cv-max` (today) — biased toward overfitters.
  - `mean − λ·std` — penalize configs whose cv swings across folds (fragility ≈ overfit).
  - **1-standard-error rule** (Breiman/CART; tidymodels `select_by_one_std_err`): among configs
    within one SE of the numerical best, pick the **simplest** (fewest features / most reg). This is
    *exactly* "prefer the less-overfitting near-winner" — the operator's intuition, named.

Synthesis: **swap the inner selection objective to 1-SE/robust, and nested CV then honestly
estimates that better policy.** They compose — nested CV measures whatever selection rule you nest
inside it. In algebra terms: the objective *is the reducer of the config-selection axis*, and the
estimator *is* whether a config axis is wrapped by an outer resample axis. Same frame.

## The pipeline consequence: FOLD becomes a first-class threaded value

Today "fold" is hidden *inside* one experiment (`cv-split` → `train`'s internal cv). Two of the new
capabilities force it *out* into the pipeline:

1. **Nested CV** needs an outer data-partition that wraps N whole sweeps — above the config axis.
2. **Leakage-safe features** (the ticket-survival case): a group-survival feature uses the **target**,
   so it must be fit **per fold** (train-fold members only) or it leaks catastrophically (a test
   passenger's feature encodes test labels via the group aggregate → inflated cv that won't
   reproduce). Today `features` runs **once, before `cv-split`, on the whole train** — fine for
   imputation, **wrong** for any target-based feature.

Both need the **fold boundary visible to `features` AND `cv-split` AND `train`** — i.e. fold promoted
from a `train` internal to a first-class value the expansion threads through the pipeline. This is
the central design decision, and it unifies the evaluation work and the fold-aware-feature work: they
are the same restructuring. (sklearn solves it with `Pipeline` fit-per-fold; that's the reference.)

## Prior art (this is a well-trodden design — study it, don't reinvent)

- **tidymodels (R) — the canonical realization of exactly this algebra:**
  - `rsample` makes resamples **first-class data** — `vfold_cv(5)`, `bootstraps()`, and
    **`nested_cv(outside=vfold_cv(5), inside=vfold_cv(5))`** (a table whose rows are outer splits,
    each carrying a nested `inner_resamples` column). Nesting is *data*, built declaratively.
  - `workflow_set(preproc, models)` = the **cross-product** of preprocessors × models (the config axis).
  - `tune_grid(wf, resamples=…, grid=…)` **crosses** the resample axis with the config grid.
  - `collect_metrics()` **aggregates over the resample axis** (mean + std_err per config);
    `select_best()` / **`select_by_one_std_err()`** **reduce the config axis**. Two axis-kinds, two
    reducers, in a shipping system.
  - Refs: tidymodels.org/learn/work/nested-resampling, rsample.tidymodels.org/reference/nested_cv,
    tmwr.org ch.10 (resampling for evaluation).
- **scikit-learn (lighter):** `GridSearchCV(Pipeline(...), param_grid, cv=StratifiedKFold(5))` — `cv`
  is a parameter crossed with the grid, and `Pipeline` makes feature-fitting **fold-aware for free**
  (re-fits per fold). The fold-aware-feature answer, off the shelf.
- **Leakage-aware modeling** is a named research area (e.g. a 2026 "bioLeak: Leakage-Aware Modeling"
  paper surfaced in the search) — target-based feature leakage isn't a metis quirk.

## Open design questions (resolve at spec — M1)

1. **Where do folds live?** A resample object above the sweep (tidymodels `rsample`), vs fold-id as a
   swept dimension in the expansion, vs a new "evaluation harness" layer wrapping the sweep. Must
   make the fold boundary visible to `features`/`cv-split`/`train`.
2. **How does the ledger reduce per-axis?** It currently records one cv per config-point. It needs:
   config axis → keep as rows (to select among); resample axis → collapse to mean/std columns; nested
   outer → one procedure-level estimate row. The ragged CSV codec (metis#8) is the starting point.
3. **Fold-aware features: restructure vs wrap?** Either move `features` *after* `cv-split` (fold-id
   in its `with`), or have `train` accept a fit/transform feature pipeline that CV drives per fold
   (sklearn-Pipeline style). Trade-off: step-graph clarity vs a heavier `train`.
4. **Nested outer vs metis's sweep orchestration + cache.** Within one outer fold the full sweep runs
   (cache dedups shared upstream); across outer folds the *data* differs so cache keys differ → 5
   genuinely independent sweeps (~5× compute). The engine must be aware of this cost axis.
5. **Adaptive samplers (metis#7 `Sampler` seam) slot into the config-selection axis** — the resample
   axis is orthogonal to them. Keep the two axis kinds cleanly separated so a BO sampler can drive
   config selection while resampling stays a fixed aggregation. (This is the payoff of the metis#17
   tagged-`$any` legibility discussion — a hierarchical sampler reads the tagged config structure.)

## Why this is the right frame (and the honest done-when)

`$any` (metis#17) generalized "config choice" into one primitive. This generalizes *one more step*:
the whole **experiment design** — config **and** data-splits **and** the reduce semantics — into one
algebra of a linear step-list + typed operators + per-axis reducers. tidymodels proves it's buildable
and worth building. The honesty test (not just a bigger number): a Titanic run whose **nested-CV
estimate tracks its public score within noise** — the workbench telling the truth about what it
found — while GBM + ensembling + a leakage-safe ticket feature push the honest number up.
