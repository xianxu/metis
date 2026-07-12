---
id: 000019
status: done
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-11
estimate_hours: 3.7
started: 2026-07-08T12:01:41-07:00
actual_hours: 6.20
---

# selection objectives — 1-SE rule + mean-std (configurable sweeper select rule, not raw cv-max)

## Problem

The sweeper selects by **raw cv-max** (`argmax-mean`), biased toward overfitters (the max over N noisy
configs inflates + favors fragile high-variance fits). There's no way to prefer a *robust* or *simpler*
config. Nested CV (metis#23) *estimates* the consequence but doesn't change *which* config is picked —
the **select rule** is the actual lever.

## Spec

metis-v2 M2. Empirically forced by the real Titanic acceptance: `argmax-mean` selected the md=8
overfitter (cv 0.844 → **public 0.770**); a shallower md=4 config from the SAME cached ledger →
**public 0.782**. The honest ledger *contained* the better config; greedy cv-max walked past it.
#19 fixes the **selection lever** — the rule that decides *which* config ships — plus the one input
it needs: a **model-reported complexity** so "prefer the simpler near-winner" is grounded in the
*fitted* model, not guessed from hyperparameters.

### Two pieces

1. **The select rule** — a configurable policy over each config's `(mean_score, SE, mean_complexity)`
   (M1a's read-time reduction, now reducing a complexity metric too), replacing the hard-wired
   `argmax-mean`.
2. **Model-reported complexity** — each model class reports its **fitted** model's realized
   complexity; the model step emits it as a per-fold metric, and the sweeper reduces + consumes it.

### A. The select rule

**Config surface — a tagged union that mirrors `driver`.** `objective.select` becomes a labeled sum:
exactly one branch, its params bound to it (you cannot set `tolerance` on `argmax-mean`, and
`pct-loss` cannot omit it):

```yaml
select:
  pct-loss: {tolerance: 0.02}   # %-width band
  # argmax-mean: {}             # raw cv-max (no params)
  # one-std-err: {}             # 1×SE band (no params)
  # mean-std: {lambda: 1.0}     # mean − λ·std re-score
```

This mirrors the **existing `driver` union faithfully**: CUE **optional struct fields**
(`argmax-mean?: {}`, `pct-loss?: {tolerance: float & >0}`, …) with exactly-one enforced **in Go**
(a `Select` struct of `*ArgmaxMean|*OneStdErr|*PctLoss|*MeanStd` pointers + an "exactly one set"
count check, identical to `Driver`'s `single|cv`). The param is bound to its branch because it's a
field of that branch's sub-struct. (Deliberately *not* a CUE closed disjunction — that would be a
sharper but *different* idiom than driver; ARCH-DRY favors the one already in the file. A later
consistency pass could lift both to disjunctions together.)

**Required-explicit; `pct-loss` is canonical.** Validation rejects an omitted `select` (the honesty
gate — every shape states its rule; matches today's behavior). `pct-loss` is the recommended choice
authors reach for (argmax-mean overfits); `argmax-mean` stays valid but documented as simplistic —
kept because the acceptance test *needs* it (show argmax-mean picks md=8 while pct-loss picks a
shallower config over the same ledger).

**The rules (within-family selection policy):**
- `argmax-mean` — highest mean (M1a; mean only; no complexity).
- `mean-std` — argmax of `mean − λ·std` (penalize fold-to-fold fragility ≈ overfit; uses std, **not**
  complexity → never needs the complexity metric).
- `one-std-err` (Breiman) — contention = configs within **1×SE** of the family best; then **minimize
  measured complexity**, tie-break by mean. **Band too tight here:** SE≈0.005 but the real cv→public
  gap was 0.074 (**15×** the SE), so the md=4 config sits below the 1-SE floor. Inherits the
  over-confident inner-CV SE.
- `pct-loss` — contention = configs within **tolerance** (%) of the family best; then **minimize
  measured complexity**, tie-break by mean. Decoupled from SE (~2% floor 0.827 includes the md=4
  config). **The rule expected to recover today's case** (verified empirically — see Done-when).

**Two-level selection (general, not a model special-case):**
- **Group by family** = the `$any`-map (tagged-sum) branch label — the model family. It is recovered
  from the point's resolved **`With` bundling**: a tagged sum always resolves to a single-key map
  `{label: sub}` (e.g. `With["train"]["model"] == {"rf": {...}}`), which `shape.Expand` emits
  unconditionally, even for an empty branch body — specifically a single-key-map at a path that is
  *also a swept* `FreeParam` (a fixed literal `model: {rf: {...}}` with no `$any` is one constant
  model, not a family axis). (This is the robust signal; `FreeParam` alone can't distinguish a tagged
  label from an untagged bare-string alternative. The plan may instead enrich `Expand` to emit the
  discriminants explicitly — decide there.) In `titanic-sweep` the only tagged sum is `train.model` →
  families `logreg`/`rf`; a shape with no tagged sum is one implicit family.
- **Within a family** — the `select` rule chooses that family's winner: the band (SE / % / none) sets
  the contention set; then **minimize the config's one measured-complexity scalar**, tie-break by
  higher mean, then Expand-order (deterministic). No Pareto, no per-axis anything — one number.
- **Across families** — **always `argmax-mean` over the (already-robust) per-family winners** for the
  single ship pick, never a cross-family complexity comparison. This is not a shortcut: an RF is
  non-parametric (no likelihood, no parameter count), so its complexity (realized leaves) is **not
  commensurable** with a logistic regression's (coefficient count) — cross-family selection is an
  *estimation* problem (nested-CV, #23), not a complexity one. It also makes `argmax-mean` a true
  special case: within = argmax-mean, across = argmax-mean ⇒ global argmax-mean (M1a, unchanged).

**Sampler evolution (`pkg/sampler`).** `GridConfigs.Done` returns a **per-family winner map + the
cross-family ship pick** (evolves M1a's single `Winner`), reading each config's
`(mean_score, SE, mean_complexity)`. The per-family set is the honest leaderboard #22 (ensembling)
blends and #23 (nested-CV) estimates one-per-family — group-by-family is the seam the rest of the
project already wanted, not a workaround. `promote` gains a family selector; `driver:single` ships the
cross-family pick. Pure: a DIFFERENT `Done` over the SAME cached fold records — no re-run once
complexity is emitted (the M1a cache makes offline rule-testing free).

**Interface ripple (the plan sizes M1/M2 around this).** The fold-level Sampler output is a bare
`float64` today — carrying per-fold complexity widens it to a `{score, complexity}` struct, rippling
through `FoldScore`/`MeanSE`/`Aggregate`/`GridConfigs`/`Winner` + the `sweep.go` closures. And there
are **two** reduction/selection surfaces, both of which need the complexity metric *and* family
grouping: `pkg/sampler` (in-memory → the shipped `Winner`) and `pkg/ledger` (`AggregateView`/`Best`/
`TopN` + `promote`, the offline CSV leaderboard — today single-metric, **no** family grouping; the
acceptance counterfactual runs over *this* one). Raw-metric *storage* is already plumbed end-to-end
(`map[string]float64` through record→cache→ledger), so "captured + cached" is cheap and true — it's
the *reduction + two-level selection* that ripples, not storage.

### B. Model-reported complexity

**Complexity is measured on the *fitted* model, not predicted from hyperparameters.** The literature
is clear (cost-complexity pruning penalizes realized terminal-node count `|T|`; `2^max_depth`
*overstates* — real trees rarely fill and `min_samples_leaf`/data cap leaves; "effective degrees of
freedom" is a flawed metaphor for adaptive models — Breiman 1984/2001, Janson-Fithian-Hastie 2015,
Leboeuf et al. 2020). Trees prune, linear models regularize, NNs sparsify — so the *realized*
structure is the capacity, and it's only knowable after training.

- **Each model class implements `complexity(fitted_model) → float`** reporting realized complexity:
  `rf` → **mean realized leaf count per tree** (`mean(tree_.n_leaves)` over estimators — *mean, not
  total*: a total scales with `n_estimators` and refolds that capacity-neutral knob (Breiman's LLN)
  back into the number, wrongly ranking 200 trees "simpler" than 500); `logreg` → coefficient count
  (L2 zeroes nothing, so = feature count); GBM (#21) → mean leaves/tree; NN → non-zero params. The
  **model step owns this** (the model class introspects its own fitted object). **Family-specific by
  design:** rf's measure is feature-neutral (why the mean tie-break can recover the more-feature
  config), while logreg's *is* the feature count (so logreg-parsimony genuinely minimizes features —
  textbook Occam; `C` already shrinks). That asymmetry is intended; the measures are
  cross-family-incommensurable (§A).
- **Emitted as a per-fold metric** named `train.complexity` (step-id-namespaced like
  `train.fold_score`; with `#StepManifest` gone, this naming is *convention*, written here — the
  reducer/select find complexity by this name, nothing declares it). This refactors the model step's
  scoring path, which today trains and **discards** the fitted estimator (`fold_score` returns only
  the float) — it must expose the fitted model (or compute complexity inline) and thread the per-class
  `complexity()`. Captured in the fold record, **cached**, **reduced (mean across folds)**.
  Deterministic given `(config, data, seed)` (tree structure is seeded), so cache-sound.
- **The parsimony rules minimize this scalar within the band, tie-break by mean.** One measured number
  per config collapses the earlier Pareto/`{form,basis}`/`2^depth` machinery — and it likely
  **auto-resolves the feature-count question** for rf: with continuous features a tree fills to its
  depth/leaf-size limit regardless of feature count, so realized leaves are ~feature-independent →
  among equally-shallow configs the mean tie-break selects the higher-CV (more-features) config, no
  hand-declared exclusion. **But `minimize` is unforgiving of near-ties** — if the 6-feature config
  has even one more leaf than the 1-feature one, `minimize` re-selects the sparse v1 corner. So
  complexity is compared **with a tolerance (binned)**: configs within ε realized-complexity are
  "equally simple", decided by mean (the same band idea `pct-loss` applies to the score; ε
  plan-pinned). Whether that recovers the more-feature config is **verified over the ledger, not
  asserted** (Done-when).
- **Guard (LOUD, hard error):** a parsimony rule is active but **any swept family's** model class does
  not report complexity → halt **before selection** with a next-action message (each family's
  within-winner needs complexity, not only the eventual cross-family winner). Per-model-class, not
  per-knob. `argmax-mean`/`mean-std` never read complexity, so they never trip it.

**Dropped vs the earlier draft:** the CUE per-knob `#StepManifest` complexity schema, `{form, basis}`
value-functions, the `2^max_depth` magnitude, and Pareto/rank selection — all replaced by one measured
scalar. This **de-entangles #4**: collocated step manifests remain #4's for docs/learn-notes, not
load-bearing for selection.

### Config-surface churn
- `construct/vocabulary/experiment.cue` — `#ExperimentShape.sweeper.objective.select` → the union
  (optional fields, mirroring `driver`).
- `pkg/experiment/shape.go` — `Objective.Select string` → the `Select` struct + exactly-one validate.
- `pkg/sampler` — fold output `float64` → a `{score, complexity}` struct; `FoldScore`/`MeanSE`/
  `Aggregate` reduce complexity too; `GridConfigs.Done`/`Winner` → per-family map + cross-family pick;
  the four rules + the complexity-bin ε.
- `pkg/ledger` — `AggregateView`/`Best`/`TopN` + `cmd promote` gain the complexity column + family
  grouping (the offline leaderboard the acceptance runs over; today single-metric, no grouping).
- `metis/train` + model classes — expose the fitted estimator from the scoring path (today
  discarded); a `complexity(fitted) → float` per model class (rf, logreg now); emit `train.complexity`
  per fold.
- Shapes: `select: argmax-mean` → the union — `metis/testdata/experiment/titanic-baseline-shape.md`;
  `kbench .../titanic-sweep.md` (→ `pct-loss: {tolerance}`) + `titanic-sweep-smoke.md`.
- Docs: `kbench .../titanic-sweep.md` prose + `RUNBOOK-sweep.md` (STALE for v2: `--sort
  train.cv_score` → `train.fold_score` + the select rule).

### Scope boundaries (non-goals)
Cross-family complexity comparison (unsound — non-parametric RF, incommensurable units; #23 estimates
instead); nested-CV `driver:cv` (#23); adaptive samplers (the ask/tell seam, #7); static/pre-training
complexity estimation (YAGNI — the grid trains every config anyway, so measured suffices); the #4
collocated-manifest catalog (de-entangled — docs/learn-notes stay #4).

### ARCH
Pure `pkg/sampler` (**ARCH-PURE**): the select rule is a re-reduction over cached fold records —
complexity is just another cached metric, no new IO in the hot path. **ARCH-simplicity:** one measured
scalar per config collapses the per-knob/Pareto/magnitude machinery into "minimize one number." The
`select` union reuses the `driver` idiom (consistency). Grounded in the model-selection literature
(realized `|T|`, Breiman's LLN for n_estimators-neutrality, cross-family incommensurability).

## Done when

- `objective.select` is a **required tagged union** (argmax-mean | one-std-err | pct-loss | mean-std),
  params bound per branch; validate rejects omission + multi-branch (mirrors driver).
- Each swept model class reports its fitted complexity (`rf` realized leaves, `logreg` feature count);
  `metis/train` emits it per fold; the reducer aggregates it; it round-trips through the cache.
- Over the ledger **with complexity emitted** (re-fit over the warm data cache — cheap, no creds),
  `pct-loss` **selects a shallower rf config than `argmax-mean`'s md=8**, and the specific pick (incl.
  its feature count) is **reported and verified against the ledger, not asserted** (per-rule table).
  If the verified pick is the sparse corner rather than the more-feature config, the complexity-bin ε
  is tuned (or the outcome documented + a fallback chosen) — not left to chance.
- Both selection surfaces carry it: `GridConfigs.Done` (ship) **and** the offline path group by family
  and expose complexity; `Done` returns the per-family leaderboard + cross-family pick. *(Shipped: the
  offline family-grouped surface is the new `metis ledger select` command — reusing the same pure
  `SelectConfigs`+`FamilyOf` — NOT a `promote --family` flag. `promote` keeps raw `--best`/`--point`;
  the `ledger select`↔in-memory DRY property is tested via identical path-qualified family keys. See the
  plan `## Revisions`.)*
- A parsimony rule + **any swept family** whose model class does not report complexity → **hard error
  before selection** with a next-action message.
- Each rule unit-tested; `complexity()` per model class tested.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.5 impl=0.4
item: smaller-go-module      design=0.2 impl=0.3
item: smaller-go-module      design=0.2 impl=0.3
item: smaller-go-module      design=0.2 impl=0.3
item: smaller-go-module      design=0.2 impl=0.4
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
item: atlas-docs             design=0.05 impl=0.05
design-buffer: 0.15
total: 3.70
```

Design pre-settled (extensive brainstorm + spec + 2 spec-review rounds + an RF-complexity literature
pass + a plan-review round) → design near the floor. **M1**: greenfield `pkg/sampler/select.go` (the
pure `SelectConfigs` rule + `familyOf` — the keystone); a sampler type-widening/threading `smaller`
(fold `float64`→`FoldOutcome`, `Done`→`SweepResult`); the `shape.Select` union + CUE + 3 shape
migrations `smaller`. **M2**: `metis.model.complexity` + per-fold emit `smaller`; ledger complexity +
offline `metis ledger select` + guard + verified acceptance `smaller`. Two `milestone-review` (M1, M2
boundaries) + a small atlas note. Impl at 40%-of-v2 (v3.1); +15% thorough-plan buffer.

## Plan

Durable plan: **`workshop/plans/000019-selection-objectives-plan.md`** (Core-concepts tables + TDD
tasks, grounded in code recon; passed a fresh-eyes plan review). Two review boundaries:

- [x] **M1** — Select rule + sampler evolution: `shape.Select` tagged union (mirrors `driver`) + CUE +
  shape migrations; pure `SelectConfigs` (group-by-family → band → ε-binned min-complexity → mean
  tie-break; cross-family argmax-mean) + `familyOf`; fold-output widening `float64`→`{score,complexity}`;
  `GridConfigs.Done`→`SweepResult` threaded through the driver. (Complexity wired as 0 e2e; rule
  unit-tested with hand-built stats incl. the corner regression.) `sdlc milestone-close M1`.
- [x] **M2** — Measured complexity: `metis.model.complexity()` per class (rf mean-leaves, logreg
  coef-count) emitted per fold → cache → ledger; offline `metis ledger select` reuses `SelectConfigs`;
  the parsimony-guard; the verified acceptance counterfactual (per-rule table, tune ε if it lands on
  the sparse corner). `sdlc close M2`.

## Log


- 2026-07-09: closed M1 — M1 green: go build+test+vet ./... all ok (incl. hermetic sweep+ship e2e w/ real python steps); objective.select union parses+validates (exactly-one, param bounds); pure SelectConfigs corner-regression TestSelect_PctLoss_TieBreaksToMean passes w/ ε=0.10; drove smoke shape through real binary w/ select:{pct-loss} → per-family winners logreg 0.7935/rf 0.8103, cross-family argmax→rf shipped ok (complexity 0.0 until M2); review verdict: FIX-THEN-SHIP
- 2026-07-09: M1 FIX-THEN-SHIP applied (0 Critical, 3 Important + minors): (1) updated stale authoring doc `construct/datatype/experiment-shape.md` select→tagged-union (hidden doc consumer — bare scalar now rejected); (2) added minimize-direction tests (`TestSelect_PctLoss_MinimizeDirection` + `TestSelect_MeanStd_MinimizeDirection` — the whole minimize band/parsimony/mean-std-penalty path was unexercised) + `mean-std lambda<0` validation test; (3) plan `## Revisions` reconciling the Core-concepts table with as-built shapes. **Amended plan Task 12 (guard against a silent M2 DRY break):** the offline ledger `Family` MUST match `familyOf`'s path-qualified format `train.model=rf`, not bare `rf`. Guard framing → post-fold/pre-selection. Whole module green after fixes.
### 2026-07-07
- Filed as metis-v2 M2. The **selection** knob (separate from estimation/metis#23). Design in the pensive.
### 2026-07-07 (design converged)
- Home clarified: the select rule is INTERNAL to the black-box sweeper and consumes `(mean, SE)` from
  M1a's read-time reduction (not a "ledger objective"). Prior-art survey: 1-SE is uncontested across all
  six frameworks — our sharpest differentiator; parsimony ordering falls out of the tagged `$any` tree.
### 2026-07-08 (spec — 3 open knobs resolved)
- **select = tagged union** mirroring `driver` (params bound per branch; ~~CUE closed disjunction~~
  [superseded — v2 uses CUE optional-fields, not a disjunction; see the v2 review entry below] + Go
  pointer-struct + "exactly one" validate). **First manifests = swept step-types only** (metis/train +
  titanic/features); missing-complexity is a **hard error** scoped to swept free params under a
  parsimony-consuming rule (`one-std-err`/`pct-loss`); `const` = swept-but-neutral. **select
  required-explicit, `pct-loss` canonical** (argmax-mean valid-but-discouraged, kept for the acceptance
  counterfactual).
- **Corrected** the pensive's `C: linear·inverse` → `linear·value` (small C = strong reg = simpler ⇒
  complexity increases with C); `inverse` retained for genuine penalty-weight knobs.
- Two-level selection: `select` rule chooses WITHIN family; across families always `argmax-mean` over
  the robust winners ⇒ argmax-mean is a true special case. Parsimony = Pareto per-axis (rank-invariant)
  + rank-tie-break; rules consume only the monotone direction today (form/scale declared-for-forward).
- Full converged design: `workshop/pensive/2026-07-08-select-rule-step-param-schema.md` (see its
  `## Revisions`). Spec written this session; next gate is the durable plan (writing-plans) then
  `sdlc change-code`.
### 2026-07-08 (spec v2 — measured complexity, after spec-review + RF-literature research)
- **Fresh-eyes spec review traced the rule over the real cached ledger** and found the drafted
  mechanism shipped an unvalidated corner (md=4/nfeat=1), not the 0.782 config — because feature-count
  was a 2nd complexity axis and multi-axis Pareto drove to the joint corner. It also caught: "mirrors
  driver" was false at the CUE layer (driver is optional-fields + Go check, not a disjunction); the
  family discriminant is not cleanly recoverable from `FreeParam` (use `With` bundling); the
  per-knob undeclared-knob guard was under-specified (path mapping, family-axis exemption).
- **RF-complexity literature research** (Breiman 1984/2001, Probst-Wright-Boulesteix 2019,
  Janson-Fithian-Hastie 2015, Leboeuf et al. 2020, tidymodels): tree complexity = realized leaf count
  (`2^depth` overstates); n_estimators capacity-neutral (LLN); feature-count dominated by tree size
  (near-neutral for continuous features); cross-family param-count **unsound** (RF non-parametric);
  tidymodels doesn't compute complexity — you *declare the ordering*.
- **Pivot (operator-led): complexity is MEASURED on the fitted model, not declared per-knob.** Each
  model class reports `complexity(fitted) → float` (rf realized leaves; logreg feature count); emitted
  per-fold, cached, reduced; the select rule minimizes it within the band, tie-break mean. This drops
  the whole CUE per-knob schema / `{form,basis}` / `2^depth` / Pareto machinery (one scalar), fixes
  the corner for the right reason, de-entangles #4, and keeps cross-family = argmax-mean (units
  incommensurable). Spec rewritten (v2). Re-running the fresh-eyes review next.
### 2026-07-08 (spec v2 re-review — folded in)
- Re-review confirmed all **4 prior blockers genuinely fixed** (not reworded): union mirrors driver's
  optional-fields form (verified vs shape.go); family from `With` bundling (verified vs Expand's
  tagged-sum `{label: sub}`); per-model-class guard replaces the per-knob one. New fixes folded into
  the spec: **rf complexity = mean leaves/tree, not total** (total refolds n_estimators, breaking
  LLN-neutrality); **guard fires for ANY swept family** (not just the winner), before selection;
  **complexity binned with ε** so near-tie leaf counts tie → mean tie-break can recover the
  more-feature config (the "minimize is unforgiving of near-ties" risk that re-invites the v1 corner);
  **interface ripple made explicit** — fold output `float64`→struct, and BOTH surfaces (`pkg/sampler`
  ship-path + `pkg/ledger`/`promote` offline) need complexity + family grouping (storage already
  plumbed; reduction+selection ripples); **`train.complexity` naming is convention** (schema dropped);
  emission refactors the scoring path to expose the discarded fitted estimator; logreg-vs-rf
  feature-neutrality asymmetry stated. Remaining items are plan-level (M1/M2 sizing around the ripple).
### 2026-07-09 (M2 built — measured complexity + VERIFIED acceptance)
- 2026-07-09: closed — Issue done: both milestones shipped (M1 select machinery + M2 measured complexity), both Review-Verdict:FIX-THEN-SHIP with fixes applied. go test ./... + go vet green (9 pkgs), pytest 46 + kbench 36. Independently verified acceptance over real Titanic ledger: pct-loss ships rf md=4/6-feat (public 0.782) over argmax-mean md=8 (public 0.770) — the differentiator works. Project row ticked in brain repo (cross-repo; --no-project). est 3.7h/actual 6.20h.; review verdict: SHIP
- 2026-07-09: closed M2 — M2 DONE + independently verified: go test ./... + go vet ./... green (9 pkgs), uv run pytest 46 + kbench 36 passed. INDEPENDENTLY re-ran acceptance offline via metis ledger select over the real 891-row Titanic ledger (sweep 4b90538): pct-loss ships rf md=4/all-6-features (cx 14.6 -> public 0.782) vs argmax-mean md=8/3-feat (cx 66.3 -> public 0.770) — recovers shallower regime, NOT sparse nfeat=1 corner; one-std-err confirms band-too-tight. Measured complexity (rf mean leaves, logreg coef count) per-fold+cached+reduced; guard hard-errors on parsimony+unmodeled family; SelectConfigs reused by in-memory ship + offline ledger w/ identical path-qualified family keys. Project file updated in brain repo (separate; cross-repo). est 3.7h/actual 6.20h (over: estimate assumed design pre-settled; this session did full design + 2 spec reviews + literature + plan + review).; review verdict: FIX-THEN-SHIP
- 2026-07-09: M2 FIX-THEN-SHIP applied (0 Critical, 1 Important): the Spec/Plan promised `promote --family` but the offline family-grouped surface shipped as the new `metis ledger select` command instead (purpose met — the acceptance runs over it; `promote` keeps raw `--best`/`--point`). Reconciled: Spec Done-when item note + plan `## Revisions` recording the substitution + atlas/experiment.md now documents `ledger select`. Deferred minor (in lessons + plan Revisions): offline `ledger select` over a >1-`sweep_sha` ledger misdiagnoses the guard error → wants a "multiple cohorts, scope with --sweep" warning (follow-up). Whole module still green.
M2 landed TDD (Tasks 9–14): `metis.model.complexity` (rf mean leaves/tree via `fold_fit` — one
fit feeds score+complexity; logreg coef count); `train` emits `complexity` per fold; `runPipelineFold`
threads it (`FoldOutcome{Score,Complexity,HasComplexity}`); `AggregateView` means every metric column;
`metis ledger select --rule R` applies the pure `SelectConfigs` OFFLINE (2nd consumer — reuses
exported `sampler.FamilyOf` by matching each aggregate row to its Expanded Point, so both surfaces key
families identically `train.model=rf`); `GuardComplexity` (parsimony rule + any swept family lacking
complexity → hard error) wired into both surfaces.

**VERIFIED acceptance (real 891-row Titanic, re-fit over the warm `.metis-cache` — get-data/adapt/features
HIT, no creds; 210 train folds re-fit; sweep_sha `4b90538`). Per-rule ship pick over the SAME ledger:**

| rule | ship pick | mean | complexity |
|---|---|---|---|
| argmax-mean | rf **md=8** n=500 [title,family,age] | 0.8440 | **cx 66.3** (the overfitter → public 0.770) |
| one-std-err | rf md=8 (same) | 0.8440 | cx 66.3 |
| **pct-loss** | rf **md=4** n=200 **[all 6 features]** | 0.8339 | **cx 14.6** (→ public 0.782) |
| mean-std | rf md=8 (variance penalty insufficient) | 0.8440 | cx 66.3 |

**pct-loss recovered rf md=4 (shallower than argmax-mean's md=8) — and it's the 6-FEATURE config, NOT
the sparse nfeat=1 corner** (the v1 failure): measured rf complexity is ~feature-independent (leaf count,
not feature count), so the 2%-band + min-complexity + mean-tie-break lands on the higher-CV 6-feature
md=4. **No ε-tuning needed** (ε=0.10 held). The 4.5× complexity ratio (66.3 vs 14.6 leaves) is the real
overfitting-capacity signal; `one-std-err` empirically confirms the "1-SE band too tight" finding (md=4
at 0.8339 is 2×SE below best → outside the floor → md=8). Reported, not asserted — verified over the
cached ledger via `metis ledger select`. RUNBOOK de-staled to v2 (kbench 14b6334).
