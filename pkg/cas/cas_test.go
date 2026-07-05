package cas

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

// fakeClock returns a Clock reading a mutable current time plus an advance func,
// so eviction recency is exercised deterministically (no wall-clock).
func fakeClock(base time.Time) (Clock, func(time.Duration)) {
	cur := base
	return func() time.Time { return cur }, func(d time.Duration) { cur = cur.Add(d) }
}

func TestHashOf_DeterministicAndContentAddressed(t *testing.T) {
	a := HashOf([]byte("hello"))
	b := HashOf([]byte("hello"))
	c := HashOf([]byte("world"))
	if a != b {
		t.Errorf("same content must hash equal: %q != %q", a, b)
	}
	if a == c {
		t.Errorf("distinct content must hash differently: both %q", a)
	}
	if len(a) != 64 { // hex sha256
		t.Errorf("expected 64-char hex sha256, got %d chars: %q", len(a), a)
	}
}

// storeFactory builds a fresh, unbounded Store for the shared contract.
type storeFactory func(t *testing.T) Store

// runStoreContract exercises the Store interface's semantics that every backend
// must satisfy, so MemStore (the fake) and FSStore share one source of truth.
func runStoreContract(t *testing.T, newStore storeFactory) {
	t.Run("put returns the content hash", func(t *testing.T) {
		s := newStore(t)
		h, err := s.Put([]byte("payload"))
		if err != nil {
			t.Fatal(err)
		}
		if want := HashOf([]byte("payload")); h != want {
			t.Errorf("Put hash = %q, want %q", h, want)
		}
	})

	t.Run("get round-trips the bytes", func(t *testing.T) {
		s := newStore(t)
		data := []byte("round trip me")
		h, err := s.Put(data)
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.Get(h)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, data) {
			t.Errorf("Get = %q, want %q", got, data)
		}
	})

	t.Run("has reflects membership", func(t *testing.T) {
		s := newStore(t)
		h := HashOf([]byte("member"))
		if ok, err := s.Has(h); err != nil || ok {
			t.Fatalf("Has before Put = (%v, %v), want (false, nil)", ok, err)
		}
		if _, err := s.Put([]byte("member")); err != nil {
			t.Fatal(err)
		}
		if ok, err := s.Has(h); err != nil || !ok {
			t.Fatalf("Has after Put = (%v, %v), want (true, nil)", ok, err)
		}
	})

	t.Run("get of an absent hash returns ErrNotFound", func(t *testing.T) {
		s := newStore(t)
		_, err := s.Get(HashOf([]byte("never stored")))
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Get absent = %v, want ErrNotFound", err)
		}
	})

	t.Run("identical content dedupes to one hash", func(t *testing.T) {
		s := newStore(t)
		h1, err := s.Put([]byte("dup"))
		if err != nil {
			t.Fatal(err)
		}
		h2, err := s.Put([]byte("dup"))
		if err != nil {
			t.Fatal(err)
		}
		if h1 != h2 {
			t.Errorf("re-Put of identical bytes gave %q then %q", h1, h2)
		}
	})
}

func TestMemStore_Contract(t *testing.T) {
	runStoreContract(t, func(t *testing.T) Store { return NewMemStore() })
}

func TestFSStore_Contract(t *testing.T) {
	runStoreContract(t, func(t *testing.T) Store {
		clk, _ := fakeClock(time.Unix(1_700_000_000, 0))
		return NewFSStore(t.TempDir(), 0, clk) // unbounded: no eviction during contract
	})
}

// --- selectEvictions: the pure eviction-victim math (no filesystem) ---

func TestSelectEvictions_UnderBudgetEvictsNothing(t *testing.T) {
	base := time.Unix(0, 0)
	entries := []entry{
		{hash: "a", size: 100, atime: base},
		{hash: "b", size: 100, atime: base.Add(time.Second)},
	}
	if got := selectEvictions(entries, 250, "b"); len(got) != 0 {
		t.Errorf("under budget should evict nothing, got %v", got)
	}
}

func TestSelectEvictions_UnboundedEvictsNothing(t *testing.T) {
	entries := []entry{{hash: "a", size: 1 << 30, atime: time.Unix(0, 0)}}
	if got := selectEvictions(entries, 0, "a"); got != nil {
		t.Errorf("maxBytes<=0 is unbounded, got %v", got)
	}
}

func TestSelectEvictions_EvictsOldestFirstUntilUnderBudget(t *testing.T) {
	base := time.Unix(0, 0)
	entries := []entry{
		{hash: "new", size: 100, atime: base.Add(2 * time.Second)},
		{hash: "old", size: 100, atime: base},
		{hash: "mid", size: 100, atime: base.Add(time.Second)},
	}
	got := selectEvictions(entries, 150, "new") // 300 total, keep 'new', trim to <=150
	// Must drop the two oldest (old, then mid) — 'new' is protected.
	want := []Hash{"old", "mid"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("evicted %v, want %v (oldest-first, keep protected)", got, want)
	}
}

func TestSelectEvictions_NeverEvictsProtectedEntry(t *testing.T) {
	base := time.Unix(0, 0)
	// One oversized blob that alone exceeds the budget; it is the just-written 'keep'.
	entries := []entry{
		{hash: "keep", size: 500, atime: base.Add(time.Second)},
		{hash: "other", size: 100, atime: base},
	}
	got := selectEvictions(entries, 200, "keep")
	for _, h := range got {
		if h == "keep" {
			t.Fatalf("protected just-written entry was evicted: %v", got)
		}
	}
	if len(got) != 1 || got[0] != "other" {
		t.Errorf("evicted %v, want [other] (keep protected even when oversized)", got)
	}
}
