---
id: '000041'
status: done
started: 2026-07-14T17:07:52-07:00
created: 2026-07-14
updated: 2026-07-14
estimate_hours: 0.47
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
