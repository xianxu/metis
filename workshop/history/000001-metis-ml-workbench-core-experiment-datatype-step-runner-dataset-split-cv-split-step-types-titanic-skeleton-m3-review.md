# Boundary Review — metis#1 (milestone M3)

| field | value |
|-------|-------|
| issue | 1 — metis ML-workbench core: experiment datatype + step-runner + Dataset/Split/cv-split + step-types (Titanic skeleton) |
| repo | metis |
| issue file | workshop/issues/000001-metis-ml-workbench-core-experiment-datatype-step-runner-dataset-split-cv-split-step-types-titanic-skeleton.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | a63f8c4ac6f0053f3c09041b9e2081b0f8b790a6..HEAD |
| command | sdlc milestone-close --issue 1 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-07-01T18:57:42-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
Both test suites pass green (Go incl. the non-cached e2e; 27 pytest), the working tree stays clean, wrappers are executable, and every Core-concepts entity exists at its stated path. Here's my review.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** M3 delivers exactly what the plan's Chunk 3 scoped: a pure Python numeric core (`schema`/`dataset`/`split`/`model`) with colocated IO-free pytests, a single thin `metis.io` encoding of the M2 step contract, three thin `io → pure → io` step entrypoints, hermetic uv wrappers, and a toy pipeline that runs end-to-end through the real Go-runner → subprocess → Python thread to a real CV score. I verified independently: `go test ./...` green (including `TestToyPipeline_EndToEnd`, uncached ~3.7s so it actually shelled uv), `uv run pytest` 27 passed, `git status` clean afterward. Atlas and plan are both updated; the CUE `#Run` field names match the Go struct's JSON tags so the new drift-guard is real. The three M2-deferred items (recursive `collectArtifacts`, `#Run` drift-guard, deduped go.mod walk) are genuinely cleared. Nothing blocks the boundary; the findings below are all future-notes.

### 1. Strengths
- **Clean pure/IO seam (ARCH-PURE).** The numeric core takes no IO — `cv_folds`/`train`/`predict`/`cv_score` operate on in-memory frames/arrays and are pytested directly (`tests/test_{schema,split,model,dataset}.py`), while `metis/io.py` and the `steps/metis/*` wrappers are the only filesystem/subprocess touch. `cv_score` (`metis/model.py:41`) even converts to numpy so per-fold models are name-free and reproducible.
- **The M2-deferred `collectArtifacts` fix is correct and well-tested.** The recursion excludes reserved channels *only* at the step-dir top level (`cmd/metis/exec.go:130`), and `TestCollectArtifacts_RecursiveExcludesReserved` pins the exact subtlety (a nested `sub/metrics.json` is a real artifact) with a sorted-deterministic assertion.
- **Reproducibility handled as a runner concern, not per-step (ARCH-DRY).** `METIS_SEED`/`METIS_EXP_DIR` injected once by the runner (`cmd/metis/exec.go:63-64`) rather than duplicated into every `with`; `TestExecStep_InjectsEnv` asserts the full contract env via a no-uv fake.
- **The `#Run` drift-guard closes a real gap** (`pkg/experiment/conformance_test.go`): a record with no `type:` discriminator can't use `validate-instance`, so marshaling a Go `Run` and `cue vet -d '#Run'`-ing it against the closed schema is the right technique, with a skip guard for bare checkouts.
- **DRY consolidation of the go.mod walk** into `internal/repo.Root` genuinely removes three near-identical copies (`cmd/metis/main.go`, both `helpers_test.go`), with its own unit tests.

### 2. Critical findings
None.

### 3. Important findings
None.

### 4. Minor findings
- **Repeated step preamble across the three entrypoints (ARCH-DRY, marginal).** `ctx = io.step_context(); w = io.read_with(ctx); ds = io.load_dataset(io.exp_path(ctx, w["dataset"]))` is identical in `metis/steps/{cv_split,train,predict}.py`. A one-liner `io.load_input_dataset(ctx, w)` helper would collapse it; borderline, only 3 lines, divergence after is real.
- **Missing `with`-key errors are raw `KeyError`s** (`w["dataset"]`, `w["k"]`, `w["model"]`) whereas `io._require_env` gives friendly messages — inconsistent author-facing error quality. Non-blocking (the KeyError still names the key and exits non-zero, surfaced by the runner's combined output).
- **`stratify: true` with no target column silently falls back to plain KFold** (`metis/steps/cv_split.py:29`: `target_col()` → `None` → unstratified) rather than erroring — an author who asked to stratify gets non-stratified without notice.
- **Spec's "Adapter protocol" bullet is delivered only implicitly.** No formal `Adapter` Protocol/ABC exists; the "contract metis defines" is the `load_dataset` serialization format (schema.json + train/test tables). The plan scoped the real adapter to kbench#1 (a genuine separable consumer), so this is consistent with the approved plan — noting only that the Spec bullet reads as more than what shipped.

### 5. Test coverage notes
- The **predict fallback path is untested**: `frame = ds.test if ds.test is not None else ds.train` (`metis/steps/predict.py`) — the no-test branch, and the no-`id_col` branch (`if id_col is not None`), are never exercised (the toy always has both). Simple branches, low risk, but a regression here would ship silently.
- **No end-to-end reproducibility assertion** (two full runs → identical score/predictions) despite the Done-when claiming reproducibility. It's transitively covered by the component determinism tests (`test_deterministic`, `test_deterministic_under_seed`, `cv_score` s1==s2), so this is an observation, not a gap that should block.
- Pure/IO test placement is exactly right: PURE entities tested with no IO; `io` via tmp_path + a committed CSV fixture; steps via injected env + files (`test_steps.py`), mirroring the Go-side contract.

### 6. Architectural notes for upcoming work
- **The cross-language contract strings** (`METIS_STEP_DIR`, etc.) now live in Go (`exec.go`), Python (`io.py`), the `env-dump` fake, and atlas. That's inherent to the files+subprocess seam (no shared source without codegen) and correctly centralized *within* each language — but when kbench adds step-types, the atlas `## Surface (M3)` contract note is the thing to keep authoritative; consider it the canonical prose source.
- **Submission-column naming is hardcoded** (`out["prediction"]`, id from schema). kbench's Titanic thread will need `Survived`/`PassengerId`; that mapping belongs in kbench's adapter/post-step, not here — flagging so it's a conscious downstream design point, not a surprise.
- The newly-introduced `metis.io`/`Schema`/`Dataset` surface reads as stable for kbench to consume; the `Dataset` dir format (schema.json + parquet/csv) is the load-bearing contract to keep backward-compatible once instances exist.

### 7. Plan revision recommendations
None — the plan's Chunk 3 Core-concepts table matches the code (every row exists at its stated path with the stated PURE/INTEGRATION kind), and the two documented deviations (`METIS_EXP_DIR`/`METIS_SEED` contract additions; no extra `load` step) are already recorded in the issue Log. The plan still describes what the code delivers.
