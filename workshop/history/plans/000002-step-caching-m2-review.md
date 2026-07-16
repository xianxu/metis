# Boundary Review — metis#2 (milestone M2)

| field | value |
|-------|-------|
| issue | 2 — Uniform DAG step caching: content-address step inputs, skip unchanged, recompute only what changed |
| repo | metis |
| issue file | workshop/issues/000002-step-caching.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | d0eae45991a603403ab0e64d8d7f19423ca9a85a^..HEAD |
| command | sdlc milestone-close --issue 2 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-05T15:32:19-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. Verified: build + vet + full suite (incl. `-race` on the new pure pkgs) green; uv/git/cue all present so the gated sensor/conformance/e2e tests genuinely executed (toy pipeline ran 3.11s through the new sensor path); the M2 surface (`metis/trace.py`, `cmd/metis/trace.go`) is correct and its load-bearing claims are pinned by real tests; `runSummary`→`recordSummary`/`formatMetrics` consolidation left no duplication; no user-facing README exists to update.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M2 boundary — the read-set sensor (Python) + the git blob-hasher (Go) — delivers exactly its scoped surface as tested-but-not-yet-integrated units (the runner skip/materialize is correctly deferred to M3). Build/vet/test are green, and the two correctness-critical claims are pinned by tests that exercise real logic, not mocks: `TestGitBlobHashes_MatchesGit` proves `gitBlobHashes` equals real `git hash-object` (the hash the future `Validate` re-computes), and `TestSensor_RecordsFirstPartyCodeReads` proves the sensor captures the first-party code closure (`metis/io.py`, `steps/train.py`) + the `used_site_packages` flag while keeping data out of D — through a real `uv` subprocess. Nothing is unsound: every gap I found is safe-direction (a spurious MISS, never a false HIT). What keeps this from a clean SHIP is a genuine test-coverage gap on the sensor's *filters* (the load-bearing run-dir exclusion and the write-vs-read distinction are both unexercised) plus plan-text drift (`uvLockDigest`, listed under M2, wasn't built) — both cheap and non-blocking.

### 1. Strengths
- **`gitBlobHashes` is validated against the source of truth, not re-asserted** (`cmd/metis/trace.go:46`; `trace_test.go:61`). The test hashes real files with the store's code *and* with `git hash-object` and compares — so the "git's blob-hash IS the content-hash" invariant the whole cache rests on is actually pinned, and the batch (one `git hash-object` call for N paths) is length-checked (`trace.go:53`).
- **The sensor's code-closure capture is robust to import caching** (`trace.py:66` `_snapshot_modules`). Folding `sys.modules[*].__file__` in at exit means a module imported *before* the audit hook installed still lands in D — the correct answer to the obvious "the hook missed the early imports" objection, and it's tested (train.py imports happen at module top, before the step fails).
- **Graceful absence handling is the safe direction** (`trace.go:27` `loadReadSet`): an absent `reads.json` is an empty read-set, not an error — a step that ran without the sensor becomes a K_pre-only entry (more likely to MISS, never a false HIT). Tested (`TestLoadReadSet_AbsentIsEmpty`).
- **`buildD` is pure over an injected hasher** (`trace.go:69`) — the runner will close it over a batch-computed map, so the reads→D mapping unit-tests with zero git. Clean ARCH-PURE seam; error propagation tested.
- **ARCH-DRY on the summary renderer**: `runSummary` was fully deleted from `run.go` and its metric-formatting reused via `formatMetrics` in `record.go:184` — no leftover duplicate formatter.

### 2. Critical findings
None.

### 3. Important findings
- **The sensor's load-bearing *filters* are untested** (`metis/trace.py:52-58`). `TestSensor_RecordsFirstPartyCodeReads` asserts what D *includes* (first-party code) and that no data leaks, but never asserts what D must *exclude*: (a) the **`METIS_RUN_DIR` exclusion** (`trace.py:55`) — a step's own outputs and upstream artifacts live under `runs/` and must NOT enter D, or at M3 *every step MISSes forever* (its upstream artifacts change every run); (b) that a **write** to a first-party path isn't recorded (see Minor below). Both are the exact regressions that would silently defeat the cache and only surface later via the M3 e2e. Fix sketch: extend the sensor test (or add a sibling) to `open()` a file under `METIS_RUN_DIR` and a first-party source read, then assert only the source appears in `rs.Reads`. Cheap, and it locks the filter contract at the unit level where it's debuggable.

### 4. Minor findings
- **`uvLockDigest` (plan M2 bullet) not delivered** (`workshop/plans/000002-step-caching-plan.md` M2). The sensor captures `used_site_packages` (`trace.go:20`), but nothing computes the uv.lock digest and `record.CodeManifest.Deps` (`pkg/record/record.go:38`) is never populated. Deferring the *computation* to M3 (where it's consumed) is defensible, but the plan text claims M2 — record it (plan revision below).
- **Docstring overstates: D is not "reads only."** `_audit` (`trace.py:61`) records every `open` audit event — which fires for **writes too**, not just reads — yet the module docstring (`trace.py:15`) and design say "reads only (never writes)." It doesn't manifest today because metis writes all outputs under the excluded run-dir, and it's safe-direction regardless (a stray first-party write → spurious MISS, never a false HIT). Either mode-filter in `_audit` (skip `args[1]` write modes) or soften the doc to "opens; writes are excluded because they land under the run dir."
- **The sensor's own code is in every step's D** (`trace.py:66` snapshots `sys.modules`, which includes `metis.trace`/`metis.__init__`). Editing the sensor cold-busts the entire pool. Over-invalidation (safe), but worth a one-line acknowledgment.
- **`reads.json` defaults to `os.getcwd()`** when `METIS_STEP_DIR` is unset (`trace.py:77`). Fine in-runner (execStep always sets it), but a manual `python -m metis.trace …` writes into cwd — could dirty the tree.
- **`gitBlobHashes` batches all paths into one argv** (`trace.go:50`) — a very large D could exceed `ARG_MAX`. Fine for v1's handful of files; note for scale.

### 5. Test coverage notes
Coverage matches the M2 plan item-for-item (sensor records a known read, uv-gated; D-builder maps reads→D with a fake hasher + a real `git hash-object` skip-guarded; site-packages/stdlib excluded). Real-logic, not mock-reassertion. Gaps, in priority order: (1) the run-dir/write exclusion (Important, above); (2) no `gitBlobHashes` test for a vanished path (error propagation — relevant because it errors the *whole batch*, which M3's `Validate` will read as MISS for all of D); (3) `loadReadSet` parse-error path untested (low value). The `pkg/cache` M1 tests remain strong (5-term K_pre sensitivity pins each false-HIT vector).

### 6. Architectural notes for upcoming work (M3)
- **The uv.lock digest is a soundness prerequisite for going live, not a nicety.** `used_site_packages` is a *bool* — it cannot distinguish pandas 2.1 from 2.2. When M3 wires the cache, a dependency upgrade under an unchanged flag would serve a stale HIT. M3 must fold the uv.lock digest into `Code.Deps` and thence K_pre (or the D-hash) before the runner honors HITs. This is the one place M2's "capture the flag now, use it later" staging leaves a live hole to close first.
- **Run-dir exclusion is what keeps upstream artifacts out of D** (they're meant to enter K_pre via upstream output-hashes, per the design's class-1 vs class-2 split). M3's "re-run HITs every step" e2e is the real integration test of both the exclusion and sensor determinism — make that assertion count subprocess executions, not just outcomes.
- **`gitBlobHashes` fails the whole batch on one missing path.** Combined with `Validate`'s hasher-error→MISS, a single vanished D file → MISS (safe). But a *persistent* IO/permission fault on a D file would silently degrade every run to a cold MISS — as the M1 review already flagged, log (don't just swallow) such a fault when the real hasher is wired.
- **Feed K_pre from the freshly-parsed experiment** (carried over from M1's arch note) so `With` Go value types stay stable across runs — never reconstruct from a round-tripped `record.json`.

### 7. Plan revision recommendations
- **`workshop/plans/000002-step-caching-plan.md` — add a `## Revisions` entry:** "M2 (2026-07-05): shipped the read-sensor (`metis/trace.py`) + git blob-hasher (`cmd/metis/trace.go`: `loadReadSet`/`gitBlobHashes`/`buildD`); the step wrappers now launch through `metis.trace`. **`uvLockDigest` (M2 bullet) deferred to M3** — the sensor captures the `used_site_packages` signal now; the digest is computed where it's consumed (K_pre folding). NB for M3: fold the uv.lock digest before the runner honors any HIT — a bool flag can't invalidate a dependency-version change (soundness)."
- **Same file, M2 test list:** note the run-dir/write exclusion test as still-owed (or add it now) so the deferral is a recorded decision, not an omission.
