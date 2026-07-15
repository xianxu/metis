---
id: 000044
status: open
deps: []
github_issue:
created: 2026-07-14
updated: 2026-07-14
estimate_hours:
---

# leaf executor: warm fork-server — kill per-step interpreter+import cost

## Problem

Every step execution spawns `uv run → python -m metis.trace <module>`: a fresh interpreter +
`import pandas, sklearn` = **~1.0s measured** (venv python: `import pandas, sklearn, numpy` 0.99s
vs bare interpreter 0.02s) + uv resolver overhead, before any step work runs. A kbench#9-scale
sweep executes ~5,000 leaf steps → ~10-15 min of a ~30-min wall clock is interpreter+import,
repeated identically. Observed: ~4.5s wall per train at 8 slots when the actual rf/gbm fit on
~800 rows is 0.3-1s. Operator question that filed this: "can the nested-CV run be a single
process, or a limited set of processes, each handling many training configurations?"

## Spec

(at claim) Ranked options — the constraint is preserving metis's per-step semantics
(process isolation, `metis.trace` read-audit + reads.json per step, env contract, crash
isolation, per-step cache/content-addressing):

1. **Fork-server (recommended):** one warm server process per (python env × run), preloading
   the heavy imports; each step = fork() → child chdirs/sets env per the existing contract →
   runs the step module → exits. Fork ≈ 10-50ms vs ~1.2s spawn+import → ~5-10× sweep throughput.
   Preserves EVERYTHING (each step is still its own process; trace hooks install post-fork).
   macOS caveat: fork safety wants single-threaded BLAS in the server (we already pin leaves).
2. **Persistent worker pool** (N workers, step calls dispatched as messages, no fork): max reuse
   but per-call state bleed (module caches, RNG, trace scoping) breaks the isolation the
   trace/capture semantics assume — needs explicit per-call reset audit. Riskier than 1.
3. **Vectorized/batch executor** (one process runs a LIST of configs in-process): biggest raw
   win but hostile to per-(config,fold) content-addressing/artifacts — would need the worker to
   emit per-item artifacts the runner stores individually. Out of scope unless 1 proves
   insufficient.

Quick wins independent of the above: exec the venv python directly instead of the `uv run`
indirection (resolver cost per spawn); check whether cv-split can be engine-internal (it is
conceptually the engine's fold loop already — one fewer subprocess per run).

## Done when

-

## Plan

- [ ]

## Log

### 2026-07-14
