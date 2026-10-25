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
