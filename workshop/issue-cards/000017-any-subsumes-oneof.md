---
id: '000017'
status: done
started: 2026-07-07T00:46:24-07:00
created: 2026-07-07
updated: 2026-07-07
estimate_hours: 0.93
actual_hours: 0.71
---

# unify $oneof into $any — list=untagged / map=tagged sum, both recursive; delete $oneof

## Problem

The shape algebra has **two** choice primitives — `$any:[…]` (a flat set, verbatim) and
`$oneof:{L:sub,…}` (a labeled sum, recursive+bundled). They are the **untagged** and **tagged**
forms of the *same* concept ("pick one; across a sweep, try all"). The distinction we baked in —
"$any is verbatim, $oneof recurses" — is an implementation asymmetry, **not** fundamental: recursion
is orthogonal to tagged-vs-list. The real difference is purely the **argument shape** (a list vs a
map), which the syntax *already* signals. So `$oneof` is a redundant keyword.

Prior art (why this axis is real, not invented): the two most-used tools use the flat **list** form —
sklearn `param_grid` as a **list of dicts** (sum of grids), hyperopt `hp.choice` over a **list of
dicts** with a `type` field (nested exprs → conditional params). The tools that optimize for
readability/structure use the **tagged** form — Hydra **config groups** (`model=rf` selects a file;
the group name IS the tag) and Ax **HierarchicalSearchSpace** (conditionality first-class). The
tagged form reads better AND carries conditional structure an **adaptive sampler** can exploit
(hierarchical/conditional BO) — which matters because metis#7's `Sampler` seam exists precisely so
adaptive samplers slot in. So we keep BOTH *forms* — but under **one keyword**, dispatched on shape.
