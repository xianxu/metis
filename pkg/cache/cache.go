// Package cache is the metis#2 content-addressed step cache — the *policy* layer
// (keying + skip/recompute decision) over the #9 CAS (byte storage) and the #3
// per-step record (key-material). It is the "validating trace": a step's output is
// keyed by two hashes computed at different times —
//
//   - K_pre (ex-ante): hash(step-id, uses, resolved-with, seed, upstream-output-hashes)
//     — everything determining the output that is known BEFORE the step runs.
//   - D (ex-post): the recorded read-set of first-party code+config files, each an
//     (path, git-blob-hash) pair, captured by the metis#2 read-sensor after a run.
//
// A run is a HIT iff a stored entry for its K_pre exists AND re-hashing every path in
// its D still matches (an edited code file is a path in D whose hash moved → MISS).
// Code-version invalidation thus falls out of the trace — no git-SHA / import-closure
// term. The pure core lives here (Kpre / Validate / OutputKey / the Entry codec); the
// read-sensor and the git blob-hasher (metis#2 M2) and the runner skip/materialize
// integration (M3) are the thin IO shell in cmd/metis.
package cache

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/xianxu/metis/pkg/cas"
	"github.com/xianxu/metis/pkg/record"
)

// Hash re-exports the content-hash type (identical to cas.Hash / record.Hash).
type Hash = cas.Hash

// Kpre computes the ex-ante cache key from a step's determinants known before it runs.
// seed is an explicit arg because it lives on RunRecord, not StepRecord. uses is
// included so two steps sharing id/with/seed/upstream but a different step-type cannot
// false-HIT and serve the wrong step's output. The upstream output-hashes are sorted
// so the key is invariant to the author's `needs` declaration order.
func Kpre(rec record.StepRecord, seed int) (Hash, error) {
	upstream := sortedHashes(rec.Upstream)
	h, err := record.CanonicalHash(struct {
		StepID   string         `json:"step_id"`
		Uses     string         `json:"uses"`
		With     map[string]any `json:"with"`
		Seed     int            `json:"seed"`
		Upstream []record.Hash  `json:"upstream"`
	}{rec.StepID, rec.Uses, rec.With, seed, upstream})
	if err != nil {
		return "", fmt.Errorf("cache: K_pre: %w", err)
	}
	return h, nil
}

// Validate reports whether a stored read-set D still matches the working tree — a HIT
// (reuse the cached output) vs a MISS (re-run). It re-hashes each path in D via the
// injected hasher (the caller supplies git-blob-hashing) and compares. Any mismatch,
// or a hasher failure (a vanished/unreadable file), is a MISS — the safe direction,
// since a MISS only recomputes; it never serves stale bytes. An empty D is a vacuous
// HIT: the step read no first-party code to invalidate, so K_pre alone determines it.
func Validate(storedD []record.CodeRef, hash func(path string) (record.Hash, error)) bool {
	for _, ref := range storedD {
		got, err := hash(ref.Path)
		if err != nil || got != ref.BlobHash {
			return false
		}
	}
	return true
}

// OutputKey is the CAS address of a step's output for a given (values × code) pair:
// hash(K_pre, hash(D)). K_pre carries the values (params/seed/upstream); the D-hash
// carries the code+config bytes. Same code + different `with` → shared D-hash,
// different K_pre; a code edit → shared K_pre, different D-hash. Invariant to D order.
func OutputKey(kpre Hash, storedD []record.CodeRef) (Hash, error) {
	dHash, err := record.CanonicalHash(sortedRefs(storedD))
	if err != nil {
		return "", fmt.Errorf("cache: output-key D-hash: %w", err)
	}
	h, err := record.CanonicalHash(struct {
		Kpre  Hash `json:"kpre"`
		DHash Hash `json:"d_hash"`
	}{kpre, dHash})
	if err != nil {
		return "", fmt.Errorf("cache: output-key: %w", err)
	}
	return h, nil
}

// Entry is a cache index record: for a K_pre, the read-set D recorded on the last
// miss plus the output key its bytes live under in the CAS. Persisted as small
// git-trackable JSON so the index survives across runs, sessions, and branches.
type Entry struct {
	Kpre      Hash             `json:"kpre"`
	D         []record.CodeRef `json:"d"`
	OutputKey Hash             `json:"output_key"`
}

// EncodeEntry / DecodeEntry are the index codec (pure) — the thin IO layer (M3) reads
// and writes these under cache/index/<K_pre>.json.
func EncodeEntry(e Entry) ([]byte, error) { return json.MarshalIndent(e, "", "  ") }

func DecodeEntry(b []byte) (Entry, error) {
	var e Entry
	err := json.Unmarshal(b, &e)
	return e, err
}

func sortedHashes(in []record.Hash) []record.Hash {
	out := append([]record.Hash(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func sortedRefs(in []record.CodeRef) []record.CodeRef {
	out := append([]record.CodeRef(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
