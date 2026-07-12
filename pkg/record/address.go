package record

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/xianxu/metis/pkg/cas"
)

// CanonicalHash content-addresses any JSON-marshalable value by hashing its
// canonical JSON form (Go's encoding/json sorts map keys at every nesting level, so
// map-iteration order never perturbs the hash). It is the shared hashing primitive
// behind the point-address, the output key, and metis#2's cache key — one canonical
// form so those three views compare like-for-like. v1 uses json.Marshal; a stricter
// RFC-8785 canonicalizer can slot in later without changing callers.
//
// It returns an error (rather than panicking) when v is not JSON-marshalable: a
// non-finite float (.inf/.nan is valid YAML → float64 Inf/NaN, which json.Marshal
// rejects) is user-reachable config, so the caller surfaces it as a run error.
func CanonicalHash(v any) (Hash, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("record: value not canonicalizable (non-finite float?): %w", err)
	}
	return cas.HashOf(b), nil
}

// PointAddress mints the L0 INTENT-identity: the content-address of the resolved config
// across the whole DAG, the shape file's git blob-hash, and the seed. It is the pre-run
// "what I meant to run" key metis#8's ledger derives from — pure inputs, computable
// before the run (the shape blob-hash is git-hash-object'd up front). It deliberately
// does NOT carry code identity (repo_shas was dropped in metis#27) — code identity is
// the POST-run code_fingerprint (CodeFingerprint) over the run's read-set D closure, so
// same-config-different-code runs are distinguished by fingerprint, not conflated here.
func PointAddress(resolvedWith map[string]map[string]any, shapeBlobHash string, seed int) (Hash, error) {
	h, err := CanonicalHash(struct {
		ResolvedWith  map[string]map[string]any `json:"resolved_with"`
		ShapeBlobHash string                    `json:"shape_blob_hash"`
		Seed          int                       `json:"seed"`
	}{resolvedWith, shapeBlobHash, seed})
	if err != nil {
		return "", fmt.Errorf("point-address: %w", err)
	}
	return h, nil
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
	// The manifest is strings-only (paths + hex hashes), so CanonicalHash is total
	// here — absorb the unreachable error internally so this stays a total Hash
	// (don't ripple a new error return out to hashArtifacts/assembleRecord).
	h, err := CanonicalHash(sorted)
	if err != nil {
		panic("record: OutputHash string manifest not canonicalizable (unreachable): " + err.Error())
	}
	return h
}
