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

## Ownership & instances

The type + (M2) runner are **metis's** — platform-independent. *Instances* live in a
downstream **competition workspace** — `kbench/competition/<slug>/pipelines/<id>.md` — not
in metis; metis carries only test fixtures. Each layer contributes step types
(`metis/cv-split`, `kaggle/download`, `titanic/adapt`); a pipeline wires them.

## Validation split (why two validators)

CUE owns **shape** — types, enums, required fields, the `steps` list-of-structs. The
**semantic** checks — `needs` resolves to a real step id, the graph is acyclic, `uses` is
`<layer>/<steptype>` — are not expressible in `cue vet` and land with M2's **pure Go
validator** (`pkg/experiment`), which the runner invokes on read (execution-time
enforcement). This is the ARCH-PURE seam: M1 is declarative config validated by a
subprocess; the pure logic arrives with the runner.
