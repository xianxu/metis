---
id: 000029
status: open
created: 2026-07-12
updated: 2026-07-12
estimate_hours:
github_issue:
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
