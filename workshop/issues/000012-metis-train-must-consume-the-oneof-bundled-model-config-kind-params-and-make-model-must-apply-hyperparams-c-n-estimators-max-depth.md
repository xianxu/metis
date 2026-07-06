---
id: 000012
status: working
deps: []
github_issue:
created: 2026-07-05
updated: 2026-07-05
estimate_hours:
started: 2026-07-05T23:33:31-07:00
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

## Spec

- **`make_model(kind, seed, params=None)`** applies the swept hyperparams:
  - `logreg` → `LogisticRegression(C=params.get("C", 1.0), max_iter=1000, random_state=seed)`.
  - `rf` → `RandomForestClassifier(n_estimators=params.get("n_estimators", 100),
    max_depth=params.get("max_depth"), random_state=seed)`.
  - Unknown params: ignore (or validate loudly) — decide during design; default apply-known.
- **`train` / `cv_score`** thread `params` through to `make_model`.
- **`metis/steps/train.py`** parses `w["model"]` in BOTH forms (a pure helper, unit-testable):
  - a **string** (`"logreg"`) → `(kind="logreg", params={})` — backward-compat with
    `titanic-baseline.md` / `titanic-features.md`.
  - a **single-key dict** (`{"rf": {"n_estimators": 200, "max_depth": 4}}`, the `$oneof` bundle) →
    `(kind="rf", params={...})`.
  - malformed (multi-key dict, unknown kind) → a loud error.

## Done when

- A `make_model`/`train`/`cv_score` unit test proves the hyperparams are **applied** (a logreg
  with `C=0.01` differs from `C=100`; an rf with `n_estimators=10` differs from `500` /
  `max_depth=1` differs from `None`) — not silently dropped.
- A pure `parse_model_config(w["model"])` helper unit-tested on the string form, the `$oneof`
  dict form, and the malformed cases.
- `metis/steps/train.py` runs a `$oneof`-expanded point (real cv-split→train contract) to a
  `cv_score` without error — the exact input kbench#4's sweep produces.
- Backward-compat: the existing `model: logreg` bare-string thread (titanic-baseline) still trains.
- atlas: the model-config contract (`{kind: {params}}` ← `$oneof`) documented.

## Plan

- [ ] RED: `make_model`/`train` unit test — swept hyperparams change the fitted estimator (fails today: params ignored).
- [ ] GREEN: `make_model(kind, seed, params)` applies C / n_estimators / max_depth; thread through `train`/`cv_score`.
- [ ] RED/GREEN: pure `parse_model_config` (string | `$oneof` dict | malformed) + wire into `metis/steps/train.py`.
- [ ] Integration: a `$oneof`-expanded train point yields a `cv_score`; backward-compat bare-string still trains.
- [ ] atlas: model-config contract.

## Log

### 2026-07-05
- Filed from kbench#4's composition test. The shape/sweep/ledger/features-knob all compose (the
  ledger `show` renders the free-param tuple incl. list-valued `features` + `$oneof` model paths);
  the sole blocker is `metis/train` not consuming the `$oneof` model bundle + `make_model` ignoring
  hyperparams. Blocks kbench#4 (the metis-v1 acceptance demo).
