---
id: 000016
status: open
deps: []
github_issue:
created: 2026-07-06
updated: 2026-07-06
estimate_hours:
---

# metis run discovers step layers from the dependency graph — no METIS_STEP_PATH wrapper (krun collapses)

## Problem

Running a workflow is **metis's job** (`metis run`), but today metis can't find the step
*implementations* on its own — it relies on `METIS_STEP_PATH` being set for it. That env var is
assembled by a bespoke per-workspace wrapper: kbench's `bin/krun` hardcodes
`METIS_STEP_PATH="$PEERS/metis/steps:$PEERS/kaggle/steps:$KBENCH/steps"`. This mis-layers the model:
- **`run` belongs to metis** (the ML workflow engine); a workspace should not need a wrapper that
  re-implements "which step layers exist".
- The step layers are exactly the **dependency chain**: a workspace (kbench) depends on kaggle,
  which depends on metis; each contributes a `steps/` dir. So "which steps are available" is
  **dependency resolution** — the same transitive layer-walk `weave` already does for skills — not
  something to hand-list per workspace.

## Spec

- **`metis run <experiment>`, executed inside a workspace repo, discovers its step-path by walking
  that repo's dependency chain** (kbench → kaggle → metis), collecting each layer's `steps/` dir.
  The effective step-path = the current repo's `steps/` + each transitive dependency's `steps/`
  (nearest layer wins on a name clash, or error on ambiguity — decide at plan time). Source of the
  dep chain: the ariadne/`construct/` base-layer declaration weave already reads (`base.manifest` /
  the deps graph) — reuse it, don't invent a second dep list.
- **`metis run <experiment>` works with NO `METIS_STEP_PATH` set and NO `krun` wrapper.**
  `METIS_STEP_PATH` stays as an explicit override (tests, odd layouts).
- Consequence: **`krun` collapses** — kbench's `bin/krun` becomes `metis run` (or is deleted). Track
  that as a kbench follow-up once this lands.

## Done when

- In the kbench repo, `metis run competition/titanic/pipelines/titanic-features.md` (no
  `METIS_STEP_PATH`, no `krun`) resolves `titanic/adapt` (kbench), `kaggle/download` (kaggle), and
  `metis/cv-split` (metis) via the dependency walk.
- metis's own step tests + the kbench hermetic e2e still pass (via `metis run` directly).
- atlas: step-layer discovery = the dependency layer-walk (cite the weave/`construct` parallel).

## Plan

- [ ] RED: `metis run` in a fixture workspace with a dep chain resolves a dependency's step with no METIS_STEP_PATH.
- [ ] GREEN: walk the `construct/` dep chain → assemble the step-path (reuse weave's dep-graph source); METIS_STEP_PATH as override.
- [ ] Name-clash policy (nearest-wins or error) + test.
- [ ] atlas + a kbench follow-up to collapse `krun`.

## Log

### 2026-07-06
- Filed from the layering discussion (operator): run is metis's; kaggle contributes steps; kbench is
  a workspace. The `krun` wrapper's only real job — assembling METIS_STEP_PATH — is dependency
  resolution metis should do itself (the weave layer-walk), so any workspace just runs `metis run`.
