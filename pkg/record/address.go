package record

import (
	"encoding/json"
	"fmt"
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
//
// It returns an error (rather than panicking) when the config is not JSON-
// marshalable: a non-finite float (.inf/.nan is valid YAML → float64 Inf/NaN, which
// json.Marshal rejects) is user-reachable input, so the caller surfaces it as a run
// error instead of crashing the run.
func PointAddress(resolvedWith map[string]map[string]any, repoSHAs map[string]string, seed int) (Hash, error) {
	payload := struct {
		ResolvedWith map[string]map[string]any `json:"resolved_with"`
		RepoSHAs     map[string]string         `json:"repo_shas"`
		Seed         int                       `json:"seed"`
	}{resolvedWith, repoSHAs, seed}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("record: point-address config not canonicalizable (non-finite value?): %w", err)
	}
	return cas.HashOf(b), nil
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
	// Unlike PointAddress, the manifest is strings-only (paths + hex hashes), so
	// json.Marshal is total here — the error is unreachable, guarded as an invariant.
	b, err := json.Marshal(sorted)
	if err != nil {
		panic("record: OutputHash string manifest not marshalable (unreachable): " + err.Error())
	}
	return cas.HashOf(b)
}
