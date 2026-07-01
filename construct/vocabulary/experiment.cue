// experiment — the vocabulary of a reproducible ML pipeline experiment.
// An experiment file's YAML frontmatter is validated against #Experiment
// (structural conformance via `cue vet`, invoked by
// `vocabulary validate-instance --type experiment <file>`).
//
// SCOPE: this file owns SHAPE only — types, enums, required fields, the steps
// list-of-structs. SEMANTIC checks (needs → a real step id, DAG acyclicity,
// `uses` = "<layer>/<steptype>") are NOT expressible in `cue vet` and land with
// metis#1 M2's pure Go validator. Closed schema (no `...`) for sharp
// diagnostics — an experiment's frontmatter is fully known here.
package experiment

#Status: "draft" | "active" | "archived"

#Step: {
	id:   string          // unique within the experiment
	uses: string          // "<layer>/<steptype>", e.g. "metis/cv-split"
	needs?: [...string]   // ids of steps this one depends on (DAG edges)
	with?: {[string]: _}  // free config map; typed per step-type in M3
}

#Run: {
	id:         string               // run slug, e.g. "run-003"
	experiment: string               // parent experiment id
	seed:       int
	started:    string               // ISO datetime
	finished?:  string
	status:     "ok" | "failed"
	metrics?: {[string]: number}
	artifacts?: [...string]          // repo-relative paths under runs/<id>/
}

#Experiment: {
	type:         "experiment"
	id:           string   // slug; matches filename
	competition?: string   // set on kbench instances; absent on metis fixtures
	seed:         int
	status:       #Status
	steps: [...#Step]      // the pipeline (may be a single step)
}
