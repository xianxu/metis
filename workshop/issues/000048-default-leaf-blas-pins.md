---
id: 000048
status: working
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-16
estimate_hours:
started: 2026-07-16T11:10:34-07:00
---

# pin leaf BLAS threads by default — the parallelism budget belongs to the orchestrator

## Problem

Running `metis run titanic-sweep.md` bare (no env pins, default `--parallel`=NumCPU) puts the
sweep into BLAS-oversubscription thrash: NumCPU Python leaves × multi-threaded BLAS each →
load-avg ~7× cores, throughput ≈ 0. Observed on the metis#42 k10 probe (load 83, 885 trains
started / 0 finishing) and AGAIN by the operator on 2026-07-16 — the #38 board's rate line
showed the collapse as a ~3h ETA (the display did its job; the default remains a footgun).
The RUNBOOK's "ALWAYS pin OMP/OPENBLAS/VECLIB/MKL=1" is documentation doing a default's job
— the parallelism budget belongs to the ORCHESTRATOR (the #31 leaf semaphore), not to each
leaf's BLAS. (Deeper-fix candidate already flagged in workshop/lessons.md under the #42
entry; promoted to an issue by the operator's hit.)

## Spec

metis sets single-thread BLAS env for its LEAF subprocesses by default:

- At the leaf spawn seams — legacy `execStep` and the #44 fork-server spawn (the server
  process env; forked children inherit) — inject `OMP_NUM_THREADS=1 OPENBLAS_NUM_THREADS=1
  VECLIB_MAXIMUM_THREADS=1 MKL_NUM_THREADS=1` UNLESS the variable is already set in the
  parent env (an explicit operator choice always wins — escape hatch by construction).
- Loud once-per-run note when injecting (visibility: the run's env story is knowable).
- Cache-key question at design: env vars are not in the read-set D — confirm injection does
  not perturb `Kpre`/fingerprint identity (it should not; document why).
- RUNBOOK simplifies: the pins move from "ALWAYS type this" to "defaulted; override by
  exporting your own values".

## Done when

- A bare `metis run` on a real sweep spawns leaves with the four pins set (asserted at the
  spawn seam with a fake exec / env capture); operator-set values pass through untouched.
- Fork-server path covered too (server env at spawn; a test pins it).
- One loud injection note per run; RUNBOOK updated; a bare real-sweep smoke shows sane
  throughput (no thrash) — rate line evidence in the close.

## Plan

- [ ] (at claim) Confirm env/cache-identity non-interaction; TDD both spawn seams; RUNBOOK.

## Log

### 2026-07-16
- Filed from the operator's UX pass (issue 3 of 3): bare `metis run titanic-sweep.md` → board
  ETA ~3h = the #42 thrash signature at default NumCPU without pins. Workaround today:
  the RUNBOOK §1 pinned invocation (`--sample 3 --parallel 8` + env pins) — the full 7,200-fold
  grid bare is the worst case. This issue makes the safe thing the default thing.
