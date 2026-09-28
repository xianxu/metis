---
id: '000012'
status: done
started: 2026-07-05T23:33:31-07:00
created: 2026-07-05
updated: 2026-07-05
estimate_hours: 1.2
actual_hours: N/A
---

# metis/train must consume the $oneof-bundled model config {kind:{params}} and make_model must apply hyperparams (C, n_estimators, max_depth)

## Problem

Surfaced by kbench#4's acceptance sweep — the **first real sweep over model hyperparams**.
metis#6's `$oneof` expands a `model:` knob into a **labeled-sum bundle** `{kind: {params}}` —
e.g. `model: {$oneof: {logreg: {C:…}, rf: {n_estimators:…, max_depth:…}}}` expands per-point to
`model: {"rf": {"n_estimators": 200, "max_depth": 4}}`. But `metis/steps/train.py` does
`kind = w["model"]` **expecting a string** (`"logreg"`/`"rf"`), so it hands `make_model` a dict →
`ValueError: unknown model {...}`. **Every point of a hyperparam sweep fails.**

Worse, even with a bare string `make_model` **ignores hyperparams entirely**:
`LogisticRegression(max_iter=1000, random_state=seed)` (no `C`),
`RandomForestClassifier(n_estimators=100, random_state=seed)` (hardcoded `n_estimators`, no
`max_depth`). So a hyperparam sweep would be a **sham** even if the dict parsed — all `logreg`
points identical, all `rf` points identical.

Root cause: `metis/model.py` + `metis/steps/train.py` predate metis#6's `$oneof`, and metis#7's
sweep was only ever exercised with the `test/echo` step — the sweep→train **hyperparam path was
never integration-tested** (contract-correct ≠ invocation-correct). This is the integration gap
the kbench#4 acceptance demo exists to catch.
