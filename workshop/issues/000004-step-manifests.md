---
id: 000004
status: open
deps: []
github_issue:
created: 2026-07-02
updated: 2026-07-02
estimate_hours:
---

# Collocated step manifests + a generated single step reference (declare inputs/outputs/knobs)

## Problem

There is no learner-facing catalog of step-types, and the step **contract lives
implicitly in code**. Concretely:

- A step's `with` keys aren't self-describing: whether a value is an experiment
  path, an upstream-step reference, or a literal is decided only by what the
  step's code does with it (`exp_path` vs `upstream_path` vs literal). You must
  read the code to know.
- A step's **output filenames are undeclared**. `get-data` emits `train.csv`/
  `test.csv` only because `adapt` hard-codes those names in `upstream_path(...,
  "train.csv")`. The producer/consumer agree by convention, declared nowhere.
- Docs are scattered by layer (metis steps in metis atlas, kaggle in kaggle atlas,
  titanic in kbench atlas) and engineer-facing (the contract), not a "here are all
  the steps, what each does, its knobs, which are interesting to tinker" reference.

## Spec

Give each step a small **collocated structured manifest** and **generate** the
single reference from them (single-source; the manual derives, doesn't drift — the
same instinct as SKILL.md / the typed-document work).

- **Manifest per step** (next to the executable, e.g. `steps/<ns>/<name>.md` with a
  schematized frontmatter block or a sidecar): `summary`, `inputs` (each with
  **kind**: `exp-path` | `upstream-ref` | `literal`, + type), `outputs` (declared
  filenames + what they are), `knobs` (the tunable `with` keys + defaults/choices),
  and a freeform `learn-notes` (why it matters, what to tinker). This also fixes
  the implicit producer/consumer filename coupling (outputs become declared).
- **Generated single reference** — a command (or merge-check) that concatenates all
  manifests on the step path into one catalog (`metis/atlas/steps.md` or similar),
  grouped by layer, flagging the learner-interesting steps. Consumers derive from
  the manifests; no hand-maintained restatement.
- **Agent-legible:** the manifest is what lets an agent invoke a step correctly
  (pairs with a future `metis run-step`) and what a learner reads first.

## Done when

- Each existing step (`kaggle/download`, `kaggle/submit`, `titanic/adapt`,
  `titanic/submission`, `metis/cv-split|train|predict`) has a manifest declaring
  inputs(+kind)/outputs/knobs/learn-notes.
- A generator produces the single catalog from the manifests; running it is
  covered by a test (fixture manifests → expected catalog).
- The manifest schema is documented (ideally a `construct/vocabulary/` type so it's
  validated like experiments).
- atlas points at the generated catalog as the step reference.

## Plan

- [ ] Design the manifest schema (fields incl. input `kind`, declared outputs); model it in construct/vocabulary if apt.
- [ ] Author manifests for the 7 existing steps (spans metis/kaggle/kbench — coordinate; the schema + generator are metis's).
- [ ] Generator: manifests on the step path → single catalog; test against fixtures.
- [ ] atlas: reference the generated catalog; note the input-kind + declared-output convention.

## Log

### 2026-07-02
- Filed at operator request from the kbench#1 walkthrough. Motivated by three concrete confusions: `with`-value kind is implicit, output filenames are undeclared (get-data↔adapt coupling), and there's no single learner reference. Spans layers (manifests live beside each step); the schema + generator are metis-owned. Pairs with a future `metis run-step` for single-step invocation.
