---
id: 000037
status: open
created: 2026-07-14
updated: 2026-07-14
estimate_hours:
github_issue:
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
