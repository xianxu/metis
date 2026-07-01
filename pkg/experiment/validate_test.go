package experiment

import (
	"strings"
	"testing"
)

// TestValidate table-drives the semantic checks CUE cannot express. The two
// fixtures (invalid-cycle, invalid-dangling-needs) are parsed from disk to prove
// the shape-valid-but-semantics-invalid split; the remaining cases are inline
// structs so the specific violation is legible at the assertion site.
func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		exp     Experiment
		wantErr string // substring the joined error must contain; "" = must pass
	}{
		{
			name: "valid two-step pipeline",
			exp: Experiment{Steps: []Step{
				{ID: "prep", Uses: "metis/cv-split"},
				{ID: "train", Uses: "metis/train", Needs: []string{"prep"}},
			}},
			wantErr: "",
		},
		{
			name:    "cycle (from fixture)",
			exp:     mustParse(t, "invalid-cycle.md"),
			wantErr: "cycle",
		},
		{
			name:    "dangling needs (from fixture)",
			exp:     mustParse(t, "invalid-dangling-needs.md"),
			wantErr: "needs",
		},
		{
			name: "bad uses format (no slash)",
			exp: Experiment{Steps: []Step{
				{ID: "a", Uses: "metis-cv-split"},
			}},
			wantErr: "uses",
		},
		{
			name: "bad uses format (uppercase)",
			exp: Experiment{Steps: []Step{
				{ID: "a", Uses: "metis/CV-Split"},
			}},
			wantErr: "uses",
		},
		{
			name: "duplicate step id",
			exp: Experiment{Steps: []Step{
				{ID: "a", Uses: "metis/one"},
				{ID: "a", Uses: "metis/two"},
			}},
			wantErr: "duplicate",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.exp)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate: want pass, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate: want error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// TestValidate_ReportsAllViolations asserts Validate joins violations rather than
// returning only the first — a single call surfaces every problem (errors.Join).
func TestValidate_ReportsAllViolations(t *testing.T) {
	exp := Experiment{Steps: []Step{
		{ID: "a", Uses: "BAD"},                               // bad uses
		{ID: "a", Uses: "metis/x"},                           // duplicate id
		{ID: "b", Uses: "metis/y", Needs: []string{"ghost"}}, // dangling needs
	}}
	err := Validate(exp)
	if err == nil {
		t.Fatal("want joined error, got nil")
	}
	for _, want := range []string{"uses", "duplicate", "needs"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("joined error missing %q:\n%s", want, err.Error())
		}
	}
}

// TestTopoSort asserts dependency-order output and stable ordering, and that a
// cycle is reported (not silently truncated).
func TestTopoSort(t *testing.T) {
	exp := Experiment{Steps: []Step{
		{ID: "train", Uses: "metis/train", Needs: []string{"prep"}},
		{ID: "prep", Uses: "metis/cv-split"},
		{ID: "eval", Uses: "metis/eval", Needs: []string{"train"}},
	}}
	order, err := TopoSort(exp)
	if err != nil {
		t.Fatalf("TopoSort: %v", err)
	}
	got := make([]string, len(order))
	for i, s := range order {
		got[i] = s.ID
	}
	pos := map[string]int{}
	for i, id := range got {
		pos[id] = i
	}
	if pos["prep"] > pos["train"] || pos["train"] > pos["eval"] {
		t.Fatalf("TopoSort order violates deps: %v", got)
	}

	cyc := mustParse(t, "invalid-cycle.md")
	if _, err := TopoSort(cyc); err == nil {
		t.Fatal("TopoSort accepted a cyclic experiment; want error")
	}
}

func mustParse(t *testing.T, fixture string) Experiment {
	t.Helper()
	exp, err := Parse(readFixture(t, fixture))
	if err != nil {
		t.Fatalf("parse %s: %v", fixture, err)
	}
	return exp
}
