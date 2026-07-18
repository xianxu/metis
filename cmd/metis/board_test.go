package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/sampler"
)

// renderBoard is the PURE frame: aggregate line, outer-fold rows (✓ done /
// ▸ in-flight / queued), overflow cap, and the slots/rate line. NO ANSI —
// escape codes live only in boardWriter (the paint/content split keeps this
// byte-testable).
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
		for i := 0; i < 16; i++ {
			st.rate.add(at(i * 1000))
		}
		return st
	}
	rows := []passRow{
		{done: true, heldOut: 0.798},
		{configK: 8, foldK: 25, best: 0.834, hasBest: true},
		{}, // queued: no events yet
	}
	lines := renderBoard(mkState(rows), boardEnv{width: 100, now: at(21176), busy: 8, capacity: 8})
	frame := strings.Join(lines, "\n")
	for _, want := range []string{
		"outer folds 1/3", "configs scored 14/36", "inner-CV runs 47/108", "est 0.7980",
		"outer fold 0 ✓ held-out 0.7980",
		"outer fold 1 ▸ configs scored 8/12 · inner-CV runs 25/36 · best 0.8340",
		"outer fold 2 — queued",
		"~slots 8/8", "42.5 inner-CV runs/min", "~ETA",
	} {
		if !strings.Contains(frame, want) {
			t.Errorf("frame missing %q:\n%s", want, frame)
		}
	}
	if len(lines) != 5 { // aggregate + 3 outer-fold rows + slots/rate
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
	frame = strings.Join(renderBoard(st, boardEnv{width: 100, now: at(21176), busy: 0, capacity: 8}), "\n")
	if strings.Contains(frame, "▸") || strings.Contains(frame, "ETA") {
		t.Errorf("all-done: no in-flight rows, no ETA:\n%s", frame)
	}

	// Flat (no rows): exactly 2 lines — the aggregate + slots/rate.
	flat := boardState{st: progressState{foldK: 3, foldTotal: 5, foldKind: sampler.SizeExact, flatScores: []float64{0.8}}}
	if got := renderBoard(flat, boardEnv{width: 100, now: at(0), busy: 2, capacity: 8}); len(got) != 2 {
		t.Errorf("flat board = aggregate + leaves, got %d lines: %v", len(got), got)
	}

	// Overflow: 14 folds → 12 rows + "… +2 more" + slots/rate + aggregate = 15 lines.
	many := make([]passRow, 14)
	st = mkState(many)
	st.st.outerTotal = 14
	if got := renderBoard(st, boardEnv{width: 100, now: at(21176), busy: 8, capacity: 8}); len(got) != 15 {
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

	// No gauge (capacity 0): slots segment absent, throughput still present.
	frame = strings.Join(renderBoard(mkState(rows), boardEnv{width: 100, now: at(21176)}), "\n")
	if strings.Contains(frame, "slots") {
		t.Errorf("no gauge → no slots segment:\n%s", frame)
	}
	if !strings.Contains(frame, "inner-CV runs/min") {
		t.Errorf("throughput must survive a missing gauge:\n%s", frame)
	}

	// Rate unavailable (fresh ring): "— inner-CV runs/min".
	st = mkState(rows)
	st.rate = movingRate{}
	frame = strings.Join(renderBoard(st, boardEnv{width: 100, now: at(0), busy: 1, capacity: 8}), "\n")
	if !strings.Contains(frame, "— inner-CV runs/min") {
		t.Errorf("unavailable rate renders —:\n%s", frame)
	}
}

func TestRenderBoardStartupObservationsAreFactual(t *testing.T) {
	now := at(5000)
	bs := boardState{st: progressState{
		nested: true, foldTotal: 10, foldKind: sampler.SizeExact,
		stepK: 37, lastStepAt: at(4000),
	}}
	frame := strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 8, capacity: 12}), "\n")
	for _, want := range []string{
		"starting", "~slots 8/12", "37 steps completed", "last step 1s ago", "no inner-CV run complete",
	} {
		if !strings.Contains(frame, want) {
			t.Fatalf("startup frame missing %q:\n%s", want, frame)
		}
	}
	for _, forbidden := range []string{"not hung", "warming"} {
		if strings.Contains(frame, forbidden) {
			t.Fatalf("startup frame made diagnosis %q:\n%s", forbidden, frame)
		}
	}

	bs.st.foldK = 1
	bs.st.lastRunAt = now
	frame = strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 8, capacity: 12}), "\n")
	if strings.Contains(frame, "starting") || strings.Contains(frame, "no inner-CV run complete") {
		t.Fatalf("startup line must disappear after first eligible run:\n%s", frame)
	}
}

func TestRenderBoardFlatStartupAndConfidenceLabels(t *testing.T) {
	now := at(30000)
	bs := boardState{st: progressState{
		foldTotal: 20, foldKind: sampler.SizeExact,
		stepK: 4, lastStepAt: at(28000),
	}}
	frame := strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 3, capacity: 8}), "\n")
	for _, want := range []string{
		"CV runs 0/20", "starting", "~slots 3/8", "4 steps completed", "last step 2s ago", "no CV run complete",
	} {
		if !strings.Contains(frame, want) {
			t.Fatalf("flat startup frame missing %q:\n%s", want, frame)
		}
	}
	if strings.Contains(frame, "inner-CV") || strings.Contains(frame, "warming") || strings.Contains(frame, "not hung") {
		t.Fatalf("flat startup frame contains nested or diagnostic wording:\n%s", frame)
	}

	bs.st.foldK = 1
	bs.st.lastRunAt = at(29000)
	frame = strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 3, capacity: 8}), "\n")
	for _, want := range []string{"CV runs 1/20", "last CV run 1s ago", "— CV runs/min"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("flat pre-confidence frame missing %q:\n%s", want, frame)
		}
	}
	if strings.Contains(frame, "starting") || strings.Contains(frame, "ETA") {
		t.Fatalf("flat pre-confidence frame must be post-startup without ETA:\n%s", frame)
	}
}

func TestRenderBoardMatureShowsLastRunAge(t *testing.T) {
	now := at(20000)
	var rate movingRate
	for i := 0; i < 16; i++ {
		rate.add(at(i * 1000))
	}
	bs := boardState{
		st: progressState{
			nested: true, foldK: 16, foldTotal: 32, foldKind: sampler.SizeExact,
			lastRunAt: at(15000),
		},
		rate: rate,
	}
	frame := strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 4, capacity: 8}), "\n")
	for _, want := range []string{"45.0 inner-CV runs/min", "last inner-CV run 5s ago", "~ETA"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("mature frame missing %q:\n%s", want, frame)
		}
	}
}

func TestRenderBoardMatureSilenceAdvancesAgeAndDecaysEstimate(t *testing.T) {
	var rate movingRate
	for i := 0; i < 16; i++ {
		rate.add(at(i * 1000))
	}
	bs := boardState{
		st: progressState{
			nested: true, foldK: 16, foldTotal: 32, foldKind: sampler.SizeExact,
			lastRunAt: at(15000),
		},
		rate: rate,
	}
	var prevRate float64
	var prevETA time.Duration
	for sec := 20; sec <= 25; sec++ {
		now := at(sec * 1000)
		perMin, ok := bs.rate.rate(now)
		if !ok {
			t.Fatalf("rate unavailable at t=%ds", sec)
		}
		eta, ok := bs.rate.eta(now, bs.st.foldTotal-bs.st.foldK)
		if !ok {
			t.Fatalf("ETA unavailable at t=%ds", sec)
		}
		frame := strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 4, capacity: 8}), "\n")
		wantAge := fmt.Sprintf("last inner-CV run %ds ago", sec-15)
		if !strings.Contains(frame, wantAge) || !strings.Contains(frame, "~ETA") {
			t.Fatalf("mature silence frame at t=%ds missing age/ETA:\n%s", sec, frame)
		}
		if sec > 20 {
			if perMin > prevRate {
				t.Fatalf("rate increased during silence at t=%ds: %f > %f", sec, perMin, prevRate)
			}
			if eta < prevETA {
				t.Fatalf("ETA decreased during silence at t=%ds: %v < %v", sec, eta, prevETA)
			}
		}
		prevRate, prevETA = perMin, eta
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

// steppingClock advances a fixed step per reading — every budgeted operation sees
// the flush budget elapsed, preserving the pre-#46 inline-flush semantics in tests
// that assert immediate rendering.
func steppingClock(step time.Duration) func() time.Time {
	var mu sync.Mutex // runOpts.now is called from concurrent ParExec goroutines
	t := at(0)
	return func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		t = t.Add(step)
		return t
	}
}

// boardWriter pins the board to the bottom: passthrough writes scroll above the
// stored frame; erase sequences separate frames; close is idempotent and restores
// the cursor. Driven directly (no ticker) against a bytes.Buffer "terminal".
// (metis#46: a stepping clock keeps each write on the quiet inline-flush path.)
func TestBoardWriter_PinBottom(t *testing.T) {
	var term strings.Builder
	bw := newBoardWriter(&term, steppingClock(300*time.Millisecond), false)

	bw.paint([]string{"AGG line", "fold 0 ▸"}, 0)
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
	bw.paint([]string{"AGG line", "fold 0 ✓"}, 0)
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
	bw := newBoardWriter(&term, steppingClock(300*time.Millisecond), false)
	bw.paint([]string{"B"}, 0)
	bw.Write([]byte("tail-no-newline"))
	bw.close()
	if !strings.Contains(term.String(), "tail-no-newline\n") {
		t.Errorf("held tail must flush at close: %q", term.String())
	}
}

func TestBoardWriter_DiscardFrameErasesWithoutRedraw(t *testing.T) {
	var term strings.Builder
	bw := newBoardWriter(&term, steppingClock(300*time.Millisecond), false)
	bw.paint([]string{"folds 2/8", "31.2 folds/min · ETA 12s"}, 0)
	offset := term.Len()

	bw.discardFrame()
	bw.close()
	suffix := term.String()[offset:]
	if !strings.Contains(suffix, "\x1b[2A\x1b[J") {
		t.Fatalf("discard must erase the painted two-line frame: %q", suffix)
	}
	for _, stale := range []string{"folds 2/8", "folds/min", "ETA"} {
		if strings.Contains(suffix, stale) {
			t.Errorf("discard/close redrew stale token %q: %q", stale, suffix)
		}
	}
	if !strings.HasSuffix(suffix, "\x1b[?25h") {
		t.Errorf("close after discard must restore the cursor: %q", suffix)
	}
	n := term.Len()
	bw.close()
	if term.Len() != n {
		t.Fatal("close after discard must remain idempotent")
	}
}

// Board mode end-to-end over the fixture sweep: frames paint (cursor hide, fold rows),
// the #30 plain lines are REPLACED (not duplicated), the final frame carries the
// completed counts, and a capture warning — the plan-review bypass route (o.out) —
// lands ABOVE the board through the compositor, never after the last erase.
func TestRunExperiment_BoardMode(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b]"))
	t.Setenv("NO_COLOR", "") // metis#55: empty = color ON (no-color.org: only non-empty disables) — deterministic banding
	var term strings.Builder
	_, err := runExperiment(runOpts{
		expPath: expPath,
		now:     steppingClock(300 * time.Millisecond), // budget elapses every reading (#46)
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
	// metis#55: the RESULT (estimate + #50 summary) flushes AFTER cursor restore — the
	// terminal ends on the paste-ready commands, not the board. Restore precedes it.
	restoreIdx := strings.LastIndex(s, "\x1b[?25h")
	sumIdx := strings.LastIndex(s, "metis: nested-CV estimate")
	doneIdx := strings.LastIndex(s, "metis: done in")
	if restoreIdx < 0 || sumIdx < restoreIdx || doneIdx < sumIdx {
		t.Errorf("want cursor-restore THEN estimate THEN summary last (restore=%d est=%d done=%d)", restoreIdx, sumIdx, doneIdx)
	}
	if !strings.HasSuffix(strings.TrimRight(s, "\n"), "# cohorts") {
		t.Errorf("output must END with the summary's next-hints: %q", s[max(0, len(s)-60):])
	}
	if strings.Contains(s, "metis: progress") {
		t.Errorf("the board REPLACES the plain progress lines")
	}
	if !strings.Contains(s, "outer folds 2/2") || !strings.Contains(s, "outer fold 0 \x1b[32m✓") || !strings.Contains(s, "outer fold 1 \x1b[32m✓") {
		t.Errorf("the final frame must show completed folds (green ticks — color on, no NO_COLOR in this env):\n%s", s)
	}
	// metis#55/#56: the banding — dim separator rule + bold aggregate; the status line stays DEFAULT.
	if !strings.Contains(s, "\x1b[2m──") || !strings.Contains(s, "\x1b[1mouter folds") {
		t.Errorf("board banding missing (separator/bold aggregate):\n%q", s[:min(len(s), 300)])
	}
	if strings.Contains(s, sgrDim+"~slots") || strings.Contains(s, sgrDim+"starting") {
		t.Error("metis#56: the status line must stay DEFAULT color (regression pin)")
	}
	// metis#56: a closing rule bands the footer off the RESULT — present between the
	// cursor restore and the estimate.
	if restoreIdx >= 0 && sumIdx > restoreIdx && !strings.Contains(s[restoreIdx:sumIdx], "──") {
		t.Error("metis#56: closing rule missing between footer and result")
	}
	// The bypass route: the fake-exec fixture has no traced closure → captureSweepCode
	// notes "no first-party code closure" via o.out — which after the runExperiment
	// reorder IS the compositor, so the text must appear before the final erase, never
	// as a bare trailing write (the plan-review o.out bypass, pinned).
	warnIdx := strings.Index(s, "no first-party code closure")
	if warnIdx < 0 {
		t.Fatalf("expected the uncaptured-code note in a fake-exec fixture:\n%s", s)
	}
	if finalFrame := strings.LastIndex(s, "outer folds 2/2"); warnIdx > finalFrame {
		t.Errorf("the capture warning bypassed the compositor (after the final frame)")
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

func TestRunExperiment_BoardFailureRejectsPostPublicationTickAndDiscardsFrame(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b, c]"))
	control := newRunControl(2)
	exec := newFailureBarrierExec()
	out := &concurrentBuffer{}
	boardTick := make(chan time.Time)
	tickSelected := make(chan struct{}, 2)
	tickFinished := make(chan struct{}, 2)
	publishedOffset := make(chan int, 1)
	postFailureTickSend := make(chan error, 1)
	control.beforeFailureUnlock = func() {
		publishedOffset <- out.len()
		postFailureTickSend <- sendBoardTickWithin(boardTick, at(2000), "post-publication tick receive")
		close(exec.failurePublished)
	}
	result := make(chan error, 1)
	go func() {
		_, err := runExperiment(runOpts{
			expPath: expPath, now: fixedNow(),
			git: fakeGitProbe{name: "metis", sha: "sha"}, exec: exec, out: out,
			maxParallel: 2, runControl: control, tui: true, boardTick: boardTick,
			beforeBoardTick: func() { tickSelected <- struct{}{} },
			afterBoardTick:  func() { tickFinished <- struct{}{} },
		})
		result <- err
	}()
	for i := 0; i < 4; i++ {
		awaitRunControl(t, exec.innerEntered, "four board-mode inner run directories")
	}

	if err := sendBoardTickWithin(boardTick, at(1000), "pre-failure tick receive"); err != nil {
		t.Fatal(err)
	}
	awaitRunControl(t, tickSelected, "pre-failure board tick selection")
	awaitRunControl(t, tickFinished, "pre-failure board tick completion")
	preFailure := out.snapshot()
	for _, want := range []string{"outer folds 0/2", "outer fold 0 — queued", "no inner-CV run"} {
		if !strings.Contains(preFailure, want) {
			t.Fatalf("pre-failure board missing %q:\n%s", want, preFailure)
		}
	}

	close(exec.releaseFailure)
	offset := awaitRunControl(t, publishedOffset, "board failure publication offset")
	if err := awaitRunControl(t, postFailureTickSend, "post-publication tick send result"); err != nil {
		awaitRunControl(t, result, "board-mode failure cleanup after tick-send timeout")
		t.Fatal(err)
	}
	awaitRunControl(t, tickSelected, "post-publication board tick selection")
	awaitRunControl(t, tickFinished, "rejected post-publication board tick")
	err := awaitRunControl(t, result, "board-mode failure cleanup")
	if err == nil || !strings.Contains(err.Error(), "injected train failure") {
		t.Fatalf("board-mode error = %v, want injected train failure", err)
	}
	suffix := out.snapshot()[offset:]
	for _, forbidden := range []string{
		"outer folds 0/2", "outer fold 0 — queued", "configs ", "inner-CV runs ", "inner-CV runs/min", "ETA", "score ", "estimate", "mean ",
	} {
		if strings.Contains(suffix, forbidden) {
			t.Errorf("post-publication board output contains stale token %q:\n%q", forbidden, suffix)
		}
	}
	if !strings.Contains(suffix, "\x1b[J") || !strings.HasSuffix(suffix, "\x1b[?25h") {
		t.Errorf("failure cleanup must erase the board and restore the cursor: %q", suffix)
	}
}

func sendBoardTickWithin(ch chan<- time.Time, tick time.Time, what string) error {
	timer := time.NewTimer(runControlTestTimeout)
	defer timer.Stop()
	select {
	case ch <- tick:
		return nil
	case <-timer.C:
		return fmt.Errorf("timed out waiting for %s", what)
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
	bw := newBoardWriter(&term, steppingClock(300*time.Millisecond), false)
	bw.paint([]string{"BOARD"}, 0)
	pool := newServerPool(bw, nil) // what runExperiment does post-reorder: pool captures the compositor
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
	if !strings.HasSuffix(s, "BOARD\n\x1b[?2026l") { // frame last, then the sync-end (metis#47)
		t.Errorf("the board must be repainted below the notice: %q", s)
	}
}

// metis#46: under a warm-cache burst the compositor COALESCES — one atomic
// erase+dump+repaint per flush budget (~4Hz), never per line. Idealized emulators
// tolerate per-line repaints; real terminals and mux layers (the operator's
// ghostty-in-cmux) tear under hundreds of erase cycles per second.
func TestBoardWriter_BurstCoalesces(t *testing.T) {
	var term strings.Builder
	// Scripted clock: construction, then one reading per Write; 100 writes 5ms apart
	// (a 500ms burst) → with a 250ms budget only ~3 inline flushes may fire.
	times := []time.Time{at(0)}
	for i := 1; i <= 100; i++ {
		times = append(times, at(i*5))
	}
	bw := newBoardWriter(&term, scriptedClock(times...), false)
	bw.paint([]string{"AGG", "fold 0 ▸"}, 0)
	for i := 0; i < 100; i++ {
		bw.Write([]byte(fmt.Sprintf("⚡ step %d (cache hit)\n", i)))
	}
	bw.close()
	s := term.String()
	erases := strings.Count(s, "\x1b[J")
	if erases > 5 { // budgeted: ~500ms/250ms + paint + close — NOT ~100
		t.Errorf("burst must coalesce to ≤5 erase cycles, got %d", erases)
	}
	// Every passthrough byte still lands, in order.
	for _, want := range []string{"step 0 ", "step 50 ", "step 99 "} {
		if !strings.Contains(s, want) {
			t.Errorf("coalescing lost passthrough %q", want)
		}
	}
	if strings.Index(s, "step 50 ") < strings.Index(s, "step 0 ") {
		t.Error("passthrough order must be preserved")
	}
	// The final frame survives close.
	if !strings.Contains(s[strings.LastIndex(s, "\x1b[J"):], "AGG") {
		t.Error("close must leave the final frame")
	}
}

// Quiet writes (≥budget apart — a cold run's sparse step lines) flush INLINE: the
// operator watching a slow sweep sees lines the moment they happen.
func TestBoardWriter_QuietWritesFlushInline(t *testing.T) {
	var term strings.Builder
	bw := newBoardWriter(&term, scriptedClock(at(0), at(300), at(600), at(900)), false)
	bw.paint([]string{"B"}, 0)
	pre := term.Len()
	bw.Write([]byte("slow line 1\n"))
	if !strings.Contains(term.String()[pre:], "slow line 1") {
		t.Error("a quiet write must appear immediately")
	}
	pre = term.Len()
	bw.Write([]byte("slow line 2\n"))
	if !strings.Contains(term.String()[pre:], "slow line 2") {
		t.Error("the next quiet write must also flush inline")
	}
}

// A flood that outruns the budget still bounds memory: the pending buffer force-
// flushes at the size cap rather than growing without limit.
func TestBoardWriter_PendingSizeCap(t *testing.T) {
	var term strings.Builder
	bw := newBoardWriter(&term, func() time.Time { return at(0) }, false) // frozen: budget never elapses
	bw.paint([]string{"B"}, 0)
	line := strings.Repeat("x", 1024) + "\n"
	for i := 0; i < 100; i++ { // ~100KB > the 64KB cap
		bw.Write([]byte(line))
	}
	if !strings.Contains(term.String(), "xxxx") {
		t.Error("the size cap must force a flush during a frozen-clock flood")
	}
}

// The tick's forceFlush drains mid-budget pending output and re-pins the board —
// the path that restores the display after a burst window (close-review Important:
// a stranded-pending regression here would ship silently without this pin).
func TestBoardWriter_ForceFlushDrainsPending(t *testing.T) {
	var term strings.Builder
	bw := newBoardWriter(&term, func() time.Time { return at(0) }, false) // frozen: budget never elapses
	bw.paint([]string{"BOARD"}, 0)                                    // first flush (zero lastFlush) paints; budget now frozen shut
	pre := term.Len()
	bw.Write([]byte("mid-budget line\n"))
	if strings.Contains(term.String()[pre:], "mid-budget") {
		t.Fatal("a mid-budget write must coalesce, not flush")
	}
	bw.forceFlush() // what sp.tick() calls
	s := term.String()[pre:]
	if !strings.Contains(s, "mid-budget line\n") {
		t.Errorf("forceFlush must drain the pending line: %q", s)
	}
	if !strings.HasSuffix(term.String(), "BOARD\n\x1b[?2026l") { // metis#47 sync-end trails
		t.Errorf("forceFlush must re-pin the board below the drained output: %q", s)
	}
	// And through the sink: tick() stores a fresh frame then force-flushes.
	var term2 strings.Builder
	prog := newSweepProgress(&term2, func() time.Time { return at(0) }, "maximize",
		progressTotals{nested: true, outer: 1, outerKind: sampler.SizeExact})
	bw2 := newBoardWriter(&term2, func() time.Time { return at(0) }, false)
	prog.bw, prog.width = bw2, 100
	prog.tick()
	if !strings.Contains(term2.String(), "outer folds 0/1") {
		t.Errorf("tick must render + force-paint the current frame: %q", term2.String())
	}
}

// metis#47: every flushed update is bracketed in DEC 2026 synchronized output
// (\x1b[?2026h … \x1b[?2026l) so supporting terminals (ghostty, iTerm2, kitty) apply
// the erase+redraw atomically — no flash; others ignore the private mode (safe no-op).
func TestBoardWriter_SynchronizedOutputBrackets(t *testing.T) {
	var term strings.Builder
	bw := newBoardWriter(&term, steppingClock(300*time.Millisecond), false)
	bw.paint([]string{"AGG", "fold 0 ▸"}, 0)
	bw.Write([]byte("⚡ step x\n"))
	bw.forceFlush()
	bw.close()
	s := term.String()
	bsu, esu := strings.Count(s, "\x1b[?2026h"), strings.Count(s, "\x1b[?2026l")
	if bsu == 0 || bsu != esu {
		t.Fatalf("flushes must be BSU/ESU-bracketed and balanced: h=%d l=%d\n%q", bsu, esu, s)
	}
	// Balanced nesting: never two opens without a close between them.
	depth := 0
	for i := 0; i+8 <= len(s); i++ {
		switch s[i : i+8] {
		case "\x1b[?2026h":
			depth++
			if depth > 1 {
				t.Fatal("nested BSU without ESU")
			}
		case "\x1b[?2026l":
			depth--
			if depth < 0 {
				t.Fatal("ESU without BSU")
			}
		}
	}
	// Every erase cycle happens INSIDE a bracket (no unsynchronized erase remains).
	for i := strings.Index(s, "\x1b[J"); i >= 0; {
		before := s[:i]
		if strings.Count(before, "\x1b[?2026h") <= strings.Count(before, "\x1b[?2026l") {
			t.Fatalf("an erase at byte %d is outside a synchronized bracket", i)
		}
		j := strings.Index(s[i+1:], "\x1b[J")
		if j < 0 {
			break
		}
		i += 1 + j
	}
}

// metis#55: NO_COLOR (color=false) paints the separator PLAIN — zero SGR styling bytes
// anywhere (cursor/erase control sequences are not styling and remain).
func TestBoardWriter_NoColorHasNoSGR(t *testing.T) {
	var term strings.Builder
	bw := newBoardWriter(&term, steppingClock(300*time.Millisecond), false)
	bw.paint([]string{"AGG", "fold 0 ✓ held", "~slots"}, 40)
	bw.close()
	s := term.String()
	for _, sgr := range []string{sgrBold, sgrDim, sgrGreen, sgrYellow} {
		if strings.Contains(s, sgr) {
			t.Errorf("color=false must emit no SGR %q:\n%q", sgr, s)
		}
	}
	if !strings.Contains(s, strings.Repeat("─", 40)) {
		t.Error("the separator rule paints (plain) even with color off")
	}
}

// metis#55: the epilogue flushes once, AFTER the final frame + cursor restore; the erase
// count includes the separator (no ghost line on the next erase).
func TestBoardWriter_EpilogueAfterFinalFrame(t *testing.T) {
	var term strings.Builder
	bw := newBoardWriter(&term, steppingClock(300*time.Millisecond), true)
	bw.paint([]string{"AGG", "row"}, 20)
	fmt.Fprintln(bw.epilogueWriter(), "RESULT: est 0.84")
	bw.paint([]string{"AGG2", "row2"}, 20) // repaint after epilogue registered — erase must count separator
	bw.close()
	s := term.String()
	resIdx := strings.LastIndex(s, "RESULT: est 0.84")
	frameIdx := strings.LastIndex(s, "AGG2")
	restoreIdx := strings.LastIndex(s, "\x1b[?25h")
	if frameIdx < 0 || resIdx < frameIdx || (restoreIdx > 0 && resIdx < restoreIdx) {
		t.Errorf("epilogue must print after the final frame + restore (frame=%d restore=%d result=%d):\n%q", frameIdx, restoreIdx, resIdx, s)
	}
	if strings.Contains(s[:resIdx], "RESULT") {
		t.Error("epilogue must not leak into the scroll region before close")
	}
	// erase math: each erase's cursor-up count must equal lines painted (frame+separator=3)
	if strings.Contains(s, "\x1b[2A\x1b[J") {
		t.Error("erase used 2-line count — the separator line was not counted (ghost line)")
	}
}
