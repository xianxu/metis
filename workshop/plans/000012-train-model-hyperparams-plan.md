---
issue: 000012
title: metis/train consumes the $oneof model config + make_model applies hyperparams
status: active
created: 2026-07-05
---

# train/model hyperparam config Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3. Steps use checkbox (`- [ ]`) syntax; TDD (superpowers-test-driven-development) throughout.

**Goal:** Make a model hyperparam sweep real — `metis/train` consumes the `$oneof`-bundled `model: {kind: {params}}` config metis#6 produces, and `make_model` actually **applies** the swept hyperparams (logreg `C`, rf `n_estimators`/`max_depth`) instead of ignoring them.

**Architecture:** Two pure seams + one thin wiring. `make_model`/`train`/`cv_score` (pure, `metis/model.py`) gain a `params` argument applied to the sklearn estimators. A new pure `parse_model_config(raw)` (`metis/model.py`) normalizes the `with["model"]` value — a bare string OR the single-key `$oneof` dict — into `(kind, params)`. `metis/steps/train.py` (the thin IO shell) calls `parse_model_config` and threads `(kind, params)` into `cv_score`/`train`. Backward-compatible: `model: "logreg"` still trains with default params.

**Tech Stack:** Python, scikit-learn (`LogisticRegression`, `RandomForestClassifier`), pytest.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `parse_model_config` | `metis/model.py` | new |
| `make_model` | `metis/model.py` | modified |
| `train` | `metis/model.py` | modified |
| `cv_score` | `metis/model.py` | modified |

- **`parse_model_config(raw) -> tuple[str, dict]`** — normalize the `with["model"]` value. `str` → `(raw, {})`; a **single-key dict** `{kind: params}` (the `$oneof` bundle) → `(kind, params or {})`; anything else (multi-key dict, non-str key, empty) → `ValueError` (loud). Pure, table-tested.
  - **DRY rationale:** one place understands the model-config shape; the train step (and any future model-consuming step) reuses it rather than re-parsing `w["model"]` ad hoc.
  - **Future extensions:** more model kinds just add to `make_model`; the config shape is unchanged.
- **`make_model(kind, seed, params=None)`** — applies known hyperparams; unknown keys ignored (forward-compatible with shapes that carry extra knobs). `logreg` → `C=params.get("C", 1.0)`; `rf` → `n_estimators=params.get("n_estimators", 100)`, `max_depth=params.get("max_depth")` (None = sklearn default). `params=None` ⇒ `{}` (today's behavior, defaults).
- **`train(X, y, kind, seed, params=None)` / `cv_score(X, y, folds, kind, seed, params=None)`** — thread `params` to `make_model`. Signature widened with a defaulted arg → existing callers unaffected.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `metis/train` step `main` | `metis/steps/train.py` | modified | the step contract |

- **`metis/steps/train.py::main`** — replace `kind = w["model"]` with `kind, params = parse_model_config(w["model"])`; pass `params` to `cv_score` + `train`. Thin; all logic in the pure helpers.
  - **Test surface:** the existing step-contract test idiom (`tests/test_steps.py` `_run_step`) exercises a `$oneof`-form `model` end-to-end to a `cv_score` — the exact shape kbench#4's sweep emits — plus the bare-string backward-compat path.

## Tasks (TDD)

### Task 1: `make_model` applies hyperparams

- [ ] **1.1 RED** — `metis/model_test.py` (or wherever model tests live): assert a logreg built with `params={"C": 0.001}` has `.C == 0.001` and differs from `C=1000`; an rf with `params={"n_estimators": 7, "max_depth": 2}` has those attrs. Run → FAIL (params ignored / TypeError).
- [ ] **1.2 GREEN** — add `params=None` to `make_model`; apply `C` (logreg), `n_estimators`/`max_depth` (rf). Run → PASS.
- [ ] **1.3** — thread `params` through `train` + `cv_score` (defaulted). A test: two `cv_score`s with materially different params on the same data differ (proves params reach the fit — regression-proof the sweep isn't a sham). Commit.

### Task 2: `parse_model_config` + wire the step

- [ ] **2.1 RED** — test `parse_model_config`: `"logreg"` → `("logreg", {})`; `{"rf": {"n_estimators": 200, "max_depth": 4}}` → `("rf", {"n_estimators": 200, "max_depth": 4})`; `{"logreg": {}}` → `("logreg", {})`; malformed (`{"a": 1, "b": 2}`, `123`, `{}`) → `ValueError`. Run → FAIL.
- [ ] **2.2 GREEN** — implement `parse_model_config`. Run → PASS. Commit.
- [ ] **2.3** — wire `metis/steps/train.py`: `kind, params = parse_model_config(w["model"])` → `cv_score(..., params=params)`, `train(..., params=params)`. Commit.

### Task 3: integration (the exact kbench#4 input) + backward-compat

- [ ] **3.1** — a step-contract test (`_run_step` idiom, `tests/test_steps.py`): run `metis/train` with `model = {"rf": {"n_estimators": 50, "max_depth": 3}}` (the `$oneof` form) against the toy dataset + a folds.json → asserts a `cv_score` metric is written, no error. RED first (fails today: dict `kind`), then GREEN via Tasks 1-2. Also assert the bare-string `model: "logreg"` path still trains (backward-compat).
- [ ] **3.2** — `go build ./... && go vet ./... && go test ./...` (Go untouched but the repo gate) + `uv run pytest` (or the metis python test runner) green. Commit.

### Task 4: atlas + close

- [ ] **4.1** — atlas: document the model-config contract (`with["model"]` accepts a kind string OR the `$oneof` `{kind: {params}}` bundle; `make_model` applies `C`/`n_estimators`/`max_depth`).
- [ ] **4.2** — `sdlc close --issue 12` (single review boundary).

## Done when — mapped to the issue

- [ ] hyperparams applied (proven by a differ-test) — Task 1
- [ ] `parse_model_config` table-tested (string | dict | malformed) — Task 2
- [ ] `$oneof`-expanded point trains to a `cv_score`; bare-string still trains — Task 3
- [ ] atlas: model-config contract — Task 4

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.15  impl=0.25
item: smaller-go-module      design=0.15  impl=0.25
item: milestone-review       design=0.0   impl=0.2
item: atlas-docs             design=0.05  impl=0.1
design-buffer: 0.15
total: 1.2
```

Reconciliation: Σdesign 0.35 × 1.15 = 0.4025; Σimpl 0.80 × 1.00 = 0.80; total ≈ 1.2. `smaller-go-module` ×2 = (make_model/train/cv_score params) + (parse_model_config + step wiring + the integration test); `milestone-review` = the single close boundary; `atlas-docs` = the model-config contract note. Single-pass atomic (no Mx).
