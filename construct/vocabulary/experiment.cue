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

// _meta is the shared header field set — defined ONCE (ARCH-DRY) and embedded into the
// singleton #Experiment (via _pipeline) and the phase-structured #ExperimentShape, so
// id/competition/seed/status aren't hand-maintained in two places. (A hidden field, not
// a `#` def, so each embedder stays closed over exactly its own fields.)
_meta: {
	id:           string   // slug; matches filename
	competition?: string   // set on kbench instances; absent on metis fixtures
	seed:         int
	status:       #Status
}

// _phase is one phase's step DAG — a metis#18 shape is three of these (data|pipeline|ship).
_phase: [...#Step]

// _pipeline is the singleton experiment's field set: the shared header + a flat step DAG.
_pipeline: {
	_meta
	steps: _phase          // the pipeline DAG (may be a single step)
}

// #Experiment is the SINGLETON case: the shared pipeline narrowed to `type:
// "experiment"` with no sweeper. Closed (no stray fields) for sharp diagnostics. A
// resolved per-fold/ship run (every $any collapsed) is one of these.
#Experiment: {
	_pipeline
	type: "experiment"
}

// #ExperimentShape (metis#18 v2) is the experiment lifted into a config-space AND
// structured into three phases — `data` (once, above the resample) │ `pipeline` (the
// swept algorithm×hyperparameter atom, per-fold) │ `ship` (winner-only) — plus a
// `sweeper` (config sampler + inner resample + objective/select). metis#32 dropped the
// `driver:` field — the run mode is derived from the config count. The `$`-key
// value-algebra lives in the untyped `with` bag (value-
// level, NOT CUE-typed — pkg/shape expands it). Closed (no `...`) for sharp diagnostics.
#ExperimentShape: {
	_meta
	type: "experiment-shape"
	data:     _phase       // produced once, above the resample (shared across folds)
	pipeline: _phase       // the swept algorithm×hyperparameter atom (run per-fold)
	ship:     _phase       // winner-only (predict/submission)
	sweeper: {
		sampler: string                     // "grid" (M1a); the ask/tell seam is metis#7
		resample: {cv: {k: int, stratify?: bool}} // the inner CV — how each config is scored
		objective: {
			metric:    string
			direction: "maximize" | "minimize"
			select: {                        // metis#19 tagged union; exactly-one enforced in Go
				"argmax-mean"?: {}
				"one-std-err"?: {}
				"pct-loss"?: {tolerance: float & >0}
				"mean-std"?: {lambda:    float & >=0}
			}
		}
	}
	// metis#32: no `driver:` field — the run mode is DERIVED from the config count
	// (`metis run` on >1 config → nested CV; ==1 → a flat single-level CV), so the outer
	// evaluator is no longer a declared knob. The outer-CV fold count reuses sweeper.resample.cv.k.
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
	repo?:     string // repo root the path is relative to (metis#11: D can span repos)
	path:      string
	blob_hash: string
}

#CodeManifest: {
	commit: string          // the commit the code closure was captured at
	dirty:  bool            // was the repo dirty at run time
	d?: [...#CodeRef]       // read-set; metis#2's validating trace populates
	deps?: string           // uv.lock digest; metis#2 populates
	capture_status?: "captured" | "degraded" | "none" // metis#14: durable-snapshot status
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
	point_address:     string // the minted L0 intent-identity (config + shape-blob + seed)
	code_fingerprint?: string // the realized code identity over the run's D closure (metis#27)
	dirty: bool
	steps: [...#StepRecord]
	started:   string
	finished?: string
	status:    string
}
