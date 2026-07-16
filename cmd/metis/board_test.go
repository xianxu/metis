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
		34 * time.Second:                 "34s",
		190 * time.Second:                "3m10s",
		2*time.Hour + 5*time.Minute:      "2h5m",
		2*time.Hour + 5*time.Minute + 9*time.Second: "2h5m",
	}
	for d, want := range cases {
		if got := fmtETA(d); got != want {
			t.Errorf("fmtETA(%v) = %q, want %q", d, got, want)
		}
	}
}
