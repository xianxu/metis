---
id: 000017
status: codecomplete
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours: 0.93
started: 2026-07-07T00:46:24-07:00
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

## Spec

**`$any` dispatches on its argument type** (list literal = untagged sum; map literal = tagged sum) —
syntax carries the type. Both are **recursive** and their counts **add**; free-param coordinates
**decompose**.

- **`$any: [alt, …]` (list) → untagged sum.** Each alternative is **recursively expanded** (nested
  `$`-descriptors inside an alternative are swept — a change from today's verbatim), and the resolved
  value is placed **bare** at the leaf. Free-param = the chosen value (decomposed if structured).
  - e.g. `features: {$any: [[], [title], [title, family]]}` → `features: [title, family]` (3 pts).
- **`$any: {label: sub, …}` (map) → tagged sum.** Each branch is **recursively expanded** and bundled
  as **`{label: resolved}`**. Free-param = `{path: label}` **+** the nested sub-coords. *(This is
  today's `$oneof` logic verbatim — output shape unchanged, so consumers like `metis/train` that read
  `model` as `{rf: {…}}` need NO change.)*
  - e.g. `model: {$any: {logreg: {C: {$any: [0.1,1,10]}}, rf: {n_estimators: {$any: [200,500]}, max_depth: {$any: [4,8]}}}}`
    → `logreg`(3) + `rf`(4) = 7 pts, each `model: {rf: {n_estimators: 500, max_depth: 4}}`.
- **`$oneof` is deleted** (its map-handling folds into the `$any` map branch — one shared helper).

**Backward-compat / migration:**
- The list form only *gains* recursion; every current `$any` element is a plain scalar/string-list
  (no nested `$`-descriptor), so recursion is a **no-op** on existing shapes. Safe (the `$`-keys are
  reserved, so a real feature list can't collide with a descriptor).
- The map form is **behavior-identical** to `$oneof` — migrating a shape is a keyword rename
  `$oneof:` → `$any:`. **No `metis/train` / consumer change.**
- **Cross-repo:** kbench's `titanic-sweep.md` + `titanic-sweep-smoke.md` use `$oneof` → migrate to the
  `$any` map form (a dependent **kbench** follow-up, filed). Sequence: land metis (both keywords
  briefly co-exist if we keep a short-lived `$oneof` alias, OR migrate kbench back-to-back and verify
  kbench's e2e against the metis branch before either merge) → migrate kbench → drop any alias.

## Done when

- `$any: {map}` produces the same `{label: sub}` points a `$oneof:{map}` did (a golden/round-trip test),
  and `$any: [list]` with a **nested descriptor inside an element** expands it (the new recursion).
- `$oneof` is gone from `pkg/shape` (grammar + code + tests migrated to `$any`).
- metis shape tests cover **both** `$any` forms (list-recursive + map-tagged) + counts-add + coord-decompose.
- kbench titanic shapes migrated to `$any` and the `sweep-smoke` e2e (both forms: `features` list +
  `model` map) is green — the both-forms regression anchor (metis#12 proved the tagged→train path).
- atlas (`experiment.md` shape section): `$any` = the one choice primitive; list=untagged/bare,
  map=tagged/bundled; both recursive. `$oneof` removed from the docs.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.15  impl=0.35
item: atlas-docs             design=0.05  impl=0.15
item: milestone-review       design=0.0   impl=0.2
design-buffer: 0.15
total: 0.93
```

The engine change is localized to `expandDescriptor`'s `$any` case (a type-switch + list recursion,
the map branch is `$oneof`'s code moved). Bulk is the `$oneof`→`$any` sweep across tests, testdata,
the datatype template, atlas ×2, and CUE comments (value-algebra is untyped — no schema enum change).
Durable plan: `workshop/plans/000017-any-subsumes-oneof-plan.md` (reviewed). Cross-repo: kbench#7 migrates
the titanic shapes (verify its sweep-smoke e2e against this branch before merge).

## Plan

Single-boundary (plain checkboxes, one `sdlc close`).

- [x] RED: `$any:{map}` expands to bundled `{label: sub}` points (golden-equal to the old `$oneof`); `$any:[list]` recurses into a nested-descriptor element.
- [x] GREEN: fold `$oneof`'s map logic into `$any`'s map branch; make the `$any` list branch call `expandValue` per element (recursion); delete the `$oneof` case + grammar.
- [x] Migrate metis's own `$oneof` test fixtures/cases → `$any` map form; full `pkg/shape` + cmd/metis green (+ shape.go doc comments, cue, ledger test, python data-plane).
- [x] atlas `experiment.md`/`index.md` + datatype template reconciled; kbench#7 filed + migrated + sweep-smoke verified.

## Log

### 2026-07-07
- 2026-07-07: closed — Re-close after SHIP review: strengthened TestExpandAnyList to assert coord values (0/0.5/1/9 materialize through the list branch); reconciled plan Task 3 checkboxes with the kbench sweep-smoke evidence. No production behavior change (test + docs only). go test ./pkg/shape green.; review verdict: SHIP
- 2026-07-07: closed — $any subsumes $oneof: expandDescriptor dispatches on arg shape (list=untagged bare / map=tagged bundled), both recursive; $oneof deleted. Map form golden-identical to $oneof (migrated 36-point ADD + ragged tests pass unchanged); list form gains recursion (TestExpandAnyList_RecursesIntoElements, no coord dup); TestOneofRemoved guards deletion. Swept every consumer: testdata shape (cmd/metis e2e 21pts identical), shape.go doc comments, datatype template, atlas x2, cue, ledger test, python data-plane. go test ./... + pytest(13) green; vet clean. CROSS-REPO Done-when: kbench sweep-smoke e2e PASSED against this branch (features $any list + model $any map). Both change-code judges INFO.; review verdict: SHIP
- Filed from a design conversation (operator): `$any`/`$oneof` are the untagged/tagged forms of one
  "pick one" primitive; the list-vs-map argument shape already signals which, so the second keyword
  is redundant. Operator's proposal: `$any` dispatches on list (bare) vs map (tagged), both recursive.
  Prior-art survey (sklearn/hyperopt list-of-bags; Hydra/Ax tagged) confirmed the axis is real and the
  tagged form is worth keeping (readability + adaptive-sampler legibility, cf. metis#7). Reframes the
  distinction from "recursive vs verbatim" (an impl asymmetry) to "untagged vs tagged sum" (the type).
