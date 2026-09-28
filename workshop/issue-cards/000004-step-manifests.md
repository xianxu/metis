---
id: 000004
status: open
created: 2026-07-02
updated: 2026-07-02
estimate_hours:
github_issue:
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
