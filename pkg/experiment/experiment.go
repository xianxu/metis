// Package experiment is the pure core of metis's step-runner: the Go structs
// mirroring the CUE #Experiment/#Step/#Run (construct/vocabulary/experiment.cue),
// plus Parse/Validate/TopoSort. No IO lives here — Parse takes a string, Validate
// and TopoSort take a value; the subprocess step execution and the filesystem
// run-ledger are the thin cmd/metis layer (ARCH-PURE). CUE remains the single
// STRUCTURAL source; this package adds only the semantics CUE cannot express
// (needs-resolution, uses-format, acyclicity) — the drift between the two is
// guarded by TestParse_ConformsToCUE (ARCH-DRY).
package experiment

import (
	"fmt"

	"github.com/xianxu/ariadne/pkg/frontmatter"
	"gopkg.in/yaml.v3"
)

// Step mirrors CUE #Step: one node in the experiment's pipeline DAG.
type Step struct {
	ID    string         `yaml:"id"`              // unique within the experiment
	Uses  string         `yaml:"uses"`            // "<layer>/<steptype>", e.g. "metis/cv-split"
	Needs []string       `yaml:"needs,omitempty"` // ids of steps this one depends on (DAG edges)
	With  map[string]any `yaml:"with,omitempty"`  // free config map; typed per step-type in M3
}

// Header is the shared identity/config prefix of both an Experiment and its lifted
// Shape (metis#18) — declared ONCE here and embedded `yaml:",inline"` into both, mirroring
// the CUE `_meta` single-source (ARCH-DRY), so adding a header field is a one-place edit.
type Header struct {
	Type        string `yaml:"type"`
	ID          string `yaml:"id"`
	Competition string `yaml:"competition,omitempty"`
	Seed        int    `yaml:"seed"`
	Status      string `yaml:"status"`
}

// Experiment mirrors CUE #Experiment: the shared Header plus the pipeline (steps),
// read from a markdown file's YAML frontmatter.
type Experiment struct {
	Header `yaml:",inline"`
	Steps  []Step `yaml:"steps"`
}

// Run mirrors CUE #Run: one recorded execution, written to runs/<id>/run.json by
// the runner. JSON tags match the CUE field names so the ledger stays validatable.
type Run struct {
	ID         string             `yaml:"id" json:"id"`
	Experiment string             `yaml:"experiment" json:"experiment"`
	Seed       int                `yaml:"seed" json:"seed"`
	Started    string             `yaml:"started" json:"started"`
	Finished   string             `yaml:"finished,omitempty" json:"finished,omitempty"`
	Status     string             `yaml:"status" json:"status"`
	Metrics    map[string]float64 `yaml:"metrics,omitempty" json:"metrics,omitempty"`
	Artifacts  []string           `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
}

// Parse splits an experiment markdown document into frontmatter + body (reusing
// ariadne's frontmatter.Split — ARCH-DRY, no reimplemented fence parsing) and
// unmarshals the YAML frontmatter into an Experiment. Pure: string → (Experiment,
// error), no IO. Structural conformance is CUE's job (validate-instance); Validate
// adds the semantic checks.
func Parse(content string) (Experiment, error) {
	fm, _, err := frontmatter.Split(content)
	if err != nil {
		return Experiment{}, err
	}
	var exp Experiment
	if err := yaml.Unmarshal([]byte(fm), &exp); err != nil {
		return Experiment{}, fmt.Errorf("parse experiment frontmatter: %w", err)
	}
	return exp, nil
}
