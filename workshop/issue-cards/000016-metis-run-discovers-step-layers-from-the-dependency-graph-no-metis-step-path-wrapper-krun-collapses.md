---
id: '000016'
status: done
started: 2026-07-06T22:32:07-07:00
created: 2026-07-06
updated: 2026-07-06
estimate_hours: 0.94
actual_hours: 0.56
---

# metis run discovers step layers from the dependency graph — no METIS_STEP_PATH wrapper (krun collapses)

## Problem

Running a workflow is **metis's job** (`metis run`), but today metis can't find the step
*implementations* on its own — it relies on `METIS_STEP_PATH` being set for it. That env var is
assembled by a bespoke per-workspace wrapper: kbench's `bin/krun` hardcodes
`METIS_STEP_PATH="$PEERS/metis/steps:$PEERS/kaggle/steps:$KBENCH/steps"`. This mis-layers the model:
- **`run` belongs to metis** (the ML workflow engine); a workspace should not need a wrapper that
  re-implements "which step layers exist".
- The step layers are exactly the **dependency chain**: a workspace (kbench) depends on kaggle,
  which depends on metis; each contributes a `steps/` dir. So "which steps are available" is
  **dependency resolution** — the same transitive layer-walk `weave` already does for skills — not
  something to hand-list per workspace.
