---
id: 000001
status: working
deps: []
github_issue:
created: 2026-07-01
updated: 2026-07-01
estimate_hours: 6
started: 2026-07-01T13:51:21-07:00
---

# metis ML-workbench core: experiment datatype + step-runner + Dataset/Split/cv-split + step-types (Titanic skeleton)

## Problem

metis is an empty scaffold. The `kaggle-ml-base-layer` project (brain) needs the platform-independent ML core: a way to **define, run, and record a reproducible pipeline** (the `experiment` datatype + a Go step-runner), plus the tabular data primitives (Dataset / Schema / Split / cv-split) and the metis step-types (`cv-split`, `train`, `predict`) that the Titanic thread walks through. "Platform-independent" test: *would this be identical on a non-Kaggle platform?* — if yes it lives here.

## Spec

Design from the 2026-07-01 brainstorm. **Polyglot by seam**: a Go control plane (state/records that live in git) over a Python data plane (numbers that flow through RAM), interface = **files + subprocess, never FFI**.

- **`experiment` datatype** — markdown + CUE-validated frontmatter, issue-shaped. Frontmatter = the machine-executable **pipeline** (ordered `steps`, each `uses: <layer>/<steptype>` + `needs` + `with`) + config (slug `id`, `seed`, `status`). Body = freeform hypothesis/notes + an accreting `## Runs` log. Owned by metis (type + schema + runner); *instances* live in kbench competition workspaces so parley.nvim edits them like any issue.
- **Pipeline / Step** — the step graph inside the experiment frontmatter. Step-types are contributed per layer (`metis/cv-split`, `kaggle/download`, `titanic/adapt`); the pipeline just wires them. Move-1: **thin sequential runner** — no DAG-skip, caching, or artifact-graph.
- **Run** — one recorded execution appended to the experiment's `## Runs` log + a `runs/<id>/` artifact dir: bound params, seed, metrics, artifact paths, timestamps. This is the homegrown experiment ledger (CUE-schema'd). **DVC explicitly out** (revisit as a swappable backend only if expensive-step caching ever justifies it; the experiment file stays the single source).
- **Go step-runner** (`metis run <experiment-id>`) — reads the experiment, runs steps in dependency order, shells out to Python for numeric steps, streams **plain step progress** (bubbletea TUI polish is out of move-1), appends the Run. Unifies data-reconstruction, training, and experiment-tracking under one entrypoint.
- **Dataset** (canonical) — `{schema, splits, provenance}`, named + content-hashed; the internal format every competition's Adapter converts *into*. Envelope modality-agnostic; **tabular loaders only** now.
- **Schema** — columns + roles (`id | feature | target | weight`) + dtypes; CUE-defined.
- **Split** — a named partition (`train/valid/test`); **cv-split** = a k-way Split family (folds), deterministic/seeded.
- **Adapter protocol** — the interface `raw → Dataset` (metis defines the contract; kaggle provides the download half; kbench the titanic column-mapping).
- **metis step-types** (Python data plane) — `cv-split`, `train` (baseline logreg/RF), `predict`; each emits structured outputs (`metrics.json`, `predictions.csv`) the Go runner records.
- **Env** — local Python via `uv` + `pyproject`; scripts/modules, **no notebooks**.

## Done when

- `metis run <experiment>` executes a multi-step experiment end-to-end, shelling to Python steps, and appends a Run with metrics + artifacts.
- Dataset / Schema / Split(cv-split) + `cv-split`/`train`/`predict` step-types exist with **colocated unit tests** (PURE-majority core).
- The experiment/pipeline/step/run frontmatter is CUE-validated.
- Exercised end-to-end by kbench#1's Titanic thread reaching a local CV score through this runner.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
design-buffer: 0.30
item: typed-data-prototype   design=0.4 impl=0.8
item: greenfield-go-module   design=0.5 impl=1.5
item: greenfield-go-module   design=0.4 impl=1.5
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
total: 6.09
```

M1 = the experiment datatype + CUE schema (`typed-data-prototype`); M2 = the Go step-runner + `pkg/experiment` (`greenfield-go-module`); M3 = the Python data-plane step-types + Dataset/Schema/Split (`greenfield-go-module` as the closest proxy — the closed vocab has no Python primitive). Three `milestone-review` items for the M1/M2/M3 boundaries. AI-paired ship-wall-clock (v3.1); provisional — the calibration source flagged stale and metis has no local close history yet.

## Plan

- [ ] M1 — `experiment`/`pipeline`/`step`/`run` datatypes + CUE-validated frontmatter (schema + a fixture experiment)
- [ ] M2 — Go step-runner: read experiment, run steps sequentially via subprocess, append a Run record; plain streaming output
- [ ] M3 — Dataset/Schema/cv-split Python core + step-types (`cv-split`, `train`, `predict`) with unit tests + the files+subprocess contract (`metrics.json`/`predictions.csv`)

## Log

### 2026-07-01

Created from the `kaggle-ml-base-layer` project brainstorm (brain `data/project/kaggle-ml-base-layer.md`). This is the base of the substrate chain `kbench → kaggle → metis → ariadne`. Explicitly deferred to later projects: TUI polish, caching/DAG-skip, DVC backend, pipeline parameterization (`{param}` — the Run record already captures bound values), and good modeling.
