# Boundary Review — metis#1 (whole-issue close)

| field | value |
|-------|-------|
| issue | 1 — metis ML-workbench core: experiment datatype + step-runner + Dataset/Split/cv-split + step-types (Titanic skeleton) |
| repo | metis |
| issue file | workshop/issues/000001-metis-ml-workbench-core-experiment-datatype-step-runner-dataset-split-cv-split-step-types-titanic-skeleton.md |
| boundary | whole-issue close |
| milestone | — |
| window | ddb88b759c100143a6acf54380c179de24600ea0..HEAD |
| command | sdlc close --issue 1 |
| reviewer | claude |
| timestamp | 2026-07-01T19:03:49-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I've verified enough. Both suites green, drift guards genuinely execute (not skip), and the real CLI binary walks the toy pipeline end-to-end to `cv_score 0.9167` with a clean tree afterward. Here's the review.

```verdict
verdict: SHIP
confidence: high
```

This whole-issue boundary closes a coherent, well-layered delivery: a single-sourced CUE `experiment` schema, a pure Go step-runner behind an injected `StepExecutor`/`Clock` seam, and a pure Python numeric core wrapped by a thin `metis.io` contract encoding — the three milestones each already crossed their own review (M3: SHIP). I independently re-verified the load-bearing claims rather than trusting the Log: `go test ./...` green (e2e uncached, ~3s, so it actually shelled uv), `uv run pytest` 27 passed, both CUE drift guards **run** here (cue + vocabulary present) rather than silently skipping, and — the check the test harness bypasses — the real `metis` binary run against a **relative** `toy-pipeline.md` produced `run.json` with `cv_score 0.9166…`, all three artifacts, and an appended `## Runs` bullet, leaving the metis tree clean. Nothing blocks SHIP; all findings below are Minor future-notes, most already logged by the per-milestone reviews.

### 1. Strengths (confirmed-good ground)

- **The pure/IO seam is real, not asserted (ARCH-PURE).** `Runner.Run` orchestrates `Validate → TopoSort → execute → assemble` behind `StepExecutor` + injected `Clock` (`pkg/experiment/run.go:39-45`), unit-tested with a fake and zero subprocess; the Python core (`schema`/`dataset`/`split`/`model`) is pytested on in-memory frames with all IO isolated to `metis/io.py` + the wrappers. I confirmed the pure tests need no exec/net/fs.
- **Drift guards enforce, they don't decorate (ARCH-DRY).** I verified `cue` and `../ariadne/bin/vocabulary` are present and both `TestParse_ConformsToCUE` and `TestRunConformsToCUE` **PASS** (not SKIP). The `#Run` guard is genuinely strong: `#Run` is a *closed* CUE def and the test marshals a Run with every field populated, so a renamed/removed/extra Go field would fail `cue vet -d '#Run'`.
- **The end-to-end Done-when holds via the actual CLI**, not just `runExperiment`. I built and ran the binary from a temp workspace with a bare relative filename — the exact invocation the M2 REWORK bug broke — and it resolved steps, produced a real CV score, recorded the ledger, and appended `## Runs`. `main.go`/`cmdRun`/`stepPath()` (the seam every test bypasses) work.
- **Failure and rejection branches are tested, not just the happy path.** `TestRunExperiment_FailedStepStillWritesLedger` (status `failed` ledger + `## Runs` still written) and `TestRunExperiment_RejectsInvalidAtRunTime` (cyclic experiment rejected on read, no ledger, source untouched) both exist and pass.
- **The three M2-deferred items are genuinely cleared:** `collectArtifacts` recurses and excludes reserved channels top-level-only with an exact-set test pinning the nested-`metrics.json`-is-an-artifact subtlety (`cmd/metis/exec.go:118`), the `#Run` guard closes the ledger-drift gap, and `internal/repo.Root` removes the 3× go.mod walk with its own unit tests.
- **Merge-checks fail *closed*.** A missing `vocabulary` binary or unresolvable base makes `experiment-validate.sh`/`experiment-schema-selftest.sh` exit non-zero (loud), not silently pass — the exact silent-swallow the M1 review caught and fixed.

### 2. Critical findings
None.

### 3. Important findings
None.

### 4. Minor findings
- **Repeated step preamble across the three entrypoints (ARCH-DRY, marginal).** `ctx = io.step_context(); w = io.read_with(ctx); ds = io.load_dataset(io.exp_path(ctx, w["dataset"]))` is byte-identical in `metis/steps/{cv_split,train,predict}.py`. A one-line `io.load_input_dataset(ctx, w)` helper would collapse it; 3 lines × 3, so borderline.
- **Inconsistent author-facing errors.** Missing `with` keys surface as raw `KeyError` (`w["dataset"]`/`w["k"]`/`w["model"]`) while `io._require_env` gives friendly messages. The KeyError still names the key and exits non-zero (surfaced via the runner's combined output), so low-impact.
- **`stratify: true` with no target column silently falls back to plain KFold** (`metis/steps/cv_split.py:29` — `target_col()` → `None`), so an author who asked to stratify gets unstratified folds with no notice. Not hit by the toy (which has a target).
- **`load_dataset` ignores `schema.dtypes` on CSV read** (`metis/io.py:_read_table`) — pandas type inference is trusted; a CSV whose inferred dtype disagrees with the declared schema would load mismatched silently. Matches for the toy; note for real datasets.
- **`appendRunLog` appends at EOF and probes with `strings.Contains(body, "## Runs")`** (`cmd/metis/run.go`): a bullet lands under the wrong heading if an experiment ever has sections after `## Runs`. Fine for the move-1 "`## Runs` last" convention.

### 5. Test coverage notes
- Pure core coverage is exactly the shape the checklist wants (PURE-without-IO): `Validate` table + `errors.Join`-all + `TopoSort` order/cycle/dup-needs; `Runner.Run` order/assembly/failure via fake executor; Python schema/split/model/dataset determinism and edge cases.
- Two residual gaps (both already noted by the M3 review, both low-risk): the **predict fallback branches** — `frame = ds.test if ds.test is not None else ds.train` and the `id_col is None` path (`metis/steps/predict.py`) — are never exercised (the toy always has both); and there is **no automated two-run reproducibility assertion** despite the Log making "byte-identical on re-run" a headline claim. Reproducibility is transitively covered by the component determinism tests, and I observed the identical `cv_score` across the test and my independent CLI run, so it's an observation, not a blocker. A single `run-twice → identical run.json` e2e would make the stated property self-guarding.

### 6. Architectural notes for upcoming work (kbench#1 / M-future)
- **`$METIS_RUN_DIR/<upstreamStepID>/` is now the artifact-graph substitute.** Keep the atlas `## Surface (M3)` step-contract prose authoritative — it's the only cross-language single source for the `METIS_*` strings (which necessarily recur in Go, Python, the `env-dump` fake, and atlas; inherent to a files+subprocess seam with no shared codegen).
- **`stepPath()` walks from the runner's cwd to `go.mod`.** kbench experiment instances live outside the metis tree, so `metis run` from a competition workspace will only find `steps/` via `$METIS_STEP_PATH` — I confirmed the `METIS_STEP_PATH` path works. Define the layered precedence (metis + kaggle + kbench step dirs) before real cross-layer step-types land.
- **Submission column naming is hardcoded** (`out["prediction"]`, id from schema). Titanic needs `Survived`/`PassengerId`; that mapping belongs in kbench's adapter/post-step, so flag it as a conscious downstream design point.

### 7. Plan revision recommendations
None. I checked each Core-concepts row against the filesystem: every entity (`Schema`/`Dataset`/`cv_folds`/`train`/`predict`/`Experiment`/`Parse`/`Validate`/`TopoSort`/`Runner.Run`/`StepExecutor`/`execStep`/`metis.io`/step entrypoints/wrappers) exists at its stated path with the stated PURE/INTEGRATION kind, and the two documented deviations (`METIS_EXP_DIR`/`METIS_SEED` contract additions; no extra `load` step) are already recorded in the issue Log. The plan still describes what the code delivers.

**ARCH-PURPOSE note (not a plan revision):** the Spec's "Adapter protocol — the interface `raw → Dataset`" bullet ships only *implicitly* as the `load_dataset` serialization format (schema.json + train/test tables); there is no formal `Adapter` Protocol/ABC in metis. This is defensible for *this* close — the Done-when (metis run e2e → CV score; Dataset/Schema/Split + step-types with tests; CUE-validated frontmatter) is fully met, and the plan explicitly scopes the real adapter to kbench#1 as a separable consumer. It's a documented deferral of a separable extension, not the deferred point of the issue — but worth naming so the kbench#1 author knows the `raw → Dataset` contract they implement against is the *serialization format*, not a code interface.
