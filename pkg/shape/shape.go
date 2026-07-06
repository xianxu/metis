// Package shape is the metis#6 experiment-shape lift: the pure config-space algebra
// over v0's untyped `with` bag. An experiment-shape declares a *space* of configs via
// reserved `$`-key value-descriptors; Expand collapses it into concrete v0-shaped
// points. The lift is value-level (no CUE-typing of config leaves), so this is a pure
// Go recursion with no IO — the sweep loop that drives Expand + runs each point is
// metis#7; the ledger that keys off each point's free-param path is metis#8.
//
// The algebra:
//   - product   — a plain map {a:…, b:…}: cartesian of its fields (counts MULTIPLY).
//   - $any:[…]  — a set; each value taken VERBATIM (nested $-descriptors inside a $any
//     alternative are NOT expanded — unlike $oneof, which recurses into its branches).
//     Counts = len. Sugar for the flat sum.
//   - $oneof:{L:sub,…} — a labeled sum; counts ADD; resolves by BUNDLING (the chosen
//     branch collapses to {label: resolved-sub}, not flat siblings).
//   - $linear-range/$log-range: [lo,hi,steps?] — a domain+metric; the grid sampler
//     materializes it (linspace/logspace); steps defaults to range_steps.
//
// Every resolved point is v0-shaped `with` (nesting confined to the shape + Expand),
// and carries its free-param path — the swept coordinates that identify it (→ #8/#3).
package shape

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/xianxu/metis/pkg/experiment"
)

// FreeParam is one swept coordinate of a point: the dotted path to the space-
// descriptor leaf (e.g. "train.model", "train.model.rf.n_estimators") and the value
// chosen there (a scalar, a $any alternative, or a $oneof branch label). Fixed leaves
// and dataflow-refs never appear — only free (descriptor) leaves.
type FreeParam struct {
	Path  string
	Value any
}

// Point is one expanded experiment: the resolved per-step `with` (v0-shaped) plus the
// free-param path that identifies it within the shape.
type Point struct {
	With       map[string]map[string]any
	FreeParams []FreeParam
}

// resolved is one expansion of a single value: the concrete value plus the free-param
// choices that produced it.
type resolved struct {
	value any
	free  []FreeParam
}

// Expand collapses a shape's steps into every concrete point. rangeSteps is the
// default grid resolution for a $*-range that omits its own steps. The point set is
// the cartesian product of each step's independent expansions (a point picks one
// resolved-with per step); ordering is deterministic (sorted keys / sorted branch
// labels) so enumeration is stable across runs.
func Expand(steps []experiment.Step, rangeSteps int) ([]Point, error) {
	points := []Point{{With: map[string]map[string]any{}}}
	for _, s := range steps {
		with := s.With
		if with == nil {
			with = map[string]any{}
		}
		expansions, err := expandValue(s.ID, with, rangeSteps)
		if err != nil {
			return nil, fmt.Errorf("step %q: %w", s.ID, err)
		}
		var next []Point
		for _, base := range points {
			for _, r := range expansions {
				w, ok := r.value.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("step %q: with did not resolve to a map", s.ID)
				}
				// Deep-clone the resolved with so sibling points never alias the same
				// inner map — otherwise #7 overlaying a resolved path in place would
				// silently mutate every sibling sharing this expansion.
				np := Point{With: cloneWith(base.With), FreeParams: concat(base.FreeParams, r.free)}
				np.With[s.ID] = deepCloneMap(w)
				next = append(next, np)
			}
		}
		points = next
	}
	return points, nil
}

// expandValue is the core recursion: given a with-value at a free-param path, return
// every concrete resolution of it. A map is either a $-descriptor (sum/range) or a
// plain product; a list or scalar is a literal.
func expandValue(path string, v any, rangeSteps int) ([]resolved, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return []resolved{{value: v}}, nil // list or scalar → literal
	}
	dollar := dollarKeys(m)
	switch {
	case len(dollar) == 0:
		return expandProduct(path, m, rangeSteps)
	case len(dollar) > 1:
		return nil, fmt.Errorf("%s: multiple space-descriptors %v in one map", path, dollar)
	case len(m) > 1:
		return nil, fmt.Errorf("%s: space-descriptor %q cannot share a map with plain keys", path, dollar[0])
	}
	return expandDescriptor(path, dollar[0], m[dollar[0]], rangeSteps)
}

// expandProduct expands a plain map as a cartesian product of its fields.
func expandProduct(path string, m map[string]any, rangeSteps int) ([]resolved, error) {
	keys := sortedKeys(m)
	out := []resolved{{value: map[string]any{}}}
	for _, k := range keys {
		field, err := expandValue(join(path, k), m[k], rangeSteps)
		if err != nil {
			return nil, err
		}
		var next []resolved
		for _, base := range out {
			for _, f := range field {
				nv := cloneMap(base.value.(map[string]any))
				nv[k] = f.value
				next = append(next, resolved{value: nv, free: concat(base.free, f.free)})
			}
		}
		out = next
	}
	return out, nil
}

// expandDescriptor handles the reserved `$`-keys.
func expandDescriptor(path, key string, arg any, rangeSteps int) ([]resolved, error) {
	switch key {
	case "$any":
		alts, ok := arg.([]any)
		if !ok {
			return nil, fmt.Errorf("%s: $any takes a list of alternatives", path)
		}
		if len(alts) == 0 {
			return nil, fmt.Errorf("%s: $any is empty — an empty set would collapse the whole sweep to zero points", path)
		}
		out := make([]resolved, 0, len(alts))
		for _, a := range alts {
			out = append(out, resolved{value: a, free: []FreeParam{{Path: path, Value: a}}})
		}
		return out, nil

	case "$oneof":
		branches, ok := arg.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: $oneof takes a map of labeled branches", path)
		}
		if len(branches) == 0 {
			return nil, fmt.Errorf("%s: $oneof has no branches — would collapse the whole sweep to zero points", path)
		}
		var out []resolved
		for _, label := range sortedKeys(branches) {
			sub, err := expandValue(join(path, label), branches[label], rangeSteps)
			if err != nil {
				return nil, err
			}
			for _, r := range sub {
				bundled := map[string]any{label: r.value} // bundling: {label: resolved-sub}
				free := concat([]FreeParam{{Path: path, Value: label}}, r.free)
				out = append(out, resolved{value: bundled, free: free})
			}
		}
		return out, nil

	case "$linear-range", "$log-range":
		vals, err := materializeRange(path, key, arg, rangeSteps)
		if err != nil {
			return nil, err
		}
		out := make([]resolved, len(vals))
		for i, x := range vals {
			out[i] = resolved{value: x, free: []FreeParam{{Path: path, Value: x}}}
		}
		return out, nil

	default:
		return nil, fmt.Errorf("%s: unknown space-descriptor %q", path, key)
	}
}

// materializeRange turns a $*-range [lo, hi, steps?] into its grid points.
func materializeRange(path, key string, arg any, rangeSteps int) ([]float64, error) {
	spec, ok := arg.([]any)
	if !ok || len(spec) < 2 || len(spec) > 3 {
		return nil, fmt.Errorf("%s: %s takes [lo, hi, steps?]", path, key)
	}
	lo, ok1 := toFloat(spec[0])
	hi, ok2 := toFloat(spec[1])
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("%s: %s bounds must be numbers", path, key)
	}
	steps := rangeSteps
	if len(spec) == 3 {
		s, ok := toInt(spec[2])
		if !ok {
			return nil, fmt.Errorf("%s: %s steps must be an integer", path, key)
		}
		steps = s
	}
	if steps < 1 {
		return nil, fmt.Errorf("%s: %s needs steps >= 1 (got %d; set range_steps or a 3rd element)", path, key, steps)
	}
	if key == "$log-range" && (lo <= 0 || hi <= 0) {
		return nil, fmt.Errorf("%s: $log-range bounds must be positive", path)
	}
	out := make([]float64, steps)
	if steps == 1 {
		out[0] = lo
		return out, nil
	}
	for i := 0; i < steps; i++ {
		t := float64(i) / float64(steps-1)
		if key == "$log-range" {
			out[i] = lo * math.Pow(hi/lo, t)
		} else {
			out[i] = lo + t*(hi-lo)
		}
	}
	return out, nil
}

func dollarKeys(m map[string]any) []string {
	var ks []string
	for k := range m {
		if strings.HasPrefix(k, "$") {
			ks = append(ks, k)
		}
	}
	sort.Strings(ks)
	return ks
}

func sortedKeys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func join(path, seg string) string { return path + "." + seg }

func concat(a, b []FreeParam) []FreeParam {
	if len(b) == 0 {
		return a
	}
	out := make([]FreeParam, 0, len(a)+len(b))
	return append(append(out, a...), b...)
}

func cloneWith(in map[string]map[string]any) map[string]map[string]any {
	out := make(map[string]map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// deepCloneMap recursively copies a resolved `with` so no two points alias a shared
// inner map or slice — the value a consumer (#7) may overlay in place.
func deepCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = deepCloneValue(v)
	}
	return out
}

func deepCloneValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return deepCloneMap(x)
	case []any:
		cp := make([]any, len(x))
		for i, e := range x {
			cp[i] = deepCloneValue(e)
		}
		return cp
	default:
		return x // scalars are immutable
	}
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		if n == math.Trunc(n) {
			return int(n), true
		}
	}
	return 0, false
}
