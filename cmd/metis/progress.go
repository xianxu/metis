package main

// metis#30: the sweep progress sink — folds pkg/sampler's per-completion
// ProgressEvents (typed per level) into ONE throttled aggregated line, so a
// 2,000-fold sweep reports live without a per-fold firehose (single-threaded-
// attention budget). Plain appended lines, no escape codes — non-TTY-safe by
// construction; the TTY board is metis#38, which extends this sink behind the
// same per-pass hooks (outer-fold identity rides the forPass closure binding,
// NEVER an event payload field — pkg/sampler stays coordinate-free).

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/metis/pkg/sampler"
	"github.com/xianxu/metis/pkg/shape"
)

// progressTotals seeds the sink with each level's SizeHint AT WIRING TIME —
// stream-learned totals arrive only with a level's first completion (for the
// driver level: the first COMPLETED outer fold, near the end of a parallel run),
// which starves the display of denominators. cmd/metis constructs the samplers,
// so it reads SizeHint directly; SizeHint stays the single source (ARCH-DRY).
type progressTotals struct {
	nested     bool
	outer      int
	outerKind  sampler.SizeKind
	configs    int // aggregate across outer folds (outer × per-pass configs)
	configKind sampler.SizeKind
	folds      int // aggregate leaf count (outer × configs × inner k)
	foldKind   sampler.SizeKind
}

// progressState is the pure render input: sink-owned aggregate counters per level
// (NEVER ev.K — each concurrent Run instance counts its own 1..k), the completed
// outer scores (→ est mean±SE), and the flat path's running fold scores.
type progressState struct {
	nested               bool
	outerK, outerTotal   int
	outerKind            sampler.SizeKind
	configK, configTotal int
	configKind           sampler.SizeKind
	foldK, foldTotal     int
	foldKind             sampler.SizeKind
	stepK                int
	lastStepAt           time.Time
	lastRunAt            time.Time
	outerScores          []float64 // nested: completed outer-fold held-out scores
	flatScores           []float64 // flat: the one config's completed fold scores
}

// progressLine renders the aggregated line. Nested:
// `outer folds 1/3 · configs scored 84/216 · inner-CV runs 421/1080 · est 0.8283 ± 0.0140`
// (est — until an outer fold lands; ± only at n≥2). Flat (since metis#32: iff 1
// config): `CV runs 3/5 · score 0.8400` (the running fold mean — nothing to be
// "best" of). Kinds render k/n (exact), k/≤n (budget), k/? (unknown). Pure.
func progressLine(st progressState) string {
	return "metis: progress " + progressCore(st)
}

// progressCore is the un-prefixed aggregate segment — shared by the plain line and
// the board's first row (extracted so the board never string-strips the prefix;
// a TrimPrefix coupling would silently no-op if the prefix changed — close review).
func progressCore(st progressState) string {
	frac := func(k, total int, kind sampler.SizeKind) string {
		switch kind {
		case sampler.SizeExact:
			return fmt.Sprintf("%d/%d", k, total)
		case sampler.SizeBudget:
			return fmt.Sprintf("%d/≤%d", k, total)
		default:
			return fmt.Sprintf("%d/?", k)
		}
	}
	var parts []string
	if st.nested {
		parts = append(parts, "outer folds "+frac(st.outerK, st.outerTotal, st.outerKind))
		parts = append(parts, "configs scored "+frac(st.configK, st.configTotal, st.configKind))
		parts = append(parts, "inner-CV runs "+frac(st.foldK, st.foldTotal, st.foldKind))
		mean, se, n := meanSE(st.outerScores)
		switch {
		case n == 0:
			parts = append(parts, "est —")
		case n == 1:
			parts = append(parts, fmt.Sprintf("est %.4f", mean))
		default:
			parts = append(parts, fmt.Sprintf("est %.4f ± %.4f", mean, se))
		}
	} else {
		parts = append(parts, "CV runs "+frac(st.foldK, st.foldTotal, st.foldKind))
		if mean, _, n := meanSE(st.flatScores); n > 0 {
			parts = append(parts, fmt.Sprintf("score %.4f", mean))
		}
	}
	return strings.Join(parts, " · ")
}

// meanSE is the display-only mean ± standard-error reduce over completed scores.
// Computed locally: the honest estimate stays pkg/sampler's Aggregate/MeanSE —
// this is presentation, not selection (do not export sampler surface for it).
func meanSE(xs []float64) (mean, se float64, n int) {
	n = len(xs)
	if n == 0 {
		return 0, 0, 0
	}
	for _, x := range xs {
		mean += x
	}
	mean /= float64(n)
	if n < 2 {
		return mean, 0, n
	}
	var ss float64
	for _, x := range xs {
		ss += (x - mean) * (x - mean)
	}
	se = math.Sqrt(ss/float64(n-1)) / math.Sqrt(float64(n))
	return mean, se, n
}

// seededTotals reads each level's SizeHint on its initial state — the SAME source
// the Run loops stamp on events (ARCH-DRY; no shape math re-derived here) — and
// composes the aggregate denominators: configs = outer × per-pass configs,
// folds = outer × configs × inner k (each sealed pass sweeps the full grid).
// Flat (1 config): folds = the single pass's inner k.
func seededTotals(ctx sampler.Ctx, nested bool, runFolds int, configPts []shape.Point, k int) progressTotals {
	grid := sampler.GridConfigs{Points: configPts}
	nConfigs, kindConfigs := grid.SizeHint(grid.Init(ctx))
	foldsSmp := sampler.FixedKFolds{K: k}
	nFolds, kindFolds := foldsSmp.SizeHint(foldsSmp.Init(ctx))
	if !nested {
		return progressTotals{folds: nFolds, foldKind: kindFolds}
	}
	cv := sampler.CVDriver{K: runFolds}
	nOuter, kindOuter := cv.SizeHint(cv.Init(ctx))
	return progressTotals{
		nested: true,
		outer:  nOuter, outerKind: kindOuter,
		configs: nOuter * nConfigs, configKind: kindConfigs,
		folds: nOuter * nConfigs * nFolds, foldKind: kindFolds,
	}
}

// movingRate retains the latest eligible run completions by event time. rate(now)
// = (n-1)/(now-oldest) after the confidence gate; `now` in the denominator means
// a STALL decays live while last-run age remains the sharp freshness signal.
type movingRate struct {
	times []time.Time
}

func (m *movingRate) add(t time.Time) {
	i := sort.Search(len(m.times), func(i int) bool { return !m.times[i].Before(t) })
	m.times = append(m.times, time.Time{})
	copy(m.times[i+1:], m.times[i:])
	m.times[i] = t
	if len(m.times) > 64 {
		m.times = m.times[1:]
	}
}

// rate returns eligible runs/minute over the kept event-time window.
func (m *movingRate) rate(now time.Time) (perMin float64, ok bool) {
	if len(m.times) < 16 {
		return 0, false
	}
	oldest := m.times[0]
	newest := m.times[len(m.times)-1]
	if newest.Sub(oldest) < 15*time.Second {
		return 0, false
	}
	mins := now.Sub(oldest).Minutes()
	if mins <= 0 {
		return 0, false
	}
	return float64(len(m.times)-1) / mins, true
}

// eta = remaining / rate; ok=false when the rate is unavailable or zero.
func (m *movingRate) eta(now time.Time, remaining int) (time.Duration, bool) {
	r, ok := m.rate(now)
	if !ok || r <= 0 || remaining <= 0 {
		return 0, false
	}
	return time.Duration(float64(remaining) / r * float64(time.Minute)), true
}

type occupancySample struct {
	busy, capacity int
}

type occupancyWindow struct {
	samples []occupancySample
}

func (w *occupancyWindow) add(busy, capacity int) {
	if capacity <= 0 {
		return
	}
	w.samples = append(w.samples, occupancySample{busy: busy, capacity: capacity})
	if len(w.samples) > 4 {
		w.samples = w.samples[1:]
	}
}

func (w occupancyWindow) mean() (busy, capacity int, ok bool) {
	if len(w.samples) == 0 {
		return 0, 0, false
	}
	var sum int
	for _, s := range w.samples {
		sum += s.busy
		capacity = s.capacity
	}
	return int(math.Round(float64(sum) / float64(len(w.samples)))), capacity, true
}

// passRow is one outer fold's live board row (metis#38): in-flight counters + the
// pass's incumbent best (display-only — NOT the 1-SE select rule), collapsing to its
// held-out score when the driver reports the fold done.
type passRow struct {
	configK, foldK int
	best           float64
	hasBest        bool
	done           bool
	heldOut        float64
}

// boardState is the pure render input for metis#38's board: the #30 aggregate state
// plus the per-pass rows and the eligible-run rate window (a mutex'd snapshot — renderers never
// touch the live sink).
type boardState struct {
	st   progressState
	rows []passRow
	rate movingRate
}

// sweepProgress is the mutex'd sink shared by every pass of one shape-run. Events
// arrive concurrently (ParExec goroutines across sibling outer folds, each holding
// its own Run's event mutex); health-gated paths use the strict order runControl.mu
// → sink.mu → boardWriter.mu (never the reverse). Emit policy: fold/config events are throttled
// to one line per second (injected clock — tests script it, never sleep); a
// driver-level (outer fold) completion ALWAYS emits; finish() emits the terminal
// line. A nil *sweepProgress is a no-op everywhere (the non-sweep path is silent).
type sweepProgress struct {
	mu        sync.Mutex
	out       io.Writer
	now       func() time.Time
	direction string // the objective direction — orients each pass's display-best (#38)
	st        progressState
	rows      []passRow  // metis#38: one row per outer fold (nil on the flat path)
	rate      movingRate // metis#49: eligible-run completion rate window
	lastEmit  time.Time
	started   bool
	// metis#38 board mode (all nil/zero in plain mode): emits paint the rendered frame
	// to bw instead of printing plain lines. Lock order: sink.mu → bw.mu, ALWAYS — the
	// ticker enters via tick() (a sink method), never a boardWriter-first path.
	bw        *boardWriter
	width     int               // terminal width ($COLUMNS | 80), read once at wiring
	gauge     func() (int, int) // (busy, capacity) leaf occupancy; nil = no slots segment
	occupancy occupancyWindow
}

func newSweepProgress(out io.Writer, now func() time.Time, direction string, totals progressTotals) *sweepProgress {
	var rows []passRow
	if totals.nested && totals.outer > 0 {
		rows = make([]passRow, totals.outer)
	}
	return &sweepProgress{
		out: out, now: now, direction: direction, rows: rows,
		st: progressState{
			nested:     totals.nested,
			outerTotal: totals.outer, outerKind: totals.outerKind,
			configTotal: totals.configs, configKind: totals.configKind,
			foldTotal: totals.folds, foldKind: totals.foldKind,
		},
	}
}

// boardState snapshots the sink for a renderer (rows copied — the caller may hold
// the snapshot without racing the live fold-in).
func (sp *sweepProgress) boardState() boardState {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.snapshotLocked()
}

// snapshotLocked builds the render snapshot; caller holds sp.mu (shared by
// boardState() and the board-mode emit — one copy site, close-review DRY note).
func (sp *sweepProgress) snapshotLocked() boardState {
	rows := make([]passRow, len(sp.rows))
	copy(rows, sp.rows)
	return boardState{st: sp.st, rows: rows, rate: sp.rate}
}

// passHooks are one pass's typed event targets, closure-bound to its outer-fold
// index (-1 = the flat path's single pass) — the metis#38 identity seam.
type passHooks struct {
	config func(sampler.ProgressEvent[shape.Point, sampler.MeanSE])
	fold   func(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome])
}

// forPass hands out a pass's hooks. #30's sink aggregates across passes (the
// single-line mental model); the per-pass binding exists so #38 can add per-fold
// board rows behind the same API without touching pkg/sampler.
func (sp *sweepProgress) forPass(outer int) passHooks {
	if sp == nil {
		return passHooks{
			config: func(sampler.ProgressEvent[shape.Point, sampler.MeanSE]) {},
			fold:   func(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]) {},
		}
	}
	return passHooks{
		config: func(ev sampler.ProgressEvent[shape.Point, sampler.MeanSE]) {
			sp.mu.Lock()
			defer sp.mu.Unlock()
			sp.st.configK++
			if outer >= 0 && outer < len(sp.rows) { // #38: this pass's row
				r := &sp.rows[outer]
				r.configK++
				if !r.hasBest || better(sp.direction, ev.Out.Mean, r.best) {
					r.best, r.hasBest = ev.Out.Mean, true
				}
			}
			sp.maybeEmit(false)
		},
		fold: func(ev sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]) {
			sp.mu.Lock()
			defer sp.mu.Unlock()
			if !sp.st.nested {
				sp.st.flatScores = append(sp.st.flatScores, ev.Out.Score)
			}
			if outer >= 0 && outer < len(sp.rows) {
				sp.rows[outer].foldK++
			}
			sp.maybeEmit(false)
		},
	}
}

func (sp *sweepProgress) activity(ev activityEvent) {
	if sp == nil {
		return
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	switch ev.Kind {
	case activityStepSuccess:
		sp.st.stepK++
		at := ev.At
		if at.IsZero() {
			at = sp.now()
		}
		if at.After(sp.st.lastStepAt) {
			sp.st.lastStepAt = at
		}
	case activityRunSuccess:
		if ev.Role != runRoleNestedInnerCV && ev.Role != runRoleFlatCV {
			return
		}
		sp.st.foldK++
		at := ev.At
		if at.IsZero() {
			at = sp.now()
		}
		if at.After(sp.st.lastRunAt) {
			sp.st.lastRunAt = at
		}
		sp.rate.add(at)
	default:
		return
	}
	sp.maybeEmit(false)
}

// better orients a display-best comparison by the objective direction.
func better(direction string, candidate, incumbent float64) bool {
	if direction == "minimize" {
		return candidate < incumbent
	}
	return candidate > incumbent
}

// driverEvent folds a completed OUTER fold in — always emits (the coarse level is
// the one the operator watches; its completions are rare and load-bearing).
func (sp *sweepProgress) driverEvent(ev sampler.ProgressEvent[sampler.OuterFoldPoint, float64]) {
	if sp == nil {
		return
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.st.outerK++
	sp.st.outerScores = append(sp.st.outerScores, ev.Out)
	if i := ev.Point.Idx; i >= 0 && i < len(sp.rows) { // #38: collapse this fold's row
		sp.rows[i].done = true
		sp.rows[i].heldOut = ev.Out
	}
	sp.maybeEmit(true)
}

// finish emits the terminal state line (always).
func (sp *sweepProgress) finish() {
	if sp == nil {
		return
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.emit()
}

// tick is the board ticker's entry point (metis#38): repaint with a fresh `now` so
// the rate decay + ETA move even between events. Sink-first (sp.mu → bw.mu).
func (sp *sweepProgress) tick() {
	if sp == nil || sp.bw == nil {
		return
	}
	sp.mu.Lock()
	if sp.gauge != nil {
		busy, capacity := sp.gauge()
		sp.occupancy.add(busy, capacity)
	}
	sp.emit() // stores the fresh frame (budget may skip the draw)
	bw := sp.bw
	sp.mu.Unlock()
	bw.forceFlush() // metis#46: the tick is what re-pins the board after a burst window
}

// abort removes the stored live frame after a sweep failure. Lock order remains
// progress -> board; the controller is never called while either lock is held.
func (sp *sweepProgress) abort() {
	if sp == nil || sp.bw == nil {
		return
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.bw.discardFrame()
}

// maybeEmit writes the line if forced (driver/finish) or the throttle elapsed —
// 1s for plain lines (a log is a record), 100ms for board repaints (a board is a
// display; the 500ms ticker guarantees freshness anyway). Caller holds sp.mu.
func (sp *sweepProgress) maybeEmit(force bool) {
	now := sp.now()
	throttle := time.Second
	if sp.bw != nil {
		throttle = 100 * time.Millisecond
	}
	if !force && sp.started && now.Sub(sp.lastEmit) < throttle {
		return
	}
	sp.started = true
	sp.lastEmit = now
	sp.emit()
}

// emit renders the current state: board mode paints the frame (under the fixed
// sink.mu → bw.mu order; the snapshot is built inline — boardState() would re-lock);
// plain mode prints the #30 aggregated line. Caller holds sp.mu.
func (sp *sweepProgress) emit() {
	if sp.bw != nil {
		busy, capacity, _ := sp.occupancy.mean()
		sp.bw.paint(renderBoard(sp.snapshotLocked(),
			boardEnv{width: sp.width, now: sp.now(), busy: busy, capacity: capacity}), sp.width)
		return
	}
	fmt.Fprintln(sp.out, progressLine(sp.st))
}
