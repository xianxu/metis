// Package record is the unified per-step provenance record (metis#3) — the L0
// reproducibility atom the whole v1 cache/ledger chain keys off. One raw record per
// step per run, whose fields split by role: key-material (the determinants metis#2
// hashes into the cache key) vs. provenance-only extras (reconstruction aids +
// legibility). Three views hash LATE over the one record: the point-address (this
// package — the repro/run-identity: resolved-with + repo-SHAs + seed), the cache key
// (metis#2 — key-material incl. the read-set), and the output key (a step's output
// CAS address). Pure: PointAddress and OutputHash are deterministic, no IO — the
// cmd/metis thin shell does the git/filesystem reads and feeds them in.
package record

import "github.com/xianxu/metis/pkg/cas"

// Hash re-exports the content-hash type so record consumers need not import cas for
// it; identical to cas.Hash (git's blob-hash / a CAS address / hex sha256).
type Hash = cas.Hash

// FileHash pairs an output artifact's repo-relative path with its content hash.
type FileHash struct {
	Path string `json:"path"`
	Hash Hash   `json:"hash"`
}

// CodeRef pins one source file by its git blob-hash — git's blob-hash IS the
// content-hash (metis stores no code bytes). One entry of the read-set D.
type CodeRef struct {
	Path     string `json:"path"`
	BlobHash Hash   `json:"blob_hash"`
}

// CodeManifest identifies the code a step ran. metis#3 fills the coarse identity
// (Commit + Dirty, from the current repo state). The read-set D + Deps (uv.lock
// digest) are defined slots that stay empty in the RECORD for now: metis#2 built the
// validating trace and populates the *cache's* functional read-set (cache.Entry.D),
// but the record's code-manifest *provenance* population is deferred to metis#8 (the
// git-side-ref durability that captures the code closure).
type CodeManifest struct {
	Commit string    `json:"commit"`
	Dirty  bool      `json:"dirty"`
	D      []CodeRef `json:"d,omitempty"`    // read-set — record provenance deferred to metis#8
	Deps   string    `json:"deps,omitempty"` // uv.lock digest — deferred to metis#8
}

// StepRecord is one step's raw provenance record. Fields split by role:
//   - key-material (metis#2 hashes into the cache key): StepID, Uses, With
//     (resolved), Upstream (upstream output hashes), Code.
//   - provenance-only extras (NOT in the cache key): OutputHash, Metrics.
type StepRecord struct {
	StepID   string         `json:"step_id"`
	Uses     string         `json:"uses"`
	With     map[string]any `json:"with,omitempty"`
	Upstream []Hash         `json:"upstream,omitempty"`
	Code     CodeManifest   `json:"code"`

	OutputHash Hash               `json:"output_hash,omitempty"`
	Metrics    map[string]float64 `json:"metrics,omitempty"`
}

// RunRecord is the provenance record for one run: the DAG of step records plus the
// minted point-address (the L0 run-identity), repo-SHAs + dirty flag, and status.
// Small metadata → git (durable-small), never the CAS.
type RunRecord struct {
	RunID        string            `json:"run_id"`
	Experiment   string            `json:"experiment"`
	Seed         int               `json:"seed"`
	PointAddress Hash              `json:"point_address"`
	RepoSHAs     map[string]string `json:"repo_shas,omitempty"`
	Dirty        bool              `json:"dirty"`
	Steps        []StepRecord      `json:"steps"`
	Started      string            `json:"started"`
	Finished     string            `json:"finished,omitempty"`
	Status       string            `json:"status"`
}
