# experiment — the reproducible unit of ML work

An experiment is a git-tracked, declarative **pipeline of steps plus its run history** —
*issue-shaped*: schematized frontmatter (the machine-executable pipeline + config) over a
freeform body (hypothesis + an accreting `## Runs` log). The Go step-runner
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

`metis run [--run <id>] <experiment.md>` reads + validates an experiment, executes its
steps in dependency order as **subprocesses** (files + subprocess, never FFI), and records
a Run. Split across a pure core and a thin IO layer:

- **Pure core — `pkg/experiment/`** (no IO; unit-tested directly):
  - `Experiment` / `Step` / `Run` — Go structs mirroring the CUE `#Experiment`/`#Step`/`#Run`
    (the CUE stays the single *structural* source; a conformance test guards against drift).
  - `Parse(content) (Experiment, error)` — reuses ariadne `frontmatter.Split` + `yaml.v3`.
  - `Validate(Experiment) error` — the semantic checks CUE can't express (unique ids, `needs`
    resolution, `uses` = `^[a-z0-9-]+/[a-z0-9-]+$`, acyclicity); joins all violations.
  - `TopoSort(Experiment) ([]Step, error)` — Kahn's algorithm; the one acyclicity impl.
  - `Runner.Run(exp, runID, runDir)` — orchestrates Validate → TopoSort → execute-each →
    assemble the `Run`. Step execution is injected via the `StepExecutor` interface, so the
    orchestration is fake-executor tested with **no subprocess** (the ARCH-PURE line).
- **Thin IO — `cmd/metis/`:** `execStep` (the real `os/exec` `StepExecutor`) + the run-ledger
  writer. `runDir` is absolutized at this boundary so step paths resolve from any cwd.

### Step-executable contract (what M3 step-types must honor)

The runner invokes one executable per step, resolved from `uses: <layer>/<steptype>` to
`<stepdir>/<layer>/<steptype>` on the **step path** — `$METIS_STEP_PATH` (colon-separated)
if set, else `<repo-root>/steps`; first existing file wins.

- **Working dir:** `runs/<run-id>/<step-id>/`, created by the runner; the child runs with
  its **cwd set to this dir**.
- **Env:** `METIS_STEP_DIR` (that dir, absolute), `METIS_RUN_DIR` (the run dir, absolute),
  `METIS_STEP_ID`.
- **In:** `with.json` — the step's `with` config, written into the step dir by the runner.
- **Out:** an optional `metrics.json` (flat `{name: number}`, merged into `Run.metrics`) plus
  any **artifact files** the step writes into its dir. `with.json` and `metrics.json` are the
  reserved contract channels and are NOT counted as artifacts; every other file is recorded in
  `Run.artifacts` as a `runs/<id>/`-relative (step-qualified) path. A non-zero exit fails the
  step and halts the run.
- **Ledger:** `runs/<run-id>/run.json` (the `#Run` record — the record of truth) + a one-line
  summary appended to the experiment's `## Runs` section. A run rejected at validation time
  writes neither. M2 ships a process-level fake step (`testdata/steps/test/echo`) exercising
  this contract end-to-end; real `metis/*` step-types arrive in M3.

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
