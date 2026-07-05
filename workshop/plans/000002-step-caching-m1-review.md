# Boundary Review — metis#2 (milestone M1)

| field | value |
|-------|-------|
| issue | 2 — Uniform DAG step caching: content-address step inputs, skip unchanged, recompute only what changed |
| repo | metis |
| issue file | workshop/issues/000002-step-caching.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | d0eae45991a603403ab0e64d8d7f19423ca9a85a^..HEAD |
| command | sdlc milestone-close --issue 2 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-05T15:17:08-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. Verified: build + vet + full test suite (incl. `-race`) green; the M1 commit is tightly scoped to `pkg/cache` + a byte-preserving `CanonicalHash` extraction; the DRY consolidation is complete; and no `Runner.Run` callers broke.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M1 "pure cache core" boundary delivers exactly its planned scope — `Kpre`, `Validate`, `OutputKey`, the `Entry` codec, and the `record.CanonicalHash` extraction — as a clean, fully-pure package with thorough unit tests that pin real hashing behavior (not mocks reasserting the impl). Build, vet, and the full suite (including `-race` on the new pure packages) are green; the `CanonicalHash` refactor is byte-preserving, so already-shipped #3 point-addresses are stable; and a shadow-sweep confirms all four canonical-hashing views (`PointAddress`, `OutputHash`, `Kpre`, `OutputKey`) now derive from the single `CanonicalHash` source with no leftover inliner. Nothing blocks SHIP — the findings are a docs-timing gap (`pkg/cache` has no atlas entry) and plan-text drift, both cheap and non-code.

### 1. Strengths
- **`Kpre` closes the exact false-HIT vectors the plan-judge flagged** (`pkg/cache/cache.go:31`): `seed` as an explicit arg (it lives on `RunRecord`, not the step), `uses` folded in (guards the wrong-step-type serve), and `Upstream` sorted for `needs`-order invariance. `TestKpre_FiveTermSensitivity` genuinely pins each of the five determinants as a distinct false-HIT case — this is real logic, not mock reassertion.
- **ARCH-PURE is textbook here.** `Validate(storedD, hash func(path) (Hash, error)) bool` (`cache.go:64`) injects the only IO (blob re-hashing) as a seam, so the whole package unit-tests with map-based fake hashers and zero filesystem. PURE claim verified.
- **`CanonicalHash` extraction is a clean ARCH-DRY win** (`pkg/record/address.go:21`) and byte-preserving — I diffed the refactor: the marshaled struct is identical, only error-wrap strings changed, so no downstream address drift.
- **Correct "safe direction" on validation**: a mismatch *or* a hasher failure is a MISS (`cache.go:65-69`) — MISS only recomputes, never serves stale, so error-swallowing here cannot produce an unsound HIT.

### 2. Critical findings
None.

### 3. Important findings
- **atlas update appears missing for `pkg/cache`** (`atlas/index.md`). The window adds a new architectural surface (the `pkg/cache` policy layer: `Kpre`/`Validate`/`OutputKey`/`Entry`), but the atlas only references #2 as a *future* role ("the trace/cache-key are #2") in the `pkg/cas`/`pkg/record` entries — there is no `pkg/cache` package entry. AGENTS.md §8 says update the atlas at *each* milestone close, don't defer. The plan deliberately scopes the atlas write to M3 ("atlas: `pkg/cache` + the validating-trace flow … M3"), which is defensible since the *flow* doesn't exist until the runner integrates it — but the *package/terminology* surface exists now. Fix sketch: add a one-line `pkg/cache` stub to `atlas/index.md` (name + "the validating-trace policy layer over CAS+record; pure core K_pre/Validate/OutputKey shipped M1, runner integration M3 [metis#2]"), or explicitly record in the M1 `## Log` that atlas is M3-owned so the deferral is a conscious decision, not a miss. README needs no change (no user-facing CLI surface in M1). Non-blocking.

### 4. Minor findings
- `Validate` returns `bool`, but the plan/issue specify `(hit bool, err error)` (`cache.go:64`; plan M1 bullet; issue line 213). The code's choice (fold hasher error into MISS) is *sounder* than the plan, but the plan text now lies — see plan-revision note below.
- `sortedHashes` (`cache.go:104`) and `sortedRefs` (`cache.go:110`) are near-identical sort helpers, and `sortedRefs`'s sort-by-`Path` duplicates `OutputHash`'s internal sort (`address.go:54`) over the structurally-identical `CodeRef`/`FileHash`. Not worth a generic today, but note it if a third (path, hash) sorter appears.
- No explicit test for `OutputKey(kpre, nil)` (empty D) — low risk since `OutputHash`'s empty-set case is covered in `pkg/record`, but a one-liner would close the loop.
- `Kpre` hashes `nil` vs empty `With` differently (`null` vs `{}`) — benign (only an extra recompute, never a false HIT), since `With` is stably nil-or-populated per experiment definition.

### 5. Test coverage notes
Coverage matches the plan's M1 test list item-for-item: determinism + 5-term sensitivity, upstream-order invariance, non-finite error, `Validate` clean/changed/vanished/empty, `OutputKey` composition + order invariance, `Entry` round-trip, plus the new `CanonicalHash` determinism/sensitivity/non-finite test. Tests are IO-free and exercise real hashing — the kind of bug this diff could ship (a false-HIT from a dropped K_pre term) is directly covered. The only untested-but-benign gaps are the nil-vs-empty `With` and empty-D `OutputKey` edges noted above.

### 6. Architectural notes for upcoming work
- **M3 must feed `Kpre` from a single, stable `With` source.** K_pre determinism relies on the same Go value types each run. Today `Step.With` comes from YAML (`int`/`float64`/`bool`/`string`); Go's `json.Marshal` collapses whole floats (`5.0`→`"5"`), so `int(5)`/`float64(5.0)` coincide — good. Keep K_pre computed from the freshly-parsed experiment (never reconstructed from a round-tripped `record.json`) so this invariant holds. Large-int config would break float/int parity, but that's out of v1 scope.
- **M3 `Upstream` population is the load-bearing next step** (plan already names it): `buildRecord` computes `outputHashes` for every step but doesn't thread them per-step into `StepRecord.Upstream`. Until that lands, `Kpre` is upstream-blind — the sensitivity test passes only because the test sets `Upstream` by hand. Sort at population time (as `Kpre` already re-sorts defensively).
- `Validate`'s error-as-MISS is the right default, but when M3 wires the real git-blob hasher, consider logging (not failing) a hasher IO error so a *persistent* permission/disk fault on a D file doesn't silently degrade every run to a cold MISS unnoticed.

### 7. Plan revision recommendations
- **`workshop/plans/000002-step-caching-plan.md` (M1 section) + issue line 213** — add a `## Revisions` entry: "M1 (2026-07-05): `Validate` ships as `func(storedD, hash) bool` (not `(hit bool, err error)`) — a hasher failure is folded into MISS at the boundary (the safe direction: MISS only recomputes, never serves stale), so no error return is threaded out. Plan/issue signature text updated to match."
- If the operator keeps the atlas deferral to M3, add a one-line `## Revisions` note making that explicit so §8's "don't defer" is a recorded decision rather than an omission.
