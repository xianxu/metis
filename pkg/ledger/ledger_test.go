package ledger

import (
	"strings"
	"testing"
)

func row(sha, addr string, fp map[string]any, metrics map[string]float64, status string) Row {
	return Row{SweepSHA: sha, PointAddr: addr, FreeParams: fp, Metrics: metrics, Status: status}
}

// Append is append-only + dedups by point-address: re-appending a seen address is a
// no-op; a new address (e.g. a new code-version's rows) appends.
func TestAppend_DedupByPointAddress(t *testing.T) {
	var l Ledger
	l.Append(
		row("sha1", "addr-a", map[string]any{"model": "logreg"}, map[string]float64{"train.cv_score": 0.8}, "ok"),
		row("sha1", "addr-b", map[string]any{"model": "rf"}, map[string]float64{"train.cv_score": 0.82}, "ok"),
	)
	if len(l.Rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(l.Rows))
	}
	// Re-append the same addresses (idempotent re-run) → no growth.
	l.Append(row("sha1", "addr-a", map[string]any{"model": "logreg"}, map[string]float64{"train.cv_score": 0.8}, "ok"))
	if len(l.Rows) != 2 {
		t.Errorf("re-appending a seen point-address must be a no-op, got %d rows", len(l.Rows))
	}
	// A new code-version's row (new address) appends.
	l.Append(row("sha2", "addr-a2", map[string]any{"model": "logreg"}, map[string]float64{"train.cv_score": 0.85}, "ok"))
	if len(l.Rows) != 3 {
		t.Errorf("a new point-address must append, got %d rows", len(l.Rows))
	}
}

// The CSV is ragged: columns = the union of all rows' free-params + metrics, blank
// where a row lacks a key ($oneof: logreg rows blank n_estimators, rf rows blank C).
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
	for _, col := range []string{"fp.C", "fp.model", "fp.n_estimators", "metric.train.cv_score", "sweep_sha", "point_addr", "status"} {
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

// Filter selects rows by sweep-SHA (an invocation view).
func TestFilter_BySweepSHA(t *testing.T) {
	var l Ledger
	l.Append(
		row("sha1", "a", nil, nil, "ok"),
		row("sha2", "b", nil, nil, "ok"),
		row("sha1", "c", nil, nil, "ok"),
	)
	got := Filter(l, "sha1")
	if len(got.Rows) != 2 {
		t.Errorf("filter by sha1 = %d rows, want 2", len(got.Rows))
	}
}
