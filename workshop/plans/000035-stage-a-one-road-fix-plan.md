# metis#35 Stage A — One-Road Fix Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `adapt`'s output the *only* road from raw data to the pipeline — so the nested-CV
seal's substitution (`analysis_i` for the base) is sound — then run the real honest-beat that
closes the metis-v2 `done_when`.

**Architecture:** Three coupled changes restore the seal's invariant: (1) `adapt` carries the five
messy source columns (`Name/Age/Cabin/Embarked/Ticket`) into the base Dataset under a new `source`
schema role (carried, never a model input — `id`'s semantics); (2) `features` drops its
`raw: get-data` back-channel and reads source columns from the base; (3) `outer-split` carries
`test` through into each `analysis_i` so the sealed stand-in is shape-identical to the declared
base (keeps both-frames features like `ticket_size` consistent between selection and ship). No
metis sweep-engine change: `buildFoldExperiment`'s existing `dataset` repoint now covers every
read. Estimand decision recorded, not coded: **transductive** semantics (Kaggle) — `adapt`'s
`fare_median` and full-frame group features are legitimate because the deployed transform sees the
same rows (stage B, metis#36, makes this a declared knob).

**Tech Stack:** Go (metis CLI — untouched), Python (metis.io/schema/steps, kbench titanic steps),
pytest, `sdlc`.

**Research notes:** `workshop/pensive/2026-07-14-01-pensive-feature-engineering-algebra-under-cv.md`
(the design model this fix instantiates the precondition of). Stage B = metis#36, stage C = metis#37.

**ARCH notes:**
- **ARCH-PURPOSE:** the purpose is *one road* — every consumer of raw columns derives from
  `adapt`'s Dataset. The shadow-sweep at the end enumerates every `raw:`-reading surface (steps,
  shapes, tests, docs) and confirms none remain except `adapt` itself.
- **ARCH-DRY:** the source-column list moves to its producer (`adapt.SOURCE_COLS`);
  `features.RAW_SOURCE_COLS` and the join logic are deleted, not paralleled.
- **ARCH-PURE:** `build_titanic_dataset` and `apply_features` stay pure (frame-in/frame-out);
  the only IO change is `outer_split.py`'s save call gaining `test=ds.test`.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Schema.ROLES` (`source` role) | `metis/metis/schema.py` | modified |
| `build_titanic_dataset` | `kbench/kbench/titanic/adapt.py` | modified |
| `SOURCE_COLS` | `kbench/kbench/titanic/adapt.py` | new |
| `apply_features` | `kbench/kbench/titanic/features.py` | modified |
| `RAW_SOURCE_COLS` | `kbench/kbench/titanic/features.py` | deleted |

- **`Schema.ROLES` + `source`** — a fifth column role: carried through datasets, never a model
  input (exactly `id`'s semantics; `feature_cols()` already excludes non-`feature` roles, so
  `train`/`predict` need no change).
  - **Relationships:** N source columns per Schema; consumed only by steps that declare knowledge
    of them (kbench `features`).
  - **DRY rationale:** without a role, source columns would need role `feature` (wrong — string
    dtype, would enter the model) or a parallel side-table (a second mechanism).
  - **Future extensions:** metis#36's `y`-channel split will lean on roles as the typed surface.
- **`build_titanic_dataset`** — now emits base 5 features + target + the 5 carried source columns
  (role `source`, dtypes as-is: `object`/`float64`, **NaN allowed in source columns only**). The
  "numeric, NaN-free" contract narrows to feature/target roles; docstring updated.
- **`SOURCE_COLS`** — the carried-column list, defined at its producer (`adapt`). `features`
  imports it for its presence guard.
- **`apply_features`** — signature loses `raw_train`/`raw_test`; the PassengerId join is deleted
  (base already carries the columns); adds a loud guard naming metis#35 when source columns are
  absent (an old cached `../data/titanic` would otherwise KeyError obscurely). Group functions
  are untouched — they already read column names off the work frames.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `outer_split.main` | `metis/metis/steps/outer_split.py` | modified | dataset dir IO |
| `features.main` | `kbench/kbench/titanic/features.py` | modified | step contract IO |
| shape files (3) | `kbench/competition/titanic/pipelines/*.md` | modified | `metis run` |

- **`outer_split.main`** — one line: `Dataset(schema=ds.schema, train=analysis, test=ds.test)`.
  Invariant bought: *`analysis_i` differs from the declared base only in its train rows, never in
  shape* — any pipeline that runs flat also runs sealed.
- **`features.main`** — drops the two `pd.read_csv(io.upstream_path(ctx, w["raw"], …))` reads and
  the `raw` key.
- **Shapes** — `titanic-sweep.md`, `titanic-sweep-smoke.md`, `titanic-features.md`: the features
  step loses `raw: get-data` from `with` and `get-data` from `needs`. (`adapt`'s own
  `raw: get-data` stays everywhere — it's the demultiplexer.)
- **Existing fake:** e2e already uses `fake-kaggle` (process-level fake) — reused, nothing new.

**Repos/branches:** metis work on branch `issue-35` (PR + `sdlc merge` at the end, as #31/#32);
kbench work on local `main` (its convention — operator pushes). kbench e2e builds metis from the
sibling worktree, so keep the metis worktree on `issue-35` while running kbench tests.

---

## Chunk 0: bookkeeping

### Task 0: issue revision + estimate

**Files:**
- Modify: `workshop/issues/000035-nested-cv-sealed-pass-drops-get-data-features-reading-raw-dangle-blocks-the-real-honest-beat.md`

- [ ] **Step 1:** Append a `## Revisions` section to the issue: stage A supersedes the original
  Spec's approach 1 (repoint `raw` to the preamble — rejected: it would hard-code a bypass of the
  metis#23 seal into `metis/io.py`); approach 2's fit_mask-at-both-levels framing corrected (outer
  protection = row absence in `analysis_i`; fit_mask is inner-level) — pointer to this plan +
  the pensive. Rewrite `## Done when` to the stage-A list (one-road nested run completes; e2e
  un-xfailed; §6.4 leakage tell checked on the honest-beat run; honest-beat submitted).
- [ ] **Step 2:** Set `estimate_hours: 5` in the issue frontmatter.
- [ ] **Step 3:** Commit (metis, `main` — issue files live on main):
  `#35: revise Spec to stage A (one-road fix); estimate`. Then create branch: `git checkout -b issue-35`.
- [ ] **Step 4:** Run `sdlc change-code --issue 35` (plan-quality gate reads this plan).

## Chunk 1: metis — schema role + outer-split test carry

### Task 1: `source` role in Schema

**Files:**
- Modify: `metis/metis/schema.py` (ROLES + docstring)
- Test: `metis/tests/test_schema.py`

- [ ] **Step 1: Write the failing tests** (append to `tests/test_schema.py`):

```python
def test_source_role_accepted_and_excluded_from_features():
    s = Schema(
        columns={"id": "id", "y": "target", "f": "feature", "Name": "source"},
        dtypes={"id": "int64", "y": "int64", "f": "float64", "Name": "object"},
    )
    assert s.feature_cols() == ["f"]          # source never a model input
    assert s.target_col() == "y"
```

- [ ] **Step 2: Run to verify it fails** —
  `cd /Users/xianxu/workspace/metis && .venv/bin/python -m pytest tests/test_schema.py -q`
  Expected: FAIL — `unknown column role(s): {'Name': 'source'}`.
- [ ] **Step 3: Implement** — in `metis/schema.py`: `ROLES = frozenset({"id", "feature", "target",
  "weight", "source"})`; extend the module/`ROLES` comment: *`source` = a raw column carried
  through for feature-engineering steps that know it; never a model input (metis#35)*.
- [ ] **Step 4: Run to verify it passes** — same command. Expected: PASS (whole file green).
- [ ] **Step 5: Commit** (metis, `issue-35`): `#35: schema role 'source' — carried raw columns, never model input`.

### Task 2: outer-split carries `test`

**Files:**
- Modify: `metis/metis/steps/outer_split.py` (the `save_dataset` call + docstring)
- Test: `metis/tests/test_outer_split.py`

- [ ] **Step 1: Write the failing test** — in `test_outer_split.py`, mirror the existing
  `_run_step` fixture with the `"toy"` dataset (`metis/testdata/dataset/toy/` already has
  `test.csv` — no new fixture needed):

```python
def test_analysis_dirs_carry_test_frame(tmp_path, monkeypatch):
    # drive the step exactly like the existing test; toy has a non-None test frame
    ...
    full = io.load_dataset(str(TOY_DIR))
    for i in range(3):
        sub = io.load_dataset(str(sd / f"analysis_{i}"))
        assert sub.test is not None and len(sub.test) == len(full.test)  # shape-identical stand-in (metis#35)
        assert list(sub.test.columns) == list(full.test.columns)
```
- [ ] **Step 2: Run to verify it fails** — `... -m pytest tests/test_outer_split.py -q`
  Expected: FAIL — `sub.test is None`.
- [ ] **Step 3: Implement** — `outer_split.py`: `io.save_dataset(Dataset(schema=ds.schema,
  train=analysis, test=ds.test), io.out_path(ctx, f"analysis_{i}"))`; docstring: *analysis_i is a
  SHAPE-IDENTICAL stand-in for the declared base — only train rows differ (metis#35); test rows
  are unlabeled and carry no assessment labels, so carrying them is seal-neutral.*
- [ ] **Step 4: Run to verify it passes** — same command; then the full metis suite:
  `.venv/bin/python -m pytest tests -q` and `go test ./...`. Expected: all green (no Go change).
- [ ] **Step 5: Commit** (metis, `issue-35`): `#35: outer-split carries test — analysis_i shape-identical to base`.

## Chunk 2: kbench — adapt carries, features stops reaching

### Task 3: adapt carries source columns

**Files:**
- Modify: `kbench/kbench/titanic/adapt.py`
- Test: `kbench/kbench/titanic/adapt_test.py`

- [ ] **Step 1: Write the failing tests** (append to `adapt_test.py`):

```python
def test_source_columns_carried_with_source_role():
    ds = build_titanic_dataset(_raw_train(), _raw_test())
    for c in ["Name", "Age", "Cabin", "Embarked", "Ticket"]:
        assert ds.schema.columns[c] == "source"
        assert c in ds.train.columns and c in ds.test.columns
    assert ds.schema.feature_cols() == FEATURES     # model inputs unchanged (regression anchor)
    # source columns are carried VERBATIM — NaN allowed there (only feature/target are NaN-free)
    assert ds.train["Name"].equals(_raw_train()["Name"])
```

  (Use the file's existing raw fixtures; if they lack a `Ticket`/`Cabin` column, extend the
  fixtures rather than relaxing the test.)
- [ ] **Step 2: Run to verify it fails** —
  `cd /Users/xianxu/workspace/kbench && .venv/bin/python -m pytest kbench/titanic/adapt_test.py -q`
  Expected: FAIL — KeyError `'Name'` in schema columns.
- [ ] **Step 3: Implement** — in `adapt.py`:
  - `SOURCE_COLS = ["Name", "Age", "Cabin", "Embarked", "Ticket"]` (module constant, ordered).
  - In `_features()` (or just after building train/test in `build_titanic_dataset`): carry
    `out[c] = df[c]` for each `c in SOURCE_COLS`.
  - `columns` gains `{c: "source" for c in SOURCE_COLS}`; `dtypes` gains
    `{c: str(train[c].dtype) for c in SOURCE_COLS}`.
  - Docstring: contract is now *numeric, NaN-free over feature/target roles; the 5 messy source
    columns carried verbatim under role `source` (metis#35 one-road — `features` reads them from
    the base instead of re-reading get-data's csvs)*.
- [ ] **Step 4: Run to verify it passes** — same command. Expected: PASS. (The existing
  `test_schema_roles_and_target` may enumerate columns exactly — extend it, don't weaken it.)
- [ ] **Step 5: Commit** (kbench, `main`): `metis#35: adapt carries source columns (role 'source') — the one road`.

### Task 4: features reads the base only

**Files:**
- Modify: `kbench/kbench/titanic/features.py`
- Test: `kbench/kbench/titanic/features_test.py`

- [ ] **Step 1: Update the tests first** (they define the new contract):
  - `_base_dataset()` fixture: add the five source columns to both frames (values from the
    existing `_raw_train()`/`_raw_test()` fixtures, joined by PassengerId order), with
    `columns[c] = "source"` + dtypes.
  - Every `apply_features(base, _raw_train(), _raw_test(), …)` call →
    `apply_features(base, …)` (~12 sites — mechanical EXCEPT two, called out below).
  - **Two NON-mechanical sites** (they construct ticket groups by overriding the raw frames):
    `test_ticket_survival_is_leakage_safe_through_apply_features` (~:449-452) and
    `test_ticket_features_flow_through_the_step` (~:462-466) override `raw_tr["Ticket"]` /
    `raw_te["Ticket"]` — those overrides must move onto the **base** frames (the source columns
    now live there), and `_run_features_step` loses its `raw_tr`/`raw_te` parameters. (kbench's
    own `workshop/lessons.md:39-40` records a prior "mechanical/unchanged" mis-claim in exactly
    this area — treat these two with care.)
  - The step-contract tests (`_run_features_step` users: `test_step_passthrough_reproduces_base`,
    `test_pclass_survival_flows_through_the_step`, `test_ticket_features_flow_through_the_step`):
    drop `raw` from `with_cfg`; stop writing `gd/train.csv`+`gd/test.csv` for the features step.
  - Add the guard test:

```python
def test_missing_source_columns_fail_loud():
    base = _base_dataset_without_source()   # the OLD fixture shape
    with pytest.raises(ValueError, match="metis#35"):
        apply_features(base, {"title"})
```

- [ ] **Step 2: Run to verify failures** — `... -m pytest kbench/titanic/features_test.py -q`
  Expected: FAIL — signature mismatch (`apply_features() takes …`).
- [ ] **Step 3: Implement** — in `features.py`:
  - `from kbench.titanic.adapt import SOURCE_COLS`; delete `RAW_SOURCE_COLS`.
  - `apply_features(base, features, fit_mask=None, seed=0)`: replace the two `.merge(...)` lines
    with a guard + copies:

```python
    missing = [c for c in SOURCE_COLS if c not in base.train.columns]
    if missing:
        raise ValueError(
            f"base dataset lacks source column(s) {missing}: re-run adapt "
            f"(metis#35 one-road — features no longer reads get-data's raw csvs)")
    work_train = base.train.copy()
    work_test = base.test.copy()
```

  - `main()`: delete the two `raw_train/raw_test = pd.read_csv(io.upstream_path(...))` lines and
    the `w["raw"]` use; call `apply_features(base, features, fit_mask, seed=ctx.seed)`.
  - Output assembly is unchanged (source columns are dropped — `[ID] + feat_cols + [TARGET]`).
  - Docstring: rewrite the "Boundary" paragraph — *adapt carries the source columns (one road,
    metis#35); this step reads only its base dataset. Transductive semantics: full-frame fits
    (fare_median in adapt, ticket_size over train+test) are legitimate because the shipped
    transform sees the same rows; metis#36 makes this a declared estimand.*
- [ ] **Step 4: Run to verify** — features + adapt + full kbench unit suite:
  `... -m pytest kbench -q`. Expected: PASS.
- [ ] **Step 5: Commit** (kbench, `main`): `metis#35: features reads source cols from base — raw: back-channel deleted`.

### Task 5: shapes + docs sweep (the ARCH-PURPOSE shadow-sweep)

**Files:**
- Modify: `kbench/competition/titanic/pipelines/titanic-sweep.md`,
  `titanic-sweep-smoke.md`, `titanic-features.md` (features step: drop `raw: get-data`, `needs:
  [adapt, get-data]` → `[adapt]`)
- Modify: `kbench/competition/titanic/pipelines/RUNBOOK-sweep.md` (§6.4 + prose)

- [ ] **Step 1:** Edit the three shapes' features step. `titanic-baseline.md` has no features
  step — only adapt's `raw:`, which stays.
- [ ] **Step 2: Relic winner experiments** — four pre-#32 committed winners
  (`competition/titanic/pipelines/titanic-winner.md`, `-rf.md`, `-v3.1.md`, `-v3.2.md`) carry
  features steps with JSON-form `"raw":"get-data"`. metis#32's reconstruct-never-materialize made
  committed winner files obsolete; **delete the four relics** (git history preserves the
  provenance of what was actually submitted; editing their bytes would falsify it, leaving them
  would keep un-runnable shapes in the tree). Name the deletion prominently in the commit body so
  the operator sees it at review.
- [ ] **Step 3:** RUNBOOK: update §6 — the leakage tell is **§6 list item 5**
  (`RUNBOOK-sweep.md:109-112`; the plan/issue/pensive's "§6.4" is a miscitation — fix the issue
  text in the Task 0 revision and the pensive line while here). Answer the flagged question:
  outer protection = row absence in analysis_i; fit_mask = inner cross-fit; the empirical tell
  (ticket_survival outer ≤ inner + noise) stays as the honest-beat acceptance check. Add one line
  declaring **transductive semantics** (why fare_median/ticket_size full-frame fits are legitimate
  under it), pointing at metis#36 for the knob.
- [ ] **Step 4: Shadow-sweep** — from the kbench **repo root** (so `atlas/`, `workshop/`, and
  generated artifacts are in the net):
  `grep -rn 'raw. *get-data\|RAW_SOURCE_COLS\|w\[.raw.\]' . --include='*.py' --include='*.md' --include='*.json'`
  (pattern matches BOTH the YAML `raw: get-data` and JSON `"raw":"get-data"` serializations —
  derive sweep patterns from the key name, not one formatted form). Also
  `grep -rn 'raw. *get-data' /Users/xianxu/workspace/metis/ --include='*.go' --include='*.py' --include='*.md'`.
  Expected leftovers: `adapt`-step lines in shapes + issue/plan/pensive prose. Fix stragglers.
- [ ] **Step 5: kbench atlas** — `atlas/titanic-workspace.md:34-38,63` documents the old two-road
  boundary verbatim ("re-reads get-data's raw csvs", `apply_features(base, raw_train, raw_test,
  …)`). Reconcile to the new truth (atlas holds current state only). Check `atlas/index.md` for
  any other page mapping the features flow.
- [ ] **Step 6: Commit** (kbench, `main`): `metis#35: shapes drop features raw: — one road; relics deleted; RUNBOOK + atlas reconciled`.

## Chunk 3: the proof — e2e + the real honest-beat

### Task 6: un-xfail the nested smoke e2e

**Files:**
- Modify: `kbench/e2e/thread_test.py` (delete the `@pytest.mark.xfail(...)` block at ~:141-146)

- [ ] **Step 1:** Delete the xfail decorator.
- [ ] **Step 2: Run the full e2e** — `cd /Users/xianxu/workspace/kbench && .venv/bin/python -m pytest e2e/thread_test.py -q`
  (builds metis from the sibling worktree on `issue-35`). Expected: ALL PASS, including
  `test_sweep_smoke_composes_and_trains` — the first-ever green nested run through the real
  features step. If it fails, STOP and diagnose (this is the integration truth the crafted
  fixtures missed — lessons.md).
- [ ] **Step 3: Commit** (kbench, `main`): `metis#35: un-xfail nested smoke e2e — nested CV runs the real pipeline`.

### Task 7: the honest-beat run (closes the metis-v2 done_when)

**Files:** none (operational; record results in issue `## Log` + the project file)

- [ ] **Step 1:** Real sweep: `metis run competition/titanic/pipelines/titanic-sweep.md` (from
  kbench, metis binary from the `issue-35` worktree). **Operator dependency starts HERE, not at
  submit**: `get-data` is a live `kaggle/download` — confirm the download is cache-hit from prior
  runs (get-data's definition is unchanged, so it should be) or that credentials are available
  before launching. Record the nested-CV honest estimate (mean±SE) per family.
- [ ] **Step 2:** §6-item-5 leakage tell: the ledger's OUTER rows are **per-family winners** —
  a `ticket_survival` outer estimate exists only when a family's inner-winner config includes
  that feature. Read `fp.features` on the outer rows to find them; compare each such config's
  outer honest estimate vs its inner-CV mean — outer must NOT exceed inner beyond noise. Record
  the numbers in the issue Log.
- [ ] **Step 3:** `metis select --best` — does the honest family estimate pick the generalizer?
  Then `metis select --best --promote` → ship on all data.
- [ ] **Step 4:** `kaggle submit --run <ship-run-id>` — **operator step** (real submission,
  credentials + rate limits are theirs). Ask before running; the public score vs the honest
  estimate is the metis-v2 `done_when` evidence.
- [ ] **Step 5:** Update `brain/data/project/metis-v2-experiment-algebra.md` (honest-beat result,
  done_when status) + issue `## Log`.

### Task 8: close out

- [ ] **Step 1:** metis atlas: note the `source` role (schema surface) + analysis_i shape
  invariant wherever the nested-CV flow is mapped (`atlas/`, follow `atlas/index.md`).
- [ ] **Step 2:** metis: open PR from `issue-35`, `sdlc merge` (boundary review runs there).
  kbench: commits stay on local main (operator pushes — flag it).
- [ ] **Step 3:** `sdlc close --issue 35 --verified '<e2e green + honest-beat numbers>'` — let
  close compute actuals (or `sdlc actual --issue 35`); if the session's interleaving contaminates
  active-time, record N/A with the reason (per lessons).
- [ ] **Step 4:** Add new lessons to `workshop/lessons.md` (candidates: "a substitution seal is
  sound only if the substituted node is the sole road — check for bypass edges when deleting
  producers"; "a shadow-sweep grep must match every serialization of the reference (YAML `raw:
  get-data` AND JSON `\"raw\":\"get-data\"`) — derive the pattern from the key name, run from the
  repo root"; "a two-repo fix must enumerate BOTH repos' atlases — the close gate only checks the
  repo you close in").
