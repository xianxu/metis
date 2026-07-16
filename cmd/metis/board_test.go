package main

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/sampler"
)

// renderBoard is the PURE frame: aggregate line, fold rows (✓ done / ▸ in-flight /
// queued), overflow cap, leaves+throughput line. NO ANSI — escape codes live only in
// boardWriter (the paint/content split keeps this byte-testable).
func TestRenderBoard(t *testing.T) {
	mkState := func(rows []passRow) boardState {
		st := boardState{
			st: progressState{
				nested: true,
				outerK: 1, outerTotal: 3, outerKind: sampler.SizeExact,
				configK: 14, configTotal: 36, configKind: sampler.SizeExact,
				foldK: 47, foldTotal: 108, foldKind: sampler.SizeExact,
				outerScores: []float64{0.798},
			},
			rows: rows,
		}
		st.rate.add(at(0))
		st.rate.add(at(2824)) // 2 completions in 2.824s ≈ 42.5/min at now=2824
		return st
	}
	rows := []passRow{
		{done: true, heldOut: 0.798},
		{configK: 8, foldK: 25, best: 0.834, hasBest: true},
		{}, // queued: no events yet
	}
	lines := renderBoard(mkState(rows), boardEnv{width: 100, now: at(2824), busy: 8, capacity: 8})
	frame := strings.Join(lines, "\n")
	for _, want := range []string{
		"outer 1/3", "configs 14/36", "folds 47/108", "est 0.7980",
		"fold 0 ✓ held-out 0.7980",
		"fold 1 ▸ configs 8/12 · folds 25/36 · best 0.8340",
		"fold 2 — queued",
		"leaves 8/8", "42.5 folds/min", "ETA",
	} {
		if !strings.Contains(frame, want) {
			t.Errorf("frame missing %q:\n%s", want, frame)
		}
	}
	if len(lines) != 5 { // aggregate + 3 fold rows + leaves
		t.Errorf("want 5 lines, got %d:\n%s", len(lines), frame)
	}
	if strings.Contains(frame, "\x1b") {
		t.Error("renderer must emit NO escape codes")
	}

	// Per-row denominators derive from the aggregate totals (36 configs / 3 outer = 12).
	// All-done: every row ✓, no ETA segment (nothing remaining).
	allDone := []passRow{{done: true, heldOut: 0.79}, {done: true, heldOut: 0.81}, {done: true, heldOut: 0.82}}
	st := mkState(allDone)
	st.st.outerK, st.st.foldK, st.st.configK = 3, 108, 36
	st.st.outerScores = []float64{0.79, 0.81, 0.82}
	frame = strings.Join(renderBoard(st, boardEnv{width: 100, now: at(2824), busy: 0, capacity: 8}), "\n")
	if strings.Contains(frame, "▸") || strings.Contains(frame, "ETA") {
		t.Errorf("all-done: no in-flight rows, no ETA:\n%s", frame)
	}

	// Flat (no rows): exactly 2 lines — the aggregate + leaves.
	flat := boardState{st: progressState{foldK: 3, foldTotal: 5, foldKind: sampler.SizeExact, flatScores: []float64{0.8}}}
	if got := renderBoard(flat, boardEnv{width: 100, now: at(0), busy: 2, capacity: 8}); len(got) != 2 {
		t.Errorf("flat board = aggregate + leaves, got %d lines: %v", len(got), got)
	}

	// Overflow: 14 folds → 12 rows + "… +2 more" + leaves + aggregate = 15 lines.
	many := make([]passRow, 14)
	st = mkState(many)
	st.st.outerTotal = 14
	if got := renderBoard(st, boardEnv{width: 100, now: at(2824), busy: 8, capacity: 8}); len(got) != 15 {
		t.Errorf("overflow: want 15 lines (1+12+1+1), got %d", len(got))
	} else if !strings.Contains(strings.Join(got, "\n"), "+2 more") {
		t.Errorf("overflow marker missing:\n%s", strings.Join(got, "\n"))
	}

	// Width clamp: every line ≤ width, truncated with …
	narrow := renderBoard(mkState(rows), boardEnv{width: 30, now: at(2824), busy: 8, capacity: 8})
	for _, l := range narrow {
		if n := len([]rune(l)); n > 30 {
			t.Errorf("line exceeds width 30 (%d runes): %q", n, l)
		}
	}
	if !strings.Contains(strings.Join(narrow, "\n"), "…") {
		t.Error("clamped lines should truncate with …")
	}

	// No gauge (capacity 0): leaves segment absent, throughput still present.
	frame = strings.Join(renderBoard(mkState(rows), boardEnv{width: 100, now: at(2824)}), "\n")
	if strings.Contains(frame, "leaves") {
		t.Errorf("no gauge → no leaves segment:\n%s", frame)
	}
	if !strings.Contains(frame, "folds/min") {
		t.Errorf("throughput must survive a missing gauge:\n%s", frame)
	}

	// Rate unavailable (fresh ring): "— folds/min".
	st = mkState(rows)
	st.rate = movingRate{}
	frame = strings.Join(renderBoard(st, boardEnv{width: 100, now: at(0), busy: 1, capacity: 8}), "\n")
	if !strings.Contains(frame, "— folds/min") {
		t.Errorf("unavailable rate renders —:\n%s", frame)
	}
}

// fmtETA is compact and human: seconds under a minute, m+s under an hour.
func TestFmtETA(t *testing.T) {
	cases := map[time.Duration]string{
		34 * time.Second:                            "34s",
		190 * time.Second:                           "3m10s",
		2*time.Hour + 5*time.Minute:                 "2h5m",
		2*time.Hour + 5*time.Minute + 9*time.Second: "2h5m",
	}
	for d, want := range cases {
		if got := fmtETA(d); got != want {
			t.Errorf("fmtETA(%v) = %q, want %q", d, got, want)
		}
	}
}

// boardWriter pins the board to the bottom: passthrough writes scroll above the
// stored frame; erase sequences separate frames; close is idempotent and restores
// the cursor. Driven directly (no ticker) against a bytes.Buffer "terminal".
func TestBoardWriter_PinBottom(t *testing.T) {
	var term strings.Builder
	bw := newBoardWriter(&term)

	bw.paint([]string{"AGG line", "fold 0 ▸"})
	first := term.String()
	if !strings.HasPrefix(first, "\x1b[?25l") {
		t.Errorf("first paint must hide the cursor: %q", first)
	}
	if !strings.Contains(first, "AGG line\nfold 0 ▸\n") {
		t.Errorf("frame not painted: %q", first)
	}
	if strings.Contains(first, "\x1b[2A") {
		t.Errorf("first paint has nothing to erase: %q", first)
	}

	// Passthrough: erase (up 2 + clear), the step line, the repainted stored frame.
	if _, err := bw.Write([]byte("⚡ step train\n")); err != nil {
		t.Fatal(err)
	}
	s := term.String()[len(first):]
	wantOrder := []string{"\x1b[2A\x1b[J", "⚡ step train\n", "AGG line\nfold 0 ▸\n"}
	pos := 0
	for _, w := range wantOrder {
		i := strings.Index(s[pos:], w)
		if i < 0 {
			t.Fatalf("passthrough sequence missing %q in order: %q", w, s)
		}
		pos += i + len(w)
	}

	// An unterminated write is held back until its newline arrives.
	pre := term.Len()
	bw.Write([]byte("partial"))
	if got := term.String()[pre:]; strings.Contains(got, "partial") {
		t.Errorf("unterminated tail must be held, not fused into the board: %q", got)
	}
	bw.Write([]byte(" line\n"))
	if !strings.Contains(term.String(), "partial line\n") {
		t.Error("the completed line must flush")
	}

	// A fresh paint replaces the frame.
	bw.paint([]string{"AGG line", "fold 0 ✓"})
	if !strings.Contains(term.String(), "fold 0 ✓") {
		t.Error("paint must draw the new frame")
	}

	// close: final frame stays, cursor restored; idempotent.
	bw.close()
	if !strings.HasSuffix(term.String(), "\x1b[?25h") {
		t.Errorf("close must restore the cursor last: %q", term.String()[term.Len()-20:])
	}
	n := term.Len()
	bw.close()
	if term.Len() != n {
		t.Error("close must be idempotent")
	}
	// Post-close writes pass straight through (no board left to protect).
	bw.Write([]byte("after\n"))
	if !strings.HasSuffix(term.String(), "after\n") {
		t.Error("post-close writes pass through")
	}
}

// A close with a held unterminated tail flushes it (newline-completed) above the
// final frame — no output is ever swallowed.
func TestBoardWriter_CloseFlushesPending(t *testing.T) {
	var term strings.Builder
	bw := newBoardWriter(&term)
	bw.paint([]string{"B"})
	bw.Write([]byte("tail-no-newline"))
	bw.close()
	if !strings.Contains(term.String(), "tail-no-newline\n") {
		t.Errorf("held tail must flush at close: %q", term.String())
	}
}

// Board mode end-to-end over the fixture sweep: frames paint (cursor hide, fold rows),
// the #30 plain lines are REPLACED (not duplicated), the final frame carries the
// completed counts, and a capture warning — the plan-review bypass route (o.out) —
// lands ABOVE the board through the compositor, never after the last erase.
func TestRunExperiment_BoardMode(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b]"))
	var term strings.Builder
	_, err := runExperiment(runOpts{
		expPath: expPath,
		now:     fixedNow(),
		git:     fakeGitProbe{name: "metis", sha: "sha", dirty: false},
		exec:    foldFakeExec{},
		tui:     true, // board mode: runExperiment wraps out in the compositor
		out:     &term,
	})
	if err != nil {
		t.Fatalf("board-mode nested run: %v", err)
	}
	s := term.String()
	if !strings.Contains(s, "\x1b[?25l") || !strings.Contains(s, "\x1b[J") {
		t.Errorf("board mode must paint (cursor hide + erase sequences):\n%q", s[:min(len(s), 400)])
	}
	if !strings.HasSuffix(s, "\x1b[?25h") {
		t.Errorf("the deferred close must restore the cursor LAST: %q", s[len(s)-40:])
	}
	if strings.Contains(s, "metis: progress") {
		t.Errorf("the board REPLACES the plain progress lines")
	}
	if !strings.Contains(s, "outer 2/2") || !strings.Contains(s, "fold 0 ✓") || !strings.Contains(s, "fold 1 ✓") {
		t.Errorf("the final frame must show completed folds:\n%s", s)
	}
	// The bypass route: the fake-exec fixture has no traced closure → captureSweepCode
	// notes "no first-party code closure" via o.out — which after the runExperiment
	// reorder IS the compositor, so the text must appear before the final erase, never
	// as a bare trailing write (the plan-review o.out bypass, pinned).
	warnIdx := strings.Index(s, "no first-party code closure")
	if warnIdx < 0 {
		t.Fatalf("expected the uncaptured-code note in a fake-exec fixture:\n%s", s)
	}
	if lastErase := strings.LastIndex(s, "\x1b[J"); warnIdx > lastErase {
		t.Errorf("the capture warning bypassed the compositor (after the last erase)")
	}

	// Contrast: tui=false on the same fixture — byte-clean plain lines, no board.
	var plain strings.Builder
	ws2 := t.TempDir()
	if _, err := runExperiment(runOpts{
		expPath: writeShapeFile(t, ws2, foldShapeCVMD("[a, b]")),
		now:     fixedNow(), git: fakeGitProbe{name: "metis", sha: "sha"},
		exec: foldFakeExec{}, out: &plain,
	}); err != nil {
		t.Fatalf("plain run: %v", err)
	}
	if strings.Contains(plain.String(), "\x1b") {
		t.Error("tui=false must emit zero escape codes")
	}
	if !strings.Contains(plain.String(), "metis: progress") {
		t.Error("tui=false keeps the #30 plain lines")
	}
}

// --no-tui and non-TTY stdout both force tui=false through the real CLI parse; a
// dry run never boards. (isCharDevice on a test's non-terminal stdout is false, so
// the flag path is what we can pin here; the char-device branch is covered by the
// close-evidence pty run.)
func TestCmdRun_NoTUIFlagParses(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a]"))
	// The real entrypoint with the documented order; --dry-run avoids running steps.
	if err := run([]string{"run", "--no-tui", "--dry-run", expPath}); err != nil {
		t.Fatalf("--no-tui must parse: %v", err)
	}
}

// The fork-server pool's fallback notice is the OTHER construction-time-capture bypass
// route the plan review named: after the runExperiment reorder the pool is built with
// the compositor, so a mid-sweep noticeOnce must land ABOVE the board (close-review
// Important — the route is guarded by construction order; this pins it directly).
func TestServerPool_NoticeRoutesThroughBoard(t *testing.T) {
	var term strings.Builder
	bw := newBoardWriter(&term)
	bw.paint([]string{"BOARD"})
	pool := newServerPool(bw) // what runExperiment does post-reorder: pool captures the compositor
	pool.noticeOnce("k", "server died; falling back to legacy exec")
	s := term.String()
	notice := strings.Index(s, "metis: forkserver: server died")
	if notice < 0 {
		t.Fatalf("notice missing: %q", s)
	}
	// The compositor's passthrough shape: an erase precedes the notice (the old board
	// is cleared first) and the frame is repainted BELOW it — a bypassing write would
	// instead land after the final frame with no repaint following.
	if erase := strings.Index(s, "\x1b[J"); erase < 0 || erase > notice {
		t.Errorf("the notice must be preceded by the board erase: %q", s)
	}
	if !strings.HasSuffix(s, "BOARD\n") {
		t.Errorf("the board must be repainted below the notice: %q", s)
	}
}
