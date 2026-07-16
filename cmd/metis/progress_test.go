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
			[]string{"outer 0/3", "configs 84/216", "folds 421/1080", "est —"}, []string{"±"}},
		{"nested one outer", nested(1, 100, 500, []float64{0.82}),
			[]string{"outer 1/3", "est 0.8200"}, []string{"±"}},
		{"nested two outer", nested(2, 200, 900, []float64{0.80, 0.84}),
			[]string{"outer 2/3", "est 0.8200 ± 0.0200"}, nil},
		{"flat one config", progressState{
			foldK: 3, foldTotal: 5, foldKind: sampler.SizeExact,
			flatScores: []float64{0.80, 0.84, 0.88}},
			[]string{"folds 3/5", "score 0.8400"}, []string{"configs", "outer"}},
		{"unknown kind", progressState{
			nested: true,
			outerK: 1, outerTotal: 0, outerKind: sampler.SizeUnknown,
			configK: 3, configTotal: 0, configKind: sampler.SizeUnknown},
			[]string{"outer 1/?", "configs 3/?"}, nil},
		{"budget kind", progressState{
			nested: true,
			outerK: 1, outerTotal: 8, outerKind: sampler.SizeBudget},
			[]string{"outer 1/≤8"}, nil},
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
	prog := newSweepProgress(&out, scriptedClock(times...),
		progressTotals{nested: true, outer: 2, outerKind: sampler.SizeExact,
			configs: 4, configKind: sampler.SizeExact, folds: 20, foldKind: sampler.SizeExact})
	hooks := prog.forPass(0)
	for i := 1; i <= 10; i++ {
		hooks.fold(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]{K: i, Total: 5, Kind: sampler.SizeExact,
			Out: sampler.FoldOutcome{Score: 0.8}})
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
	for _, w := range []string{"outer 1/2", "folds 10/20", "est 0.8300"} {
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
	prog := newSweepProgress(safeOut, func() time.Time { return at(0) },
		progressTotals{nested: true, folds: 64, foldKind: sampler.SizeExact})
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		hooks := prog.forPass(g)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 8; i++ {
				hooks.fold(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]{K: i + 1, Out: sampler.FoldOutcome{Score: 0.5}})
			}
		}()
	}
	wg.Wait()
	prog.finish()
	if !strings.Contains(out.String(), "folds 64/64") {
		t.Errorf("sink-owned counter must see all 64 events (never ev.K):\n%s", out.String())
	}
}

// writerFunc adapts a func to io.Writer for the concurrency test.
type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }
