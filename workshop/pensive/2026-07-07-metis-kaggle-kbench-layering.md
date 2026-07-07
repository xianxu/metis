---
status: active
type: pensive
created: 2026-07-07
---

# metis / kaggle / kbench — the layer roles (run is metis's; step-path is a dep-walk)

Clarified with the operator 2026-07-07 (after I mis-layered a `krun submit` / `krun in kaggle`
proposal). Source-of-truth for **metis#16** (step-discovery) + **kaggle#5** (submit CLI).

## The corrected model (operator's words)

- **metis = the ML workflow ENGINE.** It defines what a workflow *is* (`experiment` /
  `experiment-shape`) and **runs** it (`metis run`) + `ledger` / `promote`. *Running is metis's job,
  full stop.* To run, it resolves step *implementations* from a step-path.
- **kaggle = Kaggle STEP-TYPES + thin CLI.** Contributes `kaggle/download`, `kaggle/submit` (steps
  metis runs) + thin `kaggle`-CLI wrappers (competition lookup, ad-hoc submit). Steps + commands, not
  a runner.
- **kbench = a WORKSPACE.** A container of the competitions/experiments a user wants to run. Not a
  code layer with its own CLI — just where pipelines + titanic-specific steps live.

## The key realization: `krun` is a hardcoded stand-in for dependency resolution

Today kbench's `bin/krun` hardcodes `METIS_STEP_PATH="…/metis/steps:…/kaggle/steps:…/kbench/steps"`
and execs `metis run`. Its *only* real job is assembling that step-path — and that is exactly the
**dependency chain**: kbench → kaggle → metis, each contributing a `steps/` dir. So "which steps are
available" is dependency resolution — **the same transitive layer-walk `weave` already does for
skills** — not a per-workspace hand-list. `metis run`, executed in a workspace, should walk that
chain itself (reuse the `construct/` dep source weave reads) → any workspace just runs `metis run`,
no wrapper, no `METIS_STEP_PATH` (which stays an override). `krun` then collapses to `metis run` (or
disappears).

## Mis-layerings I proposed and corrected

- **`krun run` as a kaggle CLI** — wrong; running is `metis run`. The run verb never moves off metis.
- **`krun submit`** — wrong; submit is a **kaggle step** (`kaggle/submit`, invoked by `metis run`ning
  a pipeline that has it) OR a **thin `kaggle submit` CLI** for the ad-hoc case. metis stays
  domain-agnostic. → kaggle#5.

## → Issues

- **metis#16** — `metis run` discovers step layers from the dep graph (the weave layer-walk); no
  `METIS_STEP_PATH` wrapper; `krun` collapses (a kbench follow-up).
- **kaggle#5** — thin `kaggle submit --run <id>` (+ poll `public_score`), reusing `internal/kagglecli`
  + the `kaggle/submit` step logic (one auth/submit path).
