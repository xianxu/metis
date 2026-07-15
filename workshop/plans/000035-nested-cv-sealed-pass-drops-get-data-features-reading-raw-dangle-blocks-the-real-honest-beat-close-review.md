# Boundary Review — metis#35 (whole-issue close)

| field | value |
|-------|-------|
| issue | 35 — nested-CV sealed pass drops get-data → features reading raw dangle (blocks the real honest-beat) |
| repo | metis |
| issue file | workshop/issues/000035-nested-cv-sealed-pass-drops-get-data-features-reading-raw-dangle-blocks-the-real-honest-beat.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99c48c1df5aa71fdc037762bd0db42fa15cfb56c..HEAD |
| command | sdlc close --issue 35 |
| reviewer | claude |
| timestamp | 2026-07-14T17:05:00-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** The metis half of the #35 one-road fix is small, correct, and lands exactly what the plan promised: a fifth `source` schema role (structurally excluded from model inputs because `Dataset.X()` selects by role — `dataset.py:28`), and `outer-split` carrying the test frame so `analysis_i` is shape-identical to the declared base (`outer_split.py:51`). I independently verified the whole-issue Done-when, not just this repo's window: metis 79 pytest + full Go suite green; the kbench nested smoke e2e is un-xfailed and **passes on re-run (3 passed, 55s)**; the shadow-sweep re-run from both repo roots finds no residual `raw: get-data` road except `adapt`'s own demultiplexer (the designed exception — the two metis test-fixture hits at `cmd/metis/shapesweep_test.go:116` and `select_cmd_test.go:203` are adapt-style, clean); the 4 relic winner files are deleted; both atlases are reconciled; the honest-beat numbers and leakage-tell PASS are recorded in the issue Log. Nothing blocks the boundary — the findings are cheap test hardening and one plan/issue bookkeeping mismatch.

## 1. Strengths

- **Role-based exclusion made the change nearly free.** Because `X()`/`y()` derive from `feature_cols()`/`target_col()`, adding `source` required zero changes to `train`/`predict`/`cv-split` — the plan's "exactly `id`'s semantics" claim holds in code, not just prose (`metis/dataset.py:26-35`). ARCH-DRY at its best: one role registry, all consumers derive.
- **The seal stayed sealed.** The `outer_split.py:47-51` comment states the seal-neutrality argument at the code site (test rows carry no assessment labels), and the change lives entirely in the thin IO shell — `cv_folds` and the fold mask are untouched (ARCH-PURE).
- **Cache invalidation is sound by construction:** the read-set D carries git-blob hashes of code read during a run (`pkg/cache/cache.go`), so pre-#35 cached `analysis_i` dirs (no `test.parquet`) MISS after this change — no silent stale-cache path.
- **The shadow-sweep was real.** I re-ran the plan's Task 5 greps fresh: kbench code, shapes, atlas, and generated artifacts are clean; `RAW_SOURCE_COLS` is gone; `SOURCE_COLS` lives at its producer (`adapt.py:26`) and is imported by `features.py:38` — the ARCH-DRY consolidation delivered, not paralleled.
- **The lessons (`workshop/lessons.md:147-151`) are specific and reusable** — the sole-road check and serialization-complete sweep rules directly encode what this bug taught.

## 2. Critical findings

None.

## 3. Important findings

- **`tests/test_io.py` — no metis-side round-trip test for the new role's contract.** The `ROLES` comment (`schema.py:17-19`) promises `source` columns "may hold strings/NaN," and this issue introduces the first object-dtype/NaN traffic through `io.save_dataset`/`load_dataset` (parquet). Today that path is proven only transitively, by kbench's e2e. Fix sketch: one test that saves and reloads a Dataset with a `source` column of dtype `object` containing a NaN, asserting role, dtype, and values survive (note parquet reads NaN-in-object back as `None` — pin whichever behavior is the contract). Cheap, and it defends the metis contract where it's declared rather than two repos away.

## 4. Minor findings

- `tests/test_outer_split.py:67-78` asserts only shape (len + columns) while the docstring claims the test frame is "carried through unchanged" — a one-line `sub.test.reset_index(drop=True).equals(full.test...)` would pin content, matching the sibling train-rows test at line 64.
- `outer_split.py:50` "test rows are unlabeled" states a Kaggle convention as fact; the stronger, always-true invariant is that assessment rows are fold-i *train* rows, which never appear in `test` — seal-neutrality holds even for a labeled test frame. Wording-only.
- `atlas/index.md:77` still describes `outer-split` output without the shape-identical/test-carry note (`experiment.md` has it; index is a map, so arguably fine).

## 5. Test coverage notes

- New behavior is TDD-pinned on both sides: `test_source_role_accepted_and_excluded_from_features` (pure, no IO — the PURE-entity contract holds) and `test_analysis_dirs_carry_test_frame` (INTEGRATION via the same `_run_step` fixture as its siblings).
- The gap is the io round-trip above; everything else this diff could plausibly break (fold disjointness, positional row identity, confinement seal) already has tests, and they still pass.
- The e2e evidence in the Log is reproducible — I re-ran it cold and it's green.

## 6. Architecture notes (explicit ARCH pass/flag)

- **ARCH-DRY: pass.** One role registry; `SOURCE_COLS` single-sourced at the producer; the deleted `RAW_SOURCE_COLS`/join logic was not re-created anywhere (grep-verified).
- **ARCH-PURE: pass.** Schema stays pure with IO-free tests; the outer-split change is one line inside the existing thin shell; no logic migrated into IO.
- **ARCH-PURPOSE: pass.** The purpose was *one road*, and the shadow-sweep confirms every raw-column consumer now derives from `adapt`'s Dataset — no hand-maintained second road remains in code, shapes, relics, or either atlas. The deferred work (estimand knob → metis#36, constructor algebra → metis#37) is genuinely separable extension, not the point of this issue.
- For upcoming #36: the `source` role is doing double duty as "carried, opaque to metis" — when the channel split lands, consider whether `source` columns need per-role dtype/NaN validation in `Schema.__post_init__` (today the strings/NaN allowance is comment-only).

## 7. Plan revision recommendations

- `workshop/plans/000035-stage-a-one-road-fix-plan.md` Task 0 Step 2 says "Set `estimate_hours: 5`" but the issue carries `estimate_hours: 2.05` from the later v3.1 derived block (commits 7003d98, 0f25bce). Add a `## Revisions` entry noting the estimate was superseded by the derived `estimate` block so the plan stops contradicting the issue frontmatter. Everything else in the Core concepts table matches the code as shipped.
