---
id: 000058
status: codecomplete
deps: []
github_issue:
created: 2026-07-18
updated: 2026-07-18
estimate_hours: 1.2
started: 2026-07-18T12:06:53-07:00
actual_hours: 0.50
---

# sample grammar outMinN: subsample both CV levels

## Problem

Arena2 (Playground S6E7, 690k train rows ≈ 100× titanic) makes iteration cost real: even the
7-config starter grid is `outer × configs × inner_k` leaf fits on ~620k-row analysis frames.
`--sample m` subsamples only the OUTER level; the inner per-config CV always runs all
`inner_k` (or k) folds. The alternative — editing `inner_k` in the shape — changes the inner
partition itself (a 2-way split shares no fold boundaries with a 5-way), so it re-keys every
leaf and throws iteration spend away. Demand #2 from the arena2 project (operator-proposed
design, 2026-07-18 session): a CLI dial over BOTH levels that keeps the shape's declared
estimand intact and lets iteration runs escalate into decision runs via the cache.

## Spec

**Grammar (breaking — one grammar, no bare-integer form):**

- `--sample out<M>` — run M of the k outer folds, full inner CV (today's `--sample M`).
- `--sample in<N>` — all k outer folds, N of the inner folds per config (a cheaper decision run).
- `--sample out<M>in<N>` — both.
- `--fast` ≡ `--sample out1` (unchanged semantics, re-expressed).

**Semantics (extends the metis#42 m-of-k principle to the inner level):**

- The shape declares the estimand (`k`, `inner_k`); the flag only buys precision. Partitions
  are ALWAYS materialized at the declared fold counts; the flag runs a **deterministic prefix
  subset** (folds 0..N-1, same mechanism as outer sampling) — so subset runs share leaf
  content-addresses with full runs.
- **Cache continuity is the point:** an `in5` run after an `in2` run cache-HITs folds 0–1 and
  runs 2–4; `writeSweepLedger` already dedups by point-address (fold coordinate is in the
  address), so the ledger CONVERGES to full coverage — no double-counting, no ragged
  comparisons after any completed run. Record this reasoning in the atlas: select-side
  fairness needs no new guard (residual raggedness exists only after an interrupted run, and
  a completed re-run heals it).
- Honesty semantics: inner subsampling degrades selection quality (noisier per-config mean),
  never the outer estimate's honesty (each family's inner-winner is still refit and scored on
  a fully held-out assessment fold). An `outMinN` run measures a slightly different
  *procedure* than the full run (select-by-N-fold) — indicative, same caveat as `--sample 3`'s
  2-df SE.

**Validation (the existing "misuse fails loudly" family):**

- `out ≤ k`; `in ≤ inner_k` (or `≤ k` when `inner_k` is unset — inner then runs k-way).
- No combining with `--fast`; meaningless on flat single-config runs — refuse, as today.
- Malformed strings (`out0`, `in`, `3`, `outin2`) → loud usage error naming the grammar.

**Caller sweep (breaking-change discipline — every consumer moves, ARCH-PURPOSE):**

- metis: flag parsing + `runNestedCV`/sampler plumbing; `innerk_e2e_test.go`; any `--sample`
  in cmd help text, `atlas/experiment.md`, README/docs.
- kbench: `competition/titanic/pipelines/RUNBOOK-sweep.md` (`--sample 3` → `--sample out3`);
  the s6e7 runbook (kbench#12) uses the new grammar from day one.

**Relation to metis#54 (racing/successive-halving):** this is the MANUAL dial; #54 is the
adaptive version over the same budget. Ship this first — it de-risks whether #54 is needed at
all (measure before rebuild).

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.1 impl=0.1
item: smaller-go-module   design=0.2 impl=0.2
item: smaller-go-module   design=0.1 impl=0.15
item: atlas-docs          design=0.0 impl=0.1
item: milestone-review    design=0.0 impl=0.2
design-buffer: 0.15
total: 1.21
```

(Items in plan order: parseSample+flag · splitK/runK plumb+validation+legacy-test rework ·
new e2e · docs/caller sweep · close review. Design buffer 0.15: thorough plan doc, 2× reviewed.
v3.1 impl values = 40% of v2 ranges, AI-paired ship wall-clock.)

## Done when

- `metis run --sample out1in2 <shape>` runs 1 outer × 2-inner-fold subsets per config; a
  subsequent `--sample out1` (full inner) run cache-HITs the 2 already-measured folds per
  config and the ledger shows exactly inner_k rows per (config, outer-fold) — verified by an
  e2e asserting the HIT and the converged row count.
- Bare `--sample 3` is a loud usage error; `--fast` still ≡ one outer fold.
- All callers swept (metis e2e/docs + kbench RUNBOOK); validation refusals covered by tests.

## Plan

Durable plan: `workshop/plans/000058-sample-grammar-outminn-plan.md` (fresh-eyes reviewed 2× ✅ 2026-07-18).

- [x] parseSample (pure, TDD) — grammar + overflow-loud
- [x] flag/runOpts retype + validation (`< 1` guards stay: runOpts seam) + splitK/runK split + banners (`--fast` → `1/k`) + legacy nestedcv test rework — one green commit
- [x] new-surface e2e: inner refusals, out1in2 subset ledger, cache-escalation convergence
- [x] caller sweep (metis docs/atlas + kbench runbooks/plan) + shadow-sweep grep (workshop/ exempt)
- [x] pr → merge → close (single boundary)

## Log

### 2026-07-18
- 2026-07-18: closed — unit+e2e green (go test ./... all ok): parseSample table incl overflow; refusals (flat/range/fast/negative via runOpts seam); out1in2 prefix-subset ledger; cache-escalation convergence (run B spawned only fold-2 trains + outer-refit HIT, ledger exactly {0,1,2} once each per config); CLI-verified bare --sample 3 refuses with grammar error; callers swept both repos, shadow-sweep grep clean (workshop/ exempt); review verdict: FIX-THEN-SHIP

- Filed from the arena2/kbench#12 planning session (brain repo conversation): operator
  proposed the grammar; the cache-continuity and ledger-dedupe analysis above was verified
  against `cmd/metis/ledger.go` (`writeSweepLedger` idempotent, dedups by point-address;
  fold coordinate in the row + address) before filing. Demand #2 on the arena2 demand list
  (demand #1 = the balanced-accuracy metric knob, filed separately).

### 2026-07-18 (implementation)

- Tasks 1-5 done on-branch (fork agent): 7f35c7a parseSample · 3e63fce grammar/validation/
  splitK-runK (one green commit; legacy nestedcv tests reworked, `< 1` guards kept for the
  runOpts seam, `runInnerK := splitFolds` flat fix, `--fast` renders 1/k) · c4b2bc2 e2e
  (refusals + out1in2 subset ledger + cache-escalation: run B spawned 2 trains + 1 features,
  outer-refit HIT, ledger converged to {0,1,2} once each per config) · 8e46d80 docs sweep.
  kbench docs commit 30a0b06 (RUNBOOK/shape/atlas/plan text → out3, +out1in2 iteration line).
- Full `go test ./...` green; CLI-verified: bare `--sample 3` → grammar error, out1in2 parses.
- Plan deltas (fix-forward): bare `go build ./cmd/metis` fails (pkg-dir collision) → `-o
  bin/metis`; two residual `--sample 3` mentions document the retirement itself (accepted);
  one stale test error-string fixed in 8e46d80. Lessons persisted to workshop/lessons.md.
- FIX-THEN-SHIP fixes bundled into the close commit: atlas select-fairness sentence (the
  Important finding — Spec-mandated reasoning now recorded), seededTotals param renamed
  `k` → `foldsPerConfig`, new `TestSample_FastBannerRendersSampled` pins the `1/k` --fast
  banner. Accepted-not-fixed (minor, logged): cfKey test-helper duplication (test-tier
  nicety), no in<N>-only e2e (code path ≡ covered out1in2 minus runFolds override; low risk).
