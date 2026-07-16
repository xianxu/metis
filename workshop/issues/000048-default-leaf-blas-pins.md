---
id: 000048
status: codecomplete
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-16
estimate_hours: 0.96
started: 2026-07-16T11:10:34-07:00
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
item: smaller-go-module   design=0.05 impl=0.25
item: smaller-go-module   design=0.05 impl=0.30
item: smaller-go-module   design=0.02 impl=0.15
item: atlas-docs          design=0.02 impl=0.10
design-buffer: 0.15
total: 0.96
```

Rows: (1) blasPins pure core + unit tests; (2) two spawn seams (execStep, fork-server) +
seam tests; (3) runExperiment wiring + note + full-chain e2e; (4) RUNBOOK rewrite + atlas +
stale-pin grep-sweep.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

Cache-identity question resolved at design (no cache work): `Kpre` hashes
`{step_id, uses, with, seed, upstream}`, HIT-validation re-hashes read-set D, fingerprint is
git state — env is in none of them, so injection perturbs nothing (documented in
`blaspins.go`'s doc comment).

## Plan

Durable plan: `workshop/plans/000048-default-leaf-blas-pins-plan.md` (single pass, no Mx —
one close boundary).

- [x] blasPins pure core (ambient-wins rule) + unit tests
- [x] legacy execStep seam: pins field → child env, env-dump seam test
- [x] fork-server seam: pins on server env at spawn (children inherit), real-uv test
- [x] runExperiment once-per-run wiring + loud note + full-chain e2e (note + passthrough)
- [x] docs: RUNBOOK §1 simplification (kbench), atlas + main.go --parallel help + lessons rule, stale-pin grep-sweep (Go+md+py)
- [x] bare real-sweep smoke — rate-line evidence in the Log below

## Log

### 2026-07-16
- 2026-07-16: closed — go test ./... -race green; bare cold-cache real-data metis run --fast completed 720/720 folds in 1m23s with exactly one default-pin note, with operator override plus legacy and fork-server seams covered by tests; review verdict: FIX-THEN-SHIP
- Filed from the operator's UX pass (issue 3 of 3): bare `metis run titanic-sweep.md` → board
  ETA ~3h = the #42 thrash signature at default NumCPU without pins. Workaround today:
  the RUNBOOK §1 pinned invocation (`--sample 3 --parallel 8` + env pins) — the full 7,200-fold
  grid bare is the worst case. This issue makes the safe thing the default thing.

### 2026-07-16 (built + smoke)
- **Peer RUNBOOK migration verified after close review:** kbench commit
  `bf57c5cd86f920ca9bf2827b2f1926a1e0ffee7d` changes
  `competition/titanic/pipelines/RUNBOOK-sweep.md` from four hand-typed BLAS env prefixes to bare
  `metis run`, documents all four metis#48 defaults + the operator override, and makes `--parallel`
  the one operator-facing parallelism knob. This closes the review's sole Important traceability
  finding; the peer diff is outside metis's review window by construction.
- Full SDLC single pass: plan fresh-eyes-reviewed (3 Important + hidden-trap sweep — the
  select-path bypass made an explicit decision, env-dump fixture gap promoted to a step, fixture
  syntax corrected against the real frontmatter convention; all folded), change-code judges
  plan-quality CLEAN / estimate-quality INFO. TDD red-green per task; full `go test ./... -race`
  green.
- **Cache-identity confirmed in code** (not just reasoned): `Kpre` = {step_id, uses, with, seed,
  upstream}; validation re-hashes read-set D; fingerprint is git state — env in none of them.
  Documented in blaspins.go.
- **`select --promote` deliberately unpinned** (serial single all-data fit — multi-threaded BLAS
  wanted; one leaf can't oversubscribe): decision comments at both select_cmd.go sites (plan
  review finding).
- **Bare real-sweep smoke (Done-when evidence):** disposable kbench workspace copy (kbench#10
  pattern: rsync minus .metis-cache/runs/.git, .venv symlink + UV_NO_SYNC=1, sibling symlinks for
  the metis#16 deps walk), COLD cache, real 891-row data, `metis run --fast titanic-sweep.md`
  BARE (no env pins, default --parallel, fork-server on) — the operator's exact footgun
  invocation: **`done in 1m23s — 722 rows → ledger (cohort e901889f)`, 720/720 inner folds ≈
  520 folds/min** (pinned reference ~107 trains/min; the pre-#48 bare run was a ~3h-ETA thrash
  with throughput ≈ 0). Exactly ONE note line: `metis: leaf BLAS pinned single-thread
  (MKL... OMP... OPENBLAS... VECLIB...) — the parallelism budget is --parallel; export a value
  yourself to override`. Smoke scale per the estimate-judge advisory: `--fast` (1 outer fold),
  not the full grid — sufficient to exercise both seams under real BLAS load.
- Smoke setup dead ends worth keeping: the deps-chain walk needs the ../kaggle → ../metis
  SIBLINGS present next to a workspace copy (symlinks suffice); a first attempt filtered the
  run's output through `grep -E "...error..."` and swallowed the real failure line ("no
  step-type executable") — capture full logs, filter at read time.
