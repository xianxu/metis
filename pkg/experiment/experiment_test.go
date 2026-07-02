package experiment

import (
	"reflect"
	"testing"
)

// TestParse_ValidBaseline parses the generic 2-step fixture and asserts the
// exact struct — the pipeline shape (ids, uses, needs, with) round-trips out of
// the YAML frontmatter. Pure: no IO beyond reading the fixture file.
func TestParse_ValidBaseline(t *testing.T) {
	got, err := Parse(readFixture(t, "valid-baseline.md"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := Experiment{
		Type:   "experiment",
		ID:     "valid-baseline",
		Seed:   42,
		Status: "active",
		Steps: []Step{
			{ID: "prep", Uses: "metis/cv-split", With: map[string]any{"k": 5}},
			{ID: "train", Uses: "metis/train", Needs: []string{"prep"}, With: map[string]any{"model": "logreg"}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse mismatch:\n got=%#v\nwant=%#v", got, want)
	}
}

// TestParse_NoFrontmatter surfaces the frontmatter.Split error unchanged rather
// than returning a zero Experiment silently.
func TestParse_NoFrontmatter(t *testing.T) {
	if _, err := Parse("# just a heading, no fence\n"); err == nil {
		t.Fatal("Parse accepted content with no frontmatter fence; want error")
	}
}
