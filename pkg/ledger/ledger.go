// Package ledger is the metis#8 shape-run ledger: the L1 tracking layer that turns a
// sweep's per-point results into a navigable, promotable table, so an engineer picks
// the winner by sorting rather than scrolling logs. It is an APPEND-ONLY aggregation
// VIEW over the per-run records (metis#3) — not a competing run store: a Row is the raw
// reconstructable recipe (free-param tuple + code_fingerprint + seed) plus the result. Pure —
// the CSV codec, dedup, and pick-best have no IO; reading the manifest/record.json and
// committing the sidecar are the cmd/metis shell.
//
// Keys per row (metis#27): the free-param tuple (human navigation, ragged/sparse — columns
// are the union of all branches' free-params, blank where inactive), the code_fingerprint (the
// realized code identity over the run's D closure), and the point-address (intent identity).
// Append dedup keys on (point_address, code_fingerprint) so same-config-different-code runs are
// both kept.
//
// Repeats (metis#73): a row with a repeat label is one observed sample of an execution
// (`metis run --repeat L`). Its key adds the label, so repeats of one config stay distinct;
// it is immutable (an exact re-publish is a no-op, different data under the same key is an
// error). A plain single-run row (no fold, level or repeat) mirrors its overwritable run dir,
// so Publish replaces it. Columns this version does not know are kept in Row.Extra and
// written back, so a rewrite never drops another version's data.
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

	"github.com/xianxu/metis/pkg/experiment"
)

// Row is one ledger entry: the free-param tuple + code-fingerprint + point-address
// (the (point-address, fingerprint) pair is the identity) + namespaced metrics + status.
//
// metis#18 (M1a): a Row is now a RAW per-fold row — one (config, fold) run's fold_score.
// Fold is the resample-fold coordinate (nil = a non-fold row, e.g. a v1 whole-CV row or
// an AggregateView output). Keeping the raw fold rows (not a pre-reduced mean) is what
// lets metis#19's 1-standard-error select re-reduce for free — AggregateView groups them
// read-time into per-config (mean, SE) without a re-run.
type Row struct {
	FreeParams      map[string]any
	CodeFingerprint string // metis#27: the run's realized code identity (over its D closure)
	PointAddr       string // the metis#3 point INTENT-address — half the dedup identity
	Fold            *int   // metis#18: the resample-fold index (nil = not a per-fold row)
	// metis#32 nested-CV recording: Level discriminates the two ledger levels a nested run
	// writes — "" (a v1 flat / single-level row), "inner" (a sealed inner-CV fold row), or
	// "outer" (a per-(outer-fold, family) held-out score). Level MUST enter the AggregateView
	// group key: an inner row and the outer row for the same winning config+fold otherwise
	// share (fingerprint, free-params, fold) and get averaged together (a subset inner score
	// blended with an outer held-out score) — the collision metis#32's recording must prevent.
	Level string
	// OuterFold is the nested outer-fold coordinate (metis#32): recorded for provenance +
	// the FamilyEstimate reducer's fold count. A COLUMN, not part of the group/dedup key —
	// inner/outer rows already differ by PointAddr (distinct content-addresses), and inner
	// rows POOL across outer folds by design (the pooled inner-CV config-selection signal).
	OuterFold *int
	// Stopped is the metis#66 auto-stop marker — "" (normal) | "auto" (the family's remaining
	// outer folds were cut by --auto-stop because it was <95%-likely to reach the incumbent). A
	// ragged COLUMN (present only when ≥1 row carries it), not part of the key/dedup — it's
	// provenance a reader uses to tell a truncated loser from a --sample-limited run. AggregateView
	// passes it through untouched (NOT treated as failed — a stopped fold's rows are honest).
	Stopped string
	// Repeat is the metis#73 execution coordinate: "" (not a repeat), a repeat label, or
	// AggregateRepeat on an AggregateView row reducing a repeat family. Part of the key when
	// set; a ragged column.
	Repeat  string
	Metrics map[string]float64
	Status  string // "ok" | "failed"
	// Extra holds columns this version does not know (written by a newer metis), re-emitted
	// verbatim on Encode so a load-append-rewrite never drops them (metis#73, ARCH-EVOLVE).
	Extra map[string]string
}

// AggregateRepeat marks an AggregateView row that reduces a repeat family. It is not a valid
// repeat label (labels are slugs), so the aggregate never shares a key with a sample or with
// the plain row of the same point.
const AggregateRepeat = "*"

// Ledger is an ordered, append-only set of rows (CSV-backed by the caller).
type Ledger struct {
	Rows []Row
	seen map[string]int // row key → its index in Rows (dedup, conflict check, replace)
}

// dedupKey is the metis#27 composite row identity: the intent (point-address) AND the
// realized code (fingerprint). NUL-separated (neither half contains NUL — both are hex
// hashes or a config-group id). Same config + different code → distinct key → both kept.
// metis#73: a repeat row adds its label, so a row without one keys exactly as before.
func dedupKey(r Row) string {
	k := r.PointAddr + "\x00" + r.CodeFingerprint
	if r.Repeat != "" {
		k += "\x00" + r.Repeat
	}
	return k
}

func (l *Ledger) index() {
	if l.seen == nil {
		l.seen = make(map[string]int, len(l.Rows))
		for i, r := range l.Rows {
			l.seen[dedupKey(r)] = i
		}
	}
}

// Append adds rows in order, skipping any whose (point-address, code-fingerprint) is
// already present (append-only + idempotent: re-running the SAME code re-produces the same
// pairs → no growth; a new code-version's rows carry a new fingerprint → they append,
// preserving both variations even at the same config). A repeat row (metis#73) is an
// immutable sample: an identical re-append is a no-op, while different data under its key is
// refused with an error naming the execution (rows before it are kept). Rows without a
// repeat label never error.
func (l *Ledger) Append(rows ...Row) error {
	l.index()
	for _, r := range rows {
		k := dedupKey(r)
		if i, ok := l.seen[k]; ok {
			if r.Repeat != "" && !SameRow(l.Rows[i], r) {
				return conflictError(l.Rows[i], r)
			}
			continue
		}
		l.seen[k] = len(l.Rows)
		l.Rows = append(l.Rows, r)
	}
	return nil
}

// Publish adds one execution's row (metis#73): a plain single-run row (no fold, level or
// repeat) REPLACES the row with its key, since it mirrors the run dir a rerun overwrites
// (latest replay wins); every other row follows Append.
func (l *Ledger) Publish(r Row) error {
	if r.Fold != nil || r.Level != "" || r.Repeat != "" {
		return l.Append(r)
	}
	l.index()
	if i, ok := l.seen[dedupKey(r)]; ok {
		l.Rows[i] = r
		return nil
	}
	return l.Append(r)
}

func conflictError(have, got Row) error {
	return fmt.Errorf("conflicting data for execution %s (code %s, repeat %s): the ledger already holds a different row for it (status %s, metrics %v; new: status %s, metrics %v)",
		have.PointAddr, have.CodeFingerprint, have.Repeat, have.Status, have.Metrics, got.Status, got.Metrics)
}

// SameRow reports whether two rows carry the same data: what an idempotent re-publish must
// match. Free params compare by canonical JSON (a CSV round-trip may turn 300 into 300.0),
// metrics by value (NaN equals NaN), empty and nil maps alike.
func SameRow(a, b Row) bool {
	if a.CodeFingerprint != b.CodeFingerprint || a.PointAddr != b.PointAddr || a.Status != b.Status ||
		a.Level != b.Level || a.Stopped != b.Stopped || a.Repeat != b.Repeat ||
		!sameIntPtr(a.Fold, b.Fold) || !sameIntPtr(a.OuterFold, b.OuterFold) {
		return false
	}
	fa, errA := json.Marshal(nonNil(a.FreeParams))
	fb, errB := json.Marshal(nonNil(b.FreeParams))
	if errA != nil || errB != nil || !bytes.Equal(fa, fb) {
		return false
	}
	if len(a.Metrics) != len(b.Metrics) || len(a.Extra) != len(b.Extra) {
		return false
	}
	for k, v := range a.Metrics {
		w, ok := b.Metrics[k]
		if !ok || (v != w && !(math.IsNaN(v) && math.IsNaN(w))) {
			return false
		}
	}
	for k, v := range a.Extra {
		if w, ok := b.Extra[k]; !ok || v != w {
			return false
		}
	}
	return true
}

func sameIntPtr(a, b *int) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }

func nonNil(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

// column prefixes for the CSV header namespace.
const (
	fpPrefix     = "fp."
	metricPrefix = "metric."
)

// Encode renders the ledger as append-order CSV. The header is the fixed keys
// (code_fingerprint, point_addr, status) plus the sorted UNION of every row's free-param and
// metric columns; a row blanks the columns it lacks (ragged). Sorting/filtering is a
// view (Decode → Filter/TopN), never a storage concern — the file stays append-order.
func Encode(l Ledger) ([]byte, error) {
	fpCols, metricCols := unionColumns(l.Rows)
	// The `fold`/`level`/`outer_fold`/`stopped` columns are present only when ≥1 row carries them
	// (a per-fold or nested sweep) — so a v1-shaped ledger stays byte-identical (ragged, ARCH-DRY).
	hasFold, hasLevel, hasOuterFold, hasStopped, hasRepeat := false, false, false, false, false
	extraSet := map[string]bool{}
	for _, r := range l.Rows {
		if r.Repeat != "" {
			hasRepeat = true
		}
		for k := range r.Extra {
			extraSet[k] = true
		}
		if r.Fold != nil {
			hasFold = true
		}
		if r.Level != "" {
			hasLevel = true
		}
		if r.OuterFold != nil {
			hasOuterFold = true
		}
		if r.Stopped != "" {
			hasStopped = true
		}
	}
	header := append([]string{"code_fingerprint", "point_addr", "status"}, fpCols...)
	if hasLevel {
		header = append(header, "level")
	}
	if hasFold {
		header = append(header, "fold")
	}
	if hasOuterFold {
		header = append(header, "outer_fold")
	}
	if hasStopped {
		header = append(header, "stopped")
	}
	if hasRepeat {
		header = append(header, "repeat")
	}
	header = append(header, metricCols...)
	extraCols := sortedKeys(extraSet) // unknown columns last, as read (metis#73)
	header = append(header, extraCols...)

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, r := range l.Rows {
		rec := make([]string, 0, len(header))
		rec = append(rec, r.CodeFingerprint, r.PointAddr, r.Status)
		for _, c := range fpCols {
			rec = append(rec, cell(r.FreeParams[strings.TrimPrefix(c, fpPrefix)]))
		}
		if hasLevel {
			rec = append(rec, r.Level) // "" for a v1/flat row
		}
		if hasFold {
			if r.Fold != nil {
				rec = append(rec, strconv.Itoa(*r.Fold))
			} else {
				rec = append(rec, "") // an aggregate/non-fold row blanks the fold column
			}
		}
		if hasOuterFold {
			if r.OuterFold != nil {
				rec = append(rec, strconv.Itoa(*r.OuterFold))
			} else {
				rec = append(rec, "")
			}
		}
		if hasStopped {
			rec = append(rec, r.Stopped) // "" for a normal row
		}
		if hasRepeat {
			rec = append(rec, r.Repeat) // "" for a non-repeat row
		}
		for _, c := range metricCols {
			if v, ok := r.Metrics[strings.TrimPrefix(c, metricPrefix)]; ok {
				rec = append(rec, strconv.FormatFloat(v, 'g', -1, 64))
			} else {
				rec = append(rec, "")
			}
		}
		for _, c := range extraCols {
			rec = append(rec, r.Extra[c])
		}
		if err := w.Write(rec); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// Decode parses append-order CSV back into a Ledger (blank cells → absent keys). Legacy
// columns parse leniently as before; a repeat cell must be a valid label, and an unknown
// column lands in Row.Extra. Two conflicting rows for one repeat execution are an error (a
// contradictory research record is reported, never silently resolved).
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
	for n, rec := range recs[1:] {
		row := Row{FreeParams: map[string]any{}, Metrics: map[string]float64{}}
		for i, col := range header {
			if i >= len(rec) || rec[i] == "" {
				continue
			}
			switch {
			case col == "code_fingerprint":
				row.CodeFingerprint = rec[i]
			case col == "point_addr":
				row.PointAddr = rec[i]
			case col == "status":
				row.Status = rec[i]
			case col == "level":
				row.Level = rec[i]
			case col == "fold":
				if f, err := strconv.Atoi(rec[i]); err == nil {
					row.Fold = &f
				}
			case col == "outer_fold":
				if f, err := strconv.Atoi(rec[i]); err == nil {
					row.OuterFold = &f
				}
			case col == "stopped":
				row.Stopped = rec[i]
			case col == "repeat":
				if _, err := experiment.ParseRepeatLabel(rec[i]); err != nil {
					return Ledger{}, fmt.Errorf("ledger row %d: %w", n+1, err)
				}
				row.Repeat = rec[i]
			case strings.HasPrefix(col, fpPrefix):
				row.FreeParams[strings.TrimPrefix(col, fpPrefix)] = parseCell(rec[i])
			case strings.HasPrefix(col, metricPrefix):
				if f, err := strconv.ParseFloat(rec[i], 64); err == nil {
					row.Metrics[strings.TrimPrefix(col, metricPrefix)] = f
				}
			default:
				if row.Extra == nil {
					row.Extra = map[string]string{}
				}
				row.Extra[col] = rec[i]
			}
		}
		if len(row.FreeParams) == 0 {
			row.FreeParams = nil
		}
		if len(row.Metrics) == 0 {
			row.Metrics = nil
		}
		if err := l.Append(row); err != nil {
			return Ledger{}, fmt.Errorf("ledger row %d: %w", n+1, err)
		}
	}
	return l, nil
}

// Aggregate-view metric suffixes: a per-config row carries `<metric>` (the fold mean),
// `<metric>.se` (standard error), and `<metric>.n` (fold count).
const (
	seSuffix = ".se"
	nSuffix  = ".n"
)

// A repeat family row's member counts (metis#73): all repeats, and the failed ones.
const (
	RepeatsTotal  = "repeats.total"
	RepeatsFailed = "repeats.failed"
)

// AggregateView reduces the RAW per-fold rows into one per-config row — grouping by
// (free-params, code-fingerprint) over the fold coordinate. metis#18: the ledger stores the raw
// fold rows (so metis#19's select re-reduces for free), and this is the read-time reduction
// the leaderboard + `promote --best` + `ledger select` read over. EVERY metric column is
// mean-reduced (so metis#19's `train.complexity` reaches the aggregate row for the select
// rule), but only the objective `metric` also gets `.se` (standard error) + `.n` (fold
// count) — a mean is the meaningful reduction for a complexity axis; its spread isn't a
// selection input. A group with ANY failed fold is marked failed. Rows with no fold
// coordinate (Fold==nil — a v1 whole-CV row or an already-aggregated row) pass through
// unchanged, so the view is idempotent. Grouping + ordering are deterministic (first-seen).
//
// metis#73: repeat rows (no fold, a repeat label) form a second grouping, one family per
// (code-fingerprint, point-address): the same config, seed and code. The family row carries
// Repeat=AggregateRepeat and, for EVERY metric, the mean, `.se` and `.n` (how many repeats
// reported it), plus RepeatsTotal and RepeatsFailed. A failed repeat counts but contributes no
// values, a missing value shrinks only that metric's n (nothing is filled in), and the family
// is ok when any member is. A repeat is a sample, never a fold: no CV reduction applies.
func AggregateView(l Ledger, metric string) Ledger {
	type agg struct {
		row      Row
		byMetric map[string][]float64 // every metric key → its fold values
		failed   bool
		repeat   bool // a repeat family (metis#73)
		total    int  // repeat members
		nFailed  int  // failed repeat members
	}
	var order []string
	groups := map[string]*agg{}
	var out Ledger
	for _, r := range l.Rows {
		if r.Fold == nil && r.Repeat != "" && r.Repeat != AggregateRepeat {
			key := "repeat|" + r.CodeFingerprint + "|" + r.PointAddr
			g := groups[key]
			if g == nil {
				g = &agg{row: Row{FreeParams: r.FreeParams, CodeFingerprint: r.CodeFingerprint, PointAddr: r.PointAddr, Repeat: AggregateRepeat}, byMetric: map[string][]float64{}, repeat: true}
				groups[key] = g
				order = append(order, key)
			}
			g.total++
			if r.Status == "failed" {
				g.nFailed++
				continue // counted, but a failed sample contributes no values
			}
			for name, v := range r.Metrics {
				g.byMetric[name] = append(g.byMetric[name], v)
			}
			continue
		}
		if r.Fold == nil {
			out.Append(r) // pass a pre-aggregated / v1 / plain row through untouched
			continue
		}
		fpb, _ := json.Marshal(r.FreeParams)  // Go sorts map keys → canonical group key
		// metis#32: Level is part of the key — an inner and an outer row sharing (fingerprint,
		// free-params) must NOT merge (a v1 row's Level is "", so a fold-only ledger keys
		// identically to before — back-compatible).
		key := r.CodeFingerprint + "|" + r.Level + "|" + string(fpb) // internal (level, config)-grouping
		//   id only — NOT stored on the row. The aggregate row's PointAddr is the FIRST member's
		//   REAL addr (metis#51): resolvePointRows documents "any of a config's fold rows works as
		//   a handle" and expands to the whole config, so a representative member addr is a valid,
		//   resolvable handle — the single source both `ledger show`'s point column and select's
		//   #52 pick-line handles derive from (a synthetic key here rendered as garbage and could
		//   never resolve).
		g := groups[key]
		if g == nil {
			g = &agg{row: Row{FreeParams: r.FreeParams, CodeFingerprint: r.CodeFingerprint, PointAddr: r.PointAddr, Level: r.Level, Status: "ok"}, byMetric: map[string][]float64{}}
			groups[key] = g
			order = append(order, key)
		}
		if r.Status == "failed" {
			g.failed = true
		}
		for name, v := range r.Metrics {
			g.byMetric[name] = append(g.byMetric[name], v)
		}
	}
	for _, key := range order {
		g := groups[key]
		row := g.row
		if g.repeat {
			row.Status = "ok"
			if g.nFailed == g.total {
				row.Status = "failed"
			}
			row.Metrics = map[string]float64{RepeatsTotal: float64(g.total), RepeatsFailed: float64(g.nFailed)}
			for name, vals := range g.byMetric {
				mean, se := meanSE(vals)
				row.Metrics[name] = mean
				row.Metrics[name+seSuffix] = se
				row.Metrics[name+nSuffix] = float64(len(vals))
			}
			out.Append(row)
			continue
		}
		if g.failed {
			row.Status = "failed"
		}
		if len(g.byMetric) > 0 {
			row.Metrics = map[string]float64{}
			for name, vals := range g.byMetric {
				mean, se := meanSE(vals)
				row.Metrics[name] = mean
				if name == metric { // .se/.n only for the objective metric
					row.Metrics[name+seSuffix] = se
					row.Metrics[name+nSuffix] = float64(len(vals))
				}
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

// Filter returns the sub-ledger of rows at a given code-fingerprint (a code-version
// view). Empty fingerprint returns the whole ledger.
func Filter(l Ledger, fingerprint string) Ledger {
	var out Ledger
	for _, r := range l.Rows {
		if fingerprint == "" || r.CodeFingerprint == fingerprint {
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
