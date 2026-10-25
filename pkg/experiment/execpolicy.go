package experiment

import (
	"fmt"
	"time"
)

// ExecPolicy is a step's execution directive, parsed from `with: {exec: {...}}` (metis#72) —
// namespaced beside with.cache so it cannot collide with a step type's own `with.timeout`.
type ExecPolicy struct {
	// Timeout bounds the child's runtime from launch (a leaf-budget wait is not counted).
	// Zero means none: an ordinary step runs as long as it takes.
	Timeout time.Duration
}

// ParseExecPolicy reads a step's `with.exec` map. Pure. An absent map is the zero policy; a
// non-map, an unknown key, or a timeout that is not a positive Go duration string is an error
// (Validate surfaces it before any step executes).
func ParseExecPolicy(s Step) (ExecPolicy, error) {
	raw, ok := s.With["exec"]
	if !ok {
		return ExecPolicy{}, nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return ExecPolicy{}, fmt.Errorf("step %q: with.exec must be a map (timeout: <duration>), got %v", s.ID, raw)
	}
	var p ExecPolicy
	for k, v := range m {
		if k != "timeout" {
			return ExecPolicy{}, fmt.Errorf("step %q: with.exec: unknown key %q (want timeout)", s.ID, k)
		}
		str, ok := v.(string)
		if !ok {
			return ExecPolicy{}, fmt.Errorf("step %q: with.exec.timeout: want a duration string like \"90m\", got %v", s.ID, v)
		}
		d, err := time.ParseDuration(str)
		if err != nil {
			return ExecPolicy{}, fmt.Errorf("step %q: with.exec.timeout: invalid duration %q", s.ID, str)
		}
		if d <= 0 {
			return ExecPolicy{}, fmt.Errorf("step %q: with.exec.timeout: must be positive, got %q", s.ID, str)
		}
		p.Timeout = d
	}
	return p, nil
}
