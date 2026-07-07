# Boundary Review — metis#17 (whole-issue close)

| field | value |
|-------|-------|
| issue | 17 — unify $oneof into $any — list=untagged / map=tagged sum, both recursive; delete $oneof |
| repo | metis |
| issue file | workshop/issues/000017-any-subsumes-oneof.md |
| boundary | whole-issue close |
| milestone | — |
| window | ec1295e147cc38187642fc44faf2a156b4e6ac4b..HEAD |
| command | sdlc close --issue 17 |
| reviewer | claude |
| timestamp | 2026-07-07T01:30:53-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. All Go tests pass (including the `cmd/metis` e2e that reads the migrated testdata), `go vet` is clean, no README exists (docs gate N/A), and every active `$oneof` config key across Go/CUE/testdata/Python is migrated — remaining mentions are the intentional `TestOneofRemoved` migration-signal test, "ex-$oneof" historical annotations, and archived `history/`.

```verdict
verdict: SHIP
confidence: high
```

This is a clean, well-scoped unification: `$oneof` folds into `$any`'s map branch (the same bundling code, moved not copied), `$any`'s list branch gains uniform recursion, and the `$oneof` case is deleted so a stale key errors as an unknown descriptor. The engine change in `pkg/shape/shape.go` is correct — I traced the list branch's free-param rule (use the element's own coords if it produced any, else `{path, value}`) across backward-compat scalars/lists, the new nested-descriptor recursion, and structured product elements; none duplicate or drop a coordinate. The map branch is byte-for-byte the old `$oneof` logic (`sortedKeys` determinism, `{label: r.value}` bundling), so consumers reading `model: {rf:{…}}` need no change. The consumer shadow-sweep is complete within the metis repo; the only remaining consumer is cross-repo (kbench), explicitly tracked as kbench#7 and out of this repo's scope. Nothing blocks SHIP.

**1. Strengths**
- `pkg/shape/shape.go:136-185` — the dispatch is the minimal correct change: one type-switch, the map branch preserved verbatim (zero consumer churn), the list branch's recursion a genuine no-op for today's scalar/list alternatives. Clean ARCH-DRY win — one choice primitive instead of two.
- `pkg/shape/shape_test.go:15-40` — `TestExpandAnyList_RecursesIntoElements` pins the *trickiest* property this change introduces (a nested `$linear-range` inside a `$any` list element expands, and the coord never duplicates at the element's own path). That's exactly the seam a naive `{path, value}` append would break.
- `pkg/shape/shape_test.go:53-60` — `TestOneofRemoved` locks in the migration signal (stale `$oneof` → clear error), so a future re-introduction can't slip through silently.
- Migration discipline: the committed `testdata/experiment/titanic-baseline-shape.md` was migrated *in the engine commit*, and the whole-module `go test ./...` (not a scoped `./pkg/shape/`) gates it — precisely the false-green trap the new `workshop/lessons.md:63` entry warns about. The lesson was written from this work and applied to it.
- The doc sweep is thorough and consistent: shape.go package-doc, `FreeParam` doc, CUE comments, datatype template, both atlas files, ledger test comment, and the full Python data-plane (including the `model.py:37` error *string*, not just comments) all reconciled to the one-primitive model.

**2. Critical findings** — none.

**3. Important findings** — none. (No README in the repo; atlas + datatype + CUE all updated in-range; the shipped bug classes — recursion, bad-arg, empty, removal, ragged free-params — are all covered.)

**4. Minor findings**
- `pkg/shape/shape_test.go:29-38` — the recursion test counts coords at `s.lr` but never asserts their *values* (should be `0.0/0.5/1.0/9.0`). The no-duplication property is the key one and is covered; tightening to assert the decomposed values would additionally pin that the range materialization flows through the list branch. Optional.
- No dedicated golden byte-equality test for old-`$oneof` vs new-`$any:{map}` output — impossible now that `$oneof` is deleted, and the migrated structural assertions (36-point ADD, single-key bundle, ragged paths at `shape_test.go:70,156`) plus the fact that the map branch is the same code make this a non-issue. Note only.
- `pkg/shape/shape.go:155` comment "(as verbatim $any did)" is an accurate historical reference (a leaf element still records `{path, value}` like the old verbatim path), not stale — flagging only so it isn't "corrected" away later.

**5. Test coverage notes**
Both `$any` forms are exercised: list-of-lists backward-compat (`TestExpand_TitanicSweep36Points` `features` leg), map/tagged ADD + bundling + ragged free-params (`:70`, `:156`), new list recursion (`:15`), bad-arg (`:43`), empty list *and* map (`:277`), and `$oneof` removal (`:53`). Sibling non-aliasing (`:295`) and range materialization remain green. This is exactly the surface that would catch a regression of the kind this diff could ship (the coord-duplication / verbatim-vs-recurse seam). PURE entity, tests run with no IO — ARCH-PURE holds.

**6. Architectural notes for upcoming work**
- **ARCH-DRY: pass** — the change *removes* duplication (two primitives → one); the tagged-sum logic lives once, moved into the `$any` map branch. The residual per-branch `len==0` guards and `out` loops are genuinely distinct logic (bare vs bundled, `path` vs `join(path,label)`), not extractable duplication.
- **ARCH-PURE: pass** — `pkg/shape` stays a pure deterministic recursion; no IO introduced; tests need no mocks.
- **ARCH-PURPOSE: pass (with one cross-repo caveat).** The shadow-sweep of consumers *within metis* is complete — every active `$oneof` key derives from the unified `$any`, none left as documentation-only. The single deferred consumer is **kbench** (a peer repo), which is a legitimately separable cross-repo migration (kbench#7), not the deferred point of this issue. **Pre-merge caveat:** deleting `$oneof` here will break kbench's `$oneof` shapes the moment metis merges, so confirm kbench's `sweep-smoke` e2e was actually run green against *this* branch before merging metis — see plan revision below.

**7. Plan revision recommendations**
- `workshop/plans/000017-any-subsumes-oneof-plan.md` — the plan's **Task 3 (cross-repo rollout + close) checkboxes are still `[ ]`**, while the issue's Plan item 4 claims "kbench#7 filed + migrated + sweep-smoke verified" as `[x]`. Reconcile the two: either tick Task 3 with the evidence (the actual kbench sweep-smoke run against this branch) or add a `## Revisions` note recording that kbench verification is pending and must gate the metis merge. This boundary review cannot confirm the kbench run from within the metis repo, so the claim rests unverified — surface it explicitly rather than carry it as a ticked-but-unproven Done-when.
