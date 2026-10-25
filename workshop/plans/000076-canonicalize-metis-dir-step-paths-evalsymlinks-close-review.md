# Boundary Review — metis#76 (whole-issue close)

| field | value |
|-------|-------|
| issue | 76 — Canonicalize METIS_*_DIR step paths (EvalSymlinks) |
| repo | metis |
| issue file | workshop/issues/000076-canonicalize-metis-dir-step-paths-evalsymlinks.md |
| boundary | whole-issue close |
| milestone | — |
| window | 27995863bb76020f322317132dc2c1df7783e24b..22ecea94cb4d54586e2653b0f8b7863d800ca2ac |
| command | sdlc close --issue 76 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | checkout |
| timestamp | 2026-10-25T08:22:16-07:00 |
| verdict | REWORK |

## Review

Diff reviewed; now checking sibling sites (blend.go, READ_ROOT) that may build the same env paths without canonicalization.

Found a likely regression in READ_ROOT confinement; confirming concretely and running the test suite.

Writing a scratch repro: a real uv step run through `runExperiment` via a symlinked dir, with readRoot set the way `runOuterFold` sets it (logical Abs).

GOWORK didn't persist across calls; rerunning with it set.

Regression confirmed (fails at head, passes at base). Running the full package suite to check the "stays green" claim.

```verdict
verdict: REWORK
confidence: high
```

The run.go fix does what the issue asked for. `runResolvedExperimentAdmitted` now calls `filepath.Abs`, then `filepath.EvalSymlinks` on the experiment dir, returns a wrapped error if that fails, and builds the run dir from the resolved experiment dir. The new `TestRun_StepEnvPathsArePhysical` drives the real `execStep` and passes. The problem is that this fix makes `METIS_EXP_DIR` physical while `METIS_READ_ROOT` is still built from the logical path in `sweep.go:743`, which uses `filepath.Abs` only. `metis/io.py:within_root` compares the two with `os.path.abspath`. So under a symlinked experiment dir (exactly the `/tmp` / macOS TMPDIR case this issue targets), every read a sealed nested-CV step makes of its own analysis root now gets refused as "outside METIS_READ_ROOT". I reproduced this: a scratch test failed at head and passed with base's `run.go` (details under Test coverage). There is also a second place that hands these env vars to a child: `blend.go:110-114`, which still uses `Abs` only. That contradicts the new atlas line saying the paths are always physical.

1. **Strengths**
   - `cmd/metis/run.go:247-254`: canonicalizes once at the runner boundary, never resolves the leaf run dir (which may not exist yet), and fails loudly with a wrapped error instead of falling back silently, as the Spec asks.
   - `cmd/metis/exec_test.go:262-298`: the test goes through `runExperiment` and the real `execStep` / `test/env-dump`, not a fake. It checks all three env vars with exact `+"\n"` matches.
   - Cache dir, record and `run.json` all come from the one resolved experiment dir, so the cache stays consistent with what steps see.

2. **Critical**
   - **`cmd/metis/sweep.go:743` + `metis/io.py:34-38` (made reachable by `cmd/metis/run.go:251`)**: the read root is logical while the experiment dir is now physical, so confinement wrongly refuses in-root reads.
     - Repro: an experiment reached through a symlink, `readRoot = Abs(Dir(expPath)/analysis_0)` (the same derivation as `runOuterFold`), and a real `metis/cv-split` reading `analysis_0`.
     - Result at head: `RuntimeError: read confinement (metis#23): '/…/real/experiment/analysis_0' is outside METIS_READ_ROOT '/…/link/experiment/analysis_0'`. At base: `ok`.
     - Fix sketch: canonicalize every path the runner puts into a child env in one shared helper (for example `physicalDir(p) (string, error)`). Use it for the read root (`analysisAbs`) as well as the experiment dir. Optionally also make `within_root` use `realpath` on both sides as a second line of defence.
     - Add a regression test: nested-CV or `execStep` with `readRoot`, run through a symlinked dir.

3. **Important**
   - **`cmd/metis/blend.go:110-114,204`**: `metis blend` builds its experiment dir and runs dir with `filepath.Abs` only, then passes them to `execStep` as `METIS_EXP_DIR` / `METIS_RUN_DIR`. A blend under `/tmp` still hands non-physical paths to the submission step, so the couch-bench bug still exists on this path. The atlas line ("The three dir paths are physical") is now false here. This is ARCH-DRY / ARCH-PURPOSE: the same per-path Abs/EvalSymlinks logic should live in one shared helper that every place building child env paths uses.

4. **Minor**
   - The run dir is built under the resolved experiment dir, but if `<expDir>/runs` is itself a symlink, `METIS_RUN_DIR` / `METIS_STEP_DIR` are still not physical. Resolving the deepest existing ancestor would cover this, if that layout matters.
   - The atlas note could name `METIS_READ_ROOT` once it is canonicalized too.

5. **Test coverage notes**
   - The new test pins the env-var contract for the single-run path. No test combines a symlinked dir with a non-empty `readRoot`, which is exactly the bug this diff introduces.
   - I ran `go test ./cmd/metis ./pkg/...` using a temporary go.work that points at the pinned ariadne checkout. All `pkg/*` packages pass. In `cmd/metis`, all 14 failures come from the review sandbox refusing `git init` ("outside the reviewer's directories"). They are environment failures, not code failures.
   - Reproduction: a scratch test (since removed) failed at head and passed with base's `run.go`. The checkout is clean again.

6. **Architectural notes**
   - ARCH-DRY: flag (blend.go repeats the Abs-only path logic; no shared canonicalization helper).
   - ARCH-PURE: pass (the change is in the thin IO shell).
   - ARCH-PURPOSE: flag ("every path handed to a child env is physical" is not met for `METIS_READ_ROOT` or for the blend path).
   - ARCH-MOCK: pass (uses real steps, no new external dependency).
   - ARCH-CONSTRAINTS: pass (one EvalSymlinks per run).
   - ARCH-SECURE: flag. Mixing physical and logical forms breaks a security-relevant confinement check. Here it refuses too much; the general rule is that a path-confinement check must compare like with like.
   - ARCH-ORDER: pass (no state is carried between events because this is a single synchronous path computation).
   - ARCH-FUNERAL: pass (nothing durable is created because only the location of existing artifacts changes).
   - ARCH-EVOLVE: pass (no format change; records may now hold physical paths, which is additive).

7. **Plan revision recommendations**
   - Add a `## Revisions` entry that widens the scope to every place that builds child env paths: `run.go` experiment/run dir, `sweep.go` read root (`analysisAbs`), and `blend.go` experiment/runs dir. Use one shared canonicalization helper.
   - Add a Done-when item: a test of a sealed (read-root) read through a symlinked experiment dir.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: child-env-paths-canonical-form
    title: |
      Physical METIS_EXP_DIR vs logical METIS_READ_ROOT breaks nested-CV read confinement under symlinked dirs
    detail: |
      run.go:251 now EvalSymlinks expDir, but sweep.go:743 builds readRoot (analysisAbs) with filepath.Abs only, and metis/io.py within_root compares with os.path.abspath. Under /tmp or macOS TMPDIR every sealed inner-CV base-dataset read raises "read confinement ... outside METIS_READ_ROOT". Reproduced: runExperiment through a symlink with readRoot=Abs(Dir(expPath)/analysis_0) and real metis/cv-split fails at head and passes with base run.go. Fix: canonicalize readRoot with the same helper (and optionally realpath in within_root); add a regression test.
  - id: new
    severity: Important
    kind: defect
    family: child-env-paths-canonical-form
    title: |
      blend.go still hands Abs-only METIS_EXP_DIR/METIS_RUN_DIR to the submission step
    detail: |
      blend.go:110-114 builds expDir/runsDir with filepath.Abs only and passes them to execStep (line 204), so metis blend under a symlinked dir still gives the child non-physical paths, contradicting the new atlas/experiment.md claim. Extract a shared canonicalize helper (ARCH-DRY/ARCH-PURPOSE) and use it at every place that builds child env paths.
  - id: new
    severity: Minor
    kind: defect
    family: child-env-paths-canonical-form
    title: |
      runDir is not physical if expDir/runs is itself a symlink
    detail: |
      runDir = Join(resolved expDir, "runs", runID) does not resolve an existing symlinked runs/ directory; resolving the deepest existing ancestor would cover it.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5a3d17d4-13a5-44c0-bee1-9d56eb3359e0","repository":"github.com/xianxu/metis","issue_id":"000076","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/e614388b52144281","reviewed_head":"22ecea94cb4d54586e2653b0f8b7863d800ca2ac","slot":"metis:1","requested_at":"2026-10-25T15:18:16.784037Z"},"phase":"finished","pid":51812,"started_at":"2026-10-25T15:18:16.947059Z","finished_at":"2026-10-25T15:22:14.086611Z","session_id":"d29e7622-e8c3-4be8-ad65-a7a9bd4516e5","outcome":"returned"}] -->

---

## Re-review — 2026-10-25T08:40:45-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 76 — Canonicalize METIS_*_DIR step paths (EvalSymlinks) |
| repo | metis |
| issue file | workshop/issues/000076-canonicalize-metis-dir-step-paths-evalsymlinks.md |
| boundary | whole-issue close |
| milestone | — |
| window | 27995863bb76020f322317132dc2c1df7783e24b..d357a3e0fcae7b3173c5f28a7f136a7696ac6528 |
| command | sdlc close --issue 76 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | checkout |
| timestamp | 2026-10-25T08:40:45-07:00 |
| verdict | SHIP |

## Review

Reviewing the diff; now checking trace.py's use of METIS_RUN_DIR for path-form mismatches, then running the tests.

Now running the Go tests, then reverting the fix to confirm the new tests fail without it.

Now the revert check: make `stepEnv` pass paths through unchanged (temporarily) and confirm the tests go red.

All three new tests pass at head; with the fix disabled, the two `execStep` tests fail (as expected, the run-level test stays green because run.go resolves on its own). Now the full suites.

Checking one remaining edge: whether an absolute dataset ref under a symlink could now be refused against the now-physical read root.

```verdict
verdict: SHIP
confidence: high
```

All three prior findings are fixed at the right level. Instead of patching each caller, `execStep.stepEnv` (`cmd/metis/exec.go:54-73`) now resolves symlinks in all four path vars (`METIS_STEP_DIR`, `METIS_RUN_DIR`, `METIS_EXP_DIR`, `METIS_READ_ROOT`) through the new `physicalPath` helper. `stepEnv` is the only place that builds a child's `METIS_*` env (I grepped the Go and Python code to confirm), and both executors use it: the legacy subprocess and the fork-server request. So run, blend and the sealed sweep all hand physical paths to children without each one doing its own conversion.

I checked the fix by breaking it. I changed `stepEnv` to pass paths through unresolved, and both new `execStep` tests failed: four "missing physical path" asserts, and the real `metis/cv-split` read was refused by confinement. Restoring the fix turned them green. `TestRun_StepEnvPathsArePhysical` stays green under that change, which is expected because run.go resolves the dir on its own.

The rest of the suites:
- **Python:** 140 tests pass.
- **Go `pkg/...` and `internal/...`:** pass.
- **`cmd/metis`:** 15 failures at head, none caused by this diff:
  - 14 are `git init refused … outside the reviewer's directories`, a limit of the review sandbox. Base shows the same 14.
  - The 15th is the timing test `TestForkServerPerf_LooseBound`, which passed when I re-ran it alone, so it is flaky under load.

Nothing blocks SHIP. Only Minor notes remain.

1. **Strengths**
   - The class is fixed where the paths meet (`exec.go:54-73`), not at each producer. This is the rule the family `child-env-paths-canonical-form` called for, and the new lessons entry states it.
   - `physicalPath` (`cmd/metis/physpath.go:15-36`) resolves the deepest existing ancestor. That covers a read root that does not exist yet and a `runs/` that is itself a symlink. It fails loudly on a dangling symlink or a non-ENOENT error, as the Spec's "don't fall back silently" requires.
   - `TestExecStep_ConfinementAgreesAcrossPathForms` (`exec_test.go:349`) runs the real uv `cv-split` step with a physical exp dir and a logical read root. That reproduces the exact BR-1 failure end to end.
   - run.go derives `runDir` from the already-resolved `expDir` (`run.go:246-249`), so the record, cache and step env all agree on one path form.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `workshop/lessons.md:61-62` adds an unrelated metis#70 lesson ("a check that hides a tool's stderr…"). It is scope piggybacked onto #76.
   - `physicalPath`'s edge branches have no direct test: the filesystem-root fallback (`physpath.go:31-33`), a non-NotExist `Lstat` error such as ENOTDIR, and a dangling symlink. Only the "deepest ancestor" and "symlinked ancestor" branches are exercised, and only indirectly.

5. **Test coverage notes**
   - Children see physical paths through a symlinked exp dir, a symlinked `runs/`, and a read root that does not exist yet. The run path, the generic `execStep` path (which blend and the sweep both reach) and the real confinement check are each covered.
   - The sweep → `readRoot` wiring itself is not tested end to end through `runOuterFold` under a symlink. That is acceptable: every sealed point goes through the same `stepEnv`, which is now tested directly.

6. **Architectural notes**
   - **ARCH-DRY: pass.** There is one helper, used by run.go and `stepEnv`. Pre-existing code outside the diff, `capture.go:277` and `capture.go:313`, does its own inline `EvalSymlinks` with a silent fallback. It could move to `physicalPath` later, keeping best-effort semantics.
   - **ARCH-PURE: pass, with a note.** `stepEnv` used to be a pure map builder and now does filesystem IO. That is acceptable because it sits in the IO shell (`Execute`) and is tested against real temp dirs.
   - **ARCH-PURPOSE: pass.** Shadow-sweep: `stepEnv` is the only producer of child path vars. The child cwd (`cmd.Dir` / fork-server `chdir`) stays logical, but `getcwd` returns the physical path anyway. Python's `exp_path` joins the relative `analysisRef` onto the physical `METIS_EXP_DIR`, which matches the physical `METIS_READ_ROOT`. `trace.py`'s run-dir prefix check now compares like with like, which is a small improvement.
   - **ARCH-MOCK: pass.** No new external dependency; the change only touches the local filesystem.
   - **ARCH-CONSTRAINTS: pass.** Four `EvalSymlinks` calls per step cost almost nothing next to a subprocess or fork.
   - **ARCH-SECURE: pass.** Confinement now compares a single path form. One gap is pre-existing: `within_root` uses `abspath`, so a symlink inside the read root that points outside it is not caught.
   - **ARCH-ORDER: pass.** Nothing holds state between events; the change is a per-call path transform.
   - **ARCH-FUNERAL: pass.** Nothing durable is created; only env-var string values change.
   - **ARCH-EVOLVE: pass.** No persisted format changes. `record.json` and `run.json` contents and the cache keys stay the same; only where `runs/` resolves on disk changes.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry, or extra checked Plan items, for the round-1 rework. The Plan still lists only "Resolve expDir in runResolvedExperimentAdmitted" and the symlink test. It should also record that `execStep.stepEnv` resolves every child path var through `physicalPath`, which covers blend, the sweep's `readRoot` and a symlinked `runs/`, along with the two new tests.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      stepEnv now resolves METIS_READ_ROOT via physicalPath (exec.go:54-73); TestExecStep_ConfinementAgreesAcrossPathForms (real uv cv-split) fails with the resolution disabled and passes at head.
  - id: BR-2
    disposition: addressed
    note: |
      blend calls es.Execute, which goes through stepEnv, so its Abs-only expDir/runDir are made physical there; TestExecStep_StepEnvPathsArePhysical (expDir passed as a symlink) fails without the fix.
  - id: BR-3
    disposition: addressed
    note: |
      physicalPath resolves the deepest existing ancestor; the test symlinks runs/ and asserts physical RUN_DIR/STEP_DIR/READ_ROOT, red without the fix.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: issue-scope-hygiene
    title: |
      lessons.md gains an unrelated metis#70 lesson inside the metis#76 commit
    detail: |
      workshop/lessons.md:61-62 adds a lesson about a check that hides a tool's stderr (metis#70) in commit d357a3e. Move it to the metis#70 work, or say in the Log why it rides along here.
  - id: new
    severity: Minor
    kind: defect
    family: helper-edge-branches-tested
    title: |
      physicalPath's root-fallback, non-ENOENT Lstat and dangling-symlink branches have no direct test
    detail: |
      cmd/metis/physpath.go:22-33. Add a small table test over temp dirs: a fully nonexistent path, a path whose ancestor is a regular file (ENOTDIR gives an error), and a dangling symlink (gives an error), so a later edit cannot quietly start falling back.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c0a74e83-49f3-43f3-a8ac-5869f822bc4e","repository":"github.com/xianxu/metis","issue_id":"000076","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/34f53ec0f9221ad3","reviewed_head":"d357a3e0fcae7b3173c5f28a7f136a7696ac6528","slot":"metis:1","requested_at":"2026-10-25T15:32:54.975358Z"},"phase":"finished","pid":57030,"started_at":"2026-10-25T15:32:55.09026Z","finished_at":"2026-10-25T15:40:33.650768Z","session_id":"0bcf94c3-fb51-4b1a-9aa9-b1677cd5e3f4","outcome":"returned"}] -->
