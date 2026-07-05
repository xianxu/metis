package cas

import "sync"

// MemStore is an in-memory Store — the fake consumers (metis#2) test against and
// the reference semantics for the interface. Unbounded (no eviction); integrity
// holds trivially since in-memory bytes cannot corrupt. Safe for concurrent use.
// Copies on Put and Get so callers cannot mutate stored bytes through an alias.
type MemStore struct {
	mu    sync.Mutex
	blobs map[Hash][]byte
}

// NewMemStore returns an empty in-memory store.
func NewMemStore() *MemStore {
	return &MemStore{blobs: map[Hash][]byte{}}
}

func (m *MemStore) Put(data []byte) (Hash, error) {
	h := HashOf(data)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.blobs[h]; !ok {
		cp := make([]byte, len(data))
		copy(cp, data)
		m.blobs[h] = cp
	}
	return h, nil
}

func (m *MemStore) Get(h Hash) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.blobs[h]
	if !ok {
		return nil, ErrNotFound
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp, nil
}

func (m *MemStore) Has(h Hash) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.blobs[h]
	return ok, nil
}

var _ Store = (*MemStore)(nil)
