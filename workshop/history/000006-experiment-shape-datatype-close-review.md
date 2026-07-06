# Boundary Review — metis#6 (whole-issue close)

| field | value |
|-------|-------|
| issue | 6 — experiment-shape datatype: lift the experiment config schema into a config-space (Space[T]) |
| repo | metis |
| issue file | workshop/issues/000006-experiment-shape-datatype.md |
| boundary | whole-issue close |
| milestone | — |
| window | 492bf8ae4972792f9de51b869e824c616fca3574..HEAD |
| command | sdlc close --issue 6 |
| reviewer | claude |
| timestamp | 2026-07-05T17:15:06-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. I verified the build/vet/tests are green, confirmed CUE closedness by direct `cue vet` (not the implementor's word), verified the fixture expands to its documented 21 points via the CLI, and discovered the merge-gate `vocabulary` binary has no `experiment-shape` noun. Here is the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** The heart of metis#6 — `pkg/shape.Expand` and the config-space algebra — is correct, well-tested, exemplary ARCH-PURE work, and the M2 CUE single-source refactor genuinely delivers what it claims: I confirmed by direct `cue vet` that `#Experiment` rejects a stray `sweep` and `#ExperimentShape` rejects unknown fields / requires `sweep`, so the `_pipeline` extraction single-sources the field set *and* keeps both definitions closed. Build/vet/full suite are green (shape 90.5%, experiment 85.7% coverage), plain-experiment behavior is byte-unchanged (`resolveExperiment` returns `experiment.Parse` verbatim for non-shapes), and the M1 aliasing footgun is genuinely fixed (deep-clone at the point boundary). Nothing blocks SHIP. What keeps it from a clean SHIP: (1) the **merge-gate enforcement seam was not extended to `experiment-shape`** — the ariadne `vocabulary` binary has no such noun, so committed shape files get no structural merge-time validation, breaking the "enforced, not just documented" pattern the experiment datatype established; and (2) the single-source refactor's key property — closedness — has **no automated negative test**. Both are cheap and non-blocking.

### 1. Strengths
- **ARCH-PURE, textbook.** `pkg/shape/shape.go:57` `Expand` and the whole recursion import only `fmt`/`math`/`sort`/`strings`/`experiment`; `ParseShape`/`ValidateShape` are pure `string→value`; `resolveExperiment` (`cmd/metis/run.go:65`) is a pure `raw→Experiment` and the only IO (`os.ReadFile`) stays up in `runExperiment`. Tests run with zero IO.
- **CUE single-source is real and closed (verified, not taken on faith).** `_pipeline` (`experiment.cue:37`) is embedded into both `#Experiment` and `#ExperimentShape`; I ran `cue vet` against hand-built docs and confirmed `#Experiment` rejects a stray `sweep` ("field not allowed"), `#ExperimentShape` rejects an unknown field, and a shape missing `sweep` fails. This factoring is actually *better* than the plan's proposed `#Experiment: #ExperimentShape & {…}` (which would have inherited `#ExperimentShape`'s required `sweep`). ARCH-DRY done right.
- **The keystone test pins real logic.** `TestExpand_TitanicSweep36Points` asserts `4 × [3+6] = 36` *and* that every `train.model` is a single-key bundle — the exact `$oneof`-ADDs-not-multiplies invariant; a flat-product regression fails it loudly.
- **Aliasing genuinely fixed.** `Expand` deep-clones each point's `With` (`shape.go:79` + `deepCloneMap`), and `TestExpand_PointsDoNotAliasInnerMaps` proves a `#7`-style in-place overlay on one point doesn't bleed into a sibling.
- **Fails loud on malformed shapes** (empty `$any`/`$oneof`, mixed `$`+plain keys, unknown `$`-key, non-numeric bounds, non-positive `$log-range`) rather than silently mis-expanding — the right default for an authoring surface, and covered by tests.
- **Fixture is correct.** I drove `testdata/experiment/titanic-baseline-shape.md` through the CLI: it expands to exactly the documented **21 points** and the multi-point path errors *before* any writes.

### 2. Critical findings
None.

### 3. Important findings

- **Merge-gate enforcement seam not extended to `experiment-shape` (ARCH-PURPOSE shadow-sweep).** `scripts/merge-checks.d/experiment-validate.sh` greps frontmatter with `^type:[[:space:]]*experiment[[:space:]]*$` — anchored to *exactly* `experiment`, so `type: experiment-shape` files are never validated at the gate. And it can't be trivially fixed: `vocabulary validate-instance --type experiment-shape` fails with `no vocabulary noun "experiment-shape" (have: [experiment issue pensive verdict])` — the ariadne binary's noun registry doesn't know the type. So structural validation of committed shape files rests only on the runtime `ValidateShape` + the *single-fixture* Go drift guard (which bypasses the binary via direct `cue vet`). No current impact (there are no committed `type: experiment-shape` *instances* yet — the prototype is `type: type`, testdata is skipped), but it silently drops the "schema enforced at the gate, not just documented" property the experiment datatype established, right when #7 will start committing shape instances. *Fix:* record the deferral explicitly in the issue `## Log` (+ a #7 follow-up), noting the ariadne `vocabulary`-noun dependency; don't let the close imply enforcement parity.

- **No automated closedness / negative test for the single-source refactor.** `TestShapeConformsToCUE` and `TestParse_ConformsToCUE` are positive-only; `experiment-schema-selftest.sh` covers only `#Experiment` (valid-baseline vs. invalid-bad-*status*). Nothing asserts `#Experiment` rejects a stray `sweep` or `#ExperimentShape` rejects an unknown field — the exact property the `_pipeline`-embed is built to preserve. I verified it holds *today*, but a future CUE edit (an accidental `...`, a mis-embed) would regress it silently. *Fix (cheap):* add a negative fixture (a shape with a stray field, and/or a plain experiment with a stray `sweep`) that must fail `cue vet`, wired into `experiment-schema-selftest.sh` or a Go conformance test.

### 4. Minor findings
- `titanic-baseline-shape.md`'s documented **21-point expansion is never asserted by an automated test** — only `ParseShape` + `cue vet`. The plan says the fixture "validates *and* Expands." Algebra is otherwise well covered; add `Expand(fixture)==21`.
- `resolveExperiment` parses the frontmatter **twice** for a shape (`experiment.Parse` then `experiment.ParseShape`) — minor ARCH-DRY/efficiency nit, not hot path.
- Go `ValidateShape` doesn't require `objective.metric` when an objective is present, but CUE `#ExperimentShape` marks `metric: string` required — a small Go/CUE asymmetry (low impact; objective is forward-looking for #7/#8).
- `atlas/experiment.md` got no `experiment-shape` section (only `atlas/index.md` was updated). The index bullet is adequate; a cross-link from `experiment.md` would aid discoverability.
- `$any` alternatives are taken **verbatim** (nested `$`-descriptors inside a `$any` element are *not* expanded), unlike `$oneof` which recurses. By design, but undocumented in the package doc — one line would spare #7 authors a surprise.

### 5. Test coverage notes
Strong where it counts: the 36-point keystone, `$oneof` bundling/ADD, `$linear-range` **and** `$log-range` (incl. the non-positive-bounds guard — the M1 gap is now closed), `range_steps` default, empty/malformed-descriptor errors, and sibling non-aliasing. e2e covers singleton-runs-like-v0 and multi-point-refused (with injected fake git/clock — ARCH-PURE). Gaps concentrate in **negative closedness** (Important #2) and the **fixture-expand assertion** (Minor).

### 6. Architectural notes for upcoming work
- **ARCH-DRY: pass.** `_pipeline` single-sources the CUE field set (both defs verified closed); Go `Shape` embeds `Experiment` and reuses `Validate`/`frontmatter.Split`. Only nit: the `resolveExperiment` double-parse.
- **ARCH-PURE: pass.** Pure core + thin IO seam is textbook; #7's sweep loop should keep driving `Expand` as a pure library and confine execution to the `cmd/metis` boundary.
- **ARCH-PURPOSE: pass with one shadow-sweep gap.** The single-source is *enforced* (closedness verified, drift guards), so consumers derive rather than restate — except the merge-gate consumer (Important #1), the one seam not yet mirrored for shapes.
- **Freeze the free-param path grammar.** The dotted-path form (`train.model`, `train.model.rf.n_estimators`) becomes #8's ledger key + feeds #3's point-address; pin it before #8 re-derives a parser. Note the label-inclusive path (`model.rf.n_estimators`, not flat `n_estimators`) — #8 must expect that shape.
- **Before #7 overlays points:** the deep-clone contract (`Point.With` is independent per point) is what makes in-place overlay safe — keep the `TestExpand_PointsDoNotAliasInnerMaps` guard.

### 7. Plan revision recommendations
- Append a `## Revisions` entry to `workshop/plans/000006-experiment-shape-plan.md` (or an issue `## Log` line) recording that **merge-gate structural validation of `experiment-shape` files is deferred** — it depends on the ariadne `vocabulary` binary gaining an `experiment-shape` noun — so the plan/close doesn't imply enforcement parity with `experiment`. Assign it to #7 (when shape instances start being committed).
- Optionally note the **closedness-regression test** and the **fixture-`Expand`-assertion** as cheap hardening follow-ups so the "single-source, conformance-tested" Done-when claim is backed by a *negative* test, not just positive drift guards.

---

## Re-review — 2026-07-05T17:22:48-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 6 — experiment-shape datatype: lift the experiment config schema into a config-space (Space[T]) |
| repo | metis |
| issue file | workshop/issues/000006-experiment-shape-datatype.md |
| boundary | whole-issue close |
| milestone | — |
| window | 492bf8ae4972792f9de51b869e824c616fca3574..HEAD |
| command | sdlc close --issue 6 |
| reviewer | claude |
| timestamp | 2026-07-05T17:22:48-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. I verified the build/vet/tests are green, independently confirmed CUE closedness by direct `cue vet` (not on the implementor's word), confirmed the parse dispatch is centralized and the plain-experiment path is byte-faithful, and — the key discovery — **empirically confirmed a cross-step inner-map aliasing bug** that the deep-clone comment claims to prevent but only half-delivers, with a guard test that structurally cannot catch it. Here is the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** The heart of metis#6 — `pkg/shape.Expand` and the `$`-descriptor config-space algebra — is correct, well-tested, and exemplary ARCH-PURE work, and the M2 CUE single-source refactor genuinely delivers what it claims: I confirmed by direct `cue vet` that the `_pipeline` embed keeps `#Experiment` closed (rejects a stray `sweep`) *and* `#ExperimentShape` closed (rejects unknown fields, requires `sweep`). Build/vet/full suite are green, plain-experiment behavior is preserved (`resolveExperiment` returns the parsed `Experiment` verbatim for non-shapes; validation still deferred to `Runner.Run`), and the prior close-review's two Importants were both addressed in-diff (the closedness negative test `TestCUE_ClosednessPreservedBySingleSource` now exists; the 21-point fixture expansion is now asserted). What keeps it from a clean SHIP: a **latent cross-step aliasing bug** — `Expand`'s deep-clone only protects the *current* step, so sibling points spawned by a later step share the *same* inner `with` map for every earlier step, and the guard test that claims to cover this uses a single step so it can't. It ships no wrong behavior today (the singleton path reads read-only; multi-point is refused), but it will bite metis#7 the instant it overlays a resolved value onto a non-terminal step — exactly the scenario the deep-clone comment says is prevented. Cheap to fix; non-blocking at the gate.

### 1. Strengths
- **ARCH-PURE, textbook.** `pkg/shape/shape.go` imports only `fmt`/`math`/`sort`/`strings`/`experiment`; `ParseShape`/`ValidateShape` are pure `string→value`; `resolveExperiment` (`cmd/metis/run.go:65`) is a pure `raw→Experiment` and the only IO (`os.ReadFile`) stays up in `runExperiment`. Tests run with zero IO.
- **CUE single-source is real and closed — verified, not on faith.** `_pipeline` (`experiment.cue:37`) embeds into both defs; I ran `cue vet` against hand-built docs and confirmed `#Experiment` rejects a stray `sweep` ("field not allowed"), `#ExperimentShape` rejects `bogus`, and a shape missing `sweep` fails. ARCH-DRY done right, and better than the plan's proposed `#Experiment: #ExperimentShape & {…}` (which would have inherited the required `sweep`).
- **The keystone test pins real logic.** `TestExpand_TitanicSweep36Points` asserts `4 × [3+6] = 36` *and* that every `train.model` is a single-key bundle — the exact `$oneof`-ADDs-not-multiplies invariant; a flat-product regression fails it loudly.
- **Prior close-review Importants closed in-diff.** `TestCUE_ClosednessPreservedBySingleSource` + `TestShapeFixture_ExpandsTo21Points` are exactly the negative-closedness and fixture-expansion tests the earlier review asked for.
- **Fails loud on malformed shapes** (empty `$any`/`$oneof`, mixed `$`+plain keys, unknown `$`-key, non-numeric bounds, non-positive `$log-range`) — right default for an authoring surface, and covered.
- **Parse dispatch centralized.** The only production parse path (`run.go:125`) goes through `resolveExperiment`; `experiment.Parse` survives only as the pure primitive (tests) — no orphan second dispatch.

### 2. Critical findings
None. (No shipped code path currently produces incorrect output — the finding below is latent.)

### 3. Important findings

- **Cross-step inner-map aliasing — the deep-clone is only half-delivered, and its guard test can't catch the miss (`pkg/shape/shape.go:80-81`).** `cloneWith` (`:258`) shallow-copies the outer `map[stepID]→with` map; `deepCloneMap(w)` at `:81` only deep-clones the *current* step. So all sibling points spawned from one base by a later step's expansion **share the same inner `with` map object for every earlier step**. I confirmed empirically with a 2-step shape (`split` fixed → `train` `$any` over 2 models): `points[0].With["split"]["dataset"] = "MUTATED"` bled into `points[1]`. On the real titanic fixture, the 7 `train`-siblings under each `features` value all alias the same `adapt` and `split` maps. The code comment at `:77-79` explicitly claims this is prevented so "#7 overlaying a resolved path in place [won't] silently mutate every sibling" — but it's only true for the *terminal* step. Worse, the guard `TestExpand_PointsDoNotAliasInnerMaps` uses a **single step**, so it structurally cannot exercise cross-step sharing → false confidence. Latent today (singleton path at `run.go:98` only *reads* `p.With[s.ID]`; multi-point is refused), but it's a correctness landmine for metis#7's overlay. *Fix (one line):* make `cloneWith` deep — `out[k] = deepCloneMap(v)` — (or deep-clone the whole assembled `np.With`); then extend the guard test to a **≥2-step** shape that mutates a **non-terminal** step's `with`. Fix before #7 consumes `Point.With`.

- **Merge-gate enforcement not extended to `experiment-shape` (ARCH-PURPOSE shadow-sweep) — already logged, confirming the deferral is honest.** `scripts/merge-checks.d/experiment-validate.sh`'s frontmatter probe is anchored `^type:[[:space:]]*experiment[[:space:]]*$`, which I confirmed does **not** match `type: experiment-shape`; and `vocabulary validate-instance --type experiment-shape` has no such noun in the ariadne binary. So committed shape *instances* would get no gate-time structural validation. **No current impact** (there are zero committed `type: experiment-shape` instances; the prototype is `type: type`, testdata is skipped), and the issue `## Log` (line 237) already records this as a #7 follow-up with the ariadne vocabulary-noun dependency named. I'm satisfied the deferral is complete and honest — flagging only so the close verdict doesn't imply enforcement parity with `experiment`.

### 4. Minor findings
- `ValidateShape` (`pkg/experiment/shape.go:55`) doesn't require `objective.metric` when an `objective` block is present, but CUE `#ExperimentShape` marks `metric: string` required within `objective?` — a small Go/CUE asymmetry. Low impact (objective is forward-looking for #7/#8); could tighten Go to require `metric` when an objective is non-empty.
- `atlas/experiment.md` got no `experiment-shape`/`pkg/shape` cross-link (only `atlas/index.md` was updated). The index bullet is adequate; a one-line cross-link from `experiment.md` would aid discoverability.
- `shapePointToExperiment` (`run.go:98`) sets `s.With = p.With[s.ID]`, which is a non-nil empty map for a step that declared no `with` (the original `Parse` left it nil). Immaterial for the singleton run (no source to be byte-faithful to); noting for completeness.

### 5. Test coverage notes
Strong where it counts: the 36-point keystone, `$oneof` bundling/ADD, `$linear-range` **and** `$log-range` (incl. the non-positive-bounds guard), `range_steps` default, empty/malformed-descriptor errors, the closedness negative test, the 21-point fixture, and both e2e paths (singleton-runs-like-v0, multi-point-refused with injected fake git/clock). The one real gap is the **aliasing guard**: it exercises a single step only and therefore gives false confidence on the exact invariant it names (Important #1). Add a ≥2-step non-terminal-mutation case alongside the fix.

### 6. Architectural notes for upcoming work
- **ARCH-DRY: pass.** `_pipeline` single-sources the CUE field set (both defs verified closed); Go `Shape` embeds `Experiment` and reuses `Validate`/`frontmatter.Split`; parse dispatch centralized. No duplication.
- **ARCH-PURE: pass.** Pure core + thin IO seam is textbook; #7's sweep loop should keep driving `Expand` as a pure library and confine execution to `cmd/metis`.
- **ARCH-PURPOSE: pass with the one logged shadow-sweep gap** (merge-gate consumer, deferred to #7 with a valid external dependency). Every Done-when item is delivered and the single-source is *enforced* (closedness verified), so the experiment consumer derives rather than restates.
- **Before #7 overlays points:** the `Point.With`-is-independent-per-point contract is what makes in-place overlay safe — but per Important #1 it is **not yet true across steps**. Land that fix (and its ≥2-step guard) as the first thing #7 depends on.
- **Freeze the free-param path grammar.** The dotted-path form (`train.model`, `train.model.rf.n_estimators`) becomes #8's ledger key + feeds #3's point-address; note the label-inclusive path (`model.rf.n_estimators`, not flat `n_estimators`) — #8 must expect that shape.

### 7. Plan revision recommendations
- No plan-vs-code contradiction requiring a `## Revisions` entry: the Core-concepts entities all exist at their stated paths, PURE/INTEGRATION classification holds, and the two milestone claims are delivered.
- Optionally add a one-line `## Log` note that the **`Point.With` cross-step independence** invariant (the deep-clone contract) was found incompletely delivered at close and is fixed as prerequisite hardening for #7 — so the "no sibling aliasing" claim in the M2 Log/atlas is backed by a test that actually covers the multi-step case.

---

## Re-review — 2026-07-05T17:30:34-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 6 — experiment-shape datatype: lift the experiment config schema into a config-space (Space[T]) |
| repo | metis |
| issue file | workshop/issues/000006-experiment-shape-datatype.md |
| boundary | whole-issue close |
| milestone | — |
| window | 492bf8ae4972792f9de51b869e824c616fca3574..HEAD |
| command | sdlc close --issue 6 |
| reviewer | claude |
| timestamp | 2026-07-05T17:30:34-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
Working tree is clean. I have everything I need. All verification is complete:

- **Build / vet / full suite green**; coverage `pkg/shape` 90.5%, `pkg/experiment` 85.7%.
- **CUE closedness independently verified** via direct `cue vet` (not the implementor's word): `#Experiment` rejects a stray `sweep`, `#ExperimentShape` rejects unknown fields and requires `sweep`.
- **Deep-clone fix empirically proven**: reverting `cloneWith` to a shallow copy makes the cross-step guard test FAIL with the exact alias message — so the fix is real and the guard genuinely covers the multi-step case (the M1/close-review's false-confidence trap is closed).
- **Plain-experiment path byte-faithful**: `Parse` and `ParseShape` are structurally identical; `resolveExperiment` returns `sh.Experiment` verbatim for non-shapes.
- **Merge-gate deferral honestly logged** (issue `## Log` lines 208, 239) with its ariadne-vocabulary-noun dependency named.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** metis#6 is complete and clean at HEAD. The heart of the issue — `pkg/shape.Expand` and the `$`-descriptor config-space algebra — is correct, exemplary ARCH-PURE work, and the M2 CUE single-source refactor genuinely delivers: I confirmed by independent `cue vet` that the shared `_pipeline` embed keeps *both* `#Experiment` and `#ExperimentShape` closed. Critically, the two prior close-review FIX-THEN-SHIP passes have both been fully addressed in-diff: the half-delivered deep-clone is now truly deep (and I *proved* the guard test catches a shallow regression, per the lesson the repo itself recorded), the closedness negative test exists, and the 21-point fixture expansion is asserted. Every Done-when item is delivered, the single-source is *enforced* (not just documented), and no shipped code path produces incorrect output. The only remaining shadow-sweep gap — the merge-gate not covering `experiment-shape` — is out of this issue's Done-when scope, has zero current impact (no committed shape instances), depends on an ariadne binary change, and is honestly logged as a #7 follow-up. Nothing blocks SHIP.

### 1. Strengths
- **ARCH-PURE, textbook.** `pkg/shape/shape.go` imports only `fmt`/`math`/`sort`/`strings`/`experiment`; `ParseShape`/`ValidateShape` are pure `string→value`; `resolveExperiment` (`cmd/metis/run.go:60`) is a pure `raw→Experiment` with the only IO (`os.ReadFile`) kept up in `runExperiment`. All tests run with zero IO.
- **CUE single-source is real and closed — verified, not on faith.** `_pipeline` (`experiment.cue:37`) embeds into both defs; independent `cue vet` confirms `#Experiment` rejects a stray `sweep` ("field not allowed"), `#ExperimentShape` rejects `bogus`, and a shape missing `sweep` fails. This factoring is better than the plan's proposed `#Experiment: #ExperimentShape & {…}` (which would have inherited the required `sweep`). ARCH-DRY done right.
- **The keystone test pins real logic.** `TestExpand_TitanicSweep36Points` asserts `4 × [3+6] = 36` *and* that every `train.model` is a single-key bundle — the exact `$oneof`-ADDs-not-multiplies invariant; a flat-product regression fails it loudly.
- **The deep-clone fix is genuine and regression-proofed.** `Expand` deep-clones each point's entire `With` (earlier steps via `cloneWith`→`deepCloneMap`, current step via `deepCloneMap(w)`); I empirically reverted to a shallow copy and the ≥2-step non-terminal-mutation guard `TestExpand_PointsDoNotAliasInnerMaps` failed exactly as designed. The `#7`-overlay-safety contract now holds across *all* steps.
- **Prior close-review Importants closed in-diff.** `TestCUE_ClosednessPreservedBySingleSource` + `TestShapeFixture_ExpandsTo21Points` are precisely the negative-closedness and fixture-expansion tests asked for.
- **Fails loud on malformed shapes** (empty `$any`/`$oneof`, mixed `$`+plain keys, unknown `$`-key, non-numeric bounds, non-positive `$log-range`) rather than silently mis-expanding — right default for an authoring surface, and covered.
- **Parse dispatch centralized.** The only production parse path (`run.go:125`) goes through `resolveExperiment`; `experiment.Parse` survives only as the pure primitive — no orphan second dispatch, no double-parse.

### 2. Critical findings
None.

### 3. Important findings
None new. For completeness, the one open architectural item — the **merge-gate not extended to `experiment-shape`** (`scripts/merge-checks.d/experiment-validate.sh` anchors `^type:[[:space:]]*experiment[[:space:]]*$`, which I confirmed does not match `experiment-shape`; `vocabulary validate-instance --type experiment-shape` has no such noun) — is a **confirmed-honest deferral**, not a blocker: zero committed shape instances exist, it's outside this issue's Done-when, it depends on an ariadne `vocabulary`-noun registration, and it is logged as a #7 follow-up. Flagging only so the close verdict isn't read as claiming merge-gate enforcement parity with `experiment`.

### 4. Minor findings
- **Go/CUE objective asymmetry** (`pkg/experiment/shape.go:53`): `ValidateShape` validates `direction` but not `metric`, whereas CUE `#ExperimentShape` marks `metric: string` required *within* `objective?`. A shape with `objective: {direction: maximize}` (no metric) passes `ValidateShape` but fails `cue vet`. Low impact (objective is forward-looking for #7/#8); tighten Go to require `metric` when an objective block is present.
- **Ranges always materialize to `float64`** (`shape.go:212`), so `$linear-range: [100, 500, 5]` yields `100.0…` while `$any: [100, 300]` stays `int`. Harmless for continuous domains (C, lr — the intended use), but a #7/#8 forward note: a range used for a discrete count would serialize as `300.0`, hashing differently from an int `300` in the #3 record / #8 ledger key.
- **`atlas/experiment.md` got no `pkg/shape`/`experiment-shape` cross-link** — only `atlas/index.md` was updated. The index bullet is adequate; a one-line cross-link would aid discoverability.
- **`shapePointToExperiment` (`run.go:98`)** sets `s.With = p.With[s.ID]`, a non-nil empty map for a step that declared no `with` (original `Parse` left it nil). Immaterial for the singleton run (no source to be byte-faithful to).

### 5. Test coverage notes
Strong where it counts: the 36-point keystone, `$oneof` bundling/ADD, `$linear-range` **and** `$log-range` (incl. the non-positive-bounds guard), `range_steps` default, empty/malformed-descriptor errors, the closedness negative test, the 21-point fixture, the cross-step aliasing guard (now empirically regression-proofed), and both e2e paths (singleton-runs-like-v0, multi-point-refused, with injected fake git/clock). No remaining coverage gap that would ship the class of bug this diff could produce.

### 6. Architectural notes for upcoming work
- **ARCH-DRY: PASS.** `_pipeline` single-sources the CUE field set (both defs verified closed); Go `Shape` embeds `Experiment` and reuses `Validate`/`frontmatter.Split`; parse dispatch centralized. No duplication.
- **ARCH-PURE: PASS.** Pure core + thin IO seam is textbook; #7's sweep loop should keep driving `Expand` as a pure library and confine execution to `cmd/metis`.
- **ARCH-PURPOSE: PASS.** Every Done-when item is delivered and the single-source is *enforced* (closedness independently verified, Go drift guard), so the experiment CUE consumer derives rather than restates. The one non-derived consumer (merge-gate) is a separable extension whose motivating instances don't exist until #7 — a legitimate follow-up, not the deferred point of the issue.
- **`Point.With` cross-step independence now holds** — this is the contract that makes #7's in-place overlay safe. Keep the ≥2-step `TestExpand_PointsDoNotAliasInnerMaps` guard as a load-bearing regression test when #7 starts overlaying.
- **Freeze the free-param path grammar.** The dotted, label-inclusive form (`train.model`, `train.model.rf.n_estimators` — *not* flat `n_estimators`) becomes #8's ledger key + feeds #3's point-address; pin it before #8 re-derives a parser.

### 7. Plan revision recommendations
None required. The Core-concepts entities all exist at their stated paths, PURE/INTEGRATION classification holds, both milestone claims are delivered, and the merge-gate deferral is already recorded in the issue `## Log` (lines 208, 239) — the plan/close no longer overclaims enforcement parity. No `## Revisions` entry is needed.
