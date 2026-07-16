---
id: 000049
status: working
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-16
estimate_hours:
started: 2026-07-16T12:57:08-07:00
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

## Spec

### Claim-time directions (retained)

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

### Approved design — typed, time-driven board telemetry

#### Counter vocabulary

Rename the shared `progressCore` labels once (plain output and TUI derive from it; ARCH-DRY), while
letting the already-known `nested` state select truthful mode-specific vocabulary:

- `outer folds k/n` — completed held-out outer-fold evaluations.
- `configs scored k/n` — configs whose full inner resample completed, aggregated across outer folds.
- Nested: `inner-CV runs k/n` — completed `(outer fold × config × inner fold)` pipeline runs. Flat:
  `CV runs k/n` — completed folds in the one single-level CV. These replace the ambiguous `folds`;
  per-outer rows use `outer fold N` and the nested `configs scored` / `inner-CV runs` vocabulary
  with per-pass denominators. Startup, rate, and ETA wording uses the same mode-specific noun.

#### Typed activity seam

- Add a thin `activityExecutor` decorator around the **final** step executor, outside the cache
  decorator. It emits a typed successful-step completion after either a real execution or cache hit;
  no terminal text is parsed, and failed steps continue through the existing fatal error path without
  being counted as reassuring activity (ARCH-PURE).
- The same injected activity sink has a distinct successful-run event emitted by
  `runResolvedExperiment` only when execution has `runErr == nil` **and** required run-record /
  provenance persistence succeeds. A failed execution remains uncounted even when its failure record
  is written successfully. Its typed run role distinguishes nested inner-CV, flat CV, preamble, and
  outer-score runs; only the first two feed the corresponding run counter/rate/ETA. This records
  actual completion time at the concrete-run seam, rather than the later input-ordered `sampler.Run`
  delivery of a completed batch (ARCH-PURPOSE).
- Events carry an injected-clock time and typed identity. `sweepProgress` reduces successful-step
  events to a count/last-step time and eligible run events to the run count/rate window. Concurrent
  callback delivery may differ from event-time order, so each last-time is a max and the latest 64
  eligible run timestamps remain sorted by event time before readiness/rate math. Reversed-delivery
  traces pin both invariants. Callbacks are short and mutex-protected; non-sweep callers use a no-op
  sink.
- Before the first eligible run completes, the board reports observations, never an unprovable
  diagnosis. Nested example: `starting · ~8/12 subprocess slots · 37 steps completed · last step 1s
  ago · no inner-CV run complete`; flat output substitutes `CV run`. With occupancy but no successful
  step yet, it says only that slots are occupied. It never claims “not hung” or infers an upstream
  phase from a start line.
- The startup line disappears on the first eligible inner-CV/CV run. This remains correct both before
  and after #43: #43 shortens startup; #49 truthfully describes it.

#### Time-driven smoothing and ETA confidence

- Sample leaf occupancy on the existing 500ms board tick, not per event. Display the rolling mean of
  the last four samples (two seconds) rounded to a whole slot and prefixed `~`. Equal timestamped
  occupancy traces must render identically regardless of event density.
- Keep the latest 64 eligible run-completion timestamps, but withhold rate and ETA until at least 16
  completions span at least 15 seconds. Compute rate as `(n−1) / (now−oldest)`, removing the current
  early upward bias that counts `n` completions across `n−1` observed intervals.
- Continue using the current time in the denominator on every 500ms tick. With no new completions, the
  numeric rate is non-increasing and ETA is non-decreasing, although display rounding need not change
  within an arbitrary number of seconds after a long mature window. Preserve the fast, truthful stall
  signal separately: after startup, show `last inner-CV run 8s ago` (or `last CV run …` flat), updated
  on every tick. A scripted mature trace followed by five seconds of silence must advance that age by
  five seconds while rate/ETA move monotonically; it makes no diagnosis about why work is quiet
  (ARCH-PURPOSE). After a stall, the 64-completion window recovers gradually rather than snapping to
  one interarrival.
- Render the mature estimate as `~ETA …`; nested mode labels its rate `inner-CV runs/min` and scopes
  ETA to remaining inner-CV runs, while flat mode uses `CV runs/min` and remaining CV runs. Neither
  includes held-out scoring, capture, or final ledger work. Before confidence, show the corresponding
  mode-specific `— … runs/min` and no ETA rather than a volatile fiction.

#### Documentation boundary

Update the kbench Titanic sweep RUNBOOK's board description/example after metis#49 lands. Record the
exact peer commit in this issue's Log before close so the cross-repo requirement is reviewable.

## Done when

- Operator can read the first 2 minutes of a cold real sweep without wondering if it hung:
  factual startup activity present until the first inner-CV run completes; labels self-explanatory.
- Leaves + rate/ETA move smoothly (unit tests over scripted event traces pin the damping); after a
  mature trace, five seconds without a completion visibly advances the mode-specific last-run age by
  five seconds while the numeric rate/ETA move monotonically (regression test).
- Occupied-but-silent subprocesses are never described as proven progress; typed successful step
  completions/cache hits are the only positive activity signal.
- A failed run that successfully persists its failure record advances neither the eligible run
  counter nor rate/ETA (regression test).
- Flat and nested output, width clamping, repaint cadence, and terminal cleanup remain correct.
- RUNBOOK board description/example updated and its peer commit pinned in the Log.

## Plan

- [ ] Implement the approved typed-telemetry/refined-render design through a durable TDD plan.
- [ ] Verify labels, startup truthfulness, time-based occupancy, ETA readiness, stall decay/recovery,
  terminal behavior, and the downstream RUNBOOK update.

## Log

### 2026-07-16
- Filed from the operator's mid-run feedback on the first real-sweep board session (pins set,
  full 7,200-fold grid). Companion issues from the same session: metis#47 (flash — FIXED,
  DEC 2026 sync output), metis#48 (default BLAS pins — the 3h-ETA root cause when unpinned),
  metis#43 (the phase-wave scheduler, pre-existing). The wild-ETA observation is partly a
  #43 symptom: depth-first scheduling would give early train completions → a stable rate
  much sooner.

### 2026-07-16 — paired #43/#49 design approved
- Operator chose typed step-completion telemetry over presentation-only occupancy wording because
  occupancy cannot distinguish useful work from a hung subprocess. Co-designed after mapping the
  current sink/rate/compositor flow; #43 merges first, then #49 builds against its schedule.

## Revisions

### 2026-07-16 — fresh-eyes spec review
- Split flat `CV runs` from nested `inner-CV runs`, made out-of-order callback reduction explicit,
  and replaced an unprovable seconds-level rounded-rate promise with a tick-driven last-run-age
  freshness signal plus a measurable scripted-trace criterion. Separated typed step activity from
  actual concrete-run completion so batch-order sampler callbacks cannot falsify the rate window;
  required execution success independently of successful failure-record persistence.
