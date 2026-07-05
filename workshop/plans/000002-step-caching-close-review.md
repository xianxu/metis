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

---

## Re-review — 2026-07-05T16:16:31-07:00 (FIX-THEN-SHIP)

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
| timestamp | 2026-07-05T16:16:31-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. Build/vet/full suite are green with real `uv`+`git`, and I independently verified the load-bearing soundness claim that no automated test covers: the stored `Entry.D` is a real first-party code closure (not empty), and editing one source file (`metis/steps/predict.py`) produces a correct *selective* MISS (predict re-runs; split + train still HIT) — the cache is sound, invalidation propagates downstream and not sideways. I confirmed the two prior close-review Important findings (`reads.json` exclusion, `.metis-cache` gitignore, record doc comments) are fixed in the code. Here is my fresh review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The whole-issue close delivers the issue's stated purpose: a working, **sound** validating-trace step cache. `metis run --cache` computes `K_pre`, stores a real read-set `D`, validates by re-hashing via `git hash-object`, and on a HIT materializes the output from the CAS and skips the subprocess. I proved soundness end-to-end myself (not from the docs): D is the genuine code closure per step, an identical re-run HITs all three steps, and a code edit to a file in one step's D re-runs exactly that step while leaving its unaffected siblings HITting. `pkg/cache` is textbook-pure over injected seams; `CanonicalHash` is a clean single-source hashing primitive. Nothing is unsound — every gap I found is safe-direction (a spurious MISS, never a stale/false HIT). What keeps this from a clean SHIP is that the two most soundness-critical paths — the D-revalidation-drives-a-MISS behavior I just verified by hand, and the leaf policy end-to-end from YAML — have **zero automated coverage**, so a future regression in either would ship silently. Both are cheap, non-blocking test adds plus a couple of doc/DRY nits.

### 1. Strengths
- **The cache is genuinely sound, and I verified it independently.** Stored `Entry.D` for the train step = `[metis/io.py, metis/model.py, metis/steps/train.py, metis/trace.py, uv.lock, …]` (real closure, not vacuous). Editing `metis/steps/predict.py` → `split` HIT, `train` HIT, `predict` MISS. That is the issue's headline ("a change propagates downstream but not sideways; recompute only what changed") demonstrated on the real pipeline.
- **ARCH-PURE is clean.** `pkg/cache` (`Kpre`/`Validate`/`Entry` codec) injects its only IO — `Validate(storedD, hash func(path))` (cache.go:57) and `buildD(reads, blobHash)` (trace.go:69) unit-test with map fakes and zero filesystem. The IO shell (`caching.go`, `trace.go`, `trace.py`) is the thin boundary, wired as a `StepExecutor` decorator (run.go:102-109).
- **ARCH-DRY single-source holds.** `record.CanonicalHash` (address.go:21) is the one canonical-hash primitive; `PointAddress`/`OutputHash`/`Kpre` all derive from it, with no leftover inline `json.Marshal→HashOf`. `OutputKey` was cleanly removed. `repo.Root` is the shared module-root walk.
- **Safe-direction error handling throughout.** `isHit` (caching.go) returns MISS on any hasher failure; `Validate` treats a vanished/changed file as MISS; `loadReadSet` treats an absent `reads.json` as an empty read-set. Error-swallowing here can never yield an unsound HIT — I checked each path.
- **`uv.lock` fold into D is correctly wired** (caching.go `recordMiss`) and non-aliasing (`append(append([]string(nil), rs.Reads...), depsLockFile)` copies before appending, so `rs.Reads` isn't mutated). It consumes M2's `used_site_packages` flag, closing the dep-upgrade false-HIT.
- **`K_pre` false-HIT vectors are pinned** — `TestKpre_FiveTermSensitivity` exercises each of the five determinants (`uses` included), and `METIS_STEP_DIR` (exec.go:60) is exactly the `stepDir` the cachingExecutor reads, so the sensor↔runner handoff has no path mismatch.

### 2. Critical findings
None. Full suite + real-`uv`/`git` e2es green; I additionally verified soundness (real D, selective code-edit invalidation) by hand.

### 3. Important findings

- **No test exercises real-D revalidation through the runner — the cache's core soundness claim is uncovered** (`cmd/metis/caching_test.go`). Both e2es are blind to it: `TestCache_CheapSweeps` uses `test/echo` steps that write no `reads.json` → **empty D → vacuous HIT**, and `TestCache_ToyPipelineHitsOnRerun` only checks an *identical* re-run, which HITs whether D is real or empty. So if a regression made D silently empty (a path-root break, a sensor-launch break), every step would false-HIT on a code edit and **no test would fail**. I confirmed the real behavior is correct manually; the fix is to lock it: add a run to `TestCache_ToyPipelineHitsOnRerun` that mutates a source file in one step's D (copy the repo's `metis/` into a temp project, edit a `.py`) and asserts that step MISSes while an unaffected step HITs. This is the single highest-value missing test.
- **The v1 leaf policy has no end-to-end coverage from YAML** (`isImmutableLeaf`, caching.go:~270). Both leaf tests construct `experiment.Step` values directly; nothing drives `with: {cache: {leaf: immutable}}` from a parsed experiment through the runner to a K_pre-only HIT. The predicate's `step.With["cache"].(map[string]any)` assertion silently depends on yaml.v3 decoding nested maps as `map[string]any` — I verified it does today, but a YAML-lib swap or a `With` type change would kill the leaf policy (a stated Done-when item) with a green suite. Add a runner-level test that parses a leaf-marked step and asserts it HITs on K_pre alone.

### 4. Minor findings
- **`Entry` doc overstates git-tracking** (cache.go:74 "git-trackable JSON so the index survives … across branches"), but `ensureCacheGitignore` writes `*` (run.go), ignoring the whole `.metis-cache` including `index/`. The index persists on disk but is *not* git-shared. Consciously deferred (run.go comment), but the `Entry` doc should say "on-disk-persistent (git-sharing the index is a future enhancement)."
- **Atlas `metis run` signature is stale** (`atlas/experiment.md:27` still shows `metis run [--run <id>] <experiment.md>` with no `--cache`). `atlas/index.md:37` does document `--cache`, so this is a consistency nit, not a missing-surface gap. No README.md exists, so the README gate doesn't apply.
- **Duplicated "upstream-from-needs" loop** — `cachingExecutor.kpre` (caching.go:~96) and `buildRecord` (record.go) both iterate `step.Needs` looking up an output-hash map and (for Kpre) rely on sorting. A shared `upstreamHashes(needs, outputs)` helper would consolidate (ARCH-DRY).
- **Output-hash computed twice per cached run** — `recordOutput` (caching.go:260, mid-run to feed downstream K_pre) and `assembleRecord` (record.go). Same function; justified by timing but must stay in lockstep.
- **Cross-language root coupling is implicit.** The Go `c.projectRoot` (nearest `go.mod`) must equal the Python sensor's `_PROJECT_ROOT` (parent of the `metis/` package) for D relpaths to resolve under `git hash-object`; `reads.json` even carries `project_root` but `recordMiss` never checks it against `c.projectRoot`. They coincide for the metis repo (verified), but a relocation would silently degrade to MISS. Worth an assertion or a comment.
- On a HIT, `with.json`/`reads.json` aren't restored into the step dir — a cached step's run dir is incomplete for legibility (harmless; the record carries `With`).
- `gitBlobHashes` batches all D paths into one argv (trace.go:50) — an `ARG_MAX` risk only at large-D scale; fine for v1.

### 5. Test coverage notes
Coverage pins real logic, not mock reassertion: `TestGitBlobHashes_MatchesGit` vs real `git hash-object`; `TestSensor_RecordsFirstPartyCodeReads`/`TestSensor_ExcludesRunDirAndStdlib` drive the real sensor + `_classify` filter contract; both e2es run real skip/materialize. The `pkg/cache` unit tests remain strong. The two gaps (finding #3) are exactly the soundness-critical paths: (a) real-D revalidation → selective MISS on a code edit, and (b) leaf HIT from YAML. A secondary nicety: neither e2e counts *subprocess executions*, so a "HIT that silently re-ran" would still pass on the `⚡` marker + cv reproduction — close but indirect.

### 6. Architectural notes for upcoming work
- **ARCH-PURPOSE — purpose delivered.** Shadow-sweep of Done-when: pure `pkg/cache` ✓, Python sensor + Go blob-hasher ✓, runner skip/materialize + leaf policy ✓, cheap-sweeps e2e ✓, atlas ✓. Deferring the **#3 record's** `Code.D`/`Deps` *provenance* to #8 is legitimate — the cache's *functional* `Entry.D` (what decides HIT/MISS) is fully populated (I verified it holds the real closure); only the record's code-manifest field stays empty, entangled with #8's git-side-ref durability. Pass.
- **The design's `hash(K_pre, D)` output address was collapsed to `K_pre → {single D, output}`.** Toggling between two code versions that share a K_pre always MISSes (the index stores one D per K_pre; a re-run overwrites it). Safe (never stale) and it fully serves the param-sweep purpose the issue is about, but it's a real reduction from the design's two-key output store — worth an explicit note if branch-toggle reuse ever matters.
- **#8 wiring `Code.D` into the record must source it from the cache `Entry`, not a fresh `reads.json`** — a HIT produces no new `reads.json`, so the record's D provenance on a cached run has to come from the index entry. The current empty-on-HIT behavior is the seam #8 fills.
- The "everything in the step dir except reserved channels" artifact model (exec.go:133) keeps accreting reserved-channel special cases (`reads.json` was the second). Prefer moving sensor sidecars outside the artifact tree (run-dir metadata) when #8 touches this.

### 7. Plan revision recommendations
- The plan's `## Revisions` already records the M3 realities (OutputKey drop, `reads.json` exclusion, `Code.D` deferral to #8) and matches the code — no plan-vs-code contradiction remains.
- Add one `## Revisions` line noting the coverage debt as owed, so it's a recorded decision rather than a silent gap: *"Close-review: two soundness-critical paths ship without automated coverage — (1) real-D revalidation driving a selective MISS on a source edit (verified by hand at close; both e2es are blind to it because CheapSweeps' test/echo steps yield an empty D), and (2) the leaf policy end-to-end from a YAML-parsed `cache:{leaf:immutable}` marker. Add both tests as immediate follow-up."*

---

## Re-review — 2026-07-05T16:29:45-07:00 (FIX-THEN-SHIP)

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
| timestamp | 2026-07-05T16:29:45-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. Build, vet, and the full suite are green with real `uv`+`git` present (the gated toy-pipeline e2e ran 3.37s, the sensor test 1.10s, `TestGitBlobHashes_MatchesGit` against real git — all execute, none skip), I reproduced the cache working end-to-end (cold MISS → re-run `⚡ cache hit`), and I reproduced one contract violation the three prior close-reviews all missed. Here is my fresh review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The whole-issue close delivers the issue's stated purpose: a working, sound validating-trace step cache. `metis run --cache` computes `K_pre`, stores a real read-set `D`, validates by re-hashing via `git hash-object`, and on a HIT materializes the output from the CAS and skips the subprocess. I verified soundness independently — `TestCachingExecutor_RealDMissesOnSourceEdit` and `TestCache_LeafPolicyFromParsedYAML` (the two paths the prior reviews flagged as uncovered) are now present, execute, and pass, and I ran the cache by hand (identical re-run HITs every step). `pkg/cache` is textbook-pure over injected seams; `CanonicalHash` is a clean single-source primitive. What keeps this from a clean SHIP is one fresh Important finding: the cachingExecutor **violates the CAS package's own documented consumer contract** — `materialize` hard-fails the entire run when a cached output blob is missing/corrupt/evicted, instead of recomputing. It's safe-direction (fails loudly, never serves stale), so non-blocking at the gate, but it contradicts three stated contracts and is cheap to fix.

### 1. Strengths
- **The cache is genuinely sound and works end-to-end — I reproduced both.** Cold run MISSes all steps; identical re-run emits `⚡ step … (cache hit)` for every step. `TestCache_ToyPipelineHitsOnRerun` drives the real uv/Python pipeline twice (3.37s, actually executes), HITs all three steps, reproduces `cv_score` from cache, and materializes `predictions.csv`.
- **The prior close-reviews' flagged coverage gaps are genuinely closed.** `TestCachingExecutor_RealDMissesOnSourceEdit` (caching_test.go) builds a real git repo, stores a non-empty D, asserts HIT-while-clean and MISS-on-byte-edit — the core "recompute only what changed" claim, locked at the unit level. `TestCache_LeafPolicyFromParsedYAML` drives the leaf marker through the real `experiment.Parse` (yaml.v3 → `map[string]any`). Both pass.
- **ARCH-PURE is clean.** `pkg/cache` injects its only IO — `Validate(storedD, hash func(path))` (cache.go:57) and `buildD(reads, blobHash)` (trace.go:69) unit-test with map fakes and zero filesystem. The IO shell (`caching.go`, `trace.go`, `trace.py`) is the thin boundary, wired as a `StepExecutor` decorator (run.go:102-109).
- **ARCH-DRY single-source holds.** `record.CanonicalHash` (address.go:21) is the one canonical-hash primitive; `PointAddress`/`OutputHash`/`Kpre` all derive from it — grep confirms no leftover inline `json.Marshal→HashOf`. `OutputKey` cleanly removed.
- **Safe-direction error handling on the D path.** `isHit` (caching.go:141-144) returns MISS on any git-hasher failure; `loadReadSet` treats absent `reads.json` as empty. Error-swallowing there can never produce an unsound HIT.
- **`uv.lock` fold into D is correct and non-aliasing** (`append(append([]string(nil), rs.Reads...), depsLockFile)`, caching.go:205 — copies before appending, so `rs.Reads` isn't mutated), closing the dep-upgrade false-HIT.

### 2. Critical findings
None. The cache is sound (every gap is a spurious MISS or a loud failure, never a stale serve); full suite + real-uv/git e2es green; I additionally verified soundness by hand.

### 3. Important findings

- **`cachingExecutor` violates the CAS package's documented "wipeable cache" consumer contract — a missing/corrupt/evicted output blob hard-fails the run instead of recomputing** (`cmd/metis/caching.go:161` and `:171`). `pkg/cas` states the contract explicitly (cas.go:44-46): *"A wipeable-cache consumer treats [ErrNotFound/ErrCorrupt] like ErrNotFound and recomputes."* The `cachingExecutor` is that consumer, but `materialize` propagates both `store.Get` errors verbatim (`return experiment.StepResult{}, fmt.Errorf("cache: get output manifest: %w", err)`), and `Runner.Run` turns any `Execute` error into a failed run (run.go:83-86). I **reproduced it**: run a 2-step `test/echo` experiment cached, then `rm -rf .metis-cache/cas` (keeping `index/`), re-run →
  ```
  metis: step "prep": cache: get output manifest: cas: blob not found
  ```
  **exit code 1** — a hard failure. This contradicts three stated contracts: (a) the CAS consumer contract above; (b) the issue's durability design (lines 158-165, 183: *"rm -rf cas/ loses only recomputable output bytes … wiped → recompute via the record's recipe"*); (c) the code's own comments (run.go:33 *"the cache is safe to `rm -rf` and rebuild"*; the `Entry` doc). It is safe-direction (fails loudly, never serves stale) so not Critical, but it's a genuine contract gap. Two realistic triggers even today: a partial `rm -rf cas/` (the design explicitly calls this safe) and on-disk **corruption** (`ErrCorrupt` — the exact event the CAS integrity-check defends against and instructs the consumer to recover from). It is also a **latent happy-path bug**: the FSStore is LRU-eviction-capable (constructed `maxBytes=0` → eviction off today, but eviction is the store's designed purpose per cas.go:78); enabling it makes a HIT-then-evicted step fail during normal operation. Fix: in `Execute`, treat a `materialize` failure where `errors.Is(err, cas.ErrNotFound) || errors.Is(err, cas.ErrCorrupt)` as a MISS (fall through to recompute), propagating only other errors; add a test that stores an entry, wipes the CAS blob, and asserts the step recomputes rather than erroring. This is exactly the recompute-on-eviction seam #8's durability layer will build on — honoring the CAS contract now is #2's job.

### 4. Minor findings
- **ARCH-DRY: duplicated "upstream-from-needs" loop** — `cachingExecutor.kpre` (caching.go:100-105) and `buildRecord` (record.go:96-102) both iterate `step.Needs` against an output-hash map. A shared `upstreamHashes(needs, outputs)` helper would consolidate.
- **Output-hash computed twice per cached run** — `recordOutput` (caching.go:265, mid-run to feed downstream K_pre) and `assembleRecord` (record.go:76, post-run). Same function; must stay in lockstep.
- **`Entry` doc overstates git-tracking** (cache.go:74 "survives … across branches") while `ensureCacheGitignore` writes `*`, ignoring the whole `.metis-cache` including `index/`. The index is on-disk-persistent but not git-shared (consciously deferred).
- **Immutable-leaf HIT bypasses D (code) revalidation entirely** (caching.go:134) — editing a leaf's fetch code won't bust its cache. Documented conscious v1 bet; flagged for the operator (matters only if a leaf ever carries first-party transform code vs a pure fetch).
- On a HIT, `with.json`/`reads.json` aren't restored into the step dir — a cached step's run dir is incomplete for legibility (harmless; the record carries `With`).
- `gitBlobHashes` batches all D paths into one argv (trace.go:50) — an `ARG_MAX` risk only at large-D scale; fine for v1.
- The `cache` policy knob lives inside `with` (caching.go:274) — a runner concern hashed into K_pre, written to `with.json`, and rendered in the `## Runs` knobs line. Cosmetic namespace conflation.

### 5. Test coverage notes
Coverage pins real logic, not mock reassertion: `TestGitBlobHashes_MatchesGit` vs real `git hash-object`; `TestSensor_RecordsFirstPartyCodeReads`/`_ExcludesRunDirAndStdlib` drive the real sensor + `_classify` filter contract; `TestCachingExecutor_RealDMissesOnSourceEdit` locks the core selective-invalidation soundness the e2es were blind to; both e2es run real skip/materialize. The prior reviews' two flagged gaps are now covered — good. New gaps, priority order: (1) the wipeable-cache recompute-on-missing-blob behavior (finding #1 — the test to add with the fix); (2) neither e2e counts *subprocess executions*, so a "HIT that silently re-ran" would still pass on the `⚡` marker + cv reproduction (close but indirect).

### 6. Architectural notes for upcoming work
- **ARCH-PURPOSE — purpose delivered.** Shadow-sweep of Done-when: pure `pkg/cache` ✓, Python sensor + Go blob-hasher ✓, runner skip/materialize + leaf policy ✓, cheap-sweeps + real-pipeline e2es ✓, atlas ✓ (index.md + experiment.md's `--cache` both updated; no README exists so that gate doesn't apply). Deferring the **#3 record's** `Code.D`/`Deps` *provenance* to #8 is legitimate — the cache's *functional* `Entry.D` (what decides HIT/MISS) is fully populated (I verified it holds the real closure); only the record's code-manifest field stays empty, entangled with #8's git-side-ref durability. Pass.
- **#8 wiring `Code.D` must source it from the cache `Entry`, not a fresh `reads.json`** — a HIT produces no new `reads.json`, so the record's D provenance on a cached run has to come from the index entry. The current empty-on-HIT behavior is the seam #8 fills — and finding #1's recompute-on-eviction fix is the same durability seam.
- **The design's `hash(K_pre, D)` output address collapsed to `K_pre → {one D, output}`** — toggling between two code versions that share a K_pre always MISSes (the index stores one D per K_pre; a re-run overwrites). Safe (never stale) and fully serves the param-sweep purpose; worth an explicit note only if branch-toggle reuse ever matters.
- The "everything in the step dir except reserved channels" artifact model (exec.go:133) keeps accreting reserved-channel special cases (`reads.json` was the second). Prefer moving sensor sidecars outside the artifact tree (run-dir metadata) when #8 touches this.

### 7. Plan revision recommendations
- The plan's `## Revisions` already records the M3 realities (OutputKey drop, `reads.json` exclusion, `Code.D` deferral to #8, the `hash(K_pre,D)` collapse, the soundness-coverage debt) and matches the code — no plan-vs-code contradiction remains.
- Add one `## Revisions` line to `workshop/plans/000002-step-caching-plan.md` recording finding #1 as owed, so it's a recorded decision rather than a silent gap: *"Close-review: `cachingExecutor.materialize` does not honor the CAS wipeable-cache consumer contract (cas.go: treat ErrNotFound/ErrCorrupt as recompute) — a missing/corrupt/evicted output blob hard-fails the run (reproduced: `rm -rf .metis-cache/cas` → `cas: blob not found`, exit 1) instead of falling through to a MISS. Contradicts the durability design's 'rm -rf cas/ is safe → recompute' and becomes a happy-path bug once FSStore eviction (maxBytes>0) is enabled. Fix: treat a materialize ErrNotFound/ErrCorrupt as a MISS; add a wipe-then-recompute test."*
