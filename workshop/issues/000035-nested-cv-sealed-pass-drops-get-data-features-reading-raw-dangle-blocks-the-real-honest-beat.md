---
id: 000035
status: working
deps: []
github_issue:
created: 2026-07-14
updated: 2026-07-14
estimate_hours:
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

- A nested-CV run of a real kbench-style sweep (features reading `raw: get-data`, incl. `ticket_survival`)
  completes — the sealed pass reaches get-data's raw output, no dangling read.
- A leakage test proves the fit_mask excludes the assessment rows from the target-encoding aggregate at
  BOTH inner and outer levels (the outer honest estimate for a `ticket_survival` config tracks its inner
  CV within noise, not inflated).
- The kbench nested smoke e2e (`e2e/thread_test.py::test_sweep_smoke_composes_and_trains`) un-xfails + passes.

## Plan

- [ ] Brainstorm-first: the sealed-pass repoint generalization (any dropped-data-step ref, not just
  `dataset`) + the fit_mask-both-levels leakage verification. Then spec + change-code.

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
  `brain/workshop/pensive/2026-07-14-01-pensive-feature-engineering-algebra-under-cv.md`. Operator
  redirected to a research detour (literature + framework survey toward an "ML algebra extending
  relational algebra") before speccing the fix; #35's eventual fix will be an instance of that design.
- Detour done (2 deep-research passes + 27-agent adversarial verify; findings in the pensive).
  Operator agreed the 3-stage plan: **stage A = THIS issue** — the one-road fix on the current
  seal (adapt carries source cols via a new `source` schema role; features drops `raw:`;
  outer-split carries `test` through; declare transductive semantics; un-xfail e2e; run the real
  honest-beat) — closes metis-v2. Stage B = metis#36 (channel split, deletes the cloning seal).
  Stage C = metis#37 (constructor algebra, parked behind #36). This issue's scope stays stage A.
