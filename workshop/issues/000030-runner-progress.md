---
id: 000030
status: codecomplete
deps: []
github_issue:
created: 2026-07-13
updated: 2026-07-15
estimate_hours: 1.63
started: 2026-07-15T16:07:52-07:00
actual_hours: 1.51
---

# runner progress reporting — SizeHint + progress callback (k/n + live outer-cv)

## Problem

A sweep runs blind. `titanic-sweep.md` is **495** per-fold runs (`driver: single`, 99 configs × 5
folds) and **2,475** for the honest `driver: cv` (× 5 outer) — with no live signal of how far along it
is or what it's finding. The operator wants **`k / n`** (k = points completed, n = total) **plus the
running estimate** (best-so-far / outer-cv). For grid, n is exact; for adaptive samplers (future: bayes,
racing) n may be a budget or genuinely unknown — so n must be allowed to be `?`, with k + the incumbent
still reported.

## Spec

The `pkg/sampler` **ask/tell `Run` loop already sees everything needed** — it fires a `Tell` per
completed point (that's k) and holds the accumulator `S` (which carries the incumbent / per-outer-fold
gather = the live outer-cv). Two additions, no new runner (see the metis-v2 runner design in
`workshop/pensive/` / the kbench#8 discussion — grid and adaptive share one `Run`):

1. **`SizeHint` on the `Sampler` interface** — `SizeHint(s S) (total int, kind SizeKind)` where
   `SizeKind ∈ {exact, budget, unknown}`. The static/grid sampler returns `(∏ grid, exact)`; a
   fixed-eval sampler returns `(maxEvals, budget)`; an open-ended one returns `(_, unknown)`. This is
   the ONLY per-sampler bit — the varying "n" pushed into the interface, not a runner branch.
2. **A `progress` callback injected into `Run`** (alongside the existing `runPoint` closure, ARCH-PURE)
   — `progress(ev ProgressEvent)` called on each `Tell` with `{level, k, total, kind, incumbent}`. Nesting
   composes: the driver level reports outer-fold progress + the live outer-cv from its accumulator; the
   sweeper level reports inner-config progress. `Run` stays pure over the injected callback.
3. **`cmd/metis` renders it** — a single aggregated line (NOT a 2,475-line firehose — respect the
   single-threaded-attention budget): e.g. `outer 2/5 · inner 47/99 · est 0.812 ± 0.02` for nested,
   `47/99 · best 0.834` for a flat sweep, `47/? · best 0.834` for an unknown-budget sampler.

Injected `progress` defaults to a no-op (backward-compatible; pure `Run` tests pass `nil`).

## Done when

- `SizeHint` on the grid sampler returns the exact config×fold product (unit-tested pure).
- `Run` invokes the injected `progress` exactly once per `Tell`, with a monotonically increasing k and
  the correct total/kind (unit-tested with a recording callback; a `nil` callback is a no-op).
- `metis run` prints a live `k/n` + running best for a grid sweep, and `outer j/k · est mean±SE` for a
  `driver: cv` run — verified on a real (or fixture) sweep, not just asserted.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.05 impl=0.25
item: smaller-go-module   design=0.10 impl=0.35
item: smaller-go-module   design=0.10 impl=0.40
item: smaller-go-module   design=0.02 impl=0.15
item: atlas-docs          design=0.02 impl=0.10
design-buffer: 0.30
total: 1.63
```

Row 1 = `SizeHint` on the Sampler interface + 6 impls (4 prod + 2 test fakes incl. the
`countSampler` n-refactor) + table test. Row 2 = `Run`'s completion-fired `ProgressEvent[P,O]`
(mutex-serialized wrapper) + 19 call-site updates + Seq/Par progress tests. Row 3 =
`cmd/metis/progress.go` sink (per-pass hooks, seeded totals, throttled renderer) + fake-clock
tests + the 4 wiring sites. Row 4 = fixture-sweep output pins (nested + flat). `atlas-docs` =
atlas/RUNBOOK + issue Revisions + the real smoke-sweep evidence run. Calibration doc [stale]
(#127) — provisional.

## Plan

Durable plan: `workshop/plans/000030-runner-progress-plan.md` (fresh-eyes reviewed; 5 findings
folded: Task-4 e2e premises corrected to fixture pins + real evidence at close, countSampler
refactor stated, 19 call sites grep-verified, totals seeded at wiring time, per-pass hooks carry
outer-fold identity for #38). Single-pass close, no milestones.

- [x] Task 1: `SizeKind` + `SizeHint` on `Sampler` + 6 impls (TDD)
- [x] Task 2: `Run` fires `ProgressEvent[P,O]` at point completion; 19 call sites; Seq/Par tests
- [x] Task 3: `cmd/metis` sink (per-pass hooks, seeded totals, 1s-throttled line) + wiring
- [x] Task 4: fixture-sweep output pins (nested + flat)
- [x] Task 5: docs (atlas/RUNBOOK/Revisions) + real smoke-sweep evidence + close

## Log

### 2026-07-13
- Filed from the kbench#8 sweep-scale discussion (operator, metis-v2). The runner is already an ask/tell
  fold with one `Run` loop for grid + adaptive; progress is the `SizeHint` (n) + a per-`Tell` callback
  (k + incumbent) hung off that loop — no per-sampler runner. Sibling: metis#31 (parallel batch exec) —
  same loop, the other injected seam. Near-term / high-value (you feel the blindness at 2,475 folds).

### 2026-07-14
- metis#38 filed (operator, during the #35 honest-beat run): a TUI/curses progress board over THIS
  issue's event stream — #30 stays the instrumentation + plain-line renderer (and the non-TTY
  degradation target), #38 owns the TTY presentation. This issue's scope is unchanged.

### 2026-07-15
- 2026-07-15: closed — TDD red-green per task; full module go test green (-race on sampler + progress tests); go vet clean. Real-run evidence: metis run --parallel 8 titanic-sweep-smoke.md (real uv/Python leaves, BLAS pinned, stdout redirected to file) printed 6 live progress lines — mid-run folds 1/36→21/36 while outer 0/3, est evolving 0.7980 → 0.8064±0.0084 → final outer 3/3 · configs 12/12 · folds 36/36 · est 0.8103±0.0062; zero escape codes in the captured file. Fixture pins in nested+flat sweep tests assert the final-line counts.; review verdict: SHIP
- Claimed + start-plan (T2 order, operator-set). Durable plan authored + fresh-eyes reviewed
  (verdict: issues found, all folded — see Plan section). Lessons persisted to workshop/lessons.md.
- **Design decision (spec revision, full rationale in the plan):** the callback fires at POINT
  COMPLETION, not per Tell — #31 (landed after this issue was filed) made exec batch-scoped and
  every production sampler is one-batch static, so per-Tell events would all land at batch end
  (dead as live progress). Count contract unchanged: exactly one event per point. Event =
  `ProgressEvent[P,O]{K, Total, Kind, Point, Out}` (typed per level — no `any`); incumbents are
  derived by the cmd/metis sink from outputs its closures already handle (S is opaque + un-Told
  at completion time). SizeHint stays on the interface per spec (ARCH-DRY: the sampler owns n);
  totals additionally SEEDED at wiring from direct SizeHint calls (stream-learned totals arrive
  with a level's first completion — too late). #38's outer-fold identity rides per-pass closure
  binding (`forPass(i)`), NOT a payload field (ARCH-PURE: pkg/sampler stays coordinate-free).

## Revisions

### 2026-07-15 (at change-code, implemented same-day)
1. **Per-`Tell` → per-COMPLETION firing.** The Spec's premise ("Run fires a Tell per completed
   point") predates metis#31: exec is now batch-scoped (Tell happens after the whole batch
   returns) and all four production samplers are one-batch static — per-Tell events would all
   land at batch end, dead as live progress. `Run` instead wraps `runPoint` and fires
   mutex-serialized events at point completion. Count contract unchanged (exactly one event per
   point); the event carries the completed `(Point, Out)` pair, not accumulator state.
2. **Flat-format example corrected.** The Spec's `47/99 · best 0.834` flat example is the
   pre-#32 world; since metis#32 the flat path runs iff exactly ONE config, so the flat line is
   `folds k/n · score <running mean>` (nothing to be "best" of). The Spec's nested example is
   unchanged in spirit: `outer j/k · configs a/b · folds x/y · est mean±SE`.

## Log (continued)

### 2026-07-15 (implementation)
- Tasks 1–4 done TDD (each red→green): `SizeHint` + 6 impls · `ProgressEvent[P,O]` completion-fired
  in `Run` (19 call sites, monotone-k ParExec test) · `cmd/metis/progress.go` sink (per-pass
  `forPass(i)` hooks = the #38 identity seam; totals seeded at wiring; 1s throttle on injected
  clock; error-gated driver events) · fixture pins in the nested + flat sweep tests. Full module
  green with `-race` on the touched packages; `go vet` clean.
- **Real-run evidence (done-when 3):** `metis run --parallel 8 titanic-sweep-smoke.md` (real
  uv/Python leaves, BLAS pinned, stdout redirected): 6 progress lines, live mid-run updates
  (`folds 1/36 → 21/36` while outer 0/3), est evolving `0.7980 → 0.8064 ± 0.0084 → 0.8103 ±
  0.0062`, final `outer 3/3 · configs 12/12 · folds 36/36`; zero escape codes in the captured file.
- Close review (sidecar: workshop/plans/000030-runner-progress-close-review.md): **SHIP**, no
  Critical/Important. Minor findings addressed in the close commit anyway: vestigial `direction`
  param dropped (dead since the flat-format correction; #38 reintroduces it for incumbents), the
  throttle-test comment corrected (emits are event 1 + event 6, not 5+10), and the error-gated
  driverEvent path pinned (failing nested sweep must not display sentinel-zero estimates —
  extended TestShapeSweep_FailingFoldIsFatal). Left as noted: the test-only IndexByte nit; the
  SizeHint interface break is repo-internal (no out-of-repo Sampler implementers exist).
