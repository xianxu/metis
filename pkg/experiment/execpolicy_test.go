package experiment

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func withExec(id string, policy any) Step {
	return Step{ID: id, Uses: "test/echo", With: map[string]any{"exec": policy}}
}

// execPolicyCases is shared by the Go table and the CUE drift guard (ARCH-DRY). cueOK differs
// from goErr == "" where CUE's shape (timeout?: string) cannot see the value: an unparseable or
// non-positive duration is a string to CUE and only Go Validate refuses it.
var execPolicyCases = []struct {
	name   string
	policy any
	want   ExecPolicy
	goErr  string // substring; "" = accepted
	cueOK  bool
}{
	{"timeout", map[string]any{"timeout": "90m"}, ExecPolicy{Timeout: 90 * time.Minute}, "", true},
	{"compound timeout", map[string]any{"timeout": "1h30m"}, ExecPolicy{Timeout: 90 * time.Minute}, "", true},
	{"empty map", map[string]any{}, ExecPolicy{}, "", true},
	{"not a map", "90m", ExecPolicy{}, "must be a map", false},
	{"unknown key", map[string]any{"grace": "10s"}, ExecPolicy{}, `unknown key "grace"`, false},
	{"non-string timeout", map[string]any{"timeout": 90}, ExecPolicy{}, "with.exec.timeout: want a duration string", false},
	{"unparseable timeout", map[string]any{"timeout": "ninety"}, ExecPolicy{}, "with.exec.timeout: invalid duration", true},
	{"zero timeout", map[string]any{"timeout": "0s"}, ExecPolicy{}, "must be positive", true},
	{"negative timeout", map[string]any{"timeout": "-5m"}, ExecPolicy{}, "must be positive", true},
}

func TestParseExecPolicy(t *testing.T) {
	if p, err := ParseExecPolicy(Step{ID: "plain", With: map[string]any{"timeout": "5m"}}); err != nil || p != (ExecPolicy{}) {
		t.Errorf("absent with.exec (a step type's own with.timeout): got %+v, %v; want zero policy", p, err)
	}
	for _, c := range execPolicyCases {
		t.Run(c.name, func(t *testing.T) {
			p, err := ParseExecPolicy(withExec("s", c.policy))
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

func TestValidate_RefusesMalformedExecPolicy(t *testing.T) {
	exp, err := Parse("---\ntype: experiment\nid: x\nseed: 1\nstatus: active\nsteps:\n" +
		"  - id: session\n    uses: bench/session\n    with: {exec: {timeout: 0s}}\n---\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(exp); err == nil || !strings.Contains(err.Error(), "with.exec.timeout: must be positive") {
		t.Fatalf("Validate must refuse timeout: 0s, got %v", err)
	}
}

// TestExecPolicyConformsToCUE is the drift guard between ParseExecPolicy and the CUE
// #ExecPolicy, mirroring TestCachePolicyConformsToCUE.
func TestExecPolicyConformsToCUE(t *testing.T) {
	if _, err := exec.LookPath("cue"); err != nil {
		t.Skip("cue not on PATH; skipping #ExecPolicy drift guard")
	}
	cueFile := filepath.Join(repoRoot(t), "construct", "vocabulary", "experiment.cue")
	for _, c := range execPolicyCases {
		t.Run(c.name, func(t *testing.T) {
			b, err := json.Marshal(map[string]any{"id": "s", "uses": "t/x", "with": map[string]any{"exec": c.policy, "k": 1}})
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
