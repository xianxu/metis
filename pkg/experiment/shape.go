package experiment

import (
	"fmt"

	"github.com/xianxu/ariadne/pkg/frontmatter"
	"gopkg.in/yaml.v3"
)

// Shape mirrors CUE #ExperimentShape: an experiment lifted into a config-space. It is
// #Experiment's field set (embedded) plus a `sweep:` block; the `$`-key value-algebra
// lives in the untyped `with` bags (pkg/shape expands it). `#Experiment` is the
// singleton refinement (type "experiment", no sweep) — so Shape reuses the experiment
// Step/Validate machinery rather than restating the DAG (ARCH-DRY).
type Shape struct {
	Experiment `yaml:",inline"`
	Sweep      Sweep `yaml:"sweep"`
}

// Sweep is the shape's sweep block: which sampler drives the space, what objective it
// optimizes (consumed by adaptive samplers #7 + #8's pick-best), and the default grid
// resolution for a $*-range that omits its own steps.
type Sweep struct {
	Sampler    string    `yaml:"sampler"`
	Objective  Objective `yaml:"objective"`
	RangeSteps int       `yaml:"range_steps,omitempty"`
}

// Objective names the metric to optimize and the direction — declared once, consumed
// by the sampler (what to optimize) and #8's promotion (pick-best).
type Objective struct {
	Metric    string `yaml:"metric"`
	Direction string `yaml:"direction"` // "maximize" | "minimize"
}

// ParseShape splits an experiment-shape markdown document into frontmatter + body
// (reusing ariadne's frontmatter.Split — ARCH-DRY) and unmarshals the frontmatter into
// a Shape. Pure: string → (Shape, error), no IO. The `$`-descriptors in `with` survive
// as untyped maps for pkg/shape to expand.
func ParseShape(content string) (Shape, error) {
	fm, _, err := frontmatter.Split(content)
	if err != nil {
		return Shape{}, err
	}
	var sh Shape
	if err := yaml.Unmarshal([]byte(fm), &sh); err != nil {
		return Shape{}, fmt.Errorf("parse experiment-shape frontmatter: %w", err)
	}
	return sh, nil
}

// ValidateShape runs the experiment DAG semantics (unique ids, needs-resolution, uses
// format, acyclicity — reusing Validate, ARCH-DRY) plus the shape-only checks: the
// sweep block must name a sampler and a well-formed objective direction.
func ValidateShape(sh Shape) error {
	if err := Validate(sh.Experiment); err != nil {
		return err
	}
	if sh.Sweep.Sampler == "" {
		return fmt.Errorf("shape %q: sweep.sampler is required", sh.ID)
	}
	if d := sh.Sweep.Objective.Direction; d != "" && d != "maximize" && d != "minimize" {
		return fmt.Errorf("shape %q: sweep.objective.direction %q must be maximize|minimize", sh.ID, d)
	}
	return nil
}
