package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── pure combine (metis#60 M2) ───────────────────────────────────────────────

func TestBlendCombine_TiltedLogFlipsBoundaryRow(t *testing.T) {
	// Row 0: both members confident in a → a. Row 1: boundary — the plain log-average
	// picks a, but member 2's tuned offset on b tilts the blend to b (the whole point
	// of averaging in TILTED log-space: each member's decision layer carries through).
	cols := []string{"0", "1"}
	m1 := blendMember{id: "m1", proba: [][]float64{{0.9, 0.1}, {0.55, 0.45}}, offsets: []float64{0, 0}}
	m2 := blendMember{id: "m2", proba: [][]float64{{0.8, 0.2}, {0.52, 0.48}}, offsets: []float64{0, 0.5}}

	got, err := blendCombine(cols, []blendMember{m1, m2}, []float64{0.5, 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != "0" || got[1] != "1" {
		t.Errorf("tilted blend = %v, want [0 1] (offset flips the boundary row)", got)
	}

	// Without member 2's offset the same boundary row stays with a.
	m2.offsets = []float64{0, 0}
	got, err = blendCombine(cols, []blendMember{m1, m2}, []float64{0.5, 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if got[1] != "0" {
		t.Errorf("un-tilted blend row 1 = %q, want 0 (no flip without the offset)", got[1])
	}
}

func TestBlendCombine_SingleMemberIdentityPinsClip(t *testing.T) {
	// One member at weight 1 ≡ apply_offsets semantics: argmax(log(clip(p, 1e-12)) + o).
	// The zero-probability cell pins the clip constant: log(1e-12) ≈ −27.63, so an offset
	// of +30 rescues class 0 IFF the clip is 1e-12 (a 1e-20 clip would give −46+30 < 0).
	cols := []string{"0", "1"}
	m := blendMember{id: "m", proba: [][]float64{{0.0, 1.0}}, offsets: []float64{30, 0}}
	got, err := blendCombine(cols, []blendMember{m}, []float64{1})
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != "0" {
		t.Errorf("single-member identity with clip 1e-12: got %q, want 0", got[0])
	}
}

func TestBlendCombine_MismatchedShapesLoud(t *testing.T) {
	cols := []string{"0", "1"}
	a := blendMember{id: "a", proba: [][]float64{{0.5, 0.5}}, offsets: []float64{0, 0}}
	b := blendMember{id: "b", proba: [][]float64{{0.5, 0.5}, {0.5, 0.5}}, offsets: []float64{0, 0}}
	if _, err := blendCombine(cols, []blendMember{a, b}, []float64{0.5, 0.5}); err == nil {
		t.Error("row-count mismatch must be loud")
	}
	c := blendMember{id: "c", proba: [][]float64{{1.0}}, offsets: []float64{0}}
	if _, err := blendCombine(cols, []blendMember{a, c}, []float64{0.5, 0.5}); err == nil {
		t.Error("column-count mismatch must be loud")
	}
}

func TestNormalizeWeights(t *testing.T) {
	w, err := normalizeWeights(3, nil) // default: equal
	if err != nil || len(w) != 3 || math.Abs(w[0]-1.0/3) > 1e-12 {
		t.Fatalf("default weights = %v, %v", w, err)
	}
	w, err = normalizeWeights(2, []float64{1, 3})
	if err != nil || math.Abs(w[0]-0.25) > 1e-12 || math.Abs(w[1]-0.75) > 1e-12 {
		t.Fatalf("normalized = %v, %v", w, err)
	}
	if _, err := normalizeWeights(2, []float64{1}); err == nil {
		t.Error("count mismatch must be loud")
	}
	if _, err := normalizeWeights(2, []float64{1, 0}); err == nil {
		t.Error("non-positive weight must be loud")
	}
}

func TestBlendID_SensitiveToMembersAndWeights(t *testing.T) {
	a := blendID([]string{"m1", "m2"}, []float64{0.5, 0.5})
	b := blendID([]string{"m1", "m2"}, []float64{0.25, 0.75})
	c := blendID([]string{"m1", "m3"}, []float64{0.5, 0.5})
	if a == b || a == c {
		t.Errorf("blend ids must differ by weights and members: %s %s %s", a, b, c)
	}
	if !strings.HasPrefix(a, "blend-") {
		t.Errorf("id %q must be blend-*", a)
	}
}

// ── end-to-end: materialization + submission-step exec + guards ──────────────

const blendShapeMD = `---
type: experiment-shape
id: blendtest
seed: 7
data:
  - id: get-data
    uses: kaggle/download
    with: {competition: {slug: test-comp}}
pipeline:
  - id: train
    uses: metis/train
    with:
      dataset: ../d
      model: {$any: {logreg: {C: {$any: [1.0, 2.0]}}}}
ship:
  - id: predict
    uses: metis/predict
    with: {dataset: ../d, model: train}
  - id: submission
    uses: toy/submission
    with: {predictions: predict}
---
`

// blendWS lays out a tmp workspace: pipelines/blendtest.md, a toy submission step
// (copies the predict predictions into its own step dir as submission.csv), and two
// member runs with probabilities + records. Returns (shapePath, runsDir, stepsDir).
func blendWS(t *testing.T) (string, string, []string) {
	t.Helper()
	ws := t.TempDir()
	pipes := filepath.Join(ws, "pipelines")
	if err := os.MkdirAll(pipes, 0o755); err != nil {
		t.Fatal(err)
	}
	shapePath := filepath.Join(pipes, "blendtest.md")
	if err := os.WriteFile(shapePath, []byte(blendShapeMD), 0o644); err != nil {
		t.Fatal(err)
	}
	stepDir := filepath.Join(ws, "steps", "toy")
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\ncp \"$METIS_RUN_DIR/predict/predictions.csv\" \"$METIS_STEP_DIR/submission.csv\"\n"
	if err := os.WriteFile(filepath.Join(stepDir, "submission"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	runsDir := filepath.Join(pipes, "runs")
	writeMember(t, runsDir, "m1", "fp1",
		"id,proba_0,proba_1\n10,0.9,0.1\n11,0.55,0.45\n",
		`{"offsets": [0, 0.5], "classes": [0, 1], "rule": "offsets"}`)
	writeMember(t, runsDir, "m2", "fp1",
		"id,proba_0,proba_1\n10,0.8,0.2\n11,0.52,0.48\n", "")
	return shapePath, runsDir, []string{filepath.Join(ws, "steps")}
}

func writeMember(t *testing.T, runsDir, id, fingerprint, probaCSV, offsetsJSON string) {
	t.Helper()
	predictDir := filepath.Join(runsDir, id, "predict")
	trainDir := filepath.Join(runsDir, id, "train")
	for _, d := range []string{predictDir, trainDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if probaCSV != "" {
		if err := os.WriteFile(filepath.Join(predictDir, "probabilities.csv"), []byte(probaCSV), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if offsetsJSON != "" {
		if err := os.WriteFile(filepath.Join(trainDir, "offsets.json"), []byte(offsetsJSON), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rec := map[string]any{"run_id": id, "experiment": "blendtest", "seed": 7,
		"code_fingerprint": fingerprint, "status": "ok", "started": "2026-07-19T00:00:00Z"}
	b, _ := json.Marshal(rec)
	if err := os.WriteFile(filepath.Join(runsDir, id, "record.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBlend_EndToEnd(t *testing.T) {
	shapePath, runsDir, stepPath := blendWS(t)
	var out strings.Builder
	id, err := runBlend(blendOpts{shapePath: shapePath, runs: []string{"m1", "m2"},
		stepPath: stepPath, out: &out})
	if err != nil {
		t.Fatalf("blend: %v\n%s", err, out.String())
	}

	// Tilted-log blend: row 10 → 0; row 11 flips to 1 on m1's offset (see the pure test).
	preds, err := os.ReadFile(filepath.Join(runsDir, id, "predict", "predictions.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(preds)); got != "id,prediction\n10,0\n11,1" {
		t.Errorf("blended predictions:\n%s", got)
	}

	// The submission step executed via execStep.Execute → the LITERAL path kaggle expects.
	if _, err := os.Stat(filepath.Join(runsDir, id, "submission", "submission.csv")); err != nil {
		t.Errorf("submission.csv not at the literal kaggle path: %v", err)
	}

	// record.json resolves a slug the way kaggle's runref.go does (minimal-struct lookup).
	b, err := os.ReadFile(filepath.Join(runsDir, id, "record.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Steps []struct {
			With map[string]any `json:"with"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	slug := ""
	for _, s := range doc.Steps {
		if comp, ok := s.With["competition"].(map[string]any); ok {
			if v, ok := comp["slug"].(string); ok && v != "" {
				slug = v
				break
			}
		}
	}
	if slug != "test-comp" {
		t.Errorf("record.json slug lookup (runref semantics) = %q, want test-comp", slug)
	}
	// Blend provenance fields present.
	var rec struct {
		Members []string  `json:"blend_members"`
		Weights []float64 `json:"blend_weights"`
	}
	if err := json.Unmarshal(b, &rec); err != nil || len(rec.Members) != 2 || len(rec.Weights) != 2 {
		t.Errorf("blend record members/weights: %v %v %v", rec.Members, rec.Weights, err)
	}
	if !strings.Contains(out.String(), "leaderboard") {
		t.Errorf("the honesty caveat (leaderboard-measured only) must print; got:\n%s", out.String())
	}
}

func TestBlend_ProvenanceGuard(t *testing.T) {
	shapePath, runsDir, stepPath := blendWS(t)
	// Re-write m2 with a DIFFERENT fingerprint → refuse without --allow-mixed.
	writeMember(t, runsDir, "m2", "fp2",
		"id,proba_0,proba_1\n10,0.8,0.2\n11,0.52,0.48\n", "")
	var out strings.Builder
	_, err := runBlend(blendOpts{shapePath: shapePath, runs: []string{"m1", "m2"},
		stepPath: stepPath, out: &out})
	if err == nil || !strings.Contains(err.Error(), "fingerprint") {
		t.Fatalf("mixed fingerprints must refuse loudly, got %v", err)
	}
	// --allow-mixed proceeds, loudly.
	_, err = runBlend(blendOpts{shapePath: shapePath, runs: []string{"m1", "m2"},
		allowMixed: true, stepPath: stepPath, out: &out})
	if err != nil {
		t.Fatalf("--allow-mixed must proceed: %v", err)
	}
	if !strings.Contains(out.String(), "mixed") {
		t.Errorf("--allow-mixed must still warn loudly; got:\n%s", out.String())
	}
}

func TestBlend_MissingProbabilitiesRefusal(t *testing.T) {
	shapePath, runsDir, stepPath := blendWS(t)
	if err := os.Remove(filepath.Join(runsDir, "m2", "predict", "probabilities.csv")); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	_, err := runBlend(blendOpts{shapePath: shapePath, runs: []string{"m1", "m2"},
		stepPath: stepPath, out: &out})
	if err == nil || !strings.Contains(err.Error(), "re-promote") {
		t.Fatalf("missing probabilities.csv must refuse naming re-promote, got %v", err)
	}
}

func TestBlend_ColumnRealignmentIsOrderInvariant(t *testing.T) {
	// close-review Important 2: the by-name permutation path shipped untested — a mis-indexed
	// realign silently permutes class probabilities into garbage. Feed a member with REVERSED
	// columns; the combine must be value-identical to the aligned case.
	have := []string{"proba_2", "proba_1", "proba_0"}
	want := []string{"proba_0", "proba_1", "proba_2"}
	m := [][]float64{{0.3, 0.5, 0.2}, {0.1, 0.2, 0.7}}
	re, err := realignColumns(m, have, want)
	if err != nil {
		t.Fatal(err)
	}
	wantM := [][]float64{{0.2, 0.5, 0.3}, {0.7, 0.2, 0.1}}
	for i := range wantM {
		for j := range wantM[i] {
			if re[i][j] != wantM[i][j] {
				t.Fatalf("realign[%d][%d]=%v want %v", i, j, re[i][j], wantM[i][j])
			}
		}
	}
	if _, err := realignColumns(m, []string{"proba_9", "proba_1", "proba_0"}, want); err == nil {
		t.Error("column-set mismatch must refuse loudly")
	}
}

func TestNormalizeWeights_NonFiniteRefused(t *testing.T) {
	for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 0, -1} {
		if _, err := normalizeWeights(2, []float64{1, bad}); err == nil {
			t.Errorf("weight %v must be refused", bad)
		}
	}
}
