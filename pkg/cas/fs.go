package cas

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// tmpPrefix marks in-flight atomic writes so the eviction scan skips them.
const tmpPrefix = ".tmp-"

// FSStore is a filesystem-backed Store: blobs live at root/<h[:2]>/<h> (sharded so
// no single directory holds the whole pool). Writes are atomic (temp file +
// rename), reads re-hash to verify integrity, and every put/get stamps the blob's
// mtime from the injected clock so eviction recency is deterministic. Size-bounded
// LRU eviction (maxBytes; ≤ 0 = unbounded) keeps the pool under budget — because
// the store is a pure wipeable cache, any evicted blob is simply recomputed.
//
// Concurrency: individual Put/Get/Has are safe to call from multiple goroutines
// (atomic writes, content-addressed paths). But under a tight maxBytes with
// concurrent Puts the store is best-effort, NOT strictly isolating: each Put's
// evict pass protects only its own just-written blob, so goroutine A's eviction may
// delete a blob goroutine B just wrote, after which B's next Get returns
// ErrNotFound. That is deliberately tolerable under the wipeable-cache contract — a
// missing blob is recomputed — but a consumer must not assume a Put by another
// goroutine stays resident. The recency stamp (Chtimes) is likewise best-effort: a
// failed stamp never fails an otherwise-valid Put or Get (see touch).
type FSStore struct {
	root     string
	maxBytes int64
	now      Clock
}

// NewFSStore returns a filesystem store rooted at root. maxBytes ≤ 0 is unbounded.
// now defaults to time.Now when nil. No IO happens here — directories are created
// lazily on Put — so a store over a not-yet-existing root reads as empty.
func NewFSStore(root string, maxBytes int64, now Clock) *FSStore {
	if now == nil {
		now = time.Now
	}
	return &FSStore{root: root, maxBytes: maxBytes, now: now}
}

// shardPath returns the on-disk path for h and whether h is a well-formed key. A
// base primitive must not turn an arbitrary key string into a filesystem path:
// anything that isn't the exact shape HashOf produces (64 lowercase hex chars) is
// rejected as absent, so no path separator or `..` can escape root and Get/Has
// answer ErrNotFound/false for a malformed key.
func (s *FSStore) shardPath(h Hash) (string, bool) {
	if !isHash(h) {
		return "", false
	}
	return filepath.Join(s.root, string(h)[:2], string(h)), true
}

// isHash reports whether h is exactly 64 lowercase hex characters — the shape
// HashOf emits (hex-encoded sha256).
func isHash(h Hash) bool {
	if len(h) != 64 {
		return false
	}
	for i := 0; i < len(h); i++ {
		c := h[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func (s *FSStore) Put(data []byte) (Hash, error) {
	h := HashOf(data)
	p, _ := s.shardPath(h) // HashOf always yields a well-formed 64-hex hash
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	// Dedup, but never trust file existence alone: skip the write only if the
	// existing blob still hashes to h. An absent OR corrupt blob falls through to
	// the atomic overwrite below — which HEALS corruption on re-Put, the
	// wipeable-cache "recompute-into-place" contract.
	if existing, err := os.ReadFile(p); err == nil && HashOf(existing) == h {
		s.touch(p)
		s.evict(h)
		return h, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := s.writeAtomic(p, data); err != nil {
		return "", err
	}
	s.touch(p)
	s.evict(h)
	return h, nil
}

// writeAtomic writes data to a temp file in p's directory, then renames it into
// place so a concurrent reader never observes a partial blob.
func (s *FSStore) writeAtomic(p string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(p), tmpPrefix+"*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, p); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// touch stamps p's mtime from the injected clock — the LRU recency signal, so
// eviction order is deterministic and independent of wall-clock. Best-effort: a
// failed stamp (e.g. read-only metadata mount) must never fail an otherwise-valid
// Put or Get. The only cost is a stale recency signal, which at worst evicts a
// slightly-wrong victim — and the wipeable-cache contract recomputes either way.
func (s *FSStore) touch(p string) {
	t := s.now()
	_ = os.Chtimes(p, t, t)
}

func (s *FSStore) Get(h Hash) ([]byte, error) {
	p, ok := s.shardPath(h)
	if !ok {
		return nil, ErrNotFound
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if HashOf(data) != h {
		return nil, ErrCorrupt
	}
	s.touch(p)
	return data, nil
}

func (s *FSStore) Has(h Hash) (bool, error) {
	p, ok := s.shardPath(h)
	if !ok {
		return false, nil
	}
	switch _, err := os.Stat(p); {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}

// evict lists the pool and deletes the LRU victims selectEvictions picks, keeping
// the just-written blob (keep). A no-op when unbounded. Best-effort, like touch:
// eviction is cache maintenance, so a failed scan or delete never fails the
// successful Put that triggered it — the pool just stays temporarily over budget,
// which the next Put retries and the wipeable contract tolerates.
func (s *FSStore) evict(keep Hash) {
	if s.maxBytes <= 0 {
		return
	}
	entries, err := s.list()
	if err != nil {
		return // can't scan the pool → skip this eviction pass
	}
	for _, h := range selectEvictions(entries, s.maxBytes, keep) {
		if p, ok := s.shardPath(h); ok {
			_ = os.Remove(p)
		}
	}
}

// list walks the sharded pool and returns one entry per stored blob (size + mtime
// recency), skipping in-flight temp files.
func (s *FSStore) list() ([]entry, error) {
	shards, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var entries []entry
	for _, shard := range shards {
		if !shard.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(s.root, shard.Name()))
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			if f.IsDir() || strings.HasPrefix(f.Name(), tmpPrefix) {
				continue
			}
			info, err := f.Info()
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue // raced with a concurrent eviction; skip
				}
				return nil, err
			}
			entries = append(entries, entry{
				hash:  Hash(f.Name()),
				size:  info.Size(),
				mtime: info.ModTime(),
			})
		}
	}
	return entries, nil
}

var _ Store = (*FSStore)(nil)
