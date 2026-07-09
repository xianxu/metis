# experiment — the reproducible unit of ML work

An experiment is a git-tracked, declarative **pipeline of steps** — *issue-shaped*: schematized
frontmatter (the machine-executable pipeline + config) over a freeform body (hypothesis + prose).
It is **immutable input** (#13): a run never writes back into the `.md` — run history lives in
`runs/<id>/record.json` + the `.ledger.csv` sidecar (browse via `metis ledger show`; apply the
sweeper's two-level select rule offline via `metis ledger select --rule R`, #19), so a
committed config is a stable content-hash. The Go step-runner
(`metis run <id>`, M2) executes it with **no agent in the loop**, unifying data
reconstruction, training, and experiment tracking under one entrypoint.

## Surface (M1)

- **Schema — the single source:** `construct/vocabulary/experiment.cue`
  - `#Experiment` — `type` / `id` / `competition?` / `seed` / `status` / `steps`
  - `#Step` — `id` / `uses` (`"<layer>/<steptype>"`) / `needs?` (DAG edges) / `with?`
  - `#Status` — `draft | active | archived`
  - `#Run` — the ledger record shape (produced by the runner in M2)
- **Authoring form:** `construct/datatype/experiment.md` — the datatype prototype, merged
  into metis's `xx-datatype` skill (DAG-merge, leaf-wins).
- **Structural validator:** `vocabulary validate-instance --type experiment <file>` — the
  inherited ariadne binary; `cue vet`s extracted frontmatter against `#Experiment`.
- **Enforcement:** `scripts/merge-checks.d/experiment-validate.sh` — a merge-gate hook that
  validates changed `type: experiment` files (skips `testdata/`, which holds intentionally
  malformed fixtures).
- **Fixtures:** `testdata/experiment/{valid-baseline,invalid-bad-status}.md`.

## Surface (M2) — the Go step-runner

`metis run [--run <id>] [--cache] <experiment.md>` reads + validates an experiment, executes
its steps in dependency order as **subprocesses** (files + subprocess, never FFI), and records
a Run. `--cache` (default on) enables the metis#2 validating-trace cache (see `atlas/index.md`
`pkg/cache`). Split across a pure core and a thin IO layer:

- **Pure core — `pkg/experiment/`** (no IO; unit-tested directly):
  - `Experiment` / `Step` / `Run` — Go structs mirroring the CUE `#Experiment`/`#Step`/`#Run`
    (the CUE stays the single *structural* source; a conformance test guards against drift).
  - `Parse(content) (Experiment, error)` — reuses ariadne `frontmatter.Split` + `yaml.v3`.
  - `Validate(Experiment) error` — the semantic checks CUE can't express (unique ids, `needs`
    resolution, `uses` = `^[a-z0-9-]+/[a-z0-9-]+$`, acyclicity); joins all violations.
  - `TopoSort(Experiment) ([]Step, error)` — Kahn's algorithm; the one acyclicity impl.
  - `Runner.Run(exp, runID, runDir) (Run, []StepRun, error)` — orchestrates Validate → TopoSort →
    execute-each → assemble the `Run`, and returns the per-step `[]StepRun` (topo order) so the
    provenance record (`pkg/record`) can reach the per-step data the flat `Run` merge discards.
    Step execution is injected via the `StepExecutor` interface, so the orchestration is
    fake-executor tested with **no subprocess** (the ARCH-PURE line).
- **Provenance record — `pkg/record/`** (metis#3, pure; leaf over `pkg/cas`):
  - `RunRecord` / `StepRecord` / `CodeManifest` / `FileHash` — the unified per-step provenance
    record (the L0 reproducibility atom), emitted as `runs/<id>/record.json`. Mirrors the CUE
    `#RunRecord`/`#StepRecord` (drift-guarded). Fields split by role: key-material (`With`,
    `Upstream`, `Code`) vs. provenance-extras (`OutputHash`, `Metrics`).
  - `PointAddress(resolvedWith, repoSHAs, seed) (Hash, error)` — mints the **L0 run-identity**
    (repro key; the coarse config+repo+seed content-address, NOT the per-step trace) via canonical
    `json.Marshal` → `cas.HashOf`; errors on non-finite config.
  - `OutputHash([]FileHash) Hash` — reduces a step's multi-file output set to one address (sorted
    `(path, content-hash)` manifest).
  - **Scope line:** #3 owns the record + point-address; the read-set trace `D` + cache key are
    **metis#2** (they populate `Code.D`/`Deps`/`Upstream`); side-ref code *capture* is **metis#7/#8**
    (#3 records the current commit + dirty flag).
- **Thin IO — `cmd/metis/`:** `execStep` (the real `os/exec` `StepExecutor`) + the record assembler
  (`assembleRecord`/`buildRecord`, git provenance via an injected `gitProbe`, per-step
  output-hashing) that writes `record.json` (+ the sweep `.ledger.csv` sidecar). It does **not**
  write to the experiment `.md` (#13 — immutable input). `runDir` is absolutized at this boundary so
  step paths resolve from any cwd.

### Step-executable contract (what M3 step-types must honor)

The runner invokes one executable per step, resolved from `uses: <layer>/<steptype>` to
`<stepdir>/<layer>/<steptype>` on the **step path**; first existing file wins
(`cmd/metis/exec.go:resolve`). The step path (`cmd/metis/steppath.go:stepPath`) is:
1. `$METIS_STEP_PATH` (OS-list-separated) if set — the explicit override; else
2. **discovered from the workspace's dependency graph** (metis#16): anchor on the
   experiment's nearest `construct/base.manifest` ancestor, walk that repo's
   `construct/deps` chain with **`ariadne/pkg/layergraph`** — *the same topology source
   `weave` reads for skills* (ARCH-DRY, one dep-graph walk, not a second parser) — and
   take each layer's `steps/` dir, **nearest (leaf) first**. So `metis run` in kbench
   discovers `kbench/steps` → `kaggle/steps` → `metis/steps` with no wrapper (the old
   `kbench/bin/krun` collapses — a kbench follow-up); else
3. `<repo.Root(cwd)>/steps` (a bare repo with no construct marker).

Because `resolve` is first-match-wins, **leaf-first ordering = nearest-layer-wins**: a
workspace step shadows a base-layer step of the same name (the correct layer-override
semantics). This **inverts** the retired `krun` wrapper's base-first order — harmless
today (the `metis`/`kaggle`/`titanic` namespaces are disjoint, so no clash), but the
krun-collapse follow-up must not assume byte-identical resolution. A found-anchor-but-
broken-graph surfaces layergraph's actionable error rather than degrading silently.

- **Working dir:** `runs/<run-id>/<step-id>/`, created by the runner; the child runs with
  its **cwd set to this dir**.
- **Env:** `METIS_STEP_DIR` (that dir, absolute), `METIS_RUN_DIR` (the run dir, absolute),
  `METIS_STEP_ID`, plus (M3) `METIS_EXP_DIR` (the experiment dir, absolute — the stable
  anchor for experiment-relative inputs, since the run dir is ephemeral) and `METIS_SEED`
  (the experiment's seed, so steps are reproducible without duplicating it into every `with`).
- **In:** `with.json` — the step's `with` config, written into the step dir by the runner.
- **Out:** an optional `metrics.json` (flat `{name: number}`, merged into `Run.metrics`) plus
  any **artifact files** the step writes into its dir. `with.json` and `metrics.json` are the
  reserved contract channels and are NOT counted as artifacts; every other file is recorded in
  `Run.artifacts` as a `runs/<id>/`-relative (step-qualified) path. A non-zero exit fails the
  step and halts the run.
- **Ledger:** `runs/<run-id>/run.json` (the `#Run` record — the record of truth) + `record.json`
  (provenance) + the sweep `.ledger.csv` sidecar. **The experiment `.md` is not touched** (#13 —
  immutable input; the human top-N view is `metis ledger show`). A run rejected at validation time
  writes nothing. M2 ships a process-level fake step (`testdata/steps/test/echo`) exercising this
  contract end-to-end; real `metis/*` step-types arrive in M3.

## Surface (M3) — the Python data plane

The real `metis/*` step-types the M2 runner invokes: a **pure Python numeric core**
wrapped by **thin step-executables** honoring the contract above. Hermetic via **uv**
(pinned CPython 3.12 — the system 3.14 has no scientific-stack wheels yet).

- **Pure core — `metis/`** (pytested on in-memory frames, no IO — ARCH-PURE):
  - `schema.py` `Schema` — column roles (`id`/`feature`/`target`/`weight`) + dtypes;
    `feature_cols()`/`target_col()`/`id_col()`.
  - `dataset.py` `Dataset` — `{schema, train, test?, provenance}` (pandas) + `X()`/`y()`
    selectors. The modality-agnostic envelope adapters produce (tabular now).
  - `split.py` `cv_folds(df, k, seed, stratify_col?)` — deterministic (Stratified)KFold
    fold assignment.
  - `model.py` `train`/`predict`/`cv_score` — sklearn `logreg`/`rf`, deterministic by seed;
    `cv_score` averages per-fold validation accuracy. `make_model(kind, seed, params)` **applies
    the swept hyperparams** (`logreg` C; `rf` n_estimators/max_depth); `params` threads through
    `train`/`cv_score` (default `{}` = sklearn defaults).
  - **Model-config contract (`parse_model_config`, metis#12):** the `with["model"]` value is EITHER
    a kind string (`"logreg"`) OR the **`$any` map** (tagged, ex-`$oneof`) single-key bundle carrying the
    swept hyperparams (`{"rf": {"n_estimators": 200, "max_depth": 4}}`); `parse_model_config(raw) →
    (kind, params)` normalizes both (malformed = loud error). This is what lets a **hyperparam
    sweep** (kbench#4) train — the `$any`-map branch reaches the estimator, not just the kind.
- **Thin IO — `metis/io.py`:** the SINGLE Python encoding of the step contract (ARCH-DRY):
  `step_context()` (reads the `METIS_*` env), `read_with`, `exp_path` (experiment-relative),
  `upstream_path` (`$METIS_RUN_DIR/<step-id>/<file>`), `out_path`, `write_metrics`, plus
  Dataset `load_dataset`/`save_dataset` (parquet canonical; CSV also read, so fixtures stay
  git-legible).
- **Step entrypoints — `metis/steps/{cv_split,train,predict}.py`:** thin `io → pure core → io`.
  - `cv-split`: load Dataset (`with.dataset`, exp-relative) → `cv_folds` → `folds.json` + `{k,n}`.
  - `train`: load Dataset + upstream `folds.json` → `cv_score` + fit-on-all → `model.pkl` + `{cv_score}`.
  - `predict`: load Dataset + upstream `model.pkl` → predict test rows → `predictions.csv` + `{n_predictions}`.
- **Wrappers — `steps/metis/{cv-split,train,predict}`:** bash bridges that `exec uv run
  --project <root> python -m metis.steps.<type>`, resolving `<root>` from `$0` (cwd is the
  step dir, not the root).
- **Data-flow:** the dataset is referenced experiment-relative (`METIS_EXP_DIR`); `folds` and
  `model` flow between steps via the upstream-artifact convention (the step id is named in the
  consumer's `with`, e.g. `train` `with:{folds: split}`).
- **Proof:** `testdata/experiment/toy-pipeline.md` (cv-split → train → predict on the toy
  `testdata/dataset/toy/`) runs end-to-end via `metis run` to a real CV score — the metis#1
  Done-when. `cmd/metis` `TestToyPipeline_EndToEnd` drives the real wrappers (skips without uv);
  the pure core + contract are pytested under `tests/`. The Titanic thread is kbench#1.

## Ownership & instances

The type + (M2) runner are **metis's** — platform-independent. *Instances* live in a
downstream **competition workspace** — `kbench/competition/<slug>/pipelines/<id>.md` — not
in metis; metis carries only test fixtures. Each layer contributes step types
(`metis/cv-split`, `kaggle/download`, `titanic/adapt`); a pipeline wires them.

## Validation split (why two validators)

CUE owns **shape** — types, enums, required fields, the `steps` list-of-structs. The
**semantic** checks — `needs` resolves to a real step id, the graph is acyclic, `uses` is
`<layer>/<steptype>` — are not expressible in `cue vet`. As of **M2** they live in the
**pure Go validator** `pkg/experiment.Validate` (with `TopoSort` for acyclicity), and
`metis run` invokes it **on read** — a cyclic or dangling-`needs` experiment is rejected
before any step executes, closing the SHAPE-only gap M1 deferred (execution-time
enforcement). This is the ARCH-PURE seam: the parse/validate/orchestrate core is pure;
the subprocess step execution + run-ledger are the thin `cmd/metis` IO layer.
