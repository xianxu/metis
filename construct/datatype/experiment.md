---
type: type
name: experiment
description: Use when creating or editing a runnable ML experiment — a git-tracked, reproducible pipeline of steps plus a runs log. Triggers on "create an experiment", "author a pipeline", "save this recipe", editing markdown with `type: experiment`, "/xx-datatype experiment". The reproducible unit of ML work in metis/kbench; the Go step-runner (`metis run <id>`) executes it. Distinct from issue (a work item) and project (a portfolio) — an experiment is a machine-re-runnable recipe, not a conversation log.
---

# experiment

An experiment is the **reproducible unit of ML work** — a git-tracked, declarative pipeline of steps. It is *issue-shaped*: schematized frontmatter (the machine-executable pipeline + config) over a freeform body (hypothesis + notes). It is **immutable input** (#13): the runner never writes back into the `.md`, so a committed config is a stable content-hash. The Go step-runner (`metis run <id>`) reads it, runs the steps in dependency order, and writes the run record to `runs/<id>/` — so the experiment reconstructs data, retrains, and regenerates a submission **with no agent in the loop**.

The `experiment` *type* and its runner are owned by **metis** (platform-independent). *Instances* live in a competition workspace in a downstream repo — e.g. `kbench/competition/titanic/pipelines/titanic-baseline.md` — so they're editable like any other markdown (parley.nvim, etc.).

Distinct from siblings:
- `issue` — a unit of *work* that exists regardless of timing. An experiment is a *runnable recipe*.
- `project` — the operator's portfolio view. An experiment is a single line of inquiry.

## Frontmatter shape

The frontmatter is validated structurally against `#Experiment` (`vocabulary validate-instance --type experiment <file>`). Semantic checks (DAG acyclicity, `needs` resolution, `uses` format) are enforced by the runner at read time.

| Field | Required | Notes |
|---|---|---|
| `type` | yes | `experiment` |
| `id` | yes | Slug, lowercase-hyphenated. Matches the filename without `.md`. |
| `competition` | optional | The competition slug this instance belongs to (set on kbench instances; absent on generic metis fixtures). |
| `seed` | yes | Integer seed — the reproducibility anchor. Every stochastic step derives from it. |
| `status` | yes | `draft` \| `active` \| `archived`. |
| `steps` | yes | The pipeline — an ordered list of step objects. See *Step structure*. |

### Step structure

Each entry in `steps` is one node of the pipeline:

- `id` — step identifier, unique within the experiment.
- `uses` — the step type as `"<layer>/<steptype>"` (e.g. `metis/cv-split`, `kaggle/download`, `titanic/adapt`). Each layer contributes step types; the pipeline just wires them.
- `needs` — optional list of step `id`s this step depends on. These are the DAG edges; the runner topo-sorts them. Every id must resolve to a real step, and the graph must be acyclic.
- `with` — optional config map, interpreted by the step type (e.g. `{k: 5, stratify: Survived}` for `metis/cv-split`).

## Run history (lives outside the config)

The experiment `.md` is **immutable input** — the runner does **not** write run history into it (#13). Each run's full record lives under `runs/<run-id>/record.json` (params, seed, metrics, artifacts; conforms to `#Run`); a sweep also accumulates rows in the `<shape>.ledger.csv` sidecar. "Which parameters produced which score" is answered by `metis ledger show` over that sidecar. Records accumulate, never overwrite.

## Authoring instructions

When the dispatcher applies this prototype:

1. **Distill before asking.** Pull the experiment's intent (hypothesis), the steps, and their `uses`/`with` from the conversation. Pre-fill what's clear; ask only for what's missing.
2. **Resolve the required fields:** `id` (slug, usually obvious), `seed` (pick one and record it — it's the reproducibility anchor), `status` (new experiments start `draft` or `active`), and the `steps` list.
3. **Name each step and its `uses`.** For each step, decide its `id`, its `uses: <layer>/<steptype>`, its `needs` (which earlier steps it consumes), and its `with` config. Keep the graph a DAG — a step only `needs` steps declared before it makes the intent readable.
4. **Body = hypothesis + notes.** Write a one-paragraph hypothesis/lede. Do **not** add a `## Runs` section — the config is immutable input; run history lives in `runs/<id>/` + the ledger sidecar (browse via `metis ledger show`), never written back into the `.md`.
5. **Default location:** a competition workspace in the consuming repo, `competition/<slug>/pipelines/<id>.md`. Generic metis fixtures live under `testdata/experiment/`.
6. **Confirm before writing:** show `id`, the step list (`id — uses — needs`), and the destination path. One round of confirmation.

## Rules

- One experiment per file. Slug, filename, and `id:` must agree.
- `steps` form a **DAG** — step `id`s are unique, every `needs` entry resolves to a real step, and there are no cycles. (Structural shape is CUE-validated; the DAG/resolution checks are enforced by the runner.)
- `uses` is always `<layer>/<steptype>` — the step type must be contributed by a layer in the repo's substrate chain.
- Runs accumulate in `runs/<id>/` + the ledger sidecar (never in the config `.md`); never overwrite a recorded run.
- An experiment is a *recipe*, not a transcript. If you find yourself narrating what an agent did, that belongs in an issue `## Log`, not here.
