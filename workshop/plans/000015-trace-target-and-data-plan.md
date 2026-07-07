---
issue: 000015
title: Trace sensor — capture the target module + exclude data from D
status: active
created: 2026-07-06
---

# Trace target + data-exclusion Implementation Plan

> **For agentic workers:** AGENTS.md §3. TDD. Two focused `metis/trace.py` fixes surfaced by the kbench#5 flip.

**Goal:** `D` reliably = **the traced module's own first-party code + its imports, and ONLY code (`.py`) + the dep lock (`uv.lock`)** — no data. Unblocks kbench#5 (route kbench steps through `metis.trace`).

**Architecture:** Two small changes in `metis/trace.py`:
- **A. Capture the target module's file.** `runpy.run_module(target, run_name="__main__")` runs the target as `__main__` and does NOT leave it under its qualified name in `sys.modules`, so `_snapshot_modules` captures its *parent packages* but not the module itself (bytecode-cache-dependent luck for metis's own steps). In `main()`, after installing the hook, resolve `importlib.util.find_spec(target).origin` and `_classify` it — the target's own `.py` is always in D.
- **B. `D = .py + uv.lock` only.** The sensor's *intended* contract (already asserted by `TestSensor_RecordsFirstPartyCodeReads`: "unexpected non-code path" for anything not `.py`/`uv.lock`) was broken by metis#11's multi-root — kbench's exp-relative Dataset (`.parquet`, `schema.json` under the kbench repo, not `METIS_RUN_DIR`) now leaks in as "first-party." Enforce the contract in `_classify`: keep a read only if it ends in `.py` or its basename is `uv.lock`; drop everything else (data is class-1, keyed via upstream output-hashes in K_pre, never D).

## Tasks (TDD)

- [ ] **1a RED** — `tests/test_trace.py`: trace a module via `metis.trace` (or drive `_classify` + a target-resolution helper) and assert the **module's own `.py`** (not just its package `__init__`) is in `roots`. Fails today.
- [ ] **1b GREEN** — add `_capture_target(target)` (find_spec → `_classify(origin)`), call it in `main()` before `run_module`. PASS.
- [ ] **2a RED** — assert `_classify` of a first-party `.parquet`/`schema.json` read does NOT enter `_roots` (data excluded), while `.py` + `uv.lock` still do. Fails today (data leaks).
- [ ] **2b GREEN** — add the `.py`/`uv.lock` allowlist gate in `_classify`. PASS. Regression: the existing `TestSensor_RecordsFirstPartyCodeReads` (metis .py + uv.lock) + the metis#11 cross-repo/HIT→MISS tests stay green.
- [ ] **3** — full `uv run pytest` + `go test ./...` green (the reads.json shape is unchanged — still `{roots, used_site_packages}` — so the Go side is untouched); atlas: the target-module capture + the "D is code, not data" rule.
- [ ] **4** — `sdlc close --issue 15`. Log: unblocks kbench#5.

## Done when (issue) — mapped

- [ ] a traced module has its OWN `.py` in D (bytecode-robust) — Task 1
- [ ] exp-relative data (`.parquet`) excluded from D — Task 2
- [ ] unblocks kbench#5 — verified when kbench#5 resumes

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.15  impl=0.3
item: smaller-go-module      design=0.15  impl=0.3
item: milestone-review       design=0.0   impl=0.2
item: atlas-docs             design=0.05  impl=0.1
design-buffer: 0.15
total: 1.3
```

Σdesign 0.35 × 1.15 = 0.4025; Σimpl 0.90 × 1.00 = 0.90; total ≈ 1.3. `smaller-go-module` #1 = Fix A (target-module capture + test); #2 = Fix B (`.py`/`uv.lock` allowlist + the leaked-data regression test); `milestone-review` = close; `atlas-docs` = the two rules. Single-pass atomic. `reads.json` shape unchanged → no Go change.
