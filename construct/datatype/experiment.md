---
type: type
name: experiment
description: Use when creating or editing a runnable ML experiment — a git-tracked, reproducible pipeline of steps plus a runs log. Triggers on "create an experiment", "author a pipeline", "save this recipe", editing markdown with `type: experiment`, "/xx-datatype experiment". The reproducible unit of ML work in metis/kbench; the Go step-runner (`metis run <id>`) executes it. Distinct from issue (a work item) and project (a portfolio) — an experiment is a machine-re-runnable recipe, not a conversation log.
---

# experiment

An experiment is the **reproducible unit of ML work** — a git-tracked, declarative pipeline of steps plus its execution history. It is *issue-shaped*: schematized frontmatter (the machine-executable pipeline + config) over a freeform body (hypothesis, notes, and an accreting `## Runs` log). The Go step-runner (`metis run <id>`) reads it, runs the steps in dependency order, and appends a run record — so the experiment reconstructs data, retrains, and regenerates a submission **with no agent in the loop**.

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

## Runs convention

The body carries a `## Runs` section — the execution ledger, appended by the runner (one line per run: id, timestamp, key metrics, artifact dir). Full run records live under `runs/<run-id>/` (params, seed, metrics, artifacts) and conform to `#Run`. Runs are the answer to "which parameters produced which score"; they accumulate, never overwrite.

## Authoring instructions

When the dispatcher applies this prototype:

1. **Distill before asking.** Pull the experiment's intent (hypothesis), the steps, and their `uses`/`with` from the conversation. Pre-fill what's clear; ask only for what's missing.
2. **Resolve the required fields:** `id` (slug, usually obvious), `seed` (pick one and record it — it's the reproducibility anchor), `status` (new experiments start `draft` or `active`), and the `steps` list.
3. **Name each step and its `uses`.** For each step, decide its `id`, its `uses: <layer>/<steptype>`, its `needs` (which earlier steps it consumes), and its `with` config. Keep the graph a DAG — a step only `needs` steps declared before it makes the intent readable.
4. **Body = hypothesis + Runs.** Write a one-paragraph hypothesis/lede. Leave a `## Runs` heading for the runner to append to. Do not hand-write run records — `metis run` produces them.
5. **Default location:** a competition workspace in the consuming repo, `competition/<slug>/pipelines/<id>.md`. Generic metis fixtures live under `testdata/experiment/`.
6. **Confirm before writing:** show `id`, the step list (`id — uses — needs`), and the destination path. One round of confirmation.

## Rules

- One experiment per file. Slug, filename, and `id:` must agree.
- `steps` form a **DAG** — step `id`s are unique, every `needs` entry resolves to a real step, and there are no cycles. (Structural shape is CUE-validated; the DAG/resolution checks are enforced by the runner.)
- `uses` is always `<layer>/<steptype>` — the step type must be contributed by a layer in the repo's substrate chain.
- Runs accumulate in `## Runs` + `runs/<id>/`; never overwrite a recorded run.
- An experiment is a *recipe*, not a transcript. If you find yourself narrating what an agent did, that belongs in an issue `## Log`, not here.
