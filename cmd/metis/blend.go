// metis blend (metis#60 M2) — weights-only soft-vote over PROMOTED runs' probabilities.
//
// Averages member probabilities in TILTED log-space — member i contributes
// w_i · (log(clip(p_i, 1e-12)) + o_i) where o_i is ITS persisted offsets.json (zeros when
// absent) — then argmax (first-max ties). Each member's tuned decision layer carries
// through without re-tuning. The result is materialized as runs/blend-<hash>/ with the
// ship predict step's artifact shape + a blend-flavored record.json, and the shape's own
// ship SUBMISSION step is executed on it via execStep.Execute — so `kaggle submit --run
// blend-...` works unchanged (record.json carries the shape's steps; runref takes the
// first steps[].with.competition.slug; the CSV lands at the literal
// runs/<id>/submission/submission.csv because the ship submission step's id is
// `submission`, inherited from the shape).
//
// HONESTY (recorded in metis#60 Revisions): a blend has no in-sweep OOF material — its
// quality is leaderboard-measured ONLY; the verb prints that caveat on every run.
//
// Deliberately NOT runResolvedExperiment: that path DAG-validates `needs`, overwrites the
// blend's record.json via assembleRecord, and fires captureSingleRun — execStep.Execute is
// the single-step seam with the full env contract and none of the side effects.
package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/record"
)

const blendClip = 1e-12 // MUST match metis/model.py apply_offsets' clip (single-member identity)

type blendMember struct {
	id      string
	proba   [][]float64 // [row][col], columns aligned to the shared label order
	offsets []float64   // per column; zeros when the run has no offsets.json
}

type blendOpts struct {
	shapePath  string
	runs       []string
	weights    []float64 // raw (pre-normalization); nil = equal
	allowMixed bool
	stepPath   []string // test seam; nil → stepPath(shapePath)
	out        io.Writer
}

func cmdBlend(args []string) error {
	fs := flag.NewFlagSet("blend", flag.ContinueOnError)
	runsFlag := fs.String("runs", "", "comma-separated PROMOTED run ids to blend (each needs probabilities.csv — re-promote pre-#60 runs)")
	weightsFlag := fs.String("weights", "", "comma-separated member weights (default equal; normalized; must be positive)")
	allowMixed := fs.Bool("allow-mixed", false, "proceed (loudly) when members span different code fingerprints or experiments")
	shapePath, flags, err := hoistShapePath(args)
	if err != nil {
		return fmt.Errorf("blend: %w (usage: metis blend <shape.md> --runs id1,id2[,...] [--weights w1,w2] [--allow-mixed])", err)
	}
	if err := fs.Parse(flags); err != nil {
		return err
	}
	if *runsFlag == "" {
		return fmt.Errorf("blend: --runs is required (comma-separated promoted run ids)")
	}
	o := blendOpts{shapePath: shapePath, runs: strings.Split(*runsFlag, ","), allowMixed: *allowMixed, out: os.Stdout}
	if *weightsFlag != "" {
		for _, w := range strings.Split(*weightsFlag, ",") {
			f, err := strconv.ParseFloat(strings.TrimSpace(w), 64)
			if err != nil {
				return fmt.Errorf("blend: bad --weights entry %q: %v", w, err)
			}
			o.weights = append(o.weights, f)
		}
	}
	id, err := runBlend(o)
	if err != nil {
		return err
	}
	fmt.Fprintf(o.out, "metis: %s ok\n", id)
	return nil
}

func runBlend(o blendOpts) (string, error) {
	raw, err := os.ReadFile(o.shapePath)
	if err != nil {
		return "", err
	}
	sh, err := experiment.ParseShape(string(raw))
	if err != nil {
		return "", err
	}
	predictStep, submissionStep, trainStepID, err := shipSteps(sh)
	if err != nil {
		return "", err
	}
	weights, err := normalizeWeights(len(o.runs), o.weights)
	if err != nil {
		return "", err
	}

	baseDir := filepath.Dir(o.shapePath)
	runsDir, err := filepath.Abs(filepath.Join(baseDir, "runs"))
	if err != nil {
		return "", err
	}
	expDir, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}

	// Load members: probabilities (loud if absent — pre-#60 promotions need a re-promote),
	// offsets aligned BY CLASS LABEL, provenance for the mixed-cohort guard.
	var cols, ids []string
	var idCol string
	members := make([]blendMember, 0, len(o.runs))
	type prov struct{ experiment, fingerprint string }
	var provs []prov
	for _, runID := range o.runs {
		runDir := filepath.Join(runsDir, runID)
		probaPath := filepath.Join(runDir, predictStep.ID, "probabilities.csv")
		mIDCol, mIDs, mCols, matrix, err := readProbabilities(probaPath)
		if err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("blend: %s has no %s/probabilities.csv — promoted before metis#60; re-promote it (metis select ... --point/--best --promote)", runID, predictStep.ID)
			}
			return "", fmt.Errorf("blend: %s: %v", runID, err)
		}
		if cols == nil {
			idCol, ids, cols = mIDCol, mIDs, mCols
		} else {
			if !equalStrings(ids, mIDs) {
				return "", fmt.Errorf("blend: %s id set/order differs from %s — members must predict the same rows", runID, o.runs[0])
			}
			var reordered [][]float64
			if reordered, err = realignColumns(matrix, mCols, cols); err != nil {
				return "", fmt.Errorf("blend: %s: %v", runID, err)
			}
			matrix = reordered
		}
		offsets, err := readOffsets(filepath.Join(runDir, trainStepID, "offsets.json"), cols)
		if err != nil {
			return "", fmt.Errorf("blend: %s: %v", runID, err)
		}
		members = append(members, blendMember{id: runID, proba: matrix, offsets: offsets})

		var rec struct {
			Experiment      string `json:"experiment"`
			CodeFingerprint string `json:"code_fingerprint"`
		}
		if b, err := os.ReadFile(filepath.Join(runDir, "record.json")); err == nil {
			_ = json.Unmarshal(b, &rec)
		}
		provs = append(provs, prov{rec.Experiment, rec.CodeFingerprint})
	}

	// Provenance guard (the sibling-verb posture: select refuses mixed cohorts).
	for _, p := range provs[1:] {
		if p.experiment != provs[0].experiment || p.fingerprint != provs[0].fingerprint {
			if !o.allowMixed {
				return "", fmt.Errorf("blend: members span different code fingerprint/experiment provenance (%v vs %v) — a cross-version blend is not one procedure; pass --allow-mixed to proceed loudly", provs[0], p)
			}
			fmt.Fprintf(o.out, "metis: WARNING — blending across mixed provenance (%v vs %v); --allow-mixed given\n", provs[0], p)
			break
		}
	}

	preds, err := blendCombine(cols, members, weights)
	if err != nil {
		return "", err
	}

	// Materialize runs/blend-<hash>/: the predict-step-shaped predictions + the record.
	id := blendID(o.runs, weights)
	blendRunDir := filepath.Join(runsDir, id)
	predictDir := filepath.Join(blendRunDir, predictStep.ID)
	if err := os.MkdirAll(predictDir, 0o755); err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString(idCol + ",prediction\n")
	for i, rowID := range ids {
		sb.WriteString(rowID + "," + preds[i] + "\n")
	}
	if err := os.WriteFile(filepath.Join(predictDir, "predictions.csv"), []byte(sb.String()), 0o644); err != nil {
		return "", err
	}
	if err := writeBlendRecord(blendRunDir, id, sh, o.runs, weights); err != nil {
		return "", err
	}

	// Execute the shape's ship submission step — execStep.Execute, nothing else (see header).
	sp := o.stepPath
	if sp == nil {
		sp = stepPath(o.shapePath)
	}
	es := execStep{stepPath: sp, expDir: expDir, seed: sh.Seed, out: o.out}
	if _, err := es.Execute(submissionStep, blendRunDir); err != nil {
		return "", fmt.Errorf("blend: submission step: %w", err)
	}

	fmt.Fprintf(o.out, "metis: blended %d run(s) → %s (weights %v)\n", len(o.runs), id, weights)
	fmt.Fprintf(o.out, "  CAVEAT: a blend has no in-sweep honest estimate — leaderboard-measured only.\n")
	fmt.Fprintf(o.out, "  submit: kaggle submit --run %s   (submission/submission.csv + record.json slug)\n", id)
	return id, nil
}

// shipSteps derives the ship predict + submission steps and the train step id from the
// shape — NEVER hardcoded ids: predict = the ship step whose `with` names a `model`
// (its value IS the train step id); submission = the ship step consuming `predictions`.
func shipSteps(sh experiment.Shape) (experiment.Step, experiment.Step, string, error) {
	var predict, submission experiment.Step
	var haveP, haveS bool
	for _, s := range sh.Ship {
		if _, ok := s.With["model"]; ok && !haveP {
			predict, haveP = s, true
		}
		if _, ok := s.With["predictions"]; ok {
			submission, haveS = s, true
		}
	}
	if !haveP || !haveS {
		return experiment.Step{}, experiment.Step{}, "", fmt.Errorf("blend: shape %s ship phase needs a predict step (with.model) and a submission step (with.predictions)", sh.ID)
	}
	train, _ := predict.With["model"].(string)
	if train == "" {
		return experiment.Step{}, experiment.Step{}, "", fmt.Errorf("blend: ship predict step's with.model must name the train step id")
	}
	return predict, submission, train, nil
}

// readProbabilities parses a predict-step probabilities.csv → (id column name, ids,
// class-label columns (the proba_ suffix IS the label), row-major matrix).
func readProbabilities(path string) (string, []string, []string, [][]float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil, nil, nil, err
	}
	defer f.Close()
	recs, err := csv.NewReader(f).ReadAll()
	if err != nil || len(recs) < 2 {
		return "", nil, nil, nil, fmt.Errorf("probabilities.csv unreadable or empty: %v", err)
	}
	header := recs[0]
	if len(header) < 2 {
		return "", nil, nil, nil, fmt.Errorf("probabilities.csv needs id + proba_* columns")
	}
	idCol := header[0]
	var cols []string
	for _, h := range header[1:] {
		if !strings.HasPrefix(h, "proba_") {
			return "", nil, nil, nil, fmt.Errorf("unexpected probabilities column %q (want proba_<class>)", h)
		}
		cols = append(cols, strings.TrimPrefix(h, "proba_"))
	}
	var ids []string
	var matrix [][]float64
	for _, rec := range recs[1:] {
		ids = append(ids, rec[0])
		row := make([]float64, len(cols))
		for j := range cols {
			v, err := strconv.ParseFloat(rec[j+1], 64)
			if err != nil {
				return "", nil, nil, nil, fmt.Errorf("bad probability %q: %v", rec[j+1], err)
			}
			row[j] = v
		}
		matrix = append(matrix, row)
	}
	return idCol, ids, cols, matrix, nil
}

// readOffsets loads a run's offsets.json aligned BY CLASS LABEL to the blend's column
// order (blend has no model — this label alignment is its only classes check). A missing
// file = an argmax run = zero offsets.
func readOffsets(path string, cols []string) ([]float64, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return make([]float64, len(cols)), nil
	}
	if err != nil {
		return nil, err
	}
	var payload struct {
		Offsets []float64 `json:"offsets"`
		Classes []any     `json:"classes"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, fmt.Errorf("offsets.json: %v", err)
	}
	if len(payload.Offsets) != len(payload.Classes) {
		return nil, fmt.Errorf("offsets.json offsets/classes length mismatch")
	}
	byLabel := map[string]float64{}
	for i, c := range payload.Classes {
		byLabel[classLabel(c)] = payload.Offsets[i]
	}
	out := make([]float64, len(cols))
	for j, c := range cols {
		v, ok := byLabel[c]
		if !ok {
			return nil, fmt.Errorf("offsets.json classes %v do not cover probabilities column %q", payload.Classes, c)
		}
		out[j] = v
	}
	return out, nil
}

// classLabel renders a JSON class value the way predict.py's f-string suffixes do
// (int-coded targets → "0"/"1"/...).
func classLabel(v any) string {
	if f, ok := v.(float64); ok && f == math.Trunc(f) {
		return strconv.Itoa(int(f))
	}
	return fmt.Sprintf("%v", v)
}

// blendCombine is the pure tilted-log soft vote: score = Σ_i w_i·(log(clip(p_i)) + o_i),
// argmax with first-max ties. Returns the winning class LABEL per row.
func blendCombine(cols []string, members []blendMember, weights []float64) ([]string, error) {
	if len(members) == 0 || len(members) != len(weights) {
		return nil, fmt.Errorf("blend: %d members vs %d weights", len(members), len(weights))
	}
	n := len(members[0].proba)
	for _, m := range members {
		if len(m.proba) != n {
			return nil, fmt.Errorf("blend: member %s has %d rows, want %d", m.id, len(m.proba), n)
		}
		if len(m.offsets) != len(cols) {
			return nil, fmt.Errorf("blend: member %s offsets/columns mismatch", m.id)
		}
		for _, row := range m.proba {
			if len(row) != len(cols) {
				return nil, fmt.Errorf("blend: member %s column count mismatch", m.id)
			}
		}
	}
	out := make([]string, n)
	for r := 0; r < n; r++ {
		best, bestScore := 0, math.Inf(-1)
		for c := range cols {
			score := 0.0
			for i, m := range members {
				score += weights[i] * (math.Log(math.Max(m.proba[r][c], blendClip)) + m.offsets[c])
			}
			if score > bestScore { // strict: first-max wins ties
				best, bestScore = c, score
			}
		}
		out[r] = cols[best]
	}
	return out, nil
}

func normalizeWeights(n int, raw []float64) ([]float64, error) {
	if raw == nil {
		w := make([]float64, n)
		for i := range w {
			w[i] = 1.0 / float64(n)
		}
		return w, nil
	}
	if len(raw) != n {
		return nil, fmt.Errorf("blend: %d weights for %d runs", len(raw), n)
	}
	sum := 0.0
	for _, v := range raw {
		if v <= 0 {
			return nil, fmt.Errorf("blend: weights must be positive, got %v", v)
		}
		sum += v
	}
	out := make([]float64, n)
	for i, v := range raw {
		out[i] = v / sum
	}
	return out, nil
}

// blendID mints the run id from the (member, NORMALIZED weight) pairs — same members
// with different weights must not collide.
func blendID(runs []string, weights []float64) string {
	type pair struct {
		Run    string  `json:"run"`
		Weight float64 `json:"weight"`
	}
	pairs := make([]pair, len(runs))
	for i := range runs {
		pairs[i] = pair{runs[i], weights[i]}
	}
	b, _ := json.Marshal(pairs)
	h := sha256.Sum256(b)
	return "blend-" + hex.EncodeToString(h[:])[:8]
}

// writeBlendRecord persists the blend-flavored record: a record.RunRecord (readers are
// PointAddr/ledger-driven and never scan blend-*, kaggle's runref reader is a minimal
// struct — extra fields are safe) carrying the SHAPE's steps so `kaggle submit --run`
// resolves the slug from the first steps[].with.competition.slug.
func writeBlendRecord(runDir, id string, sh experiment.Shape, members []string, weights []float64) error {
	steps := make([]record.StepRecord, 0, len(sh.Data)+len(sh.Pipeline)+len(sh.Ship))
	for _, group := range [][]experiment.Step{sh.Data, sh.Pipeline, sh.Ship} {
		for _, s := range group {
			steps = append(steps, record.StepRecord{StepID: s.ID, Uses: s.Uses, With: s.With})
		}
	}
	rec := struct {
		record.RunRecord
		Members []string  `json:"blend_members"`
		Weights []float64 `json:"blend_weights"`
	}{
		RunRecord: record.RunRecord{RunID: id, Experiment: sh.ID, Seed: sh.Seed, Steps: steps,
			Started: time.Now().UTC().Format(time.RFC3339), Status: "ok"},
		Members: members, Weights: weights,
	}
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "record.json"), append(b, '\n'), 0o644)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// realignColumns reorders a member's matrix columns from `have` order to `want` order
// (BY NAME — never positional; sklearn's sorted classes_ makes orders agree in practice,
// but nothing may assume it).
func realignColumns(matrix [][]float64, have, want []string) ([][]float64, error) {
	if len(have) != len(want) {
		return nil, fmt.Errorf("probabilities column sets differ: %v vs %v", have, want)
	}
	idx := make([]int, len(want))
	for j, w := range want {
		found := -1
		for k, h := range have {
			if h == w {
				found = k
				break
			}
		}
		if found == -1 {
			return nil, fmt.Errorf("probabilities column sets differ: %v vs %v", have, want)
		}
		idx[j] = found
	}
	out := make([][]float64, len(matrix))
	for r, row := range matrix {
		nr := make([]float64, len(want))
		for j, k := range idx {
			nr[j] = row[k]
		}
		out[r] = nr
	}
	return out, nil
}
