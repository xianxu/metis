package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xianxu/metis/pkg/cas"
	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/record"
)

// gitProbe reports a repo's provenance — its short name, HEAD sha, and dirty flag —
// injected (like the clock) so record assembly is testable without shelling git.
type gitProbe interface {
	Probe(dir string) (name, sha string, dirty bool, err error)
}

// gitCLI is the production gitProbe: it shells `git -C <dir> …` (files + subprocess,
// the ARCH-PURE IO seam), never a git library.
type gitCLI struct{}

func (gitCLI) Probe(dir string) (name, sha string, dirty bool, err error) {
	top, err := gitOut(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", false, err
	}
	sha, err = gitOut(dir, "rev-parse", "HEAD")
	if err != nil {
		return "", "", false, err
	}
	status, err := gitOut(dir, "status", "--porcelain")
	if err != nil {
		return "", "", false, err
	}
	return filepath.Base(top), sha, strings.TrimSpace(status) != "", nil
}

func gitOut(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// assembleRecord builds the provenance record for one run: it probes git for the
// repo provenance, content-hashes each step's output artifacts (IO), then hands the
// pieces to the pure buildRecord. dir is the experiment's dir (its repo is the
// provenance anchor for v1; multi-repo capture is later, metis#7/#8).
//
// Git provenance degrades gracefully: if the probe fails (e.g. running outside a git
// repo) the run does NOT fail — it warns and records no repo-SHAs (the design's "v1:
// warn" for a non-reproducible run). The point-address still mints from config+seed.
func assembleRecord(git gitProbe, out io.Writer, dir, runDir string, run experiment.Run, steps []experiment.StepRun) (record.RunRecord, error) {
	if git == nil {
		git = gitCLI{}
	}
	name, sha, dirty, err := git.Probe(dir)
	if err != nil {
		fmt.Fprintf(out, "metis: warning: no git provenance for %s (%v) — record omits repo-SHAs; the run is not commit-reproducible\n", dir, err)
		name, sha, dirty = "", "", false
	}
	outputHashes := make(map[string]record.Hash, len(steps))
	for _, sr := range steps {
		fhs, err := hashArtifacts(runDir, sr.Result.Artifacts)
		if err != nil {
			return record.RunRecord{}, err
		}
		outputHashes[sr.Step.ID] = record.OutputHash(fhs)
	}
	return buildRecord(run, steps, outputHashes, name, sha, dirty)
}

// buildRecord assembles the RunRecord from the executed steps, their per-step output
// hashes (computed by the caller from artifact bytes), and the git provenance, and
// mints the point-address from the resolved config + repo SHA + seed. Pure aside from
// PointAddress (which errors on non-finite config). #3 fills the coarse code identity
// (commit + dirty); Upstream is populated below (each step's needs → the upstream
// output-hashes, sorted — the metis#2 K_pre wiring). Code.D / Deps stay empty in the
// record — that provenance population is deferred to metis#8 (git-side-ref durability).
func buildRecord(run experiment.Run, steps []experiment.StepRun, outputHashes map[string]record.Hash, repoName, sha string, dirty bool) (record.RunRecord, error) {
	resolvedWith := make(map[string]map[string]any, len(steps))
	stepRecs := make([]record.StepRecord, 0, len(steps))
	for _, sr := range steps {
		resolvedWith[sr.Step.ID] = sr.Step.With
		// Populate Upstream (the metis#3 slot #2 fills): this step's needs → the
		// upstream steps' output-hashes (sorted — shared upstreamHashes helper, so this
		// and cachingExecutor.kpre derive K_pre's upstream term identically).
		stepRecs = append(stepRecs, record.StepRecord{
			StepID:     sr.Step.ID,
			Uses:       sr.Step.Uses,
			With:       sr.Step.With,
			Upstream:   upstreamHashes(sr.Step.Needs, outputHashes),
			Code:       record.CodeManifest{Commit: sha, Dirty: dirty},
			OutputHash: outputHashes[sr.Step.ID],
			Metrics:    sr.Result.Metrics,
		})
	}
	// Single-source the {repoName: sha} construction (repoSHAsOf) so the sweep driver's
	// pre-computed point-address runID can't drift from this record's internal address.
	repoSHAs := repoSHAsOf(repoName, sha)
	addr, err := record.PointAddress(resolvedWith, repoSHAs, run.Seed)
	if err != nil {
		return record.RunRecord{}, err
	}
	return record.RunRecord{
		RunID:        run.ID,
		Experiment:   run.Experiment,
		Seed:         run.Seed,
		PointAddress: addr,
		RepoSHAs:     repoSHAs,
		Dirty:        dirty,
		Steps:        stepRecs,
		Started:      run.Started,
		Finished:     run.Finished,
		Status:       run.Status,
	}, nil
}

// hashArtifacts content-hashes each of a step's output artifacts (slash paths under
// runDir) into the FileHash list record.OutputHash reduces to one address.
func hashArtifacts(runDir string, artifacts []string) ([]record.FileHash, error) {
	fhs := make([]record.FileHash, 0, len(artifacts))
	for _, rel := range artifacts {
		b, err := os.ReadFile(filepath.Join(runDir, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("hash artifact %s: %w", rel, err)
		}
		fhs = append(fhs, record.FileHash{Path: rel, Hash: cas.HashOf(b)})
	}
	return fhs, nil
}

// writeRecordJSON writes the provenance record to runs/<id>/record.json — small
// git-trackable metadata (durable-small → git, per the design; NOT the CAS).
func writeRecordJSON(runDir string, rec record.RunRecord) error {
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "record.json"), append(b, '\n'), 0o644)
}

// recordSummary renders the one-line knob→score `## Runs` entry the record makes
// possible: the resolved config knobs beside the metrics, so the log reads as
// knob→score rather than an opaque metric line. (The full free-param *table* is
// metis#8's ledger; this is the per-run line.)
func recordSummary(rec record.RunRecord) string {
	s := fmt.Sprintf("%s — %s — %s", rec.RunID, rec.Status, rec.Finished)
	if knobs := formatKnobs(rec); knobs != "" {
		s += " — knobs: " + knobs
	}
	metrics := map[string]float64{}
	for _, st := range rec.Steps {
		for k, v := range st.Metrics {
			metrics[k] = v // flat merge across steps, matching the run ledger
		}
	}
	if m := formatMetrics(metrics); m != "" {
		s += " — metrics: " + m
	}
	return s
}

// formatKnobs renders each step's resolved config as `stepID.key=value`, steps in
// topo order, keys sorted — the "knobs" half of the knob→score line.
func formatKnobs(rec record.RunRecord) string {
	var parts []string
	for _, st := range rec.Steps {
		keys := make([]string, 0, len(st.With))
		for k := range st.With {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s.%s=%v", st.StepID, k, st.With[k]))
		}
	}
	return strings.Join(parts, " ")
}

// formatMetrics renders a flat metric map as `key=value` pairs, sorted — extracted
// from the old runSummary so the record renderer reuses one formatter (ARCH-DRY).
func formatMetrics(m map[string]float64) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s=%g", k, m[k])
	}
	return strings.Join(parts, " ")
}
