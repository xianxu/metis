---
id: '000048'
status: done
started: 2026-07-16T11:10:34-07:00
created: 2026-07-16
updated: 2026-07-16
estimate_hours: 0.96
actual_hours: 0.71
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
