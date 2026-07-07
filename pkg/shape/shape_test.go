package shape

import (
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
)

// $any over a LIST recurses into each alternative (the new uniform recursion): a
// nested descriptor inside an element expands, and coords decompose without a
// duplicate at the element's own path.
func TestExpandAnyList_RecursesIntoElements(t *testing.T) {
	steps := []experiment.Step{step("s", map[string]any{
		"lr": map[string]any{"$any": []any{
			map[string]any{"$linear-range": []any{0.0, 1.0, 3}}, // → 0, 0.5, 1
			9.0,
		}},
	})}
	pts, err := Expand(steps, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 4 { // 3 from the range + 1 scalar
		t.Fatalf("got %d points, want 4 (range(3)+scalar)", len(pts))
	}
	got := map[float64]bool{}
	for _, p := range pts {
		n := 0
		for _, fp := range p.FreeParams {
			if fp.Path == "s.lr" { // exactly one coord at s.lr, never duplicated
				n++
				if v, ok := fp.Value.(float64); ok {
					got[v] = true
				}
			}
		}
		if n != 1 {
			t.Errorf("point %v has %d s.lr coords, want exactly 1", p.With, n)
		}
	}
	// The nested $linear-range materialized THROUGH the list branch (0/0.5/1) + the scalar (9).
	for _, want := range []float64{0.0, 0.5, 1.0, 9.0} {
		if !got[want] {
			t.Errorf("missing s.lr coord value %v; got %v", want, got)
		}
	}
}

// $any rejects a non-list, non-map argument.
func TestExpandAny_BadArg(t *testing.T) {
	_, err := Expand([]experiment.Step{step("s", map[string]any{
		"x": map[string]any{"$any": 5},
	})}, 5)
	if err == nil {
		t.Fatal("$any with a scalar arg must error (want: list or map)")
	}
}

// A stale $oneof is a clear error after the merge (the migration signal).
func TestOneofRemoved(t *testing.T) {
	_, err := Expand([]experiment.Step{step("s", map[string]any{
		"x": map[string]any{"$oneof": map[string]any{"a": 1}},
	})}, 5)
	if err == nil || !strings.Contains(err.Error(), "$oneof") {
		t.Fatalf("stale $oneof should error as unknown descriptor; got %v", err)
	}
}

// step is a terse constructor for a shape step in tests.
func step(id string, with map[string]any, needs ...string) experiment.Step {
	return experiment.Step{ID: id, Uses: "test/" + id, Needs: needs, With: with}
}

// The keystone: the worked titanic-sweep example. features(4) × [ logreg:C(3) +
// rf:n_estimators(3)×max_depth(2)=6 ] = 4 × 9 = 36. Proves the $any-MAP (tagged) form
// ADDs (not multiplies) — a flat product would give features(4) × C(3) × n_est(3) × depth(2).
func TestExpand_TitanicSweep36Points(t *testing.T) {
	steps := []experiment.Step{
		step("adapt", map[string]any{
			"features": map[string]any{"$any": []any{
				[]any{}, []any{"title"}, []any{"title", "family"}, []any{"title", "family", "age_bin"},
			}},
		}),
		step("train", map[string]any{
			"model": map[string]any{"$any": map[string]any{
				"logreg": map[string]any{"C": map[string]any{"$any": []any{0.1, 1.0, 10.0}}},
				"rf": map[string]any{
					"n_estimators": map[string]any{"$any": []any{100, 300, 500}},
					"max_depth":    map[string]any{"$any": []any{4, 8}},
				},
			}},
		}, "adapt"),
	}
	points, err := Expand(steps, 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 36 {
		t.Fatalf("expected 36 points (4 × [3+6]), got %d", len(points))
	}
	// Every point's train.model is a single-key bundle {logreg:{…}} or {rf:{…}}.
	for _, p := range points {
		model, ok := p.With["train"]["model"].(map[string]any)
		if !ok || len(model) != 1 {
			t.Fatalf("train.model should be a single-key bundle, got %#v", p.With["train"]["model"])
		}
		for label, sub := range model {
			if label != "logreg" && label != "rf" {
				t.Errorf("unexpected branch label %q", label)
			}
			if _, ok := sub.(map[string]any); !ok {
				t.Errorf("bundled branch %q should carry a sub-product map, got %#v", label, sub)
			}
		}
	}
}

// $any over scalars is a flat set; a plain map is a product (counts multiply).
func TestExpand_ProductAndSet(t *testing.T) {
	steps := []experiment.Step{step("s", map[string]any{
		"a": map[string]any{"$any": []any{1, 2}},
		"b": map[string]any{"$any": []any{"x", "y", "z"}},
		"c": "fixed",
	})}
	points, err := Expand(steps, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 6 { // 2 × 3 (c is fixed)
		t.Fatalf("product of sets: want 6, got %d", len(points))
	}
	for _, p := range points {
		if p.With["s"]["c"] != "fixed" {
			t.Errorf("fixed leaf must pass through: %v", p.With["s"]["c"])
		}
	}
}

// An all-singleton (no $-descriptor) shape expands to exactly ONE point = the v0
// experiment, with its `with` byte-identical (nesting confined to the shape).
func TestExpand_AllSingletonIsOnePoint(t *testing.T) {
	steps := []experiment.Step{
		step("adapt", map[string]any{"features": []any{"title"}}),
		step("train", map[string]any{"model": "logreg", "C": 1.0}, "adapt"),
	}
	points, err := Expand(steps, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 {
		t.Fatalf("all-singleton shape must yield exactly one point, got %d", len(points))
	}
	p := points[0]
	if p.With["train"]["model"] != "logreg" || p.With["train"]["C"] != 1.0 {
		t.Errorf("singleton with not passed through: %#v", p.With["train"])
	}
	if len(p.FreeParams) != 0 {
		t.Errorf("a singleton shape has no free params, got %v", p.FreeParams)
	}
}

// Free-param paths: only space-descriptor leaves contribute; ragged across branches.
func TestExpand_FreeParamPathsRagged(t *testing.T) {
	steps := []experiment.Step{step("train", map[string]any{
		"model": map[string]any{"$any": map[string]any{
			"logreg": map[string]any{"C": map[string]any{"$any": []any{0.1, 1.0}}},
			"rf":     map[string]any{"n_estimators": map[string]any{"$any": []any{100, 300}}},
		}},
	})}
	points, err := Expand(steps, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 4 { // logreg:C(2) + rf:n_est(2)
		t.Fatalf("want 4, got %d", len(points))
	}
	// Collect the free-param key sets — logreg points carry model+C, rf carry
	// model+n_estimators (ragged).
	seen := map[string]bool{}
	for _, p := range points {
		keys := make([]string, 0, len(p.FreeParams))
		for _, fp := range p.FreeParams {
			keys = append(keys, fp.Path)
		}
		sort.Strings(keys)
		seen[joinKeys(keys)] = true
	}
	if !seen["train.model,train.model.logreg.C"] {
		t.Errorf("logreg points should carry model + C; saw %v", seen)
	}
	if !seen["train.model,train.model.rf.n_estimators"] {
		t.Errorf("rf points should carry model + n_estimators; saw %v", seen)
	}
}

// A $*-range materializes to a grid; its free-param records the MATERIALIZED value
// (a concrete coordinate for the #8 ledger key), not the descriptor.
func TestExpand_RangeMaterializesToGrid(t *testing.T) {
	steps := []experiment.Step{step("train", map[string]any{
		"C": map[string]any{"$linear-range": []any{0.0, 10.0, 3}}, // 0, 5, 10
	})}
	points, err := Expand(steps, 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 3 {
		t.Fatalf("linear-range [0,10,3] → 3 points, got %d", len(points))
	}
	got := []float64{}
	for _, p := range points {
		v, ok := p.With["train"]["C"].(float64)
		if !ok {
			t.Fatalf("range leaf must resolve to a scalar float, got %#v", p.With["train"]["C"])
		}
		got = append(got, v)
		// the free-param value is the materialized scalar, not the descriptor
		if len(p.FreeParams) != 1 || p.FreeParams[0].Path != "train.C" || p.FreeParams[0].Value != v {
			t.Errorf("range free-param should be the materialized value; got %+v", p.FreeParams)
		}
	}
	sort.Float64s(got)
	want := []float64{0, 5, 10}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("linear grid = %v; want %v", got, want)
		}
	}
}

// $log-range materializes on a geometric grid: [1, 1000, 4] → 1, 10, 100, 1000.
func TestExpand_LogRangeGeometricGrid(t *testing.T) {
	steps := []experiment.Step{step("s", map[string]any{
		"lr": map[string]any{"$log-range": []any{1.0, 1000.0, 4}},
	})}
	points, err := Expand(steps, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]float64, 0, len(points))
	for _, p := range points {
		got = append(got, p.With["s"]["lr"].(float64))
	}
	sort.Float64s(got)
	want := []float64{1, 10, 100, 1000}
	if len(got) != 4 {
		t.Fatalf("log-range [1,1000,4] → 4 points, got %d", len(points))
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Errorf("log grid[%d] = %g; want %g (full %v)", i, got[i], want[i], got)
		}
	}
}

// $log-range with a non-positive bound is an error (log of ≤0 is undefined).
func TestExpand_LogRangeRejectsNonPositive(t *testing.T) {
	steps := []experiment.Step{step("s", map[string]any{
		"lr": map[string]any{"$log-range": []any{0.0, 100.0, 3}},
	})}
	if _, err := Expand(steps, 0); err == nil {
		t.Error("$log-range with lo=0 must error (log undefined)")
	}
}

// steps==1 yields the low bound; steps<1 (and no range_steps) errors.
func TestExpand_RangeSingleAndZeroSteps(t *testing.T) {
	one, err := Expand([]experiment.Step{step("s", map[string]any{
		"x": map[string]any{"$linear-range": []any{7.0, 9.0, 1}},
	})}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(one) != 1 || one[0].With["s"]["x"] != 7.0 {
		t.Errorf("steps=1 → single point at lo=7; got %+v", one)
	}
	if _, err := Expand([]experiment.Step{step("s", map[string]any{
		"x": map[string]any{"$linear-range": []any{0.0, 1.0}}, // no steps, range_steps=0
	})}, 0); err == nil {
		t.Error("a range with no steps and range_steps=0 must error, not silently emit nothing")
	}
}

// An empty $any (list OR map) must error, not silently collapse the sweep to zero.
func TestExpand_EmptyDescriptorErrors(t *testing.T) {
	if _, err := Expand([]experiment.Step{step("s", map[string]any{
		"x": map[string]any{"$any": []any{}},
	})}, 6); err == nil {
		t.Error("empty $any list must error")
	}
	if _, err := Expand([]experiment.Step{step("s", map[string]any{
		"x": map[string]any{"$any": map[string]any{}},
	})}, 6); err == nil {
		t.Error("empty $any map must error")
	}
}

// Sibling points must NOT alias each other's inner `with` maps — mutating one point's
// resolved value (as #7's overlay will) must not bleed into a sibling. Uses a ≥2-step
// shape and mutates a NON-TERMINAL (earlier) step, the case a single-step guard misses:
// the terminal step's expansion spawns the siblings, so they must not share the earlier
// step's map.
func TestExpand_PointsDoNotAliasInnerMaps(t *testing.T) {
	steps := []experiment.Step{
		step("prep", map[string]any{"dataset": "titanic"}), // fixed, non-terminal
		step("train", map[string]any{ // the $any here spawns the siblings
			"model": map[string]any{"$any": []any{"logreg", "rf"}},
		}, "prep"),
	}
	points, err := Expand(steps, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 2 {
		t.Fatalf("want 2 points, got %d", len(points))
	}
	// Mutate point 0's EARLIER-step `with` in place (what #7's overlay of a resolved
	// dataflow path would do) — must not bleed into the sibling.
	points[0].With["prep"]["dataset"] = "MUTATED"
	if points[1].With["prep"]["dataset"] != "titanic" {
		t.Error("sibling aliased an earlier step's `with` map — cross-step mutation bled across points")
	}
}

// range_steps default applies when a $*-range omits its own steps.
func TestExpand_RangeStepsDefault(t *testing.T) {
	steps := []experiment.Step{step("s", map[string]any{
		"x": map[string]any{"$linear-range": []any{0.0, 1.0}}, // no steps → use default
	})}
	points, err := Expand(steps, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 5 {
		t.Errorf("range with no steps should use range_steps=5, got %d points", len(points))
	}
}

// Malformed descriptors are surfaced as errors, not silently mis-expanded.
func TestExpand_MalformedDescriptorErrors(t *testing.T) {
	cases := map[string]map[string]any{
		"$-key mixed with plain keys": {"$any": []any{1}, "plain": 2},
		"unknown $-key":               {"$bogus": []any{1}},
		"non-numeric range bound":     {"$linear-range": []any{"lo", "hi", 3}},
	}
	for name, with := range cases {
		if _, err := Expand([]experiment.Step{step("s", map[string]any{"leaf": with})}, 6); err == nil {
			t.Errorf("%s: expected an error, got none", name)
		}
	}
}

func joinKeys(ks []string) string {
	out := ""
	for i, k := range ks {
		if i > 0 {
			out += ","
		}
		out += k
	}
	return out
}
