# Boundary Review — 000055-board-color-separation-and-summary-after-footer#55 (whole-issue close)

| field | value |
|-------|-------|
| issue | 55 — board color separation and summary after footer |
| repo | 000055-board-color-separation-and-summary-after-footer |
| issue file | workshop/issues/000055-board-color-separation-and-summary-after-footer.md |
| boundary | whole-issue close |
| milestone | — |
| window | 553f5856ea98cefa2a5026f569c962446b1eef4c..HEAD |
| command | sdlc close --issue 55 |
| reviewer | claude |
| timestamp | 2026-07-18T09:43:01-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Static review complete (note: I could not execute `go test` — Bash is broken in this session at the harness level, EPERM creating its session-env dir even with the sandbox disabled; the `/sandbox` command can manage restrictions, but the failure here is outside my commands). Findings verified by reading the code, tests, spec, and wiring.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The implementation is clean and well-pinned: the paint/content split is genuinely preserved (renderBoard untouched and still asserted escape-free at board_test.go:55), the epilogue channel is a small, correctly-locked buffer flushed exactly once after the final frame + cursor restore, and the erase math counts the new separator row (pinned by an explicit ghost-line test). What blocks a clean SHIP is a contract drift the tests actively pin: the issue's Spec and Done-when both promise "NO_COLOR → byte-identical to pre-#55," but the separator rule paints unconditionally whenever width > 0 — and production width is always ≥ 80 (`boardWidth()` falls back to 80, main.go:85) — so a NO_COLOR run gets a new plain `─` row. `TestBoardWriter_NoColorHasNoSGR` asserts this deviation as intended behavior. Either the code needs a one-line gate or the Spec/Done-when needs a revision; as written, a Done-when bullet is unmet at the close boundary.

### 1. Strengths

- **Paint/content split honored exactly as specced** (`board.go:360-387`): SGR wraps after clamping in `redraw`, renderBoard returns plain lines, and the existing renderer byte-clean test needed zero changes — the #38 architecture absorbed this feature with no churn in the pure half.
- **Erase-count bookkeeping done right and pinned**: the separator increments `painted` (`board.go:384-386`), and `TestBoardWriter_EpilogueAfterFinalFrame` (board_test.go:753) explicitly asserts the absence of the stale 2-line erase — exactly the ghost-line regression this diff could ship.
- **Epilogue ordering asserted on the raw writer end-to-end** (board_test.go:398-407): restore → estimate → summary → output *ends* on the paste-ready `# cohorts` hint — the Done-when's ordering bullet is tested at the right layer, not on a mock.
- **`summaryWriter` is a clean single seam** (sweep.go:633-640): a type-assertion router, passthrough in plain mode, used at all three summary/estimate sites — good ARCH-DRY shape.
- **Test determinism handled thoughtfully**: color injected at construction with the env read at the one production wiring point (run.go:151), and the e2e pins `NO_COLOR=""` via `t.Setenv` so harness env can't flake the SGR assertions.

### 2. Critical findings

1. **NO_COLOR output is not byte-identical to pre-#55, contradicting Spec and Done-when** — `board.go:361-367` paints the separator whenever `len(frame) > 0 && width > 0`, independent of `b.color`; production width is never 0 (`$COLUMNS | 80`, main.go:81-90). The Spec's gating bullet ("With NO_COLOR set, output is byte-identical to today") and the Done-when ("NO_COLOR → byte-identical to pre-#55") are both violated, and `TestBoardWriter_NoColorHasNoSGR` (board_test.go:729) pins the deviation ("the separator rule paints (plain) even with color off"). Two coherent fixes, pick one:
   - *Code route:* add `&& b.color` to the two separator guards in `redraw` (`board.go:361` and `board.go:384`) — restores literal byte-identity under NO_COLOR.
   - *Doc route (my recommendation):* the separator is structural separation, not color, and NO_COLOR by convention governs only color — keep the behavior and revise Spec + Done-when (see §7). The plan-judge amendment (commit effc7c2) already conceded "the separator is one NEW row" for the color-on clause but left the NO_COLOR clause contradicting.
   Either way, the issue file must stop claiming what the code doesn't do before close.

### 3. Important findings

1. **Done-when and Plan cite tests that do not exist** (traceability). The Done-when requires "pyte-replayed EXISTING lines' text unchanged" and the Plan checks off "tests: … NO_COLOR byte-identity … pyte content unchanged"; the Log claims "pyte content tests green as-is." There is no pyte anywhere in this repo (every grep hit is `pytest`; no Go or Python board-replay test exists), and no NO_COLOR byte-identity test exists (`TestBoardWriter_NoColorHasNoSGR` asserts SGR-absence, a weaker property — necessarily, given Critical #1). The substance is partially covered by the renderer's zero-escape pin (board_test.go:55), but the plan asserts coverage it doesn't have. Fix: revise the Done-when/Log to cite the tests that actually exist (see §7), or add the missing assertions.
2. **`reportWinner` is not routed through `summaryWriter` — a flat board run's RESULT stays above the footer** (ARCH-PURPOSE). Board mode covers flat shape sweeps too (renderBoard has an explicit flat path), and for a flat sweep the winner leaderboard (sweep.go:842-864, writing to `ss.out`) *is* the run result — yet only `printRunSummary` lands in the epilogue (sweep.go:376-377), so the final terminal reads winner → separator → footer → summary, splitting the result across the band. The Spec names only `reportEstimate` + `printRunSummary`, so this may be deliberate scope (the motivating run was nested k10) — but it's the same burial class the issue exists to fix, and the fix is one line (`summaryWriter(ss.out)` inside `reportWinner`, matching `reportEstimate`). If deliberately deferred, record it in a Revisions entry rather than leaving it implicit.

### 4. Minor findings

- Issue file has a duplicated empty `## Done when` section (issue lines 50-52) — stale scaffold, delete before close.
- ARCH-DRY nit: `sp.width` is passed twice at the emit site (inside `boardEnv` and again as `paint`'s second arg, progress.go:466-467); consider `paint(lines, env.width)` derived once, or storing width via the env — one source per call.
- Routing-seam inconsistency: `reportEstimate` wraps internally (`out := summaryWriter(ss.out)`, sweep.go:626) while `printRunSummary` is wrapped at its call sites — pick one convention so future result-printers copy the right pattern.
- The e2e's tail assertion `HasSuffix(…, "# cohorts")` couples the test to `printRunSummary`'s cohort-present wording; the cohort-absent branch ends differently — fine today, brittle if the fixture's capture behavior changes.

### 5. Test coverage notes

- **I could not run the suite** (Bash unavailable in this review session — harness-level EPERM, not a sandbox policy I could override). The Log claims full `-race` green; the main agent should re-run `go test -race ./...` before close since I couldn't independently confirm.
- Covered well: epilogue ordering on the raw writer (e2e + unit), erase-count-with-separator (explicit anti-ghost-line pin), SGR-absence with color off, plain/`--no-tui` zero-escape invariant (board_test.go:441, satisfying the "redirected → zero ESC" bullet via the existing test, as the Done-when allows).
- Gap: the production NO_COLOR-*set* path is never exercised end-to-end — the e2e only pins `NO_COLOR=""` (color on). A sibling e2e with `t.Setenv("NO_COLOR", "1")` asserting no styling SGR in the full run stream would close the loop on the one env read at run.go:151.
- Gap (inherits Critical #1's resolution): whichever way #1 resolves, add the matching test — byte-identity against a color-gated separator, or a revised assertion documenting separator-always.

### 6. Architectural notes

- **ARCH-DRY: pass** — `summaryWriter` consolidates the routing decision; no copy-paste blocks. Only the width double-pass nit above.
- **ARCH-PURE: pass** — the pure renderer stays pure; styling lives in the designated ANSI layer and is tested against an injected `strings.Builder` (no real IO, no mocks re-asserting implementation). If the classification switch in `redraw` grows more cases, extract a pure `stylizeFrame(lines, width, color) []string` so it's unit-testable without the compositor — not needed at current size.
- **ARCH-PURPOSE: flagged** — nested-path purpose fully delivered; the flat-path winner report (Important #2) is the one consumer of "result ends the terminal" that doesn't yet route through the seam. Core-concepts table: the plan has none (checklist-only), so no table cross-check applies.
- Docs gates: **atlas updated** (atlas/experiment.md:215-221, accurate to the implementation including the NO_COLOR-at-wiring detail). **README gate N/A** — this repo has no README.md; atlas is its user-facing doc surface and it covers the new NO_COLOR knob.

### 7. Plan revision recommendations

Add a `## Revisions` section to the issue:

1. *(resolves Critical #1, if the doc route is chosen)* "Revised: the separator rule is structural (band boundary), painted in board mode regardless of NO_COLOR — no-color.org governs color, not layout. Done-when 'NO_COLOR → byte-identical to pre-#55' is replaced by: 'NO_COLOR → zero SGR styling bytes; output differs from pre-#55 only by the one plain separator row' (pinned by TestBoardWriter_NoColorHasNoSGR)." Also amend the Spec gating bullet ("byte-identical to today") to match. If the code route is chosen instead, no revision — the original wording becomes true.
2. *(resolves Important #1)* Replace the "pyte-replayed … unchanged" clause and the plan item "NO_COLOR byte-identity; … pyte content unchanged" with the tests that exist: renderBoard's zero-escape/content pins (unchanged by this diff) + the NoColorHasNoSGR and EpilogueAfterFinalFrame tests. Strike the Log's "pyte content tests green as-is" claim — no pyte tests exist in this repo.
3. *(resolves Important #2)* Either add "reportWinner routes through summaryWriter (flat board runs end on the winner + summary)" as a delivered item, or an explicit deferral entry stating the flat-path winner remains above the board and why.
4. Delete the duplicate empty `## Done when` section.

---

## Re-review — 2026-07-18T09:48:47-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 55 — board color separation and summary after footer |
| repo | 000055-board-color-separation-and-summary-after-footer |
| issue file | workshop/issues/000055-board-color-separation-and-summary-after-footer.md |
| boundary | whole-issue close |
| milestone | — |
| window | 553f5856ea98cefa2a5026f569c962446b1eef4c..HEAD |
| command | sdlc close --issue 55 |
| reviewer | claude |
| timestamp | 2026-07-18T09:48:47-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I have everything I need for the review. One process note up front: the Bash tool is broken in this session at the harness level (EPERM creating its session-env dir, even with the sandbox override disabled — the same condition recorded in this repo's #44 and prior #55 close reviews), so I could not run `go test -race ./...` myself. Everything below comes from close reading of the full diff, the tree at HEAD, the issue file, and the surrounding code.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The code side of this boundary is genuinely done and well-pinned: the paint/content split holds (renderBoard returns plain lines; all SGR lives in `redraw`), the epilogue channel is a small correctly-locked buffer flushed exactly once after the final frame + cursor restore, the separator is counted in the erase math with an explicit anti-ghost-line test, and all three result printers — `reportEstimate`, `printRunSummary`, and (per the prior review's fold) `reportWinner` — route through the single `summaryWriter` seam. What keeps this at FIX-THEN-SHIP is that the prior review's **documentation fold is incomplete while the Log claims it is complete**: the issue's Done-when still requires "NO_COLOR → byte-identical to pre-#55" and cites pyte-replay tests, both of which the fold commit (a8db0d8, "NO_COLOR contract doc'd to code, phantom citations removed") claims were fixed — but the Spec's *gating* bullet is the only place actually revised. As written, a Done-when bullet is false at the close boundary and the issue file contradicts itself. The fix is doc-only.

### 1. Strengths

- **Paint/content split preserved exactly as designed** (`cmd/metis/board.go:355-387`): styling is applied post-clamp in `redraw` only; `renderBoard` is untouched and stays byte-plain, so the pure half absorbed the feature with zero churn — validated ARCH-PURE ground.
- **Erase-count bookkeeping is right and pinned**: the separator increments `painted` (`board.go:384-386`), and `TestBoardWriter_EpilogueAfterFinalFrame` (`board_test.go:753`) explicitly asserts the absence of a stale 2-line erase — precisely the ghost-line regression this diff could ship.
- **Epilogue ordering asserted end-to-end on the raw writer** (`board_test.go:398-408`): restore → estimate → summary, and the stream must *end* on the paste-ready `# cohorts` hint — the Done-when's ordering bullet tested at the right layer, plus post-close epilogue writes degrade safely to passthrough (`board.go:229-237`).
- **`summaryWriter` is one clean seam** (`sweep.go:635-640`), now used by all three result sites including the flat-path `reportWinner` (`sweep.go:843`) — the prior review's ARCH-PURPOSE flag on the flat path is delivered in code.
- **Determinism handled properly**: color injected at construction, env read once at the production wiring point (`run.go:151`), and the e2e pins `NO_COLOR=""` via `t.Setenv` (`board_test.go:381`) so harness env can't flake SGR assertions.

### 2. Critical findings

1. **The declared "doc route" fix for NO_COLOR was only half-applied; the Done-when is still false and the Log claims otherwise** (requirements traceability / contract drift). The fold Log (issue lines 98-102) states "Spec/Done-when revised to match the code" and "phantom 'pyte' test citations removed" — but at HEAD:
   - Done-when (issue `workshop/issues/000055-...md:48`) still requires "NO_COLOR → byte-identical to pre-#55". The code paints the plain separator under NO_COLOR whenever width > 0 (`board.go:361-367`), production width is never 0 (`boardWidth()` falls back to 80), and `TestBoardWriter_NoColorHasNoSGR` (`board_test.go:728-730`) pins the deviation as intended. The bullet is unmet as written, and it contradicts the *revised* Spec gating bullet (lines 31-35) three lines above it.
   - Phantom pyte citations remain at issue `:27` ("the pyte-replay + byte-clean tests stay untouched"), `:48` ("pyte-replayed EXISTING lines' text unchanged"), and Plan `:78` ("NO_COLOR byte-identity; … pyte content unchanged"). I grepped the whole repo: there is no pyte harness (only `pytest` hits and #46's *interactive* pyte replays in history) and no byte-identity test — exactly what the new `workshop/lessons.md:227-230` lesson warns against, in the same commit window.
   
   Fix (doc-only, finishing the route already chosen): rewrite Done-when `:48` to the revised contract ("NO_COLOR → zero SGR styling bytes; differs from pre-#55 only by the one plain separator row — pinned by TestBoardWriter_NoColorHasNoSGR"), replace the pyte citations at `:27`/`:48`/`:78` with the real pins (renderBoard's zero-escape test + NoColorHasNoSGR + EpilogueAfterFinalFrame), and amend or correct the fold Log entry so it no longer claims removals that didn't land.

### 3. Important findings

1. **Flat-path result routing has zero board-mode test coverage** (missing coverage for the kind of bug this diff ships). `reportWinner` → `summaryWriter` (`sweep.go:843`) is only ever exercised through plain writers (`shapesweep_test.go:279` and all other flat sweep tests use bare buffers), where `summaryWriter` is a passthrough — so the fold's Important-2 fix runs as a no-op in the entire suite. A regression re-burying the flat winner above the footer (e.g. someone reverting the one-line wrap, or `ss.out` gaining a non-board wrapper) ships silently. Cheap pin: a unit that hands `reportWinner`'s output path a real `*boardWriter` and asserts the leaderboard lands after cursor restore, or a flat-fixture variant of `TestRunExperiment_BoardMode`.
2. **I could not independently run the suite** (harness-level Bash failure, above). The Log claims "full -race green" and "Suite green" post-fold; the main agent should re-run `go test -race ./...` before recording the close verdict, since neither this review nor the prior one could confirm it.

### 4. Minor findings

- Duplicate empty `## Done when` section still present (issue `:53-55`) — flagged by the prior review's §7-4, not removed in the fold; delete before close.
- ARCH-DRY nit (carried): width passed twice at the emit site — inside `boardEnv` and again as `paint`'s second arg (`progress.go:466-467`); derive once.
- Seam-convention inconsistency (carried, now three-way): `reportEstimate`/`reportWinner` wrap internally while `printRunSummary` is wrapped at its two call sites (`sweep.go:377,473`) — pick one convention so the next result-printer copies the right pattern.
- The e2e tail assertion couples to `printRunSummary`'s cohort-present wording (`HasSuffix(…, "# cohorts")`, `board_test.go:406`) — brittle if the fixture's capture behavior changes.

### 5. Test coverage notes

- Well covered: epilogue ordering (unit + e2e on the raw writer), separator-in-erase-math (explicit anti-ghost-line assertion), SGR absence with color off, color-on banding presence in a full run, plain/`--no-tui` zero-escape invariant (`board_test.go:441-443`), post-close epilogue passthrough by construction.
- Gap (carried from the prior review, not addressed in the fold): the production NO_COLOR-*set* path is never exercised end-to-end — the e2e only pins `NO_COLOR=""` (color on). A sibling run with `t.Setenv("NO_COLOR", "1")` asserting no styling SGR in the full stream would close the loop on the one env read at `run.go:151`.
- Gap: flat-path board routing (Important #1 above).
- Unverified: `-race` green (Bash broken in this session; the Log's claim is unconfirmed by two independent reviews now).

### 6. Architectural notes

- **ARCH-DRY: pass** — `summaryWriter` is the single routing decision point, used at all five result-print sites; no duplicated blocks. Only the width double-pass and wrap-convention nits above.
- **ARCH-PURE: pass** — `renderBoard` remains pure and plain; SGR lives solely in the compositor's `redraw`; every new test runs against injected `strings.Builder`s and injected clocks, no IO, no mocks re-asserting implementation. If the `redraw` classification switch grows, extract a pure `stylizeFrame(lines, width, color)` — not warranted at current size.
- **ARCH-PURPOSE: pass on the code** — shadow-sweep of result printers: `reportEstimate`, `reportWinner`, and both `printRunSummary` sites all derive from the `summaryWriter` seam; the capture warning intentionally stays above the board (it's a warning, not the result). The purpose — "the terminal ends on the result" — is delivered for both nested and flat paths. The remaining under-delivery is documentary (Critical #1), not behavioral.
- Core-concepts table: the plan is checklist-only (no table), so no cross-check applies.
- Docs gates: **atlas updated and accurate** (`atlas/experiment.md:215-221` matches the implementation, including NO_COLOR-at-wiring and plain-output-unchanged). **README gate N/A** — no repo README exists (confirmed; consistent with this repo's precedent in the #7 close review).

### 7. Plan revision recommendations

The issue file needs a finishing pass to make the already-chosen doc route true (all resolve Critical #1 unless noted):

1. Done-when (issue `:47-49`): replace "NO_COLOR → byte-identical to pre-#55" with "NO_COLOR → zero SGR styling bytes; output differs from pre-#55 only by the one plain separator row (TestBoardWriter_NoColorHasNoSGR)", and replace "pyte-replayed EXISTING lines' text unchanged" with "renderBoard's byte-clean/content tests unchanged".
2. Spec (issue `:27`): "the pyte-replay + byte-clean tests stay untouched" → "the renderBoard byte-clean tests stay untouched".
3. Plan (issue `:78`): strike "NO_COLOR byte-identity" and "pyte content unchanged"; cite NoColorHasNoSGR, EpilogueAfterFinalFrame, and the renderBoard zero-escape pin instead.
4. Log: amend the fold entry (issue `:101-102`) so it no longer claims the phantom citations were removed until they actually are (or simply complete the removal, making the claim true; same for `:84`'s "pyte content tests green unchanged").
5. Delete the duplicate empty `## Done when` section (issue `:53-55`).
6. *(resolves Important #1, optional but recommended)* Add a delivered-item line for the flat-path pin once the reportWinner board-mode test lands.
