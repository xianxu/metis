---
id: 000029
status: open
deps: []
github_issue:
created: 2026-07-12
updated: 2026-07-12
estimate_hours:
---

# real-data driver:cv confinement e2e — leak caught through exp_path within the orchestration

## Problem

metis#23's L2 read-confinement is proven through the **real chain in isolation**
(`TestExecStep_ConfinesRealUvStep_OutOfRootRead`: `execStep` → real uv `metis/cv-split` → `exp_path`
catches an out-of-root read), and the `driver:cv` **orchestration** wiring is code-confirmed
(`runOuterFold` sets `readRoot=analysis_i` on the sealed sweep). But **no test drives a leak through
the real `exp_path` chokepoint from *within* a real `driver:cv` orchestration** — the fake exec used
by the nested-CV e2e (`nestedcv_e2e_test.go`) bypasses `metis.io` entirely. So the composition
(real-chain enforcement + code-confirmed wiring) is strong, but the end-to-end L2 seal in the actual
driver:cv path is a **recorded deferral**, not an exercised guarantee (surfaced by the #23 close review, I-A).

**Blocker:** a real-data `driver:cv` e2e needs a data-phase step that produces a base dataset from the
already-materialized `testdata/dataset/toy` (so `outer-split` has something to subset). metis's real
step-types are `cv-split`/`outer-split`/`train`/`predict` — none is a dataset-producing "adapt" for the
toy case (the titanic path uses kbench's `titanic/adapt`). So this needs a small `test/`-style toy
data-step (or a `metis/identity` passthrough) first.

## Spec

Add a minimal toy data-producing step so a real `driver:cv` shape can run end-to-end over `toy`, then a
real-subprocess e2e (mirror `e2e_test.go:TestToyPipeline_EndToEnd`, skip if uv absent) that:
1. runs a real `driver:cv` over toy → completes + reports mean±SE + ships nothing (the happy path with
   confinement ACTIVE, proving no false-positive → sealed-sweep reads correctly confined to `analysis_i`);
2. injects a sealed-sweep step reading **outside** `analysis_i` (a full-dataset exp-relative ref) and
   asserts the run **fails** with the confinement error — the leak caught through the real `exp_path`
   *within the orchestration* (the sub-assertion #23 Task 2.5 item 4 deferred).

Optionally fold the toy data-step into a shared test fixture reused by the #23 nested-CV e2e.

## Done when

- A real `driver:cv` over toy runs end-to-end (real steps) and an injected sealed-sweep out-of-root read
  is caught through the real `exp_path` chokepoint; the happy path (in-root) completes with no false-positive.

## Plan

- [ ] (spec at claim) minimal toy data-step; real-subprocess `driver:cv` e2e (happy + injected-leak-caught).

## Log

### 2026-07-12
- Filed from metis#23's close review (I-A). The L2 real-chain enforcement + the code-confirmed driver
  wiring already ship in #23; this closes the orchestration-level e2e gap once the toy data-step exists.
  Low urgency — the honest-estimate-tracks-public acceptance is operator-gated Titanic regardless, which
  exercises the real driver:cv path with real data.
