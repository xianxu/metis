# Boundary Review — metis#6 (milestone M1)

| field | value |
|-------|-------|
| issue | 6 — experiment-shape datatype: lift the experiment config schema into a config-space (Space[T]) |
| repo | metis |
| issue file | workshop/issues/000006-experiment-shape-datatype.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | cc04241c08f9e55782e2a41cbd72716c9fc6caee^..HEAD |
| command | sdlc milestone-close --issue 6 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-05T16:57:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. Let me record the key process finding as a memory note and then deliver the review.

The verdict follows.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** The true M1 deliverable — `pkg/shape` (`Expand` + the config-space algebra) — is correct, exemplary ARCH-PURE work: a pure recursion with no IO, and the keystone 36-point titanic test genuinely proves the one semantic the design worried about (`$oneof` *adds*, doesn't multiply). Build, `go vet`, and the suite are green, and `pkg/shape` is an unwired pure leaf (nothing imports it yet), so M1 ships no risk to existing code. What keeps it from a clean SHIP: the plan's M1 test list explicitly names **`$log-range` grid materialization**, but only `$linear-range` is tested — the logspace formula and its positive-bounds guard ship uncovered. Plus a latent aliasing footgun in the returned `Point.With`. Both are cheap to fix and non-blocking.

**⚠️ Process finding (read first).** The review window `cc04241^..HEAD` spans **38 commits** and re-includes already-merged, already-reviewed work: #9/pkg/cas (PR #2), #3/pkg/record (PR #3), #2/pkg/cache+trace (PR #4). The genuine M1 boundary is a single commit (`134e5ef`) touching only `pkg/shape/*` + the issue/plan. I reviewed the true M1 delta and did **not** re-litigate the merged cache/CAS/record code (it passed its own gates). The harness's `BASE_SHA` for this boundary looks mis-derived — it should be the branch point / the PR #4 merge (`b85b30c`), not `cc04241^`. Worth fixing so future boundary reviews here don't re-scan merged history (and so no one mistakes this verdict as having vetted the cache work).

### 1. Strengths
- **ARCH-PURE, textbook.** `shape.go:57` `Expand` and the whole recursion are pure (imports only `fmt`/`math`/`sort`/`strings`/`experiment`); tests run with zero IO. The IO seam (YAML parse, `metis run`) is correctly deferred to M2/#7. This is exactly the pure-core/thin-shell split the principle asks for.
- **The keystone test pins real logic, not a mock.** `TestExpand_TitanicSweep36Points` asserts `4 × [3+6] = 36` *and* that every `train.model` is a single-key bundle — this is the precise `$oneof`-ADDs-not-multiplies invariant from the Spec, and a flat-product regression would fail it loudly.
- **The Spec's core invariant is pinned.** `TestExpand_AllSingletonIsOnePoint` locks "experiment ⊂ experiment-shape" (all-singleton → exactly one byte-identical v0 point, no free params) — the Done-when item, tested directly.
- **Deterministic enumeration** (`dollarKeys`/`sortedKeys` sort; `$oneof` iterates sorted labels) — reproducibility is baked into the algebra, not incidental.
- **Fails loud on malformed shapes** (`shape.go:97-100`, `expandDescriptor` default) rather than silently mis-expanding — the right default for an authoring surface.
- **Clean scope discipline** matching the plan's scope line: grid materialization lives in `Expand`; iteration/execution is left to #7.

### 2. Critical findings
None.

### 3. Important findings
- **`$log-range` is entirely untested (`shape.go:198`, `:208`).** The plan's M1 test list explicitly names "`$linear-range`/`$log-range` grid materialization," but `TestExpand_RangeMaterializesToGrid` covers only `$linear-range`. Coverage confirms `materializeRange` at 74% with the logspace formula (`lo * math.Pow(hi/lo, t)`) and the `lo<=0||hi<=0` guard uncovered — a transposed logspace formula would ship green. *Fix:* add a `$log-range` test (e.g. `[1, 1000, 4]` → `1,10,100,1000`) + a non-positive-bounds error case. The formula itself reads correct; this is a coverage gap on a claimed deliverable, not a known bug.
- **`Point.With` shares inner maps across sibling points (`shape.go:71`+`:75`).** `w := r.value.(map[string]any)` is reused across base iterations and `cloneWith` is a shallow copy, so points that share a step's expansion share the *same* `map[string]any` object. Latent today (Expand's output is correct, and the runner marshals `with` read-only). But when #7 overlays a point onto steps and anything resolves a value in place — e.g. `point.With["adapt"]["raw"] = <resolved path>` — every sibling point sharing that map is silently mutated, giving wrong/non-deterministic sweeps. *Fix:* deep-clone inner maps in `Expand`, or document `Point.With` as read-only on the type. Cheap now; expensive to debug after #7.

### 4. Minor findings
- Empty `$any: []` / `$oneof: {}` silently collapses the **entire** point set to zero points (a field with no expansions short-circuits the product) — an authoring typo yields a no-op sweep with no error. Consider erroring, or catch it in M2's validator.
- Uncovered edge branches, all cheap table additions: `steps < 1` error (`:195`), `steps == 1` single-point case (`:202`), two `$`-keys in one map (`:97`, distinct from the tested mixed-with-plain case), and the `$any`-not-a-list / `$oneof`-not-a-map arg-type errors (`:132`, `:143`).
- `toInt`/`toFloat` int64/float64 branches are dead under Go-literal tests (33%/60%) — they'll first execute under YAML parsing in M2; add a YAML round-trip test there.

### 5. Test coverage notes
85.7% statements. High-value tests pin real logic (keystone, ragged free-param paths, all-singleton, product×set, `range_steps` default). The gap is concentrated in `$log-range` + range-edge error paths (see Important #1 and Minor). No test would currently catch a `$log-range` formula regression or the sibling-map aliasing hazard.

### 6. Architectural notes for upcoming work
- **ARCH-DRY, the M2 crux:** the CUE `#Experiment = #ExperimentShape & {type:"experiment"}` refinement (plan finding 1) is the DRY-critical piece — make `#Experiment` *derive* from `#ExperimentShape`, not a hand-maintained second schema, else ARCH-PURPOSE's "every consumer derives from the source" is violated at the schema layer.
- **Freeze the free-param path grammar now.** The dotted-path format (`"train.model"`, `"train.model.rf.n_estimators"`) becomes #8's ledger key and feeds #3's point-address. Pin its shape before #8 re-derives a parser for it.
- Resolve the `Point.With` aliasing (Important #2) before #7 consumes points.

### 7. Plan revision recommendations
- The issue `## Log` claims M1 delivered "range→grid + `range_steps` default" as tested, but `$log-range` specifically has no test. Preferred: add the `$log-range` test before milestone-close (cheapest, makes the claim true). If deferring, append a `## Revisions` entry to `workshop/plans/000006-experiment-shape-plan.md` narrowing the M1 test claim to `$linear-range` and moving `$log-range` coverage to M2 — so the plan stops claiming coverage the code doesn't have.
- No revision needed for the atlas deferral: `pkg/shape` is new surface with no atlas entry in this range, but the plan legitimately schedules `atlas` at M2 (the immediate next, issue-closing boundary) and the package is internal-only until M2 wires it. Confirm M2 actually lands the `atlas/index.md` + `experiment.md` entry so the package doesn't stay unmapped.
