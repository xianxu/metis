# Channel Split — y as a Runner-Scoped Keyed Channel (metis-v3) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the row-cloning nested-CV *seal* with a runner-owned label channel — y becomes a separately-keyed artifact split from X at `adapt`, a CV fold becomes a *domain restriction of y* (by row or by cluster), and label leakage is prevented structurally (X carries no label) for feature reads and by a sanctioned-API-plus-chokepoint for label reads.

**Architecture:** Physically separate the label channel (`id → label`) from the feature frame (`X`) at the first schema'd node. The **runner** owns y's domain: it hands each fold a compact *domain restriction* (which ids' labels are in-scope, with polarity), never a copy of the data. A metis-provided **y-loader** (`ctx.y()`) is the sole sanctioned label reader and enforces the restriction; the full y-artifact lives in a **runner-owned location outside a step's read-root**, so a *direct* read (bypassing `ctx.y()`) fails loudly (a repurposed chokepoint — not the deleted analysis-subset confinement). A terminal **`score` step** is the one declassification point that ever touches held labels. Nested CV becomes the *same experiment* as flat, run under a restriction pair — deleting `analysis_i` row-cloning and `buildFoldExperiment`'s sealed branch. **O(k·N) → O(1)** storage; X-restriction becomes a domain *filter* (prospective only), not a clone.

**Tech Stack:** Go (`cmd/metis` runner + `pkg/experiment|record|ledger|channel` pure cores), Python steps (`metis/steps/*.py` + pure cores in `metis/*.py`), CUE vocabulary (`construct/vocabulary/experiment.cue`), git side-refs for code/spec capture (unchanged).

---

## Revisions

- **2026-07-19 (fresh-eyes review, 3 critical + 4 important folded in):** corrected the overstated "structurally impossible" leakage claim → X-reads structural + label-reads API+chokepoint-enforced, y stored outside the step read-root, e2e must prove the *direct* read fails (C1). Reframed the M2 regression anchor: **prospective** mode reproduces the seal number; transductive diverges *by design* (metis#42), and the shipped **public** score is refactor-invariant (C2). Gave prospective a live X-restriction mechanism (`dataset.X()` honors the restriction under `Estimand.RestrictX()`) and pulled it into M2 (C3). Added **M0 regression support** — metis is classification-only; rogii is RMSE regression (I1). Gave `DomainRestriction` a **polarity** (fit reads `y|A` = complement-of-held; score reads `y|B` = only-held) and corrected the declassification model (inner+outer score are the only held-label readers; fits never see held labels) (I2). Fixed M1's leak-truth to the leaderboard / out-of-engine holdout (I3). Called out the `objective.metric` rewiring `train.fold_score → score.<metric>` (I4). Plus minors: seeded grouped split, the `source`-role scope caveat, `adapt` is workspace-owned (metis ships the demux *primitive*; every `df[TARGET]` reader ports to `ctx.y()`).

---

## Context — the driver, the scope, the anchor

### rogii-first (operator decision, 2026-07-19; rules accepted)

Driven by arena3 = `rogii-wellbore-geology-prediction` (live Featured, $50k, ~5,262 teams, deadline 2026-08-05). Workbench pattern: build the feature when the competition pulls it out of you (as arena2 pulled metis#58/#59). Rogii's structure (digest in the issue `## Log`) makes the demand concrete:

- **Grouped SEQUENCE data**: a directory of **773 per-well CSV pairs** (`horizontal_well` + `typewell`), each a depth-ordered (`MD`, ~1-ft) sequence. The **well** (`WELLNAME`) is the group. Hidden test = ~200 **disjoint** held-out wells.
- **Regression** (RMSE on `TVT`), toe-end **extrapolation** (heel given, `TVT_input=NaN` over the eval zone). Submission `id = {WELLNAME}_{row_index}`.
- **Naive row-CV LEAKS**: adjacent ~1-ft samples within a well are near-identical → random row-split puts a well's rows in both train+val → optimistic score that won't hold on unseen wells. **Correct CV = hold out whole wells + mask the toe-end.** The well is the CV group — the concrete demand for #36's cluster-unit CV.

**Scope boundary.** This plan is the **metis-side infra** (issue #36) + the metis-core **regression support** M0. The **rogii workspace** (kbench: grouped-sequence `get-data`, a grouped-sequence `adapt` step-type, features, model wiring, toe-masking, submission) is the *driver* and belongs to a companion **arena3 project + kbench issues** (arena2 pattern). See Open Questions.

**Regression anchor (risk mitigation).** rogii-first entangles a new ingestion regime with the refactor, so keep a known-good anchor: the metis#35-era honest-beat on titanic/s6e7. **The anchor is PROSPECTIVE mode** (see C2 fix): prospective reproduces the seal's internal CV estimate; **transductive is EXPECTED to diverge** (that divergence is #36's point — metis#42 quantified it); the **shipped public score is refactor-invariant** (ship refits on all rows regardless of the CV mode), so it reproduces trivially and is the coarse smoke check. The refactor is correct iff *prospective's internal estimate* matches the seal AND the public ship score is unchanged.

### Design source of truth

`workshop/pensive/2026-07-14-01-pensive-feature-engineering-algebra-under-cv.md` — the converged theory (two-channel data; `fit∘apply` scope signatures; **the fit boundary is a declassification point, cross-fitting is the declassification policy for y**; aggregate algebraic classes decide fold-recompute cost). **#37** (R-scope constructor algebra: declared scope signatures, aggregate classes, derived placement) is the *next* stage, OUT of this plan — #36 delivers the y-channel + label-scope (S) restriction + cluster unit; #37 layers the feature-scope (R) algebra on top.

---

## Core Concepts

The channel split rests on one inversion: **today the label travels *with* the data and each step self-splits (masks internally); after #36 the label is a separate channel the runner restricts, and X-reading steps never see it.** Two leakage roads follow: an **X-read** road (closed structurally — X has no label column) and a **direct-y-read** road (closed by a chokepoint — the y-artifact is unreachable except through `ctx.y()`).

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `DomainRestriction` (+polarity) | `pkg/channel/restriction.go` | new |
| `ResampleUnit` (row \| cluster) | `pkg/channel/restriction.go` | new |
| `Estimand` (transductive \| prospective) | `pkg/channel/estimand.go` | new |
| `Header` (+`semantics`, +`cluster`) | `pkg/experiment/experiment.go`, `pkg/experiment/shape.go` | modified |
| `#ExperimentShape._meta` (+`semantics`, +`cluster`) | `construct/vocabulary/experiment.cue` | modified |
| regression model kind + RMSE scorer + regression complexity | `metis/model.py` | new (M0) |
| `foldWith` → `restrictionWith` | `cmd/metis/sweep.go:1045` | modified |
| `buildFoldExperiment` (sealed branch removed) | `cmd/metis/sweep.go:962` | modified |
| `analysis_i` row-cloning | `metis/steps/outer_split.py` | deleted (M4) |
| `dropNeeds` / `baseRef` surgery | `cmd/metis/sweep.go:984-1016` | deleted (M4) |

- **`DomainRestriction` (+polarity)** — the compact spec of which ids' labels a step may see: `(partition-ref, held-fold-set, unit, polarity)`. **Polarity is load-bearing (I2):** a `fit` step reads `y|A` (complement — ids *not* in the held folds); a `score` step reads `y|B` (only-held — ids *in* the held fold). `InDomain(id, folds, polarity) bool` is the whole contract; composes by intersection for nesting (`(y|A)|B = y|A∩B`). Pure; the runner computes it, `ctx.y()` consumes it.
  - **Relationships:** N:1 with a partition (`folds.json`); 1:1 with a fold-run (nested = a composed pair). Replaces the physical `analysis_i` subset dir.
  - **DRY rationale:** one restriction serves outer folds, inner folds, the score step's inverse view, and any future bootstrap replicate. Eliminates the `outer_split.py` clone + the analysis-subset confinement (two mechanisms → one).
  - **Future extensions:** per-row `S(k)` (LOO/cross-fit self-exclusion) widens `InDomain` from a fold-set to a per-row predicate — the #37 hook.

- **`ResampleUnit`** — `row` (default; each row its own unit — reproduces today) or `cluster` (hold out whole clusters, keyed by a column). A fold's held set is a set of *units*; `InDomain` maps id → unit → in/out.
  - **DRY rationale:** row-CV is the degenerate cluster-CV (unit = the id). One split path. Kills the silent row-exchangeability assumption every resampling surface makes.
  - **Future extensions:** stratified-group (rogii's `StratifiedGroupKFold`), spatial-block, time-forward — all are "how units map to folds," an axis on the split.

- **`Estimand`** — `transductive` (default — X fold-invariant, only y restricted) vs `prospective` (mask labels AND filter the held rows' X — reproduces today's row-hiding, the M2 regression anchor). Pure decisions `RestrictX() bool` + `Hoist(scopeSig) bool`. **Q4 lives here (C3 fix):** `RestrictX()==true` makes `dataset.X()` honor the same `DomainRestriction` as `ctx.y()` — a **domain filter, not a clone** (so it survives the M4 deletion of `analysis_i`).
  - **DRY rationale:** makes the estimand *declared*, not fold-count-implied (metis#42 showed the seal silently under-measures vs the transductive deployment).

- **`Header` (+`semantics`, +`cluster`)** — two new optional header fields, double-sourced (Go struct + CUE `_meta`, drift-guarded by `TestParse_ConformsToCUE`). ⚠️ `ParseShape` uses `KnownFields(true)` — both MUST be added to `Header` + `_meta` or every shape fails to parse.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| y-loader (`ctx.y()` domain-enforcing) | `metis/io.py` + `metis/channel.py` | new | the label channel read |
| y-channel chokepoint (direct-read guard) | `metis/io.py` (repurposed read-root) | modified | filesystem confinement |
| X/y demux **primitive** | `metis/io.py` + `metis/dataset.py` | new | dataset serialization |
| restriction injector (runner) | `cmd/metis/sweep.go` (`restrictionWith`) | modified | fold→step wiring |
| `metis/score` step | `metis/steps/score.py` + `metis/model.py` | new | metric subprocess |
| `metis/train` → fit/predict | `metis/steps/train.py` + `metis/model.py` | modified | model subprocess |
| regression model + predict path | `metis/model.py` + `metis/steps/predict.py` | modified | model subprocess |

- **y-loader (`ctx.y()`)** — the sole sanctioned label reader and the access-control enforcement point. Given the current `DomainRestriction`, returns labels only for in-domain ids (by polarity); out-of-domain ids are *absent* (NaN), so a reduce over labels cannot include a held one. `fit_mask ≡ k ∈ dom(y)` is derived here. **Q2 (C1-corrected):** the full y-artifact is written once (O(1)); the runner passes the compact restriction (via `with`→Kpre for cache-distinctness); the loader applies it. This does NOT make leakage "structurally impossible" on its own — it's the *sanctioned API*; the structural teeth are (a) X has no label column and (b) the chokepoint below.
  - **Injected into:** every step that consumes labels (`fit`, `score`, any target-encoding feature — **every current `df[TARGET]` reader ports to `ctx.y()`**, a real migration cost, M-c).

- **y-channel chokepoint** — the y-artifact lives in a **runner-owned location outside a step's read-root**; a *direct* read (`read_parquet(<y_path>)`, bypassing `ctx.y()`) fails loudly. This is the **repurposed** metis#23 confinement (`METIS_READ_ROOT`/`exp_path`) — NOT deleted, re-pointed from "confine reads to the `analysis_i` subset" to "the y-channel is reachable only via the domain-restricting loader." **The honest guarantee (C1):** structural for X-reads (X carries no target column); for label-reads, API-enforced + this chokepoint refusing the direct road. **The leakage e2e MUST assert the direct read fails, not just the API road** (else "structural" is untested).
  - **Caveat (M-b):** "X has no label" means "no *declared-target* column"; a `source`-role column (raw column carried into X for feature engineering, `schema.py:20`) carrying label-equivalent info is a workspace-author concern the structural claim is scoped around, not a metis guarantee.

- **X/y demux primitive** — metis provides the *primitive* (`io.save_dataset` writes X with **no target column** + a separately-keyed y-artifact to the runner-owned location); **each competition's `adapt` must USE it** (M-c: `adapt` is kbench-workspace-owned — `kbench/kbench/titanic/adapt.py` etc., not metis). So the demux is a metis primitive realized per-workspace. `dataset.X()` returns features (honoring the restriction under prospective); `dataset.y()` is the domain-enforcing loader.

- **restriction injector** — replaces `foldWith` (which overlaid `folds`+`_fold` and set `METIS_READ_ROOT`). Overlays the `DomainRestriction` (partition-ref + held-fold-set + unit + polarity + estimand) onto each fold-run's `with`, entering Kpre. `buildFoldExperiment` loses its sealed branch — flat and nested build the *same* DAG under different restrictions.

- **`metis/score` step** — the declassification boundary. Input: predictions + the held-fold labels (`y|B`, polarity=only-held); output: a **scalar** (`{<metric>: float}`). **Q3 + I4:** splits `train`'s fit+score monolith; the shape's `objective.metric` rewires `train.fold_score → score.<metric>` and the ledger namespacing follows. **I2 correction:** inner AND outer score steps are the only held-label readers; the inner score runs *during* selection (it IS the selection signal), the outer *after*; **fits never see held labels**; selection consumes only scalars.

- **`metis/train` → fit/predict** — `fit` (label-restricted `y|A`, produces a model) + `predict` (label-free, predictions on all rows — reuse the existing ship-path `predict.py`). Scoring moves to `metis/score`.

- **regression model + predict path (M0, I1)** — metis is classification-only (`model.py:19` MODELS all sklearn Classifiers; `_SCORERS` = accuracy/balanced_accuracy; `predict` on `estimator.classes_`). rogii needs a **regression model kind** + **RMSE scorer** + a **regression predict path** (no `classes_`) + **regression complexity**. A metis-core prerequisite for *all* of rogii — M0, before M1.

### The four mechanism open-Qs — resolved (post-review)

| # | Question | Resolution |
|---|---|---|
| 1 | y-artifact on-disk shape | Keyed columnar `(id, label)` in a **runner-owned location outside the step read-root**; X carries no target column. |
| 2 | runner injects restricted y | **Per-artifact + compact runner-restriction** (via `with`→Kpre), enforced by `ctx.y()`; a **chokepoint** (repurposed `METIS_READ_ROOT`) refuses the direct read. Full y written once (O(1)). |
| 3 | score-step metric contract | `score(predictions, y\|B) → {metric: scalar}`, polarity=only-held; sole declassification boundary; rewires `objective.metric`. |
| 4 | prospective mode's row-drop | `Estimand.RestrictX()` → `dataset.X()` honors the restriction (a domain **filter**, symmetric to `ctx.y()`) — survives the M4 clone deletion. |

---

## Invariant preservation — the correctness contract

The code map surfaced 10 load-bearing invariants the seal guarantees. The channel split must preserve each *without physically cloning rows*:

| # | Seal invariant (today) | Channel-split preservation |
|---|---|---|
| 1 | Assessment rows physically absent in `analysis_i` | Assessment **labels** absent from `ctx.y()`; the y bytes exist on disk but are unreachable except via the loader (the chokepoint) — an honest *API+chokepoint* guarantee, not physical absence (C1). |
| 2 | Shape-identical substitution | **Moot** — no substitution. X is the full frame under **transductive**; under **prospective** X is domain-*filtered* (not cloned). |
| 3 | Read confinement (`METIS_READ_ROOT` chokepoint) | **Repurposed** to guard the y-channel (direct read refused) + channel separation for X-reads. Not deleted — re-pointed. |
| 4 | Sole-road / no bypass (the #35 hole) | **Two roads, two guards:** the static one-road *parse* check catches a `with`-reference road (the #35 `raw: get-data` class); the *direct-read* road is caught by the y-channel chokepoint (a `with`-check cannot, since it's the same `dataset: adapt` reference the legit X-read uses). |
| 5 | Deterministic partition reproduction | `folds.json` minted by a **seeded** splitter; cluster-CV needs a *seeded grouped* split (`StratifiedGroupKFold`/custom — sklearn `GroupKFold` is seedless, M-a). |
| 6 | DAG integrity after surgery | **Moot** — no surgery (`dropNeeds`/`baseRef` deleted); flat and nested share one DAG under a restriction. |
| 7 | Selection sealed, scoring post-selection | **Corrected (I2):** inner+outer score steps are the only held-label readers; **fits never see held labels**; the inner score *is* the selection signal (runs during selection), the outer runs after; selection consumes only scalars. |
| 8 | Per-pass isolation | Preserved — each fold's restriction is independent. |
| 9 | Fold-context cache-distinctness | The `DomainRestriction` enters Kpre (as `_fold` did). |
| 10 | Label travels with data, step self-splits | **Inverted by design** — the inversion *is* the feature. |

The structural **leakage e2e** (Done-when) pins invariants 1/3/4: (a) a feature step reading X cannot see a label (X has no target column); (b) a step attempting a **direct** read of the y-artifact fails at the chokepoint; (c) a `fit` handed `y|A` cannot see a held label (absent).

---

## Milestone spine (review boundaries)

Each `Mx` is an `sdlc milestone-close` boundary. Bite-sized TDD steps are fleshed per-milestone **after** this conceptual model passes operator + plan-reviewer review.

- [x] **M0 — regression support (metis core; I1). DONE 2026-07-19.** Regressor kinds (`rf_reg`/`hist_gbm_reg`/`ridge`) + `rmse` scorer + `is_regression()` + `predict()` branching on `classes_` + `complexity()` reuse + a `decide=offsets` guard, in `metis/model.py`. **12 new unit tests, 57 pass, no regression.** Vocabulary needs no change (`objective.metric` is an open string; `model` is a `$any`-map). The train-step *fold* path is regression-clean (verified); the `predict.py` *submission* path is deferred to M1 (demand-gated). **The full-engine regression run is exercised in M1** with rogii's continuous-target `Dataset` — no such `Dataset`/adapt exists until then, so the engine smoke belongs where the data is (verify-before-claim at the integration boundary). *Prerequisite for rogii; no channel-split content.*

- [ ] **M1 [external: kbench#18] — rogii hits the wall (the demand).** *Its `sdlc milestone-close` lives on kbench#18, not #36 — a gate/dependency in this spine, not a #36 boundary.* `get-data` over the 773 well pairs + a minimal grouped-sequence `adapt` (directory → schema'd `Dataset`, `WELLNAME` group, `TVT_input`/`TVT` roles, using the M0 regression kind) + a naive baseline. Run **naive row-CV** and **demonstrate the leak** — truth reference is the **leaderboard** (submit the row-CV winner, watch it not hold) or a **one-off manual by-well holdout OUTSIDE the engine** (I3: an in-engine grouped estimate *is* M3, so it can't be M1's truth). *Deliverable: a reproducible "row-CV lies here" measurement + the rogii schema/adapt skeleton.*

- [ ] **M2 — channel split core + prospective anchor (metis).** y demux primitive (X without target + keyed y in the runner-owned location) + the `ctx.y()` loader + the y-channel chokepoint; `DomainRestriction` with polarity; split `train`→`fit`/`predict` + new `metis/score`; rewire `objective.metric`; the `Estimand` knob (transductive + prospective) with `dataset.X()` honoring the restriction under prospective; the static one-road parse check. **Anchor (C2): PROSPECTIVE reproduces the titanic/s6e7 seal internal-CV estimate; transductive is expected to diverge; public ship score unchanged.** Leakage e2e proving the **direct-read** road fails (C1). *Large — may sub-split M2a (channel + fit/predict/score, transductive) / M2b (prospective + chokepoint + one-road + leakage e2e) when fleshed.*

- [ ] **M3 — cluster-unit CV (metis).** `ResampleUnit` + `cluster:` header; a **seeded grouped** cv-split (M-a); the restriction holds out whole clusters. **Resolves rogii's wall: group-by-well CV.** Anchor: titanic `cluster: Ticket` reproduces the ticket-group honest estimate; rogii's M1 row-CV-leak closes under well-CV.

- [ ] **M4 — delete the seal + finalize (metis).** Once M2/M3 prove equivalence: **delete** `analysis_i` row-cloning, `buildFoldExperiment`'s sealed branch + `dropNeeds`/`baseRef`, and the analysis-subset confinement (the y-channel chokepoint remains). Confirm O(k·N)→O(1). e2e green.

- [ ] **M5 — acceptance (the open research question).** Rogii under well-CV → honest estimate; operator submits → does it track the leaderboard? Run the titanic `ticket_size` **hoisted (transductive) vs fold-scoped (prospective)** experiment — which honest estimate tracks public (Moscovich-Rosset is inductive-only; the transductive case is verifiably open). Record in the pensive + arena3 project.

---

## Decisions (resolved 2026-07-19, operator)

1. **C1 guarantee** → **O(1) storage + repurposed chokepoint + the "API-enforced, direct-read-refused" framing** (over O(k) physical-absence). Held bytes live once in the runner-owned location; unreachable except via `ctx.y()`; the e2e proves a direct read fails. Keeps the O(k·N)→O(1) win.
2. **arena3 project** → create `metis/workshop/projects/arena3-rogii-wellbore.md` (center-of-gravity metis; spans metis#36 + kbench) + **ONE** kbench "rogii workspace" issue (get-data + grouped-sequence adapt + baseline), not per-step-type.
3. **#37 boundary** → #36 **excludes** the R-scope constructor algebra (scope signatures / aggregate classes / derived placement). #36 = y-channel + label-scope (S) + cluster unit + estimand knob; #37 = the feature-scope (R) algebra on top.
4. **`pkg/channel`** → yes, a new package for `DomainRestriction`/`ResampleUnit`/`Estimand` (distinct concept, single responsibility — ARCH-PURE).
5. **rogii toe-masking** → a rogii-specific `adapt`/feature detail for M1 (demand-gated); generalize to a within-cluster restriction axis only on a second competition's demand.

---

## Next

After operator sign-off on the open questions, flesh bite-sized TDD steps per milestone (M0 first, then M1) as `## Chunk N` sections, re-run the plan-document-reviewer per chunk, then hand off to execution per AGENTS.md §3.
