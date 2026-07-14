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
  - `PointAddress(resolvedWith, shapeBlobHash, seed) (Hash, error)` — mints the **intent identity**
    (metis#27: the pre-run config+shape-blob+seed content-address, NOT repo HEAD, NOT the per-step
    trace) via canonical `json.Marshal` → `cas.HashOf`; errors on non-finite config.
  - `CodeFingerprint([]CodeRef) (Hash, error)` — the **realized code identity** (metis#27): a
    `{path, blob_hash}`-only manifest of the run-end read-set `D` closure, sorted + canonical-hashed
    (excludes the absolute repo root, so it's checkout-portable). Ledger dedups on `(point_address,
    code_fingerprint)` so same-config-different-code runs are both kept.
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
  - `model.py` `train`/`predict`/`cv_score` — sklearn `logreg`/`rf`/`hist_gbm`, deterministic by seed;
    `cv_score` averages per-fold validation accuracy. `make_model(kind, seed, params)` **applies
    the swept hyperparams** (`logreg` C; `rf` n_estimators/max_depth; `hist_gbm`
    learning_rate/max_iter/max_leaf_nodes/max_depth — metis#21); `params` threads through
    `train`/`cv_score` (default `{}` = sklearn defaults). Adding a model kind is Python-only (`MODELS`
    + `make_model` + `complexity`); the Go layer derives the family structurally (`FamilyOf`), zero edits.
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
- **Read confinement (metis#23 nested-CV, L2 chokepoint) — `metis/io.py`:** `within_root` +
  `assert_within_read_root` wired into **`exp_path`** — the single base-dataset resolver
  (`cv_split` direct + `dataset_dir`'s exp-relative fallback). When `METIS_READ_ROOT` is set
  (an outer-fold sweep runs sealed on its `analysis_i/`), every exp-relative data read is
  asserted within that root; a violation is a loud `RuntimeError` naming the file. Upstream
  run-dir **handoffs** (`dataset_dir`'s upstream branch) bypass `exp_path` → never confined, so
  a legit `features→train` handoff isn't flagged. Var unset (flat `driver:single`) → no-op.
  Injected by `execStep` (`exec.go`, iff `readRoot` non-empty) from `runOpts.readRoot`; decoded
  into `StepContext.read_root`. **Deferred (documented):** syscall-level airtightness (rogue
  non-`metis.io` opens, parquet-via-C bypass of the audit hook) — the airtight version is a
  syscall sensor swap. Pairs with the **L1 structural** seal (`outer-split` subset dirs).
- **Nested-CV (metis#23; derived, records — metis#32) — `cmd/metis/sweep.go`:** `metis run` on a
  `>1`-config sweep runs nested CV (the mode is now DERIVED by config-count — the shape `driver:` field
  was removed in #32; outer folds = `sweeper.resample.cv.k`, or 1 under `--fast`). `runNestedCV` wraps
  the black-box sweeper in an OUTER resample (the pure `sampler.CVDriver` over the unchanged `Run`
  loop). Preamble (`materializeOuterAnalysis`) runs `{data + outer-split(k=outerK)}` once →
  `analysis_i/` dirs. Per outer fold (`runOuterFold`): (a) a **sealed** sweep (`runSweeper` repointed
  at `analysis_i`, `readRoot`=analysis_i abs → L1+L2) selects per-family winners — `GuardComplexity` runs
  here too, so a parsimony rule + non-reporting model is rejected exactly as on the flat path (not silently
  mis-selected); (b) refit-and-score **each family's inner-winner** as a full-data fold at the OUTER k,
  held=i (post-selection → unconfined; `cv_folds` determinism reproduces `analysis_i`'s partition).
  `Aggregate` → **mean±SE**, the honest procedure estimate (`reportEstimate`). **metis#32:** the run now
  **records** per-`(outer-fold, config)` inner rows + per-`(outer-fold, family)` outer rows to the ledger
  (`Level`-keyed) — the signal `metis select` reduces to pick the family. `metis run` **measures only,
  never ships** (shipping moved to `metis select --promote`). Honesty of the score-over-full-data
  refit holds while features are stateless; stateful features (metis#20) inherit fold-safety via the
  fold-expressed score run.
- **Parallel batch executor (metis#31) — `pkg/sampler/exec.go` + `cmd/metis/{exec,run,sweep}.go`:**
  `Run` takes an injected `exec(batch, runPoint) []O` that runs one `Ask` batch and returns outputs
  **in batch order** (`SeqExec` serial default · `ParExec` goroutine fan-out · `ExecFor(parallel)`
  selects). A batch is independent by construction, so the order-independent reduce (metis#18) yields a
  byte-identical `Done` either way — parallelism is a pure speedup, not a semantic change. The ONE
  budgeted resource is the real subprocess spawn: a single shared `chan struct{}` semaphore (cap
  `--parallel`, default `NumCPU`, `METIS_MAX_PARALLEL`) acquired around `cmd.CombinedOutput()` in
  `execStep` — a cache HIT never reaches there, so only misses draw budget, and orchestration
  goroutines never hold a slot while awaiting children ⇒ **≤ n concurrent step subprocesses across ALL
  driver×sweeper×resample nesting, deadlock-free**. `runExperiment` establishes the parallel invariant
  (non-nil sem + a `syncWriter` over `out`) in one home. Determinism of persisted artifacts: the fan-out's
  completion-order `pass.points` are `sortPointRuns`-sorted before the manifest/ledger write; the
  `sweepPass` mutex guards the shared `configs`/`points`/`err` bookkeeping (the honest reduce stays pure
  in the sampler). Caveats (flag help): each leaf is a Python process that may itself multi-thread
  (BLAS/`n_jobs`) so `n=NumCPU` can oversubscribe; a COLD cache thundering-herds the shared upstream;
  clean per-`k/n` progress is deferred to metis#30.
- **Step entrypoints — `metis/steps/{cv_split,train,predict,outer_split}.py`:** thin `io → pure core → io`.
  - `cv-split`: load Dataset (`with.dataset`, exp-relative) → `cv_folds` → `folds.json` + `{k,n}`.
  - `train`: load Dataset + upstream `folds.json` → `cv_score` + fit-on-all → `model.pkl` + `{cv_score}`.
  - `predict`: load Dataset + upstream `model.pkl` → predict test rows → `predictions.csv` + `{n_predictions}`.
  - `outer-split` (metis#23, L1 structural seal): read the FULL dataset (**unconfined** — it must
    see all rows to split them) → `cv_folds` → k `analysis_i/` **subset dataset dirs** (train where
    `outer_fold != i`; assessment rows physically absent) + `outer_folds.json`. The sealing spine
    **#20 (leakage-safe features) + kbench#8 (ticket-group survival) inherit.**
- **Wrappers — `steps/metis/{cv-split,train,predict,outer-split}`:** bash bridges that `exec uv run
  --project <root> python -m metis.trace metis.steps.<type>`, resolving `<root>` from `$0` (cwd is the
  step dir, not the root).
- **Data-flow:** the dataset is referenced experiment-relative (`METIS_EXP_DIR`); `folds` and
  `model` flow between steps via the upstream-artifact convention (the step id is named in the
  consumer's `with`, e.g. `train` `with:{folds: split}`).
- **Proof:** `testdata/experiment/toy-pipeline.md` (cv-split → train → predict on the toy
  `testdata/dataset/toy/`) runs end-to-end via `metis run` to a real CV score — the metis#1
  Done-when. `cmd/metis` `TestToyPipeline_EndToEnd` drives the real wrappers (skips without uv);
  the pure core + contract are pytested under `tests/`. The Titanic thread is kbench#1.

### Leakage-safe target features (metis#20)

A *target* feature (value derived from other rows' labels, e.g. group-survival) has two leak
layers. Cross-**fold** leakage is already structural (features live in the `pipeline` phase → fit
per-fold via `fit_mask`). The remaining *within*-fold self-leak (a row's own label building its own
feature) is the **feature step's own job** — no engine `fit_scope` marker (dropped, pensive).

- **`metis/encode.py::cross_fit_target_encode(groups, y, *, fit_mask, strategy, n_folds, m, loo_noise, seed)`**
  — the reusable, competition-agnostic primitive. Fit rows get an internal cross-fit encoding (own
  label never enters via the group aggregate); non-fit rows (assessment + test) get the full-fit
  shrunk group mean (prior when unseen). `strategy ∈ {kfold (default, reuses `metis.split.cv_folds`
  for the internal folds), loo}`; m-estimate shrinkage toward the global prior. Pure, seed-deterministic.
- **Consumer protocol** lives downstream in the competition workspace: `kbench/titanic/features.py`
  carries a `TARGET_GROUPS` registry (separate from the stateless `GROUPS`) + a `target_encode_group`
  adapter that concats train+test keys, marks analysis rows as fit, and calls the primitive. Metis
  owns the leakage-safe math; the step owns the wiring. kbench#8 adds the `ticket` group on top.

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
