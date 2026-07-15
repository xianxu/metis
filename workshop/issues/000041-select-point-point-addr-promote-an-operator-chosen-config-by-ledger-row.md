---
id: 000041
status: codecomplete
deps: []
github_issue:
created: 2026-07-14
updated: 2026-07-14
estimate_hours: 0.47
started: 2026-07-14T17:07:52-07:00
actual_hours: 0.20
---

# select --point <point_addr> — promote an operator-chosen config by ledger row

## Problem

metis#32's reconstruct-never-materialize deliberately removed committed winner files — but with
only `--best` / `--best-per-model-class`, there is now NO principled route to ship an
operator-chosen config. Concrete case (metis#35 honest-beat, 2026-07-14): the operator's prior
says insist on `ticket_survival` (grounded — the nested measurement structurally under-ranks
group features: labeled-partner coverage 38.6%→~30% under the seal); the best ticket config
(rf md=8 n=200, pooled inner 0.8297, Δ−0.0008 from the shipped pick — inside noise) cannot be
promoted by any command. A human-prior override is a production experiment the workbench should
support, auditably.

## Spec

- `metis select <shape> --point <point_addr> [--fingerprint <fp>] --promote`: look up the ledger
  row by `point_addr` (unique per (config, outer-fold, inner-fold); ANY row of the config works —
  verified on the 2026-07-14 ledger: 25 rows/config, 25 distinct addrs), read its `fp.*` tuple,
  reconstruct the ship experiment via the SAME `promotedExperiment` path `--best` uses, ship on
  all data → `runs/point-{family}-{hash}` (or reuse the `best-` naming with a `point-` prefix so
  provenance shows it was operator-chosen, not rule-chosen).
- Prefix matching on the addr (like git SHAs); ambiguous prefix = loud error listing candidates.
- The cohort guard applies unchanged (`--fingerprint` pins; a `--point` whose row is outside the
  pinned cohort is an error, not a silent cross-version ship).
- Without `--promote`: print the config's board line (pooled inner mean±SE, any outer rows) — the
  single-config inspect.
- **This is v1 of the ledger-publish track** (operator, 2026-07-14: "we need to be able to publish
  any rows in a ledger... first implement would just filter based on some form of row id"). The
  future extension on the SAME surface is `--where` predicates over the `fp.*` free-variable space
  (the sweep algebra reversed: constraints on the swept free variables). Design the selector flag
  so `--point` coexists with a later `--where` (both resolve to ledger rows → the same
  reconstruct path). Still out: any committed winner artifact.

## Done when

- `select --point <addr> --promote` ships a run whose record shows the reconstructed config ==
  the ledger row's `fp.*` tuple; prefix + ambiguity + wrong-cohort cases tested.
- The promoted run id makes the operator-chosen provenance visible (`point-` prefix or record
  field).
- The 2026-07-14 use case works end-to-end: promote the best ticket config from the b7aee3de
  cohort and `kaggle submit` it.

## Estimate

Derived per estimate-logic-v3.1 (thorough plan doc → 15% design buffer; impl at 40% of v2 ranges).
One well-specced Go module extension (resolve fn + flag + tests on existing fixtures) + the close
boundary review. Issue-spec/plan authoring already spent under #35's window.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module        design=0.1 impl=0.2
item: milestone-review         design=0.0 impl=0.15
design-buffer: 0.15
total: 0.47
```

Item→task mapping: smaller-go-module = Tasks 1–2 (resolve + promote, one module's surface);
milestone-review = the close boundary review (Task 3).

> Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
> `baseline-v3.1.md`. Method A only.

## Plan

Durable plan: `workshop/plans/000041-select-point-plan.md`.

- [x] Task 1 — resolve + errors (TDD)
- [x] Task 2 — promote path (TDD)
- [x] Task 3 — real-ledger verification (operator submit) + atlas/RUNBOOK + close

## Log

### 2026-07-14
- 2026-07-14: closed — go test ./... green (6 new TestSelectPoint_* incl. ambiguity, no-match, wrong-cohort, --best conflict, promote-reconstructs-row-config); real-ledger verification: --point 04fb2b62 on the b7aee3de cohort resolved rf md=8 n=200 all-6+tickets and printed 0.8297±0.0043 (matches independent computation); --promote shipped point-rf-3daa6310 end-to-end (features→train→predict→submission on all 891 rows).; review verdict: FIX-THEN-SHIP
- Filed from the metis#35 honest-beat session; operator scoped it as the v1 of publishing any
  ledger row (row-id filter first; `--where` predicates later on the same surface). The actuation
  seam for metis#40 (/metis-select skill). Sibling: kaggle#6 (submit auto-description). Immediate
  use waiting on it: promote rf md=8 n=200 [all-6+ticket_size+ticket_survival] from the b7aee3de
  cohort (the operator-prior production experiment).

- Built same-day (TDD, 6 new tests incl. ambiguity/no-match/wrong-cohort/conflict + promote
  reconstruct==row). Real-ledger verification: `--point 04fb2b62` resolved the operator's target
  config (rf md=8 n=200 all-6+tickets, pooled inner 0.8297±0.0043 — matches the hand computation),
  `--promote` shipped `point-rf-3daa6310` (submission.csv materialized; operator submits). Atlas +
  kbench RUNBOOK updated.
