package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

)

// TestNestedCV_InnerKSplit (metis#45 lever (a)): a shape declaring inner_k ≠ k runs the
// OUTER level at k (estimand: split dirs, driver, held-out scoring partition) and the INNER
// per-config CV at inner_k — asserted on the banner, the ledger's per-fold rows, the
// outer-split dirs, and the held-out scoring run's recorded cv-split k (the leakage tooth).
func TestNestedCV_InnerKSplit(t *testing.T) {
	ws := t.TempDir()
	shape := strings.Replace(foldShapeCVMD("[a, b]"),
		"resample: {cv: {k: 2, stratify: false}}",
		"resample: {cv: {k: 2, inner_k: 3, stratify: false}}", 1)
	expPath := writeShapeFile(t, ws, shape)

	var out strings.Builder
	if err := runFoldSweep(t, expPath, false, nil, &out, nil); err != nil {
		t.Fatalf("nested inner_k run: %v", err)
	}

	// (i) banner: outer 2, inner 3
	if !strings.Contains(out.String(), "2 outer fold(s) × (2 configs × 3 inner folds)") {
		t.Errorf("banner must show outer k=2 / inner_k=3; got:\n%s", out.String())
	}

	led, err := loadLedger(expPath)
	if err != nil {
		t.Fatal(err)
	}
	// (ii) INNER rows: per (config, outer fold) exactly folds {0,1,2}
	type cfKey struct {
		model string
		outer int
	}
	innerFolds := map[cfKey]map[int]bool{}
	outerFolds := map[int]bool{}
	for _, r := range led.Rows {
		switch r.Level {
		case "inner":
			k := cfKey{model: r.FreeParams["train.model"].(string), outer: *r.OuterFold}
			if innerFolds[k] == nil {
				innerFolds[k] = map[int]bool{}
			}
			innerFolds[k][*r.Fold] = true
		case "outer":
			outerFolds[*r.OuterFold] = true
		}
	}
	if len(innerFolds) != 4 { // 2 configs × 2 outer folds
		t.Fatalf("want 4 (config, outer) groups, got %d", len(innerFolds))
	}
	for k, folds := range innerFolds {
		if len(folds) != 3 || !folds[0] || !folds[1] || !folds[2] {
			t.Errorf("(%s, outer %d): inner folds = %v, want {0,1,2}", k.model, k.outer, folds)
		}
	}
	// (iii) OUTER rows at k=2 only
	if len(outerFolds) != 2 || !outerFolds[0] || !outerFolds[1] {
		t.Errorf("outer folds = %v, want {0,1}", outerFolds)
	}
	// (iv) the outer-split preamble ran at OUTER k=2: its run recorded with.k == 2 (under
	// the fake exec the analysis dirs aren't physically materialized — the recorded split
	// config is the ground truth; a dir-glob would be vacuous here)
	splitChecked := false
	recs, _ := filepath.Glob(filepath.Join(ws, "runs", "*", "record.json"))
	for _, rp := range recs {
		b, err := os.ReadFile(rp)
		if err != nil {
			continue
		}
		var rec struct {
			Steps []struct {
				Uses string         `json:"uses"`
				With map[string]any `json:"with"`
			} `json:"steps"`
		}
		if err := json.Unmarshal(b, &rec); err != nil {
			continue
		}
		for _, st := range rec.Steps {
			if st.Uses != "metis/outer-split" {
				continue
			}
			splitChecked = true
			if k, _ := st.With["k"].(float64); k != 2 { // JSON round-trip: numbers are float64
				t.Errorf("outer-split must run at OUTER k=2, got %v (%s)", st.With["k"], rp)
			}
		}
	}
	if !splitChecked {
		t.Error("no outer-split record found — assertion (iv) vacuous")
	}
	// (iii cont.) the leakage tooth: every OUTER row's scoring run recorded cv-split with.k
	// == 2 (the OUTER partition — inner_k must not touch the held-out reproduction).
	// JSON round-trip: with.k decodes as float64.
	checked := 0
	for _, r := range led.Rows {
		if r.Level != "outer" {
			continue
		}
		recPath := filepath.Join(ws, "runs", r.PointAddr, "record.json")
		b, err := os.ReadFile(recPath)
		if err != nil {
			continue // not every addr materializes a record under the fake; check what exists
		}
		var rec struct {
			Steps []struct {
				Uses string         `json:"uses"`
				With map[string]any `json:"with"`
			} `json:"steps"`
		}
		if err := json.Unmarshal(b, &rec); err != nil {
			t.Fatalf("record %s: %v", recPath, err)
		}
		for _, st := range rec.Steps {
			if st.Uses != "metis/cv-split" {
				continue
			}
			checked++
			if k, _ := st.With["k"].(float64); k != 2 { // float64 after the JSON round-trip
				t.Errorf("outer scoring run %s cv-split must stay at OUTER k=2, got %v", r.PointAddr, st.With["k"])
			}
		}
	}
	if checked == 0 {
		t.Error("leakage tooth vacuous: no outer scoring record with a cv-split step was checked")
	}
}

// TestFlatCV_InnerKIgnoredLoudly (plan-review Important 3): a FLAT (single-config) run's CV
// IS the reported estimate — inner_k must NOT change the estimand; it is ignored with ONE
// loud note, and the CV runs at k.
func TestFlatCV_InnerKIgnoredLoudly(t *testing.T) {
	ws := t.TempDir()
	shape := strings.Replace(foldShapeCVMD("[a]"), // 1 config → flat mode
		"resample: {cv: {k: 2, stratify: false}}",
		"resample: {cv: {k: 2, inner_k: 3, stratify: false}}", 1)
	expPath := writeShapeFile(t, ws, shape)

	var out strings.Builder
	if err := runFoldSweep(t, expPath, false, nil, &out, nil); err != nil {
		t.Fatalf("flat inner_k run: %v", err)
	}
	if n := strings.Count(out.String(), "inner_k ignored"); n != 1 {
		t.Errorf("flat run must note the inert knob exactly once, got %d:\n%s", n, out.String())
	}
	led, err := loadLedger(expPath)
	if err != nil {
		t.Fatal(err)
	}
	folds := map[int]bool{}
	for _, r := range led.Rows {
		if r.Fold != nil {
			folds[*r.Fold] = true
		}
	}
	if len(folds) != 2 {
		t.Errorf("flat CV must run at k=2 (the estimand), got folds %v", folds)
	}
}
