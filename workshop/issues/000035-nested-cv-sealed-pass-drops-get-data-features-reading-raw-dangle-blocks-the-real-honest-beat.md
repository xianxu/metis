---
id: 000035
status: working
deps: []
github_issue:
created: 2026-07-14
updated: 2026-07-14
estimate_hours: 1.74
started: 2026-07-14T07:43:17-07:00
---

# nested-CV sealed pass drops get-data → features reading raw dangle (blocks the real honest-beat)

## Problem

**`driver: cv` / nested CV cannot run a pipeline whose `features` step reads `raw: get-data` — which is
every real kbench titanic sweep.** Surfaced 2026-07-14 when the metis#32 migration rewrote the kbench
smoke e2e to actually run the sweep under nested CV for the first time (before, the nested path was only
exercised with toy pipelines + crafted ledgers). The real `titanic/features` step always reads the raw
Kaggle download (`with.raw: get-data`) to join the raw `Ticket` column (needed for `ticket_size` /
`ticket_survival` — the both-frames features). Under nested CV it fails:

```
FileNotFoundError: .../runs/<sealed>/get-data/train.csv
  features.py:282  raw_train = pd.read_csv(io.upstream_path(ctx, w["raw"], "train.csv"))
```

**Root cause** (`cmd/metis/sweep.go` `buildFoldExperiment`, sealed branch ~:604-607): for a sealed
outer-fold pass it repoints **only** `s.With["dataset"]` → `analysis_i` and `dropNeeds(ps.Needs,
dataIDs)` **drops get-data** — but it does NOT repoint the features step's **`raw: get-data`**. So `raw`
points at a step that isn't in the sealed experiment → the read dangles. (`dataset` is handled; `raw` and
any other get-data-referencing `with` leaf are not.)

This is exactly the **"`ticket_survival` is the first target-encoding feature ever swept under nested CV
— verify fit_mask at BOTH levels"** risk flagged in `kbench …/RUNBOOK-sweep.md §6.4`, now confirmed as a
hard failure. **It blocks the metis-v2 `done_when`** (the honest-beat nested run on real data) and the
kbench nested smoke e2e (xfailed against this issue).

## Spec

Two entangled concerns to resolve (brainstorm-first):
1. **Availability:** the sealed pass must make get-data's **raw** output reachable for a `raw`-reading
   step — likely repoint `raw: get-data` (and any get-data ref) to the **preamble's** materialized
   get-data output (`materializeOuterAnalysis` already runs `{data + outer-split}` once, so get-data's
   output exists), the way `dataset` is repointed to `analysis_i`. Generalize the sealed-branch repoint
   from just `dataset` to any leaf referencing a dropped data step.
2. **Leakage (the deeper half):** raw is the FULL train+test download, so a target-encoding feature that
   reads it in a sealed fold could see the assessment rows' labels. The intended protection is the
   **fit_mask** (the cross-fit excludes assessment rows from the target aggregate) applied at BOTH the
   inner (sweeper) and outer levels — not hiding the raw. Verify the fit_mask actually reaches the
   features step under nested CV at both levels (RUNBOOK §6.4's check): a `ticket_survival` config's
   outer honest estimate must NOT exceed its inner-CV by more than noise.

## Done when

- Nested CV runs the real kbench titanic sweep end-to-end under the ONE-ROAD model: `adapt` carries
  the source columns (schema role `source`), `features` reads only its base dataset (no `raw:`),
  `analysis_i` is shape-identical to the declared base (carries `test`). No dangling read.
- The kbench nested smoke e2e (`e2e/thread_test.py::test_sweep_smoke_composes_and_trains`) un-xfails + passes.
- The leakage tell (RUNBOOK §6 item 5) checked on the real honest-beat run: for each family whose
  inner-winner includes `ticket_survival`, the outer honest estimate does NOT exceed its inner CV
  beyond noise. (Outer-level protection = row absence in `analysis_i`; fit_mask = the INNER
  cross-fit — the original "fit_mask at both levels" framing was wrong.)
- The real honest-beat ran: `metis run titanic-sweep.md` → `select --best --promote` → operator
  submit; numbers recorded in the issue Log + project file (closes the metis-v2 `done_when`).

## Estimate

Derived per estimate-logic-v3.1 (design from v2 table with the thorough-plan 15% buffer;
impl at 40% of v2 ranges). Design is low across the board — the plan doc pre-resolves the
decisions; the two metis changes and adapt are well-specced module extensions; features is the
multi-file signature refactor (12 call sites, 2 non-mechanical); docs sweep spans two repos'
atlases; the e2e/honest-beat runs are the real-API operational budget; one boundary review.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module        design=0.1 impl=0.2
item: smaller-go-module        design=0.1 impl=0.2
item: cross-cutting-refactor   design=0.2 impl=0.2
item: atlas-docs               design=0.1 impl=0.05
item: atlas-docs               design=0.1 impl=0.05
item: real-api-discovery       design=0.0 impl=0.2
item: milestone-review         design=0.0 impl=0.15
design-buffer: 0.15
total: 1.74
```

## Plan

Durable plan: `workshop/plans/000035-stage-a-one-road-fix-plan.md` (review-hardened).

- [ ] Task 0 — issue revision + estimate + `sdlc change-code`
- [ ] Task 1 — metis: `source` schema role (TDD)
- [ ] Task 2 — metis: outer-split carries `test` (TDD)
- [ ] Task 3 — kbench: adapt carries source columns (TDD)
- [ ] Task 4 — kbench: features reads base only (TDD)
- [ ] Task 5 — kbench: shapes + relic deletion + RUNBOOK + atlas shadow-sweep
- [ ] Task 6 — un-xfail nested smoke e2e; full e2e green
- [ ] Task 7 — real honest-beat run (operator: kaggle submit)
- [ ] Task 8 — close out (atlas, PR + sdlc merge, close, lessons)

## Revisions

### 2026-07-14 — stage-A scope (supersedes the original Spec's approach)
- **Reason:** the brainstorm + research detour (see Log) rejected Spec item 1 (repoint `raw:` to
  the preamble's get-data): `upstream_path` bypasses the metis#23 confinement chokepoint, so that
  fix would hard-code a seal bypass into `metis/io.py`. Spec item 2's "fit_mask at BOTH levels"
  framing was diagnosed wrong: outer protection is row ABSENCE in `analysis_i`; the fit_mask does
  inner-level work only.
- **Delta:** the fix is the ONE-ROAD model (adapt carries source columns under a new `source`
  role; features drops `raw:`; outer-split carries `test` so `analysis_i` is a shape-identical
  stand-in), leaving the metis sweep engine untouched. Transductive semantics declared (RUNBOOK),
  not coded — the estimand knob is metis#36. Two sibling defects folded in: `analysis_i` lacked
  `test`; `adapt`'s `fare_median` reclassified as legitimate-under-transduction (not a bug).
  Done-when rewritten accordingly; plan at `workshop/plans/000035-stage-a-one-road-fix-plan.md`.
  Note: prior references to "RUNBOOK §6.4" (including this issue's Problem section) mean **§6
  list item 5** — miscitation, corrected at the RUNBOOK edit.

## Log

### 2026-07-14
- Filed from the metis#32 kbench migration: the rewritten nested smoke e2e is the first time nested CV
  ran through the real kbench `features` step (which reads `raw: get-data`), and it hard-fails —
  `buildFoldExperiment` drops get-data but repoints only `dataset`, not `raw`. Confirms RUNBOOK §6.4's
  flagged risk. Deps conceptually on metis#23 (the sealing) + kbench#8 (the ticket features). Blocks the
  metis-v2 `done_when` (honest-beat) and the kbench nested smoke e2e (xfailed pending this).
- Brainstorm (claimed, with operator) escalated the root cause from "missing repoint" to a design gap:
  the seal substitutes a derived artifact and deletes its producers, sound only if that artifact is the
  sole road raw→pipeline — `raw: get-data` is a second road. Two sibling defects surfaced: `analysis_i`
  lacks `test` (features.py:233 crashes once the raw read is fixed; ticket_size would silently differ
  selection-vs-ship), and `adapt`'s `fare_median` fits above the split. Converged on a general model
  (label = domain-restricted keyed channel; constructors = fit/apply with scope signatures; aggregate
  class decides fold-recompute) — captured in
  `workshop/pensive/2026-07-14-01-pensive-feature-engineering-algebra-under-cv.md`. Operator
  redirected to a research detour (literature + framework survey toward an "ML algebra extending
  relational algebra") before speccing the fix; #35's eventual fix will be an instance of that design.
- Detour done (2 deep-research passes + 27-agent adversarial verify; findings in the pensive).
  Operator agreed the 3-stage plan: **stage A = THIS issue** — the one-road fix on the current
  seal (adapt carries source cols via a new `source` schema role; features drops `raw:`;
  outer-split carries `test` through; declare transductive semantics; un-xfail e2e; run the real
  honest-beat) — closes metis-v2. Stage B = metis#36 (channel split, deletes the cloning seal).
  Stage C = metis#37 (constructor algebra, parked behind #36). This issue's scope stays stage A.
