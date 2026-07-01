# metis experiment datatype + step-runner — Implementation Plan (metis#1)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give metis a validated, agent-authorable `experiment` datatype — a pipeline of steps plus a run history — as the reproducible unit of ML work, so M2's Go step-runner and every kbench competition derive from one *enforced* schema.

**Architecture:** The experiment noun is modeled **once** as a CUE vocabulary schema (`#Experiment`/`#Step`/`#Status`/`#Run`) — the single source. Consumers derive from it: the `xx-datatype` authoring skill (via a datatype prototype), the `vocabulary validate-instance` structural validator (via `cue vet`), and — in M2 — the Go runner's types. This milestone (M1) delivers the schema + prototype + fixtures + *enforced* validation; M2 adds the Go parser/runner; M3 adds the Python data-plane step-types. It reuses ariadne's `datatype` + `vocabulary` compilers **unchanged** (ARCH-DRY): metis only adds source files to its own layer tree, and the inherited dynamic-skill markers DAG-merge them (leaf-wins).

**Tech Stack:** CUE (schema + `cue vet`), the ariadne `datatype`/`vocabulary` binaries (inherited via the layer graph), `weave` (skill regeneration), metis merge-check hooks (enforcement). Go arrives in M2; Python in M3.

**Milestone boundary note:** metis#1 has 3 review boundaries (M1/M2/M3), each closing via `sdlc milestone-close`. This plan details **M1**; M2/M3 are sketched at low resolution and get their own detailed planning when reached (re-run `sdlc start-plan` per design).

---

## Core concepts

M1 is **declarative** — its conceptual core is the schema, not code. The pure-function core (parse + semantic validate) arrives in **M2** with the runner. That is deliberate and ARCH-PURE-clean: M1 is configuration + a structural validator invoked as a subprocess; there is no business logic to bury in IO yet. The `at-plan` ARCH-PURE check ("name what's pure vs the thin IO seam") is satisfied by *there being no pure logic in M1* — the seam is entirely "author files → run the inherited validator binary."

### Pure entities (the schema — the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `#Status` (CUE) | `metis/construct/vocabulary/experiment.cue` | new |
| `#Step` (CUE) | `metis/construct/vocabulary/experiment.cue` | new |
| `#Experiment` (CUE) | `metis/construct/vocabulary/experiment.cue` | new |
| `#Run` (CUE) | `metis/construct/vocabulary/experiment.cue` | new |
| `experiment` datatype prototype | `metis/construct/datatype/experiment.md` | new |

- **#Experiment / #Step / #Status** — the structural contract for an experiment file's frontmatter: the pipeline (`steps`) + config (`id`, `seed`, `status`, optional `competition`).
  - **Relationships:** `#Experiment` 1:N `#Step` (the pipeline); `#Step.needs` → other `#Step.id` (intra-experiment DAG edges). **Referential integrity + acyclicity + `uses` format are SEMANTIC checks CUE cannot express** — they are deferred to M2's pure Go validator. This `.cue` owns **shape only** (types, enums, required fields, the `steps` list-of-structs). Stated here so the M2 planner knows what M1 intentionally leaves unguarded.
  - **DRY rationale:** one schema; the datatype skill, the `cue vet` validator, and (M2) the Go types all derive from it (ARCH-DRY). First occurrence of the *metis-owned datatype* pattern that kaggle/kbench will follow (`ARCH-DRY`, per the #115 per-repo DAG-merged datatype system).
  - **Future extensions:** `{param}` templating (deferred — the `#Run` record already carries bound values, so it's additive); modality beyond tabular; richer step-`with` typing per step-type.
- **#Run** — the shape of one recorded execution (produced by M2's runner; defined here so the schema is complete): `id`, parent `experiment`, `started`/`finished`, `seed`, `status`, `metrics`, `artifacts`.
  - **Relationships:** N:1 with `#Experiment`.
  - **Future extensions:** per-metric provenance; code-version/commit stamping.
- **experiment datatype prototype** — the human/agent **authoring** form (frontmatter shape + step structure + `## Runs` convention + authoring instructions + rules), compiled into metis's `xx-datatype` skill. Mirrors the structure of `ariadne/construct/datatype/project.md`.

### Integration points (where the schema meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| datatype / vocabulary compile | inherited from ariadne | reused | the DAG-merge compilers, run by `weave` |
| `vocabulary validate-instance` | inherited from ariadne | reused | `cue vet` over extracted frontmatter |
| experiment-validate merge-check | `metis/scripts/merge-checks.d/experiment-validate.sh` | new | `vocabulary validate-instance` over changed experiment files |

- **datatype / vocabulary compile** — `weave compile` runs the inherited `.dynamic-skill` markers with `cwd=metis`, DAG-merging metis's `construct/datatype/*.md` and `construct/vocabulary/*.cue` into `metis/construct/generated/{datatype,vocabulary}` (gitignored). No metis code; we only add source files. (ARCH-DRY — the compilers are reused, not reimplemented.)
- **experiment-validate merge-check** — the **enforcement** seam (ARCH-PURPOSE): a thin shell hook, run by the inherited `scripts/run-merge-checks.sh`, that validates every changed `experiment`-typed markdown file with `vocabulary validate-instance --type experiment`. This is what makes the CUE schema *enforced* rather than decorative — the difference the ARCH-PURPOSE lens demands. metis guards its own fixtures with it now; kbench inherits the same check when it holds real experiments (kbench#1). Execution-time enforcement (the runner validating on read) lands in M2.
  - **Injected into:** the inherited `scripts/run-merge-checks.sh` runner (discovers `scripts/merge-checks.d/*`).
  - **Future extensions:** also invoke M2's Go semantic validator (DAG/needs/uses) once it exists.

---

## Chunk 1: M1 — experiment schema + datatype prototype + enforced validation

### Task 1: CUE vocabulary schema (`#Experiment`) + fixtures

**Files:**
- Create: `metis/construct/vocabulary/experiment.cue`
- Create: `metis/testdata/experiment/valid-baseline.md` (a generic, platform-independent fixture — NOT titanic; that lives in kbench)
- Create: `metis/testdata/experiment/invalid-bad-status.md` (structural violation for the red test)

- [ ] **Step 1: Write the invalid fixture (red).** `metis/testdata/experiment/invalid-bad-status.md` — a well-formed experiment except `status: running` (not in the enum):

```markdown
---
type: experiment
id: invalid-bad-status
seed: 42
status: running
steps:
  - id: a
    uses: metis/cv-split
---
# invalid fixture (bad status enum) — must be REJECTED by cue vet
```

- [ ] **Step 2: Write `experiment.cue`.** The single-source schema — shape only:

```cue
// experiment — the vocabulary of a reproducible ML pipeline experiment.
// An experiment file's YAML frontmatter is validated against #Experiment
// (structural conformance via `cue vet`, invoked by `vocabulary
// validate-instance --type experiment`).
//
// SCOPE: this file owns SHAPE only — types, enums, required fields, the
// steps list-of-structs. SEMANTIC checks (needs → real step id, DAG
// acyclicity, `uses` = "<layer>/<steptype>") are NOT expressible in `cue
// vet` and land with M2's pure Go validator. Closed schema (no `...`) for
// sharp diagnostics — experiment frontmatter is fully known here.
package experiment

#Status: "draft" | "active" | "archived"

#Step: {
	id:   string          // unique within the experiment
	uses: string          // "<layer>/<steptype>", e.g. "metis/cv-split"
	needs?: [...string]   // ids of steps this one depends on (DAG edges)
	with?: {[string]: _}  // free config map; typed per step-type in M3
}

#Run: {
	id:         string                 // run slug, e.g. "run-003"
	experiment: string                 // parent experiment id
	seed:       int
	started:    string                 // ISO datetime
	finished?:  string
	status:     "ok" | "failed"
	metrics?:   {[string]: number}
	artifacts?: [...string]            // repo-relative paths under runs/<id>/
}

#Experiment: {
	type:         "experiment"
	id:           string   // slug; matches filename
	competition?: string   // set on kbench instances; absent on metis fixtures
	seed:         int
	status:       #Status
	steps: [...#Step]      // the pipeline (may be a single step)
}
```

- [ ] **Step 3: Compile the vocabulary.** Regenerate metis's generated vocabulary so `experiment.cue` is picked up:

Run: `cd /Users/xianxu/workspace/metis && /Users/xianxu/workspace/ariadne/bin/weave compile | tail -1`
Expected: `weave: applied N action(s)` with no CUE parse error. Confirm: `ls construct/generated/vocabulary/experiment.json` exists.

- [ ] **Step 4: Write the valid fixture + verify it PASSES (green).** `metis/testdata/experiment/valid-baseline.md`:

```markdown
---
type: experiment
id: valid-baseline
seed: 42
status: active
steps:
  - id: prep
    uses: metis/cv-split
    with: {k: 5}
  - id: train
    uses: metis/train
    needs: [prep]
    with: {model: logreg}
---
# valid-baseline — a generic 2-step fixture proving the schema accepts a well-formed experiment.

## Runs
```

Run: `cd /Users/xianxu/workspace/metis && vocabulary validate-instance --type experiment testdata/experiment/valid-baseline.md`
Expected: PASS (exit 0), no diagnostics.
(Resolve the `vocabulary` binary the same way as weave/sdlc: `/Users/xianxu/workspace/ariadne/bin/vocabulary`, building it once with `cd ../ariadne && go build -o bin/vocabulary ./cmd/vocabulary` if absent.)

- [ ] **Step 5: Verify the invalid fixture is REJECTED.**

Run: `vocabulary validate-instance --type experiment testdata/experiment/invalid-bad-status.md`
Expected: FAIL (non-zero) with a diagnostic like `status: "running" is not valid (want: draft|active|archived)`.

- [ ] **Step 6: Commit.**

```bash
git add construct/vocabulary/experiment.cue testdata/experiment/
git commit -m "#1 M1: experiment CUE vocabulary schema + fixtures"
```

### Task 2: `experiment` datatype prototype

**Files:**
- Create: `metis/construct/datatype/experiment.md`

- [ ] **Step 1: Author the prototype** (mirror `ariadne/construct/datatype/project.md`'s structure: frontmatter `type: type` + `name` + `description`; body = intro, `## Frontmatter shape` table, `### Step structure`, `## Runs convention`, `## Authoring instructions`, `## Rules`). Key `description:` must be trigger-rich (used by the skill index):

```markdown
---
type: type
name: experiment
description: Use when creating or editing a runnable ML experiment — a git-tracked, reproducible pipeline of steps plus a runs log. Triggers on "create an experiment", "author a pipeline", editing markdown with `type: experiment`, "/xx-datatype experiment". The reproducible unit of ML work in metis/kbench; the Go step-runner (`metis run <id>`) executes it. Distinct from issue (work item) and project (portfolio).
---

# experiment
... (frontmatter shape table mirroring #Experiment; step structure mirroring #Step;
    the `## Runs` body-log convention; authoring instructions; rules — one experiment
    per file, filename == id, steps form a DAG, instances live in a competition
    workspace) ...
```

- [ ] **Step 2: Regenerate + verify the skill lists it.**

Run: `cd /Users/xianxu/workspace/metis && /Users/xianxu/workspace/ariadne/bin/weave compile >/dev/null && /Users/xianxu/workspace/ariadne/bin/weave skills | grep -i experiment`
Expected: a line `experiment — Use when creating or editing a runnable ML experiment …`.

- [ ] **Step 3: Commit.**

```bash
git add construct/datatype/experiment.md
git commit -m "#1 M1: experiment datatype prototype (xx-datatype authoring form)"
```

### Task 3: enforced validation (ARCH-PURPOSE)

**Files:**
- Create: `metis/scripts/merge-checks.d/experiment-validate.sh`

- [ ] **Step 1: Write the merge-check** — validate every changed `type: experiment` markdown file against the schema, so the schema is enforced at the merge gate, not just documented:

```bash
#!/usr/bin/env bash
# experiment-validate — fail the merge if any changed experiment file violates
# the CUE schema. Enforcement seam for the experiment datatype (metis#1 M1,
# ARCH-PURPOSE). Discovers changed *.md with `type: experiment` frontmatter and
# runs `vocabulary validate-instance --type experiment` on each.
set -euo pipefail
VOCAB="${VOCAB:-vocabulary}"   # on PATH during `make weave`; else ../ariadne/bin/vocabulary
base="${MERGE_CHECK_BASE:-origin/main}"
fail=0
while IFS= read -r f; do
  [ -f "$f" ] || continue
  head -5 "$f" | grep -q '^type: experiment$' || continue
  if ! "$VOCAB" validate-instance --type experiment "$f"; then fail=1; fi
done < <(git diff --name-only "$base"...HEAD -- '*.md')
exit "$fail"
```

- [ ] **Step 2: Make it executable + run it against the fixtures.**

Run:
```bash
cd /Users/xianxu/workspace/metis && chmod +x scripts/merge-checks.d/experiment-validate.sh
MERGE_CHECK_BASE=$(git rev-list --max-parents=0 HEAD) VOCAB=/Users/xianxu/workspace/ariadne/bin/vocabulary \
  bash scripts/merge-checks.d/experiment-validate.sh; echo "exit=$?"
```
Expected: the valid fixture passes; if the invalid fixture is in the diff, `exit=1`. (This proves the hook actually rejects bad experiments — the enforcement test.)

- [ ] **Step 3: Commit.**

```bash
git add scripts/merge-checks.d/experiment-validate.sh
git commit -m "#1 M1: enforce experiment schema via merge-check (ARCH-PURPOSE)"
```

### Task 4: milestone-close M1

- [ ] **Step 1:** Tick `- [ ] M1` → `- [x]` in `workshop/issues/000001-*.md` `## Plan`; add a `## Log` entry (what shipped, ARCH-* citations, the SHAPE-only/SEMANTIC-deferred boundary).
- [ ] **Step 2:** Run the mandatory boundary review + close the milestone:

Run: `/Users/xianxu/workspace/ariadne/bin/sdlc milestone-close --issue 1 --milestone M1`
Expected: the auto-dispatched fresh-context review runs (window = branch point → HEAD); fix any Critical/Important; the `Review-Verdict:` trailer lands and the milestone-close log line is written.

---

## Later milestones (sketch — detailed-plan each when reached, re-run `sdlc start-plan`)

- **M2 — Go step-runner.** Introduces metis's `go.mod` (`module github.com/xianxu/metis`, `replace github.com/xianxu/ariadne => ../ariadne` to reuse `pkg/frontmatter`). New: `metis/pkg/experiment` — pure types (`Experiment`/`Step`/`Run`) + `Parse(content)` (reuse `frontmatter.Split` + `yaml.v3`) + `Validate(Experiment)` (the SEMANTIC core M1 deferred: `needs`→id resolution, DAG acyclicity via topo-sort, `uses` = `<layer>/<steptype>`) — colocated unit tests, no IO (ARCH-PURE). New: `metis/cmd/metis` with `run <experiment-id>` — reads + validates (execution-time enforcement), topo-orders steps, shells each step-type as a subprocess (files + subprocess, never FFI), appends a `#Run` record + `runs/<id>/` artifacts. Plain streaming output (TUI deferred).
- **M3 — Python data-plane step-types.** `metis` Python package via `uv` + `pyproject`; `Dataset`/`Schema`/`Split` (tabular) + step-types `metis/cv-split`, `metis/train`, `metis/predict`, each a subprocess entrypoint emitting `metrics.json` / `predictions.csv` per the files+subprocess contract the M2 runner reads. Colocated pytest.

## References
- Project: `brain/data/project/kaggle-ml-base-layer.md`
- Issue spec: `metis/workshop/issues/000001-*.md`
- Datatype model to mirror: `ariadne/construct/datatype/project.md`
- CUE + validation machinery: `ariadne/construct/vocabulary/{issue,pensive}.cue`, `ariadne/cmd/vocabulary/` (`validate-instance`), `ariadne/cmd/datatype/`, `ariadne/pkg/frontmatter`
- Known gap in the compile tooling this project already filed: `ariadne#155`
