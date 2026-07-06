# Boundary Review — metis#7 (milestone M1)

| field | value |
|-------|-------|
| issue | 7 — Sweep runner + grid sampler (propose_next / should_stop abstraction) |
| repo | metis |
| issue file | workshop/issues/000007-sweep-runner-grid-sampler.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | cc04241c08f9e55782e2a41cbd72716c9fc6caee^..HEAD |
| command | sdlc milestone-close --issue 7 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-05T17:49:38-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. The M1 delta is `pkg/sweep/` (commit `638d232`); the rest of the window is already-merged, already-reviewed deps (#9/#3/#2/#6, each with a `*-close-review.md` sidecar). Build clean, `go vet` clean, full suite green, tree clean, `pkg/sweep` correctly not-yet-wired (wiring is M2). Here is the boundary review.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** The metis#7 M1 boundary delivers exactly what its Plan scopes — a pure `pkg/sweep` ask/tell sampler (`Sampler`, `Grid`, `StopPredicate`, `MaxPoints`/`TargetReached`/`AnyStop`) with no IO and no premature wiring — and it's clean: correct enumeration/stop semantics, both objective directions handled, good edge coverage, all M1 Done-when test items present, and the whole tree builds + tests green. I found no Critical or Important issues in the M1 delta; the findings below are Minor + forward-looking notes for M2. A scoping note: the review window (`cc04241^..HEAD`) pulls in the already-merged deps (cas/record/cache/shape); those crossed their own boundary reviews (sidecars in `workshop/history/`), so I treated them as reviewed context and focused fresh-eyes energy on `pkg/sweep`, spot-confirming only that they don't break the boundary (they don't — suite is green).

**1. Strengths**
- **ARCH-PURE, exemplary.** `pkg/sweep` is fully pure — deterministic enumeration, stop predicates pure over tell-history, no clock/net/fs. Tests run with zero IO and zero mocks (`sweep_test.go`). This is the textbook PURE-core entity.
- **Correct scope discipline (ARCH-PURPOSE).** M1 ships only the sampler; the driver/`metis run` flip, manifest, and detect-and-abort are correctly deferred to M2. Nothing here is half-wired — `pkg/sweep` has no consumers yet, which is the *intended* M1 state, not an omission.
- **Stop semantics are right.** `MaxPoints(3)` over 10 points → exactly 3 (`sweep.go:83`/`sweep_test.go:60`); `TargetReached` stops on the Ask *after* the crossing Tell, in both directions, and a failed/missing-metric point never trips it (`sweep.go:96`, tested at `sweep_test.go:99`). No off-by-one.
- **ARCH-DRY.** Reuses `shape.Point`; `AnyStop` composes predicates instead of duplicating stop logic. No copy-paste.
- **Clean, stable ask/tell seam** (`sweep.go:34`) matching the ecosystem standard (Optuna/Nevergrad) — `Ask() (Point, bool)` + `Tell(Point, Result)`, stop folded into the sampler so the M2 driver loop stays trivial.

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**
- `pkg/sweep/sweep.go:56` — the "seeded slot" is a **doc comment only**; `NewGrid`/`Sampler` carry no seed parameter. Defensible (grid is deterministic by enumeration, an ignored seed param would be dead API/YAGNI), but it diverges from the Plan/Spec wording "the interface carries a `sampler_seed` slot." See plan-revision note below.
- `sweep.go:96` (`TargetReached`) — any non-`"minimize"` direction (including `""` and a typo) silently takes the maximize branch. Fine given `ValidateShape` constrains the objective upstream, but at the M2 wiring site an unrecognized direction would mask a config typo rather than error. Consider asserting `direction ∈ {maximize,minimize}` when constructing the predicate from `sweep.objective`.
- `TargetReached` re-scans the full history each `Ask` (O(N²) across a sweep). Negligible at grid sizes; noting only, no change warranted.

**5. Test coverage notes**
- Coverage pins the real logic (enumeration order, both stop directions, missing-metric skip, compose, empty grid, exhaustion-stays-done) — the bug classes this diff could ship are caught.
- The Plan named a "Tell-is-no-op for grid" unit test; it's only *implicitly* covered (the enumeration test Tells while asserting order). Cheap to make explicit — e.g. Tell out-of-order results and assert the enumeration sequence is unchanged.
- Optional edges, not blocking: a `TargetReached` first-point crossing (should yield exactly 1 run) and `MaxPoints(0)` (zero-run budget) aren't asserted.

**6. Architectural notes for upcoming work (M2)**
- **`Ask() bool` can't distinguish exhaustion from early-stop.** #8's manifest and the "stopped at k/N — code changed / budget hit" UX will likely want a stop-*reason*. `pkg/sweep` has zero consumers today, so decide this at M2 *before* the interface calcifies — extend `Ask` or add a `StopReason()` — rather than after the driver + manifest depend on it.
- **Point aliasing at the Tell boundary.** `Ask` returns `g.points[i]` by value, so its inner `With`/`FreeParams` maps/slices are shared with the grid's slice and with `history` after `Tell`. `shape.Expand` already deep-clones per point (no sibling aliasing), but the M2 driver overlays resolved `with` per point — ensure it doesn't mutate a point *after* `Tell` if the manifest reads `TellRecord.Point`, or the recorded snapshot won't be faithful.
- **Failed-point contract.** `TargetReached` relies on a failed point carrying no objective metric. Confirm the M2 driver feeds `Result{Status:"failed", Metrics:nil}` (not a zero-valued metric) so a failed run can't accidentally trip/relax a target.

**7. Plan revision recommendations**
- `workshop/plans/000007-sweep-runner-plan.md` (M1 bullet) + issue `## Design`/`## Spec` say the sampler "is seeded" / "the interface carries a `sampler_seed` slot." The delivered code provides the reproducibility **property** (deterministic enumeration) but no seed API. Add a `## Revisions` entry: *"M1: the `sampler_seed` slot is documented but not a constructor/interface parameter — grid is deterministic without one, so the seed lands at the adaptive-sampler constructor (post-v1). M1 delivers the reproducibility property, not an ignored parameter."* This keeps the plan from over-claiming an API surface the code (correctly) doesn't ship.
