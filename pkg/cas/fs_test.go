package cas

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func mustPut(t *testing.T, s Store, data []byte) Hash {
	t.Helper()
	h, err := s.Put(data)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func mustHas(t *testing.T, s Store, h Hash) bool {
	t.Helper()
	ok, err := s.Has(h)
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

func TestFSStore_ShardsByHashPrefix(t *testing.T) {
	root := t.TempDir()
	clk, _ := fakeClock(time.Unix(0, 0))
	s := NewFSStore(root, 0, clk)
	h := mustPut(t, s, []byte("shard me"))

	want := filepath.Join(root, string(h)[:2], string(h))
	if _, err := os.Stat(want); err != nil {
		t.Errorf("blob not at sharded path %s: %v", want, err)
	}
}

func TestFSStore_GetDetectsCorruption(t *testing.T) {
	root := t.TempDir()
	clk, _ := fakeClock(time.Unix(0, 0))
	s := NewFSStore(root, 0, clk)
	h := mustPut(t, s, []byte("trustworthy"))

	// Corrupt the on-disk bytes behind the store's back.
	p := filepath.Join(root, string(h)[:2], string(h))
	if err := os.WriteFile(p, []byte("tampered!!!"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := s.Get(h)
	if !errors.Is(err, ErrCorrupt) {
		t.Errorf("Get of corrupted blob = %v, want ErrCorrupt", err)
	}
}

func TestFSStore_ReputHealsCorruption(t *testing.T) {
	root := t.TempDir()
	clk, _ := fakeClock(time.Unix(0, 0))
	s := NewFSStore(root, 0, clk)
	h := mustPut(t, s, []byte("heal me"))

	// Corrupt the on-disk bytes behind the store's back.
	p := filepath.Join(root, string(h)[:2], string(h))
	if err := os.WriteFile(p, []byte("corrupted!!"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(h); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Get corrupt = %v, want ErrCorrupt", err)
	}

	// Recompute-into-place: re-Put the correct bytes must HEAL the blob (the
	// wipeable-cache contract), not dedup-skip past the corrupt file.
	if h2 := mustPut(t, s, []byte("heal me")); h2 != h {
		t.Fatalf("re-Put hash %q != %q", h2, h)
	}
	got, err := s.Get(h)
	if err != nil {
		t.Fatalf("Get after re-Put = %v, want healed bytes", err)
	}
	if string(got) != "heal me" {
		t.Errorf("healed bytes = %q, want %q", got, "heal me")
	}
}

func TestFSStore_RejectsMalformedKeys(t *testing.T) {
	root := t.TempDir()
	clk, _ := fakeClock(time.Unix(0, 0))
	s := NewFSStore(root, 0, clk)

	for _, bad := range []Hash{
		"../../etc/hosts",
		"..",
		"abc",                          // too short
		Hash(strings.Repeat("g", 64)),  // right length, non-hex
		Hash(strings.Repeat("A", 64)),  // uppercase (HashOf emits lowercase)
		"a/b/c",                        // path separators
	} {
		if ok, err := s.Has(bad); err != nil || ok {
			t.Errorf("Has(%q) = (%v, %v), want (false, nil)", bad, ok, err)
		}
		if _, err := s.Get(bad); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get(%q) = %v, want ErrNotFound", bad, err)
		}
	}
}

func TestFSStore_EvictsLeastRecentlyUsed(t *testing.T) {
	root := t.TempDir()
	clk, adv := fakeClock(time.Unix(1_700_000_000, 0))
	s := NewFSStore(root, 250, clk) // fits two 100-byte blobs, not three

	a := HashOf(mkblob('a'))
	b := HashOf(mkblob('b'))
	c := HashOf(mkblob('c'))

	mustPut(t, s, mkblob('a'))
	adv(time.Second)
	mustPut(t, s, mkblob('b'))
	adv(time.Second)
	mustPut(t, s, mkblob('c')) // 300 bytes > 250 → evict oldest (a)

	if mustHas(t, s, a) {
		t.Errorf("oldest blob a should have been evicted")
	}
	if !mustHas(t, s, b) || !mustHas(t, s, c) {
		t.Errorf("b and c should remain (b=%v c=%v)", mustHas(t, s, b), mustHas(t, s, c))
	}
}

func TestFSStore_EvictThenRefetchRestores(t *testing.T) {
	root := t.TempDir()
	clk, adv := fakeClock(time.Unix(1_700_000_000, 0))
	s := NewFSStore(root, 250, clk)

	a := mustPut(t, s, mkblob('a'))
	adv(time.Second)
	mustPut(t, s, mkblob('b'))
	adv(time.Second)
	mustPut(t, s, mkblob('c')) // evicts a

	if _, err := s.Get(a); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get evicted a = %v, want ErrNotFound", err)
	}
	// A wipeable-cache consumer recomputes and re-Puts; the blob comes back.
	adv(time.Second)
	if a2 := mustPut(t, s, mkblob('a')); a2 != a {
		t.Fatalf("re-Put hash %q != original %q", a2, a)
	}
	if !mustHas(t, s, a) {
		t.Errorf("re-Put should restore a")
	}
}

func TestFSStore_GetRefreshesRecency(t *testing.T) {
	root := t.TempDir()
	clk, adv := fakeClock(time.Unix(1_700_000_000, 0))
	s := NewFSStore(root, 250, clk)

	a := mustPut(t, s, mkblob('a'))
	adv(time.Second)
	b := mustPut(t, s, mkblob('b')) // a,b both fit (200 <= 250)

	// Touch a — it becomes most-recently-used, so b is now the LRU victim.
	adv(time.Second)
	if _, err := s.Get(a); err != nil {
		t.Fatal(err)
	}
	adv(time.Second)
	mustPut(t, s, mkblob('c')) // 300 > 250 → evict LRU, which is now b (not a)

	if !mustHas(t, s, a) {
		t.Errorf("a was accessed most recently and must survive eviction")
	}
	if mustHas(t, s, b) {
		t.Errorf("b was least-recently-used and should be evicted")
	}
}

// mkblob returns a distinct 100-byte blob filled with c.
func mkblob(c byte) []byte {
	b := make([]byte, 100)
	for i := range b {
		b[i] = c
	}
	return b
}
