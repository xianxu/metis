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

// The provenance record (metis#3) — the L0 reproducibility atom, emitted as
// runs/<id>/record.json. Field names are snake_case to match the Go json tags
// (pkg/record). Like #Run there is no `type` discriminator, so the drift guard
// `cue vet`s a marshaled RunRecord against #RunRecord (closed schema → a renamed /
// removed / extra field fails). Fields metis#2/#8 populate (read-set d, deps) are
// OPTIONAL here — metis#3 fills only the coarse code identity (commit + dirty).

#FileHash: {
	path: string
	hash: string   // content hash of the file's bytes
}

#CodeRef: {       // one file of the read-set D, pinned by its git blob-hash
	path:      string
	blob_hash: string
}

#CodeManifest: {
	commit: string          // the commit the code closure was captured at
	dirty:  bool            // was the repo dirty at run time
	d?: [...#CodeRef]       // read-set; metis#2's validating trace populates
	deps?: string           // uv.lock digest; metis#2 populates
}

#StepRecord: {
	// key-material (metis#2 hashes into the cache key):
	step_id: string
	uses:    string
	with?: {[string]: _}    // resolved config
	upstream?: [...string]  // upstream output hashes
	code: #CodeManifest
	// provenance-only extras:
	output_hash?: string
	metrics?: {[string]: number}
}

#RunRecord: {
	run_id:        string
	experiment:    string
	seed:          int
	point_address: string          // the minted L0 run-identity
	repo_shas?: {[string]: string} // repo-name → SHA at run time
	dirty: bool
	steps: [...#StepRecord]
	started:   string
	finished?: string
	status:    string
}
