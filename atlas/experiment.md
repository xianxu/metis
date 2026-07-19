# experiment — the reproducible unit of ML work

An experiment is a git-tracked, declarative **pipeline of steps** — *issue-shaped*: schematized
frontmatter (the machine-executable pipeline + config) over a freeform body (hypothesis + prose).
It is **immutable input** (#13): a run never writes back into the `.md` — run history lives in
`runs/<id>/record.json` + the `.ledger.csv` sidecar (browse via `metis ledger show`; choose + ship via
`metis select [--best|--best-per-model-class|--point ADDR] [--promote]`, metis#32; `--point` = metis#41's operator-chosen publish-any-ledger-row, shipping as `point-{family}-{hash}` — retired `metis ledger select`), so a
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
  - `schema.py` `Schema` — column roles (`id`/`feature`/`target`/`weight`/`source`) + dtypes;
    `feature_cols()`/`target_col()`/`id_col()`. `source` (metis#35): a raw column carried
    through for feature-engineering steps that know it — never a model input, may hold
    strings/NaN (the one-road invariant: the adapter's Dataset is the sole road from raw
    data into the pipeline, so the nested-CV seal's substitution is complete).
  - `dataset.py` `Dataset` — `{schema, train, test?, provenance}` (pandas) + `X()`/`y()`
    selectors. The modality-agnostic envelope adapters produce (tabular now).
  - `split.py` `cv_folds(df, k, seed, stratify_col?)` — deterministic (Stratified)KFold
    fold assignment.
  - `model.py` `train`/`predict`/`cv_score` — sklearn `logreg`/`rf`/`hist_gbm`, deterministic by seed;
    `cv_score` averages per-fold validation scores under a **metric knob** (metis#59:
    `accuracy` default | `balanced_accuracy`; `resolve_scorer` is the ONE name→scorer site,
    loud on unknown; `metric=` threads keyword-default through `fold_fit`/`fold_score`/`cv_score`).
    **The decision layer (metis#60)** — the cost-sensitive plug-in rule, LEAF-LOCAL:
    `decide: {"offsets": {"holdout"}}` tunes per-class log-offsets (grid ±4, no-op-anchored)
    as a FITTED PARAMETER — aux stratified holdout inside the fold's training rows, main
    model unchanged on all training rows, assessment scored through the tuned decision, so
    the EXISTING seal covers fit+tune as one procedure (the impute-median precedent; no
    engine change). Two honest costs, deliberate: (i) SE INFLATION — the procedure's
    variance now includes tuning variance, so decide=offsets configs carry wider SEs and
    the 1-SE band widens (the estimate measures the whole procedure — correct, not noise);
    (ii) AUX/MAIN MISMATCH — offsets tune against the 80%-fit aux model's probabilities and
    apply to the 100%-fit main model's (no leakage; standard CV-style pessimism, assumed
    not measured). Price: 2 fits/leaf — pin down with `--sample out1in2` before a decision
    run. `make_model(kind, seed, params)` **applies
    the swept hyperparams** (`logreg` C; `rf` n_estimators/max_depth/class_weight; `hist_gbm`
    learning_rate/max_iter/max_leaf_nodes/max_depth/class_weight — metis#21, #59); `params` threads through
    `train`/`cv_score` (default `{}` = sklearn defaults). Adding a model kind is Python-only (`MODELS`
    + `make_model` + `complexity`); the Go layer derives the family structurally (`FamilyOf`), zero edits.
    **`ensemble` kind + seed passthrough (metis#65):** `ensemble` is a soft-vote blend built as
    an sklearn `VotingClassifier(voting="soft")` over `params["members"]` (a list of the SAME
    `$any`-map bundles, parsed by `parse_model_config` — one level of recursion) with optional
    `params["weights"]` — **the blend made scorable INSIDE nested CV** (an honest OOF estimate),
    as opposed to `metis blend`'s post-hoc leaderboard-only combine over promoted runs. It
    exposes the estimator API, so it composes with decide/metric/seal unchanged; offsets tune on
    the ensemble's AVERAGED proba. `complexity(ensemble)` = SUM of member realized complexities,
    each member's kind recovered from its `VotingClassifier` NAME (`<kind>-<i>`, set from the
    parse_model_config label — DRY, no estimator-type→kind reverse map). **Seed passthrough:**
    `make_model` reads `eff_seed = params.get("seed", ctx_seed)` at every estimator — absent =
    byte-identical (no re-key); present = a swept seed dimension that re-keys the leaf, and,
    composed with `ensemble` (one config × distinct member seeds), IS seed-bagging.
    **`catboost` kind (metis#65 M2):** the M5 mechanism bet (per-node ordered target
    statistics). Lazy-imported inside make_model (heavy dep — keep the forkserver preload
    light for other kinds). Params: `iterations` (aka `max_iter`, default 200), `depth`
    (default 6), optional `learning_rate` (else CatBoost auto); `class_weight: balanced` →
    `auto_class_weights="Balanced"` (loud on any other value). **ARCH-PURE pins:**
    `allow_writing_files=False` (no `catboost_info/` FS write), `logging_level="Silent"`,
    `thread_count=1` (metis#48 — the orchestrator owns parallelism; also the determinism
    guarantee). `complexity` = `tree_count_ × 2^depth` (oblivious/symmetric trees are full
    binary → the summed-leaves capacity proxy). CatBoost's `.predict()` returns `(n,1)`, so
    **`model.predict` ravels to 1-D at the one call site** (`reshape(-1)`, a no-op for sklearn
    kinds) — fixing catboost everywhere predict flows (fold scoring + the ship predict step).
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
  was removed in #32; outer folds = `sweeper.resample.cv.k`, or 1 under `--fast`, or M under
  `--sample out<M>` — metis#42's m-of-k sparse sampling, grammar metis#58: `--sample out<M>`,
  `in<N>`, or `out<M>in<N>` prefix-subsamples the OUTER and/or INNER fold enumeration (the
  partitions are ALWAYS split at the declared k / inner_k — the estimand, the train fraction each
  fold simulates; sampling = precision/cost only). The seam is `sweepPass.splitK` vs `runK`
  (sweep.go): splitK feeds the partition + leaf content-addresses, runK bounds `FixedKFolds` —
  so an `in2` iteration run's leaves are byte-identical addresses to a full run's first 2 folds
  and CACHE-ESCALATE into it (the ledger's point-address dedupe absorbs re-emitted rows; e2e
  `TestSample_CacheEscalationConverges`). select needs NO raggedness guard — residual ledger
  raggedness exists only after an INTERRUPTED run, and any completed re-run heals it (#58). `--fast` ≡ `--sample out1` (bare `--sample 3` is retired,
  loudly), and misuse (M>k, N>inner_k, single-config flat run, combined with `--fast`) fails
  loudly). `runNestedCV` wraps
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
  never ships** (shipping moved to `metis select --promote`). **metis#52:** `select --cohort` lists the
  fingerprint cohorts (delegates to the #39 core), and every pick line carries its
  `· point <addr>` override handle (a representative ledger-row addr; round-trips through
  `--point`). **metis#53:** `select --promote`
  (both `--best*` and `--point`) runs the **fingerprint-consistency guard** before executing the
  promoted run: the cohort's captured D closure (per-path blob hashes from `record.json`) is
  re-hashed against the working tree with the SAME `gitBlobHashes` capture uses — any drifted
  path REFUSES the promote with a diff-shaped message (path + captured→working blobs + the
  capture-commit restore hint; `--no-fingerprint-check` overrides loudly; absent/legacy
  provenance warns-and-proceeds, never blocks). Closes at the promote seam the silent-blend
  class the #32 cohort guard stops at the ledger; restore itself is metis#28. **metis#50:** a sweep ends with the
  run-end summary — elapsed wall-clock, rows→ledger, the cohort fingerprint, and the paste-ready
  `metis select … --fingerprint <fp>` follow-ups (completing #39's visibility loop: the operator
  never scrapes scrollback to assemble the next command; degraded capture degrades to `cohort ?`
  with un-pinned hints). Honesty of the score-over-full-data
  refit holds while features are stateless; stateful features (metis#20) inherit fold-safety via the
  fold-expressed score run.
- **Honest family selection (metis#32) — three commands, `run` measures / `select` chooses / `kaggle
  submit` uploads:** `metis run` on a `>1`-config sweep records the whole nested CV to the ledger — a
  **`Level`-keyed** `ledger.Row` (`inner` per `(outer-fold, config, inner-fold)` + `outer` per
  `(outer-fold, family)`; `Level` enters the `AggregateView` group key so inner/outer rows for one config
  don't merge). **`metis select`** (`cmd/metis/select_cmd.go`) reads it and chooses two-signal: the FAMILY
  by the honest OUTER estimate — `FamilyEstimate` (`cmd/metis/family.go`, a `FamilyOf`-keyed reduce, distinct
  from `AggregateView` because a family's winner differs across outer folds) → `sampler.FamilySelect`
  (lowest-SE-within-1-SE; NOT `SweepResult.Ship`'s cross-family inner-argmax) — and the CONFIG-within-family
  by the inner CV (`SelectConfigs.PerFamily`, the metis#19 rule). `--promote` reconstructs the winner
  (`promotedExperiment`) and runs it on ALL data → `runs/best-{family}-{hash}/submission.csv`. A multi-family
  ledger with no `outer` rows is a sharp error (never a silent inner-argmax). `metis run --fast` = one outer
  fold (a ~1/k honest single-point for iteration); `--sample out<M>[in<N>]` = M of the k outer folds
  and/or N of the inner_k per-config folds (metis#42/#58 — probe-cost control; an M<k SE has M−1 df
  and an in<N run selects on a noisier N-fold mean: probe with it, never re-select what ships on
  it). Retired `metis ledger select` + `metis promote`.
- **`metis blend <shape.md> --runs a,b[,...] [--weights ...] [--allow-mixed]`** (metis#60 M2,
  `cmd/metis/blend.go`): weights-only soft vote over PROMOTED runs — members averaged in
  TILTED log-space (`w_i·(log(clip(p_i)) + o_i)`, each member's persisted offsets baked in;
  zeros when absent), argmax → `runs/blend-<hash>/` (hash over member+normalized-weight
  pairs) with a blend-flavored `record.json` (embeds RunRecord + members/weights; carries
  the shape's steps so `kaggle submit --run blend-...` resolves the slug), then the shape's
  ship `submission` step execs via `execStep.Execute` → the literal
  `submission/submission.csv`. Guards: id/column agreement by NAME; per-member
  offsets↔columns by class label; missing probabilities.csv → "re-promote"; mixed
  `code_fingerprint`/`experiment` refused without `--allow-mixed`. HONESTY (printed): blends
  have no in-sweep OOF — leaderboard-measured only.
- **Parallel batch executor (metis#31) — `pkg/sampler/exec.go` + `cmd/metis/{exec,run,sweep}.go`:**
  `Run` takes an injected `exec(batch, runPoint) []O` that runs one `Ask` batch and returns outputs
  **in batch order** (`SeqExec` serial default · `ParExec` goroutine fan-out · `ExecFor(parallel)`
  selects). A batch is independent by construction, so the order-independent reduce (metis#18) yields a
  byte-identical `Done` either way — parallelism is a pure speedup, not a semantic change. The ONE
  budgeted resource is the real subprocess spawn: a single shared `leafBudget` (metis#66;
  cap `--parallel`, default `NumCPU`, `METIS_MAX_PARALLEL`) acquired around `cmd.CombinedOutput()` in
  `execStep` — a cache HIT never reaches there, so only misses draw budget, and orchestration
  goroutines never hold a slot while awaiting children ⇒ **≤ n concurrent step subprocesses across ALL
  driver×sweeper×resample nesting, deadlock-free**. `runExperiment` establishes the parallel invariant
  (non-nil sem + a `syncWriter` over `out`) in one home. Determinism of persisted artifacts: the fan-out's
  completion-order `pass.points` are `sortPointRuns`-sorted before the manifest/ledger write; the
  `sweepPass` mutex guards the shared `configs`/`points`/`err` bookkeeping (the honest reduce stays pure
  in the sampler). Caveats (flag help): a COLD cache thundering-herds the shared upstream; clean
  per-`k/n` progress is deferred to metis#30.
- **Live fold-ordered scheduling + graceful Q (metis#66) — `cmd/metis/prioritysem.go` + `{exec,run,sweep,main,progress,runcontrol}.go`:**
  the metis#31 `chan struct{}` semaphore became a `leafBudget` interface with two impls: `chanSem`
  (the DEFAULT — priority-blind global fan-out, today's behavior) and `prioritySem` (a min-heap
  semaphore that grants a freed slot to the LOWEST outer-fold index waiting). `execStep.priority` =
  the leaf's outer-fold index (threaded via `runOpts.priority`/`sweepPass.priority`;
  `runOuterFold`/`scoreOnOuterFold` set it). `--live` (implied by `--auto-stop`) builds the
  prioritySem so **outer fold 0 finishes first → the running mean±SE tightens fold-by-fold**, while
  the backfill invariant (`len(waiters)>0 ⟹ inflight==capacity`) keeps every core busy — "backfill"
  is emergent from the priority queue, not separate logic. **CRITICAL invariant: `--live` is
  byte-identical to the default run** (scheduling-only; the reduce is order-independent + sortPointRuns
  normalizes) — locked by `TestLive_ByteIdenticalToDefault`. Board **Q** (a `q`/`Q` line on stdin →
  `stdinStopSignal` → `runControl.requestStop`, a clean soft-latch distinct from a failure) is a
  graceful finalize: admitted-but-unstarted leaves short-circuit with `errRunStopped`, in-flight
  outer folds drain fast and are ABANDONED (`ss.abandoned`, excluded from `driverEvent`/the estimate),
  and `finalizeStopped` reports an honest partial `out<n>` over the completed folds + writes the
  partial ledger (both the full and stopped tails funnel through `persistNestedAndReport`, ARCH-DRY).
- **`--auto-stop` — incumbent-referenced loser stop (metis#66 M2) — `cmd/metis/autostop.go` + `sweep.go`:**
  reads the incumbent ONCE at run start from the shape's EXISTING ledger (`readIncumbent`: the best
  per-family OUTER aggregate mean by direction — no `--baseline`; prior-runs-only because
  `writeSweepLedger` runs at finalize). Runs the OUTER folds SEQUENTIALLY (`outerParallel=false`;
  inner sweeper/resample stay parallel — cores busy within a fold) so each fold's decision cleanly
  gates the next. After each completed fold, `evaluateAutoStop` applies the PURE `shouldStop` rule:
  a family with n≥2 scores whose one-sided 95% predictive bound on its full-k mean
  (`SEpred² = s²·r/k²·(1+r/n)`, `t_{n-1}` via `tCrit`) can't reach the incumbent is added to
  `stoppedFams` — **losers only** (a would-be winner's bound straddles the incumbent → runs full k;
  a truncated optimistic estimate is never shippable). `activeConfigs` drops stopped families' configs
  from later folds' sealed sweeps (the real budget reclaim — the inner sweep is the cost), never to
  empty. `markStoppedRows` retroactively tags a stopped family's outer rows `stopped: auto`
  (`ledger.Row.Stopped`, a ragged CSV column like fold/level/outer_fold). The rule is documented +
  unit-tested (`autostop_test.go`: loser stops / winner never truncated / borderline spared / both
  directions / `tCrit` table); the e2e (`autostop_e2e_test.go`) stops a known loser while the winner
  runs full k. Composes with `--live` (`--auto-stop` implies it); the `--live` determinism guarantee
  scopes to `--live` — an `--auto-stop` run is intentionally a smaller, different computation.
- **Board banding + result-last (metis#55):** color lives in the PAINTER only (renderBoard
  stays plain — the paint/content split): `redraw` adds a dim full-width `─` separator rule
  above the frame (counted in the erase math), bolds the aggregate line, colors ✓/▸ glyphs
  (the status line stays default — live telemetry is not de-emphasized, #56) — gated on `NO_COLOR` (env read at the one production wiring point;
  tests inject). Scroll-region chunks dump in bright-black GRAY (`writeScroll`, both pending-dump
  sites — #57) so the footer + result carry the eye. The run RESULT (estimate + #50 summary) routes through `summaryWriter` into
  the board's EPILOGUE, flushed after the final frame + cursor restore — the terminal ends on
  the paste-ready commands, not the board. Plain/redirected output is unchanged (zero SGR).
- **inner_k — the partial-inner-CV cost knob (metis#45):** `sweeper.resample.cv.k` is the
  ESTIMAND knob (outer fold count + inner default — the #42 principle); optional `inner_k`
  (>=2) overrides the INNER per-config CV only (selection precision/cost — `10×72×inner_k:5`
  halves the decision grid vs k:10 inner). One accessor (`CVResample.InnerFolds()`) feeds the
  nested inner passes, the partition ref, totals, and banners; the OUTER level (split dirs,
  driver, held-out scoring partition — the #23 determinism invariant) never reads it. FLAT
  runs ignore it loudly (their CV IS the reported estimate). `--sample in<N>` (metis#58)
  prefix-subsamples this inner enumeration WITHOUT touching the inner_k partition (splitK/runK
  seam — cache-continuous escalation); `--fast` stays outer-only. The adaptive racing sampler
  over the same budget is the filed follow-up.
- **Path is location, never identity (metis#34):** run ids, point-addresses, and the sweep
  identity are content-addressed (`shapeBlobHash`/`PointAddress`/`shapeRunIdentity` — no
  path-string term); output anchors are `Abs(Dir(expPath))`-derived and the ledger sidecar sits
  next to the shape file — so `metis run`/`select` are cwd-independent by construction (pinned
  by `TestRun_CwdIndependentIdentityAndLocation`). The bare-repo steppath fallback anchors on
  the SHAPE's repo, never cwd; `kaggle submit -C <pipeline-dir>` anchors `runs/` from any cwd.
- **External-ingest identity (metis#25) — declared content pins:** the interior is input-addressed
  (`Kpre` + transitive-D) and upstream artifacts are class-1 keyed, but a ROOT step ingesting
  EXTERNAL data (today: `kaggle/download`, a remote fetch) has unknowable content at key time.
  The rule (Nix fixed-output-derivation model): **the shape declares the expected content in the
  step's `with.sha256` map** — it rides the existing `with → Kpre` channel (a pin edit re-keys
  the whole downstream; first run after = cold + new cohort, by design), and the step VERIFIES
  post-download (mismatch/missing/extra = loud step failure; verify lives in
  `kaggle/cmd/kaggle-download/pins.go`, contract files excluded mirroring `collectArtifacts`).
  Unpinned ingest is LOUD (a paste-ready pin block on stderr), never silent. Dual-use shapes
  driven by both live CLI and the hermetic e2e's fixture data stay unpinned (one static block
  can't satisfy two truths); a future local-file get-data root uses the same declared-pin rule.
- **Default leaf BLAS pins (metis#48) — `cmd/metis/blaspins.go`:** the parallelism budget belongs
  to the ORCHESTRATOR (the #31 semaphore), so `runExperiment` computes the four single-thread pins
  (`OMP/OPENBLAS/VECLIB/MKL_NUM_THREADS=1`) ONCE per top-level run — minus any name the operator
  exported (an explicit value always wins) — announces one loud note, and injects them at BOTH
  spawn seams: the legacy `execStep` child env and the fork-server process env (children inherit).
  `metis select --promote` is deliberately unpinned (serial single all-data fit — multi-threaded
  BLAS wanted). Env is outside run identity by design (`Kpre` = {step_id, uses, with, seed,
  upstream}; HIT-validation re-hashes read-set D; fingerprint is git state), so pins perturb no
  cache key or fingerprint.
- **Warm fork-server leaf executor (metis#44) — `metis/forkserver.py` + `cmd/metis/forkexec.go`:**
  kills the per-leaf `uv run → fresh python → import pandas/sklearn` tax (~1s measured/spawn, ~5k
  spawns/sweep). One warm server per **project root** (metis's and kbench's venvs differ), started
  lazily as `uv run --project <root> python -m metis.forkserver`; it preloads **third-party only**
  (D is the child's first-party sys.modules snapshot — preloading first-party would widen every
  step's D; a delta rule would under-capture it → stale hits; the zygote-per-module tier is the
  documented future extension) and serves JSONL requests: per step it **forks (main thread only)**
  — the child scrubs `METIS_*`, applies the request env (absence is authoritative — the seal),
  chdirs, pipes its output, runs `metis.trace.run_traced(module, force_site_packages=True)`
  (forced because a warm child never OBSERVES the site-packages reads that set the uv.lock dep
  flag), and `os._exit`s. Every step keeps its own process/reads.json/crash boundary; the metis#31
  leaf semaphore is held around the in-flight fork exactly as around a legacy spawn. Routing:
  `execStep.Execute` parses the two-repo wrapper convention (`parseWrapper` — the ROOT line + the
  `uv run … metis.trace <module>` exec line + a pyproject.toml at the derived root); non-conforming
  wrappers and failed/dead servers fall back to the legacy subprocess LOUDLY, once per uses-type/
  root. `metis run --forkserver=false` is the escape hatch. Step authoring is UNCHANGED (a step is
  still a wrapper file; discovery/resolve untouched).
- **Step entrypoints — `metis/steps/{cv_split,train,predict,outer_split}.py`:** thin `io → pure core → io`.
  - `cv-split`: load Dataset (`with.dataset`, exp-relative) → `cv_folds` → `folds.json` + `{k,n}`.
  - `train`: load Dataset + upstream `folds.json` → `cv_score` + fit-on-all → `model.pkl` + `{cv_score}`.
    `with.metric` (metis#59, optional, default `accuracy`) picks the scorer on BOTH paths; validated
    EAGERLY at entry so an unknown metric fails loudly even on the foldless ship refit (which never
    scores). `with.decide` (metis#60, optional, default `argmax`) selects the decision rule the same
    eager-loud way; the ship refit persists `offsets.json` (+classes) and `predict` validates + applies
    it, ALWAYS emitting `probabilities.csv` (class-labeled columns — blend/diagnostics material).
    Setting either key re-keys the leaf address (Kpre hashes the resolved With); absent keys =
    existing cohorts untouched, argmax behavior byte-identical.
  - `predict`: load Dataset + upstream `model.pkl` → predict test rows → `predictions.csv` + `{n_predictions}`.
  - `outer-split` (metis#23, L1 structural seal): read the FULL dataset (**unconfined** — it must
    see all rows to split them) → `cv_folds` → k `analysis_i/` **subset dataset dirs** (train where
    `outer_fold != i`; assessment rows physically absent; the test frame CARRIED through, metis#35 —
    `analysis_i` is a SHAPE-IDENTICAL stand-in for the declared base, only train rows differ, so
    both-frames features see the same test rows sealed as at ship) + `outer_folds.json`. The sealing
    spine **#20 (leakage-safe features) + kbench#8 (ticket-group survival) inherit.**
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
