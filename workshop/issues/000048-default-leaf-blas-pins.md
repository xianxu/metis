---
id: 000048
status: working
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-16
estimate_hours: 0.96
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

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.05 impl=0.25   # blasPins pure core + unit tests
item: smaller-go-module   design=0.05 impl=0.30   # two spawn seams (execStep, fork-server) + seam tests
item: smaller-go-module   design=0.02 impl=0.15   # runExperiment wiring + note + full-chain e2e
item: atlas-docs          design=0.02 impl=0.10   # RUNBOOK rewrite + atlas + stale-pin grep-sweep
design-buffer: 0.15
total: 0.96
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

Cache-identity question resolved at design (no cache work): `Kpre` hashes
`{step_id, uses, with, seed, upstream}`, HIT-validation re-hashes read-set D, fingerprint is
git state — env is in none of them, so injection perturbs nothing (documented in
`blaspins.go`'s doc comment).

## Plan

Durable plan: `workshop/plans/000048-default-leaf-blas-pins-plan.md` (single pass, no Mx —
one close boundary).

- [ ] blasPins pure core (ambient-wins rule) + unit tests
- [ ] legacy execStep seam: pins field → child env, env-dump seam test
- [ ] fork-server seam: pins on server env at spawn (children inherit), real-uv test
- [ ] runExperiment once-per-run wiring + loud note + full-chain e2e (note + passthrough)
- [ ] docs: RUNBOOK §1 simplification (kbench), atlas, stale-pin grep-sweep
- [ ] bare real-sweep smoke — rate-line evidence in the close

## Log

### 2026-07-16
- Filed from the operator's UX pass (issue 3 of 3): bare `metis run titanic-sweep.md` → board
  ETA ~3h = the #42 thrash signature at default NumCPU without pins. Workaround today:
  the RUNBOOK §1 pinned invocation (`--sample 3 --parallel 8` + env pins) — the full 7,200-fold
  grid bare is the worst case. This issue makes the safe thing the default thing.
