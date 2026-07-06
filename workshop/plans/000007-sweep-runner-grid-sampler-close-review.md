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
