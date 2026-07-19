package main

import (
	"reflect"
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/ledger"
	"github.com/xianxu/metis/pkg/record"

	"github.com/xianxu/metis/pkg/shape"
)

// rowsFromManifest is pure: it turns a sweep manifest + the per-point records into
// ledger rows (namespaced per-step metrics from the records; sweep-SHA + point-address
// from the manifest). Unit-testable without disk.
func TestRowsFromManifest_NamespacedMetrics(t *testing.T) {
	man := sweepManifest{
		ShapeRunID: "srun1", Shape: "titanic", Seed: 42,
		Points: []pointRun{
			{RunID: "addr-a", FreeParams: map[string]any{"train.model": "logreg"}, Status: "ok"},
			{RunID: "addr-b", FreeParams: map[string]any{"train.model": "rf"}, Status: "failed"},
		},
	}
	// Per-point records carry per-STEP metrics (the namespacing fix — train.cv_score,
	// not a flat cv_score that would collide across steps).
	records := map[string]record.RunRecord{
		"addr-a": {PointAddress: "addr-a", CodeFingerprint: "cf1", Steps: []record.StepRecord{
			{StepID: "train", Metrics: map[string]float64{"cv_score": 0.81}},
		}},
		"addr-b": {PointAddress: "addr-b", CodeFingerprint: "cf1", Steps: []record.StepRecord{}},
	}
	rows := rowsFromManifest(man, records)
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if rows[0].PointAddr != "addr-a" || rows[0].CodeFingerprint != "cf1" {
		t.Errorf("row 0 keys wrong: %+v", rows[0])
	}
	if rows[0].Metrics["train.cv_score"] != 0.81 {
		t.Errorf("metric should be NAMESPACED train.cv_score=0.81; got %v", rows[0].Metrics)
	}
	if rows[1].Status != "failed" {
		t.Errorf("failed point should carry failed status; got %+v", rows[1])
	}
}

// promotedExperiment reconstructs the winner's experiment for a row by matching its
// free-params against the shape's expanded PIPELINE configs — pure, no repo. Reuses
// shape.Expand(sh.Pipeline) + shapeConfigToExperiment (metis#18 phase model; no fragile
// expand-inversion).
func TestPromotedExperiment_MatchesByFreeParams(t *testing.T) {
	sh := experiment.Shape{
		Header: experiment.Header{Type: "experiment-shape", ID: "titanic", Seed: 42, Status: "active"},
		Data:   []experiment.Step{{ID: "adapt", Uses: "titanic/adapt", With: map[string]any{"out": "../data/x"}}},
		Pipeline: []experiment.Step{
			{ID: "train", Uses: "metis/train", Needs: []string{"adapt"}, With: map[string]any{
				"model": map[string]any{"$any": []any{"logreg", "rf"}},
				"fixed": "keep",
			}},
		},
		Ship: []experiment.Step{{ID: "predict", Uses: "metis/predict", Needs: []string{"train"}}},
	}
	exp, err := promotedExperiment(sh, map[string]any{"train.model": "rf"})
	if err != nil {
		t.Fatal(err)
	}
	if exp.Type != "experiment" {
		t.Errorf("promoted should be a plain experiment, got %q", exp.Type)
	}
	// The reconstruction threads all three phases: data (adapt) ++ pipeline (train) ++ ship
	// (predict). Find the train step and confirm it pinned model=rf + kept the fixed leaf.
	var tw map[string]any
	ids := make([]string, len(exp.Steps))
	for i, s := range exp.Steps {
		ids[i] = s.ID
		if s.ID == "train" {
			tw = s.With
		}
	}
	if len(exp.Steps) != 3 || ids[0] != "adapt" || ids[2] != "predict" {
		t.Errorf("promoted experiment should be data++pipeline++ship (adapt, train, predict); got %v", ids)
	}
	if tw["model"] != "rf" {
		t.Errorf("promoted train.model should be the pinned $any value 'rf'; got %#v", tw["model"])
	}
	if tw["fixed"] != "keep" {
		t.Errorf("fixed leaf must be preserved; got %v", tw["fixed"])
	}
	// A non-existent free-param set errors (no matching config).
	if _, err := promotedExperiment(sh, map[string]any{"train.model": "nope"}); err == nil {
		t.Error("promotedExperiment must error when no config matches the free-params")
	}
}

// The row's free-params round-trip through the CSV as the same values the shape's
// points carry (so match-by-free-params works after a Decode).
func TestPromote_RowFreeParamsMatchPoint(t *testing.T) {
	want := map[string]any{"train.model": "logreg"}
	var l ledger.Ledger
	l.Append(ledger.Row{PointAddr: "a", FreeParams: want, Status: "ok",
		Metrics: map[string]float64{"train.cv_score": 0.8}})
	b, err := ledger.Encode(l)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ledger.Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Rows[0].FreeParams, want) {
		t.Errorf("free-params must round-trip for match-by-free-params: got %v want %v", got.Rows[0].FreeParams, want)
	}
}

// ── metis#64: null free-params vs the CSV round-trip ─────────────────────────

func TestFreeParamsEqual_NullEqualsAbsent(t *testing.T) {
	// A cw=null rung: the expanded point carries an explicit nil; the CSV round-trip
	// (empty cell, skipped at decode) leaves the row's map KEY-ABSENT. Must match.
	p := shape.Point{FreeParams: []shape.FreeParam{
		{Path: "train.model", Value: "hist_gbm"},
		{Path: "train.model.hist_gbm.class_weight", Value: nil},
	}}
	if !freeParamsEqual(p, map[string]any{"train.model": "hist_gbm"}) {
		t.Error("nil-valued free param must equal key-absent (the CSV round-trip)")
	}
	// Distinct configs stay unequal: a different value on the surviving key.
	if freeParamsEqual(p, map[string]any{"train.model": "rf"}) {
		t.Error("distinct configs must not match")
	}
	// And a NON-null param must still require presence.
	p2 := shape.Point{FreeParams: []shape.FreeParam{
		{Path: "train.model", Value: "hist_gbm"},
		{Path: "train.model.hist_gbm.class_weight", Value: "balanced"},
	}}
	if freeParamsEqual(p2, map[string]any{"train.model": "hist_gbm"}) {
		t.Error("a real (non-null) param must not be droppable")
	}
}

func TestFreeParamStr_CompositeValuesRenderAsJSON(t *testing.T) {
	s := freeParamStrFromParams([]shape.FreeParam{
		{Path: "train.decide", Value: map[string]any{"offsets": map[string]any{"holdout": 0.2}}},
	})
	want := `train.decide={"offsets":{"holdout":0.2}}`
	if s != want {
		t.Errorf("composite free-param rendering: got %q, want %q", s, want)
	}
}
