package record

import (
	"encoding/json"
	"sort"

	"github.com/xianxu/metis/pkg/cas"
)

// PointAddress mints the L0 run-identity: the content-address of the resolved config
// across the whole DAG, the repo SHAs, and the seed. It is the repro/identity key
// metis#8's ledger derives from — deliberately git-SHA-based and coarse (NOT a
// per-step read-set trace; that is metis#2's cache key). Pure + deterministic: the
// payload is canonicalized to JSON (Go's encoding/json sorts map keys, including the
// nested maps in resolvedWith) so map-iteration order never perturbs the address. v1
// uses json.Marshal as the canonical form; a stricter RFC-8785 canonicalizer can
// slot in later without changing callers.
func PointAddress(resolvedWith map[string]map[string]any, repoSHAs map[string]string, seed int) Hash {
	payload := struct {
		ResolvedWith map[string]map[string]any `json:"resolved_with"`
		RepoSHAs     map[string]string         `json:"repo_shas"`
		Seed         int                       `json:"seed"`
	}{resolvedWith, repoSHAs, seed}
	b, err := json.Marshal(payload)
	if err != nil {
		// resolvedWith values come from YAML→any (JSON-marshalable); an un-marshalable
		// value is a programming error at the call site, not a runtime condition.
		panic("record: PointAddress payload not marshalable: " + err.Error())
	}
	return cas.HashOf(b)
}

// OutputHash reduces a step's output artifacts — a SET of files — to a single
// content-address: it hashes a canonical manifest of (relpath, content-hash) pairs
// sorted by path, so the address is stable regardless of walk order and changes iff
// a path or its bytes change. The empty set yields a defined, stable hash (nil and
// empty normalize to the same manifest). Pure; the caller (cmd/metis) computes each
// file's content-hash via cas.HashOf and feeds the FileHash list in.
func OutputHash(files []FileHash) Hash {
	sorted := make([]FileHash, len(files)) // non-nil even for a nil arg → "[]" not "null"
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	b, err := json.Marshal(sorted)
	if err != nil {
		panic("record: OutputHash manifest not marshalable: " + err.Error())
	}
	return cas.HashOf(b)
}
