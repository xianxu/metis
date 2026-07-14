---
id: 000041
status: open
deps: []
github_issue:
created: 2026-07-14
updated: 2026-07-14
estimate_hours:
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

## Plan

- [ ] Small: a ledger row lookup + the existing promotedExperiment path. TDD on a fixture ledger;
  e2e assert reconstruct==row.

## Log

### 2026-07-14
- Filed from the metis#35 honest-beat session; operator scoped it as the v1 of publishing any
  ledger row (row-id filter first; `--where` predicates later on the same surface). The actuation
  seam for metis#40 (/metis-select skill). Sibling: kaggle#6 (submit auto-description). Immediate
  use waiting on it: promote rf md=8 n=200 [all-6+ticket_size+ticket_survival] from the b7aee3de
  cohort (the operator-prior production experiment).
