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
