# Boundary Review — metis#1 (milestone M2)

| field | value |
|-------|-------|
| issue | 1 — metis ML-workbench core: experiment datatype + step-runner + Dataset/Split/cv-split + step-types (Titanic skeleton) |
| repo | metis |
| issue file | workshop/issues/000001-metis-ml-workbench-core-experiment-datatype-step-runner-dataset-split-cv-split-step-types-titanic-skeleton.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 3e1afc09da0ab3ae4e799f1d98e5da008a4833ca..HEAD |
| command | sdlc milestone-close --issue 1 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-01T15:24:04-07:00 |
| verdict | REWORK |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: REWORK
confidence: high
```

**Summary.** The pure core is genuinely well-built — `Parse`/`Validate`/`TopoSort`/`Runner.Run` are clean pure functions behind an injected `StepExecutor` and `Clock`, unit-tested with no subprocess, and the CUE↔Go drift guard actually runs and passes in this environment. But the milestone's headline deliverable — `metis run <experiment.md>` executing steps end-to-end — is **broken for the normal (relative-path) invocation**: I reproduced every step failing because `METIS_STEP_DIR`/`METIS_RUN_DIR` are injected as *relative* paths while the child's cwd is set to that same relative dir, so a step-type resolving `$METIS_STEP_DIR/with.json` double-joins and can't find its input. The e2e test is green only because it feeds an absolute `t.TempDir()` path, structurally masking the bug. That is a confirmed Critical in the core feature and blocks the boundary until fixed and re-run.

### 1. Strengths
- **ARCH-PURE seam is real, not claimed.** `pkg/experiment/run.go:38` `StepExecutor` + injected `Clock` let `Runner.Run` be exercised with a fake and no subprocess (`pkg/experiment/run_test.go`); all IO sits in `cmd/metis`. This is the milestone's best work.
- **Drift guard is enforced, not decorative.** `conformance_test.go` shells the real `vocabulary validate-instance` and I confirmed it executes and passes here (binary present + `cue` on PATH) — the Go structs genuinely can't silently diverge from `construct/vocabulary/experiment.cue` (ARCH-DRY).
- **`TopoSort` reused by `Validate` for acyclicity** (`validate.go:44`) — one implementation, and edges to unknown ids are deliberately ignored so a typo'd `needs` can't misfire cycle detection (`validate.go` doc). Good ARCH-DRY discipline.
- **Execution-time enforcement path is correct and tested.** Invalid-on-read rejection writes no ledger and no `## Runs` line (`run.go:64-71`, `run_test.go:TestRunExperiment_RejectsInvalidAtRunTime`) — I reproduced this working correctly.
- **`uses` regex validated before `resolve`**, so `filepath.FromSlash(uses)` can't path-traverse out of `stepPath` — defense in depth.

### 2. Critical findings
- **`cmd/metis/exec.go:53-57` (with root cause at `run.go:55-56`): relative experiment path breaks every step.** `runDir` derives from `filepath.Dir(o.expPath)` with no absolutization; `execStep` then sets `cmd.Dir = stepDir` **and** injects `METIS_STEP_DIR=stepDir` / `METIS_RUN_DIR=runDir` as relative strings. Because the child runs *in* `stepDir`, its `$METIS_STEP_DIR/with.json` resolves to `<stepDir>/<stepDir>/with.json`. Reproduced: `metis run --run run-xyz run-echo.md` → `cp: runs/run-xyz/first/with.json: No such file or directory`, run status `failed`. This hits the primary CLI usage (`metis run pipelines/foo.md`) and every M3 Python step-type that reads its inputs via the documented `$METIS_STEP_DIR` env var.
  - **Fix sketch:** absolutize once at the boundary — in `runExperiment` after computing `runDir` (`run.go:56`), `runDir, err = filepath.Abs(runDir)`; or in `execStep.Execute`, resolve `stepDir`/`runDir` with `filepath.Abs` before setting `cmd.Env`/`cmd.Dir`. An absolute path is correct from any cwd.
  - **Regression test:** add an e2e case that invokes with a **relative** `expPath` (chdir into the temp dir, pass the bare filename), and assert the step's declared output artifact (`echoed.json`) exists — the current absolute-path test cannot catch this class.

### 3. Important findings
- **`cmd/metis/exec.go:67` — `metrics.json` leaks into the `run.json` artifact list.** `collectArtifacts` excludes only `with.json` and directories, so `metrics.json` (the metrics channel, already parsed into `run.Metrics`) is recorded as a data artifact. This contradicts exec.go's own contract comment ("optional metrics.json **+** any artifact files"). The ledger is the surface M3/kbench derive from, so the contract should be clean. **Fix:** skip `metrics.json` alongside `with.json` in `collectArtifacts`.
- **Atlas update is partial (Atlas gate).** `atlas/experiment.md` gained a "Validation split" paragraph and the roadmap moved, but the new **runner/step-executable contract** — the load-bearing convention M3 step-types must implement (`with.json` in; `metrics.json`/artifacts out; `METIS_STEP_DIR`/`METIS_RUN_DIR`/`METIS_STEP_ID` env; `steps/<layer>/<steptype>` + `METIS_STEP_PATH` resolution; `runs/<id>/run.json` + `## Runs` ledger) — is documented **only** in code comments. The `## Surface (M1)` heading was left with no M2 surface section. Add an atlas entry mapping the step contract + `cmd/metis` surface so M3 derives from the atlas, not from reading `exec.go`.
- **Test coverage: the e2e test asserts only `len(Artifacts) != 0`** (`run_test.go:68`), never the artifact *set*, so it catches neither the relative-path bug nor the `metrics.json`-in-artifacts leak. Assert the exact artifact list (`["echoed.json"]`).

### 4. Minor findings
- `#Run`↔Go `Run` drift is **unguarded** — the conformance test validates an `Experiment` fixture but nothing validates a produced `run.json` against CUE `#Run`. Hard to guard with the current markdown-frontmatter validator; note for M3 when a `run` consumer lands.
- "walk up to `go.mod`" is implemented **3×**: `cmd/metis/main.go:findRepoRoot`, `cmd/metis/helpers_test.go:repoRoot`, `pkg/experiment/helpers_test.go:repoRoot` (the two test copies are byte-identical). ARCH-DRY nit — cross-package test-helper sharing in Go is awkward, but the two `repoRoot` copies could live in a small `internal/testutil`.
- `appendRunLog` appends the bullet at EOF and probes with `strings.Contains(body, "## Runs")` (`run.go:appendRunLog`) — if an experiment ever has sections *after* `## Runs`, the bullet lands under the wrong heading. Fine for the move-1 convention (`## Runs` last); note it.
- A `failed` `run.json` records only `status`, not the failure reason (the error goes to stderr only). Consider stamping the error into the ledger later.
- Wrapped step error prefixes the step id twice (`exec.go:60` "step %s (%s) failed" + `run.go:` "step %q: %w").

### 5. Test coverage notes
- Pure core is well covered: `Validate` table (cycle/dangling/bad-uses/dup) + `errors.Join`-all-violations + `TopoSort` order/cycle; `Runner.Run` order/assembly + step-failure-stops-pipeline with a fake executor (no IO). This is exactly the PURE-tested-without-IO shape the checklist wants.
- The **gap** is at the IO boundary: the one e2e test uses an absolute path and loose assertions, so it validates the happy path but masks the Critical relative-path defect and the `metrics.json` leak. Both need a targeted test (relative-path invocation + exact-artifact-set assertion).
- The drift guard correctly `t.Skip`s when the toolchain is absent (verified it *runs* here), keeping bare checkouts green — good.

### 6. Architectural notes for upcoming work
- **`METIS_RUN_DIR` semantics for M3.** Once absolutized, `$METIS_RUN_DIR/<otherStepID>` becomes the documented way a step reaches an upstream step's outputs (the move-1 substitute for an artifact graph). Worth stating explicitly in the atlas step-contract so M3 `train`/`predict` steps consume `cv-split`'s folds through a stable path.
- **`findRepoRoot`/`stepPath` for downstream instances.** `stepPath()` walks up from the *runner's* cwd to `go.mod`; kbench experiment instances live outside the metis tree, so `metis run` from a competition workspace will not find `steps/` via `go.mod` and must rely on `$METIS_STEP_PATH`. Fine for now, but the layered step-resolution (metis + kaggle + kbench step dirs) will need a defined precedence — note it before M3 wires real `metis/*` steps.

### 7. Plan revision recommendations
- Add to `workshop/plans/000001-experiment-datatype-plan.md` a `## Revisions` entry once the Critical is fixed, e.g.:
  > **2026-07-01 — M2 boundary review (REWORK → fixed).** The subprocess executor injected `METIS_STEP_DIR`/`METIS_RUN_DIR` as relative paths while setting the child cwd to the same relative dir, so every step failed under the natural `metis run <relative-path>` invocation (the e2e test masked it with an absolute `t.TempDir()` path). Absolutized `runDir` at the runner boundary; added a relative-path e2e regression asserting the step's output artifact exists. Also excluded `metrics.json` from the `run.json` artifact list (it is the metrics channel, not an artifact) and mapped the step-executable contract into `atlas/`.
- The Chunk-2 "Integration points" table (`plan.md:296-304`) is otherwise faithful to the code (`StepExecutor`, `execStep`, run-ledger writer all at the stated paths) — no other table/code contradiction found.

---

## Re-review — 2026-07-01T16:19:17-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 1 — metis ML-workbench core: experiment datatype + step-runner + Dataset/Split/cv-split + step-types (Titanic skeleton) |
| repo | metis |
| issue file | workshop/issues/000001-metis-ml-workbench-core-experiment-datatype-step-runner-dataset-split-cv-split-step-types-titanic-skeleton.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 3e1afc09da0ab3ae4e799f1d98e5da008a4833ca..HEAD |
| command | sdlc milestone-close --issue 1 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-01T16:19:17-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
Confirmed — `TopoSort` reports `cycle in step dependencies among: [b]` for an **acyclic** graph whose only quirk is a duplicated `needs` edge (`needs: [a, a]`). Cleanup done: `git status` shows only the untracked review sidecar, my temp file is gone. I now have everything I need.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** This window is a re-review after a prior REWORK cycle, and the earlier blocker plus its three companions are genuinely fixed and verified: the relative-path Critical (absolutized `runDir` at the boundary, `run.go:56`, with a dedicated `TestRunExperiment_RelativePath` regression), the `metrics.json`-in-artifacts leak (excluded in `collectArtifacts`, `exec.go:118`, asserted by an exact-set check), the atlas step-contract gap (now documented in `atlas/experiment.md`), and the doubled step-id prefix. `go test ./...` is green and `go vet` clean. The pure/IO seam is real and well-tested. What keeps this from a clean SHIP is one **new, confirmed correctness bug** the prior pass missed: `TopoSort`/`Validate` mis-classify a valid DAG as cyclic when a step lists the same predecessor twice — a latent defect in the milestone's headline pure logic. It fails safe (a wrong rejection, not silent bad execution) and needs a degenerate input, so it's non-blocking at the gate but must be fixed before M3 wires hand-authored pipelines.

### 1. Strengths
- **ARCH-PURE seam is real, not asserted.** `Runner.Run` orchestrates `Validate → TopoSort → execute → assemble` behind an injected `StepExecutor` + `Clock` (`pkg/experiment/run.go:24-45`), unit-tested with a fake and zero subprocess (`run_test.go`). All IO lives in `cmd/metis`. This is the milestone's best work.
- **The prior Critical fix is correct and now guarded.** `TestRunExperiment_RelativePath` (`cmd/metis/run_test.go`) chdirs into the workspace and passes a bare filename — exactly the invocation that was broken — and asserts the step's real output artifact exists. That is the right regression, not a restatement of the fix.
- **Exact-artifact-set assertion** (`run_test.go:66`) now pins `["first/echoed.json","second/echoed.json"]` and nothing else, so both the metrics.json leak and any stray-file regression are caught.
- **`TopoSort` reused by `Validate`** for acyclicity (`validate.go:44`) — one implementation (ARCH-DRY), and unknown-id edges are deliberately ignored so a typo'd `needs` reports as dangling rather than misfiring cycle detection.
- **`uses` regex is validated before `resolve` ever runs**, so `filepath.FromSlash(uses)` can't path-traverse out of `stepPath` — defense in depth on the subprocess boundary.

### 2. Critical findings
None. The prior boundary's Critical (relative-path step failure) is fixed at `run.go:56` and verified by `TestRunExperiment_RelativePath`.

### 3. Important findings
- **`pkg/experiment/validate.go:97-101` (in-degree count) vs `:112-119` (relaxation): duplicate `needs` edge → false "cycle".** In-degree is incremented once per element of `s.Needs` (`indeg[s.ID]++` for each `n`), but relaxation decrements once per *dependent step* (`dependsOn` is boolean). So `needs: [a, a]` gives `indeg[b]=2`, processing `a` drops it only to `1`, `b` is never dequeued, and `Validate`/`TopoSort` return `cycle in step dependencies among: [b]` for an acyclic graph. **Confirmed** with a throwaway test (`TopoSort falsely rejected … cycle in step dependencies among: [b]`). Fails safe (rejection, not bad execution) and needs an uncommon input, hence Important not Critical — but it's a real defect in the semantic gate that is M2's point.
  - **Fix sketch:** count distinct known predecessors — e.g. per step, dedupe `s.Needs` into a set before incrementing `indeg` (and skip a self-edge). Add a table case `needs: [a, a] → passes` to `validate_test.go`/`TestTopoSort`.
- **No IO-level test for a failing real subprocess step writing a `failed` ledger.** `runExperiment` deliberately writes `run.json` + appends `## Runs` even on a mid-run step error (`run.go:78-89`), but every `cmd/metis` test drives either the happy path or a validation rejection. The `status:"failed"` ledger-on-failure branch — a stated design guarantee — is unexercised, so a regression that returned early before `writeRunJSON` would ship silently. **Fix:** add an e2e case with a step-type that exits non-zero (a second fixture executable or a `with`-driven fail flag), asserting `runExperiment` returns an error, `run.json` exists with `status:"failed"`, and a `## Runs` bullet was appended.

### 4. Minor findings
- **`collectArtifacts` is non-recursive** (`exec.go:116` skips `ent.IsDir()`): a step that writes `sub/out.csv` gets neither the dir nor the nested file recorded. Fine for the flat M2 fake step; note before M3 steps emit nested outputs.
- **`run.json` is never validated against CUE `#Run`.** The conformance guard covers only the `Experiment` fixture; the produced ledger (the record M3/kbench derive from) has no drift guard. Hard with the markdown-frontmatter validator; note for M3.
- **"walk up to `go.mod`" is implemented 3×** — `cmd/metis/main.go:findRepoRoot` + two byte-identical `repoRoot` test helpers (`cmd/metis/helpers_test.go`, `pkg/experiment/helpers_test.go`). ARCH-DRY nit; cross-package test-helper sharing in Go is awkward, acceptable to leave.
- **The drift guard is weaker than it reads.** `TestParse_ConformsToCUE` only proves Go `Parse` *and* `vocabulary validate-instance` both accept the *same one fixture* — it would not catch a Go field renamed/dropped as long as the fixture still round-trips. Honest about the tension, but it's co-acceptance, not field correspondence.
- **`appendRunLog` probes with `strings.Contains(body, "## Runs")` and appends at EOF** (`run.go:129-134`): a bullet lands under the wrong heading if an experiment ever has sections after `## Runs`. Fine for the move-1 "`## Runs` last" convention.
- **Re-running with an explicit `--run <existing-id>`** re-collects any stale files left in the step dir from the prior run (no clean). User-error path; note only.

### 5. Test coverage notes
- Pure core is well covered: `Validate` table (cycle/dangling/bad-uses/dup-id) + `errors.Join`-all-violations + `TopoSort` order/cycle; `Runner.Run` order/assembly + step-failure-stops-pipeline via a fake executor — exactly the PURE-without-IO shape the checklist wants.
- Two real gaps: (a) the **duplicate-needs** case (the confirmed bug above) is untested; (b) the **failing-real-step ledger-on-failure** branch is untested at the `cmd/metis` boundary. Both are cheap to add and would have caught a shipped-class bug.

### 6. Architectural notes for upcoming work
- **`METIS_RUN_DIR` becomes the move-1 artifact-graph substitute.** Now that it's absolute, `$METIS_RUN_DIR/<upstreamStepID>/…` is how an M3 `train`/`predict` step reaches `cv-split`'s folds. State the exact convention in the atlas step-contract so downstream steps depend on a stable path, not on reading `exec.go`.
- **`stepPath()` walks from the runner's cwd to `go.mod`.** kbench experiment instances live outside the metis tree, so `metis run` from a competition workspace won't find `steps/` and must rely on `$METIS_STEP_PATH`. Define the layered precedence (metis + kaggle + kbench step dirs) before M3 wires real `metis/*` steps.

### 7. Plan revision recommendations
- **Append an M2 `## Revisions` entry to `workshop/plans/000001-experiment-datatype-plan.md`.** The plan's only Revisions block is M1's; the M2 REWORK→fixed delta (relative-path Critical absolutized at the boundary + relative-path regression test; `metrics.json` excluded from artifacts; step-contract mapped into atlas) is undocumented there, and the issue `## Log`'s "M2 shipped" narrative under-reports it (it reads as a clean first pass — "go test ./... green" — with no mention of the rework). Per AGENTS.md §1 the delta should be appended, not silently absorbed. If the duplicate-needs bug is fixed as part of crossing this boundary, note that too.
- The Chunk-2 core-concepts tables (`plan.md:280-304`) still match the code — `Experiment`/`Parse`/`Validate`/`TopoSort`/`Runner.Run`, `StepExecutor`, `execStep`, run-ledger writer all exist at the stated paths with the stated PURE/INTEGRATION split. No table/code contradiction.
