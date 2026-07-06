# Boundary Review — metis#12 (whole-issue close)

| field | value |
|-------|-------|
| issue | 12 — metis/train must consume the $oneof-bundled model config {kind:{params}} and make_model must apply hyperparams (C, n_estimators, max_depth) |
| repo | metis |
| issue file | workshop/issues/000012-metis-train-must-consume-the-oneof-bundled-model-config-kind-params-and-make-model-must-apply-hyperparams-c-n-estimators-max-depth.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5ccbc644bd00fedc25a12fc35e6136775c2b7950..HEAD |
| command | sdlc close --issue 12 |
| reviewer | claude |
| timestamp | 2026-07-05T23:47:09-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
All 13 tests pass. I've verified the `$oneof` bundling contract against the actual Go expander (`shape.go:164` — `bundled := map[string]any{label: r.value}`), which produces exactly the `{kind: {params}}` single-key shape `parse_model_config` handles, and confirmed the titanic-baseline-shape uses that `$oneof` form. Here's my review.

```verdict
verdict: SHIP
confidence: high
```

This is a clean, tightly-scoped, spec-faithful change. It delivers the issue's full purpose: `parse_model_config` normalizes both the bare-string (v0/backward-compat) and `$oneof` single-key-bundle forms into `(kind, params)`; `make_model` applies the swept hyperparams (logreg `C`; rf `n_estimators`/`max_depth`); `train`/`cv_score` thread `params` through; and the thin `metis/steps/train.py` shell wires it. Every "Done when" item is delivered and test-covered, the pure/IO separation is exemplary, and I confirmed the integration test exercises the *exact* shape the Go `$oneof` expander emits. Nothing blocks SHIP — only a couple of minor, non-blocking notes.

**1. Strengths**
- **Contract verified end-to-end, not just asserted.** The `$oneof` bundle premise (`{kind: {params}}`) is real: `pkg/shape/shape.go:164` bundles to a single-key map, and `testdata/experiment/titanic-baseline-shape.md:26-29` uses exactly `{$oneof: {logreg: {C:…}, rf: {n_estimators:…, max_depth:…}}}`. `parse_model_config` (`metis/model.py:29-38`) handles precisely this. Excellent traceability from motivating sweep → contract → code → test.
- **Regression-proof, not tautological, testing.** `test_hyperparams_change_the_fit` (`tests/test_model.py:53-65`) proves params flow all the way through `cv_score→train→make_model` into the actual fit (if params were ignored, both settings collapse to defaults and `weak == strong` fails). `test_make_model_applies_hyperparams` separately pins each attr on the constructed estimator. Together they cover both "reaches the estimator" and "reaches the fit."
- **Backward-compat is genuinely byte-faithful.** Old `LogisticRegression(max_iter=1000, random_state=seed)` had implicit `C=1.0`; new code makes it explicit `C=1.0`. Old rf had implicit `max_depth=None`; new is explicit `max_depth=None`. Same estimators → the existing `test_train_then_predict_chain` and determinism tests still pass unchanged (verified: 13 passed).
- **ARCH-PURE / ARCH-DRY / ARCH-PURPOSE all pass** (details below). Defensive copy `dict(params or {})` (`model.py:34`) prevents callers mutating the shape's dict.
- Docstrings on the step (`train.py:10-13`) and atlas (`experiment.md`) accurately describe the new contract.

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**
- `metis/model.py:34` — `dict(params or {})` raises `TypeError`, not the promised `ValueError`, when a single-key bundle carries a non-dict, non-None value (e.g. an exotic shape `{$oneof: {logreg: 0.1}}` → `{"logreg": 0.1}` → `dict(0.1)` → `TypeError: 'float' object is not iterable`). No real model sweep produces this (branches are always param maps), and the error is still loud, but the docstring/test contract promises a clean `ValueError` for "malformed." Cheap hardening: guard `isinstance(params, (dict, type(None)))` before the `dict(...)` and fall through to the existing `raise ValueError`. The malformed-case test (`test_model.py:75`) could then add `{"rf": 5}` to the tuple.
- Test coverage gap for the above: the malformed table exercises multi-key/non-dict/empty/list, but not a single-key dict with a non-mappable value — the one input shape that currently escapes the intended `ValueError`.

**5. Test coverage notes**
- Coverage is strong and matches the Done-when 1:1: hyperparams-applied (attr + fit), `parse_model_config` table (string | `$oneof` dict | malformed), the exact-kbench#4 integration input, and the bare-string backward-compat path (`test_train_then_predict_chain:50`). INTEGRATION test uses the real step contract via `monkeypatch`+`tmp_path` (injected filesystem), not function-mocks — correct per §5.
- `assert weak != strong` (`test_model.py:65`) is deterministic under fixed seeds (not run-to-run flaky), so it's fine for CI. Only theoretical fragility: a future sklearn numerics change could equalize the two scores. A directional `assert weak < strong` would be marginally more legible about *why* they differ, but `!=` is the safest params-reached-the-fit assertion — leave as-is.

**6. Architectural notes for upcoming work**
- **ARCH-DRY: pass.** `parse_model_config` is the single source for the model-config shape; the train step reuses it rather than re-parsing `w["model"]` ad hoc. Confirmed no other consumer re-parses a model config — the predict step's `w["model"]` is an upstream *step-id* reference (`toy-pipeline.md:17`), a different meaning, so no consolidation is owed.
- **ARCH-PURE: pass.** `parse_model_config`/`make_model`/`train`/`cv_score` are pure and unit-tested with no IO (`tests/test_model.py`); `steps/train.py::main` is the thin IO seam calling the pure helpers. No business logic buried in the handler.
- **ARCH-PURPOSE: pass (shadow-sweep clean).** The purpose — a *real* sweep, not a sham — is fulfilled: the single consumer (train step) derives `(kind, params)` from `parse_model_config`, and the estimator derives from `params`; the integration test proves the actual kbench#4 input trains. No hand-maintained restatement left as a deferred consumer.
- **Forward note (design latitude, not a defect):** `make_model` silently ignores *unknown* param keys (`model.py:44-45`). The Spec explicitly sanctioned "ignore (or validate loudly)," so this is a permitted choice, and all keys kbench#4 emits are known. But the issue's own motivating failure *is* silently-dropped hyperparams — so if a future shape adds a new knob (e.g. `min_samples_leaf`) without extending `make_model`, that axis would silently no-op and every point would be identical on it. Worth revisiting when the model surface grows: validate-loudly-on-unknown, or at least log dropped keys, would close the milder version of the exact sham this issue killed.

**7. Plan revision recommendations** — none. The Core-concepts table (plan §"Core concepts") matches the code exactly: `parse_model_config` (new), `make_model`/`train`/`cv_score` (modified) all at `metis/model.py`; `metis/steps/train.py::main` (modified). All PURE entities are tested without IO; the one INTEGRATION seam is injected. The plan still faithfully describes the delivered code.
