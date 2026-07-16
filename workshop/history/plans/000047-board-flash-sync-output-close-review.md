# Boundary Review — metis#47 (whole-issue close)

| field | value |
|-------|-------|
| issue | 47 — board flashes on repaint — wrap flushes in DEC 2026 synchronized output |
| repo | metis |
| issue file | workshop/issues/000047-board-flash-sync-output.md |
| boundary | whole-issue close |
| milestone | — |
| window | d4c15436c45815f850ac681b015c78ec9f50b86d..HEAD |
| command | sdlc close --issue 47 |
| reviewer | claude |
| timestamp | 2026-07-16T00:29:40-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything needed. The change is small and well-pinned; verifying the erase-call sites confirms both are bracketed. Here is the review.

```verdict
verdict: SHIP
confidence: high
```

This boundary delivers exactly what the issue commits to: every flush path (`Write`-inline, `paint`, `forceFlush` — all funneling through `flushLocked` — plus `close`) now brackets its erase→dump→redraw cycle in DEC 2026 BSU/ESU, and the new test pins the invariant (balanced pairs, no nesting, every erase inside a bracket) rather than the implementation. A shadow-sweep of `erase()` call sites confirms only two callers exist (`board.go:223`, `board.go:244`) and both are bracketed — no unsynchronized erase remains. The board is TTY-gated (`run.go:139`), so the private-mode escapes never reach piped/CI output. The `-race` side-fix to `steppingClock` is legitimate: `TestRunExperiment_BoardMode` (board_test.go:234) passes it as `runOpts.now`, which ParExec goroutines call concurrently. I could not execute the test suite (this review context is read-only; `go test` is blocked), so "existing tests keep passing" rests on inspection — the two updated suffix assertions are strengthened, not weakened, and are consistent with the emission order I traced by hand.

**1. Strengths**

- The bracket test (board_test.go:431) asserts the *invariant* — balance, non-nesting, and "every `\x1b[J]` is inside an open bracket" — instead of matching an exact byte stream. That survives refactors; it's the right shape for an escape-sequence contract.
- ANSI stays exclusively in `boardWriter`; `renderBoard` remains pure and untouched. The paint/content split held under modification (ARCH-PURE: pass).
- The two pre-existing suffix assertions (board_test.go:320, :413) were updated to expect `BOARD\n\x1b[?2026l` — they still verify the board repaints last *and* now additionally pin the trailing sync-end. Correctly tightened, not loosened.
- The `steppingClock` mutex is a real concurrency fix at the right scope, with the why recorded in the comment (board_test.go:132) and the Log (including the pathological 169s→2.6s run it explains).

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**

- **ARCH-DRY** (board.go:243–253): `close()` hand-duplicates `flushLocked`'s bracket+erase+dump+redraw sequence. If `close` newline-completed `b.pending` and delegated to `b.flushLocked(b.now())`, the bracket pair, erase, and redraw would have one source — and `close` would inherit the `defer`-based ESU for free. Two harmless behavior deltas to weigh (a never-painted board would gain a hide/show cursor pair; `lastFlush` gets set at close): both inert. Pre-existing duplication that this diff extended by one more pair; fine to leave, worth consolidating next time this file is touched.
- Panic-safety asymmetry: `flushLocked` emits ESU via `defer`, `close` emits it inline (board.go:253). No realistic panic path exists between them (`Fprint` to a builder/stdout doesn't panic), and terminals time-box an unclosed BSU anyway — noting only because the DRY consolidation above erases the asymmetry too.
- `scriptedClock` (progress_test.go:66) mutates `i` unlocked — the same latent race shape `steppingClock` just had. All current usages are single-goroutine, but if one is ever passed as `runOpts.now` it re-ships the #46 race. A one-line mutex (or a comment forbidding concurrent use) would close the parity gap.

**5. Test coverage notes**

The exact bug class this diff could ship — an erase path escaping the bracket, or unbalanced pairs corrupting terminal state — is directly covered, including through `close()`. `TestRunExperiment_BoardMode` still pins `\x1b[?25h` as the final byte sequence, which transitively verifies the ESU-before-cursor-restore ordering at close. The "live pty run" Done-when item is evidence-logged (BSU=23/ESU=23) but is inherently a manual step; nothing more to automate there. No gaps worth blocking on.

**6. Architectural notes**

- ARCH-PURE: pass — escape emission stayed at the IO boundary; the pure renderer is untouched.
- ARCH-PURPOSE: pass — the shadow-sweep found no unbracketed erase consumer; nothing was deferred as "follow-up."
- ARCH-DRY: flagged above (Minor) — `close`/`flushLocked` consolidation.
- Forward note: post-close `Write` passthrough (board.go:179) is deliberately unbracketed — correct today since no erase can follow, but if a "reopen board" path is ever added, the bracket invariant test should be extended to cover it.
- The issue's remaining UX siblings (startup delay, 3h ETA) are separate filings per the Log; no scope bleed here.

**7. Plan revision recommendations** — none. The plan's single checkbox matches the delivered code; no Core concepts table exists (appropriately, for a one-seam change), and the Spec/Done-when are satisfied as written.
