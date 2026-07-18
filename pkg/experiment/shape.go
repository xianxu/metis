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
// and `ship` (winner-only, activated by `metis select --promote`) — plus a `sweeper`
// (the config-level Sampler: sampler + inner resample + objective+select). metis#32
// DELETED the `driver:` field: the run mode is now DERIVED from the config count
// (`metis run` on >1 config → nested CV; ==1 → a flat single-level CV), so the outer
// evaluator is no longer a declared knob. Each phase is a []Step DAG, reusing the
// experiment Step/Validate machinery rather than restating the DAG (ARCH-DRY). The
// `$`-key value-algebra lives in the untyped `with` bags (pkg/shape expands it).
type Shape struct {
	Header   `yaml:",inline"` // shared type/id/competition/seed/status (ARCH-DRY, mirrors CUE _meta)
	Data     []Step           `yaml:"data"`
	Pipeline []Step           `yaml:"pipeline"`
	Ship     []Step           `yaml:"ship"`
	Sweeper  Sweeper          `yaml:"sweeper"`
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

// CVResample is k-fold cross-validation config. K is the ESTIMAND knob — the outer
// driver's fold count (train fraction each outer fold simulates, metis#42's principle) AND
// the inner default. InnerK (metis#45, optional) overrides the INNER per-config CV only —
// the selection-precision/cost knob; a flat (single-config) run has no inner level and
// ignores it (loudly). json omitempty is LOAD-BEARING: the Sweeper struct reaches
// record.CanonicalHash (shapeRunIdentity) — an absent inner_k must not enter the hash.
type CVResample struct {
	K        int  `yaml:"k"`
	InnerK   int  `yaml:"inner_k,omitempty" json:"inner_k,omitempty"` // metis#45: inner-CV override (0 = use K)
	Stratify bool `yaml:"stratify,omitempty"`
}

// InnerFolds is the inner per-config CV's fold count — inner_k if declared, else k. The ONE
// derivation (metis#45); no consumer reads InnerK directly.
func (c CVResample) InnerFolds() int {
	if c.InnerK > 0 {
		return c.InnerK
	}
	return c.K
}

// Objective names the metric to optimize, the direction, and the select rule that
// reduces per-config (mean, SE, complexity) → a winner (metis#19).
type Objective struct {
	Metric    string `yaml:"metric"`
	Direction string `yaml:"direction"` // "maximize" | "minimize"
	Select    Select `yaml:"select"`
}

// Select is the tagged-union select rule (metis#19) — exactly one branch is non-nil
// (optional pointer fields + a Go "exactly one" count check in ValidateShape; each rule's
// param is bound to it as a sub-struct field). The parsimony rules (one-std-err/pct-loss)
// minimize the per-config MEASURED complexity within a band, tie-break by mean;
// argmax-mean/mean-std ignore complexity.
type Select struct {
	ArgmaxMean *ArgmaxMean `yaml:"argmax-mean,omitempty"` // raw cv-max (M1a); mean only
	OneStdErr  *OneStdErr  `yaml:"one-std-err,omitempty"` // band = 1×SE, then min-complexity
	PctLoss    *PctLoss    `yaml:"pct-loss,omitempty"`    // band = tolerance %, then min-complexity
	MeanStd    *MeanStd    `yaml:"mean-std,omitempty"`    // argmax(mean − λ·std); no complexity
}

type ArgmaxMean struct{}
type OneStdErr struct{}
type PctLoss struct {
	Tolerance float64 `yaml:"tolerance"` // relative fraction of the family-best mean (0.02 = 2%)
}
type MeanStd struct {
	Lambda float64 `yaml:"lambda"`
}

// Kind returns the single set branch's name and true; "" and false if not exactly one is set.
func (s Select) Kind() (string, bool) {
	name, n := "", 0
	if s.ArgmaxMean != nil {
		name, n = "argmax-mean", n+1
	}
	if s.OneStdErr != nil {
		name, n = "one-std-err", n+1
	}
	if s.PctLoss != nil {
		name, n = "pct-loss", n+1
	}
	if s.MeanStd != nil {
		name, n = "mean-std", n+1
	}
	if n != 1 {
		return "", false
	}
	return name, true
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
// sampler, a valid inner resample, a valid objective direction, and exactly one select-rule
// branch (metis#19). metis#32: no driver-mode check (the field is gone; mode is run-derived).
func ValidateShape(sh Shape) error {
	combined := Experiment{Header: sh.Header, Steps: combinedSteps(sh)}
	combined.Type = "experiment"
	if err := Validate(combined); err != nil {
		return err
	}
	// Phase-ordering: a step may only `needs` steps in an earlier-or-equal phase
	// (data=0 │ pipeline=1 │ ship=2). A backward edge (e.g. a `data` step depending on a
	// `pipeline` step) validates clean under acyclicity but would silently break the
	// data│pipeline run-once/per-fold leakage cut when M1a-4 wires execution — reject it
	// here with a sharp diagnostic. (needs already resolve, per Validate above.)
	if err := validatePhaseOrdering(sh); err != nil {
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
	if ik := sh.Sweeper.Resample.CV.InnerK; ik != 0 && ik < 2 {
		return fmt.Errorf("shape %q: sweeper.resample.cv.inner_k must be >= 2 when set (or absent to use k), got %d", sh.ID, ik)
	}
	// Match CUE's required objective (metric + direction present); Go was looser (empty
	// direction / absent metric passed) — a semantic validator should not be laxer than
	// the structural one.
	if sh.Sweeper.Objective.Metric == "" {
		return fmt.Errorf("shape %q: sweeper.objective.metric is required", sh.ID)
	}
	if d := sh.Sweeper.Objective.Direction; d != "maximize" && d != "minimize" {
		return fmt.Errorf("shape %q: sweeper.objective.direction %q must be maximize|minimize", sh.ID, d)
	}
	// Exactly one select branch. Params bound to
	// their branch: pct-loss.tolerance > 0, mean-std.lambda >= 0.
	if _, ok := sh.Sweeper.Objective.Select.Kind(); !ok {
		return fmt.Errorf("shape %q: sweeper.objective.select must set exactly one of argmax-mean|one-std-err|pct-loss|mean-std", sh.ID)
	}
	if p := sh.Sweeper.Objective.Select.PctLoss; p != nil && p.Tolerance <= 0 {
		return fmt.Errorf("shape %q: sweeper.objective.select.pct-loss.tolerance must be > 0, got %v", sh.ID, p.Tolerance)
	}
	if m := sh.Sweeper.Objective.Select.MeanStd; m != nil && m.Lambda < 0 {
		return fmt.Errorf("shape %q: sweeper.objective.select.mean-std.lambda must be >= 0, got %v", sh.ID, m.Lambda)
	}
	// metis#32: no `driver:` validation — the field is gone; the run mode is derived from the
	// config count at run time (nested for >1 config, single-level CV for 1). The outer-CV fold
	// count reuses sweeper.resample.cv.k (validated ≥2 above).
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

// phaseNames indexes the phase order: data(0) │ pipeline(1) │ ship(2).
var phaseNames = [...]string{"data", "pipeline", "ship"}

// validatePhaseOrdering enforces that every `needs` edge runs monotonically by phase — a
// step in phase P may only depend on steps in phase ≤ P. This defends the data│pipeline
// structural cut: a backward edge (a `data`/`pipeline` step reaching forward into a later
// phase) is acyclic-legal but would corrupt the run-once/per-fold execution invariant.
// Assumes needs already resolve (Validate ran first), so every need is in the phase map.
func validatePhaseOrdering(sh Shape) error {
	phase := map[string]int{}
	for _, s := range sh.Data {
		phase[s.ID] = 0
	}
	for _, s := range sh.Pipeline {
		phase[s.ID] = 1
	}
	for _, s := range sh.Ship {
		phase[s.ID] = 2
	}
	for _, s := range combinedSteps(sh) {
		for _, need := range s.Needs {
			if phase[need] > phase[s.ID] {
				return fmt.Errorf("shape %q: %s step %q needs %s step %q — a step may not depend on a later phase (defends the data│pipeline leakage cut)",
					sh.ID, phaseNames[phase[s.ID]], s.ID, phaseNames[phase[need]], need)
			}
		}
	}
	return nil
}
