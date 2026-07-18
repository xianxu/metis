# Boundary Review — metis#57 (whole-issue close)

| field | value |
|-------|-------|
| issue | 57 — board scrolling log renders gray |
| repo | metis |
| issue file | workshop/issues/000057-board-scrolling-log-renders-gray.md |
| boundary | whole-issue close |
| milestone | — |
| window | 029d38acad14a24c565ef5c17b2d7ab4c5628dbb^..HEAD |
| command | sdlc close --issue 57 |
| reviewer | claude |
| timestamp | 2026-07-18T10:39:23-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
Bash is unavailable in this sandbox (session-env write denied), but the window is a single commit (`029d38a`, the HEAD in the snapshot) and the supplied diff covers it fully — board.go + board_test.go only. I have everything needed from reads.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

This is a tight, correct refinement: the two scroll-region dump sites (`flushLocked` and `close`) consolidate into one `writeScroll` helper that gray-wraps chunks with `\x1b[90m…\x1b[0m` when color is on, exactly as the Spec states. The chunk always ends in a newline before the wrap closes, the reset precedes `redraw`, and both call sites emit inside the DEC 2026 sync bracket, so the frame and epilogue can't inherit the gray. The NO_COLOR path is unchanged and now test-pinned for scroll writes too. The only thing keeping this off a clean SHIP is the docs gate: the atlas banding entry wasn't extended for this surface, the same fold-in discipline #56 got.

**1. Strengths**
- Routing both dump sites through one helper (`board.go:357`) instead of wrapping inline twice is the right ARCH-DRY move — the SGR bracketing has one source of truth, so a future change (different color, OSC hyperlink stripping, whatever) lands once.
- The wrap is placed correctly relative to the compositor invariants: after `erase()`, before `redraw()`, inside the `?2026` bracket at both sites, and the trailing `sgrReset` rides after a guaranteed-final `\n` (both callers only pass newline-terminated chunks; `close` appends the newline first at `board.go:328-331`), so the frame's own SGRs start from a clean state.
- `TestBoardWriter_ScrollChunksAreGray` (`board_test.go:768`) pins all three contract halves in one scenario: gray scroll chunk, un-grayed frame, un-grayed epilogue — not just the happy wrap.
- Extending `TestBoardWriter_NoColorHasNoSGR` with a scroll write *and* adding `sgrGray` to its sweep list closes the exact regression class this diff could ship (a gray wrap leaking past the `b.color` gate).

**2. Critical findings** — none.

**3. Important findings**
- **Atlas update appears missing for the gray scroll surface** (docs gate). `atlas/experiment.md:215-221` is the banding entry and it absorbed the #56 refinement inline ("the status line stays default … #56"), but this window adds the third banding element — gray scroll-region chunks via `writeScroll` — with no atlas change in the range. The entry now under-describes the surface it documents. Fix is one clause in that bullet, e.g. "scroll-region chunks dump in bright-black gray (`writeScroll`, both pending-dump sites — #57) so the footer + result carry the eye." Note the close gate's atlas guard only auto-satisfies on docs-only windows, which this isn't — don't reach for `--no-atlas` when the update is this cheap.

**4. Minor findings**
- `board.go:353` — the `writeScroll` comment says "DIM when color is on" but the code emits gray/bright-black; the `sgrGray` constant's own comment explains dim was *rejected* for rendering inconsistently. Change "DIM" → "GRAY".
- The `close`-site dump path is never exercised with color on: in `TestBoardWriter_ScrollChunksAreGray` the stepping clock makes `Write` flush inline, so `writeScroll` is only hit via `flushLocked` and `pending` is empty at close. An unterminated-tail write (`bw.Write([]byte("partial"))`) before `close()` would pin the second site plus the newline-append + gray interaction in one assertion. Cheap to add, not blocking — both sites share the helper, so divergence risk is low.
- The issue's `## Done when` section is empty (`-`) — tracker hygiene; the Spec effectively serves as it, but the section should either state the criterion or be dropped.

**5. Test coverage notes**
The two failure modes this diff could ship — gray leaking under NO_COLOR, and gray bleeding into the frame/epilogue — are both directly pinned. The suite drives `boardWriter` against a `strings.Builder` with an injected clock (no IO, deterministic), consistent with the file's paint/content split. The one gap is the close-site dump noted above. The post-close `Write` passthrough (`board.go:245`) intentionally bypasses gray; that's outside the Spec's "both pending-dump sites" contract and fine, but if anyone later routes late output through it expecting gray, no test documents the distinction.

**6. Architectural notes**
- **ARCH-DRY: pass** — this diff *removes* a would-be duplication by introducing the shared helper. Pre-existing (not this window): the `sgrDim + rule + sgrReset` separator-rule wrap appears in both `close` (`board.go:343`) and `redraw` (`board.go:386`); if a fourth banding refinement touches the rule, extract it then.
- **ARCH-PURE: pass** — ANSI stays confined to the `boardWriter` compositor; `renderBoard` remains plain and byte-testable, honoring the #38 paint/content split the file header commits to. Tests run without mocks or IO.
- **ARCH-PURPOSE: pass** — shadow-sweep of pending-dump sites confirms both (`flushLocked:309`, `close:331`) derive from the one helper; frame, separator rule, and epilogue are the enumerated non-scroll surfaces and all stay un-grayed. Nothing the issue promised is deferred.

**7. Plan revision recommendations** — none; the single Plan item matches what shipped. The only doc that lags the code is the atlas (finding above), not the plan.

---

## Re-review — 2026-07-18T10:41:40-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 57 — board scrolling log renders gray |
| repo | metis |
| issue file | workshop/issues/000057-board-scrolling-log-renders-gray.md |
| boundary | whole-issue close |
| milestone | — |
| window | 029d38acad14a24c565ef5c17b2d7ab4c5628dbb^..HEAD |
| command | sdlc close --issue 57 |
| reviewer | claude |
| timestamp | 2026-07-18T10:41:40-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
All checks complete. Writing up the verdict.

```verdict
verdict: SHIP
confidence: high
```

This window is the #57 gray-scroll refinement plus the fold-in of the prior boundary review's FIX-THEN-SHIP findings — and every one of those findings is verifiably resolved in the range. The Spec's full contract is delivered and test-pinned: both pending-dump sites (`flushLocked` at `cmd/metis/board.go:309`, `close` at `cmd/metis/board.go:331`) route through the single `writeScroll` helper, which gray-wraps in `\x1b[90m…\x1b[0m` only when `b.color` is on; the frame, separator rule, and epilogue stay un-grayed; NO_COLOR emits zero SGR including `sgrGray`; and the atlas banding entry now names the gray scroll surface (`atlas/experiment.md:219-220`). I traced the test by hand against `steppingClock` (each `now()` call advances 300ms > the 250ms budget, so writes flush inline) and confirmed `TestBoardWriter_ScrollChunksAreGray` genuinely exercises *both* helper sites — the terminated line dumps via `flushLocked`, and the unterminated `"partial tail"` is held back and dumped by `close`'s newline-append path. One environment note: Bash was unavailable in this review sandbox (harness-level session-env failure), so I could not re-run the suite myself; the Log claims full `-race` green and my hand-trace of the assertions agrees, but the gate should have a green run on record.

**1. Strengths**
- The prior review's fold-in is complete and honest: atlas clause added, the `writeScroll` comment now correctly says GRAY (`board.go:353`), the close-site tail test landed (`board_test.go:775,784`), and the issue's `## Done when` was filled with the real criterion — no finding was quietly dropped.
- `TestBoardWriter_ScrollChunksAreGray` (`board_test.go:769`) is a model contract test: one scenario pins gray-on-scroll, un-grayed frame, un-grayed epilogue, *and* the close-time tail (newline-append + gray interaction) — the exact assertion `sgrGray+"partial tail\n"+sgrReset` proves the append happens inside the wrap.
- The single `writeScroll` helper (`board.go:357`) gives the SGR bracketing one source of truth (ARCH-DRY), and both call sites sit inside the DEC 2026 sync bracket, after `erase()` and before `redraw()`, with `sgrReset` riding after a guaranteed-final `\n` — the frame's own SGRs always start clean.
- Extending `TestBoardWriter_NoColorHasNoSGR` with a scroll write and adding `sgrGray` to the sweep list (`board_test.go:728,732`) closes the one regression class this diff could ship: gray leaking past the `b.color` gate.

**2. Critical findings** — none.

**3. Important findings** — none. The docs gate is satisfied in-window (atlas clause present; no new README-class surface — this is a visual output change with no new flags, commands, or config keys).

**4. Minor findings**
- Pre-existing, not this window: the `sgrDim + rule + sgrReset` separator wrap is duplicated between `close` (`board.go:342-344`) and `redraw` (`board.go:385-387`) — extract on the next banding touch (ARCH-DRY, deferred is fine).
- `writeScroll` issues three sink writes per chunk (prefix, body, reset); harmless inside the sync bracket and consistent with the file's style — noting only so nobody "optimizes" it into a buffer copy without reason.

**5. Test coverage notes**
Both failure modes this diff could ship — gray leaking under NO_COLOR, and gray bleeding into frame/epilogue — are directly pinned, and both `writeScroll` call sites are now exercised with color on (the gap the prior review flagged). Tests drive `boardWriter` against a `strings.Builder` with an injected clock: no IO, no mocks, deterministic. Remaining un-pinned distinction (acceptable): the post-close passthrough (`board.go:245`) intentionally bypasses gray; it's outside the Spec's "both pending-dump sites" contract.

**6. Architectural notes**
- **ARCH-DRY: pass.** The diff consolidates the two dump sites into one helper rather than wrapping inline twice; the only duplication in sight (separator rule) predates this window and is noted above.
- **ARCH-PURE: pass.** ANSI stays confined to the `boardWriter` compositor; `renderBoard` remains plain per the #38 paint/content split, and the compositor is tested purely via injected writer + clock.
- **ARCH-PURPOSE: pass.** Shadow-sweep of scroll-dump sites: `flushLocked:309` and `close:331` are the only two, both derive from `writeScroll`; frame, rule, and epilogue are the enumerated non-scroll surfaces and all stay un-grayed. Every Done-when clause — including the atlas naming — is delivered in-window; nothing the issue promised is deferred.

**7. Plan revision recommendations** — none. The Plan's single checked item matches what shipped, the Done-when now states the real criterion, and the atlas no longer lags the code.
