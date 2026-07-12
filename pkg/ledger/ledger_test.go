package ledger

import (
	"strings"
	"testing"
)

func row(fingerprint, addr string, fp map[string]any, metrics map[string]float64, status string) Row {
	return Row{CodeFingerprint: fingerprint, PointAddr: addr, FreeParams: fp, Metrics: metrics, Status: status}
}

// Append is append-only + dedups by (point-address, code-fingerprint) — the metis#27
// composite identity: re-appending a seen (addr, fingerprint) is a no-op; a new address
// appends; AND the SAME address with a DIFFERENT fingerprint (same config, different code)
// appends too, so both variations are preserved (neither silently overwritten).
func TestAppend_DedupByPointAddrAndFingerprint(t *testing.T) {
	var l Ledger
	l.Append(
		row("cf1", "addr-a", map[string]any{"model": "logreg"}, map[string]float64{"train.cv_score": 0.8}, "ok"),
		row("cf1", "addr-b", map[string]any{"model": "rf"}, map[string]float64{"train.cv_score": 0.82}, "ok"),
	)
	if len(l.Rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(l.Rows))
	}
	// Re-append the same (addr, fingerprint) (idempotent re-run) → no growth.
	l.Append(row("cf1", "addr-a", map[string]any{"model": "logreg"}, map[string]float64{"train.cv_score": 0.8}, "ok"))
	if len(l.Rows) != 2 {
		t.Errorf("re-appending a seen (addr, fingerprint) must be a no-op, got %d rows", len(l.Rows))
	}
	// A new point-address appends.
	l.Append(row("cf1", "addr-c", map[string]any{"model": "logreg"}, map[string]float64{"train.cv_score": 0.85}, "ok"))
	if len(l.Rows) != 3 {
		t.Errorf("a new point-address must append, got %d rows", len(l.Rows))
	}
	// SAME point-address, DIFFERENT code fingerprint (same config, different code) → appends
	// (the metis#27 identity split — the exact collision this fixes; both must survive).
	l.Append(row("cf2", "addr-a", map[string]any{"model": "logreg"}, map[string]float64{"train.cv_score": 0.9}, "ok"))
	if len(l.Rows) != 4 {
		t.Errorf("same addr + new fingerprint must append (not overwrite), got %d rows", len(l.Rows))
	}
}

// The CSV is ragged: columns = the union of all rows' free-params + metrics, blank
// where a row lacks a key ($any-map: logreg rows blank n_estimators, rf rows blank C).
func TestCSV_RaggedRoundTrip(t *testing.T) {
	var l Ledger
	l.Append(
		row("sha1", "a1", map[string]any{"model": "logreg", "C": 1.0}, map[string]float64{"train.cv_score": 0.80}, "ok"),
		row("sha1", "a2", map[string]any{"model": "rf", "n_estimators": 300}, map[string]float64{"train.cv_score": 0.82}, "ok"),
		row("sha1", "a3", map[string]any{"model": "rf", "n_estimators": 100}, nil, "failed"),
	)
	csv, err := Encode(l)
	if err != nil {
		t.Fatal(err)
	}
	header := strings.SplitN(string(csv), "\n", 2)[0]
	// Union columns present (ragged): C (logreg-only) AND n_estimators (rf-only).
	for _, col := range []string{"fp.C", "fp.model", "fp.n_estimators", "metric.train.cv_score", "code_fingerprint", "point_addr", "status"} {
		if !strings.Contains(header, col) {
			t.Errorf("header missing union column %q: %s", col, header)
		}
	}
	// Round-trip.
	got, err := Decode(csv)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Rows) != 3 {
		t.Fatalf("round-trip row count = %d, want 3", len(got.Rows))
	}
	// The logreg row has no n_estimators (blank), the rf row has no C.
	byAddr := map[string]Row{}
	for _, r := range got.Rows {
		byAddr[r.PointAddr] = r
	}
	if _, has := byAddr["a1"].FreeParams["n_estimators"]; has {
		t.Error("logreg row must not carry n_estimators (ragged blank)")
	}
	if _, has := byAddr["a2"].FreeParams["C"]; has {
		t.Error("rf row must not carry C (ragged blank)")
	}
	if byAddr["a3"].Status != "failed" {
		t.Errorf("failed row status not preserved: %+v", byAddr["a3"])
	}
}

// List-valued free-params (kbench#4 sweeps `features: [[], [title], [title,family]]`)
// must round-trip as LISTS, not collapse to an unparseable string.
func TestCSV_ListFreeParamsRoundTrip(t *testing.T) {
	var l Ledger
	l.Append(row("s", "a", map[string]any{"features": []any{"title", "family"}}, nil, "ok"))
	b, err := Encode(l)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	feat, ok := got.Rows[0].FreeParams["features"].([]any)
	if !ok {
		t.Fatalf("features must round-trip as a list, got %T: %v", got.Rows[0].FreeParams["features"], got.Rows[0].FreeParams["features"])
	}
	if len(feat) != 2 || feat[0] != "title" || feat[1] != "family" {
		t.Errorf("list free-param corrupted on round-trip: %v", feat)
	}
	// Empty-list feature (the v0 baseline anchor) round-trips too.
	var l2 Ledger
	l2.Append(row("s", "b", map[string]any{"features": []any{}}, nil, "ok"))
	b2, _ := Encode(l2)
	g2, _ := Decode(b2)
	if f, ok := g2.Rows[0].FreeParams["features"].([]any); !ok || len(f) != 0 {
		t.Errorf("empty-list feature must round-trip as an empty list, got %#v", g2.Rows[0].FreeParams["features"])
	}
}

// TopN with a negative n returns empty (not a panic).
func TestTopN_NegativeN(t *testing.T) {
	var l Ledger
	l.Append(row("s", "a", nil, map[string]float64{"m": 0.5}, "ok"))
	if got := TopN(l, "m", "maximize", -1); len(got) != 0 {
		t.Errorf("TopN(-1) should return empty, got %d", len(got))
	}
}

// Best is objective-driven (maximize/minimize), skipping failed / metric-missing rows.
func TestBest_ObjectiveDriven(t *testing.T) {
	var l Ledger
	l.Append(
		row("s", "a", nil, map[string]float64{"train.cv_score": 0.80}, "ok"),
		row("s", "b", nil, map[string]float64{"train.cv_score": 0.88}, "ok"),
		row("s", "c", nil, map[string]float64{"train.cv_score": 0.95}, "failed"), // higher but failed → skip
		row("s", "d", nil, nil, "ok"),                                            // no metric → skip
	)
	best, ok := Best(l, "train.cv_score", "maximize")
	if !ok || best.PointAddr != "b" {
		t.Errorf("maximize best = %+v (ok=%v); want row b (0.88, the best OK row)", best, ok)
	}
	// minimize.
	var l2 Ledger
	l2.Append(
		row("s", "x", nil, map[string]float64{"loss": 0.3}, "ok"),
		row("s", "y", nil, map[string]float64{"loss": 0.1}, "ok"),
	)
	if b, _ := Best(l2, "loss", "minimize"); b.PointAddr != "y" {
		t.Errorf("minimize best = %s; want y (0.1)", b.PointAddr)
	}
	// Empty / all-skipped → not ok.
	if _, ok := Best(Ledger{}, "m", "maximize"); ok {
		t.Error("Best of an empty ledger must be !ok")
	}
}

// TopN returns the n best rows in objective order.
func TestTopN_Ordering(t *testing.T) {
	var l Ledger
	l.Append(
		row("s", "a", nil, map[string]float64{"m": 0.5}, "ok"),
		row("s", "b", nil, map[string]float64{"m": 0.9}, "ok"),
		row("s", "c", nil, map[string]float64{"m": 0.7}, "ok"),
	)
	top := TopN(l, "m", "maximize", 2)
	if len(top) != 2 || top[0].PointAddr != "b" || top[1].PointAddr != "c" {
		t.Errorf("TopN(2) = %v; want [b c] (0.9, 0.7)", []string{top[0].PointAddr, top[1].PointAddr})
	}
}

// SortAll ranks the qualified rows by the objective but KEEPS failed / metric-missing
// rows (appended) — a `show` view must not drop rows the way TopN (a leaderboard) does.
func TestSortAll_KeepsFailedRows(t *testing.T) {
	var l Ledger
	l.Append(
		row("s", "a", nil, map[string]float64{"m": 0.7}, "ok"),
		row("s", "bad", nil, nil, "failed"), // failed → kept, appended
		row("s", "b", nil, map[string]float64{"m": 0.9}, "ok"),
	)
	got := SortAll(l, "m", "maximize")
	if len(got) != 3 {
		t.Fatalf("SortAll must keep all 3 rows (incl. failed), got %d", len(got))
	}
	if got[0].PointAddr != "b" || got[1].PointAddr != "a" {
		t.Errorf("qualified rows should sort first (b, a); got %s, %s", got[0].PointAddr, got[1].PointAddr)
	}
	if got[2].PointAddr != "bad" {
		t.Errorf("the failed row must be kept (appended last); got %s", got[2].PointAddr)
	}
}

// foldRow builds a raw per-fold Row (metis#18) with the given fold coordinate.
func foldRow(fingerprint, addr string, fp map[string]any, fold int, score float64, status string) Row {
	f := fold
	return Row{CodeFingerprint: fingerprint, PointAddr: addr, FreeParams: fp, Fold: &f,
		Metrics: map[string]float64{"train.fold_score": score}, Status: status}
}

// metis#19: AggregateView means EVERY metric column (so train.complexity flows to the
// aggregate rows for the select rule), keeping .se/.n only for the objective metric.
func TestAggregateView_MeansAllMetrics(t *testing.T) {
	var l Ledger
	cfg := map[string]any{"train.model": "rf"}
	f0, f1 := 0, 1
	l.Append(
		Row{CodeFingerprint: "s", PointAddr: "r0", FreeParams: cfg, Fold: &f0, Status: "ok",
			Metrics: map[string]float64{"train.fold_score": 0.80, "train.complexity": 16}},
		Row{CodeFingerprint: "s", PointAddr: "r1", FreeParams: cfg, Fold: &f1, Status: "ok",
			Metrics: map[string]float64{"train.fold_score": 0.90, "train.complexity": 18}},
	)
	agg := AggregateView(l, "train.fold_score")
	if len(agg.Rows) != 1 {
		t.Fatalf("2 fold rows / 1 config → 1 aggregate row, got %d", len(agg.Rows))
	}
	m := agg.Rows[0].Metrics
	if got := m["train.fold_score"]; got < 0.8499 || got > 0.8501 {
		t.Errorf("objective mean = %v, want ~0.85", got)
	}
	if _, ok := m["train.fold_score.se"]; !ok {
		t.Errorf(".se present only for the objective metric; missing")
	}
	if got := m["train.complexity"]; got != 17 { // mean(16,18)
		t.Errorf("complexity mean = %v, want 17", got)
	}
	// a non-objective metric gets NO .se/.n (only its mean is meaningful for selection).
	if _, ok := m["train.complexity.se"]; ok {
		t.Errorf("non-objective metric should not carry .se")
	}
}

// AggregateView reduces raw per-fold rows → one per-config (mean, SE, n) row, grouping by
// (free-params, sweep-SHA). The read-time reduction the leaderboard + promote sort over.
func TestAggregateView_ReducesPerConfig(t *testing.T) {
	var l Ledger
	cfgA := map[string]any{"model": "a"}
	cfgB := map[string]any{"model": "b"}
	l.Append(
		foldRow("s", "a0", cfgA, 0, 0.80, "ok"),
		foldRow("s", "a1", cfgA, 1, 0.90, "ok"),
		foldRow("s", "b0", cfgB, 0, 0.70, "ok"),
		foldRow("s", "b1", cfgB, 1, 0.70, "ok"),
	)
	agg := AggregateView(l, "train.fold_score")
	if len(agg.Rows) != 2 {
		t.Fatalf("4 fold rows over 2 configs → 2 aggregate rows, got %d", len(agg.Rows))
	}
	byModel := map[any]Row{}
	for _, r := range agg.Rows {
		if r.Fold != nil {
			t.Errorf("an aggregate row must NOT carry a fold coordinate: %+v", r)
		}
		byModel[r.FreeParams["model"]] = r
	}
	// config a: mean(0.80,0.90)=0.85, n=2, SE = sd/√2 = (0.0707.../√2); config b: mean=0.70, SE=0.
	if got := byModel["a"].Metrics["train.fold_score"]; got < 0.8499 || got > 0.8501 {
		t.Errorf("config a mean = %v, want ~0.85", got)
	}
	if got := byModel["a"].Metrics["train.fold_score.n"]; got != 2 {
		t.Errorf("config a fold count = %v, want 2", got)
	}
	if got := byModel["a"].Metrics["train.fold_score.se"]; got <= 0 {
		t.Errorf("config a SE = %v, want > 0 (folds differ)", got)
	}
	if got := byModel["b"].Metrics["train.fold_score.se"]; got != 0 {
		t.Errorf("config b SE = %v, want 0 (identical folds)", got)
	}
}

// A group with ANY failed fold is marked failed in its aggregate row.
func TestAggregateView_FailedFoldMarksConfigFailed(t *testing.T) {
	var l Ledger
	cfg := map[string]any{"model": "a"}
	l.Append(
		foldRow("s", "a0", cfg, 0, 0.80, "ok"),
		foldRow("s", "a1", cfg, 1, 0, "failed"),
	)
	agg := AggregateView(l, "train.fold_score")
	if len(agg.Rows) != 1 || agg.Rows[0].Status != "failed" {
		t.Errorf("a config with a failed fold must aggregate to a failed row, got %+v", agg.Rows)
	}
}

// A row with no fold coordinate (Fold==nil) passes through unchanged — the view is idempotent.
func TestAggregateView_NonFoldRowPassesThrough(t *testing.T) {
	var l Ledger
	l.Append(row("s", "plain", map[string]any{"model": "a"}, map[string]float64{"train.fold_score": 0.9}, "ok"))
	agg := AggregateView(l, "train.fold_score")
	if len(agg.Rows) != 1 || agg.Rows[0].PointAddr != "plain" {
		t.Errorf("a non-fold row must pass through unchanged, got %+v", agg.Rows)
	}
	// Idempotent: aggregating an already-aggregated view is a no-op fixpoint.
	agg2 := AggregateView(AggregateView(l, "train.fold_score"), "train.fold_score")
	if len(agg2.Rows) != 1 {
		t.Errorf("AggregateView must be idempotent on non-fold rows, got %d", len(agg2.Rows))
	}
}

// Filter selects rows by code-fingerprint (a code-version view).
func TestFilter_ByFingerprint(t *testing.T) {
	var l Ledger
	l.Append(
		row("cf1", "a", nil, nil, "ok"),
		row("cf2", "b", nil, nil, "ok"),
		row("cf1", "c", nil, nil, "ok"),
	)
	got := Filter(l, "cf1")
	if len(got.Rows) != 2 {
		t.Errorf("filter by cf1 = %d rows, want 2", len(got.Rows))
	}
}
