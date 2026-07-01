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

## Revisions

### 2026-07-01 — M1 milestone review (FIX-THEN-SHIP → fixed)

The post-M1 boundary review (sidecar: `workshop/plans/…-m1-review.md`) found the Task-3 enforcement merge-check defective; addressed before crossing the boundary:

- **C1 (contract + silent-swallow).** The planned hook (`base="${MERGE_CHECK_BASE:-origin/main}"` + hardcoded `HEAD`, scanned via `< <(git diff …)`) ignored `run-merge-checks.sh`'s `<base> <head>` positional args, and silently passed when the base didn't resolve (the `git diff` failure was swallowed by `set -e` inside the process substitution). Rewrote `experiment-validate.sh` to consume `$1`/`$2` and assign the changed-file list to a variable first, so an unresolvable base aborts loudly. Verified: `HEAD HEAD` → exit 0 (scopes nothing), bad base → `fatal: bad revision` exit 128, detection intact → exit 1.
- **I1 (no automated fixture test).** Added `scripts/merge-checks.d/experiment-schema-selftest.sh` — an always-run (diff-independent) merge-check asserting `valid-baseline` → 0 and `invalid-bad-status` → 1, so a schema regression is caught even though `experiment-validate.sh` skips `testdata/`.
- **Minor.** The frontmatter probe now parses the `---` fenced block (robust to reordered fields), superseding the earlier `head -5`/`head -8` snippets in Task 3.

### 2026-07-01 — M2 boundary review (REWORK → FIX-THEN-SHIP → fixed)

M2's Go step-runner needed two review cycles before crossing:
- **REWORK (Critical).** The subprocess executor injected `METIS_STEP_DIR`/`METIS_RUN_DIR` as *relative* paths while setting the child cwd to that same dir, so every step failed under the natural `metis run <relative-path>` invocation — masked by an e2e test feeding an absolute `t.TempDir()` path. Fixed: absolutize `runDir` at the runner boundary + `TestRunExperiment_RelativePath` (chdir + bare filename). Also excluded `metrics.json` from the `run.json` artifact set, mapped the step-executable contract into `atlas/experiment.md` (`## Surface (M2)`), and deduped the doubled step-id error prefix.
- **FIX-THEN-SHIP (2 Important).** (I1) `Validate`/`TopoSort` falsely reported a cycle for a duplicated predecessor (`needs: [a, a]`) — in-degree counted per edge, relaxation per dependent step; fixed by deduping each step's `needs` into a set used for both (self-edges dropped). (I2) added an e2e for the `status:"failed"` ledger-on-failure branch (a `with:{fail:true}` step → `run.json` `failed` + `## Runs` bullet).
- Minor items (non-recursive `collectArtifacts`, `#Run` CUE drift-guard, `findRepoRoot`-3×) deferred to M3.

## Chunk 2: M2 — Go step-runner (supersedes the M2 sketch in "Later milestones")

**Goal:** `metis run <experiment>` reads an experiment, validates it (the semantic checks M1 deferred), executes its steps in dependency order as subprocesses, and records a Run — the Go control plane over the (M3) Python data plane, **files + subprocess** between them. This is where "CUE-validated" becomes "actually runnable."

### Core concepts

#### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Experiment` / `Step` / `Run` (Go structs) | `pkg/experiment/experiment.go` | new |
| `Parse` | `pkg/experiment/experiment.go` | new |
| `Validate` | `pkg/experiment/validate.go` | new |
| `TopoSort` | `pkg/experiment/validate.go` | new |
| `Runner.Run` (orchestration) | `pkg/experiment/run.go` | new |

- **Experiment/Step/Run** — Go structs mirroring the CUE `#Experiment`/`#Step`/`#Run`. **ARCH-DRY tension (name it):** these restate the CUE shape. Mitigation: a test round-trips `testdata/experiment/valid-baseline.md` through `Parse` **and** asserts `vocabulary validate-instance` still accepts it — so the Go structs cannot silently drift from the CUE source. CUE stays the single *structural* source; Go adds only what CUE can't express.
- **Parse(content) (Experiment, error)** — reuse ariadne `pkg/frontmatter.Split` (ARCH-DRY — do **not** re-implement fence parsing) + `yaml.v3` unmarshal of the frontmatter. Pure.
- **Validate(Experiment) error** — the semantics M1 deferred (ARCH-PURPOSE): every `needs` id resolves to a real step, `uses` matches `^[a-z0-9-]+/[a-z0-9-]+$`, the graph is acyclic. Pure; returns a joined error listing all violations (`errors.Join`).
- **TopoSort(Experiment) ([]Step, error)** — Kahn's algorithm over `needs`; execution order or a cycle error. Pure. Validate calls it for the acyclicity check (one implementation — ARCH-DRY).
- **Runner.Run** — orchestrates Validate → TopoSort → execute-each → assemble Run. The ordering + wiring is pure/thin; the actual step execution is injected (below), so `Runner.Run` is unit-tested **with a fake executor, no subprocess** (the ARCH-PURE line).

#### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `StepExecutor` (interface) | `pkg/experiment/run.go` | new | the seam |
| `execStep` (subprocess impl) | `cmd/metis/exec.go` | new | `os/exec` |
| run-ledger writer | `pkg/experiment/run.go` | new | filesystem `runs/<id>/` |

- **StepExecutor** — `Execute(step Step, dir string) (StepResult, error)`. The injected seam: `Runner.Run` takes a `StepExecutor`, so orchestration is testable with a fake; the real subprocess executor is the thin `cmd/metis` impl.
- **execStep (cmd/metis)** — resolves `uses: <layer>/<steptype>` to an executable (M2 convention: `steps/<layer>/<steptype>` under the repo, `$METIS_STEP_PATH` override), invokes it with the run dir + the step's `with` config (a `with.json` in the step dir), captures exit + reads the step's `metrics.json`. Real step-types (`metis/cv-split` …) arrive in M3; M2 ships a **process-level fake step** (`testdata/steps/echo`) to exercise the real executor end-to-end (per the "model external services with a process-level fake" rule).
- **run-ledger writer** — writes `runs/<id>/run.json` (`#Run` shape) + appends a `## Runs` line to the experiment. Thin IO.

### Tasks (TDD — bite-sized; mirror M1's per-task commit rhythm)

- **Task 1 — module.** Create `go.mod` (`module github.com/xianxu/metis`, `go 1.26`, `replace github.com/xianxu/ariadne => ../ariadne`). Verify `go build ./...` clean. Commit.
- **Task 2 — types + Parse.** `pkg/experiment/experiment.go`: `Experiment`/`Step`/`Run` + `Parse` (frontmatter.Split + yaml.v3). Tests: parse `valid-baseline` → expected struct; **CUE-conformance round-trip** guard. Commit.
- **Task 3 — Validate + TopoSort.** `pkg/experiment/validate.go`. Table-driven pure tests: dangling `needs` → err; cycle → err; bad `uses` → err; valid → topo order. Add `testdata/experiment/invalid-{cycle,dangling-needs}.md`. Commit.
- **Task 4 — Runner + fake executor.** `pkg/experiment/run.go`: `StepExecutor` interface + `Runner.Run`. Test a 2-step experiment with a fake executor: both run in order, Run assembled, no subprocess. Commit.
- **Task 5 — cmd/metis run + real subprocess.** `cmd/metis/{main,run,exec}.go` + `steps/`-resolution + `testdata/steps/echo` fake step. E2e test: `metis run <fixture>` → exit 0, `runs/<id>/run.json` written, `## Runs` appended. Commit.
- **Task 6 — execution-time enforcement.** The runner Validates on read (semantic checks enforced at run time); note in the plan/atlas that this closes the M1 SHAPE-only gap. Commit.
- **Task 7 — milestone-close M2.** Atlas update (`pkg/experiment` + runner surface, the `steps/` subprocess contract); `sdlc actual`; `--verified`; the boundary review. Fix Critical/Important before crossing.
