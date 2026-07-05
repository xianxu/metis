# Boundary Review — metis#2 (whole-issue close)

| field | value |
|-------|-------|
| issue | 2 — Uniform DAG step caching: content-address step inputs, skip unchanged, recompute only what changed |
| repo | metis |
| issue file | workshop/issues/000002-step-caching.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0630b426ff15f84f11cb3121289f7a6f83936279..HEAD |
| command | sdlc close --issue 2 |
| reviewer | claude |
| timestamp | 2026-07-05T15:56:34-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. Build, vet, and the full suite (including the real-`uv` toy-pipeline e2e) are green, and I've verified the load-bearing claims against the running code — including reproducing the one real defect (the sensor's `reads.json` is being folded into every step's output identity). Here's the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M3 boundary (whole-issue close) delivers the issue's stated purpose: a working validating-trace step cache — `metis run --cache` computes `K_pre`, validates the stored read-set `D` by re-hashing via `git hash-object`, and on a HIT materializes the output from the CAS and skips the subprocess. I reproduced the payoff end-to-end (cold run MISSes all steps, identical re-run HITs all three, cheap-sweeps knob change re-runs only downstream) and confirmed build/vet/full-suite green with the gated uv/git tests actually executing. Nothing is unsound: every gap I found is safe-direction (a spurious MISS, never a stale/false HIT). What keeps this from a clean SHIP is one real defect — the sensor's `reads.json` bookkeeping is collected as a step artifact and folded into each step's output-hash (which feeds downstream `K_pre`), embedding a machine-absolute `project_root` and causing every upstream *code* edit to bust all downstream even when the upstream output is byte-identical — plus a `.metis-cache` gitignore gap and a matching test-coverage hole. All cheap and non-blocking.

### 1. Strengths
- **The cache genuinely works end-to-end, proven with real artifacts.** `TestCache_ToyPipelineHitsOnRerun` drives the real uv/Python pipeline twice; the re-run HITs all three steps, reproduces `cv_score` from cache, and materializes `predictions.csv`. I independently reproduced this (r1 cold → r2 all `⚡ cache hit`).
- **ARCH-PURE is clean.** `pkg/cache` (`Kpre`/`Validate`/`Entry` codec) is pure over injected seams; `Validate` (cache.go:64) takes `hash func(path)` so the whole package unit-tests with map fakes and zero filesystem. The IO shell (`caching.go`, `trace.go`, `trace.py`) is the thin boundary. `buildD` is pure over an injected hasher (trace.go:69).
- **ARCH-DRY single-source verified.** `record.CanonicalHash` (address.go:21) is the one hashing primitive; shadow-sweep confirms all three surviving views derive from it — `PointAddress` (address.go:34), `OutputHash` (address.go:58), `Kpre` (cache.go) — no leftover inline `json.Marshal→HashOf`. `OutputKey` was cleanly removed (only comment mentions remain). `repo.Root` shared.
- **`isHit`'s safe-direction default is correct** (caching.go:141-144): a hasher failure → MISS, so error-swallowing here can never produce an unsound HIT. The `uv.lock`-into-D fold (caching.go:203) correctly consumes M2's `used_site_packages` flag, closing the dep-upgrade false-HIT the M2 review flagged.
- **`K_pre` false-HIT vectors are pinned** — `TestKpre_FiveTermSensitivity` genuinely exercises each of the five determinants, `uses` included.

### 2. Critical findings
None. The cache is sound (every gap is a spurious MISS, never a stale serve); full suite + real-uv e2e are green.

### 3. Important findings

- **`reads.json` (sensor bookkeeping) is collected as a step artifact and folded into the output-hash** (`cmd/metis/exec.go:131`; evidence: the stored manifests are `split/[folds.json, reads.json]`, `train/[model.pkl, reads.json]`, `predict/[predictions.csv, reads.json]`, and each `reads.json` carries `"project_root": "/Users/xianxu/workspace/metis"`). `collectArtifacts` excludes only `with.json`/`metrics.json` at the top level, so the sensor's sidecar becomes a genuine artifact → it enters `record.OutputHash` → downstream `K_pre`. Two consequences, both safe-direction (never stale) so not Critical, but they erode the core promise:
  - *Within a machine:* editing any upstream step's **code** so its read-set shifts (a refactor, an added helper import, a logging change) but its real output is byte-identical still moves that step's output-hash (because `reads.json` changed) → every downstream step MISSes. That directly undercuts the issue's "a change propagates downstream but not sideways / recompute only what changed" — editing upstream code busts *all* downstream even when the upstream output is unchanged.
  - *Across machines / relocated checkout:* the absolute `project_root` in `reads.json` differs → output-hash differs → downstream MISS, defeating the design's git-trackable index "survives across runs, sessions, and branches" claim.
  - Fix: add `reads.json` to the reserved-channel exclusion in `collectArtifacts` (top level, alongside `with.json`/`metrics.json`) — `recordMiss` still reads it via `loadReadSet(stepDir)`, it just shouldn't be a hashed output. Add a test asserting `reads.json ∉ artifacts`.

- **`.metis-cache` is not gitignored and nothing writes an ignore file** (`cmd/metis/run.go:87`). The store lands `cas/<hash>` output **blobs** (`model.pkl`, parquet) under `<expDir>/.metis-cache/`. In a real experiment repo (kbench), `git status` shows them untracked and a `git add -A` commits binary blobs — yet the design explicitly says the CAS is a wipeable *local* cache and only the *index* is git-trackable. Nothing enforces that split. Fix: on first `--cache` use, write `.metis-cache/.gitignore` ignoring `cas/` (or the whole dir), or document it. Non-blocking (safe), but a production footgun the moment the cache runs in a tracked repo.

- **Test-coverage gaps on the two behaviors most likely to silently break the cache.** (a) No test asserts `reads.json` stays out of the output (finding #1 would have been caught at the unit level). (b) The **immutable-leaf runner HIT** — HIT on `K_pre` alone, *bypassing* D re-hash (caching.go:133-136) — is only marker-tested (`TestCache_ImmutableLeafMarker` checks the predicate); a regression that made a leaf re-validate D, or never HIT, would pass CI. Add a runner-level test: a leaf whose D would MISS still HITs on the K_pre match.

### 4. Minor findings
- Stale package doc: `pkg/cache/cache.go:14` still lists `OutputKey` as part of the pure core, but M3 removed it — drop it from the doc comment.
- ARCH-DRY: per-step output-hash is computed twice per run — `cachingExecutor.recordOutput` (caching.go:260) and `assembleRecord` (record.go:70-77) both run `record.OutputHash(hashArtifacts(...))`. Justified by timing (the executor needs it mid-run to feed downstream K_pre), but two paths that must stay in lockstep; consider letting `assembleRecord` reuse the executor's accumulated `outputs`.
- Root inconsistency: `recordMiss` hashes D against `rs.ProjectRoot` (fallback `c.projectRoot`) (caching.go:195-208) while `isHit` always uses `c.projectRoot` (caching.go:141). If they ever diverge the HIT re-hash resolves D paths against a different root → spurious MISS (safe, but the two should share one root).
- Library-vs-CLI default drift: `runOpts.cache` zero-value is OFF (run.go:38); the CLI `--cache` defaults ON (main.go). Existing e2e tests therefore run uncached — intentional, but easy to trip on.
- The `cache` policy knob lives inside `with`, so it's hashed into `K_pre`, rendered in the `## Runs` knob→score line (`get.cache=map[leaf:immutable]`), and written to the step's `with.json`. Cosmetic.
- Immutable-leaf HIT bypasses **code** (D) validation entirely, so editing the leaf's fetch code won't bust its cache — a documented conscious v1 bet, flagged for the operator (matters only if leaves ever carry nontrivial first-party transform code vs a pure fetch).

### 5. Test coverage notes
Coverage matches the M3 plan for the happy paths and pins real logic (not mock reassertion): `gitBlobHashes` vs real `git hash-object`, the sensor's first-party closure via a real uv subprocess, the sensor filter contract (`_classify` run-dir/stdlib/venv exclusion), and both cheap-sweeps + real-pipeline e2es. Gaps, priority order: (1) `reads.json ∉ artifacts` (would catch #1); (2) immutable-leaf runner HIT (only the marker is tested); (3) a HIT/MISS assertion that counts *subprocess executions*, not just outcomes, so a "HIT that silently re-ran" can't pass. The `pkg/cache` M1 unit tests remain strong.

### 6. Architectural notes for upcoming work
- **ARCH-PURPOSE — the issue's purpose is delivered.** The cache skips unchanged steps, recomputes changed ones, and proves cheap sweeps with real artifacts. The deferral of the **#3 record's** `Code.D`/`Deps` *provenance* to #8 is legitimate: the cache's *functional* `Entry.D` (what decides HIT/MISS) is fully populated; only the record's code-manifest field — entangled with #8's git-side-ref durability — stays empty. That's a separable extension, not the deferred point. Pass. (But note finding #1 partially undercuts the "recompute only what changed" sub-promise for the upstream-code-refactor case — worth closing before this becomes load-bearing.)
- When fixing #1, prefer a longer-term move to an explicit sensor-sidecar location outside the artifact tree (e.g. run-dir metadata) over relying on an exclusion list — the current "everything in the step dir except reserved channels" model will keep accreting reserved-channel special cases.
- #8 wiring `Code.D` into the record must reconcile the HIT case (no fresh `reads.json` is produced on a HIT) — the D provenance has to come from the cache `Entry`, not the step dir.

### 7. Plan revision recommendations
- **`workshop/plans/000002-step-caching-plan.md` — add a `## Revisions` entry:** "M3 follow-up: `reads.json` is currently collected as a step artifact and folded into the output-hash (embedding the sensor's read-set + absolute `project_root`). Exclude it in `collectArtifacts` so the output identity doesn't depend on the sensor's bookkeeping — required for the 'recompute only what changed' propagation guarantee and the git-trackable index's cross-session/branch/machine sharing claim. Add a test asserting `reads.json ∉ artifacts`."
- Same file — record the `.metis-cache/cas` gitignore gap as owed (or a conscious defer), so the untracked-binary-blob footgun is a recorded decision rather than an omission.

---

## Re-review — 2026-07-05T16:06:46-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 2 — Uniform DAG step caching: content-address step inputs, skip unchanged, recompute only what changed |
| repo | metis |
| issue file | workshop/issues/000002-step-caching.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0630b426ff15f84f11cb3121289f7a6f83936279..HEAD |
| command | sdlc close --issue 2 |
| reviewer | claude |
| timestamp | 2026-07-05T16:06:46-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. Build, vet, and the full suite (including the real-`uv` toy-pipeline e2e at 3.54s and the sensor tests at 1.18s) are green with `uv`+`git` present, and I've verified the load-bearing claims against the running code — including confirming the two Important findings from the prior close-review (`reads.json` exclusion, `.metis-cache` gitignore) are now correctly applied, and that the DRY single-source consolidation holds.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The whole-issue close delivers the issue's stated purpose: a working validating-trace step cache. `metis run --cache` computes `K_pre`, validates the stored read-set `D` by re-hashing via `git hash-object`, and on a HIT materializes the output from the CAS and skips the subprocess. I independently ran the suite: the toy-pipeline e2e HITs all three steps on re-run and reproduces `cv_score` from cache; `TestCache_CheapSweeps` proves the knob-change propagation; the sensor genuinely captures the first-party code closure through a real `uv` subprocess. The prior close-review's two Important findings are verifiably fixed in the code I read (`collectArtifacts` now excludes `reads.json` at exec.go:134; `ensureCacheGitignore` writes the ignore at run.go:34). Nothing is unsound — every gap I found is safe-direction (a spurious MISS, never a stale/false HIT). What keeps this from a clean SHIP is one doc-contract defect at the exact #3→#8 seam #2 hands off to: the shared `record` package still claims "metis#2 populates" fields that #2 either already populated differently or explicitly deferred to #8 — a one-line-each fix, non-blocking, but it will mislead #8's implementor if left.

### 1. Strengths
- **The cache works end-to-end, proven with real artifacts, and I reproduced it.** `TestCache_ToyPipelineHitsOnRerun` drives the real uv/Python pipeline twice; re-run HITs all three steps, reproduces `cv_score` from cache, and materializes `predictions.csv`. Ran green (3.54s) — the gated tests actually execute, not skip.
- **ARCH-PURE is clean.** `pkg/cache` (`Kpre`/`Validate`/`Entry` codec) is pure over injected seams — `Validate(storedD, hash func(path))` (cache.go:57) unit-tests with map fakes and zero filesystem; `buildD` is pure over an injected hasher (trace.go:69). The IO shell (`caching.go`, `trace.go`, `trace.py`) is the thin boundary, injected as a `StepExecutor` decorator (run.go:102-109).
- **ARCH-DRY single-source verified by grep.** `record.CanonicalHash` (address.go:21) is the one canonical-hashing primitive; `OutputKey` is fully removed from code (only a comment explaining its removal survives at cache.go:71); no leftover inline `json.Marshal→HashOf` of a struct anywhere. `PointAddress`/`OutputHash`/`Kpre` all derive from it.
- **`isHit`'s safe-direction default is correct** (caching.go:141-144): a hasher failure → MISS, so the error-swallow there can never produce an unsound HIT. The `uv.lock`-into-D fold (caching.go:203) correctly consumes M2's `used_site_packages` flag, closing the dep-upgrade false-HIT the M2 review flagged.
- **`K_pre` false-HIT vectors are genuinely pinned** — `TestKpre_FiveTermSensitivity` exercises each of the five determinants (`uses` included), and `TestCachingExecutor_ImmutableLeafBypassesDValidation` pins the runner-level leaf bypass (a leaf whose D would MISS still HITs; a normal step MISSes) — the prior review's requested coverage was added.

### 2. Critical findings
None. The cache is sound (every gap is a spurious MISS, never a stale serve); full suite + real-uv e2e green.

### 3. Important findings

- **The shared `record` package still claims "metis#2 populates" fields that #2 did not — stale contract at the #3→#8 handoff** (`pkg/record/record.go:33,37,38` and `cmd/metis/record.go:85`). Concretely: `record.go:37-38` annotate `D`/`Deps` with `// metis#2 populates`, and `record.go:33` says they "are defined slots the metis#2 validating trace populates" — but #2 explicitly **deferred** the record's `Code.D`/`Deps` provenance to #8 (per the issue Log and plan Revisions), so those slots stay empty. Worse, `cmd/metis/record.go:85` reads "Upstream / Code.D / Deps are left empty — metis#2 populates them" sitting directly above lines 91-105 that **do** populate `Upstream` — a comment contradicting its own adjacent code. Because #8 is the next consumer of exactly this record surface, the misstatement is a real handoff hazard, not cosmetic. Fix: update the four annotations to reflect the delivered state — `Upstream` is populated by `buildRecord`; `Code.D`/`Deps` remain empty, deferred to #8. Doc-only, cheap, non-blocking.

- **The marquee "cheap sweeps with a real read-set" combination isn't tested as one path.** The two e2es split the payoff: `TestCache_CheapSweeps` uses `test/echo` steps, which write no `reads.json` → empty `D` → **vacuous** HIT (never exercises D-revalidation during a partial hit), while `TestCache_ToyPipelineHitsOnRerun` exercises real `D` but only for a full identical re-run. There is no single test proving the issue's headline: *real pipeline, change one downstream knob → upstream's real `D` re-hashes clean and HITs while only downstream MISSes*. The constituent behaviors are each covered, so this is a coverage gap rather than a suspected bug — but it's the exact bug class the issue exists to prevent. Cheap fix: add a third run to `TestCache_ToyPipelineHitsOnRerun` that flips a `predict`/`train` knob and asserts `split` still HITs (real D) while the changed step + downstream MISS.

### 4. Minor findings
- The `cache` policy knob lives inside `with` (caching.go:274) — a runner-level concern threaded through the step's *business* config, so it's hashed into `K_pre`, written to the step's `with.json`, and rendered in the `## Runs` knobs line. Namespace conflation; works because no step-type validates `with` strictly. Consider a dedicated step attribute later.
- `ensureCacheGitignore` writes `*`, ignoring the **whole** `.metis-cache` including the index — so the `Entry` doc's "git-trackable JSON so the index survives … across branches" (cache.go:74) is aspirational; the index is on-disk-persistent but not actually git-tracked. Consciously deferred ("Sharing the git-trackable index … is a future enhancement"), but the Entry doc overstates it.
- Output-hash is computed twice per cached run — `cachingExecutor.recordOutput` (caching.go:260, mid-run to feed downstream K_pre) and `assembleRecord` (record.go:70-77, post-run). Same function, must stay in lockstep; justified by timing but note it if a third caller appears.
- On a HIT, neither `with.json` nor `reads.json` is written into the step dir, so a cached step's run dir is incomplete for legibility (harmless for correctness — the record carries `With` directly).
- `ensureCacheGitignore` runs before experiment validation (run.go:105), so an invalid experiment still creates an empty `.metis-cache/`. Harmless.
- `gitBlobHashes` batches all D paths into one argv (trace.go:50) — an ARG_MAX risk only at large-D scale; fine for v1 (carried over from the M2 review).

### 5. Test coverage notes
Coverage pins real logic, not mocks: `TestGitBlobHashes_MatchesGit` compares against real `git hash-object`; `TestSensor_ExcludesRunDirAndStdlib` drives `metis.trace._classify` to lock the run-dir/stdlib/venv exclusion contract; `TestSensor_RecordsFirstPartyCodeReads` proves the first-party closure via a real uv subprocess; both e2es run real skip/materialize. The `pkg/cache` M1 unit tests remain strong. Gaps, priority order: (1) the real-pipeline knob-change cheap-sweep (finding #2); (2) no assertion that counts *subprocess executions* (a "HIT that silently re-ran" would pass — the tests check for the `⚡` marker and cv reproduction, which is close but indirect). Both are enhancements, not blockers.

### 6. Architectural notes for upcoming work
- **ARCH-PURPOSE — purpose delivered.** Shadow-sweep of the "Done when": pure `pkg/cache` ✓, Python sensor + Go blob-hasher ✓, runner skip/materialize + leaf policy ✓, cheap-sweeps e2e + atlas ✓. The deferral of the **#3 record's** `Code.D`/`Deps` *provenance* to #8 is legitimate — the cache's *functional* `Entry.D` (what decides HIT/MISS) is fully populated; only the record's code-manifest field stays empty, and it's entangled with #8's git-side-ref durability. That's a separable extension, not the deferred point. Pass — provided the finding-#1 comments are corrected so the deferral is *documented* as such rather than mis-attributed to #2.
- **#8 wiring `Code.D` must source it from the cache `Entry`, not a fresh `reads.json`** — a HIT produces no new `reads.json`, so the record's D provenance on a cached run has to come from the index entry. The current empty-on-HIT behavior is the seam #8 fills.
- The "everything in the step dir except reserved channels" artifact model (exec.go:133) will keep accreting reserved-channel special cases (`reads.json` was the second). Prefer moving sensor sidecars outside the artifact tree (run-dir metadata) when #8 touches this.

### 7. Plan revision recommendations
- The plan (`workshop/plans/000002-step-caching-plan.md`) already records the M3 realities (OutputKey drop, `reads.json` exclusion, `Code.D` deferral to #8) in its `## Revisions` — it matches the code. No new plan revision required for the code itself.
- Add one `## Revisions` line noting the finding-#1 doc cleanup as owed: "record-package comments still attribute `Code.D`/`Deps` population to #2; corrected to reflect the #8 deferral + `buildRecord`'s `Upstream` population at close." So the stale-comment fix is a recorded decision, not a silent edit.
