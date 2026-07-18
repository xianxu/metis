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
