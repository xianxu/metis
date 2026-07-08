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
// term. The pure core lives here (Kpre / Validate / the Entry codec); the
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
// false-HIT and serve the wrong step's output.
//
// INPUT-ADDRESSED (metis#24): the executor feeds `rec.Upstream` = the sorted upstream
// steps' **Kpres** (input identity), NOT their output-hashes. So a step's key is its
// *input recipe* (config + seed + upstream input-identities), computable pre-run from
// the DAG and invariant to upstream output non-determinism. Upstream **code**-edit
// propagation is restored separately by the transitive-D snapshot (Entry.TransitiveD),
// since `D` deliberately excludes data/upstream artifacts. The entries are sorted so the
// key is invariant to the author's `needs` declaration order. (The record-provenance
// path keeps its own StepRecord with Upstream = output-hashes — a SEPARATE construction.)
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
func Validate(storedD []record.CodeRef, hash func(ref record.CodeRef) (record.Hash, error)) bool {
	for _, ref := range storedD {
		got, err := hash(ref)
		if err != nil || got != ref.BlobHash {
			return false
		}
	}
	return true
}

// Entry is a cache index record, stored at index/<K_pre>.json: for a K_pre, the
// read-set D recorded on the last miss plus Output — the CAS content-hash of the
// stored output manifest (metrics + artifact hashes). One index keyed by K_pre
// realizes the design's "output at hash(K_pre, D)" mapping: a lookup finds D (to
// re-hash) and, on a HIT, the output to materialize. (A separate derived OutputKey =
// hash(K_pre, D) was dropped — it can't be computed before a run, and cross-run dedup
// within a sweep is already provided by the K_pre-keyed index.) Persisted as small
// on-disk JSON so the index survives across runs, sessions, and branches. (v1 gitignores
// the whole cache dir; git-sharing the index across clones is a future enhancement.)
type Entry struct {
	Kpre Hash             `json:"kpre"`
	D    []record.CodeRef `json:"d"` // this step's OWN read-set (provenance / debug)
	// TransitiveD is the metis#24 soundness snapshot: this step's own D UNIONED with the
	// transitive closure of its upstream steps' D — captured at THIS step's computation and
	// stored in its OWN entry. isHit re-hashes TransitiveD (not D), so an edit to any
	// transitively-upstream code file invalidates this step (restoring the code-propagation
	// the input-addressed Kpre drops). Stored + validated on the SAME bytes (symmetric),
	// needs no upstream-entry lookup at validate (eviction-robust), and folds a diamond once.
	// NOT `omitempty`: a #24 entry ALWAYS serializes TransitiveD (as `[]` when the closure is
	// empty — MergeTransitiveD returns a non-nil slice), so an empty-closure step round-trips
	// to a non-nil slice and still HITs vacuously; only a LEGACY (pre-#24) entry decodes to a
	// nil TransitiveD, which isHit treats as a MISS (cmd/metis isHit migration guard).
	TransitiveD []record.CodeRef `json:"transitive_d"`
	Output      Hash             `json:"output"`
}

// MergeTransitiveD folds a step's transitive-D snapshot: its own read-set unioned with
// each already-computed upstream snapshot, deduped by (repo, path) and returned in a
// canonical sort order — so the persisted bytes are stable (a diamond S←A,S←B,both←R
// folds R once; order-independent). Pure: the topo-order accumulation lives in the
// executor (transitiveD[id] = MergeTransitiveD(ownD, transitiveD[needs]...)).
func MergeTransitiveD(ownD []record.CodeRef, upstream ...[]record.CodeRef) []record.CodeRef {
	seen := make(map[[2]string]record.CodeRef)
	add := func(refs []record.CodeRef) {
		for _, r := range refs {
			seen[[2]string{r.Repo, r.Path}] = r
		}
	}
	add(ownD)
	for _, u := range upstream {
		add(u)
	}
	out := make([]record.CodeRef, 0, len(seen))
	for _, r := range seen {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Repo != out[j].Repo {
			return out[i].Repo < out[j].Repo
		}
		return out[i].Path < out[j].Path
	})
	return out
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
