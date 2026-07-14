---
id: 000037
status: open
deps: [metis#36]
github_issue:
created: 2026-07-14
updated: 2026-07-14
estimate_hours:
---

# feature-constructor algebra: declared scope signatures + aggregate classes + derived placement

## Problem

After metis#36, leakage safety is structural but placement/cost is still coarse: `features` is one
monolithic step, so any fold-varying constructor forces recomputing every constructor, and the
hoist-vs-per-fold decision is made per step, not per feature. The general model (see research
notes) says placement is derivable: every feature constructor decomposes as fit (θ = reduce over
declared scopes: R = feature-rows, S = label-rows, per-row S(k) for LOO/cross-fit) ∘ apply
(per-row map), and the fit's aggregate class (map < commutative monoid < abelian-group/
subtractable < holistic) decides whether per-fold θ is shared, merged from per-fold partials,
subtracted, or re-reduced. Research status (two deep-research passes + adversarial verify,
2026-07-14): fit∘apply-as-pushable-aggregates and the class ladder are established prior art
(factorized in-DB ML; Gray data cube; DBSP); **no existing system types the label channel or
derives CV placement from declared signatures** — this composition is the novel part.

**Research notes (read first):**
`workshop/pensive/2026-07-14-01-pensive-feature-engineering-algebra-under-cv.md` — the
model, the verified design-space map (SystemDS = nearest system; SeLINQ = column-IFC existence
proof; fit-as-declassification framing; Amsterdamer/Deutch/Tannen semimodules as the
aggregation-taint bridge), and the open questions.

## Spec

Make constructors the unit, signatures the contract, placement the derivation:

- **Constructor-level nodes**: features become individual DAG nodes (Hamilton-style column
  granularity), not one `features` step. A fold recomputes only the constructors whose scope
  actually changed (e.g. a holistic per-fold median re-runs alone; `title` never does).
- **Declared scope signature per constructor**: (R-policy, S-policy, aggregate class). Cheney et
  al. (MSCS 2011) proves exact dependency inference undecidable — declaration + runtime domain
  restriction is the sound architecture, not a fallback. A false S declaration is *ineffective*
  (the runner never hands labels outside the domain); a false R declaration under prospective
  semantics is trusted — document that boundary.
- **Derived placement table**: R=∅,S=∅ → hoist always · R≠∅,S=∅ → hoist iff transductive ·
  S≠∅ → per-fold with y|A. Cost lowering by class: fold axis = partition + monoid merge (the
  SystemDS shape); per-row S(k) = subtraction where abelian (counts/sums target encodings — the
  sklearn TargetEncoder aggregate shape), re-reduce where holistic (priced honestly; skrub ships
  holistic median/mode in its default vocabulary, so this rung is not exotic).
- **Scope check as a type rule**: the metis#36 one-road rule generalizes — a constructor's label
  access is exactly its declared S under the runner's restriction; fit is the declassification
  point (SeLINQ's explicitly-deferred aggregate-release policy, instantiated).
- Candidate research outputs beyond the code: the CV-folds-as-partition-merge/deltas framing
  (verify novelty with a targeted differential-dataflow/Feldera + ML-systems search first) and
  the transductive estimand question (metis#36's ticket_size experiment feeds this).

## Done when

- (Gate at design time — this is research-track; done-when will be re-scoped by the plan.)
- A constructor registry with declared signatures replaces the monolithic features step for the
  kbench titanic pipeline; placement (hoist / per-fold / merged-partials) is derived, not
  hand-assigned; per-constructor recompute measurably beats stage-level on a fold sweep.
- A signature-violation test: a constructor declared S=∅ that attempts label access fails
  structurally; a mis-declared R is documented as the trust boundary.
- The novelty check ran (folds-as-deltas targeted search) and the claim is either substantiated
  or corrected in the pensive.

## Plan

- [ ] Do not start before metis#36 lands and metis has a second competition or the operator
  green-lights the research track (this issue is optional for Titanic-scale correctness — #36
  already gives structural safety; this is performance + generality + publishable structure).

## Log

### 2026-07-14
- Filed as stage C of the three-stage plan agreed with operator (A = metis#35, B = metis#36,
  C = this). Deliberately parked behind #36: correctness does not need it; build when scale or
  the research/book track justifies it. Full model + verified citations in the in-repo pensive.
