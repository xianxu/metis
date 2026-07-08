// Package ledger is the metis#8 shape-run ledger: the L1 tracking layer that turns a
// sweep's per-point results into a navigable, promotable table, so an engineer picks
// the winner by sorting rather than scrolling logs. It is an APPEND-ONLY aggregation
// VIEW over the per-run records (metis#3) — not a competing run store: a Row is the raw
// reconstructable recipe (free-param tuple + code SHA + seed) plus the result. Pure —
// the CSV codec, dedup, and pick-best have no IO; reading the manifest/record.json and
// committing the sidecar are the cmd/metis shell.
//
// Three keys per row: the free-param tuple (human navigation, ragged/sparse — columns
// are the union of all branches' free-params, blank where inactive), the sweep-SHA (the
// code-version, git short-SHA human address), and the point-address (global content
// identity, used for append dedup).
package ledger

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Row is one ledger entry: the free-param tuple + sweep-SHA + point-address (identity)
// + namespaced metrics + status. The raw recipe + result.
//
// metis#18 (M1a): a Row is now a RAW per-fold row — one (config, fold) run's fold_score.
// Fold is the resample-fold coordinate (nil = a non-fold row, e.g. a v1 whole-CV row or
// an AggregateView output). Keeping the raw fold rows (not a pre-reduced mean) is what
// lets metis#19's 1-standard-error select re-reduce for free — AggregateView groups them
// read-time into per-config (mean, SE) without a re-run.
type Row struct {
	FreeParams map[string]any
	SweepSHA   string
	PointAddr  string // the metis#3 point content-address — row identity for dedup
	Fold       *int   // metis#18: the resample-fold index (nil = not a per-fold row)
	Metrics    map[string]float64
	Status     string // "ok" | "failed"
}

// Ledger is an ordered, append-only set of rows (CSV-backed by the caller).
type Ledger struct {
	Rows []Row
	seen map[string]bool // point-addresses already present (dedup)
}

// Append adds rows in order, skipping any whose point-address is already present
// (append-only + idempotent: re-running the same code re-produces the same addresses →
// no growth; a new code-version's rows carry new addresses → they append).
func (l *Ledger) Append(rows ...Row) {
	if l.seen == nil {
		l.seen = make(map[string]bool, len(l.Rows)+len(rows))
		for _, r := range l.Rows {
			l.seen[r.PointAddr] = true
		}
	}
	for _, r := range rows {
		if l.seen[r.PointAddr] {
			continue
		}
		l.seen[r.PointAddr] = true
		l.Rows = append(l.Rows, r)
	}
}

// column prefixes for the CSV header namespace.
const (
	fpPrefix     = "fp."
	metricPrefix = "metric."
)

// Encode renders the ledger as append-order CSV. The header is the fixed keys
// (sweep_sha, point_addr, status) plus the sorted UNION of every row's free-param and
// metric columns; a row blanks the columns it lacks (ragged). Sorting/filtering is a
// view (Decode → Filter/TopN), never a storage concern — the file stays append-order.
func Encode(l Ledger) ([]byte, error) {
	fpCols, metricCols := unionColumns(l.Rows)
	// The `fold` column is present only when ≥1 row carries a fold coordinate (a per-fold
	// sweep) — so a v1-shaped ledger (no folds) stays byte-identical (ragged, ARCH-DRY).
	hasFold := false
	for _, r := range l.Rows {
		if r.Fold != nil {
			hasFold = true
			break
		}
	}
	header := append([]string{"sweep_sha", "point_addr", "status"}, fpCols...)
	if hasFold {
		header = append(header, "fold")
	}
	header = append(header, metricCols...)

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, r := range l.Rows {
		rec := make([]string, 0, len(header))
		rec = append(rec, r.SweepSHA, r.PointAddr, r.Status)
		for _, c := range fpCols {
			rec = append(rec, cell(r.FreeParams[strings.TrimPrefix(c, fpPrefix)]))
		}
		if hasFold {
			if r.Fold != nil {
				rec = append(rec, strconv.Itoa(*r.Fold))
			} else {
				rec = append(rec, "") // an aggregate/non-fold row blanks the fold column
			}
		}
		for _, c := range metricCols {
			if v, ok := r.Metrics[strings.TrimPrefix(c, metricPrefix)]; ok {
				rec = append(rec, strconv.FormatFloat(v, 'g', -1, 64))
			} else {
				rec = append(rec, "")
			}
		}
		if err := w.Write(rec); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// Decode parses append-order CSV back into a Ledger (blank cells → absent keys).
func Decode(b []byte) (Ledger, error) {
	r := csv.NewReader(bytes.NewReader(b))
	recs, err := r.ReadAll()
	if err != nil {
		return Ledger{}, err
	}
	if len(recs) == 0 {
		return Ledger{}, nil
	}
	header := recs[0]
	var l Ledger
	for _, rec := range recs[1:] {
		row := Row{FreeParams: map[string]any{}, Metrics: map[string]float64{}}
		for i, col := range header {
			if i >= len(rec) || rec[i] == "" {
				continue
			}
			switch {
			case col == "sweep_sha":
				row.SweepSHA = rec[i]
			case col == "point_addr":
				row.PointAddr = rec[i]
			case col == "status":
				row.Status = rec[i]
			case col == "fold":
				if f, err := strconv.Atoi(rec[i]); err == nil {
					row.Fold = &f
				}
			case strings.HasPrefix(col, fpPrefix):
				row.FreeParams[strings.TrimPrefix(col, fpPrefix)] = parseCell(rec[i])
			case strings.HasPrefix(col, metricPrefix):
				if f, err := strconv.ParseFloat(rec[i], 64); err == nil {
					row.Metrics[strings.TrimPrefix(col, metricPrefix)] = f
				}
			}
		}
		if len(row.FreeParams) == 0 {
			row.FreeParams = nil
		}
		if len(row.Metrics) == 0 {
			row.Metrics = nil
		}
		l.Append(row)
	}
	return l, nil
}

// Aggregate-view metric suffixes: a per-config row carries `<metric>` (the fold mean),
// `<metric>.se` (standard error), and `<metric>.n` (fold count).
const (
	seSuffix = ".se"
	nSuffix  = ".n"
)

// AggregateView reduces the RAW per-fold rows into one per-config (mean, SE) row —
// grouping by (free-params, sweep-SHA) over the fold coordinate. metis#18: the ledger
// stores the raw fold rows (so metis#19's 1-standard-error select re-reduces for free),
// and this is the read-time reduction the leaderboard + `promote --best` sort over.
// `metric` is the per-fold metric to reduce (e.g. "train.fold_score"); each aggregate row
// carries `metric` (the mean) + `metric+".se"` (standard error) + `metric+".n"` (fold
// count). A group with ANY failed fold is marked failed. Rows with no fold coordinate
// (Fold==nil — a v1 whole-CV row or an already-aggregated row) pass through unchanged, so
// the view is idempotent. Grouping + ordering are deterministic (first-seen config order).
func AggregateView(l Ledger, metric string) Ledger {
	type agg struct {
		row    Row
		scores []float64
		failed bool
	}
	var order []string
	groups := map[string]*agg{}
	var out Ledger
	for _, r := range l.Rows {
		if r.Fold == nil {
			out.Append(r) // pass a pre-aggregated / v1 row through untouched
			continue
		}
		fpb, _ := json.Marshal(r.FreeParams) // Go sorts map keys → canonical group key
		key := r.SweepSHA + "\x00" + string(fpb)
		g := groups[key]
		if g == nil {
			g = &agg{row: Row{FreeParams: r.FreeParams, SweepSHA: r.SweepSHA, PointAddr: key, Status: "ok"}}
			groups[key] = g
			order = append(order, key)
		}
		if r.Status == "failed" {
			g.failed = true
		}
		if v, ok := r.Metrics[metric]; ok {
			g.scores = append(g.scores, v)
		}
	}
	for _, key := range order {
		g := groups[key]
		row := g.row
		if g.failed {
			row.Status = "failed"
		}
		if n := len(g.scores); n > 0 {
			mean, se := meanSE(g.scores)
			row.Metrics = map[string]float64{
				metric:           mean,
				metric + seSuffix: se,
				metric + nSuffix:  float64(n),
			}
		}
		out.Append(row)
	}
	return out
}

// meanSE reduces fold scores → (mean, standard error). SE = sample-std/√n (n−1
// denominator; 0 when n<2). Order-independent. Mirrors sampler.Aggregate — the ledger
// re-derives it read-time from the raw rows rather than trusting a stored mean.
func meanSE(scores []float64) (mean, se float64) {
	n := len(scores)
	if n == 0 {
		return 0, 0
	}
	var sum float64
	for _, s := range scores {
		sum += s
	}
	mean = sum / float64(n)
	if n > 1 {
		var ss float64
		for _, s := range scores {
			d := s - mean
			ss += d * d
		}
		se = math.Sqrt(ss/float64(n-1)) / math.Sqrt(float64(n))
	}
	return mean, se
}

// Best returns the row optimizing the objective metric (maximize / minimize), skipping
// failed rows and rows missing the metric. ok=false if no row qualifies.
func Best(l Ledger, metric, direction string) (Row, bool) {
	var best Row
	found := false
	for _, r := range l.Rows {
		if r.Status == "failed" {
			continue
		}
		v, ok := r.Metrics[metric]
		if !ok {
			continue
		}
		if !found || betterThan(v, best.Metrics[metric], direction) {
			best, found = r, true
		}
	}
	return best, found
}

// TopN returns the n best rows in objective order (skipping failed / metric-missing).
func TopN(l Ledger, metric, direction string, n int) []Row {
	qualified := make([]Row, 0, len(l.Rows))
	for _, r := range l.Rows {
		if r.Status == "failed" {
			continue
		}
		if _, ok := r.Metrics[metric]; ok {
			qualified = append(qualified, r)
		}
	}
	sort.SliceStable(qualified, func(i, j int) bool {
		return betterThan(qualified[i].Metrics[metric], qualified[j].Metrics[metric], direction)
	})
	if n < 0 {
		n = 0
	}
	if n < len(qualified) {
		qualified = qualified[:n]
	}
	return qualified
}

// SortAll returns ALL rows for a `show` view: those with the metric sorted by the
// objective first, then the rest (failed / metric-missing) in append order — so sorting
// never DROPS a row (unlike TopN, which is a ranked leaderboard for the body summary).
func SortAll(l Ledger, metric, direction string) []Row {
	var qualified, rest []Row
	for _, r := range l.Rows {
		if r.Status != "failed" {
			if _, ok := r.Metrics[metric]; ok {
				qualified = append(qualified, r)
				continue
			}
		}
		rest = append(rest, r)
	}
	sort.SliceStable(qualified, func(i, j int) bool {
		return betterThan(qualified[i].Metrics[metric], qualified[j].Metrics[metric], direction)
	})
	return append(qualified, rest...)
}

// Filter returns the sub-ledger of rows at a given sweep-SHA (an invocation / code-
// version view). Empty sweepSHA returns the whole ledger.
func Filter(l Ledger, sweepSHA string) Ledger {
	var out Ledger
	for _, r := range l.Rows {
		if sweepSHA == "" || r.SweepSHA == sweepSHA {
			out.Append(r)
		}
	}
	return out // a fresh Ledger with its own Rows/seen — appending to it never touches l
}

func betterThan(a, b float64, direction string) bool {
	if direction == "minimize" {
		return a < b
	}
	return a > b // default / maximize
}

// unionColumns returns the sorted union of free-param and metric column names (prefixed).
func unionColumns(rows []Row) (fp, metric []string) {
	fpSet, metricSet := map[string]bool{}, map[string]bool{}
	for _, r := range rows {
		for k := range r.FreeParams {
			fpSet[fpPrefix+k] = true
		}
		for k := range r.Metrics {
			metricSet[metricPrefix+k] = true
		}
	}
	return sortedKeys(fpSet), sortedKeys(metricSet)
}

func sortedKeys(m map[string]bool) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// cell renders a free-param value for CSV; parseCell inverts it. Scalars stay bare for
// human navigation (`rf`, `300`, `true`); lists/maps (a `features: [title, family]`
// free-param — kbench#4 sweeps these) are JSON-encoded so they round-trip losslessly
// instead of collapsing to an unparseable `[title family]` string.
func cell(v any) string {
	switch v.(type) {
	case nil:
		return ""
	case []any, map[string]any:
		b, err := json.Marshal(v)
		if err == nil {
			return string(b)
		}
	}
	return fmt.Sprintf("%v", v)
}

func parseCell(s string) any {
	switch s {
	case "true":
		return true
	case "false":
		return false
	}
	if len(s) > 0 && (s[0] == '[' || s[0] == '{') {
		var v any
		if err := json.Unmarshal([]byte(s), &v); err == nil {
			return v
		}
	}
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}
