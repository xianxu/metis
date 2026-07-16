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
	outerScores          []float64 // nested: completed outer-fold held-out scores
	flatScores           []float64 // flat: the one config's completed fold scores
}

// progressLine renders the aggregated line. Nested:
// `outer 1/3 · configs 84/216 · folds 421/1080 · est 0.8283 ± 0.0140`
// (est — until an outer fold lands; ± only at n≥2). Flat (since metis#32: iff 1
// config): `folds 3/5 · score 0.8400` (the running fold mean — nothing to be
// "best" of). Kinds render k/n (exact), k/≤n (budget), k/? (unknown). Pure.
func progressLine(st progressState) string {
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
		parts = append(parts, "outer "+frac(st.outerK, st.outerTotal, st.outerKind))
		parts = append(parts, "configs "+frac(st.configK, st.configTotal, st.configKind))
		parts = append(parts, "folds "+frac(st.foldK, st.foldTotal, st.foldKind))
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
		parts = append(parts, "folds "+frac(st.foldK, st.foldTotal, st.foldKind))
		if mean, _, n := meanSE(st.flatScores); n > 0 {
			parts = append(parts, fmt.Sprintf("score %.4f", mean))
		}
	}
	return "metis: progress " + strings.Join(parts, " · ")
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

// movingRate is metis#38's throughput window: a ring of the last 64 fold-completion
// instants. rate(now) = n / (now − oldest) — `now` in the denominator means a STALL
// decays the rate live (the k10-probe BLAS-thrash signature: throughput → 0 while the
// process looks alive). Moving-average by construction (the operator's requirement:
// per-leaf times vary by config, rf500 ≫ logreg). Pure over passed-in instants.
type movingRate struct {
	times [64]time.Time
	n     int // total adds (ring index = n % len)
}

func (m *movingRate) add(t time.Time) {
	m.times[m.n%len(m.times)] = t
	m.n++
}

// rate returns completions/minute over the kept window; ok=false until 2 completions.
func (m *movingRate) rate(now time.Time) (perMin float64, ok bool) {
	if m.n < 2 {
		return 0, false
	}
	kept := m.n
	if kept > len(m.times) {
		kept = len(m.times)
	}
	oldest := m.times[(m.n-kept)%len(m.times)]
	mins := now.Sub(oldest).Minutes()
	if mins <= 0 {
		return 0, false
	}
	return float64(kept) / mins, true
}

// eta = remaining / rate; ok=false when the rate is unavailable or zero.
func (m *movingRate) eta(now time.Time, remaining int) (time.Duration, bool) {
	r, ok := m.rate(now)
	if !ok || r <= 0 || remaining <= 0 {
		return 0, false
	}
	return time.Duration(float64(remaining) / r * float64(time.Minute)), true
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
// plus the per-pass rows and the throughput ring (a mutex'd snapshot — renderers never
// touch the live sink).
type boardState struct {
	st   progressState
	rows []passRow
	rate movingRate
}

// sweepProgress is the mutex'd sink shared by every pass of one shape-run. Events
// arrive concurrently (ParExec goroutines across sibling outer folds, each holding
// its own Run's event mutex); lock order is strictly Run-mu → sink-mu → the
// syncWriter under `out` — acyclic. Emit policy: fold/config events are throttled
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
	rate      movingRate // metis#38: fold-completion throughput window
	lastEmit  time.Time
	started   bool
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
			sp.st.foldK++
			sp.rate.add(sp.now()) // #38: throughput window feeds off every leaf completion
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

// maybeEmit writes the line if forced (driver/finish) or the 1s throttle elapsed.
// Caller holds sp.mu.
func (sp *sweepProgress) maybeEmit(force bool) {
	now := sp.now()
	if !force && sp.started && now.Sub(sp.lastEmit) < time.Second {
		return
	}
	sp.started = true
	sp.lastEmit = now
	sp.emit()
}

func (sp *sweepProgress) emit() {
	fmt.Fprintln(sp.out, progressLine(sp.st))
}
