package experiment

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/xianxu/ariadne/pkg/frontmatter"
	"gopkg.in/yaml.v3"
)

// Shape mirrors CUE #ExperimentShape (metis#18 v2): an experiment lifted into a
// config-space, structured into three PHASES — `data` (produced once, above the
// resample), `pipeline` (the swept algorithm×hyperparameter atom, run per-fold),
// and `ship` (winner-only) — plus a `sweeper` (the config-level Sampler: sampler +
// inner resample + objective+select) and a `driver` (the outer Sampler: single | cv).
// Each phase is a []Step DAG, reusing the experiment Step/Validate machinery rather
// than restating the DAG (ARCH-DRY). The `$`-key value-algebra lives in the untyped
// `with` bags (pkg/shape expands it). Supersedes the v1 flat `steps` + `sweep` shape.
type Shape struct {
	Type        string  `yaml:"type"`
	ID          string  `yaml:"id"`
	Competition string  `yaml:"competition,omitempty"`
	Seed        int     `yaml:"seed"`
	Status      string  `yaml:"status"`
	Data        []Step  `yaml:"data"`
	Pipeline    []Step  `yaml:"pipeline"`
	Ship        []Step  `yaml:"ship"`
	Sweeper     Sweeper `yaml:"sweeper"`
	Driver      Driver  `yaml:"driver"`
}

// Sweeper is the config-level Sampler (mlr3 AutoTuner): the sampler that proposes
// configs, the INNER resample that scores each (owned here, not a peer layer), and
// the objective+select that turns per-config (mean,SE) into the winner.
type Sweeper struct {
	Sampler   string    `yaml:"sampler"`  // "grid" (M1a); the ask/tell seam for adaptive samplers later
	Resample  Resample  `yaml:"resample"` // the inner CV — how each config is scored
	Objective Objective `yaml:"objective"`
}

// Resample is the inner resampling strategy. M1a: fixed k-fold CV only.
type Resample struct {
	CV CVResample `yaml:"cv"`
}

// CVResample is k-fold cross-validation config.
type CVResample struct {
	K        int  `yaml:"k"`
	Stratify bool `yaml:"stratify,omitempty"`
}

// Objective names the metric to optimize, the direction, and the select rule that
// reduces per-config (mean,SE) → a winner. M1a supports `argmax-mean` only; the
// 1-standard-error / pct-loss rules (metis#19) are a different `select` over the
// same cached fold-scores.
type Objective struct {
	Metric    string `yaml:"metric"`
	Direction string `yaml:"direction"`        // "maximize" | "minimize"
	Select    string `yaml:"select,omitempty"` // "argmax-mean" (M1a); one-std-err | pct-loss later (#19)
}

// Driver is the OUTER Sampler — the honest evaluator, and it is optional. `single`
// (the degenerate outer Sampler: fit the sweeper on all data, ship the winner) is
// M1a; `cv` (nested-CV, the honest procedure estimate) is metis#23. Exactly one is set.
type Driver struct {
	Single *SingleDriver `yaml:"single,omitempty"`
	CV     *CVDriver     `yaml:"cv,omitempty"`
}

// SingleDriver carries no config — it's the "no outer resample, ship the winner" case.
type SingleDriver struct{}

// CVDriver is the outer k-fold resample (metis#23 nested-CV). Parsed here so #23 is a
// purely additive change (no schema churn), but M1a rejects it at validate time.
type CVDriver struct {
	K        int  `yaml:"k"`
	Stratify bool `yaml:"stratify,omitempty"`
}

// ParseShape splits an experiment-shape markdown document into frontmatter + body
// (reusing ariadne's frontmatter.Split — ARCH-DRY) and unmarshals the frontmatter into
// a Shape. Pure: string → (Shape, error), no IO. Decodes with KnownFields(true) so an
// unknown top-level or sub-key is a LOUD error (matching CUE's closed #ExperimentShape),
// rather than yaml.v3's silent-drop (ARCH-PURE root-cause on a footgun). The
// `$`-descriptors in `with` survive as untyped maps for pkg/shape to expand.
func ParseShape(content string) (Shape, error) {
	fm, _, err := frontmatter.Split(content)
	if err != nil {
		return Shape{}, err
	}
	var sh Shape
	dec := yaml.NewDecoder(strings.NewReader(fm))
	dec.KnownFields(true)
	if err := dec.Decode(&sh); err != nil && !errors.Is(err, io.EOF) {
		return Shape{}, fmt.Errorf("parse experiment-shape frontmatter: %w", err)
	}
	return sh, nil
}

// ValidateShape runs the DAG semantics over the COMBINED phase DAG plus the shape-only
// checks. The three phases (data │ pipeline │ ship) form ONE dependency graph with
// cross-phase edges (features(pipeline) needs adapt(data); predict(ship) needs
// train(pipeline)) — so they are concatenated into one synthetic Experiment and run
// through Validate (unique ids across all phases, cross-phase needs-resolution, uses
// format, acyclicity — reusing Validate, ARCH-DRY). Validating each phase in isolation
// would wrongly reject cross-phase needs. Shape-only checks: a non-empty pipeline, a
// sampler, a valid inner resample, a valid objective direction, an M1a-supported select
// rule, and exactly one driver mode (single | cv).
func ValidateShape(sh Shape) error {
	combined := Experiment{
		Type:        "experiment",
		ID:          sh.ID,
		Competition: sh.Competition,
		Seed:        sh.Seed,
		Status:      sh.Status,
		Steps:       combinedSteps(sh),
	}
	if err := Validate(combined); err != nil {
		return err
	}

	if len(sh.Pipeline) == 0 {
		return fmt.Errorf("shape %q: pipeline phase is required (non-empty)", sh.ID)
	}
	if sh.Sweeper.Sampler == "" {
		return fmt.Errorf("shape %q: sweeper.sampler is required", sh.ID)
	}
	if sh.Sweeper.Resample.CV.K < 2 {
		return fmt.Errorf("shape %q: sweeper.resample.cv.k must be >= 2, got %d", sh.ID, sh.Sweeper.Resample.CV.K)
	}
	if d := sh.Sweeper.Objective.Direction; d != "" && d != "maximize" && d != "minimize" {
		return fmt.Errorf("shape %q: sweeper.objective.direction %q must be maximize|minimize", sh.ID, d)
	}
	if s := sh.Sweeper.Objective.Select; s != "argmax-mean" {
		return fmt.Errorf("shape %q: sweeper.objective.select %q must be argmax-mean (M1a; one-std-err is metis#19)", sh.ID, s)
	}
	n := 0
	if sh.Driver.Single != nil {
		n++
	}
	if sh.Driver.CV != nil {
		n++
	}
	if n != 1 {
		return fmt.Errorf("shape %q: driver must set exactly one of single|cv, got %d", sh.ID, n)
	}
	if sh.Driver.CV != nil {
		return fmt.Errorf("shape %q: driver:cv (nested-CV) is metis#23 — M1a supports driver:single only", sh.ID)
	}
	return nil
}

// combinedSteps concatenates the three phases into one step list (data → pipeline →
// ship) for the combined-DAG validation. Order is data, pipeline, ship — but needs are
// resolved by id, not position, so any acyclic cross-phase edge is fine.
func combinedSteps(sh Shape) []Step {
	all := make([]Step, 0, len(sh.Data)+len(sh.Pipeline)+len(sh.Ship))
	all = append(all, sh.Data...)
	all = append(all, sh.Pipeline...)
	all = append(all, sh.Ship...)
	return all
}
