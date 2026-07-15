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
// ExtraCommits is SET-cardinality (distinct commits − 1), not an incremental count:
// ledger rows are NOT time-ordered (append order is sweep-completion order), so a
// count-on-displacement fold would overcount under interleaved timestamps.
// PURE: records arrive as a map (loadLedgerRecords is the IO seam).
func cohortSummaries(led ledger.Ledger, recs map[string]record.RunRecord) []cohortSummary {
	byFP := map[string]*cohortSummary{}
	latest := map[string]string{}           // fingerprint → Finished of the record that set the headline fields
	commits := map[string]map[string]bool{} // fingerprint → distinct commit set
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
		}
		// Latest record (by Finished; ties → later ledger row) wins the headline fields —
		// even when its commit is empty (degraded capture): the dirty flag + capture status
		// are still real and must not vanish behind a missing commit (close-review finding).
		if rec.Finished >= latest[r.CodeFingerprint] {
			cs.Commit, cs.Dirty, cs.CaptureStatus = recCommit(rec), rec.Dirty, recStatus(rec)
			latest[r.CodeFingerprint] = rec.Finished
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

// recCommit / recStatus read the per-step CodeManifest a capture backfilled — every
// step of a run shares one manifest (backfillCodeManifest writes them all), so the
// first step is authoritative.
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
// DISTINCT full fingerprints: matches = every distinct fingerprint having the prefix
// (legacy blank rows are never matchable). Unique match → (full, [full]); ambiguous →
// ("", all); zero → ("", empty); "" prefix → no filter ("", nil). Pure.
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

// levelStr renders the per-level row counts in the fixed flat/inner/outer order.
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

// codeStr renders a cohort's code identity: commit (+extra-commit count), dirty, capture.
// An unknown commit (degraded capture / no records) still renders the dirty + capture
// markers when known — the dirty bit must not vanish behind a missing commit.
func codeStr(c cohortSummary) string {
	s := "commit ?"
	if c.Commit != "" {
		s = "commit " + short(c.Commit)
	}
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

// pinFingerprint resolves a --fingerprint prefix against the ledger and filters to that
// cohort. Zero match → an error that SAYS so and lists the cohorts present (the #39 fix
// for the "no scored configs" lie); ambiguous → an error listing the candidates; "" →
// no filter. The cohort table is rendered from records (IO) on the ERROR PATH only —
// a resolved pin never reads record files. Shared by select and ledger show (ARCH-DRY).
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
		return led, fmt.Errorf("nothing in the ledger matches --fingerprint %q — cohorts present:\n%s  (inspect: metis ledger fingerprints %s)",
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

// cohortGuardErr renders the metis#32 multi-cohort refusal WITH the per-cohort summary
// inline + the inspect command (metis#39) — the operator resolves the pin without
// opening the csv. Record IO happens here, on the error path only.
func cohortGuardErr(shapePath string, led ledger.Ledger, n int) error {
	var b strings.Builder
	renderCohorts(&b, cohortSummaries(led, loadLedgerRecords(shapePath, led)))
	return fmt.Errorf("select: %s spans %d code-fingerprint cohorts — a cross-version reduce would silently blend them:\n%spin one with `--fingerprint <hash>` (inspect: metis ledger fingerprints %s; or re-run `metis run %s` to refresh)",
		ledgerPath(shapePath), n, b.String(), filepath.Base(shapePath), filepath.Base(shapePath))
}

// distinctFingerprintCount counts the ledger's distinct NON-empty fingerprints — the
// cheap (no record IO) multi-cohort predicate; legacy blank rows don't count as a cohort.
func distinctFingerprintCount(led ledger.Ledger) int {
	seen := map[string]bool{}
	for _, r := range led.Rows {
		if r.CodeFingerprint != "" {
			seen[r.CodeFingerprint] = true
		}
	}
	return len(seen)
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
		if r.PointAddr == "" {
			continue
		}
		if _, ok := out[r.PointAddr]; ok {
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
