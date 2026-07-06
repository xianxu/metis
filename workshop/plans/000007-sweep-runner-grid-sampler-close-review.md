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
