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
