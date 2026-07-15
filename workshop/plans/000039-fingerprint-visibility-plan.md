# Fingerprint Visibility Implementation Plan (metis#39)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `code_fingerprint` visible: `metis run` prints the cohort it records under, a new `metis ledger fingerprints` command lists the ledger's cohorts with pick-enabling attributes, `--fingerprint` accepts git-style unique prefixes, and the select cohort-guard / zero-match errors render the cohort summary instead of bare hashes or lies.

**Architecture:** Everything is presentation over existing capture data — no new instrumentation. A pure cohort-summary reducer (`cohortSummaries`) folds ledger rows + their per-run `record.json` metadata into per-fingerprint summaries; one renderer serves the new command, the upgraded guard error, and the zero-match error (ARCH-DRY). A pure prefix resolver (`resolveFingerprint`) is shared by `metis select` and `metis ledger show` (ARCH-DRY). Record-file IO stays in one thin loader; records are only read on the inspect command and on error paths, never on the happy select path (ARCH-PURE).

**Tech Stack:** Go (cmd/metis), stdlib only. Tests: `go test ./cmd/metis/`.

---

## Why each piece exists (issue → design)

| Issue spec item | Design |
|---|---|
| 1. `metis run` prints its cohort | `backfillCodeManifest` already mints the fingerprint and has the record open (it reads `rec.Dirty`); it now *returns* `(fp, dirty)` (declared at capture.go:324, mint at :349). The two capture sites (`captureSweepCode`, `captureSingleRun`) print one line via a shared `printFingerprintLine`. Nested and `--fast` runs go through `captureSweepCode`; a flat run through `captureSingleRun` — both covered, no third mint site created. |
| 2. inspect command | **Name decided: `metis ledger fingerprints <shape.md>`** — it is a view over the ledger sidecar, so it sits beside `metis ledger show` rather than claiming a top-level verb. Discoverability is carried by the guard error, which names the command verbatim. |
| 3. guard upgrades | (a) prefix resolution before `ledger.Filter` (which stays exact — the storage primitive is untouched); (b) zero-match error says "nothing matches" and inlines the cohort table; (c) the multi-cohort refusal inlines the same table + names the command. |

**Data flow for the summary:** ledger `Row.PointAddr` **is** the run ID (`rowsFromManifest`: `PointAddr: p.RunID`), and each run's `runs/<id>/record.json` carries `Started`/`Finished` (RFC3339 — sorts lexically = chronologically), `Dirty`, and `Steps[].Code.{Commit,CaptureStatus}`. A run dir the operator cleaned → record missing → the summary shows `?` for that cohort's unknowns (tolerated, never fatal).

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `cohortSummary` / `cohortSummaries` | `cmd/metis/fingerprints.go` | new |
| `resolveFingerprint` | `cmd/metis/fingerprints.go` | new |
| `renderCohorts` | `cmd/metis/fingerprints.go` | new |
| `printFingerprintLine` | `cmd/metis/capture.go` | new |
| `backfillCodeManifest` | `cmd/metis/capture.go` | modified |
| `distinctFingerprints` | `cmd/metis/select_cmd.go` | deleted |

- **`cohortSummary` / `cohortSummaries(led ledger.Ledger, recs map[string]record.RunRecord) []cohortSummary`** — the reducer: group rows by full `CodeFingerprint` ("" groups as legacy), count rows total + by `Level` ("" = flat), and fold matched records into `FirstStart`/`LastFinish` (min Started / max Finished), the *latest* record's `Commit`/`Dirty`/`CaptureStatus`, `ExtraCommits`, and `Records` (how many rows had a record — surfaces cleaned run dirs). **`ExtraCommits` is SET-CARDINALITY, not an incremental count**: accumulate the cohort's distinct commits in a set during the fold, track the latest-`Finished` record separately for the headline fields, and set `ExtraCommits = len(set)-1` at fold end — ledger rows are NOT time-ordered (`sortPointRuns` orders by content key; two interleaved sweeps of the same code produce non-monotone `Finished`), so an incremental "count on displacement" fold overcounts (plan-review finding). Ordering: by `LastFinish` ascending (**newest last**, per Done-when), unknowns first, tie-break by fingerprint. Pure — no IO, tests need no mocks (ARCH-PURE).
  - **Relationships:** 1:N with ledger rows; N:1 records-per-cohort. Consumed by the command, the guard error, and the zero-match error (1 reducer : 3 surfaces — ARCH-DRY).
  - **DRY rationale:** without it, the guard error, the zero-match error, and the command would each re-derive per-cohort stats (the exact drift #39 exists to stop).
  - **Future extensions:** a `--json` output for agents; per-cohort config counts.
- **`resolveFingerprint(led ledger.Ledger, prefix string) (full string, matches []string)`** — git-style prefix resolution over the *distinct full* fingerprints in the ledger: returns all distinct fingerprints having `prefix` (a full hash matches itself trivially). Caller semantics: `len==1` → resolved; `0` → zero-match error; `>1` → ambiguous error listing matches. `prefix==""` → no filter. Pure.
  - **Relationships:** shared by `runSelect` and `showLedger` — one resolution semantics for every `--fingerprint` flag (ARCH-DRY; `--point` already resolves prefixes this way, so the two flags stop diverging).
  - **DRY rationale:** the operator-hit defect is exactly that select's filter had *different* matching semantics than `--point`; one resolver ends the split.
- **`renderCohorts(w io.Writer, cs []cohortSummary)`** — the one table renderer (fingerprint short-8 or `(legacy)`, rows by level, first…last, commit + dirty marker, capture status, records/rows visibility). Reuses `short()`.
- **`backfillCodeManifest`** — modified to return `(fp record.Hash, dirty bool, err error)`. It is already the ONE post-capture site where the fingerprint is minted (capture.go:349) and it has the decoded record in hand — returning what it minted avoids a second mint site or a re-read (ARCH-DRY). Missing record (early-nil path) → zero values.
- **`printFingerprintLine(out io.Writer, fp record.Hash, commit string, dirty bool)`** — one formatter for the run-time cohort line (signature single-sourced here; Task 3's sketch is the same function), e.g. `metis: recording under code_fingerprint b7aee3de (commit 9cea652, dirty)` — capture *status* is deliberately NOT in this line (`warnOnUncaptured` already owns durability messaging; this line states identity only, matching the spec's example). Prints whatever will actually land in the ledger, even for degraded capture. Nil/short-circuit on `out == nil` (like `warnOnUncaptured`) and on `fp == ""`.
- **`distinctFingerprints`** — deleted: its only caller was the guard, which now consumes `cohortSummaries` (whose keys are the full fingerprints; the 8-char truncation moves to render time where it belongs).

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `loadLedgerRecords` | `cmd/metis/fingerprints.go` | new | filesystem (`runs/<id>/record.json`) |
| `cmdLedger` (`fingerprints` verb) | `cmd/metis/ledger_cmd.go` | modified | CLI |
| `runSelect` guard + filter | `cmd/metis/select_cmd.go` | modified | CLI |
| `showLedger` filter | `cmd/metis/ledger_cmd.go` | modified | CLI |
| capture sites | `cmd/metis/capture.go` | modified | stdout |

- **`loadLedgerRecords(shapePath string, led ledger.Ledger) map[string]record.RunRecord`** — reads `record.json` for each distinct `PointAddr` in the ledger (missing/unparseable → skipped, never fatal — an inspect surface must not refuse over a cleaned run dir; contrast `loadSweepRecords`, which is manifest-driven and write-path). Called ONLY by the inspect command and by error-path rendering — the happy select path never touches record files.
  - **Injected into:** `cohortSummaries` receives its output as a plain map — the reducer stays pure, unit-tested with literal maps.
- **Existing fakes suffice:** no external service is touched; e2e rides the existing toy-pipeline workspace + `fakeGitProbe` harness.

**Test surface:** `fingerprints_test.go` (pure: reducer fixture with two cohorts + legacy blanks; resolver unique/ambiguous/zero/empty cases; renderer golden-ish assertions) · `select_cmd_test.go` (guard message upgrade, prefix pin works — the operator's exact 8-char repro, zero-match error content) · `ledger_cmd_test.go` (CLI `fingerprints` verb through the real `run([]string{...})` entrypoint with the documented arg order — lessons.md: never call the handler directly) · `capture_test.go`/e2e (the recording line for nested AND flat runs, per Done-when).

---

## Tasks

Single-pass atomic work — plain checkboxes, no `Mx` tags (one `sdlc close` boundary, AGENTS.md §3).

### Task 1: Pure core — reducer + resolver (`fingerprints.go`)

**Files:**
- Create: `cmd/metis/fingerprints.go`
- Test: `cmd/metis/fingerprints_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// cmd/metis/fingerprints_test.go
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
	// Ordering: legacy (no records → unknown) first, then aaaa (last finish 11:05), then bbbb (09:01 on the 15th).
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
	// A full hash resolves to itself even when another fingerprint shares its prefix… (not
	// possible for equal-length hex hashes, but exact input must always work:)
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
```

- [ ] **Step 2: Run to verify failure**

Run: `cd /Users/xianxu/workspace/metis && go test ./cmd/metis/ -run 'TestCohortSummaries|TestResolveFingerprint|TestRenderCohorts'`
Expected: FAIL — `undefined: cohortSummaries` etc.

- [ ] **Step 3: Implement `cmd/metis/fingerprints.go`**

```go
package main

// metis#39: fingerprint visibility — the cohort-summary reducer + git-style prefix
// resolver + renderer behind `metis ledger fingerprints`, the select cohort-guard
// error, and the zero-match error. Presentation over existing capture data only
// (ledger rows + per-run record.json); no new instrumentation.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xianxu/metis/pkg/ledger"
	"github.com/xianxu/metis/pkg/record"
)

// cohortSummary is one code-fingerprint cohort's operator-facing identity: what ran,
// when, from what code — the attributes that let an operator pick a `--fingerprint`.
type cohortSummary struct {
	Fingerprint   string         // full hash; "" = pre-fingerprint legacy rows
	Rows          int            // ledger rows in the cohort
	ByLevel       map[string]int // "" (flat) | "inner" | "outer" → count
	FirstStart    string         // min record Started (RFC3339; "" unknown)
	LastFinish    string         // max record Finished ("" unknown)
	Commit        string         // the LATEST matched record's commit
	ExtraCommits  int            // distinct commits beyond Commit (same code, different capture sessions)
	Dirty         bool           // the latest matched record's dirty flag
	CaptureStatus string         // the latest matched record's capture status
	Records       int            // rows whose record.json was found (visibility into cleaned run dirs)
}

// cohortSummaries reduces ledger rows + their run records to per-fingerprint summaries,
// ordered oldest→NEWEST-LAST by LastFinish (record-less cohorts first, fingerprint
// tie-break) — deterministic, and the freshest cohort lands at the eye's resting point.
// PURE: records arrive as a map (loadLedgerRecords is the IO seam).
func cohortSummaries(led ledger.Ledger, recs map[string]record.RunRecord) []cohortSummary {
	byFP := map[string]*cohortSummary{}
	latest := map[string]string{}          // fingerprint → Finished of the record that set the headline fields
	commits := map[string]map[string]bool{} // fingerprint → DISTINCT commit set (ExtraCommits = len-1 at fold end;
	// ledger rows are NOT time-ordered, so an incremental count would overcount — set-cardinality is order-proof)
	for _, r := range led.Rows {
		cs := byFP[r.CodeFingerprint]
		if cs == nil {
			cs = &cohortSummary{Fingerprint: r.CodeFingerprint, ByLevel: map[string]int{}}
			byFP[r.CodeFingerprint] = cs
			commits[r.CodeFingerprint] = map[string]bool{}
		}
		cs.Rows++
		cs.ByLevel[r.Level]++
		rec, ok := recs[r.PointAddr]
		if !ok {
			continue
		}
		cs.Records++
		if cs.FirstStart == "" || rec.Started < cs.FirstStart {
			cs.FirstStart = rec.Started
		}
		if rec.Finished > cs.LastFinish {
			cs.LastFinish = rec.Finished
		}
		if commit := recCommit(rec); commit != "" {
			commits[r.CodeFingerprint][commit] = true
			// Latest record (by Finished; ties → later ledger row) wins the headline fields.
			if rec.Finished >= latest[r.CodeFingerprint] {
				cs.Commit, cs.Dirty, cs.CaptureStatus = commit, rec.Dirty, recStatus(rec)
				latest[r.CodeFingerprint] = rec.Finished
			}
		}
	}
	out := make([]cohortSummary, 0, len(byFP))
	for fp, cs := range byFP {
		if n := len(commits[fp]); n > 1 {
			cs.ExtraCommits = n - 1
		}
		out = append(out, *cs)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastFinish != out[j].LastFinish {
			return out[i].LastFinish < out[j].LastFinish
		}
		return out[i].Fingerprint < out[j].Fingerprint
	})
	return out
}

func recCommit(rec record.RunRecord) string {
	if len(rec.Steps) == 0 {
		return ""
	}
	return rec.Steps[0].Code.Commit
}

func recStatus(rec record.RunRecord) string {
	if len(rec.Steps) == 0 {
		return ""
	}
	return rec.Steps[0].Code.CaptureStatus
}

// resolveFingerprint resolves a --fingerprint value git-style against the ledger's
// DISTINCT full fingerprints: matches = every distinct fingerprint having the prefix.
// Unique match → (full, [full]); ambiguous → ("", all); zero → ("", nil-len-0);
// "" prefix → no filter. Pure.
func resolveFingerprint(led ledger.Ledger, prefix string) (string, []string) {
	if prefix == "" {
		return "", nil
	}
	seen := map[string]bool{}
	var matches []string
	for _, r := range led.Rows {
		if r.CodeFingerprint != "" && strings.HasPrefix(r.CodeFingerprint, prefix) && !seen[r.CodeFingerprint] {
			seen[r.CodeFingerprint] = true
			matches = append(matches, r.CodeFingerprint)
		}
	}
	sort.Strings(matches)
	if len(matches) == 1 {
		return matches[0], matches
	}
	return "", matches
}

// renderCohorts prints the cohort table — the ONE renderer behind `metis ledger
// fingerprints`, the select cohort-guard error, and the zero-match error (ARCH-DRY).
func renderCohorts(w io.Writer, cs []cohortSummary) {
	for _, c := range cs {
		fp := "(legacy)"
		if c.Fingerprint != "" {
			fp = short(c.Fingerprint)
		}
		fmt.Fprintf(w, "  %-10s %5d rows (%s)  %s … %s  %s\n",
			fp, c.Rows, levelStr(c.ByLevel), orQ(c.FirstStart), orQ(c.LastFinish), codeStr(c))
	}
}

func levelStr(byLevel map[string]int) string {
	var parts []string
	for _, lv := range []string{"", "inner", "outer"} {
		if n := byLevel[lv]; n > 0 {
			name := lv
			if name == "" {
				name = "flat"
			}
			parts = append(parts, fmt.Sprintf("%s:%d", name, n))
		}
	}
	return strings.Join(parts, " ")
}

func orQ(s string) string {
	if s == "" {
		return "?"
	}
	return s
}

func codeStr(c cohortSummary) string {
	if c.Commit == "" {
		return "commit ?"
	}
	s := "commit " + short(c.Commit)
	if c.ExtraCommits > 0 {
		s += fmt.Sprintf(" (+%d more)", c.ExtraCommits)
	}
	if c.Dirty {
		s += ", dirty"
	}
	if c.CaptureStatus != "" {
		s += ", " + c.CaptureStatus
	}
	return s
}

// loadLedgerRecords reads record.json for each distinct run the ledger references
// (Row.PointAddr IS the run id — rowsFromManifest). Missing/unparseable records are
// SKIPPED (an inspect surface tolerates cleaned run dirs; the summary shows "?").
// The IO seam for cohortSummaries — called on the inspect command and on select's
// error paths only, never on the happy select path.
func loadLedgerRecords(shapePath string, led ledger.Ledger) map[string]record.RunRecord {
	dir := filepath.Dir(shapePath)
	out := map[string]record.RunRecord{}
	for _, r := range led.Rows {
		if r.PointAddr == "" || len(out) > 0 && func() bool { _, ok := out[r.PointAddr]; return ok }() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, "runs", r.PointAddr, "record.json"))
		if err != nil {
			continue
		}
		var rec record.RunRecord
		if err := json.Unmarshal(b, &rec); err != nil {
			continue
		}
		out[r.PointAddr] = rec
	}
	return out
}
```

*(Implementation note: simplify the dedup guard in `loadLedgerRecords` to a plain `if _, ok := out[r.PointAddr]; ok { continue }` — write it cleanly, the sketch above is compressed.)*

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./cmd/metis/ -run 'TestCohortSummaries|TestResolveFingerprint|TestRenderCohorts'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/metis/fingerprints.go cmd/metis/fingerprints_test.go
git commit -m "#39: cohort-summary reducer + git-style fingerprint resolver (pure core)"
```

### Task 2: `metis ledger fingerprints` command

**Files:**
- Modify: `cmd/metis/ledger_cmd.go` (extend `cmdLedger`)
- Test: `cmd/metis/ledger_cmd_test.go`

- [ ] **Step 1: Write the failing CLI test — through the REAL entrypoint, documented arg order**

(lessons.md: an e2e that calls the handler directly bypasses the CLI parse.) Build a temp shape + ledger CSV + two `runs/<id>/record.json` fixtures (two cohorts + a legacy blank row) — copy the fixture style of the existing ledger-writing helpers in `ledger_cmd_test.go`/`select_cmd_test.go` (e.g. `writePerFoldLedger`), then:

```go
// Drives the real `metis ledger fingerprints <shape.md>` CLI path (run() → cmdLedger),
// documented arg order, and asserts the per-cohort table: short fingerprints, legacy
// group, row counts by level, timestamps, commit+dirty, capture status.
func TestLedgerFingerprints_CLI(t *testing.T) { ... run([]string{"ledger", "fingerprints", shapePath}) ... }
```

Assert stdout contains the two short hashes, `(legacy)`, level counts, a timestamp from the fixture records, and `commit`. (Capture stdout the same way sibling CLI tests in this file do; if they don't, swap `os.Stdout` via a pipe helper.)

- [ ] **Step 2: Verify it fails** — `go test ./cmd/metis/ -run TestLedgerFingerprints_CLI` → FAIL (usage error: only `show` is known).

- [ ] **Step 3: Implement — extend `cmdLedger`**

In `ledger_cmd.go`, replace the `args[0] != "show"` gate with a verb switch:

```go
func cmdLedger(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: metis ledger show|fingerprints <shape.md> [flags]")
	}
	switch args[0] {
	case "show":
		return cmdLedgerShow(args[1:]) // existing body, extracted
	case "fingerprints":
		return cmdLedgerFingerprints(args[1:])
	default:
		return fmt.Errorf("unknown ledger subcommand %q (want: show | fingerprints)", args[0])
	}
}

// cmdLedgerFingerprints handles `metis ledger fingerprints <shape.md>` — metis#39's
// inspect surface: the ledger's code-fingerprint cohorts with the attributes that let
// an operator pick a --fingerprint (rows by level, first…last run, commit+dirty, capture).
func cmdLedgerFingerprints(args []string) error {
	shapePath, flags, err := hoistShapePath(args)
	if err != nil {
		return fmt.Errorf("ledger fingerprints: %w (usage: metis ledger fingerprints <shape.md>)", err)
	}
	if len(flags) > 0 { // no flags yet — don't silently swallow a typo'd --bogus
		return fmt.Errorf("ledger fingerprints: unexpected args %v (usage: metis ledger fingerprints <shape.md>)", flags)
	}
	return showFingerprints(shapePath, os.Stdout)
}

// showFingerprints is the testable core: load ledger + records, reduce, render.
func showFingerprints(shapePath string, out io.Writer) error {
	led, err := loadLedger(shapePath)
	if err != nil {
		return err
	}
	if len(led.Rows) == 0 {
		fmt.Fprintf(out, "(no ledger rows in %s)\n", ledgerPath(shapePath))
		return nil
	}
	cs := cohortSummaries(led, loadLedgerRecords(shapePath, led))
	fmt.Fprintf(out, "metis: %s — %d code-fingerprint cohort(s):\n", ledgerPath(shapePath), len(cs))
	renderCohorts(out, cs)
	return nil
}
```

Also update `main.go`'s `unknown subcommand` text if it enumerates ledger verbs (it doesn't — only top-level; no change).

- [ ] **Step 4: Verify pass** — `go test ./cmd/metis/ -run TestLedgerFingerprints_CLI` → PASS. Also run the full package: `go test ./cmd/metis/` (the extracted `cmdLedgerShow` must not break existing ledger-show tests).

- [ ] **Step 5: Commit** — `git commit -m "#39: metis ledger fingerprints — cohort inspect command"`

### Task 3: `metis run` prints its cohort line

**Files:**
- Modify: `cmd/metis/capture.go` (`backfillCodeManifest` returns `(fp, dirty, err)`; `captureSweepCode` + `captureSingleRun` print once; add `printFingerprintLine`)
- Test: `cmd/metis/capture_test.go` (or the e2e output assertions — wherever the existing capture-status prints are asserted)

- [ ] **Step 1: Write the failing tests.** Two assertions, per Done-when: a **nested/sweep** run's output and a **flat single** run's output each contain `recording under code_fingerprint <short>`. Find the existing tests that drive `captureSweepCode`/`captureSingleRun` (capture_test.go, nestedcv_e2e_test.go, record_e2e_test.go) and extend the nearest output-asserting ones; if none capture output, add focused tests that call the capture functions with a `bytes.Buffer` out.

- [ ] **Step 2: Verify failure.**

- [ ] **Step 3: Implement.**

```go
// backfillCodeManifest ... now returns the minted fingerprint + the record's dirty flag
// (the ONE mint site already has the record open — callers print, not re-derive).
func backfillCodeManifest(expPath, runID string, d []record.CodeRef, commit, status string) (record.Hash, bool, error) {
	// ... existing body; early os.IsNotExist return → ("", false, nil)
	return fp, rec.Dirty, writeRecordJSON(...)
}

// printFingerprintLine states the cohort identity a run records under — the line the
// select cohort-guard will later name, so the operator has SEEN the hash (metis#39).
func printFingerprintLine(out io.Writer, fp record.Hash, commit string, dirty bool) {
	if out == nil || fp == "" {
		return
	}
	state := "clean"
	if dirty {
		state = "dirty"
	}
	c := "?"
	if commit != "" {
		c = short(commit)
	}
	fmt.Fprintf(out, "metis: recording under code_fingerprint %s (commit %s, %s)\n", short(string(fp)), c, state)
}
```

`captureSweepCode`: keep the first **non-empty** `(fp, dirty)` the backfill loop returns (a point whose `record.json` is missing returns `("", false, nil)` — all points in a sweep share one fingerprint, so any non-empty one is THE cohort) and print ONCE after the loop. `captureSingleRun`: print after its backfill. Zero-points / all-records-missing degenerates safely: `printFingerprintLine` short-circuits on `fp == ""`. (The capture-status warning stays separate — `warnOnUncaptured` is about durability, this line is about identity.)

- [ ] **Step 4: Verify** — `go test ./cmd/metis/` all green (compile fixes at other `backfillCodeManifest` call sites, if any).

- [ ] **Step 5: Commit** — `git commit -m "#39: metis run prints its code_fingerprint cohort line"`

### Task 4: Prefix resolution + honest guard/zero-match errors in `select` and `ledger show`

**Files:**
- Modify: `cmd/metis/select_cmd.go` (resolution before `ledger.Filter`; guard message; delete `distinctFingerprints`)
- Modify: `cmd/metis/ledger_cmd.go` (`showLedger` uses the same resolution)
- Test: `cmd/metis/select_cmd_test.go`, `cmd/metis/ledger_cmd_test.go`

- [ ] **Step 1: Write the failing tests** (fixture: multi-cohort ledger + records as in Task 2):

1. `--fingerprint <8-char prefix>` on select resolves and filters (the operator's exact repro: prefix of a full hash present → rows found, no zero-match lie).
2. Zero-match: `--fingerprint cccc` → error contains `nothing in the ledger matches` AND the cohort table (a short fingerprint + row count) — not "no scored configs".
3. Ambiguous: two cohorts sharing a prefix → error lists both.
4. The multi-cohort no-pin guard error now contains `metis ledger fingerprints` and at least one per-cohort summary line (update the existing `TestSelect_MixedFingerprintCohortsError` — the func below the comment block at select_cmd_test.go:134).
5. `ledger show --fingerprint <prefix>` filters the table the same way (shared resolver).

- [ ] **Step 2: Verify failures.**

- [ ] **Step 3: Implement.** Shared helper in `fingerprints.go`:

```go
// pinFingerprint resolves a --fingerprint prefix against the ledger and filters to that
// cohort. Zero match → an error that says so and LISTS the cohorts present (the #39 fix
// for the "no scored configs" lie); ambiguous → an error listing the candidates. The
// cohort table is rendered from records (IO) on the ERROR PATH only.
func pinFingerprint(shapePath string, led ledger.Ledger, prefix string) (ledger.Ledger, error) {
	if prefix == "" {
		return led, nil
	}
	full, matches := resolveFingerprint(led, prefix)
	switch len(matches) {
	case 1:
		return ledger.Filter(led, full), nil
	case 0:
		var b strings.Builder
		renderCohorts(&b, cohortSummaries(led, loadLedgerRecords(shapePath, led)))
		return led, fmt.Errorf("nothing in the ledger matches --fingerprint %q — cohorts present:\n%s(inspect: metis ledger fingerprints %s)",
			prefix, b.String(), filepath.Base(shapePath))
	default:
		shorts := make([]string, len(matches))
		for i, m := range matches {
			shorts[i] = short(m)
		}
		return led, fmt.Errorf("--fingerprint %q is ambiguous across %d cohorts %v — disambiguate (inspect: metis ledger fingerprints %s)",
			prefix, len(matches), shorts, filepath.Base(shapePath))
	}
}
```

In `runSelect`: replace `led = ledger.Filter(led, o.fingerprint)` with `led, err = pinFingerprint(o.shapePath, led, o.fingerprint)` (+ error return). Replace the multi-cohort guard body — **order matters (ARCH-PURE / no record IO on the happy path):** first count distinct NON-empty `CodeFingerprint`s straight off `led.Rows` (cheap, pure — preserves current semantics: legacy blank rows don't count as a cohort, exactly as `distinctFingerprints` skipped `""`); only INSIDE the `>1` error branch call `loadLedgerRecords` + `cohortSummaries` + `renderCohorts` into the error text, ending with `pin one with --fingerprint <hash> (inspect: metis ledger fingerprints <shape>)`. Delete `distinctFingerprints`. In `showLedger`: same `pinFingerprint` call. **Noted behavior change:** `ledger show --fingerprint <no-match>` currently prints `(no rows)` and exits 0; it now errors with the cohort listing — intentional, this is exactly Log defect (b), and no existing test pins the old behavior (verified at plan review).

- [ ] **Step 4: Verify** — full `go test ./cmd/metis/` and `go vet ./...`.

- [ ] **Step 5: Commit** — `git commit -m "#39: git-style --fingerprint prefixes + honest zero-match/cohort-guard errors"`

### Task 5: Sweep the operator-facing docs + real-CLI smoke

- [ ] **Step 1:** Grep kbench's `RUNBOOK-sweep.md` + metis `docs/`/`atlas/` for the interim recipe (`tail -1 <ledger>.csv | cut -d, -f1`) and for `--fingerprint` guidance; replace with `metis ledger fingerprints` where present (the recipe is documented in the issue Log — check RUNBOOK §1 forkserver note area).
- [ ] **Step 2:** Real-binary smoke against the live kbench ledger (read-only): `go build -o /tmp/claude/metis ./cmd/metis && /tmp/claude/metis ledger fingerprints <kbench titanic shape.md>` — expect the 566995b9 cohort (2,166 rows) + any legacy cohorts, timestamps populated. Then `metis select <shape> --fingerprint 566995b9` (8-char prefix) → resolves, prints the board (no promote).
- [ ] **Step 3:** Update `atlas/` (CLI surface page: new verb + run-line + prefix semantics), issue `## Log`.
- [ ] **Step 4:** Commit docs; then `sdlc close --issue 39 --verified '<evidence incl. smoke output>'`.

## Verification (Done-when → checks)

| Done-when | Check |
|---|---|
| nested + flat run print their fingerprint | Task 3 tests (sweep + single capture output) |
| `fingerprints` on multi-cohort ledger: per-cohort rows-by-level, first/last, commit+dirty, deterministic newest-last | Task 1 reducer test + Task 2 CLI test |
| guard names the command / inlines summary; operator resolves without opening the csv | Task 4 test 4 + Task 5 smoke on the real 566995b9 ledger |
| (Log defect a) prefix `--fingerprint` works | Task 4 test 1 + smoke |
| (Log defect b) zero-match no longer lies | Task 4 test 2 |
