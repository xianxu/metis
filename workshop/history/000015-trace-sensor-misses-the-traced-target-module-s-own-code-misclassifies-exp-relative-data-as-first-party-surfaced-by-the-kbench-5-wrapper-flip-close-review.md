# Boundary Review — metis#15 (whole-issue close)

| field | value |
|-------|-------|
| issue | 15 — Trace sensor misses the traced target module's own code + misclassifies exp-relative data as first-party (surfaced by the kbench#5 wrapper flip) |
| repo | metis |
| issue file | workshop/issues/000015-trace-sensor-misses-the-traced-target-module-s-own-code-misclassifies-exp-relative-data-as-first-party-surfaced-by-the-kbench-5-wrapper-flip.md |
| boundary | whole-issue close |
| milestone | — |
| window | c12b0bebcccb4af2162951ac1baf00515ce1b684..HEAD |
| command | sdlc close --issue 15 |
| reviewer | claude |
| timestamp | 2026-07-06T18:06:47-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
Both suites are green (39 Python, Go sensor tests ok), and I independently verified the real sensor end-to-end. Here is the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both defects the issue targets are correctly fixed and I verified them independently — not just via the committed tests. Fix A (`_capture_target`) reliably lands the traced module's own `.py` in D: I ran the real sensor (`python -m metis.trace metis.steps.predict`) and confirmed `metis/steps/predict.py` is captured with **no** data leak. Fix B's allowlist in `_classify` drops `.parquet`/`schema.json` while keeping `.py`/`uv.lock`, and is byte-consistent with the Go contract test that consumes `reads.json`. The one thing that keeps this from a clean SHIP is a documentation-accuracy gap: the module's own docstring still describes the *pre-#15* contract, actively contradicting the narrowed behavior the diff introduces. That's cheap and non-blocking, hence FIX-THEN-SHIP.

**1. Strengths**
- `_capture_target` (metis/trace.py:164) resolves the origin and routes through the existing `_classify` rather than re-implementing classification — clean reuse (ARCH-DRY pass).
- Verified end-to-end against the *real* sensor, not only unit tests: `metis/steps/predict.py` now appears in `roots`, `used_site_packages=True`, no `.csv`/`.json` leak.
- The Python producer predicate (`_classify` metis/trace.py:122) is exactly consistent with the Go checker `TestSensor_RecordsFirstPartyCodeReads` (cmd/metis/trace_test.go:160) — the producer now provably satisfies the contract the checker asserts.
- Robust-by-construction: `_capture_target` uses `find_spec(...).origin` (the `.py` source path, `.pyc`-independent), so it fixes the bytecode-cache-luck fragility the issue describes rather than papering over it.
- Atlas updated in the same window (atlas/index.md) for both fixes.

**2. Critical findings**
- None.

**3. Important findings**
- **Module docstring drift** — metis/trace.py:10-18. The header still says *"D is **first-party code + config under the project root only**"* and *"keep: project files that are not under the venv / … / the run dir"*. After Fix B, D keeps **only** `.py` + `uv.lock`; a committed `.yaml`/`.json`/`.toml` config read by a step is now dropped. A maintainer reading this docstring (the primary in-code definition of D) would be misled about the exact contract this issue redefined. Fix sketch: update the D definition to state the `.py + uv.lock` allowlist (e.g. append to line 10-12: "kept reads are further gated to `.py` + `uv.lock`; all other first-party files, including data like `.parquet`/`schema.json`, are class-1 and excluded (#15)"). The atlas gate is satisfied, but the in-code contract doc is now wrong.

**4. Minor findings**
- `_capture_target`'s `find_spec` (which imports parent packages) runs **outside** `main()`'s `try/finally` (metis/trace.py:185 vs 186-189). A parent `__init__.py` raising a *non-caught* exception (RuntimeError/KeyError — `ValueError`/`ImportError` are caught) would now skip the `finally: _write_reads()` partial write that the `run_module` path would have produced. Extreme edge (metis/kbench parents are trivial); note only.
- `ModuleNotFoundError` in the except tuple (metis/trace.py:172) is redundant — it's a subclass of `ImportError` already listed. Harmless.
- Design note (not a bug): narrowing D to `.py + uv.lock` means any *future* first-party non-`.py` config a step depends on (committed `.yaml`, `importlib.resources` data) will silently not invalidate the cache. I confirmed no current metis step reads such a committed config (`schema.json` is data; `with.json` is runner-injected under the run dir), so there's no present correctness gap — the issue explicitly settled this tradeoff.

**5. Test coverage notes**
- Fix A: `test_capture_target_records_own_module_file` + my real-sensor run; robust to `.pyc` state by construction.
- Fix B: `test_classify_excludes_data_keeps_code` (parquet/schema.json dropped, `.py`/`uv.lock` kept) + the Go e2e contract test. Good coverage of the exact bug class shipped.
- Small gap: no test that `_capture_target("does.not.exist")` returns without raising (the swallow path). Low risk; optional.

**6. Architectural notes**
- **ARCH-DRY — pass.** `_capture_target` reuses `_classify`; the Python-predicate vs Go-test-predicate "duplication" is a test asserting the producer's contract (defense-in-depth), not two producers of the rule.
- **ARCH-PURE — pass (with note).** The classification predicate is pure; `_capture_target` is thin IO glue at the boundary. Tests are integration-flavored (git-init real repos, `find_spec` on real modules) — appropriate for a sensor whose whole purpose is IO; no mock-only entity masquerading as pure.
- **ARCH-PURPOSE — pass.** Both defects are fixed (not the cheap subset); the single source of the D contract is `_classify` (Go side is a checker, not a second producer); unblocking kbench#5 is a genuinely separable consumer-repo follow-up, correctly deferred.

**7. Plan revision recommendations**
- None required — the plan's two entities (`_capture_target` new, `_classify` modified) match the code, and all Plan/Done-when items for this boundary are delivered. (The docstring fix above falls under Task 3's atlas/docs item; folding it in resolves the sole Important finding.)

---

## Re-review — 2026-07-06T18:12:45-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 15 — Trace sensor misses the traced target module's own code + misclassifies exp-relative data as first-party (surfaced by the kbench#5 wrapper flip) |
| repo | metis |
| issue file | workshop/issues/000015-trace-sensor-misses-the-traced-target-module-s-own-code-misclassifies-exp-relative-data-as-first-party-surfaced-by-the-kbench-5-wrapper-flip.md |
| boundary | whole-issue close |
| milestone | — |
| window | c12b0bebcccb4af2162951ac1baf00515ce1b684..HEAD |
| command | sdlc close --issue 15 |
| reviewer | claude |
| timestamp | 2026-07-06T18:12:45-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: SHIP
confidence: high
```

Both defects the issue targets are correctly fixed, and I verified them independently rather than trusting the commit messages: full suites are green (39 Python, all Go packages; `go build`/`vet` clean), and the real-sensor contract test (`TestSensor_RecordsFirstPartyCodeReads`, which drives `python -m metis.trace metis.steps.train`) confirms the target module's own `.py` lands in D with no non-code path leaking. Fix A (`_capture_target`) resolves the traced module's file via `find_spec(...).origin` and routes it through the existing `_classify`, so an edit to the target step's own code always invalidates its cache regardless of bytecode-cache state. Fix B's `.py`/`uv.lock` allowlist in `_classify` drops exp-relative data (`.parquet`/`schema.json`) while keeping code + the dep lock, byte-consistent with the Go checker that consumes `reads.json`. The `reads.json` shape is unchanged, so no Go change was needed — the single source of the D contract is `_classify`. The round-1 Important (module docstring describing the pre-#15 contract) is already reconciled inside this window (commit 98be991), so nothing blocks SHIP; only two Minor notes remain.

**1. Strengths**
- `_capture_target` (metis/trace.py:172) reuses `_classify` instead of re-implementing classification — ARCH-DRY pass, and the `find_spec(...).origin` approach is `.pyc`-independent so it fixes the fragility at root (not the bytecode-cache luck the issue describes).
- The Python producer predicate (`_classify`, metis/trace.py:130) is exactly the contract the Go checker asserts (cmd/metis/trace_test.go:160) — producer now provably satisfies the checker; the `.py`/`.so`/"built-in" origin edge cases are safely dropped by the same gate.
- Docstring (metis/trace.py:10-19) reconciled to the narrowed contract within the same window; atlas/index.md updated for both rules in the same range (Docs gate satisfied).
- Fix B is correctly ordered *after* the run-dir/stdlib/venv exclusions, so it only narrows already-first-party reads.

**2. Critical findings**
- None.

**3. Important findings**
- None. (Round-1's docstring-drift Important is resolved in-window at metis/trace.py:10-19.)

**4. Minor findings**
- `_capture_target(target)` runs *outside* `main()`'s `try/finally` (metis/trace.py:193 vs 194-197), and its `except` catches only `ImportError/AttributeError/ValueError/ModuleNotFoundError`. If a parent `__init__.py` raises a *non-caught* exception at import time (e.g. `RuntimeError`), `main()` aborts before `finally: _write_reads()` — skipping the partial reads.json the `run_module` path would have written, contradicting the module's own "record what it read before failing" contract. Failure direction is safe (absent reads.json → treated as empty → MISS, never a false HIT) and the trigger is narrow, hence Minor. Cheap fix: move `_capture_target(target)` inside the `try` block (before `run_module`) so any exception it raises still hits the `finally`.
- `ModuleNotFoundError` in the `except` tuple (metis/trace.py:180) is redundant — it's a subclass of the already-listed `ImportError`. Harmless.
- Loose (not wrong): `cmd/metis/trace.go:16` and `pkg/cache/cache.go:8` still describe D as "first-party code+config" — accurate only if read as "code + the `uv.lock` config"; a stricter phrasing (`.py` + `uv.lock`) would match the sensor now that general config is dropped. These are downstream descriptive comments, not the enforcement point, so no change required.

**5. Test coverage notes**
- Fix A: `test_capture_target_records_own_module_file` pins the helper; `TestSensor_RecordsFirstPartyCodeReads` covers the runpy-as-`__main__` end-to-end path (asserts `metis/steps/train.py` is captured). Good coverage of the exact bug class.
- Fix B: `test_classify_excludes_data_keeps_code` (parquet/schema.json dropped, `.py`/`uv.lock` kept) + the Go contract test. Solid.
- Optional gap: no test that `_capture_target("does.not.exist")` returns without raising (the swallow path). Low risk.

**6. Architectural notes for upcoming work**
- ARCH-DRY / ARCH-PURE / ARCH-PURPOSE all pass. Shadow-sweep of the D-contract consumers: `_classify` is the single source; the Go runner derives from the unchanged `reads.json` shape; the Go test is a checker (defense-in-depth), not a second producer; atlas + docstring are updated. No hand-maintained restatement left un-derived.
- **For kbench#5 (the unblocked consumer):** excluding kbench's *committed* exp-relative Dataset from D is correct on the metis side, but it shifts the invalidation burden entirely to K_pre. If that Dataset is a committed dir rather than a run-dir/CAS upstream artifact (as the issue's Problem-B and the kbench#3 e2e note flag), kbench must ensure it's keyed via an upstream output-hash — otherwise a Dataset edit won't invalidate `features` and produces a false HIT on the kbench side. This is a kbench-side concern, correctly out of scope here, but worth pinning when kbench#5 resumes.

**7. Plan revision recommendations**
- None. The plan's two entities (`_capture_target` new, `_classify` modified) match the code at the stated locations, and every Plan/Done-when item for this boundary is delivered. The docstring fix that round-1 flagged now falls under the completed atlas/docs task.
