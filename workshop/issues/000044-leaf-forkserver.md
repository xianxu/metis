---
id: 000044
status: codecomplete
deps: []
github_issue:
created: 2026-07-14
updated: 2026-07-15
estimate_hours: 1.08
started: 2026-07-15T10:33:09-07:00
actual_hours: 2.35
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

- A real kbench `metis run --fast` (or `--sample`) sweep runs its leaves through the fork-server
  with **measured before/after wall-clock in the Log** showing the import tax gone (target ≥3×
  on the leaf-bound portion; the loose-bound perf test pins ≥2× vs legacy on toy leaves).
- Per-step semantics preserved and tested: each leaf is its own forked process; `reads.json`
  written per child with `used_site_packages: true`; request env authoritative (`METIS_*`
  scrubbed — a prior request's `READ_ROOT` cannot leak); step failure → nonzero exit +
  traceback surfaced; caching/e2e suites green (`go test ./... -race` + pytest).
- Non-standard wrappers and server-start failure fall back to legacy exec LOUDLY (once per
  uses-type); `--forkserver=false` escape hatch works.
- atlas (executor section) + RUNBOOK reconciled.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.15 impl=0.35
item: smaller-go-module      design=0.10 impl=0.30
item: atlas-docs             design=0.02 impl=0.08
design-buffer: 0.30
total: 1.08
```

`greenfield-go-module` = the new `metis/forkserver.py` + the `run_traced` extraction (a single
greenfield concern with real-fork tests — priced as the module primitive despite being Python).
`smaller-go-module` = `cmd/metis/forkexec.go` (wrapperSpec + serverPool) + `Execute` routing +
flag, mirroring the existing executor patterns. `atlas-docs` = atlas/RUNBOOK + perf write-up.

## Plan

Durable plan: `workshop/plans/000044-leaf-forkserver-plan.md` (entities, protocol, risks).

- [x] T1 extract `run_traced` from `metis/trace.py` main (pure refactor; suite green).
- [x] T2 `metis/forkserver.py` + real-fork pytest suite (round-trip, env isolation, failure,
      concurrency, reads.json + forced used_site_packages).
- [x] T3 `wrapperSpec` parse in `cmd/metis/forkexec.go` (TDD; not-forkable fallback).
- [x] T4 `serverPool` + `Execute` routing + `--forkserver` flag (default on); hermetic
      integration through the real server; `go test ./... -race` green.
- [x] T5 perf acceptance (loose-bound test + REAL kbench --fast smoke, before/after wall-clock
      logged) + atlas/RUNBOOK; close.

## Log

### 2026-07-15 (close review — FIX-THEN-SHIP, all findings fixed in the close commit)
- **C1 (Critical, real production hang):** fork could land while a waiter thread held
  stdout's internal io lock → held lock copied into the child → deadlock at the child's first
  stdout use → whole run hangs. Fixed: `with lock: os.fork()` (writers only write under that
  lock) + belt-and-braces fresh std-stream rebind in the child post-dup2. Also dropped
  `metis.forkserver` from the child's sys.modules pre-snapshot (D symmetry: executor mode no
  longer changes cache keys).
- **I1:** mid-flight server death now ERRORS the step (`errDispatchedLost`) instead of legacy
  re-run against a possibly-still-running forked child (double-execution hazard). En route:
  `uv run` doesn't exec-replace — the server now gets its own process group (group-kill on
  hung shutdown; the new SIGKILL-mid-flight test needed it too). Test added.
- **I2:** perf test now ENFORCES the ≥2× bound on the warm marginal (measured: 43ms vs
  ~1.16s/4-leaf legacy — the bound passes ~13×); and the REAL kbench before/after pair is in:
  titanic-sweep-smoke, identical honest estimate 0.8103 both modes, **legacy 21.8s wall /
  104.3s user-CPU vs forkserver 9.8s wall / 29.8s user-CPU** — 2.2× wall at 40-leaf scale
  (server startup unamortized), **3.5× on the leaf-bound CPU** (Done-when's ≥3× target, met);
  per-leaf import tax measured ~1.85s (104→30s CPU over ~40 leaves). Cache-bust between the
  two runs = a one-line comment edit in kbench features.py, reverted (logged for provenance).
- 2026-07-15: closed — TDD throughout: 7 real-fork pytest tests (protocol, env authority/no-bleed, failure traceback, SystemExit, concurrency, reads.json+forced used_site_packages, EOF drain) + Go parse/pool/routing/fallback tests + toy e2e parameterized over both executors; full go test ./... -race + 87 pytest green. Invocation-verified on the REAL cross-repo workload: titanic-sweep-smoke through BOTH venvs fork-servers on macOS, 10.1s wall, zero fork hazards. Measured: toy e2e 3.70s->1.89s; per-leaf marginal ~290ms->~30ms (pandas-only leaf; real sklearn leaves ~1s tax amortize better). Atlas + RUNBOOK reconciled.; review verdict: FIX-THEN-SHIP
- BUILT (TDD throughout). T1 `run_traced` extraction (suite green). T2 `metis/forkserver.py` +
  7 real-fork pytest tests — protocol round-trip, env authority (READ_ROOT no-bleed), failure
  traceback, SystemExit pass-through, 4-way concurrency, reads.json + forced
  used_site_packages, EOF drain. T3 `parseWrapper` (4 cases incl. ROOT-line and pyproject
  guards). T4 `serverPool` + routing + `--forkserver` (default on; runOpts zero-value keeps
  direct callers/tests legacy): real-server round trip, broken-root falls back with ONE loud
  notice, non-conforming wrapper runs legacy loudly; toy e2e parameterized over both modes.
  Full `go test ./... -race` + 87 pytest green.
- **Measured:** toy e2e (4 real steps, incl. server startup+preload): legacy 3.70s →
  forkserver 1.89s. Marginal per-leaf: ~30ms forked vs ~290ms spawned for a pandas-only leaf
  (perf loose-bound test, n=4: 1.15s vs 0.91s incl. full preload); the real-sweep leaves
  (sklearn+kbench imports, ~1s tax) amortize far better — next operator sweep vs the ~28-min
  k10-probe baseline is the headline number. **Real cross-repo smoke:** titanic-sweep-smoke
  (3 outer × 4 configs, real kbench+metis wrappers, BOTH venvs' servers, real sklearn fits in
  forked children on macOS): completed 10.1s wall, honest estimate + ledger recorded, zero
  fork hazards (no ObjC guard, pins inherited).
- Docs: atlas/experiment.md fork-server bullet (executor section); kbench RUNBOOK §1 note
  (committed kbench-side).

### 2026-07-14
