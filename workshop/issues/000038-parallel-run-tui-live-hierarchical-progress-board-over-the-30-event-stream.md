---
id: 000038
status: open
deps: [metis#30]
github_issue:
created: 2026-07-14
updated: 2026-07-14
estimate_hours:
---

# parallel-run TUI — live hierarchical progress board over the #30 event stream

## Problem

With metis#31, a nested sweep fans out across NumCPU leaves — 5 outer folds × 99 configs × 5 inner
folds = 2,475 fold runs executing concurrently — and the terminal shows **nothing** until the pipe
flushes at exit (felt acutely on the metis#35 honest-beat run: minutes of silence, no way to tell
"downloading" from "hung" from "3/5 outer folds done"). metis#30's aggregated single line fixes
blindness for a SERIAL mental model, but under parallelism one line can't render what's actually
happening: several outer folds in flight at once, each with its own inner progress, plus a shared
leaf-semaphore occupancy. The operator (2026-07-14) wants a TUI/curses implementation so parallel
progress is comprehensible.

## Spec

A live terminal progress board rendered from the #30 event stream (`SizeHint` + per-`Tell`
`ProgressEvent`s) — presentation only, no new instrumentation in `pkg/sampler` beyond what #30 adds:

- **Hierarchical board**, one row per active level: overall `outer j/k`, then a row per in-flight
  outer fold (`fold 2 · inner 47/99 · best 0.834`), plus a leaf line (`leaves 8/10 busy`, from the
  #31 semaphore occupancy). Completed folds collapse to their result (`fold 0 ✓ est 0.812`).
- **Running estimates in place**: incumbent per level from the #30 event payload (best-so-far
  inner; outer mean±SE as folds land).
- **Terminal-shaped first** (single-threaded-attention): a full-screen curses board when stdout is
  a TTY; **degrade to #30's aggregated line(s) when piped/non-TTY** (CI, logs, `metis run ... >
  file`) — the TUI must never corrupt captured logs. `--no-tui` forces the plain path.
- Implementation sketch: a Go TUI lib (bubbletea or tcell — pick at design; bubbletea's
  model/update/view fits the event-stream shape) consuming the same injected `progress` callback,
  in `cmd/metis` only (ARCH-PURE: `pkg/sampler` stays pure; the callback remains the seam).
- Explicitly out: historical run browsing, ledger visualization, anything beyond the live run.

## Done when

- A nested `metis run` on a TTY shows a live board: overall outer progress, per-in-flight-fold
  inner progress + incumbent, leaf occupancy; folds collapse to results as they complete.
- Piped / non-TTY / `--no-tui` output is exactly the #30 plain rendering (byte-stable, no escape
  codes) — verified by running with stdout redirected.
- No `pkg/sampler` API change beyond #30's (`SizeHint` + `progress`) — the TUI is a `cmd/metis`
  renderer over the same seam.

## Plan

- [ ] Blocked on metis#30 (the event stream). At claim: pick the TUI lib, design the board layout
  with the operator, then spec + change-code.

## Log

### 2026-07-14
- Filed by operator request during the metis#35 honest-beat run — the run sat minutes with zero
  output (pipe-buffered, no progress channel), indistinguishable from a hang; #31 parallelism makes
  a single aggregated line (the #30 renderer) insufficient for comprehension. Layering: #30 owns
  the instrumentation (SizeHint + progress callback), this issue owns the TUI presentation; deps
  reflect that.
