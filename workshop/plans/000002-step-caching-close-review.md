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
