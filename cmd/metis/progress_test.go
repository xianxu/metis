package main

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/sampler"
	"github.com/xianxu/metis/pkg/shape"
)

// progressLine renders the aggregated one-line view of a sweep's progressState.
func TestProgressLine(t *testing.T) {
	nested := func(outerK, configK, foldK int, scores []float64) progressState {
		return progressState{
			nested: true,
			outerK: outerK, outerTotal: 3, outerKind: sampler.SizeExact,
			configK: configK, configTotal: 216, configKind: sampler.SizeExact,
			foldK: foldK, foldTotal: 1080, foldKind: sampler.SizeExact,
			outerScores: scores,
		}
	}
	cases := []struct {
		name string
		st   progressState
		want []string
		not  []string
	}{
		{"nested pre-outer", nested(0, 84, 421, nil),
			[]string{"outer folds 0/3", "configs scored 84/216", "inner-CV runs 421/1080", "est —"}, []string{"±"}},
		{"nested one outer", nested(1, 100, 500, []float64{0.82}),
			[]string{"outer folds 1/3", "est 0.8200"}, []string{"±"}},
		{"nested two outer", nested(2, 200, 900, []float64{0.80, 0.84}),
			[]string{"outer folds 2/3", "est 0.8200 ± 0.0200"}, nil},
		{"flat one config", progressState{
			foldK: 3, foldTotal: 5, foldKind: sampler.SizeExact,
			flatScores: []float64{0.80, 0.84, 0.88}},
			[]string{"CV runs 3/5", "score 0.8400"}, []string{"configs", "outer", "folds 3/5"}},
		{"unknown kind", progressState{
			nested: true,
			outerK: 1, outerTotal: 0, outerKind: sampler.SizeUnknown,
			configK: 3, configTotal: 0, configKind: sampler.SizeUnknown},
			[]string{"outer folds 1/?", "configs scored 3/?"}, nil},
		{"budget kind", progressState{
			nested: true,
			outerK: 1, outerTotal: 8, outerKind: sampler.SizeBudget},
			[]string{"outer folds 1/≤8"}, nil},
	}
	for _, c := range cases {
		got := progressLine(c.st)
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: missing %q in %q", c.name, w, got)
			}
		}
		for _, n := range c.not {
			if strings.Contains(got, n) {
				t.Errorf("%s: unwanted %q in %q", c.name, n, got)
			}
		}
	}
}

// scriptedClock returns a now() that steps through the given instants (sticky last).
func scriptedClock(times ...time.Time) func() time.Time {
	i := 0
	return func() time.Time {
		t := times[min(i, len(times)-1)]
		i++
		return t
	}
}

func at(ms int) time.Time {
	return time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC).Add(time.Duration(ms) * time.Millisecond)
}

// The sink throttles fold-level emits to 1/s, ALWAYS emits on an outer (driver)
// completion, and emits a final line at finish().
func TestSweepProgress_ThrottleAndAlwaysEmit(t *testing.T) {
	var out strings.Builder
	// Clock: one reading per event. 10 fold events at 200ms spacing starting at t=0.
	// Emits: event 1 (the FIRST event always emits — started=false) and event 6
	// (t=1000, the first ≥1s after t=0). Events 2-5 and 7-10 are throttled.
	times := []time.Time{at(0)}
	for i := 1; i <= 10; i++ {
		times = append(times, at(i*200))
	}
	times = append(times, at(2100), at(2200)) // driver event, finish
	prog := newSweepProgress(&out, scriptedClock(times...), "maximize",
		progressTotals{nested: true, outer: 2, outerKind: sampler.SizeExact,
			configs: 4, configKind: sampler.SizeExact, folds: 20, foldKind: sampler.SizeExact})
	for i := 1; i <= 10; i++ {
		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 200)})
	}
	throttled := strings.Count(out.String(), "metis: progress")
	if throttled != 2 { // event 1 (first) + event 6 (throttle boundary)
		t.Fatalf("want 2 throttled emits, got %d:\n%s", throttled, out.String())
	}
	// A driver-level completion always emits, regardless of throttle.
	prog.driverEvent(sampler.ProgressEvent[sampler.OuterFoldPoint, float64]{K: 1, Total: 2, Kind: sampler.SizeExact, Out: 0.83})
	if got := strings.Count(out.String(), "metis: progress"); got != 3 {
		t.Fatalf("driver completion must always emit, got %d lines", got)
	}
	prog.finish()
	if got := strings.Count(out.String(), "metis: progress"); got != 4 {
		t.Fatalf("finish must emit the final line, got %d lines", got)
	}
	final := out.String()[strings.LastIndex(out.String(), "metis: progress"):]
	for _, w := range []string{"outer folds 1/2", "inner-CV runs 10/20", "est 0.8300"} {
		if !strings.Contains(final, w) {
			t.Errorf("final line missing %q: %q", w, final)
		}
	}
	if strings.ContainsAny(out.String(), "\x1b\r") {
		t.Error("plain lines must carry no escape codes / carriage returns")
	}
}

// A nil sink is a no-op everywhere (the non-sweep path stays silent).
func TestSweepProgress_NilSafe(t *testing.T) {
	var prog *sweepProgress
	hooks := prog.forPass(0)
	hooks.fold(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]{K: 1})
	hooks.config(sampler.ProgressEvent[shape.Point, sampler.MeanSE]{K: 1})
	prog.driverEvent(sampler.ProgressEvent[sampler.OuterFoldPoint, float64]{K: 1})
	prog.finish() // must not panic
}

// Concurrent event fire from many goroutines is race-clean and loses no counts.
func TestSweepProgress_ConcurrentCounts(t *testing.T) {
	var out strings.Builder
	var mu sync.Mutex
	safeOut := writerFunc(func(p []byte) (int, error) { mu.Lock(); defer mu.Unlock(); return out.Write(p) })
	prog := newSweepProgress(safeOut, func() time.Time { return at(0) }, "maximize",
		progressTotals{nested: true, folds: 64, foldKind: sampler.SizeExact})
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 8; i++ {
				prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 1000)})
			}
		}()
	}
	wg.Wait()
	prog.finish()
	if !strings.Contains(out.String(), "inner-CV runs 64/64") {
		t.Errorf("sink-owned counter must see all 64 events (never ev.K):\n%s", out.String())
	}
}

// writerFunc adapts a func to io.Writer for the concurrency test.
type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

// movingRate: keep the latest 64 eligible completion times by event time. It is
// ready only after enough event-time evidence, and rate = (n-1)/(now-oldest).
func TestMovingRate(t *testing.T) {
	var r movingRate
	if _, ok := r.rate(at(0)); ok {
		t.Error("n=0 must be not-ok")
	}
	for i := 0; i < 15; i++ {
		r.add(at(i * 1000))
	}
	if _, ok := r.rate(at(15000)); ok {
		t.Error("15 completions must be below confidence")
	}
	var short movingRate
	for i := 0; i < 16; i++ {
		short.add(at(i * 900))
	}
	if _, ok := short.rate(at(15000)); ok {
		t.Error("16 completions spanning under 15s must be below confidence")
	}
	var ready movingRate
	for i := 0; i < 16; i++ {
		ready.add(at(i * 1000))
	}
	if got, ok := ready.rate(at(20000)); !ok || got < 44.9 || got > 45.1 {
		t.Errorf("16+ completions spanning ≥15s at now=20s → 45/min, got %v ok=%v", got, ok)
	}
	if got, ok := ready.rate(at(25000)); !ok || got >= 45 {
		t.Errorf("stall must decay the rate using now in the denominator, got %v ok=%v", got, ok)
	}

	// Reversed delivery: 65 completions 1s apart keeps the newest 64 by event time.
	var r2 movingRate
	for i := 64; i >= 0; i-- {
		r2.add(at(i * 1000))
	}
	// newest kept window = t=1s..64s; at now=65s: 63 completions over 64s ≈ 59.06/min.
	if got, ok := r2.rate(at(65000)); !ok || got < 58.9 || got > 59.2 {
		t.Errorf("reversed delivery latest-64 window: want ~59.06/min, got %v ok=%v", got, ok)
	}
	// ETA: remaining/rate.
	if eta, ok := ready.eta(at(20000), 45); !ok || eta != time.Minute {
		t.Errorf("eta 45 remaining at 45/min → 1m, got %v ok=%v", eta, ok)
	}
}

func TestOccupancyWindowRoundedMeanOfLastFourSamples(t *testing.T) {
	var w occupancyWindow
	for _, busy := range []int{1, 2, 3, 4} {
		w.add(busy, 12)
	}
	busy, capacity, ok := w.mean()
	if !ok || busy != 3 || capacity != 12 {
		t.Fatalf("[1,2,3,4]/12 mean = (%d,%d,%v); want (3,12,true)", busy, capacity, ok)
	}
	w.add(9, 12)
	busy, capacity, ok = w.mean()
	if !ok || busy != 5 || capacity != 12 {
		t.Fatalf("[2,3,4,9]/12 rounded mean = (%d,%d,%v); want (5,12,true)", busy, capacity, ok)
	}
}

func TestSweepProgressOccupancySamplesOnlyOnTick(t *testing.T) {
	var out strings.Builder
	busy := 0
	prog := newSweepProgress(&out, func() time.Time { return at(20000) }, "maximize",
		progressTotals{nested: true, folds: 32, foldKind: sampler.SizeExact})
	prog.bw = newBoardWriter(&out, func() time.Time { return at(20000) })
	prog.gauge = func() (int, int) { return busy, 12 }

	for _, v := range []int{1, 2, 3, 4} {
		busy = v
		prog.tick()
	}
	got, cap, ok := prog.occupancy.mean()
	if !ok || got != 3 || cap != 12 {
		t.Fatalf("tick samples [1,2,3,4] mean = (%d,%d,%v); want (3,12,true)", got, cap, ok)
	}

	for i := 0; i < 10; i++ {
		busy = 12
		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 1000)})
	}
	got, cap, ok = prog.occupancy.mean()
	if !ok || got != 3 || cap != 12 {
		t.Fatalf("activity burst changed occupancy mean to (%d,%d,%v); want unchanged (3,12,true)", got, cap, ok)
	}

	busy = 5
	prog.tick()
	got, cap, ok = prog.occupancy.mean()
	if !ok || got != 4 || cap != 12 {
		t.Fatalf("fifth tick samples [2,3,4,5] mean = (%d,%d,%v); want (4,12,true)", got, cap, ok)
	}
}

// Per-pass rows: each forPass(i) hook folds into ITS row (closure-bound identity);
// driverEvent's Point.Idx collapses the right row to its held-out score.
func TestSweepProgress_PerPassRows(t *testing.T) {
	var out strings.Builder
	prog := newSweepProgress(&out, func() time.Time { return at(0) }, "maximize",
		progressTotals{nested: true, outer: 2, outerKind: sampler.SizeExact,
			configs: 4, configKind: sampler.SizeExact, folds: 12, foldKind: sampler.SizeExact})
	h0, h1 := prog.forPass(0), prog.forPass(1)
	fev := func(score float64) sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome] {
		return sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]{Out: sampler.FoldOutcome{Score: score}}
	}
	cev := func(mean float64) sampler.ProgressEvent[shape.Point, sampler.MeanSE] {
		return sampler.ProgressEvent[shape.Point, sampler.MeanSE]{Out: sampler.MeanSE{Mean: mean}}
	}
	h0.fold(fev(0.7))
	h0.fold(fev(0.7))
	h1.fold(fev(0.8))
	h0.config(cev(0.70))
	h0.config(cev(0.75)) // pass 0's best (maximize)
	h1.config(cev(0.85))
	h1.config(cev(0.80))
	st := prog.boardState()
	if len(st.rows) != 2 {
		t.Fatalf("want 2 pass rows, got %+v", st.rows)
	}
	if r := st.rows[0]; r.foldK != 2 || r.configK != 2 || !r.hasBest || r.best != 0.75 || r.done {
		t.Errorf("row 0: %+v", r)
	}
	if r := st.rows[1]; r.foldK != 1 || r.configK != 2 || r.best != 0.85 || r.done {
		t.Errorf("row 1 (maximize keeps 0.85): %+v", r)
	}
	// Minimize direction flips the incumbent.
	prog2 := newSweepProgress(&out, func() time.Time { return at(0) }, "minimize",
		progressTotals{nested: true, outer: 1})
	h := prog2.forPass(0)
	h.config(cev(0.5))
	h.config(cev(0.3))
	h.config(cev(0.4))
	if r := prog2.boardState().rows[0]; r.best != 0.3 {
		t.Errorf("minimize keeps 0.3: %+v", r)
	}
	// driverEvent collapses row 1 by its Point.Idx; row 0 stays in-flight.
	prog.driverEvent(sampler.ProgressEvent[sampler.OuterFoldPoint, float64]{
		K: 1, Total: 2, Kind: sampler.SizeExact, Point: sampler.OuterFoldPoint{Idx: 1}, Out: 0.83})
	st = prog.boardState()
	if r := st.rows[1]; !r.done || r.heldOut != 0.83 {
		t.Errorf("row 1 must collapse to held-out 0.83: %+v", r)
	}
	if st.rows[0].done {
		t.Errorf("row 0 must stay in-flight: %+v", st.rows[0])
	}
}

func TestSweepProgressActivityRunEventsOwnAggregateRunCounterAndRate(t *testing.T) {
	var out strings.Builder
	prog := newSweepProgress(&out, func() time.Time { return at(20000) }, "maximize",
		progressTotals{nested: true, outer: 1, outerKind: sampler.SizeExact,
			configs: 2, configKind: sampler.SizeExact, folds: 16, foldKind: sampler.SizeExact})
	hooks := prog.forPass(0)
	hooks.fold(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]{
		Out: sampler.FoldOutcome{Score: 0.7},
	})
	st := prog.boardState()
	if st.st.foldK != 0 {
		t.Fatalf("sampler fold callback advanced aggregate run counter to %d; want typed events only", st.st.foldK)
	}
	if st.rows[0].foldK != 1 {
		t.Fatalf("sampler fold callback should retain per-row duties; row = %+v", st.rows[0])
	}

	prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedPreamble, RunID: "pre", At: at(0)})
	if got := prog.boardState().st.foldK; got != 0 {
		t.Fatalf("ineligible preamble advanced aggregate run counter to %d", got)
	}

	for i := 15; i >= 0; i-- {
		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, RunID: "inner", At: at(i * 1000)})
	}
	st = prog.boardState()
	if st.st.foldK != 16 {
		t.Fatalf("typed eligible run events advanced foldK to %d; want 16", st.st.foldK)
	}
	if got, ok := st.rate.rate(at(20000)); !ok || got < 44.9 || got > 45.1 {
		t.Fatalf("typed event-time rate = %v ok=%v; want 45/min", got, ok)
	}
}
