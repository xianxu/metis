package experiment

import "fmt"

// CachePolicy is a step's cache directive, parsed from `with: {cache: {...}}` — the one
// home for cache policy (metis#2's leaf policy, metis#71's never-cache). Domain-free: a
// nondeterministic agent session and a human grading step both say `policy: never`.
type CachePolicy struct {
	// ImmutableLeaf (`leaf: immutable`): a pinned external fetch, cached on K_pre alone.
	ImmutableLeaf bool
	// Never (`policy: never`): the step is nondeterministic by declaration — no lookup,
	// no store, it runs every time, and so does every descendant (UncachedSteps).
	Never bool
}

// ParseCachePolicy reads a step's `with.cache` map. Pure. An absent map is the zero
// policy; a non-map, an unknown key or value, or leaf + never together is an error
// (Validate surfaces it before any step executes — a typo must not silently cache).
func ParseCachePolicy(s Step) (CachePolicy, error) {
	raw, ok := s.With["cache"]
	if !ok {
		return CachePolicy{}, nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return CachePolicy{}, fmt.Errorf("step %q: with.cache must be a map (leaf: immutable | policy: never), got %v", s.ID, raw)
	}
	var p CachePolicy
	for k, v := range m {
		switch {
		case k == "leaf" && v == "immutable":
			p.ImmutableLeaf = true
		case k == "policy" && v == "never":
			p.Never = true
		case k == "leaf" || k == "policy":
			return CachePolicy{}, fmt.Errorf("step %q: with.cache.%s: unknown value %v", s.ID, k, v)
		default:
			return CachePolicy{}, fmt.Errorf("step %q: with.cache: unknown key %q (want leaf | policy)", s.ID, k)
		}
	}
	if p.ImmutableLeaf && p.Never {
		return CachePolicy{}, fmt.Errorf("step %q: with.cache: leaf: immutable and policy: never contradict", s.ID)
	}
	return p, nil
}

// UncachedSteps returns the ids of steps that must bypass the cache (metis#71): a step
// declaring `policy: never`, and — conservatively — every transitive descendant of one.
// A descendant's input-addressed key cannot tell two realized outputs of a
// nondeterministic producer apart, so it must not reuse on input identity alone; binding
// it to the realized artifacts instead was rejected (sound only if declared artifacts
// capture everything read, which metis cannot verify). Pure; one memoized DFS, O(steps +
// edges). Fails closed: a malformed policy marks its step uncached (Validate refuses it
// before execution anyway); dangling needs and cycles are ignored here (Validate's job).
func UncachedSteps(exp Experiment) map[string]bool {
	byID := make(map[string]Step, len(exp.Steps))
	for _, s := range exp.Steps {
		byID[s.ID] = s
	}
	memo := make(map[string]bool, len(exp.Steps))
	visiting := map[string]bool{}
	var uncached func(id string) bool
	uncached = func(id string) bool {
		if v, ok := memo[id]; ok {
			return v
		}
		s, ok := byID[id]
		if !ok || visiting[id] {
			return false
		}
		visiting[id] = true
		p, err := ParseCachePolicy(s)
		u := err != nil || p.Never
		for _, n := range s.Needs {
			u = uncached(n) || u
		}
		delete(visiting, id)
		memo[id] = u
		return u
	}
	out := map[string]bool{}
	for _, s := range exp.Steps {
		if uncached(s.ID) {
			out[s.ID] = true
		}
	}
	return out
}
