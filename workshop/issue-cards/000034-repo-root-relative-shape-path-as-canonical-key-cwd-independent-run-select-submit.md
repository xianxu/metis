---
id: '000034'
status: done
started: 2026-07-17T17:12:38-07:00
created: 2026-07-13
updated: 2026-07-17
estimate_hours: 0.38
actual_hours: 0.35
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
