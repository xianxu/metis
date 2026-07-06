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
// each row's metrics are NAMESPACED per step (train.cv_score, not a flat cv_score that
// v0's merge collided). Keys: point-address (manifest run-id) + sweep-SHA (the
// manifest's repo SHA, read from any point's record).
func rowsFromManifest(man sweepManifest, records map[string]record.RunRecord) []ledger.Row {
	rows := make([]ledger.Row, 0, len(man.Points))
	for _, p := range man.Points {
		rec := records[p.RunID]
		rows = append(rows, ledger.Row{
			FreeParams: p.FreeParams,
			SweepSHA:   sweepSHAOf(rec),
			PointAddr:  p.RunID,
			Metrics:    namespacedMetrics(rec),
			Status:     p.Status,
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

// sweepSHAOf returns the code-version SHA from a record's repo-SHAs (the single repo in
// v1). Empty if none (a no-git run).
func sweepSHAOf(rec record.RunRecord) string {
	for _, sha := range rec.RepoSHAs {
		return sha
	}
	return ""
}

// promotedExperiment reconstructs the all-singleton experiment for a ledger row by
// EXPANDING the shape and matching the point whose free-params equal the row's — PURE,
// no repo. This inverts the sweep by re-derivation (reusing shape.Expand +
// shapePointToExperiment), not a fragile flat-map inversion: the free-param tuple is the
// human key that uniquely identifies a point within the shape (metis#6/#8 design).
func promotedExperiment(sh experiment.Shape, freeParams map[string]any) (experiment.Experiment, error) {
	points, err := shape.Expand(sh.Steps, sh.Sweep.RangeSteps)
	if err != nil {
		return experiment.Experiment{}, err
	}
	for _, p := range points {
		if freeParamsEqual(p, freeParams) {
			return shapePointToExperiment(sh, p), nil
		}
	}
	return experiment.Experiment{}, fmt.Errorf("no point in shape %q matches free-params %v", sh.ID, freeParams)
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

// writeSweepLedger appends a finished sweep's rows to the shape's ledger sidecar
// (idempotent — dedups by point-address). Called by runSweep after the manifest is
// written. It does NOT touch the experiment .md (#13 — the config is immutable input);
// the human top-N view is on-demand via `metis ledger show` over the sidecar.
func writeSweepLedger(shapePath string, man sweepManifest, objective experiment.Objective) error {
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
	if err := os.WriteFile(ledgerPath(shapePath), b, 0o644); err != nil {
		return err
	}
	warnIfObjectiveMissing(led, objective)
	return nil
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

// warnIfObjectiveMissing surfaces the one genuinely-useful check the old body-summary
// carried: a sweep whose objective metric matches NO ledger row — almost always a
// namespacing mistake (rows carry `<step>.<metric>`, e.g. train.cv_score). It's loud,
// not a silently-empty result. (#13: the top-N summary is no longer written into the
// experiment .md — the config is immutable input; the human view is `metis ledger show`.)
func warnIfObjectiveMissing(led ledger.Ledger, obj experiment.Objective) {
	if obj.Metric == "" || len(led.Rows) == 0 {
		return
	}
	if len(ledger.TopN(led, obj.Metric, obj.Direction, 10)) == 0 {
		fmt.Fprintf(os.Stderr, "metis: warning: objective metric %q is not present in any ledger row — metrics are namespaced `<step>.<metric>` (e.g. train.%s)\n", obj.Metric, obj.Metric)
	}
}

// freeParamTuple renders a row's free-params as a compact `(k=v, …)` human key
// (delegates to freeParamTupleMap over the row's free-param map — one renderer).
func freeParamTuple(r ledger.Row) string {
	return freeParamTupleMap(r.FreeParams)
}
