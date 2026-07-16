package main

// metis#38: the live progress board — the PURE frame renderer (this half) and the
// pin-bottom ANSI compositor (boardWriter, below). Presentation only, over the #30
// sink's boardState snapshot: no pkg/sampler change, no TUI library (the board is
// output-only — a hand-rolled repaint of N lines; see the plan's no-lib rationale).
// The paint/content split is deliberate: renderBoard returns plain lines (byte-
// testable, zero escape codes); ANSI lives ONLY in boardWriter.

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// boardEnv is the render-time environment: terminal width, the current instant
// (for rate/ETA), and the leaf-semaphore gauge (capacity 0 = no gauge → segment
// omitted, e.g. a serial run).
type boardEnv struct {
	width          int
	now            time.Time
	busy, capacity int
}

// maxFoldRows caps the per-fold section; beyond it the remainder collapses to
// an "… +N more" line (a 20-fold sweep must not paint a 22-line board).
const maxFoldRows = 12

// renderBoard renders the frame: the #30 aggregate line, one row per outer fold
// (✓ done → held-out score · ▸ in-flight → per-pass counters + incumbent · queued),
// and the leaves/throughput/ETA line. Pure; width-clamped (a wrapped line would
// break the compositor's erase-count bookkeeping).
func renderBoard(bs boardState, env boardEnv) []string {
	var lines []string
	// Row 1: the aggregate — the same core the plain line prints (one source, no
	// prefix stripping).
	lines = append(lines, progressCore(bs.st))

	// Per-fold rows (nested only; flat runs have no rows).
	shown := len(bs.rows)
	if shown > maxFoldRows {
		shown = maxFoldRows
	}
	// Per-row denominators derive from the seeded aggregate totals (per-pass share).
	perConfigs, perFolds := 0, 0
	if n := len(bs.rows); n > 0 {
		perConfigs = bs.st.configTotal / n
		perFolds = bs.st.foldTotal / n
	}
	for i := 0; i < shown; i++ {
		r := bs.rows[i]
		switch {
		case r.done:
			lines = append(lines, fmt.Sprintf("  fold %d ✓ held-out %.4f", i, r.heldOut))
		case r.configK == 0 && r.foldK == 0:
			lines = append(lines, fmt.Sprintf("  fold %d — queued", i))
		default:
			b := ""
			if r.hasBest {
				b = fmt.Sprintf(" · best %.4f", r.best)
			}
			lines = append(lines, fmt.Sprintf("  fold %d ▸ configs %d/%d · folds %d/%d%s",
				i, r.configK, perConfigs, r.foldK, perFolds, b))
		}
	}
	if hidden := len(bs.rows) - shown; hidden > 0 {
		lines = append(lines, fmt.Sprintf("  … +%d more", hidden))
	}

	// Leaves / throughput / ETA.
	var segs []string
	if env.capacity > 0 {
		segs = append(segs, fmt.Sprintf("leaves %d/%d", env.busy, env.capacity))
	}
	if perMin, ok := bs.rate.rate(env.now); ok {
		segs = append(segs, fmt.Sprintf("%.1f folds/min", perMin))
	} else {
		segs = append(segs, "— folds/min")
	}
	if remaining := bs.st.foldTotal - bs.st.foldK; remaining > 0 {
		if eta, ok := bs.rate.eta(env.now, remaining); ok {
			segs = append(segs, "ETA "+fmtETA(eta))
		}
	}
	lines = append(lines, strings.Join(segs, " · "))

	for i, l := range lines {
		lines[i] = clampLine(l, env.width)
	}
	return lines
}

// fmtETA renders a duration compactly: 34s · 3m10s · 2h5m.
func fmtETA(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// (Height analog of the width limitation: a terminal SHORTER than the board clamps
// cursor-up at the screen top and desyncs the erase count — the board caps at ~15
// lines; terminals that small are out of scope, same accepted trade as resize.)

// clampLine truncates to width runes with a trailing … (a wrapped physical line
// would desync the compositor's cursor-up erase count — width is load-bearing).
func clampLine(s string, width int) string {
	if width <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	return string(r[:width-1]) + "…"
}

// ── boardWriter: the pin-bottom ANSI compositor ──────────────────────────────

// boardWriter owns the terminal: the board is pinned to the bottom while every
// other write (step logs, warnings) scrolls ABOVE it. Paint-only — it stores the
// last rendered frame and NEVER calls back into the sink (the one global lock
// order is sink.mu → bw.mu; a callback here would invert it). All output must
// route through this writer once it exists — a bypassing write corrupts the board
// (see the plan's writer-plumbing note: writer identity is temporal).
//
// metis#46: DOUBLE-BUFFERED with a bounded flush rate. The original design ran a
// full erase→write→repaint cycle per passthrough write; a warm-cache sweep emits
// hundreds of lines/second and real terminals — especially mux layers (the
// operator's ghostty-in-cmux) — paint asynchronously mid-sequence and tear under
// that flood. Now passthrough COALESCES into `pending` and the terminal sees one
// atomic erase→dump→repaint per flushBudget (~4Hz) — quiet writes (a cold run's
// sparse lines) still flush inline because the budget has long elapsed.
//
//	Write(p)     append to pending; flush inline iff the budget elapsed or the
//	             size cap is hit (bound memory under a frozen-budget flood).
//	paint(lines) store the frame; flush under the same budget.
//	tick/close   force-flush (the 500ms tick restores the board after a burst;
//	             close is idempotent, flushes everything, restores the cursor).
//
// It serializes internally (replacing syncWriter in board mode — one wrap, not two).
type boardWriter struct {
	mu        sync.Mutex
	w         io.Writer
	now       func() time.Time
	frame     []string // the stored last frame (drawn on each flush)
	painted   int      // physical lines currently on screen (the erase count)
	closed    bool
	pending   []byte // coalesced passthrough awaiting the next flush
	lastFlush time.Time
	hidden    bool // cursor-hide emitted (once, at the first flush that paints)
}

// flushBudget bounds the erase/repaint rate: under a flood the terminal gets one
// atomic update per budget window (~4Hz reads calm; per-line strobed at 500Hz).
const flushBudget = 250 * time.Millisecond

// pendingCap force-flushes a frozen-budget flood so pending can't grow unbounded.
const pendingCap = 64 << 10

func newBoardWriter(w io.Writer, now func() time.Time) *boardWriter {
	if now == nil {
		now = time.Now
	}
	return &boardWriter{w: w, now: now}
}

// Write is the passthrough seam: everything the sweep prints lands above the board.
func (b *boardWriter) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return b.w.Write(p)
	}
	b.pending = append(b.pending, p...)
	if now := b.now(); now.Sub(b.lastFlush) >= flushBudget || len(b.pending) > pendingCap {
		b.flushLocked(now)
	}
	return len(p), nil
}

// paint stores a fresh frame (the sink's emit target) and flushes under the budget.
func (b *boardWriter) paint(lines []string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.frame = lines
	if now := b.now(); now.Sub(b.lastFlush) >= flushBudget {
		b.flushLocked(now)
	}
}

// forceFlush is the tick/finish path: draw the freshest state regardless of budget
// (this is what re-pins the board after a burst window and keeps ETA/rate moving).
func (b *boardWriter) forceFlush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.flushLocked(b.now())
}

// discardFrame atomically erases the live board and forgets it. Pending complete
// ordinary lines still flush, but neither this update nor the later deferred close
// can redraw stale progress after a sweep failure.
func (b *boardWriter) discardFrame() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.frame = nil
	b.flushLocked(b.now())
}

// flushLocked is the ONE atomic terminal update: erase the painted board, dump the
// complete pending lines, redraw the stored frame — bracketed in DEC 2026 synchronized
// output (metis#47: supporting terminals apply it atomically, killing the erase→redraw
// flash; others ignore the private mode). Caller holds b.mu.
func (b *boardWriter) flushLocked(now time.Time) {
	if !b.hidden && b.frame != nil {
		fmt.Fprint(b.w, "\x1b[?25l") // first painting flush hides the cursor
		b.hidden = true
	}
	fmt.Fprint(b.w, "\x1b[?2026h")
	defer fmt.Fprint(b.w, "\x1b[?2026l")
	b.erase()
	// Hold back an unterminated tail: a partial line fused into the board's first
	// row would corrupt both; it flushes when its newline arrives (or at close).
	if i := lastNewline(b.pending); i >= 0 {
		b.w.Write(b.pending[:i+1])
		b.pending = b.pending[i+1:]
	}
	b.redraw()
	b.lastFlush = now
}

// close flushes everything (pending tail newline-completed, final frame) and
// restores the cursor. Idempotent (deferred at construction — error returns must
// not leak a hidden cursor).
func (b *boardWriter) close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	fmt.Fprint(b.w, "\x1b[?2026h") // metis#47: the final update is atomic too
	b.erase()
	if len(b.pending) > 0 {
		if b.pending[len(b.pending)-1] != '\n' {
			b.pending = append(b.pending, '\n')
		}
		b.w.Write(b.pending)
		b.pending = nil
	}
	b.redraw()
	fmt.Fprint(b.w, "\x1b[?2026l")
	if b.hidden {
		fmt.Fprint(b.w, "\x1b[?25h")
	}
	b.closed = true
}

// erase clears the painted board region: cursor up N, clear to screen end.
// Caller holds b.mu.
func (b *boardWriter) erase() {
	if b.painted == 0 {
		return
	}
	fmt.Fprintf(b.w, "\x1b[%dA\x1b[J", b.painted)
	b.painted = 0
}

// redraw paints the stored frame. Caller holds b.mu (and has erased).
func (b *boardWriter) redraw() {
	for _, l := range b.frame {
		fmt.Fprintln(b.w, l)
	}
	b.painted = len(b.frame)
}

func lastNewline(p []byte) int {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '\n' {
			return i
		}
	}
	return -1
}
