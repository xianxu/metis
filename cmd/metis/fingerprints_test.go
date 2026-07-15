package main

import (
	"strings"
	"testing"

	"github.com/xianxu/metis/pkg/ledger"
	"github.com/xianxu/metis/pkg/record"
)

func fpRow(fp, addr, level string) ledger.Row {
	return ledger.Row{CodeFingerprint: fp, PointAddr: addr, Level: level,
		FreeParams: map[string]any{"train.model": "rf"}, Status: "ok"}
}

// cohortSummaries groups by full fingerprint (legacy "" included), counts rows by level,
// folds record metadata (first/last timestamps, latest commit/dirty/status, distinct extra
// commits, matched-record count), and orders newest LAST (unknowns first).
func TestCohortSummaries(t *testing.T) {
	var led ledger.Ledger
	led.Append(
		fpRow("aaaa1111aaaa", "r1", "inner"), fpRow("aaaa1111aaaa", "r2", "outer"),
		fpRow("bbbb2222bbbb", "r3", "inner"),
		fpRow("", "r0", ""), // legacy blank
	)
	recs := map[string]record.RunRecord{
		"r1": {Started: "2026-07-14T10:00:00Z", Finished: "2026-07-14T10:05:00Z", Dirty: true,
			Steps: []record.StepRecord{{Code: record.CodeManifest{Commit: "c1", CaptureStatus: "captured"}}}},
		"r2": {Started: "2026-07-14T11:00:00Z", Finished: "2026-07-14T11:05:00Z", Dirty: true,
			Steps: []record.StepRecord{{Code: record.CodeManifest{Commit: "c2", CaptureStatus: "captured"}}}},
		"r3": {Started: "2026-07-15T09:00:00Z", Finished: "2026-07-15T09:01:00Z",
			Steps: []record.StepRecord{{Code: record.CodeManifest{Commit: "c3", CaptureStatus: "captured"}}}},
		// r0: no record (cleaned run dir) → unknowns
	}
	cs := cohortSummaries(led, recs)
	if len(cs) != 3 {
		t.Fatalf("want 3 cohorts, got %d: %+v", len(cs), cs)
	}
	// Ordering: legacy (no records → unknown) first, then aaaa (last finish 11:05 on the 14th),
	// then bbbb (09:01 on the 15th) — newest last.
	if cs[0].Fingerprint != "" || cs[1].Fingerprint != "aaaa1111aaaa" || cs[2].Fingerprint != "bbbb2222bbbb" {
		t.Fatalf("ordering (newest last, unknown first): %+v", cs)
	}
	a := cs[1]
	if a.Rows != 2 || a.ByLevel["inner"] != 1 || a.ByLevel["outer"] != 1 {
		t.Errorf("aaaa row counts: %+v", a)
	}
	if a.FirstStart != "2026-07-14T10:00:00Z" || a.LastFinish != "2026-07-14T11:05:00Z" {
		t.Errorf("aaaa timestamps: %+v", a)
	}
	if a.Commit != "c2" || a.ExtraCommits != 1 || !a.Dirty || a.CaptureStatus != "captured" {
		t.Errorf("aaaa latest-record fold: %+v", a)
	}
	if a.Records != 2 {
		t.Errorf("aaaa matched records: %+v", a)
	}
	if leg := cs[0]; leg.Rows != 1 || leg.Records != 0 || leg.LastFinish != "" {
		t.Errorf("legacy cohort: %+v", leg)
	}
}

// ExtraCommits is DISTINCT-commit cardinality minus one, order-proof: rows arrive
// non-monotone in Finished (ledger order is content-keyed, not temporal), and repeated
// rows of the same second commit must not inflate the count (plan-review finding).
func TestCohortSummaries_ExtraCommitsSetCardinality(t *testing.T) {
	var led ledger.Ledger
	led.Append(fpRow("ffff0000ffff", "r1", "inner"), fpRow("ffff0000ffff", "r2", "inner"),
		fpRow("ffff0000ffff", "r3", "inner"), fpRow("ffff0000ffff", "r4", "inner"))
	step := func(commit string) []record.StepRecord {
		return []record.StepRecord{{Code: record.CodeManifest{Commit: commit, CaptureStatus: "captured"}}}
	}
	recs := map[string]record.RunRecord{
		// Interleaved times: c2 latest, but rows of c1 and c2 alternate out of order.
		"r1": {Started: "2026-07-14T12:00:00Z", Finished: "2026-07-14T12:05:00Z", Steps: step("c2")},
		"r2": {Started: "2026-07-14T10:00:00Z", Finished: "2026-07-14T10:05:00Z", Steps: step("c1")},
		"r3": {Started: "2026-07-14T11:00:00Z", Finished: "2026-07-14T11:05:00Z", Steps: step("c2")},
		"r4": {Started: "2026-07-14T09:00:00Z", Finished: "2026-07-14T09:05:00Z", Steps: step("c1")},
	}
	cs := cohortSummaries(led, recs)
	if len(cs) != 1 {
		t.Fatalf("want 1 cohort, got %+v", cs)
	}
	if cs[0].Commit != "c2" {
		t.Errorf("headline commit must be the latest-Finished record's: %+v", cs[0])
	}
	if cs[0].ExtraCommits != 1 { // {c1,c2} → 2 distinct → 1 extra, regardless of row order
		t.Errorf("ExtraCommits must be set-cardinality-1, got %d", cs[0].ExtraCommits)
	}
}

// resolveFingerprint is git-style: unique prefix → the one full hash; ambiguous → all
// matches; zero → none; "" → no filter (empty matches, full "").
func TestResolveFingerprint(t *testing.T) {
	var led ledger.Ledger
	led.Append(fpRow("aaaa1111ffff", "r1", ""), fpRow("aaaa2222ffff", "r2", ""), fpRow("bbbb3333ffff", "r3", ""))
	if full, m := resolveFingerprint(led, "bbbb"); full != "bbbb3333ffff" || len(m) != 1 {
		t.Errorf("unique prefix: full=%q matches=%v", full, m)
	}
	if full, m := resolveFingerprint(led, "aaaa"); full != "" || len(m) != 2 {
		t.Errorf("ambiguous prefix: full=%q matches=%v", full, m)
	}
	if _, m := resolveFingerprint(led, "cccc"); len(m) != 0 {
		t.Errorf("zero match: %v", m)
	}
	if full, m := resolveFingerprint(led, ""); full != "" || m != nil {
		t.Errorf("empty prefix = no filter: full=%q matches=%v", full, m)
	}
	// Exact full-hash input must always resolve to itself.
	if full, _ := resolveFingerprint(led, "aaaa1111ffff"); full != "aaaa1111ffff" {
		t.Errorf("exact full hash: %q", full)
	}
}

// renderCohorts prints one line per cohort: short fingerprint (or "(legacy)"), row counts
// by level, first…last, commit (+dirty marker), capture status.
func TestRenderCohorts(t *testing.T) {
	var b strings.Builder
	renderCohorts(&b, []cohortSummary{
		{Fingerprint: "", Rows: 495, ByLevel: map[string]int{"": 495}},
		{Fingerprint: "566995b9deadbeef", Rows: 2166, ByLevel: map[string]int{"inner": 1944, "outer": 222},
			FirstStart: "2026-07-14T18:02:11Z", LastFinish: "2026-07-15T02:40:00Z",
			Commit: "9cea652aa", Dirty: true, CaptureStatus: "captured", Records: 2166},
	})
	out := b.String()
	for _, want := range []string{"(legacy)", "566995b9", "2166", "inner:1944", "outer:222",
		"2026-07-14T18:02:11Z", "2026-07-15T02:40:00Z", "9cea652a", "dirty", "captured"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "566995b9deadbeef") {
		t.Errorf("fingerprint should render short-8:\n%s", out)
	}
}
