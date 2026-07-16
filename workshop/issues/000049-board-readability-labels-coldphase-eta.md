---
id: 000049
status: open
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-16
estimate_hours:
---

# board readability — label semantics, cold-phase "no progress" confusion, jumpy leaves, wild early ETA

## Problem

Operator's first real-sweep board session (titanic-sweep.md, BLAS-pinned, 2026-07-16) surfaced
four readability issues — the board is mechanically correct but hard to READ in exactly the
early phase where the operator most wants reassurance:

1. **Label semantics unclear:** `outer 0/10 · configs 0/720 · folds 0/7200` — operator asked
   "is folds about inner folds?" `folds` = leaf-level (config × inner-fold) RUNS aggregated
   across the whole sweep (10 outer × 72 configs × 10 inner); `configs` = per-outer-pass
   config completions aggregated (72 × 10). Neither is legible from the labels.
2. **Cold-phase "lack of progress":** early in a cold run every fold row shows `queued`, all
   counters sit at 0 for many minutes. Root cause is the metis#43 phase-wave scheduling (all
   cv-splits/features across the grid complete before ANY train chain finishes → zero fold
   completions for ~10 min while heavy upstream work happens). The board renders that
   truthfully but unhelpfully — nothing distinguishes "working through upstream steps" from
   "hung". (#43 fixes the schedule; this issue covers what the board shows MEANWHILE.)
3. **`leaves 2/12` jumps around:** the gauge samples instantaneous `len(leafSem)` at flush
   time — honest, but at 4Hz it reads as noise, and low occupancy during the upstream herd
   phase looks like a problem when it isn't.
4. **ETA changes wildly:** the 64-completion moving window over sparse, phase-heterogeneous
   early completions (fast cache-hit folds vs slow cold trains) swings the rate — the ETA
   flapped across hours on the operator's run. An estimate that volatile is worse than none.

## Spec (directions, refine at design)

1. Rename/annotate segments (e.g. `inner-folds 421/7200` or a one-line legend on the first
   frame); consider showing per-outer-fold denominators in the fold rows as the source of
   truth for "what is 7200".
2. A PHASE indicator when zero folds have completed but leaves are active (e.g.
   `warming: upstream steps running · 0 folds complete yet`) — distinguish wave-phase from
   hang using signals the sink already has (leaf occupancy > 0, passthrough step lines
   flowing).
3. Smooth the leaves gauge (peak-or-mean over the flush window, or EWMA) — the sink already
   samples at each event; keep a tiny window instead of the instant.
4. ETA damping: suppress until n≥K completions AND the window spans ≥T seconds; consider
   EWMA on the rate; show a range or `~` marker while confidence is low. (The stall-decay
   property — rate → 0 on thrash — must SURVIVE damping; that alarm is the line's purpose.)

## Done when

- Operator can read the first 2 minutes of a cold real sweep without wondering if it hung:
  phase indicator present until the first fold completes; labels self-explanatory.
- Leaves + rate/ETA move smoothly (unit tests over scripted event traces pin the damping);
  the BLAS-thrash decay signature still shows within seconds (regression test).
- RUNBOOK screenshot/description updated.

## Plan

- [ ] (at claim) Design the four refinements against a real cold-sweep event trace; TDD.

## Log

### 2026-07-16
- Filed from the operator's mid-run feedback on the first real-sweep board session (pins set,
  full 7,200-fold grid). Companion issues from the same session: metis#47 (flash — FIXED,
  DEC 2026 sync output), metis#48 (default BLAS pins — the 3h-ETA root cause when unpinned),
  metis#43 (the phase-wave scheduler, pre-existing). The wild-ETA observation is partly a
  #43 symptom: depth-first scheduling would give early train completions → a stable rate
  much sooner.
