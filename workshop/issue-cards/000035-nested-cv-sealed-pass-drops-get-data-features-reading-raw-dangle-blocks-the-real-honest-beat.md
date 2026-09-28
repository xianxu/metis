---
id: '000035'
status: done
started: 2026-07-14T07:43:17-07:00
created: 2026-07-14
updated: 2026-07-14
estimate_hours: 2.05
actual_hours: 3.89
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
