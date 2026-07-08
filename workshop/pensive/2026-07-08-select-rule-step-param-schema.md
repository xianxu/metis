---
type: pensive
status: active
created: 2026-07-08
topic: metis#19 — configurable select rule + per-step-type complexity schema
---

# Select rule (metis#19) + the step-param complexity schema it needs

Converged design from the 2026-07-08 brainstorm. Empirically motivated by the real Titanic
acceptance run: `argmax-mean` selected the md=8 overfitter (cv 0.844 → **public 0.770**); the
md=4 counterfactual from the SAME cached ledger → **public 0.782**. The honest ledger *contained*
the better config; greedy selection walked past it. #19 fixes the selection lever.

## Scope (one #19 deliverable)

The select rule **and** the per-step-type complexity schema it consumes. The schema is the #4
("collocated step manifests") facet — #19 populates `knobs.complexity` + what it needs; #4's full
catalog / learn-notes for all steps stays #4's tail.

## The complexity substrate

- **Central CUE `#StepManifest`** in `construct/vocabulary/step-manifest.cue` (sibling to
  `#ExperimentShape`): `knobs: [path]: {type, domain, complexity}` + #4's `summary`/`inputs`/
  `outputs`/`learn-notes`.
- **Per-step `.md` sidecar** next to each executable (`steps/metis/train.md`,
  `kbench/steps/titanic/features.md`), frontmatter conforms to `#StepManifest`, drift-guarded by a
  merge-check (like shapes conform to `#ExperimentShape`). **Steps stay files** (not promoted to dirs).
- **Complexity = a per-knob value-function**: `{form: const|linear|log, basis: value|count|inverse}`
  — depth `linear·value`, features `linear·count`, C `linear·inverse` (more reg = simpler),
  `n_estimators` **`const`** (more trees ≠ more overfit; complexity means *overfitting-capacity*, not
  resource size). Code escape-hatch for full-math deferred (YAGNI).
- **LOUD** when a swept knob has no declared complexity — never silently ignored (silent → parsimony
  quietly wrong).
- **Explicit, per-step-type** (rejected: `$any`-list-position — too much authoring convention;
  fixed heuristic — not general). Engine stays generic (reads declarations); each step owns its knobs'
  semantics (layer-appropriate: metis owns metis/train; kbench owns titanic/features).

## The select rule (consumer)

- Family: `argmax-mean | one-std-err | mean−λ·std | pct-loss`. **Default stays `argmax-mean`**
  (no silent behavior change); `titanic-sweep.md` opts into `pct-loss` to demonstrate + recover md=4.
- **Band rule** sets contention; **parsimony** picks within it.
  - `one-std-err` = SE-width band — **too tight** here: SE 0.005, but the real cv→public gap was
    0.074 (**15×** the SE), so md=4 (0.834) sits *below* the 1-SE floor (0.839). Vanilla 1-SE would
    NOT recover md=4 — it inherits the over-confident inner-CV SE.
  - `pct-loss` = %-width band (decoupled from SE) — ~2% floor 0.827 includes md=4; parsimony picks it.
    **This is the rule that actually recovers today's case.**
- **Group-by labeled-sum branch (model family)** — the `$any`-map discriminant IS the family
  (general, not a "model" special-case). Within each family → parsimony over the value-functions
  (**Pareto** per-axis, scale-free; sum-of-normalized-ranks only to break Pareto-incomparables) →
  that family's *robust* winner. **Across families → argmax-mean over the (already-robust) family
  winners** for the single ship (no fake cross-family complexity currency — that's the hard,
  unprincipled thing; the winner's-curse is largely spent inside the within-family step).
- Cross-family *complexity* is deliberately NOT computed — cross-family selection is an **estimation**
  problem → nested-CV (#23). Structural parsimony is the within-family tool; #23 the cross-family one.
  They compose (the pensive's selection-vs-estimation split).

## Sampler evolution (pkg/sampler)

`GridConfigs.Done` returns a **per-family winner map + the shipped cross-family pick** (evolves M1a's
single `Winner`). All per-family winners are promotable (`promote` gains a family selector);
`driver:single` ships the cross-family-best. The per-family set is the honest leaderboard + exactly
the inputs #22 (ensembling) blends and #23 (nested-CV) estimates one-per-family — so group-by-family
is the seam the rest of the project already wanted, not a workaround.

## Done-when

Selectable rule (documented default); a config that loses on `argmax-mean` but wins on a robust rule
is **demonstrably selected** (the md=4-over-md=8 case, offline from the existing cached ledger — no
re-run); group-by-family leaderboard; loud warning on an unmodeled knob.

## Open knobs (settle in the spec)

- Band-tolerance config surface: `objective: {select: pct-loss, tolerance: 0.02}` / `λ` for mean−λ·std.
- Which step-types get manifests first (the swept ones: `titanic/features`, `metis/train`; then
  cv-split/adapt/get-data/predict/submission as needed).
- Exact `#StepManifest` CUE shape for the labeled-sum branch (each branch declares its own knobs +
  optional `base`; the `base` cross-family prior is opt-in and explicitly NOT used for structural
  cross-family comparison).

ARCH: pure `pkg/sampler` + CUE + manifests, no new IO in the hot path (ARCH-PURE); the schema is the
single source, Go + Python + the merge-check derive from it (ARCH-DRY / single-source).

## Revisions

### 2026-07-08 — spec pass (metis#19 `## Spec` is now the record of truth)
- **`C` complexity basis corrected: `linear·inverse` → `linear·value`.** The shorthand above was
  wrong taken literally — sklearn `C` is *inverse* regularization strength, so small C = strong reg =
  *simpler*; complexity must INCREASE with C (`basis: value`). `inverse` is kept in the vocabulary for
  a true penalty-weight knob (`alpha`/`lambda`), not C.
- **Two-level selection made explicit:** the `select` rule is the WITHIN-family policy; ACROSS families
  is *always* `argmax-mean` over the robust per-family winners. This makes `argmax-mean` a true special
  case (within=argmax, across=argmax ⇒ global argmax-mean, M1a unchanged), not a fourth branch of ad-hoc
  logic.
- **Rules consume only the monotone direction today:** Pareto + rank-tie-break are invariant to any
  monotone transform, so `form: linear|log`/scale is declared-for-forward (a complexity-penalty rule),
  NOT consumed by the current rules. Documented as declared-not-yet-consumed (the direction IS used).
- **3 open knobs settled** (see the issue Log): tagged-union `select` mirroring `driver`; first
  manifests = swept step-types only; select required-explicit with `pct-loss` canonical.
