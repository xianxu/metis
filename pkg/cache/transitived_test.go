package cache

import (
	"reflect"
	"testing"

	"github.com/xianxu/metis/pkg/record"
)

func ref(repo, path, hash string) record.CodeRef {
	return record.CodeRef{Repo: repo, Path: path, BlobHash: record.Hash(hash)}
}

// MergeTransitiveD folds own-D + upstream snapshots into a deduped, canonically-sorted
// closure. The diamond case (a root reachable via two upstreams) must fold the root ONCE,
// and the output must be order-independent (stable persisted bytes).
func TestMergeTransitiveD_UnionDedupCanonicalDiamond(t *testing.T) {
	root := ref("m", "root.py", "r1")
	a := ref("m", "a.py", "a1")
	b := ref("m", "b.py", "b1")
	own := ref("k", "s.py", "s1") // different repo — sorts after "m"

	// S ← A, S ← B, both A,B ← root. Upstream snapshots each already carry root.
	tdA := MergeTransitiveD([]record.CodeRef{a}, []record.CodeRef{root})
	tdB := MergeTransitiveD([]record.CodeRef{b}, []record.CodeRef{root})
	got := MergeTransitiveD([]record.CodeRef{own}, tdA, tdB)

	// Canonical order: by (repo, path). "k" < "m", then path-sorted within "m".
	want := []record.CodeRef{
		ref("k", "s.py", "s1"),
		ref("m", "a.py", "a1"),
		ref("m", "b.py", "b1"),
		ref("m", "root.py", "r1"), // folded ONCE despite arriving via both tdA and tdB
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diamond fold wrong:\n got %v\nwant %v", got, want)
	}

	// Order-independence: swapping the upstream args yields identical bytes.
	if swapped := MergeTransitiveD([]record.CodeRef{own}, tdB, tdA); !reflect.DeepEqual(swapped, got) {
		t.Fatalf("not order-independent:\n %v\n %v", swapped, got)
	}
}

func TestMergeTransitiveD_EmptyIsEmpty(t *testing.T) {
	if got := MergeTransitiveD(nil); len(got) != 0 {
		t.Fatalf("empty own + no upstream should be empty, got %v", got)
	}
}

// The metis#24 migration guard rests entirely on a JSON-codec invariant: a genuine #24 empty
// closure ([]) MUST round-trip to a NON-NIL slice (so an empty-closure step still HITs
// vacuously), while a legacy (pre-#24) entry with no transitive_d key decodes to NIL (→ the
// isHit guard MISSes it). Dropping `omitempty` on Entry.TransitiveD is what makes [] survive as
// non-nil; re-adding it would silently break the guard AND make every empty-closure step MISS
// forever. Pinned directly here (not just implicitly via the warm-HIT e2e) so a regression
// fails THIS test loudly, independent of any e2e fixture's step choices.
func TestEntry_TransitiveDCodec_EmptyIsNonNil_LegacyIsNil(t *testing.T) {
	empty := MergeTransitiveD(nil) // an empty #24 closure — never nil
	if empty == nil {
		t.Fatal("MergeTransitiveD must return a non-nil slice (the #24 empty-closure marker)")
	}
	b, err := EncodeEntry(Entry{Kpre: "k", TransitiveD: empty, Output: "o"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeEntry(b)
	if err != nil {
		t.Fatal(err)
	}
	if got.TransitiveD == nil {
		t.Error("a #24 empty closure ([]) must round-trip NON-NIL (else it MISSes forever) — did omitempty return?")
	}

	// A legacy entry (no transitive_d key) and an explicit null both decode to nil → MISS.
	for _, raw := range []string{`{"kpre":"k","d":[],"output":"o"}`, `{"kpre":"k","transitive_d":null,"output":"o"}`} {
		e, err := DecodeEntry([]byte(raw))
		if err != nil {
			t.Fatalf("decode %s: %v", raw, err)
		}
		if e.TransitiveD != nil {
			t.Errorf("a legacy/null transitive_d must decode to nil (the migration-guard signal); raw=%s", raw)
		}
	}
}

// A step's TransitiveD snapshot must survive the on-disk index codec (isHit re-hashes it).
func TestEntry_TransitiveDRoundtrip(t *testing.T) {
	e := Entry{
		Kpre:        "k1",
		D:           []record.CodeRef{ref("m", "s.py", "s1")},
		TransitiveD: []record.CodeRef{ref("m", "root.py", "r1"), ref("m", "s.py", "s1")},
		Output:      "o1",
	}
	b, err := EncodeEntry(e)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeEntry(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, e) {
		t.Fatalf("Entry roundtrip lost TransitiveD:\n got %+v\nwant %+v", got, e)
	}
}
