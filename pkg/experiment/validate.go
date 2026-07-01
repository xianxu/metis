package experiment

import (
	"errors"
	"fmt"
	"regexp"
)

// usesRE matches a step's `uses`: "<layer>/<steptype>", each segment a lowercase
// slug (letters, digits, hyphens). This is the SEMANTIC check CUE cannot express
// (M1 owns shape; the format lives here).
var usesRE = regexp.MustCompile(`^[a-z0-9-]+/[a-z0-9-]+$`)

// Validate runs the semantic checks CUE's structural validator cannot express
// (ARCH-PURPOSE — the SHAPE-vs-SEMANTICS split M1 deferred):
//   - step ids are unique and non-empty,
//   - every `needs` id resolves to a real step in the same experiment,
//   - every `uses` matches "<layer>/<steptype>",
//   - the needs-graph is acyclic (delegated to TopoSort — one impl, ARCH-DRY).
//
// Pure. Returns a single joined error (errors.Join) listing ALL violations, so an
// author/agent sees every problem in one pass rather than fixing them one at a time.
func Validate(exp Experiment) error {
	var errs []error

	ids := make(map[string]bool, len(exp.Steps))
	for _, s := range exp.Steps {
		if s.ID == "" {
			errs = append(errs, errors.New("step with empty id"))
			continue
		}
		if ids[s.ID] {
			errs = append(errs, fmt.Errorf("duplicate step id %q", s.ID))
		}
		ids[s.ID] = true
	}

	for _, s := range exp.Steps {
		if !usesRE.MatchString(s.Uses) {
			errs = append(errs, fmt.Errorf("step %q: uses %q is not \"<layer>/<steptype>\"", s.ID, s.Uses))
		}
		for _, n := range s.Needs {
			if !ids[n] {
				errs = append(errs, fmt.Errorf("step %q: needs %q which is not a step in this experiment", s.ID, n))
			}
		}
	}

	// Acyclicity through the single TopoSort implementation (ARCH-DRY): if it can't
	// linearize the graph, the leftover steps form a cycle.
	if _, err := TopoSort(exp); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// TopoSort returns exp's steps in dependency order (each step after every step it
// needs) via Kahn's algorithm over the `needs` edges. Pure. Ties are broken by
// declaration order for deterministic output. Edges to unknown ids are ignored
// here (Validate reports dangling `needs` separately) so a typo'd need can't make
// cycle detection misfire. Returns an error naming the steps left in a cycle.
func TopoSort(exp Experiment) ([]Step, error) {
	known := make(map[string]bool, len(exp.Steps))
	for _, s := range exp.Steps {
		known[s.ID] = true
	}

	indeg := make(map[string]int, len(exp.Steps))
	for _, s := range exp.Steps {
		if _, seen := indeg[s.ID]; !seen {
			indeg[s.ID] = 0
		}
		for _, n := range s.Needs {
			if known[n] {
				indeg[s.ID]++
			}
		}
	}

	// Seed the queue with in-degree-0 steps in declaration order (determinism).
	var queue []Step
	for _, s := range exp.Steps {
		if indeg[s.ID] == 0 {
			queue = append(queue, s)
		}
	}

	var order []Step
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		order = append(order, cur)
		// Relax dependents, scanning in declaration order for stable output.
		for _, s := range exp.Steps {
			if !dependsOn(s, cur.ID) {
				continue
			}
			indeg[s.ID]--
			if indeg[s.ID] == 0 {
				queue = append(queue, s)
			}
		}
	}

	if len(order) != len(exp.Steps) {
		placed := make(map[string]bool, len(order))
		for _, s := range order {
			placed[s.ID] = true
		}
		var stuck []string
		for _, s := range exp.Steps {
			if !placed[s.ID] {
				stuck = append(stuck, s.ID)
			}
		}
		return nil, fmt.Errorf("cycle in step dependencies among: %v", stuck)
	}
	return order, nil
}

// dependsOn reports whether s lists id in its needs.
func dependsOn(s Step, id string) bool {
	for _, n := range s.Needs {
		if n == id {
			return true
		}
	}
	return false
}
