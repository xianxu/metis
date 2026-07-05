package cache

import (
	"fmt"
	"math"
	"testing"

	"github.com/xianxu/metis/pkg/record"
)

func sampleStep() record.StepRecord {
	return record.StepRecord{
		StepID:   "train",
		Uses:     "metis/train",
		With:     map[string]any{"model": "logreg"},
		Upstream: []record.Hash{"u1", "u2"},
	}
}

// K_pre must be deterministic and change when ANY of its five determinants changes —
// each per-term case pins a distinct false-HIT vector (esp. `uses`).
func TestKpre_FiveTermSensitivity(t *testing.T) {
	base, err := Kpre(sampleStep(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := Kpre(sampleStep(), 7); again != base {
		t.Fatalf("K_pre not deterministic: %q vs %q", again, base)
	}
	if len(base) != 64 {
		t.Errorf("K_pre should be a 64-hex hash, got %d chars", len(base))
	}

	mut := func(f func(s *record.StepRecord, seed *int)) record.Hash {
		s, seed := sampleStep(), 7
		f(&s, &seed)
		h, _ := Kpre(s, seed)
		return h
	}
	cases := map[string]record.Hash{
		"step-id":  mut(func(s *record.StepRecord, _ *int) { s.StepID = "other" }),
		"uses":     mut(func(s *record.StepRecord, _ *int) { s.Uses = "metis/other" }),
		"with":     mut(func(s *record.StepRecord, _ *int) { s.With = map[string]any{"model": "rf"} }),
		"seed":     mut(func(_ *record.StepRecord, seed *int) { *seed = 8 }),
		"upstream": mut(func(s *record.StepRecord, _ *int) { s.Upstream = []record.Hash{"u9"} }),
	}
	for term, h := range cases {
		if h == base {
			t.Errorf("K_pre must change when %s changes — it did not (false-HIT vector)", term)
		}
	}
}

// K_pre is invariant to the declaration order of `needs`/upstream hashes.
func TestKpre_InvariantToUpstreamOrder(t *testing.T) {
	a, b := sampleStep(), sampleStep()
	a.Upstream = []record.Hash{"u1", "u2"}
	b.Upstream = []record.Hash{"u2", "u1"}
	ha, _ := Kpre(a, 7)
	hb, _ := Kpre(b, 7)
	if ha != hb {
		t.Errorf("K_pre must be invariant to upstream order: %q != %q", ha, hb)
	}
}

func TestKpre_ErrorsOnNonFiniteWith(t *testing.T) {
	s := sampleStep()
	s.With = map[string]any{"lr": math.Inf(1)}
	if _, err := Kpre(s, 0); err == nil {
		t.Error("K_pre must return an error on a non-finite `with` value")
	}
}

// Validate re-hashes D: all-match → HIT; a changed or vanished file → MISS; empty D
// is a vacuous HIT (K_pre alone determines the output).
func TestValidate_HitAndMiss(t *testing.T) {
	d := []record.CodeRef{{Path: "a.py", BlobHash: "h1"}, {Path: "b.py", BlobHash: "h2"}}
	clean := func(p string) (record.Hash, error) {
		return map[string]record.Hash{"a.py": "h1", "b.py": "h2"}[p], nil
	}
	if !Validate(d, clean) {
		t.Error("all-match D must HIT")
	}
	changed := func(p string) (record.Hash, error) {
		if p == "b.py" {
			return "CHANGED", nil
		}
		return "h1", nil
	}
	if Validate(d, changed) {
		t.Error("a changed file must MISS")
	}
	missing := func(string) (record.Hash, error) { return "", fmt.Errorf("no such file") }
	if Validate(d, missing) {
		t.Error("a vanished file must MISS")
	}
	if !Validate(nil, missing) {
		t.Error("empty D is a vacuous HIT (no code files to invalidate)")
	}
}

// OutputKey composes hash(K_pre, hash(D)) — same K_pre + different D differ; same D +
// different K_pre differ; and it's invariant to D order.
func TestOutputKey_Composition(t *testing.T) {
	d1 := []record.CodeRef{{Path: "a", BlobHash: "h1"}}
	d2 := []record.CodeRef{{Path: "a", BlobHash: "h2"}} // code changed
	k1, k2 := record.Hash("kpreA"), record.Hash("kpreB")

	base, err := OutputKey(k1, d1)
	if err != nil {
		t.Fatal(err)
	}
	if same, _ := OutputKey(k1, d2); same == base {
		t.Error("different D (code edit) must change the output key")
	}
	if same, _ := OutputKey(k2, d1); same == base {
		t.Error("different K_pre must change the output key")
	}
	// invariant to D order
	dA := []record.CodeRef{{Path: "a", BlobHash: "h1"}, {Path: "b", BlobHash: "h2"}}
	dB := []record.CodeRef{{Path: "b", BlobHash: "h2"}, {Path: "a", BlobHash: "h1"}}
	oa, _ := OutputKey(k1, dA)
	ob, _ := OutputKey(k1, dB)
	if oa != ob {
		t.Error("output key must be invariant to D listing order")
	}
}

func TestEntry_JSONRoundTrip(t *testing.T) {
	e := Entry{
		Kpre:      "kpre1",
		D:         []record.CodeRef{{Path: "metis/io.py", BlobHash: "b1"}},
		OutputKey: "out1",
	}
	b, err := EncodeEntry(e)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeEntry(b)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kpre != e.Kpre || got.OutputKey != e.OutputKey || len(got.D) != 1 || got.D[0].Path != "metis/io.py" {
		t.Errorf("Entry round-trip mismatch: %+v", got)
	}
}
