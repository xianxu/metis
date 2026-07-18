# Boundary Review — metis#56 (whole-issue close)

| field | value |
|-------|-------|
| issue | 56 — board status line keeps default color |
| repo | metis |
| issue file | workshop/issues/000056-board-status-line-keeps-default-color.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3d2592328faed03bddcae42228aca6b0fbee7fc6^..HEAD |
| command | sdlc close --issue 56 |
| reviewer | claude |
| timestamp | 2026-07-18T10:02:36-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
Test execution is blocked by a harness-level EPERM (the session-env mkdir fails even unsandboxed), so I verified by reading the tests instead: no test asserts the dim, so the removal can't break the suite on that axis. I have everything needed for the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code change itself is exactly what the Spec asked for and is correct: the `i == len(b.frame)-1` dim case is gone from `boardWriter.redraw`, so the status line falls through to default color like the fold rows, while the dim separator rule and bold aggregate (the #55 banding) are untouched. The paint/content split holds — the change lives entirely in the painter. What keeps this from a clean SHIP is the second half of the Spec: "reconcile the #55 wording in atlas **+ the archived issue's claims**" was only half-delivered. The atlas was reconciled, but the archived #55 issue still claims "the status line DIM", and two in-repo comments (the `redraw` doc comment and a test comment) still describe the removed behavior. All three fixes are one-line edits.

**1. Strengths**

- The removal is surgically minimal — one switch case deleted, nothing else touched in the paint path (`cmd/metis/board.go:368-378`). The erase math and separator accounting are unaffected, and I confirmed nothing downstream keyed off the last-line dim.
- The atlas reconciliation (`atlas/experiment.md:217-218`) doesn't just delete the stale claim — it states the new invariant with its rationale ("the status line stays default — live telemetry is not de-emphasized, #56"), which is exactly what a future reader needs.
- The Log is honest about test state ("no test pinned the dim — the #55 e2e asserts separator + bold aggregate only") — I verified this against `board_test.go:416` and the `sgrDim` grep; the claim is accurate, not aspirational.

**2. Critical findings**

None.

**3. Important findings**

- **Spec-stated reconciliation incomplete — the archived #55 issue still says "the status line DIM"** (`workshop/history/issues/000055-board-color-separation-and-summary-after-footer.md:30`). The Spec names this file class explicitly ("the archived issue's claims where they say 'dims the status line'"), and the Plan checkbox marks it done. Fix: edit that line to say the status line stays default (per #56). ARCH-PURPOSE — the reconciliation sweep was the second half of the issue's purpose, and it stopped at the atlas.
- **Stale doc comment on `redraw` contradicts the code** (`cmd/metis/board.go:358`): "...✓/▸ glyphs get state color, the status line is dim." The very function this diff edited still documents the removed behavior. Fix: "...the status line stays default (metis#56)". Same class as above — the shadow-sweep missed the code's own comments.
- **Stale test comment** (`cmd/metis/board_test.go:415`): "dim separator rule + bold aggregate + dim status line" — drop the last clause.
- **No test pins the new behavior.** Nothing asserts the status line is *not* dimmed, so a regression re-adding the dim would ship silently — the exact bug class this issue exists to fix. Cheap fix: in the #55 e2e (near `board_test.go:416`) or `TestBoardWriter_NoColorHasNoSGR`'s color-on sibling, assert the final frame's status line (`~slots`) is not preceded by `sgrDim` — e.g. `!strings.Contains(s, sgrDim+"~slots")` or that `sgrDim` occurs only on separator-rule lines.

**4. Minor findings**

- `## Done when` in the issue file is empty (line 29) — fill or drop the section.
- Duplicate `### 2026-07-18` headings in `## Log` (lines 37/39).
- `.goto` worktree-pointer churn rode along in the window — expected, no action.

**5. Test coverage notes**

I could not execute the suite (harness-level EPERM on Bash, even unsandboxed — not a repo problem), so "full suite green" from the Log is unverified by me. Static verification: `sgrDim` appears in tests only in `TestBoardWriter_NoColorHasNoSGR` (`board_test.go:723`, a no-SGR-at-all assertion), and the #55 e2e asserts only separator + bold aggregate — so no existing test contradicts the change, consistent with the Log. The gap is the missing positive pin of the new default-color behavior (Important finding above). The main agent should re-run the suite when adding that assertion.

**6. Architectural notes**

- ARCH-DRY: **pass** — pure case removal, no duplication introduced.
- ARCH-PURE: **pass** — decoration stays in the painter (`redraw`); `renderBoard` remains plain, preserving the #38 paint/content split. The `boardWriter` tests run against an injected `strings.Builder`, no IO.
- ARCH-PURPOSE: **flag** — the diff delivers the behavior change but under-delivers the reconciliation sweep that was the Spec's second clause. Three hand-maintained restatements of the old model survive (archived #55 issue, `board.go:358` comment, `board_test.go:415` comment). All are one-line fixes; see Important findings.

**7. Plan revision recommendations**

Add a `## Revisions` entry to issue #56 noting the checkbox "adjust tests/docs" was closed with the sweep incomplete (atlas only), and either extend the same checkbox's scope note or add a follow-through line: reconcile the archived #55 issue + the two in-code comments, and add the negative dim assertion. Once those land, the plan and code match and no further revision is needed.

---

## Re-review — 2026-07-18T10:07:19-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 56 — board status line keeps default color |
| repo | metis |
| issue file | workshop/issues/000056-board-status-line-keeps-default-color.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3d2592328faed03bddcae42228aca6b0fbee7fc6^..HEAD |
| command | sdlc close --issue 56 |
| reviewer | claude |
| timestamp | 2026-07-18T10:07:19-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything needed for the verdict. All prior-review fixes are verified folded; remaining findings are light.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

This window delivers the #56 behavior change plus the operator's scope addition, and — importantly — folds the previous FIX-THEN-SHIP review faithfully: the `redraw` dim case is gone (`cmd/metis/board.go:375-384`), the stale "dim status" restatements were reconciled in the `redraw` doc comment (board.go:365), the #55 e2e comment (board_test.go:415), and the archived #55 issue's Spec (history line 30), and both requested pins landed (the negative `sgrDim` regression pin and the closing-rule ordering assert, board_test.go:419-426). The closing rule itself is correct: painted only at `close()`, only when an epilogue exists and `width > 0`, dim only when color is on — so the NO_COLOR "SGR-only" doctrine from #55 holds (the rule stays structural), the erase math is untouched (nothing erases after close), and idempotency is preserved (`epi` nil'd, `closed` guard). What keeps this off a clean SHIP is one cheap docs gap: the closing rule is new visual surface added via the issue's `## Revisions`, and the atlas paragraph wasn't extended for it — the atlas edit only reconciled the status-line wording. Everything else is minor. I could not execute the suite (harness-level EPERM on Bash — the session-env mkdir fails, same blocker the prior reviewer recorded; not a repo problem), so test claims are verified by reading, not running.

**1. Strengths**

- The review fold is complete and honest — every Important finding from the prior verdict has a verifiable fix in this window, and the issue's `## Revisions` entry (issue file lines 47-52) records both the scope addition and the fold explicitly rather than silently absorbing them. This is the Revisions discipline working as designed.
- The regression pin (board_test.go:419) is well-constructed: the old bug wrapped the whole last line as `sgrDim + l + sgrReset`, and the e2e's status line begins with either `~slots` (running) or `starting` (startup, per renderBoard board.go:81-85) — so `sgrDim+"~slots"` / `sgrDim+"starting"` catches exactly the re-introduction of that wrap in both frame phases.
- The closing-rule assert (board_test.go:424) is properly anchored — it searches `s[restoreIdx:sumIdx]`, and a bad `restoreIdx`/`sumIdx` already fails the ordering assert at line 403 first, so the guard can't silently skip.
- Edge handling on the rule is right: `width == 0` (board never painted → no footer to band off) skips it; post-close epilogue writes pass straight through unruled, which is the correct degenerate behavior.

**2. Critical findings**

None.

**3. Important findings**

- **Atlas update appears missing for the closing rule** (`atlas/experiment.md:219-221`). The paragraph still describes the final state as epilogue "flushed after the final frame + cursor restore" — the new `log → separator → footer → closing rule → result` shape isn't there, even though the same paragraph was edited in this window for the status-line change. Fix is half a sentence: "…flushed after the final frame + cursor restore, banded off the footer by a second dim rule (#56)". Docs gate per AGENTS.md §8.

**4. Minor findings**

- ARCH-DRY: the 4-line rule block (`strings.Repeat("─", b.width)` + dim wrap + `Fprintln`) is now verbatim-duplicated between `close()` (board.go:340-344) and `redraw()` (board.go:369-373); if the glyph or dim policy ever changes in one, the other silently diverges. Extract a `func (b *boardWriter) ruleLine() string` helper.
- Archived #55 issue still says "dim status" in two spots the sweep didn't reach: the Done-when at `workshop/history/issues/000055-…md:47` ("dim status present as SGR in the raw stream" — now a false current-behavior claim) and the "(built)" log at line 90 (historical record, arguably fine as-is). One-line edit to line 47 if you touch the file anyway.
- `## Done when` in issue #56 is still empty (carried over from the prior review's minors); duplicate `### 2026-07-18` headings in `## Log` (lines 37/41).
- The NO_COLOR path of the closing rule is untested — `TestBoardWriter_NoColorHasNoSGR` (board_test.go:725) closes without an epilogue, so the rule never paints there. The guard is trivial; a one-line `epilogueWriter` write in that test would cover it. No action required.
- Theoretical only: `discardFrame` clears the frame but not `epi`, so a close after discard with a buffered epilogue would print a dangling rule above nothing — currently unreachable (epilogue writes happen only on the success path). Noting for the record.
- `.goto` worktree-pointer churn rode along in the window — expected, no action.

**5. Test coverage notes**

Suite unexecuted by me (harness EPERM on Bash, pre-existing environment blocker — the Log's "full suite green" claim is the implementor's, not independently verified). Static verification: both new asserts are consistent with the byte stream `close()` actually produces (restore → rule → epi, with `sumIdx` inside epi); the changes are assertion-additive so no existing pin can newly fail — `TestBoardWriter_EpilogueAfterFinalFrame` (board_test.go:743) uses ordering/containment asserts, not exact bytes, and tolerates the inserted rule; the NoColor test has no epilogue so the rule path doesn't fire there. The bug classes this diff could ship (dim regression, rule missing/misordered) are both pinned in the e2e. Coverage verdict: adequate, with the NO_COLOR-rule gap noted above as optional.

**6. Architectural notes**

- **ARCH-DRY: flag (minor)** — the duplicated rule block, see Minor findings. Otherwise the diff reuses existing seams (`epi`, `width`, `color`) rather than adding parallel state.
- **ARCH-PURE: pass** — decoration remains entirely in the painter/close IO shell; `renderBoard` stays plain (the #38 paint/content split holds), and all boardWriter tests run against an injected `strings.Builder`, no mocks, no real IO.
- **ARCH-PURPOSE: pass** — the shadow-sweep that the prior review flagged is now substantively complete: atlas, the archived #55 Spec claim, and both in-code comments derive from the new model, and the new behavior is enforced by tests, not just documented. The two archived-history stragglers (lines 47/90) are log-grade restatements in an explicitly low-signal directory, not live consumers — noted as Minor, not a purpose gap.

**7. Plan revision recommendations**

None required — the existing `## Revisions` entry already records both the scope addition and the review fold accurately, and the code matches it. If you take the atlas half-sentence, no plan change is needed for it (it completes the existing reconciliation claim rather than changing scope).
