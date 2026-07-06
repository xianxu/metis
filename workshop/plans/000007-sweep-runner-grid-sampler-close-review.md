# Boundary Review — metis#7 (whole-issue close)

| field | value |
|-------|-------|
| issue | 7 — Sweep runner + grid sampler (propose_next / should_stop abstraction) |
| repo | metis |
| issue file | workshop/issues/000007-sweep-runner-grid-sampler.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4dd6f118b65fa5475aae2d684d21ab1ec9c4f46b..HEAD |
| command | sdlc close --issue 7 |
| reviewer | claude |
| timestamp | 2026-07-05T18:05:28-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. Build + vet clean, full suite green (`cmd/metis` 7.9s, all pkgs ok), identity-matching verified against the record path, and I've traced the failure/abort/stop semantics against the fixtures. Here is the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** metis#7 delivers its stated purpose end-to-end: `metis run` on a multi-point shape now sweeps — Ask→run→Tell over the shared cached runner, per-point content-addressed run dirs, a shape-run manifest, `--max-points`/`--dry-run`, cache reuse across points, and HEAD-sha detect-and-abort with the dirty-tree false-abort regression pinned. The pure/IO split is clean (`pkg/sweep` is genuinely pure; the IO lives in `cmd/metis/sweep.go`), the `runResolvedExperiment` extraction is the right ARCH-DRY move, and I verified the correctness-critical claim that the sweep's precomputed runID equals the record's internal `PointAddress` (same `resolvedWith`, same seed via the inlined `Experiment`, same repoSHAs construction). No Critical issues; nothing blocks the boundary. Three Important findings are worth addressing — a test that can't actually prove the "sweep continues" contract, a swallowed post-run IO error in the continue path, and a duplicated repoSHAs construction that is the exact drift the plan warned about — none of which are common-path correctness bugs.

**1. Strengths**
- **Identity matching is real, not asserted.** `runSweep` computes `runID = PointAddress(p.With, repoSHAs, sh.Seed)` (sweep.go:93) and I confirmed it equals `buildRecord`'s internal address: `Expand` populates `p.With[stepID]` for *every* step (shape.go:81), `shapePointToExperiment` copies those into `step.With`, and `buildRecord` rebuilds `resolvedWith[stepID]=step.With` — same content, same seed (`Shape` embeds `Experiment` inline), same repoSHAs. Run-dir name and `record.PointAddress` can't diverge on the common path. This is the load-bearing property and it holds.
- **ARCH-PURE, exemplary.** `pkg/sweep` imports only `pkg/shape`; ask/tell + stop predicates are pure over tell-history, unit-tested with zero IO/mocks (sweep_test.go). The run/record/cache IO is entirely in the `cmd/metis` shell. Textbook.
- **ARCH-DRY seam extraction.** `runResolvedExperiment` (run.go:132) is called by both the 1-point path and the sweep loop — the run/cache/record wiring lives in one place, exactly as the plan scoped.
- **The dirty-tree regression is the right test.** `TestSweep_DirtyTreeDoesNotFalseAbort` (constant `dirty=true`) pins the real-CLI bug the Log describes — freezing on HEAD-sha-only, not the whole-repo dirty flag. The lessons.md entry generalizes it well.
- **Detect-and-abort semantics correct.** Freeze `codeID` at start, re-probe per point, abort on sha drift before running the point → no manifest written for a mixed-code sweep (protects the one-code identity). Traced against `mutatingGitProbe` — aborts at 2/3 as claimed.

**2. Critical findings** — none.

**3. Important findings**

- **`cmd/metis/sweep_e2e_test.go` (`TestSweep_FailingPointRecordedAndContinues`)** — the test can't prove the "sweep continues past a failure" contract, because the failing point is *last*. `fail: {$any: [false, true]}` expands (in list order) to `[fail:false, fail:true]`, so the failure is the final Ask; a hypothetical "record the failure then stop the loop" bug would still produce a 2-row manifest and pass. This is precisely the class the repo's own metis#6 lesson names ("a guard test must exercise the dimension the bug lives in"). Fix: flip to `$any: [true, false]` (fail first) and assert the later `ok` point still ran — so a stop-after-failure regression drops it to 1 row.

- **`cmd/metis/sweep.go:101` — post-run IO/assembly errors are swallowed in the continue path.** The guard `if run.Started == "" && runErr != nil` aborts only on *never-started* (validation) errors. But `runResolvedExperiment` also returns errors from `writeRunJSON`/`assembleRecord`/`writeRecordJSON`/`appendRunLog` *after* the run started — those come back with `run.Started != ""` (often `run.Status == "ok"`), so the point is appended to the manifest as its run status and `runErr` is dropped with no log. A missing `record.json` then silently breaks #8's per-point aggregation. The 1-point path surfaces these same errors to a non-zero exit — the sweep path diverges. Fix: distinguish a step failure (`run.Status == "failed"` → record + continue, as designed) from a metis-internal error (`runErr != nil && run.Status != "failed"` → surface it, or at minimum log + mark the manifest row) so real persistence failures aren't recorded as `ok`.

- **ARCH-DRY: `cmd/metis/sweep.go:162` (`repoSHAsOf`) duplicates `cmd/metis/record.go:106-109`.** Both build `{repoName: sha}` with the `repoName != ""` guard. This is the *exact* runID↔record-address coupling the plan flagged as drift-risk ("so the runID can't drift from the record's internal address") — currently guarded only by two parallel copies + comments. Since both are in package `main`, have `buildRecord` call `repoSHAsOf(repoName, sha)` so the construction is single-sourced structurally and can't drift silently.

**4. Minor findings**
- `sweep.go:152` — `probeRepo` degrades *any* git error to `"","",false`; a transient mid-sweep git failure would then false-abort (`"code changed (realSha → )"`). Low likelihood; consider distinguishing "no repo" (empty, expected) from "probe errored" (don't treat as drift).
- The `run.Started == "" && runErr != nil` sweep-abort branch (infra/validation abort) has no sweep-level test.
- `TargetReached` is built + unit-tested (M1) but only `--max-points` is wired into the driver — consistent with the plan's decision #2, but it's currently dead in the CLI path; note for whoever wires `--target`.
- No project README exists, so the README gate is N/A; CLI surface is documented via flag help + atlas (both updated). Confirmed.

**5. Test coverage notes**
- e2e coverage is genuinely strong and integration-real (process-level `test/echo` fake, injected clock + gitProbe): N-runs+manifest, cache-reuse, max-points, dry-run, abort-on-drift, dirty-no-false-abort. The two gaps above (continuation-past-failure discrimination; infra-error abort) are the meaningful holes.
- `pkg/sweep` unit tests pin real logic (enumeration order, both stop directions, missing-metric skip, compose, empty grid, exhaustion-stays-done) — not mocks reasserting the impl.

**6. Architectural notes for upcoming work (#8)**
- `#8` aggregates "the manifest + each point's record.json" — the swallowed-IO finding directly threatens that handoff (a point recorded `ok` in the manifest with no `record.json`). Fixing #3-Important closes that gap at the source.
- `Ask() bool` still can't distinguish exhaustion from early-stop (carried from M1). Before the manifest/ledger calcify on it, decide whether a stop-*reason* is needed for the "stopped at k/N — budget hit vs code changed" UX.
- The manifest currently has no top-level status (partial/complete/aborted). On abort, no manifest is written at all, so a consumer sees orphan `runs/` dirs with no grouping — #8 may want an explicit aborted manifest.

**7. Plan revision recommendations**
- The plan (M2 bullet) claims "The `repoSHAs` come from the **same `gitProbe` read** the detect-and-abort already does — one read, shared." The code does *not* realize this: there's a setup probe (for repoSHAs), a per-point detect probe, and a per-point `assembleRecord` probe — three reads; they only *agree in value* on a stable sha. Add a `## Revisions` note: *"M2: repoSHAs and the detect-and-abort sha are separate gitProbe reads that agree by value on a stable HEAD, not one shared read; the identity guarantee rests on sha-stability, not read-sharing."* (No behavior change needed; the claim just over-states the implementation.)

---

## Re-review — 2026-07-05T18:14:10-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 7 — Sweep runner + grid sampler (propose_next / should_stop abstraction) |
| repo | metis |
| issue file | workshop/issues/000007-sweep-runner-grid-sampler.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4dd6f118b65fa5475aae2d684d21ab1ec9c4f46b..HEAD |
| command | sdlc close --issue 7 |
| reviewer | claude |
| timestamp | 2026-07-05T18:14:10-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. Build + vet clean, full suite green (`cmd/metis` 7.9s, `pkg/sweep` ok), and I've traced the load-bearing identity claim and the freshly-applied fixes against the actual code. Here is the fresh-context boundary review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** metis#7 delivers its stated purpose end-to-end: `metis run` on a multi-point shape now sweeps — Ask→run→Tell over the shared cached runner, content-addressed per-point run dirs, a shape-run manifest, `--max-points`/`--dry-run`, cache reuse across points, and HEAD-sha detect-and-abort with the dirty-tree false-abort regression pinned. I independently confirmed the load-bearing correctness property (the sweep's precomputed `runID` equals the record's internal `PointAddress`) and verified the three fixes applied in `e7cb523` (fail-first failure test, internal-error surfacing, DRY `repoSHAsOf`) are all correct. `pkg/sweep` is genuinely pure; the IO is confined to the `cmd/metis` shell. **No Critical issues; nothing blocks the boundary.** The one Important finding is that the just-applied "surface internal errors" fix — correctness-sensitive logic that changed *because a bug shipped* — has no test that would fail if it were reverted, which is exactly the regression-proofing the repo's own lessons.md demands. FIX-THEN-SHIP encodes: land that test, then ship (gate passes regardless).

**1. Strengths**
- **Identity match verified, not asserted.** `runSweep` computes `runID = PointAddress(p.With, repoSHAs, sh.Seed)` (`sweep.go:93`); `buildRecord` computes `addr = PointAddress(resolvedWith, repoSHAs, run.Seed)` (`record.go:109`). I traced all three inputs equal: `Expand` populates `p.With[stepID]` for every step, `shapePointToExperiment` copies those into `step.With` unmutated, `buildRecord` rebuilds `resolvedWith[stepID]=step.With`; `run.Seed = exp.Seed = sh.Seed` (`run.go:75`, `Shape` embeds `Experiment` inline); `repoSHAs` via the shared `repoSHAsOf`. The run-dir name and `record.json` point-address can't diverge on the common path. This is the property #8's ledger rests on, and it holds.
- **The applied fixes are correct.** (a) `TestSweep_FailingPointRecordedAndContinues` now fails *first* (`$any:[true,false]` → point 0 fails via `test/echo`'s `exit 1`, point 1 ok) and asserts the later ok point still ran — a "record-then-stop" bug now drops it to 1 row. (b) The error discrimination `runErr != nil && run.Status != "failed"` (`sweep.go:105`) is right: `run.Status` is exactly `"ok"`/`"failed"` (`run.go:77,84`), so a step failure continues while an ok-run-with-persistence-error surfaces. (c) `repoSHAsOf` (`sweep.go:166`) is now called by both `buildRecord` (`record.go:108`) and `runSweep` (`sweep.go:45`) — the runID↔record-address drift the plan flagged is structurally eliminated.
- **ARCH-PURE, exemplary.** `pkg/sweep` imports only `pkg/shape`; ask/tell + stop predicates are pure over tell-history, unit-tested with zero IO/mocks. Textbook pure core / thin IO shell.
- **ARCH-DRY seam extraction.** `runResolvedExperiment` (`run.go:132`) is called by both the 1-point path and the sweep loop — run/cache/record wiring lives in one place.
- **Detect-and-abort + dirty-tree regression.** `TestSweep_DirtyTreeDoesNotFalseAbort` (constant `dirty=true`) pins the real-CLI bug the Log describes; the lessons.md entry generalizes it well. Traced against `mutatingGitProbe` — aborts at 2/3 as claimed.
- **Docs gates satisfied.** atlas/index.md accurately describes the new surface (sweeps, manifest, `--max-points`/`--dry-run`, detect-and-abort + dirty caveat). No user-facing README exists (only `.pytest_cache/README.md`), so the README gate is genuinely N/A.

**2. Critical findings** — none.

**3. Important findings**

- **`cmd/metis/sweep.go:105` — the freshly-fixed internal-error path has no test (regression risk on correctness logic that shipped a bug).** The discrimination `if runErr != nil && run.Status != "failed"` was added to fix the prior review's swallowed-error finding, but no test exercises the surface-the-error branch: `TestSweep_FailingPointRecordedAndContinues` only covers the `status=="failed"→continue` side, and `TestSweep_DetectAndAbort` aborts *before* `runResolvedExperiment` runs. A revert to the old unconditional `run.Started=="" && runErr!=nil` swallowing would pass the entire suite green. This is precisely the "regression-proof it: revert and confirm the test FAILS" discipline lessons.md records for #6/#7. Fix: extract the classification into a pure predicate (e.g. `isPointOutcome(run experiment.Run, runErr error) bool` → step-failure is a per-point outcome; ok-run-with-error / never-started is sweep-fatal) and unit-test it directly — that both pins the behavior and moves the decision out of the IO shell (ARCH-PURE). An e2e injecting a persistence failure works too but is fiddlier to wire.

**4. Minor findings**
- `sweep.go:105` (double-fault edge) — if a step *fails* (`status=="failed"`) AND a persistence write (`writeRunJSON`/`writeRecordJSON`) *also* errors, the persistence error is swallowed (the row records `"failed"`, sweep continues) because the guard keys on `status!="failed"`. Narrow (two simultaneous faults) and #8 must tolerate a failed row with a partial record anyway; note only.
- `sweep.go:89`/`sweep.go:157` — `probeRepo` degrades *any* git error to `"","",false`, so a transient mid-sweep git failure (when the sweep started in a real repo, `codeID!=""`) yields `s=="" != codeID` → false-abort with `"code changed (sha → )"`. Low likelihood; consider distinguishing "no repo" (expected, empty) from "probe errored" (don't treat as drift).
- `sweep.go:58` — `--dry-run` lists the full grid and ignores `--max-points` (the dry-run branch precedes the stop-predicate construction). `--dry-run --max-points 2` on a 4-point grid previews 4, not the 2 that would run. Defensible ("here's the whole grid") but mildly surprising when combined.
- `sweep.go:90` (abort) and `sweep.go:105` (surfaced error) both `return` before `writeManifest` (`sweep.go:117`), so a partially-run sweep leaves orphan `runs/` dirs with no manifest grouping them. Architectural note for #8 below.
- `TargetReached`/`AnyStop` (`pkg/sweep/sweep.go:81,101`) are built + unit-tested but have no production caller — only `--max-points` is wired (`sweep.go:67`). Consistent with the plan's decision #2 (`--target` deferred/opportunistic), but currently dead in the CLI path; note for whoever wires `--target`.

**5. Test coverage notes**
- e2e coverage is genuinely strong and integration-real (process-level `test/echo` fake, injected clock + gitProbe): N-runs+manifest, cache-reuse-across-points, failure-continues, max-points, dry-run, abort-on-drift, dirty-no-false-abort. `pkg/sweep` unit tests pin real logic (enumeration order, both stop directions, missing-metric skip, compose, empty grid, exhaustion-stays-done) — not mocks reasserting the impl.
- The two holes: (a) the internal-error-surfacing branch (Important #3 above); (b) the never-started/validation-abort branch is unreachable in a sweep in practice — `ValidateShape` validates the shared DAG structure upfront, and all points share that structure — so the `run.Started==""` case can't fire per-point. The comment at `sweep.go:100-104` mentions "validation never-started" as if reachable; it's effectively dead for sweeps (harmless, but the real value of the guard is the persistence-error case).

**6. Architectural notes for upcoming work (#8)**
- The swallowed-double-fault edge and the abort-loses-manifest behavior both feed #8's "manifest + each point's record.json" aggregation: a point can be recorded in the manifest with an absent/partial `record.json`, and a partial sweep produces orphan `runs/` with no manifest at all. #8 should either tolerate this or #7 should write a partial/aborted manifest before returning.
- `shapeRunIdentity` folds `--max-points` into the identity (`sweep.go:127`), so a capped and a full sweep of the same shape write to different `sweeps/<id>/` dirs while sharing (cache-hit) point-run dirs. Consistent with the design's "sampler-config" term, but it means one point-run can belong to multiple shape-runs — #8's ledger stamping must map a point-address to potentially >1 shape-run-id.
- `Ask() (Point, bool)` still can't distinguish exhaustion from early-stop, and the manifest has no top-level status (complete/capped/aborted). Before #8's ledger calcifies on the manifest shape, decide whether a stop-*reason* / manifest-status is needed for the "stopped at k/N — budget vs code-changed" UX.

**7. Plan revision recommendations**
- `workshop/plans/000007-sweep-runner-plan.md` (M2 bullet) claims the `repoSHAs` "come from the **same `gitProbe` read** the detect-and-abort already does — one read, shared." The code does *not* share one read: there's a setup probe in `runSweep` (`sweep.go:44`), a per-point detect probe (`sweep.go:89`), and `assembleRecord`'s own probe (`record.go:65`) — three reads that only *agree in value* on a stable HEAD. Add a `## Revisions` entry: *"M2: repoSHAs and the detect-and-abort sha are separate `gitProbe` reads that agree by value on a stable HEAD, not one shared read; the identity guarantee rests on sha-stability across the reads, not on read-sharing. (No behavior change; the claim over-stated the implementation.)"*

---

## Re-review — 2026-07-05T18:22:08-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 7 — Sweep runner + grid sampler (propose_next / should_stop abstraction) |
| repo | metis |
| issue file | workshop/issues/000007-sweep-runner-grid-sampler.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4dd6f118b65fa5475aae2d684d21ab1ec9c4f46b..HEAD |
| command | sdlc close --issue 7 |
| reviewer | claude |
| timestamp | 2026-07-05T18:22:08-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. Build + vet clean, full suite green (`cmd/metis` 8.0s, `pkg/sweep`/`pkg/shape` ok), and I traced the load-bearing identity property across the three files (`p.With` → `shapePointToExperiment` → executor read-only → `buildRecord.resolvedWith`) and verified the executor never mutates `step.With`. Here is the fresh-context boundary review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** metis#7 delivers its stated purpose end-to-end: `metis run` on a multi-point shape now sweeps — Ask→run→Tell over the shared cached runner (`runResolvedExperiment`), content-addressed per-point run dirs, a shape-run manifest, `--max-points`/`--dry-run`, cache reuse across points, per-point-failure-continues, and HEAD-sha detect-and-abort with the dirty-tree false-abort regression pinned. I independently confirmed the load-bearing correctness property (the sweep's precomputed `runID` equals the record's internal `point_address` on the common path) by tracing the map through all four links and verifying `execStep`/`cachingExecutor` are read-only over `step.With`. The three prior-review Important findings are genuinely fixed (fail-first failure test, internal-error surfacing via the extracted+unit-tested `isPointOutcome`, DRY `repoSHAsOf`). `pkg/sweep` is exemplary ARCH-PURE. **No Critical issues; nothing blocks the boundary.** The one Important is that the *identity the whole #8 handoff rests on* — `manifest.run_id == runs/<id>/record.json.point_address` — has no test asserting it, despite the plan's Done-when naming it and the repo's own "regression-proof it" discipline. FIX-THEN-SHIP: land that assertion, then ship (the gate passes regardless).

**1. Strengths**
- **Identity match verified, not asserted.** `runSweep` computes `runID = PointAddress(p.With, repoSHAs, sh.Seed)` (`sweep.go:93`); `buildRecord` computes `PointAddress(resolvedWith, repoSHAs, run.Seed)` (`record.go:109`). I traced all three inputs equal on the OK path: `Expand` populates `p.With[stepID]` for every step (`shape.go:81`); `shapePointToExperiment` copies each into `step.With` (`run.go:71`); `execStep`/`cachingExecutor` only *read* `step.With` (`exec.go:44`, `caching.go:127,292`) — no mutation; `buildRecord` rebuilds `resolvedWith[id]=sr.Step.With` (`record.go:92`); seed flows `sh.Seed → exp.Seed → run.Seed`; repoSHAs via the shared `repoSHAsOf`. The run-dir name and record point-address can't diverge on the common path.
- **The prior fixes are correct.** `isPointOutcome` (`sweep.go:128`) is now a pure classifier with a 4-case unit test (`TestIsPointOutcome_Classification`) that fails on a revert to the old `run.Started=="" && runErr!=nil` swallow — exactly the regression-proofing lessons.md demands. `repoSHAsOf` (`sweep.go:177`) is called by both `buildRecord` and `runSweep`, structurally eliminating the runID↔record drift. The failure test now fails *first* (`$any:[true,false]`) and asserts the later ok point ran.
- **ARCH-PURE, textbook.** `pkg/sweep` imports only `pkg/shape`; ask/tell + stop predicates pure over tell-history, unit-tested with zero IO. Extracting the continue-vs-abort decision into `isPointOutcome` moved logic *out* of the IO shell — the right instinct.
- **ARCH-DRY seam.** `runResolvedExperiment` (`run.go:132`) is the single per-point runner both paths call; the diff correctly deletes the old monolith/`resolveExperiment` rather than copy-pasting.
- **Detect-and-abort + dirty-tree regression.** Freeze `codeID` at start, re-probe per point, abort before running on sha drift → no manifest for a mixed-code sweep. `TestSweep_DirtyTreeDoesNotFalseAbort` (constant `dirty=true`) pins the real-CLI bug the Log describes; the lessons.md entry generalizes it well. I traced `mutatingGitProbe` — aborts at 2/3 as claimed.
- **Docs gates satisfied.** `atlas/index.md` accurately describes the new surface (sweeps dir, manifest, `--max-points`/`--dry-run`, detect-and-abort + dirty caveat). No user-facing README exists (only `.venv`/`.pytest_cache`), so the README gate is genuinely N/A.

**2. Critical findings** — none.

**3. Important findings**

- **`cmd/metis/sweep_e2e_test.go` — the load-bearing `run_id == record.point_address` identity is untested.** The plan's Done-when states "Run-id = the point's `record.PointAddress`" and the manifest→record.json correspondence *is* the #8 handoff, but no sweep test reads `runs/<run_id>/record.json`: `TestSweep_RunsAllPointsAndWritesManifest` checks only manifest content + run-dir *count*. The equality holds by a 4-link chain across 3 files (`Expand`→`shapePointToExperiment`→read-only executor→`buildRecord`); a future normalization added to either call-site would silently desync the manifest from the records #8 aggregates, and the entire suite would stay green. This is exactly the "regression-proof it: revert and confirm the test FAILS" class the repo records for #6/#7. Fix (cheap): in `TestSweep_RunsAllPointsAndWritesManifest`, for each `pr` in `man.Points`, read `runs/<pr.RunID>/record.json`, unmarshal, and assert `rec.PointAddress == pr.RunID`. That pins the identity #8 depends on without any production change.

**4. Minor findings**
- `sweep.go:98`/`record.go:92` — on a **failed** point, `buildRecord`'s `resolvedWith` contains only the *pre-failure* steps (`Runner.Run` appends a `StepRun` only on success, `run.go:81-88`), so a failed point's `record.json.point_address` is computed from a partial config and diverges from its run-dir name / `manifest.run_id` (both full-`p.With`-derived). Latent #8 data-quality wrinkle for failed points; dedup/resume still holds (runID is full-`p.With`-derived regardless of failure). Note for #8.
- `shape_e2e_test.go:87` — the "superseded by … in `caching_sweep_test.go`" comment names a file that doesn't exist; the superseding tests live in `sweep_e2e_test.go`. Fix the filename.
- `sweep.go:44-56` — `probeRepo`/`repoSHAsOf`/`codeID` are computed *before* the `dryRun` early-return (`sweep.go:58`), so a `--dry-run` does unused git work; and `--dry-run` ignores `--max-points` (previews the full grid). Both harmless/defensible.
- `sweep.go:164` — `probeRepo` degrades *any* git error to `"","",false`, so a transient mid-sweep git failure (when the sweep started in a real repo, `codeID!=""`) yields `s=="" != codeID` → false-abort `"code changed (sha → )"`. Low likelihood; consider distinguishing "no repo" (expected) from "probe errored".
- Mild ARCH-DRY: two "probe git, degrade to empty provenance" impls (`probeRepo` at `sweep.go:164` and the inline `git==nil`+err→empty in `assembleRecord` at `record.go:62-69`). Not trivially unifiable (`assembleRecord` warns on the error + returns `dirty`), so note only.
- `pkg/sweep/sweep.go:70,101` — `TargetReached`/`AnyStop` are built + unit-tested but have no production caller (only `--max-points` wired at `sweep.go:67`). Consistent with the plan's decision #2 (`--target` deferred/opportunistic); dead in the CLI path until wired — note for whoever adds `--target`.

**5. Test coverage notes**
- e2e coverage is strong and integration-real (process-level `test/echo`, injected clock + gitProbe): N-runs+manifest, cache-reuse-across-points, failure-continues (now fail-first), max-points, dry-run, abort-on-drift, dirty-no-false-abort. `pkg/sweep` unit tests pin real logic (enumeration order, both stop directions, missing-metric skip, compose, empty grid, exhaustion-stays-done).
- `isPointOutcome` is now unit-tested directly — the prior review's gap is closed. The internal-error-surfacing *wiring* has no injected-failure e2e, but the pure classifier carries the load-bearing logic, so that's acceptable.
- The one real hole is Important #3 above: the manifest↔record point-address identity has no assertion.

**6. Architectural notes for upcoming work (#8)**
- The failed-point partial-`resolvedWith` divergence (Minor #1) and the abort-loses-manifest behavior both feed #8's "manifest + each point's record.json" aggregation: a failed point's `record.json.point_address` won't equal its `manifest.run_id`, and a mid-sweep abort leaves orphan `runs/` dirs with no manifest. #8 should key point identity off the *manifest* (self-consistent), and either #8 tolerates the partial record or #7 later writes a partial/aborted manifest.
- `shapeRunIdentity` folds `--max-points` into the identity (`sweep.go:138-147`), so a capped and a full sweep of the same shape write different `sweeps/<id>/` dirs while sharing (cache-hit) point-run dirs — one point-run can belong to >1 shape-run. #8's ledger stamping must map a point-address to potentially multiple shape-run-ids.
- `Ask() (Point, bool)` still can't distinguish exhaustion from early-stop, and the manifest has no top-level status (complete/capped/aborted). Decide before #8's ledger calcifies on the manifest shape whether a stop-*reason* is needed for the "stopped at k/N — budget vs code-changed" UX.

**7. Plan revision recommendations**
- `workshop/plans/000007-sweep-runner-plan.md` (M2 bullet) still over-claims: *"The `repoSHAs` come from the **same `gitProbe` read** the detect-and-abort already does — one read, shared."* The code does **not** share one read — there's a setup probe (`sweep.go:44`), a per-point detect probe (`sweep.go:89`), and `assembleRecord`'s own probe (`record.go:65`): three reads that agree only on a *stable* HEAD (the drift test even exposes their divergence when the sha isn't stable). Add a `## Revisions` entry: *"M2: repoSHAs and the detect-and-abort sha are separate `gitProbe` reads that agree by value on a stable HEAD, not one shared read; the identity guarantee rests on sha-stability across the reads, not on read-sharing. (No behavior change; the claim over-stated the implementation.)"* This entry was recommended in both prior review rounds and has not yet been folded into the plan.

---

## Re-review — 2026-07-05T18:28:48-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 7 — Sweep runner + grid sampler (propose_next / should_stop abstraction) |
| repo | metis |
| issue file | workshop/issues/000007-sweep-runner-grid-sampler.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4dd6f118b65fa5475aae2d684d21ab1ec9c4f46b..HEAD |
| command | sdlc close --issue 7 |
| reviewer | claude |
| timestamp | 2026-07-05T18:28:48-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need — build+vet clean, full suite green, and I've independently verified the load-bearing identity claim *and* empirically found where it breaks. Here is the fresh-context boundary review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** metis#7 delivers its stated purpose end-to-end: `metis run` on a multi-point shape now sweeps — Ask→run→Tell over the shared cached runner (`runResolvedExperiment`), content-addressed per-point run dirs, a shape-run manifest, `--max-points`/`--dry-run`, cache reuse across points, per-point-failure-continues, and HEAD-sha detect-and-abort with the dirty-tree false-abort regression pinned. `pkg/sweep` is genuinely pure (imports only `pkg/shape`, unit-tested with zero IO); the IO is confined to the `cmd/metis` shell. I confirmed the three prior-round fixes are real (`isPointOutcome` extracted + unit-tested, `repoSHAsOf` shared, fail-first failure test) and that on the **OK path** `manifest.run_id == record.json.point_address` holds (executor is read-only over `step.With`, so `resolvedWith == p.With`). **No Critical issues; nothing blocks the boundary.** The one open Important is one the three prior reviews saw but under-weighted as "Minor / note for #8": the `run_id == record.PointAddress` invariant — which the plan states as Done-when and the code itself tests as the "load-bearing #8 handoff identity" — is **silently false for failed points**, and I empirically reproduced a *point-address collision* across distinct failed configs. FIX-THEN-SHIP: address it (or at minimum pin + document it), then ship — the gate passes regardless.

**1. Strengths**
- **Identity match verified on the OK path.** `runSweep` computes `runID = PointAddress(p.With, repoSHAs, sh.Seed)` (`sweep.go:93`); `buildRecord` computes `PointAddress(resolvedWith, repoSHAs, run.Seed)` (`record.go:109`). I traced every input equal for a clean run: `Expand` populates `p.With[stepID]` for every step (`shape.go:81`), `shapePointToExperiment` copies each into `step.With` (`run.go:70`), `execStep`/`cachingExecutor` only *read* `step.With` (`exec.go:44`, `caching.go:127`), `buildRecord` rebuilds `resolvedWith[id]=sr.Step.With` (`record.go:92`); seed flows `sh.Seed→exp.Seed→run.Seed`; repoSHAs via the shared `repoSHAsOf`. The property holds for ok points.
- **ARCH-PURE, exemplary.** `pkg/sweep` is deterministic, pure over tell-history, no clock/net/fs; the continue-vs-abort decision was extracted into the pure `isPointOutcome` (`sweep.go:128`) — logic moved *out* of the IO loop. Textbook.
- **ARCH-DRY seam.** `runResolvedExperiment` (`run.go:132`) is the single per-point runner both the 1-point path and the sweep loop call; the diff deletes the old `resolveExperiment` monolith rather than copy-pasting. `repoSHAsOf` is now shared by `buildRecord` and `runSweep`.
- **Prior fixes correct.** `isPointOutcome` has a 4-case unit test that fails on a revert to the old `run.Started=="" && runErr!=nil` swallow; the failure test now fails *first* (`$any:[true,false]`) and asserts the later ok point ran.
- **Detect-and-abort + dirty-tree regression.** Freeze `codeID` at start, re-probe per point, abort before running on sha drift → no manifest for a mixed-code sweep. `TestSweep_DirtyTreeDoesNotFalseAbort` (constant `dirty=true`) pins the real-CLI bug; the `lessons.md` entry generalizes it well.
- **Docs gates satisfied.** `atlas/index.md` accurately describes the new surface. No user-facing README exists, so that gate is genuinely N/A.

**2. Critical findings** — none.

**3. Important findings**

- **`cmd/metis/record.go:89-92,109` + `cmd/metis/sweep.go:93,106` — the `run_id == record.point_address` identity silently breaks for *failed* points, with a point-address collision.** `Runner.Run` stops at the first step error and returns only the *pre-failure* `StepRun`s (`pkg/experiment/run.go:81-88`), so for a failed point `buildRecord`'s `resolvedWith` is **partial** (empty if the *first* step fails) — and `PointAddress` is minted from that partial config. Meanwhile `manifest.run_id` / the run-dir name is minted from the **full** `p.With` in `runSweep`. So `record.json.point_address ≠ manifest.run_id` for every failed point. I reproduced it: two distinct failed configs (`model=xx` and `model=yy`, both failing at the only step) both produced `record.point_address = 03dec73a…` — **identical** (empty `resolvedWith` → same hash regardless of free-params) while their run-dirs differ. This violates the plan's Done-when ("Run-id = the point's `record.PointAddress`") and the exact "load-bearing … #8 handoff identity" that `TestSweep_RunsAllPointsAndWritesManifest` asserts — but that test only covers **ok** points (its shape has no `fail`), and `TestSweep_FailingPointRecordedAndContinues` never reads `record.point_address`, so the divergence + collision are entirely untested. This is the repo's own metis#6 lesson ("a guard test must exercise the dimension the bug lives in"). Not Critical — the happy path is correct, failed points still record + continue + dedup by dir name, and #8 isn't built — but it's a real contract drift that will bite #8's aggregation (if #8 keys off `record.point_address`, distinct failed rows collapse into one and silently drop). Fix, in order of thoroughness: (a) stamp the record's `point_address` from the point's **full intended** config (thread `p.With` / the precomputed address into `assembleRecord`) so the identity holds by construction for ok *and* failed points; or (b) minimally, decide #8 keys point-identity off `manifest.run_id` + the run-dir (self-consistent, full-config for all points) *not* `record.point_address`, **and add a failed-point test that pins the divergence** so it's documented, not silent.

**4. Minor findings**
- `cmd/metis/shape_e2e_test.go:87` — the "superseded … in `caching_sweep_test.go`" comment names a file that doesn't exist; the tests live in `sweep_e2e_test.go`. (Flagged in all three prior rounds; still unfixed.)
- `sweep.go:164` — `probeRepo` degrades *any* git error to `"","",false`, so a transient mid-sweep git failure (when the sweep started in a real repo, `codeID!=""`) yields `s=="" != codeID` → false-abort `"code changed (sha → )"`. Low likelihood; consider distinguishing "no repo" (expected) from "probe errored" (don't treat as drift).
- `sweep.go:44-64` — `--dry-run` computes `probeRepo`/`repoSHAs`/`codeID` (unused) before the early return, and lists the full grid ignoring `--max-points`. Both harmless/defensible.
- `sweep.go:99-105` — double-fault edge: if a step *fails* (`status=="failed"`) AND a persistence write also errors, `isPointOutcome` returns true (keys on `status`), so the persistence error is swallowed and the row records `"failed"` with a possibly-absent `record.json`. Two simultaneous faults; note only.
- `pkg/sweep/sweep.go:81,101` — `TargetReached`/`AnyStop` are built + unit-tested but have no production caller (only `--max-points` is wired at `sweep.go:67`). Consistent with the plan's decision #2 (`--target` deferred); dead in the CLI path until wired.
- Mild ARCH-DRY: two "probe git, degrade to empty provenance" impls (`probeRepo` at `sweep.go:164`, inline in `assembleRecord` at `record.go:62-69`). Not trivially unifiable (`assembleRecord` warns + returns `dirty`); note only.

**5. Test coverage notes**
- e2e coverage is strong and integration-real (process-level `test/echo`, injected clock + gitProbe): N-runs+manifest, cache-reuse-across-points, failure-continues (fail-first), max-points, dry-run, abort-on-drift, dirty-no-false-abort. `pkg/sweep` unit tests pin real logic (enumeration order, both stop directions, missing-metric skip, compose, empty grid, exhaustion-stays-done). `isPointOutcome` is unit-tested directly.
- The one real hole is Important #3: the manifest↔record point-address identity is asserted for ok points only; the failed-point divergence + collision has zero coverage.

**6. Architectural notes for upcoming work (#8)**
- **Key point-identity off the manifest/run-dir, not `record.point_address`.** Per Important #3, `record.point_address` is partial (and colliding) for failed points; `manifest.run_id` and the run-dir name are always full-config and self-consistent. #8's ledger should stamp from the manifest.
- **Abort leaves orphan `runs/` with no manifest.** `sweep.go:90` (abort) and `sweep.go:104` (surfaced internal error) both `return` before `writeManifest` (`sweep.go:115`), so a partial sweep produces ungrouped run dirs. #8 may want #7 to write a partial/aborted manifest.
- **`shapeRunIdentity` folds `--max-points` into the identity** (`sweep.go:146`), so a capped and a full sweep of the same shape write different `sweeps/<id>/` dirs while sharing (cache-hit) point-run dirs — one point-run can belong to >1 shape-run. #8's stamping must map a point-address to potentially multiple shape-run-ids.
- **`Ask() (Point, bool)` can't distinguish exhaustion from early-stop**, and the manifest has no top-level status (complete/capped/aborted). Decide before #8's ledger calcifies on the manifest shape whether a stop-*reason* is needed for the "stopped at k/N — budget vs code-changed" UX.

**7. Plan revision recommendations**
`workshop/plans/000007-sweep-runner-plan.md` has **no `## Revisions` section**, and two claims still over-state the code (AGENTS.md §1 requires appending a `## Revisions` entry, not leaving the over-claim):
- **(recommended in all three prior rounds, still not folded in)** M2 bullet / plan line 101 claims the `repoSHAs` "come from the **same `gitProbe` read** the detect-and-abort already does — one read, shared." The code does **not** share one read — there's a setup probe (`sweep.go:44`), a per-point detect probe (`sweep.go:89`), and `assembleRecord`'s own probe (`record.go:65`): three reads that agree only on a *stable* HEAD. Add: *"M2: repoSHAs and the detect-and-abort sha are separate `gitProbe` reads that agree by value on a stable HEAD, not one shared read; the identity guarantee rests on sha-stability, not read-sharing."*
- **(new)** Add a `## Revisions` entry documenting the failed-point address divergence: *"Done-when 'Run-id = the point's `record.PointAddress`' holds for OK points only — a failed point's `record.point_address` is minted from the partial executed config (`Runner.Run` returns only pre-failure steps), so it diverges from the full-`p.With` manifest run_id and collides across distinct first-step failures. The identity of record for a point is `manifest.run_id`/the run-dir; #8 keys off that, not `record.point_address`."*
