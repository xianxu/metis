# Boundary Review — metis#23 (milestone M1)

| field | value |
|-------|-------|
| issue | 23 — nested-CV outer resample driver — honest procedure estimate |
| repo | metis |
| issue file | workshop/issues/000023-nested-cv-outer-resample-driver-honest-procedure-estimate.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 84fc04d72615dcb6b4cb44a1387f982896f9c584^..HEAD |
| command | sdlc milestone-close --issue 23 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-12T17:04:18-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M1 sealing spine is solid, correct, and genuinely well-tested — the load-bearing seal (out-of-root base-dataset read caught + named) is proven end-to-end through the *real* `exp_path` chokepoint, and the C1 regression the plan flagged (a legit run-dir handoff must bypass confinement) has its own explicit test. I verified the chokepoint is complete: every base-dataset read in the repo routes through `exp_path` (cv_split/outer_split direct) or `dataset_dir`'s exp-relative fallback (train/predict), while the upstream-handoff branch never touches `exp_path`. All 65 Python + all Go tests pass; the flat `driver:single` path is untouched (readRoot is declared but no production caller sets it in M1, so `METIS_READ_ROOT` is never injected — the mechanism is dormant until M2). Nothing blocks the boundary. The FIX-THEN-SHIP is for two cheap, non-blocking cleanups: a dead tautological assert in a test, and a stale entry in the plan's integration table.

### 1. Strengths
- **The seal is a real end-to-end test, not a mock reassert.** `test_seal_out_of_root_base_dataset_read_is_caught` (tests/test_outer_split.py:82) drives a *real* `cv_split.main` through the *real* `exp_path` chokepoint and asserts the confinement `RuntimeError` — it can't be faked, exactly as the plan demanded.
- **The C1 invisible-regression is covered.** `test_handoff_read_under_run_dir_not_confined` (tests/test_io_confinement.py:73) proves a run-dir handoff (sibling of the analysis root) passes while confined — the one case no other test exercises and the exact defect that would silently crash the M2 sealed sweep.
- **Positional correctness is verified against the source.** `test_outer_split_analysis_rows_carry_the_right_data` (tests/test_outer_split.py:65) checks `analysis_0` == the source's non-fold-0 rows in order — this catches the iloc-positional-vs-label bug class the subset materialization is exposed to.
- **Prefix-collision is handled and tested.** `within_root` (metis/io.py:34) uses `ar + os.sep` with an explicit equal-case, and `test_within_root_false_for_prefix_collision` pins `/data/analysis_00` ∉ `/data/analysis_0`.
- **Chokepoint is single-sourced (ARCH-DRY).** The assertion lives in exactly one place (`exp_path`), not scattered into each step; injection is guarded (`e.readRoot != ""`, exec.go:67) so the flat path stays unconfined.

### 2. Critical findings
None.

### 3. Important findings
None. (The plan-table mismatch below is plan-doc hygiene, not a code defect — the binding task, plan §1.2, matches the code; only the summary table is stale. Listed under §7.)

### 4. Minor findings
- **tests/test_outer_split.py:62** — `assert list(a0.train.index) != list(range(len(a0.train))) or True` is a tautology (`X or True` ≡ `True`); it asserts nothing. Delete it — the real positional check is the `.equals(...)` on line 65.
- **cmd/metis/exec.go:60,67** — `cmd.Env = append(os.Environ(), …)` then conditionally appends `METIS_READ_ROOT`. If the ambient shell already has `METIS_READ_ROOT` set, the flat path inherits it (readRoot=="" doesn't unset it). Failure mode is *fail-loud* (spurious confinement error on a legit read), i.e. the safe direction, and the var is internal-only — so this is defensive polish, not a bug. If you want it airtight, strip `METIS_READ_ROOT` from the inherited env when `e.readRoot == ""`.
- **metis/steps/outer_split.py:36-37 vs cv_split.py:26-27** — the `stratify_col = ds.schema.target_col() if w.get("stratify") else None` + `cv_folds(...)` pair is duplicated across both steps (ARCH-DRY, minor). Two trivial lines that diverge immediately after; not worth forcing a helper unless a third caller appears (kbench#8 grouped folds may be that caller — worth watching).

### 5. Test coverage notes
Coverage is strong and targets the shipped bug classes: predicate edge cases (child/root-itself/sibling/prefix-collision), env on/off, the exp_path chokepoint both halves, the handoff bypass, subset disjoint+covering, positional row correctness, and the end-to-end seal both directions. `StepContext.read_root` decode is pinned (test_step_context_decodes_read_root). No coverage gap that would let an M1-class bug through. The only untested runtime is the *symlink* bypass of `os.path.abspath` (no realpath) — but that is the explicitly documented-and-deferred syscall-airtightness gap, correctly out of M1 scope.

### 6. Architectural notes for upcoming work
- **ARCH-DRY — PASS.** Single chokepoint; env contract single-sourced per language side. `within_root` is a legitimately new predicate (trace.py:124's run-dir exclusion is a strict-child check without the equal-case — different semantics, not a mergeable dup). For M2/#20/kbench#8: keep the confinement at `exp_path` — resist the temptation to add asserts in `load_dataset`/`dataset_dir`-upstream (that's the C1 crash).
- **ARCH-PURE — PASS.** `within_root` is the pure core; `assert_within_read_root` is thin env-reading glue at the boundary; `outer_split.py` is a clean io→pure(`cv_folds`)→io shell.
- **ARCH-PURPOSE — PASS.** M1's stated purpose is the *sealing spine* (L1 subset dirs + L2 chokepoint) tested in isolation, explicitly not the driver (M2). Shadow-sweep of the single-source claim: confinement is *enforced* at one chokepoint that every base-dataset consumer derives from (verified across cv_split/train/predict/outer_split) — no hand-maintained restatement, no deferred-purpose masquerading as follow-up.
- For M2: `outer_split` reads unconfined by relying on the caller leaving `METIS_READ_ROOT` unset. When M2 orchestrates outer-split *above* the driver, guarantee readRoot is empty for that step, or exp_path will raise on the legitimate full-dataset read.

### 7. Plan revision recommendations
Add a `## Revisions` entry to `workshop/plans/000023-nested-cv-outer-resample-driver-plan.md`:
- **Integration-points table staleness.** The `chokepoint assertion` row states its location as `metis/io.py (load_dataset/dataset_dir/exp_path)`. Per the C1 correction (which Task 1.2 states correctly and the code implements), the assertion lives **only in `exp_path`** — `load_dataset` and `dataset_dir`-upstream must *not* assert (they serve handoff reads). Update the row to `metis/io.py (exp_path only — C1)` so the M2 implementor isn't misled into scattering the assertion.
- **"outer-partition" vs "outer-split" naming.** The plan's section headers and integration table call it the "outer-partition step"; the delivered code, wrapper, and atlas consistently use **outer-split** / `outer_split.py`. Normalize the plan to "outer-split" to match the shipped surface.
