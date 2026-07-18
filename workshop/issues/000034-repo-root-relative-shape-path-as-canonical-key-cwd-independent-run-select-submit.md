---
id: 000034
status: working
deps: []
github_issue:
created: 2026-07-13
updated: 2026-07-17
estimate_hours:
started: 2026-07-17T17:12:38-07:00
---

# repo-root-relative shape path as canonical key (cwd-independent run/select/submit)

## Problem

The user works one pipeline at a time and naturally sits **inside** that pipeline dir (e.g.
`competition/titanic/pipelines/`), invoking `metis run titanic-sweep.md`. But the shape path is used as
an identity/output anchor, so behavior can drift with cwd. We want the **repo-root-relative path** to be
the canonical key regardless of where `metis` is invoked from: `titanic-sweep.md` (from the pipeline dir)
and `competition/titanic/pipelines/titanic-sweep.md` (from the repo root) must resolve to the **same**
canonical key `competition/titanic/pipelines/titanic-sweep.md`. Then `metis run`, `metis select`, and
`kaggle submit` are all cwd-independent and consistently rooted in the pipeline dir. (Split out of metis#32's
brainstorm — orthogonal to the selection algebra.)

## Spec

- Resolve any passed shape path → repo-root-relative canonical form (walk up to the repo root; error clearly
  if outside a repo). Use that canonical string wherever the path is an identity/anchor term.
- Verify it does NOT perturb content-addressing (the point-address is the shape's *blob-hash* + config, not
  its path — confirm cwd-independence holds end-to-end for run dirs, the ledger sidecar location, and the
  record's slug rooting so `kaggle submit` resolves consistently).
- `metis run titanic-sweep.md` (from pipeline dir) and `metis run competition/titanic/pipelines/titanic-sweep.md`
  (from repo root) produce identical run identities + land outputs in the same place.

## Done when

- Invoking `metis run` / `metis select` on the same shape from the pipeline dir vs. the repo root yields
  identical run ids, ledger location, and output dirs (a cwd-independence test).
- `kaggle submit --run <id>` resolves consistently regardless of the cwd `metis run` was invoked from.

## Plan

- [ ] Brainstorm/spec the resolution point (where the path is canonicalized) + audit every path-as-identity
  use; then change-code.

## Log

### 2026-07-13
- Filed from the metis#32 brainstorm (operator): "use the relative-to-repo-root path as the full key … so
  the user can `metis run titanic-sweep.md` from inside the pipeline dir and `metis select titanic-sweep.md
  --best` consistently, and kaggle submit stays rooted in the pipelines dir." Orthogonal to #32's selection
  algebra → split out.
