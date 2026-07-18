package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xianxu/metis/pkg/ledger"
	"github.com/xianxu/metis/pkg/record"
)

// metis#53 — the promote fingerprint-consistency guard. `select --promote` selects on ONE
// code cohort but executes against the CURRENT working tree; this guard refuses when they
// are not the same code (the silent-blend class the #32 cohort guard stops at the ledger,
// closed here at the promote seam). Detection only — restore is metis#28.

// driftedPath is one closure file whose working-tree content differs from the cohort's
// captured blob (New == "" ⇒ the path is missing/unhashable in the current tree).
type driftedPath struct {
	Repo, Path string
	Old, New   record.Hash
}

// promoteDrift compares the cohort's captured D closure against the current working tree.
// records: point_addr → record (loadLedgerRecords); cohortFP: the cohort's code fingerprint
// (every pinned row carries it); hash: the SAME per-repo blob hasher capture uses
// (gitBlobHashes — normalization identical by construction).
//
// Returns (drifted, captureCommit, checked): checked=false ⇒ nothing to compare (no record
// of this cohort carries a D closure — legacy provenance; the caller warns and proceeds,
// never blocks on absent provenance).
func promoteDrift(records map[string]record.RunRecord, cohortFP string,
	hash func(repo string, paths []string) (map[string]record.Hash, error)) (drifted []driftedPath, captureCommit string, checked bool) {

	// The cohort's captured closure: union of step D refs (dedup by repo+path) from any
	// record minted under this fingerprint. One record suffices — same fingerprint ⇒ same
	// closure content by construction (the fingerprint IS the {path, blob} manifest hash).
	type key struct{ repo, path string }
	want := map[key]record.Hash{}
	for _, rec := range records {
		if string(rec.CodeFingerprint) != cohortFP {
			continue
		}
		for _, st := range rec.Steps {
			if captureCommit == "" {
				captureCommit = st.Code.Commit
			}
			for _, cr := range st.Code.D {
				want[key{cr.Repo, cr.Path}] = cr.BlobHash
			}
		}
		if len(want) > 0 {
			break
		}
	}
	if len(want) == 0 {
		return nil, captureCommit, false
	}

	// Rehash the current tree per repo (one hasher call per repo, like capture).
	byRepo := map[string][]string{}
	for k := range want {
		byRepo[k.repo] = append(byRepo[k.repo], k.path)
	}
	for repo, paths := range byRepo {
		sort.Strings(paths)
		got, err := hash(repo, paths)
		for _, p := range paths {
			old := want[key{repo, p}]
			var now record.Hash
			if err == nil {
				now = got[p]
			}
			if now != old {
				drifted = append(drifted, driftedPath{Repo: repo, Path: p, Old: old, New: now})
			}
		}
	}
	sort.Slice(drifted, func(i, j int) bool {
		if drifted[i].Repo != drifted[j].Repo {
			return drifted[i].Repo < drifted[j].Repo
		}
		return drifted[i].Path < drifted[j].Path
	})
	return drifted, captureCommit, true
}

// promoteDriftError renders the diff-shaped refusal: every drifted path with old/new short
// blobs, the restore handle (the cohort's capture commit), and the loud escape hatch.
func promoteDriftError(drifted []driftedPath, captureCommit, cohortFP string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "select --promote: the working tree is NOT the selected cohort's code (fingerprint %s) — the promoted run would ship code that never produced the honest estimate you selected on:\n", short(cohortFP))
	for _, d := range drifted {
		now := string(d.New)
		if now == "" {
			now = "<missing>"
		} else {
			now = short(string(d.New))
		}
		fmt.Fprintf(&b, "  %s: %s (captured %s → working %s)\n", d.Repo, d.Path, short(string(d.Old)), now)
	}
	if captureCommit != "" {
		fmt.Fprintf(&b, "restore: git checkout %s -- <path> (per repo; metis#28 will own a verb for this)\n", short(captureCommit))
	}
	b.WriteString("override: --no-fingerprint-check (promotes ANYWAY against the drifted tree — the mismatch will be visible on the promote run's `recording under code_fingerprint` line)")
	return fmt.Errorf("%s", b.String())
}

// guardPromoteFingerprint is the seam both promote paths call before executing the promoted
// run. cohortFP comes from the pinned rows (every row of the pinned cohort carries it).
func guardPromoteFingerprint(shapePath string, led ledger.Ledger, cohortFP string, skip bool, out func(string)) error {
	if cohortFP == "" {
		return nil // nothing pinned/known — no cohort identity to defend
	}
	records := loadLedgerRecords(shapePath, led)
	drifted, commit, checked := promoteDrift(records, cohortFP, gitBlobHashes)
	if !checked {
		out(fmt.Sprintf("metis: promote guard: cohort %s has no D-closure records (legacy provenance) — nothing to compare, proceeding", short(cohortFP)))
		return nil
	}
	if len(drifted) == 0 {
		return nil
	}
	if skip {
		out(fmt.Sprintf("metis: promote guard OVERRIDDEN (--no-fingerprint-check): %d closure path(s) drifted from cohort %s — the promoted run is NOT the selected code", len(drifted), short(cohortFP)))
		return nil
	}
	return promoteDriftError(drifted, commit, cohortFP)
}

// cohortFingerprintOf returns the (post-pin, single) cohort's fingerprint — any row carries
// it; empty when the ledger has no fingerprinted rows (legacy).
func cohortFingerprintOf(led ledger.Ledger) string {
	for _, r := range led.Rows {
		if r.CodeFingerprint != "" {
			return r.CodeFingerprint
		}
	}
	return ""
}
