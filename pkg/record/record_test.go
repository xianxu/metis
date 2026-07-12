package record

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

// CanonicalHash is the shared hashing primitive: deterministic over map order,
// content-sensitive, and errors (not panics) on a non-finite value.
func TestCanonicalHash_DeterministicAndSensitive(t *testing.T) {
	a, err := CanonicalHash(map[string]any{"x": 1, "y": 2})
	if err != nil {
		t.Fatal(err)
	}
	b, err := CanonicalHash(map[string]any{"y": 2, "x": 1}) // same content, different literal order
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Errorf("CanonicalHash must be map-order-independent: %q != %q", a, b)
	}
	c, _ := CanonicalHash(map[string]any{"x": 1, "y": 3})
	if a == c {
		t.Error("CanonicalHash must change when content changes")
	}
	if len(a) != 64 {
		t.Errorf("CanonicalHash should be a 64-hex hash, got %d chars", len(a))
	}
	if _, err := CanonicalHash(map[string]any{"lr": math.Inf(1)}); err == nil {
		t.Error("CanonicalHash(+Inf) must return an error, not panic")
	}
}

// mustAddr mints a point-address, failing the test on the (well-formed-input) error.
func mustAddr(t *testing.T, rw map[string]map[string]any, shapeBlobHash string, seed int) Hash {
	t.Helper()
	h, err := PointAddress(rw, shapeBlobHash, seed)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// PointAddress must be deterministic across calls — a guard against map-iteration
// order leaking into the address (it wouldn't if canonicalized, would if not).
func TestPointAddress_DeterministicAcrossCalls(t *testing.T) {
	rw := map[string]map[string]any{
		"prep":  {"k": 5, "shuffle": true},
		"train": {"model": "logreg", "c": 1.0},
	}
	shapeBlobHash := "abc123def456"
	first := mustAddr(t, rw, shapeBlobHash, 42)
	for i := 0; i < 25; i++ {
		if got := mustAddr(t, rw, shapeBlobHash, 42); got != first {
			t.Fatalf("PointAddress not deterministic on call %d: %q != %q", i, got, first)
		}
	}
	if len(first) != 64 {
		t.Errorf("point-address should be a 64-hex hash, got %d chars", len(first))
	}
}

// The address changes iff a determinant changes (resolved-with, shape-blob, seed).
func TestPointAddress_Sensitivity(t *testing.T) {
	base := mustAddr(t, map[string]map[string]any{"s": {"k": 5}}, "blob1", 42)
	cases := map[string]Hash{
		"changed resolved-with": mustAddr(t, map[string]map[string]any{"s": {"k": 6}}, "blob1", 42),
		"changed shape-blob":    mustAddr(t, map[string]map[string]any{"s": {"k": 5}}, "blob2", 42),
		"changed seed":          mustAddr(t, map[string]map[string]any{"s": {"k": 5}}, "blob1", 43),
	}
	for name, addr := range cases {
		if addr == base {
			t.Errorf("%s must change the point-address, but it matched base", name)
		}
	}
	// An identical determinant set reproduces the address.
	if again := mustAddr(t, map[string]map[string]any{"s": {"k": 5}}, "blob1", 42); again != base {
		t.Errorf("identical determinants must reproduce the address: %q != %q", again, base)
	}
}

// A non-finite config value (.inf/.nan is valid YAML → float64 Inf/NaN, which
// json.Marshal rejects) must surface as an error, NOT a panic — it's user-reachable
// input, so the derivation returns it for the caller to surface as a run error.
func TestPointAddress_ErrorsOnNonFiniteConfig(t *testing.T) {
	if _, err := PointAddress(map[string]map[string]any{"s": {"lr": math.Inf(1)}}, "", 0); err == nil {
		t.Error("PointAddress(+Inf) must return an error, not panic or succeed")
	}
	if _, err := PointAddress(map[string]map[string]any{"s": {"lr": math.NaN()}}, "", 0); err == nil {
		t.Error("PointAddress(NaN) must return an error, not panic or succeed")
	}
}

// OutputHash reduces a *set* of artifact files to one address — independent of the
// order they're listed in.
func TestOutputHash_OrderIndependent(t *testing.T) {
	files := []FileHash{
		{Path: "a.parquet", Hash: "h1"},
		{Path: "b/c.json", Hash: "h2"},
		{Path: "d.pkl", Hash: "h3"},
	}
	reversed := []FileHash{files[2], files[0], files[1]}
	if OutputHash(files) != OutputHash(reversed) {
		t.Error("OutputHash must be independent of file listing order")
	}
}

// OutputHash changes iff a path or its content-hash changes.
func TestOutputHash_Sensitivity(t *testing.T) {
	base := OutputHash([]FileHash{{Path: "a", Hash: "h1"}})
	if OutputHash([]FileHash{{Path: "a", Hash: "h2"}}) == base {
		t.Error("a changed content-hash must change OutputHash")
	}
	if OutputHash([]FileHash{{Path: "b", Hash: "h1"}}) == base {
		t.Error("a changed path must change OutputHash")
	}
	if OutputHash([]FileHash{{Path: "a", Hash: "h1"}, {Path: "b", Hash: "h2"}}) == base {
		t.Error("an added file must change OutputHash")
	}
}

// The empty output set has a defined, stable hash (no panic, nil == empty).
func TestOutputHash_EmptyIsDefined(t *testing.T) {
	if OutputHash(nil) != OutputHash([]FileHash{}) {
		t.Error("nil and empty output sets must hash equally")
	}
	if len(OutputHash(nil)) != 64 {
		t.Error("empty output set must still yield a well-formed hash")
	}
}

func TestRunRecord_JSONRoundTrip(t *testing.T) {
	rec := RunRecord{
		RunID: "run-001", Experiment: "exp1", Seed: 7,
		PointAddress: "abc", RepoSHAs: map[string]string{"metis": "sha"}, Dirty: false,
		Steps: []StepRecord{{
			StepID: "prep", Uses: "metis/cv-split",
			With:       map[string]any{"k": float64(5)}, // JSON numbers decode to float64
			Upstream:   []Hash{"u1"},
			Code:       CodeManifest{Commit: "c1", Dirty: false},
			OutputHash: "oh", Metrics: map[string]float64{"acc": 0.9},
		}},
		Started: "t0", Finished: "t1", Status: "ok",
	}
	b, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	var got RunRecord
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rec, got) {
		t.Errorf("JSON round-trip mismatch:\nwant %+v\ngot  %+v", rec, got)
	}
}
