package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/ledger"
	"github.com/xianxu/metis/pkg/record"
	"github.com/xianxu/metis/pkg/shape"
)

// rowsFromManifest turns a sweep manifest + the per-point records into ledger rows —
// PURE (the caller reads the record.json files). The metric collision fix lives here:
// each row's metrics are NAMESPACED per step (train.fold_score, not a flat fold_score that
// v0's merge collided). Keys: point-address (manifest run-id) + code-fingerprint (the
// realized code identity, read from the point's record — metis#27).
func rowsFromManifest(man sweepManifest, records map[string]record.RunRecord) []ledger.Row {
	rows := make([]ledger.Row, 0, len(man.Points))
	for _, p := range man.Points {
		rec := records[p.RunID]
		fold := p.Fold // fresh per-iteration var → &fold is the row's own fold coordinate
		rows = append(rows, ledger.Row{
			FreeParams:      p.FreeParams,
			CodeFingerprint: string(rec.CodeFingerprint), // metis#27: the realized code identity
			PointAddr:       p.RunID,
			Fold:            &fold, // metis#18: a RAW per-fold row (AggregateView reduces read-time)
			Metrics:         namespacedMetrics(rec),
			Status:          p.Status,
		})
	}
	return rows
}

// namespacedMetrics flattens a record's per-step metrics into `<step>.<metric>` — the
// unambiguous names that fix v0's flat last-write-wins collision.
func namespacedMetrics(rec record.RunRecord) map[string]float64 {
	m := map[string]float64{}
	for _, st := range rec.Steps {
		for k, v := range st.Metrics {
			m[st.StepID+"."+k] = v
		}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// promotedExperiment reconstructs the winner's runnable experiment for a ledger row by
// EXPANDING the pipeline config-space and matching the config whose free-params equal the
// row's — PURE, no repo. This inverts the sweep by re-derivation (reusing shape.Expand +
// shapeConfigToExperiment), not a fragile flat-map inversion: the free-param tuple is the
// human key that uniquely identifies a config within the shape (metis#6/#8 design).
func promotedExperiment(sh experiment.Shape, freeParams map[string]any) (experiment.Experiment, error) {
	configs, err := shape.Expand(sh.Pipeline, 0)
	if err != nil {
		return experiment.Experiment{}, err
	}
	for _, c := range configs {
		if freeParamsEqual(c, freeParams) {
			return shapeConfigToExperiment(sh, c), nil
		}
	}
	return experiment.Experiment{}, fmt.Errorf("no config in shape %q matches free-params %v", sh.ID, freeParams)
}

// shapeConfigToExperiment reconstructs the winner's runnable experiment for a config point:
// data ++ pipeline (config overlaid, NO fold-context → the all-rows/ship path train.py takes
// when `_fold` is absent) ++ ship. The metis#18 promote/ship reconstruction; the per-fold
// experiment is buildFoldExperiment. (M1a-5 injects the `{mode:all}` ship signal here.)
func shapeConfigToExperiment(sh experiment.Shape, c shape.Point) experiment.Experiment {
	steps := make([]experiment.Step, 0, len(sh.Data)+len(sh.Pipeline)+len(sh.Ship))
	steps = append(steps, sh.Data...)
	for _, ps := range sh.Pipeline {
		s := ps
		s.With = c.With[ps.ID]
		steps = append(steps, s)
	}
	steps = append(steps, sh.Ship...)
	exp := experiment.Experiment{Header: sh.Header, Steps: steps}
	exp.Type = "experiment"
	return exp
}

// freeParamsEqual reports whether an expanded point's free-param path equals the row's
// {path: value} map. Compares via canonical JSON so it is tolerant to the int-vs-float64
// drift a CSV/JSON round-trip introduces (a row's `300` may be int or float64; the
// point's is yaml-int) — both marshal to `300`, and lists compare structurally.
func freeParamsEqual(p shape.Point, want map[string]any) bool {
	gb, err1 := json.Marshal(freeParamMap(p)) // reuses the sweep driver's renderer
	wb, err2 := json.Marshal(want)
	return err1 == nil && err2 == nil && bytes.Equal(gb, wb)
}

// ── IO: the ledger sidecar + `metis ledger show` + `metis promote` ──────────────

// ledgerPath is the shape's append-only CSV sidecar (`<shape>.ledger.csv`).
func ledgerPath(shapePath string) string {
	base := strings.TrimSuffix(shapePath, filepath.Ext(shapePath))
	return base + ".ledger.csv"
}

// writeSweepLedger appends a finished sweep's RAW per-fold rows to the shape's ledger
// sidecar (idempotent — dedups by point-address). Called by runShapeSweep after the
// manifest is written. It does NOT touch the experiment .md (#13 — the config is immutable
// input); the human per-config (mean,SE) view is on-demand via `metis ledger show` (which
// AggregateView-reduces the raw rows).
func writeSweepLedger(shapePath string, man sweepManifest) error {
	records, err := loadSweepRecords(shapePath, man)
	if err != nil {
		return err
	}
	led, err := loadLedger(shapePath)
	if err != nil {
		return err
	}
	led.Append(rowsFromManifest(man, records)...)
	b, err := ledger.Encode(led)
	if err != nil {
		return err
	}
	return os.WriteFile(ledgerPath(shapePath), b, 0o644)
}

// loadLedger reads the shape's ledger sidecar (absent → empty).
func loadLedger(shapePath string) (ledger.Ledger, error) {
	b, err := os.ReadFile(ledgerPath(shapePath))
	if os.IsNotExist(err) {
		return ledger.Ledger{}, nil
	}
	if err != nil {
		return ledger.Ledger{}, err
	}
	return ledger.Decode(b)
}

// loadSweepRecords reads each point-run's record.json (per-step metrics + repo SHA).
func loadSweepRecords(shapePath string, man sweepManifest) (map[string]record.RunRecord, error) {
	dir := filepath.Dir(shapePath)
	out := make(map[string]record.RunRecord, len(man.Points))
	for _, p := range man.Points {
		b, err := os.ReadFile(filepath.Join(dir, "runs", p.RunID, "record.json"))
		if err != nil {
			continue // a point with no record (shouldn't happen post-run) → metrics blank
		}
		var rec record.RunRecord
		if err := json.Unmarshal(b, &rec); err != nil {
			return nil, fmt.Errorf("parse record %s: %w", p.RunID, err)
		}
		out[p.RunID] = rec
	}
	return out, nil
}

// freeParamTuple renders a row's free-params as a compact `(k=v, …)` human key
// (delegates to freeParamTupleMap over the row's free-param map — one renderer).
func freeParamTuple(r ledger.Row) string {
	return freeParamTupleMap(r.FreeParams)
}
