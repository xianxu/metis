// Package cas is a content-addressed blob store: put/get/has bytes keyed by their
// sha256 content hash. It is MECHANISM ONLY — no cache-keying, provenance, or
// durable-retention semantics (those live in metis#2 / #3 / #8). The key is the
// content hash, so a hash always maps to the same bytes and identical content is
// stored once (self-deduplicating, exactly what git's object store is).
//
// In v1 the store is a PURE WIPEABLE CACHE: any entry may be evicted with no
// correctness impact — a wiped entry is recomputed from durable roots (git owns
// code/config; external data refetches). So a filesystem pool carries size-bounded
// LRU eviction and `rm -rf` on it is always safe. The Store interface is the
// swappable seam: MemStore is the in-memory fake consumers test against, and an S3
// backend can slot in behind it later (S3 itself is out of v1).
package cas

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"time"
)

// Hash is the hex-encoded sha256 of a blob's bytes — the store's key. Content
// alone determines it, so identical bytes always map to the same Hash.
type Hash string

// HashOf returns the content hash of data.
func HashOf(data []byte) Hash {
	sum := sha256.Sum256(data)
	return Hash(hex.EncodeToString(sum[:]))
}

// Clock returns the current time. Injected so a store's eviction recency is
// deterministic in tests — no direct wall-clock calls in the core (controllable
// time as architecture). Mirrors pkg/experiment's Clock convention; kept local so
// the storage floor takes no dependency on higher layers.
type Clock func() time.Time

// ErrNotFound is returned by Get for a hash the store does not hold (never stored,
// or evicted from the wipeable cache).
var ErrNotFound = errors.New("cas: blob not found")

// ErrCorrupt is returned by Get when stored bytes no longer hash to their key —
// on-disk corruption. A wipeable-cache consumer treats it like ErrNotFound and
// recomputes.
var ErrCorrupt = errors.New("cas: blob failed integrity check")

// Store is the swappable content-addressed blob interface. The key IS the content
// hash, so Put is idempotent and self-deduplicating.
type Store interface {
	// Put stores data and returns its content hash. Storing identical bytes again
	// is a no-op that returns the same hash.
	Put(data []byte) (Hash, error)
	// Get returns the bytes for h, re-hashing to verify integrity: a mismatch
	// (corruption) returns ErrCorrupt, an absent entry ErrNotFound.
	Get(h Hash) ([]byte, error)
	// Has reports whether h is present.
	Has(h Hash) (bool, error)
}

// entry is a stored blob's eviction-relevant metadata: its hash, byte size, and
// last-access time (an FS store stamps this from the injected clock on put/get).
type entry struct {
	hash  Hash
	size  int64
	atime time.Time
}

// selectEvictions is the pure eviction-victim math: given the current pool of
// entries and a maxBytes budget, it returns the least-recently-used entries to
// delete so the remaining total is ≤ maxBytes. Kept pure — the FS store feeds it a
// directory listing and deletes what it returns — so the policy is unit-tested with
// no filesystem (ARCH-PURE). maxBytes ≤ 0 means unbounded (evict nothing). The
// just-written entry (keep) is never a victim, so a Put always leaves its own blob
// retrievable even if that blob alone exceeds the budget (best-effort budget).
// Victims are chosen oldest-atime-first; hash breaks ties for determinism.
func selectEvictions(entries []entry, maxBytes int64, keep Hash) []Hash {
	if maxBytes <= 0 {
		return nil
	}
	var total int64
	for _, e := range entries {
		total += e.size
	}
	if total <= maxBytes {
		return nil
	}
	candidates := make([]entry, 0, len(entries))
	for _, e := range entries {
		if e.hash != keep {
			candidates = append(candidates, e)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].atime.Equal(candidates[j].atime) {
			return candidates[i].hash < candidates[j].hash
		}
		return candidates[i].atime.Before(candidates[j].atime)
	})
	var victims []Hash
	for _, e := range candidates {
		if total <= maxBytes {
			break
		}
		victims = append(victims, e.hash)
		total -= e.size
	}
	return victims
}
