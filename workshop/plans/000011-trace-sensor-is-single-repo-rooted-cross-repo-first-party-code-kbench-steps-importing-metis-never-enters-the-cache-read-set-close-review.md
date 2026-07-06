# Boundary Review — metis#11 (whole-issue close)

| field | value |
|-------|-------|
| issue | 11 — Trace sensor is single-repo-rooted — cross-repo first-party code (kbench steps importing metis) never enters the cache read-set |
| repo | metis |
| issue file | workshop/issues/000011-trace-sensor-is-single-repo-rooted-cross-repo-first-party-code-kbench-steps-importing-metis-never-enters-the-cache-read-set.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1d236139fcedd8ce5e04b0d1d3fa7ab057ae6578..HEAD |
| command | sdlc close --issue 11 |
| reviewer | claude |
| timestamp | 2026-07-06T16:24:32-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The multi-root read-set is soundly designed and the core guarantee is real: `D` is now repo-qualified end-to-end, and the store side (`recordMiss`) and validate side (`isHit`) both group/hash per repo, so they can't disagree — the store/validate symmetry the plan calls out is genuinely achieved. Build is green; `go test ./cmd/metis ./pkg/cache ./pkg/record` and `uv run pytest` (37 tests) all pass. Nothing here produces wrong output — the failure directions are uniformly toward MISS/recompute. What keeps this from a clean SHIP is a **test-coverage gap the issue's own Done-when promised** (a real cross-repo `python -m metis.trace` invocation test — currently only `_classify` is driven directly, which the repo's own lessons.md warns against), plus a **factually-wrong safety comment** on the legacy-entry path. Both are cheap, non-blocking fixes.

### 1. Strengths
- **Store/validate symmetry is correct.** `recordMiss` (caching.go:249-263) and `isHit` (caching.go:159-169) both key by `ref.Repo` and hash each repo's paths in its own repo — the false-HIT/MISS pair the plan feared is closed. The persisted `D` carries the repo, so a later run re-hashes identically.
- **The loud v1 guard is the right call** (trace.go:45-54). Detecting `roots == nil` + probing for `project_root`/`reads` and erroring, rather than silently unmarshalling to an empty `D` → vacuous K_pre-only HIT, is exactly the severe-direction guard the plan's Task 3.1c demanded — and it's tested (`TestLoadReadSet_RejectsLegacyV1`).
- **ARCH-PURE upheld:** `buildD` (trace.go:86) and `cache.Validate` (cache.go:57) are pure over an injected hasher and unit-tested with no git; the git IO stays in the thin `gitBlobHashes`/`hashDByRepo` seam.
- **The stdlib-prefix exclusion** (trace.py:43-57) is a real bug the author found and fixed (mis-rooting the uv stdlib under a git-tracked HOME) — the multi-root walk needed it where the old single-`_PROJECT_ROOT` didn't. Well-documented in lessons.md.
- **`TestCachingExecutor_MultiRepoDMissesOnConsumerEdit`** (caching_test.go) pins the exact cross-repo HIT→MISS guarantee the issue exists for.

### 2. Critical findings
None.

### 3. Important findings

- **Missing the real-invocation cross-repo integration test the Done-when promised** (`tests/test_trace.py`, `cmd/metis/trace_test.go`). The issue's Done-when #1 says: *"a test: trace a kbench-style module that imports metis, assert the consumer module appears in D."* The only test that runs the actual `python -m metis.trace` path is `TestSensor_RecordsFirstPartyCodeReads` (trace_test.go:127) — **single repo (metis)**. The multi-root behavior is proven only by driving `_classify`/`_repo_root` directly (`test_classify_groups_reads_by_repo_root`) and by hand-constructing `D` in Go (`TestCachingExecutor_MultiRepoDMissesOnConsumerEdit`). This is precisely the gap the repo's own `workshop/lessons.md` warns about ("The acceptance/integration demo IS the invocation-path test"): the real audit-hook + `sys.modules`-snapshot + cross-repo import-resolution path is never exercised across two repos. *Fix:* add a test that creates a second temp git repo with a package importing metis, runs the sensor on it via `uv run … python -m metis.trace <consumer.module>`, and asserts both repos appear as keys in the emitted `reads.json` `roots`. (Not REWORK: the single-repo real path is proven and multi-root is a localized classification change on top of it — but this is the one test that would catch a real invocation-path surprise.)

- **The legacy-entry safety comment is factually wrong; add an explicit `Repo==""` guard** (caching.go:174-175, and the mirrored claim at 156-158). Both comments assert an empty repo root "git-fails → a MISS, the safe direction." That premise is false: `git -C "" hash-object -- <path>` does **not** fail — `-C ""` is a no-op and git resolves `<path>` against the current working directory (verified: returns a hash, exit 0). For a legacy (pre-#11) on-disk index entry whose `D` refs have `Repo == ""`, `hashDByRepo` therefore runs `git -C "" hash-object` against cwd rather than failing. There is **no wrong-output path** in practice (the legacy paths were relative to the metis root; if cwd is that root the HIT is correct, otherwise git can't find the file → MISS), so this is not a live bug — but the code's stated soundness argument is incorrect, and the migration story is cwd-dependent and undocumented as such. Given `loadReadSet` already chose the loud/safe posture for v1 `reads.json`, the symmetric move is a one-liner in `hashDByRepo`: if any `ref.Repo == ""`, return an error (→ MISS, cwd-independent) instead of relying on `git -C ""`. Then fix both comments to describe the real mechanism.

### 4. Minor findings
- **ARCH-DRY:** `recordMiss`'s inline per-repo hashing loop (caching.go:249-256) duplicates `hashDByRepo`'s hashing loop (caching.go:181-188). Extract `func hashByRepo(byRepo map[string][]string) (map[string]map[string]record.Hash, error)`; have `hashDByRepo` group `[]CodeRef → map[repo][]path` then call it, and have `recordMiss` call it directly on `roots`. One source for "hash each repo's paths in its repo."
- **`captureSweepCode` "primary" commit selection is effectively dead cross-repo** (capture.go:94,118). `primary := cacheProjectRoot(...)` resolves via **`go.mod`** (`internal/repo/repo.go:18`), while `closureByRepo` keys are **`.git`** roots from the Python sensor — a consumer repo (no `go.mod`) will never match, so `commits[primary]` is always `""` and it falls back to the first sorted repo. Harmless (the scalar `commit` is a #14-tracked refinement; `D` + per-repo side refs are correct), but the comment "the primary (expPath) repo's" overstates what's achieved. Worth a truer comment or deferring the whole scalar to #14.
- **Dead defensive line in the test fixture** (tests/test_trace.py:38): `trace._repo_root.cache_clear() if hasattr(...) else None` — `_repo_root` is a plain function, never has `cache_clear`; the real reset is the next line (`trace._root_cache = {}`). Drop the no-op.
- **`_stdlib_prefixes` assumes `os.__file__` exists** (trace.py:49). True on standard CPython/uv; would `AttributeError` at import on a frozen build. Acceptable for this runtime; note only.

### 5. Test coverage notes
- Core logic is well-covered at unit level: Python `_classify` grouping + `.git` dir-vs-file detection + run-dir/site-packages/stdlib exclusion (`tests/test_trace.py`), and Go `buildD` repo-qualification, `Validate` ref-hasher, and the multi-repo HIT→MISS.
- Two gaps: (a) the real two-repo sensor invocation (Important #1 above); (b) no test exercises `captureSweepCode`/`isHit` with a multi-repo capture through the real path, nor a legacy `Repo==""` entry through `isHit` — so the migration behavior the comments describe is unverified. If you add the `Repo==""` guard, pin it with a test.

### 6. Architectural notes for upcoming work
- **ARCH-PURPOSE:** the shadow-sweep of `D` consumers passes — sensor (producer), `recordMiss`, `isHit`, and `captureSweepCode` all derive from the repo-qualified grouping; no hand-maintained restatement remains. The kbench-wrapper flip (Task 3.3) is a genuinely separable cross-repo follow-up (it lives in another repo and can't land in a metis-only change), so deferring it is not under-delivery — but the substrate claim rests on unit tests until the integration test lands.
- metis#14 will want per-repo commits in the record; the current single-`commit` scalar and the `cacheProjectRoot`(go.mod)-vs-sensor(.git) root mismatch are the seams to clean up there.

### 7. Plan revision recommendations
- The plan/issue mark Done-when #1 delivered, but the promised integration test ("trace a kbench-style module that imports metis") is not present — only `_classify`-level and hand-built-`D` tests are. Add a `## Revisions` note to `workshop/plans/000011-trace-multi-root-plan.md` recording that Task 3.1/3.1b's real cross-repo *sensor invocation* test is outstanding (or add the test), so the plan stops claiming coverage the code doesn't yet have.

---

## Re-review — 2026-07-06T16:33:08-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 11 — Trace sensor is single-repo-rooted — cross-repo first-party code (kbench steps importing metis) never enters the cache read-set |
| repo | metis |
| issue file | workshop/issues/000011-trace-sensor-is-single-repo-rooted-cross-repo-first-party-code-kbench-steps-importing-metis-never-enters-the-cache-read-set.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1d236139fcedd8ce5e04b0d1d3fa7ab057ae6578..HEAD |
| command | sdlc close --issue 11 |
| reviewer | claude |
| timestamp | 2026-07-06T16:33:08-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
All green: 37 Python tests pass, no Go failures. My fresh-context review is complete.

```verdict
verdict: SHIP
confidence: high
```

The metis#11 multi-root read-set is soundly delivered end-to-end and the core cross-repo guarantee is real: `D` is repo-qualified from producer (Python sensor `roots` map) through every consumer (`recordMiss` store, `isHit`/`Validate`, `sweepClosure` capture), and the store/validate sides group + hash per-repo identically so they can't disagree. The two Important findings from the round-1 close-review are both applied and verified — the real `python -m metis.trace` cross-repo invocation test (`TestSensor_MultiRepo_CapturesConsumerCode`) closes the invocation-path gap the repo's own lessons.md warns about, and the explicit `Repo==""` guard in `hashDByRepo` (plus corrected comment) makes the legacy-entry handling cwd-independent. Build/vet clean; `go test ./...` and `uv run pytest` (37) all pass. Nothing here produces wrong output — every failure direction is toward MISS/recompute — and the atlas is updated for the new surface. My independent pass surfaces only Minor cleanups (all previously noted or cosmetic), so this crosses the boundary.

### 1. Strengths
- **Store/validate symmetry is genuinely closed.** `recordMiss` (caching.go:243–269) and `isHit` (caching.go:159–169) both key by `ref.Repo` and hash each repo's paths in its own repo; the persisted `Entry.D` carries `Repo`, so a later run re-hashes identically. The false-HIT/MISS pair the plan feared can't arise.
- **The multi-root walk correctly handles nested repos + the stdlib trap.** `_repo_root` (trace.py:67–92) returns the *nearest* `.git` ancestor (dir or file, so linked worktrees/submodules work), and `_STDLIB_PREFIXES` (trace.py:43–57) excludes the Python install so a uv interpreter under a git-tracked HOME isn't mis-rooted as first-party — a real bug the old single-`_PROJECT_ROOT` code was implicitly immune to. Well-documented in lessons.md.
- **Loud v1 rejection is the right severe-direction call** (trace.go:45–54): a legacy `reads.json` fails rather than silently unmarshalling to an empty `D` → vacuous K_pre-only HIT. Tested (`TestLoadReadSet_RejectsLegacyV1`).
- **ARCH-PURE upheld:** `buildD` (trace.go:86) and `cache.Validate` (cache.go:57) are pure over an injected hasher, unit-tested with no git; git IO stays in the thin `gitBlobHashes`/`hashDByRepo` seam.
- **`TestCachingExecutor_MultiRepoDMissesOnConsumerEdit`** pins the exact HIT→MISS-on-consumer-edit guarantee the issue exists for, using the *real* `gitBlobHashes` against two real git repos (not a mock).

### 2. Critical findings
None.

### 3. Important findings
None. (Round-1's two Important findings — missing real-invocation cross-repo test; factually-wrong legacy-entry comment + missing `Repo==""` guard — are both fixed and verified in this window.)

### 4. Minor findings
- **ARCH-DRY (caching.go:255–262 vs 187–194):** `recordMiss`'s inline per-repo hashing loop duplicates `hashDByRepo`'s hashing loop. Extract `hashByRepo(map[repo][]path) (map[repo]map[path]Hash, error)`; have `hashDByRepo` group `[]CodeRef→map` then call it, and `recordMiss` call it directly on `roots`. One source for "hash each repo's paths in its repo." (Noted round-1; still present.)
- **capture.go:94,116–118 — "primary" commit is dead cross-repo.** `cacheProjectRoot` resolves via **go.mod** (`internal/repo/repo.go:15`) while `closureByRepo` keys are **`.git`** roots from the sensor, so for a consumer repo without go.mod `commits[primary]` is `""` and it silently falls back to the first sorted repo. Harmless (the scalar `commit` is a metis#14 refinement; `D` + per-repo side refs are correct), but the comment "the primary (expPath) repo's" overstates it. Truer comment, or defer the scalar wholesale to #14.
- **exec.go:121 — stale field name in comment.** "its absolute `project_root` would defeat cross-machine cache reuse" — `reads.json` no longer has `project_root`; the absolute paths now live in the `roots` map keys. The reasoning still holds; update the term.
- **tests/test_trace.py:38 — dead defensive line.** `trace._repo_root.cache_clear() if hasattr(...) else None` — `_repo_root` is a plain function, never has `cache_clear`; the real reset is the next line (`trace._root_cache = {}`). Drop the no-op.
- **Committed review artifact noise** (`workshop/plans/000011-...-close-review.md:26`): a stray Claude trust-dialog warning line ("Ignoring 6 permissions.allow entries…") is captured inside the `## Review` section above the verdict block. Cosmetic; low-signal in a durable artifact.

### 5. Test coverage notes
- Well-covered at unit level: Python `_classify` per-repo grouping + `.git` dir-vs-file detection + run-dir/site-packages/stdlib exclusion (`tests/test_trace.py`), and Go `buildD` repo-qualification, `Validate` ref-hasher, the multi-repo HIT→MISS (real git), the empty-`Repo` guard, and the real two-repo sensor invocation.
- Residual (acceptable): no single test chains real-sensor `reads.json` → `recordMiss` → `isHit` → consumer-edit → MISS through `cachingExecutor.Execute` for the multi-repo case — but both halves are each exercised against their real seam (sensor real; caching real over actual `gitBlobHashes`), and single-repo `recordMiss` is covered by existing e2e. Not a boundary blocker.

### 6. Architectural notes for upcoming work
- **ARCH-PURPOSE shadow-sweep passes:** every `D` consumer (sensor producer, `recordMiss`, `isHit`, `sweepClosure`/`captureSweepCode`, `backfillCodeManifest`) derives from the repo-qualified grouping; no hand-maintained restatement of the model remains. The kbench-wrapper flip (plan Task 3.3) is genuinely separable — it lives in another repo and can't land in a metis-only change — so deferring it is not under-delivery.
- metis#14 is the natural home for per-repo commits in the record; the single-`commit` scalar and the `cacheProjectRoot`(go.mod)-vs-sensor(.git) root mismatch are the seams to clean up there. The repo-qualified `D` now aligns with the point-address's per-repo `repo_shas`.

### 7. Plan revision recommendations
None. The round-1 review already recorded the FIX-THEN-SHIP verdict and the two applied fixes in the issue `## Log`, and the plan/Done-when now matches the code (all three Done-when items delivered; kbench flip correctly logged as a tracked cross-repo follow-up). No `## Revisions` entry needed.
