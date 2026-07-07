---
id: 000016
status: codecomplete
deps: []
github_issue:
created: 2026-07-06
updated: 2026-07-06
estimate_hours: 0.94
started: 2026-07-06T22:32:07-07:00
actual_hours: 0.56
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

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.2   impl=0.35
item: atlas-docs             design=0.05  impl=0.1
item: milestone-review       design=0.0   impl=0.2
design-buffer: 0.15
total: 0.94
```

Small because the dep-graph walk is already a public importable package
(`ariadne/pkg/layergraph`, ARCH-DRY) — the change is one function (`stepPath`) +
a `repo.FindUp` generalization + fixture tests. The core is one `smaller-go-module`
(extend `cmd/metis`); one review at close.

## Plan

Durable plan: `workshop/plans/000016-metis-step-layer-discovery-plan.md` (reviewed).
Single-boundary (plain checkboxes, one `sdlc close`).

- [x] T1: `repo.FindUp(start, marker)` — generalize the go.mod up-walk (ARCH-DRY); `Root` delegates.
- [x] T2: `stepPathFromLayers` (pure) — reverse layergraph's base-first order → leaf-first `steps/` dirs, drop no-`steps/` layers.
- [x] T3: `stepPath(expPath)` — anchor on `construct/base.manifest`, walk `layergraph`, nearest-first; `METIS_STEP_PATH` override; fixture invocation-path test + nearest-wins clash test.
- [x] T4: real-kbench hermetic e2e proof (no `METIS_STEP_PATH`, no `krun`) + atlas + file kbench `krun`-collapse follow-up.

## Log

### 2026-07-06
- 2026-07-06: closed — Re-close to re-review the post-review delta (ec57e56): added TestStepPath_BrokenGraphDegradesLoudly for the error/degrade branch (the close-review Important finding), tightened resolve-prefix assertions to separator-terminated /steps/ (Minor), plan ## Revisions noting Walk error is surfaced not swallowed. go test ./... all green (6 steppath tests incl broken-graph). No production-code behavior change since the SHIP review — test + docs only.; review verdict: SHIP
- 2026-07-06: closed — metis run discovers the step-path from the construct/deps dep-graph (ariadne/pkg/layergraph, weave's source), leaf-first nearest-wins, METIS_STEP_PATH override. PROOF: cold hermetic titanic-baseline in real kbench, METIS_STEP_PATH unset + no krun → exit 0, all 7 steps resolved across all 3 layers (kaggle/download+submit, titanic/adapt+submission, metis/cv-split|train|predict). go test ./... all green (repo.FindUp hit/miss; stepPathFromLayers leaf-first; real stepPath→resolve 3-layer fixture; nearest-wins clash; env-override). Both change-code judges INFO. actual 0.56 = impl window (57c31e0→HEAD); design attention (exploration+plan+review forks) largely pre-first-commit/background, so this mildly under-counts design.; review verdict: SHIP
- Filed from the layering discussion (operator): run is metis's; kaggle contributes steps; kbench is
  a workspace. The `krun` wrapper's only real job — assembling METIS_STEP_PATH — is dependency
  resolution metis should do itself (the weave layer-walk), so any workspace just runs `metis run`.

### 2026-07-07 — implemented (durable plan `workshop/plans/000016-*`, both change-code judges INFO)
- **Reuse discovery:** the dep-graph walk is ALREADY a public importable package —
  `ariadne/pkg/layergraph.Walk(fs, root)` (extracted in ariadne#115 M1), "the SINGLE source of truth
  for repo R's layer graph", the same one weave consumes. metis already `replace`s ariadne in go.mod,
  so NO ariadne change and NO second dep parser — the whole change is one function + a `repo.Root`→
  `repo.FindUp` generalization.
- **Anchor = `construct/base.manifest`, not `go.mod`:** kbench has no `go.mod`, so the old `repo.Root`
  fallback would walk past it. Anchor on the experiment file's nearest `construct/base.manifest`.
- **Nearest-wins is free + INVERTS krun:** layergraph returns base-first `[ariadne,metis,kaggle,kbench]`;
  `exec.go:resolve` is first-match-wins; reversing to leaf-first gives nearest-layer-wins with zero new
  clash code. This flips krun's base-first precedence — harmless today (disjoint namespaces) but the
  krun-collapse follow-up must not assume byte-identical semantics (noted in atlas).
- **Done-when bullet 1 PROVEN in real kbench** — cold hermetic run, `METIS_STEP_PATH` unset, no `krun`:
  `env -u METIS_STEP_PATH KAGGLE_CLI=<fake> KAGGLE_FAKE=1 KAGGLE_FAKE_DATA_DIR=$PWD/competition/titanic/testdata/raw … ../metis/bin/metis run --run run-16cold competition/titanic/pipelines/titanic-baseline.md`
  → **exit 0, all 7 steps ran** resolving across all three layers: `kaggle/download`+`kaggle/submit`
  (kaggle), `titanic/adapt`+`titanic/submission` (kbench, `/Users/xianxu/workspace/kbench/steps/...`),
  `metis/cv-split|train|predict` (metis). Tree left clean (#13 immutability). `go test ./...` all green.
- **kbench follow-up filed:** collapse `bin/krun` → `metis run` (out of scope here — metis-only).
