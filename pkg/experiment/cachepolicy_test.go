package experiment

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func withCache(id string, cache any) Step {
	return Step{ID: id, Uses: "test/echo", With: map[string]any{"cache": cache}}
}

// cachePolicyCases is shared by the Go table and the CUE drift guard below, so the two
// validators are checked against ONE enumeration (ARCH-DRY). cueOK differs from goErr == ""
// only on the contradiction, which CUE's shape cannot express.
var cachePolicyCases = []struct {
	name  string
	cache any
	want  CachePolicy
	goErr string // substring; "" = accepted
	cueOK bool
}{
	{"leaf", map[string]any{"leaf": "immutable"}, CachePolicy{ImmutableLeaf: true}, "", true},
	{"never", map[string]any{"policy": "never"}, CachePolicy{Never: true}, "", true},
	{"empty map", map[string]any{}, CachePolicy{}, "", true},
	{"not a map", "never", CachePolicy{}, "must be a map", false},
	{"unknown key", map[string]any{"ttl": "1h"}, CachePolicy{}, `unknown key "ttl"`, false},
	{"unknown policy value", map[string]any{"policy": "always"}, CachePolicy{}, "with.cache.policy: unknown value", false},
	{"malformed leaf", map[string]any{"leaf": "frozen"}, CachePolicy{}, "with.cache.leaf: unknown value", false},
	{"non-string value", map[string]any{"policy": true}, CachePolicy{}, "unknown value", false},
	{"leaf and never", map[string]any{"leaf": "immutable", "policy": "never"}, CachePolicy{}, "contradict", true},
}

func TestParseCachePolicy(t *testing.T) {
	if p, err := ParseCachePolicy(Step{ID: "plain", With: map[string]any{"k": 1}}); err != nil || p != (CachePolicy{}) {
		t.Errorf("absent with.cache: got %+v, %v; want zero policy", p, err)
	}
	for _, c := range cachePolicyCases {
		t.Run(c.name, func(t *testing.T) {
			p, err := ParseCachePolicy(withCache("s", c.cache))
			if c.goErr == "" {
				if err != nil || p != c.want {
					t.Fatalf("got %+v, %v; want %+v", p, err, c.want)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.goErr) || !strings.Contains(err.Error(), `step "s"`) {
				t.Fatalf("got %v; want an error naming step \"s\" containing %q", err, c.goErr)
			}
		})
	}
}

// Validate (run by Runner.Run and ValidateShape before any step executes) refuses a
// malformed policy, from the real YAML parse path.
func TestValidate_RefusesMalformedCachePolicy(t *testing.T) {
	exp, err := Parse("---\ntype: experiment\nid: x\nseed: 1\nstatus: active\nsteps:\n" +
		"  - id: session\n    uses: bench/session\n    with: {cache: {policy: nevr}}\n---\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(exp); err == nil || !strings.Contains(err.Error(), "with.cache.policy: unknown value nevr") {
		t.Fatalf("Validate must refuse policy: nevr, got %v", err)
	}
}

func TestUncachedSteps(t *testing.T) {
	never := map[string]any{"policy": "never"}
	s := func(id string, needs ...string) Step { return Step{ID: id, Uses: "t/x", Needs: needs} }
	n := func(id string, needs ...string) Step { st := withCache(id, never); st.Needs = needs; return st }
	cases := []struct {
		name  string
		steps []Step
		want  map[string]bool
	}{
		{"no policy", []Step{s("a"), s("b", "a")}, map[string]bool{}},
		{"chain below never", []Step{s("root"), n("session", "root"), s("score", "session"), s("report", "score")},
			map[string]bool{"session": true, "score": true, "report": true}},
		{"diamond, one never arm", []Step{s("root"), n("l", "root"), s("r", "root"), s("join", "l", "r")},
			map[string]bool{"l": true, "join": true}},
		{"declared out of topo order", []Step{s("score", "session"), n("session")},
			map[string]bool{"session": true, "score": true}},
		{"malformed policy fails closed", []Step{withCache("bad", "never"), s("down", "bad")},
			map[string]bool{"bad": true, "down": true}},
		{"cycle terminates", []Step{s("a", "b"), s("b", "a")}, map[string]bool{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := UncachedSteps(Experiment{Steps: c.steps}); !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// TestCachePolicyConformsToCUE is the drift guard between ParseCachePolicy and the CUE
// #CachePolicy (construct/vocabulary/experiment.cue): each shared case is `cue vet`ted as a
// #Step. SKIPS when `cue` is unavailable, mirroring TestRunConformsToCUE.
func TestCachePolicyConformsToCUE(t *testing.T) {
	if _, err := exec.LookPath("cue"); err != nil {
		t.Skip("cue not on PATH; skipping #CachePolicy drift guard")
	}
	cueFile := filepath.Join(repoRoot(t), "construct", "vocabulary", "experiment.cue")
	for _, c := range cachePolicyCases {
		t.Run(c.name, func(t *testing.T) {
			b, err := json.Marshal(map[string]any{"id": "s", "uses": "t/x", "with": map[string]any{"cache": c.cache, "k": 1}})
			if err != nil {
				t.Fatal(err)
			}
			tmp := filepath.Join(t.TempDir(), "step.json")
			if err := os.WriteFile(tmp, b, 0o644); err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command("cue", "vet", "-d", "#Step", tmp, cueFile).CombinedOutput()
			if got := err == nil; got != c.cueOK {
				t.Errorf("cue vet accepted=%v, want %v\n%s", got, c.cueOK, out)
			}
		})
	}
}
