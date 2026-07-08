---
id: 000019
status: working
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-08
estimate_hours:
started: 2026-07-08T12:01:41-07:00
---

# selection objectives — 1-SE rule + mean-std (configurable sweeper select rule, not raw cv-max)

## Problem

The sweeper selects by **raw cv-max** (`argmax-mean`), biased toward overfitters (the max over N noisy
configs inflates + favors fragile high-variance fits). There's no way to prefer a *robust* or *simpler*
config. Nested CV (metis#23) *estimates* the consequence but doesn't change *which* config is picked —
the **select rule** is the actual lever.

## Spec

metis-v2 M2. Empirically forced by the real Titanic acceptance: `argmax-mean` selected the md=8
overfitter (cv 0.844 → **public 0.770**); the md=4 counterfactual from the SAME cached ledger →
**public 0.782**. The honest ledger *contained* the better config; greedy selection walked past it.
#19 fixes the **selection lever** — and the per-step **complexity schema** the robust rules consume
(the #4 "collocated step manifests" facet, seeded here for the swept steps).

### Two pieces

1. **The select rule** — a configurable policy over the sweeper's per-config `(mean, SE)` (M1a's
   read-time reduction), replacing the hard-wired `argmax-mean`.
2. **The complexity schema** it reads — a per-step-type declaration of each knob's
   overfitting-capacity, so parsimony ("prefer the simpler near-winner") is grounded in step-owned
   semantics, not a guess.

### A. The select rule

**Config surface — a tagged union (mirrors `driver`).** `objective.select` becomes a labeled sum:
exactly one branch, its params bound to it (you cannot set `tolerance` on `argmax-mean`, and
`pct-loss` cannot omit it):

```yaml
select:
  pct-loss: {tolerance: 0.02}   # %-width band
  # argmax-mean: {}             # raw cv-max (no params)
  # one-std-err: {}             # 1×SE band (no params)
  # mean-std: {lambda: 1.0}     # mean − λ·std re-score
```

CUE (closed disjunction → exactly-one, sharp diagnostics); Go (`Select` struct of
`*ArgmaxMean|*OneStdErr|*PctLoss|*MeanStd` pointers + an "exactly one set" validate check) — both
mirror the existing `driver` union (ARCH-DRY: same idiom, not a new pattern).

**Required-explicit; `pct-loss` is canonical.** Validation rejects an omitted `select` (the honesty
gate — every shape states its rule; matches today's behavior). `pct-loss` is the recommended choice
authors reach for (argmax-mean overfits); `argmax-mean` stays valid but documented as simplistic —
kept because the acceptance test *needs* it (show argmax-mean picks md=8 while pct-loss picks a
simpler config over the same ledger).

**The rules (within-family selection policy):**
- `argmax-mean` — highest mean (M1a; mean only).
- `mean-std` — argmax of `mean − λ·std` (penalize fold-to-fold fragility ≈ overfit; uses std, **not**
  complexity → never triggers the missing-complexity error).
- `one-std-err` (Breiman) — contention = configs within **1×SE** of the family best; parsimony picks
  the simplest. **Band too tight here:** SE≈0.005 but the real cv→public gap was 0.074 (**15×** the
  SE) → md=4 (0.834) sits below the 1-SE floor (0.839). Inherits the over-confident inner-CV SE.
- `pct-loss` — contention = configs within **tolerance** (%) of the family best; parsimony picks the
  simplest. Decoupled from SE (~2% floor 0.827 includes md=4). **The rule that actually recovers
  today's case.**

**Two-level selection (general, not a model special-case):**
- **Group by family** = the `$any`-map (tagged-sum) branch discriminant, read straight off
  `shape.FreeParam` (a tagged coord is the branch label whose deeper coords bundle under
  `path.label.*`). In `titanic-sweep` the only tagged sum is `train.model` → families `logreg`/`rf`;
  a shape with no tagged sum is one implicit family.
- **Within a family** — the `select` rule chooses that family's winner: band (SE / % / none) sets the
  contention set; **parsimony** picks within it — **Pareto per complexity-axis** (scale-free, so only
  each axis's monotone *direction* matters), **sum-of-normalized-ranks only to break
  Pareto-incomparables**.
- **Across families** — **always `argmax-mean` over the (already-robust) per-family winners** for the
  single ship pick, never a cross-family complexity comparison (no principled currency to compare
  logreg-C to rf-depth; that's an *estimation* problem → nested-CV #23). This makes `argmax-mean` a
  true special case: within = argmax-mean, across = argmax-mean ⇒ global argmax-mean (M1a, unchanged).

**Sampler evolution (`pkg/sampler`).** `GridConfigs.Done` returns a **per-family winner map + the
cross-family ship pick** (evolves M1a's single `Winner`). The per-family set is the honest
leaderboard #22 (ensembling) blends and #23 (nested-CV) estimates one-per-family — group-by-family is
the seam the rest of the project already wanted, not a workaround. `promote` gains a family selector;
`driver:single` ships the cross-family pick. Pure: a DIFFERENT `Done` over the SAME cached
`(mean, SE)` — no re-run (the M1a cache makes offline rule-testing free).

### B. The complexity schema (the #4 facet, seeded here)

**Central CUE `#StepManifest`** in `construct/vocabulary/step-manifest.cue` (sibling to
`#ExperimentShape`): `knobs: [path]: {type, domain, complexity}` plus #4's
`summary`/`inputs`/`outputs`/`learn-notes`.

**Per-step `.md` sidecar** next to each executable (`steps/metis/train.md`,
`kbench/steps/titanic/features.md`), frontmatter conforms to `#StepManifest`, drift-guarded by a
merge-check (as shapes conform to `#ExperimentShape`). **Steps stay files** (not promoted to dirs).

**Complexity = a per-knob value-function** `{form: const|linear|log, basis: value|count|inverse}`,
higher = more overfitting-capacity:
- `metis/train`: `max_depth` → `linear·value`; `C` → `linear·value` **[correction, below]**;
  `n_estimators` → **`const`** (more trees ≠ more overfit capacity — complexity means
  overfitting-capacity, not resource size).
- `titanic/features`: feature-set → `linear·count` (more features = more capacity).

**Correction from the pensive shorthand.** The pensive labeled `C` as `linear·inverse (more reg =
simpler)`. Taken literally that inverts it — sklearn's `C` is the *inverse* of regularization
strength, so **small C = strong reg = simpler**, i.e. complexity must *increase* with C
(`basis: value`). `basis: inverse` (complexity ↓ as knob ↑) is retained in the vocabulary for a
genuine penalty-weight knob (an explicit `alpha`/`lambda`), but C is the classic trap — precisely why
the step owns its knob semantics: `metis/train` declares C's direction correctly once, engine never
guesses.

**What the rules consume today = the monotone direction only.** Pareto + rank-tie-break are invariant
to any monotone transform, so `form: linear|log` and the exact scale do **not** affect selection yet —
only each axis's direction does. `form`/scale is declared for a future complexity-penalty rule and to
document intent (declared-not-yet-consumed; the *direction* IS consumed now — not aspirational status).

**LOUD on an undeclared swept knob — a hard error.** When a **parsimony-consuming rule**
(`one-std-err`/`pct-loss`) is active and a **swept free parameter** lacks a complexity declaration,
halt with a next-action message (name the knob + the fix) — a silently-dropped parsimony axis gives a
quietly-wrong winner. `const` is the explicit "swept but complexity-neutral" declaration
(n_estimators) — distinct from omission (= undeclared = error). `argmax-mean`/`mean-std` never read
complexity, so they never trigger it. Steps with no swept free parameters (get-data, submission, …)
are never implicated — their manifests are #4's tail.

**First manifests (in #19's scope): only the swept step-types** — `metis/train` + `titanic/features`.
The full per-step catalog / learn-notes stays #4.

### Config-surface churn
- `construct/vocabulary/experiment.cue` — `#ExperimentShape.sweeper.objective.select` → the union.
- `pkg/experiment/shape.go` — `Objective.Select string` → the `Select` struct + validate.
- `pkg/sampler` — `GridConfigs.Done`/`Winner` → per-family map + cross-family pick; the new rules.
- Shapes: `select: argmax-mean` → the union — `metis/testdata/experiment/titanic-baseline-shape.md`;
  `kbench .../titanic-sweep.md` (→ `pct-loss: {tolerance}`) + `titanic-sweep-smoke.md`.
- Docs: `kbench .../titanic-sweep.md` prose + `RUNBOOK-sweep.md` (STALE for v2: `--sort
  train.cv_score` → `train.fold_score` + the select rule).

### Scope boundaries (non-goals)
Cross-family complexity scalar (no currency — #23 estimates instead); the full #4 step catalog (#4
tail); nested-CV `driver:cv` (#23); adaptive samplers (the ask/tell seam, #7); a complexity-penalty
select rule that consumes `form`/scale (forward).

### ARCH
Pure `pkg/sampler` + CUE + manifests, no new IO in the hot path (**ARCH-PURE**); the CUE
`#StepManifest` is the single source — Go, Python, and the merge-check derive from it (**ARCH-DRY** /
single-source); `select` reuses the `driver` tagged-union idiom (consistency).

## Done when

- `objective.select` is a **required tagged union** (argmax-mean | one-std-err | pct-loss | mean-std),
  params bound per branch; validate rejects omission + multi-branch (mirrors driver).
- Over the **cached** `titanic-sweep.ledger.csv` (no re-run), `pct-loss` **demonstrably selects a
  lower-complexity config** (shallower rf depth) than `argmax-mean`'s md=8 — the offline
  counterfactual, reported per-rule (the mechanism, not a hard-coded config).
- `GridConfigs.Done` returns the per-family leaderboard + the cross-family ship pick.
- A parsimony rule with a swept-but-undeclared knob **hard-errors** with a next-action message;
  `const` (n_estimators) passes.
- `#StepManifest` + sidecars for `metis/train` + `titanic/features`, drift-guarded; each rule
  unit-tested.

## Plan

- [ ] (spec at claim) select rule as a pluggable reducer over the sweeper's per-config `(mean, SE)`; 1-SE + mean−std; test that a simpler within-1-SE config is chosen.

## Log

### 2026-07-07
- Filed as metis-v2 M2. The **selection** knob (separate from estimation/metis#23). Design in the pensive.
### 2026-07-07 (design converged)
- Home clarified: the select rule is INTERNAL to the black-box sweeper and consumes `(mean, SE)` from
  M1a's read-time reduction (not a "ledger objective"). Prior-art survey: 1-SE is uncontested across all
  six frameworks — our sharpest differentiator; parsimony ordering falls out of the tagged `$any` tree.
### 2026-07-08 (spec — 3 open knobs resolved)
- **select = tagged union** mirroring `driver` (params bound per branch; CUE closed disjunction + Go
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
