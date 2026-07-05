package shape

import (
	"sort"
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
)

// step is a terse constructor for a shape step in tests.
func step(id string, with map[string]any, needs ...string) experiment.Step {
	return experiment.Step{ID: id, Uses: "test/" + id, Needs: needs, With: with}
}

// The keystone: the worked titanic-sweep example. features(4) × [ logreg:C(3) +
// rf:n_estimators(3)×max_depth(2)=6 ] = 4 × 9 = 36. Proves $oneof ADDs (not
// multiplies) — a flat product would give features(4) × C(3) × n_est(3) × depth(2).
func TestExpand_TitanicSweep36Points(t *testing.T) {
	steps := []experiment.Step{
		step("adapt", map[string]any{
			"features": map[string]any{"$any": []any{
				[]any{}, []any{"title"}, []any{"title", "family"}, []any{"title", "family", "age_bin"},
			}},
		}),
		step("train", map[string]any{
			"model": map[string]any{"$oneof": map[string]any{
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
		"model": map[string]any{"$oneof": map[string]any{
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
