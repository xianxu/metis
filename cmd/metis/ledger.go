package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
// (idempotent — dedups by point-address) and regenerates the body top-N summary. Called
// by runSweep after the manifest is written.
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
	return regenLedgerSummary(shapePath, led, objective)
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

// regenLedgerSummary rewrites the shape body's generated top-N block (between markers)
// with the objective-ranked leaders + a pointer to the sidecar.
func regenLedgerSummary(shapePath string, led ledger.Ledger, obj experiment.Objective) error {
	raw, err := os.ReadFile(shapePath)
	if err != nil {
		return err
	}
	const begin, end = "<!-- metis:ledger:begin -->", "<!-- metis:ledger:end -->"
	var b strings.Builder
	b.WriteString(begin + "\n## Top runs\n")
	if obj.Metric != "" {
		fmt.Fprintf(&b, "By `%s` (%s) — see `%s` for all rows.\n\n", obj.Metric, obj.Direction, filepath.Base(ledgerPath(shapePath)))
		for i, r := range ledger.TopN(led, obj.Metric, obj.Direction, 10) {
			fmt.Fprintf(&b, "%d. %s = %g — %s\n", i+1, obj.Metric, r.Metrics[obj.Metric], freeParamTuple(r))
		}
	} else {
		fmt.Fprintf(&b, "%d rows — see `%s`.\n", len(led.Rows), filepath.Base(ledgerPath(shapePath)))
	}
	b.WriteString("\n" + end)

	body := string(raw)
	if i := strings.Index(body, begin); i >= 0 {
		if j := strings.Index(body, end); j > i {
			body = body[:i] + b.String() + body[j+len(end):]
			return os.WriteFile(shapePath, []byte(body), 0o644)
		}
	}
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return os.WriteFile(shapePath, []byte(body+"\n"+b.String()+"\n"), 0o644)
}

// freeParamTuple renders a row's free-params as a compact `(k=v, …)` human key.
func freeParamTuple(r ledger.Row) string {
	keys := make([]string, 0, len(r.FreeParams))
	for k := range r.FreeParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s=%v", k, r.FreeParams[k])
	}
	return "(" + strings.Join(parts, ", ") + ")"
}
