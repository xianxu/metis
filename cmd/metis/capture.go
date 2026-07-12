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

	"github.com/xianxu/metis/pkg/record"
)

// captureClosure is the metis#8 side-ref dirty-code capture (the "git owns code"
// durability). Given a sweep's first-party code closure (paths relative to the git
// root), it writes each file's working-tree blob into git's object DB (`hash-object
// -w`) and returns the `(path, git-blob-hash)` pointer-manifest — git's blob-hash IS
// the content-hash, so metis stores no code bytes. If ANY closure file is dirty or
// untracked, it commits the closure to the side ref `ref` (e.g. `refs/metis/sweeps/<id>`
// for a sweep, `refs/metis/runs/<id>` for a single run — parented on HEAD, so `main` stays
// clean and GC can't reap the blobs) and returns that commit as the durable code SHA; a
// clean closure returns HEAD (no ref). Recovery of even a dirty version = `git checkout
// <commit>` / `git cat-file blob <hash>`.
func captureClosure(root string, closure []string, ref string) (commit string, manifest []record.CodeRef, err error) {
	paths := append([]string(nil), closure...)
	sort.Strings(paths)

	// hash-object -w every closure file → the pointer-manifest (and the blob is now in
	// the object DB, GC-protected once the ref below points at a commit containing it).
	dirty := false
	for _, p := range paths {
		h, err := gitOut(root, "hash-object", "-w", "--", p)
		if err != nil {
			return "", nil, err
		}
		manifest = append(manifest, record.CodeRef{Repo: root, Path: p, BlobHash: record.Hash(h)})
		d, err := isPathDirty(root, p, h)
		if err != nil {
			return "", nil, err
		}
		dirty = dirty || d
	}

	head, err := gitOut(root, "rev-parse", "HEAD")
	if err != nil {
		return "", nil, err
	}
	if !dirty {
		return head, manifest, nil // clean closure → HEAD is already the real SHA
	}

	// Build a tree = HEAD's tree with the dirty closure blobs overlaid, via a throwaway
	// index (so the real index/working tree are untouched), then commit it on a side ref.
	tmpIndex := filepath.Join(os.TempDir(), fmt.Sprintf("metis-capture-index-%s", strings.ReplaceAll(ref, "/", "_")))
	defer os.Remove(tmpIndex)
	env := append(os.Environ(), "GIT_INDEX_FILE="+tmpIndex)
	if _, err := gitOutEnv(root, env, "read-tree", head); err != nil {
		return "", nil, err
	}
	for _, cr := range manifest {
		if _, err := gitOutEnv(root, env, "update-index", "--add", "--cacheinfo", "100644,"+string(cr.BlobHash)+","+cr.Path); err != nil {
			return "", nil, err
		}
	}
	tree, err := gitOutEnv(root, env, "write-tree")
	if err != nil {
		return "", nil, err
	}
	commit, err = gitOut(root, "commit-tree", tree, "-p", head, "-m", "metis: code capture "+ref)
	if err != nil {
		return "", nil, err
	}
	if _, err := gitOut(root, "update-ref", ref, commit); err != nil {
		return "", nil, err
	}
	return commit, manifest, nil
}

// captureRunCode snapshots a run's first-party code closure PLUS its run-spec `.md`
// (metis#14) to git side-refs and returns the repo-qualified read-set D, the primary
// repo's durable commit SHA, and a capture STATUS. `ref` is the side-ref name (e.g.
// `refs/metis/runs/<id>` for a single run, `refs/metis/sweeps/<id>` for a sweep). Each
// repo's dirty closure lands on its own `ref` (metis#11: a closure can span metis + a
// consumer repo). BEST-EFFORT by design — never errors; a gap becomes a non-"captured"
// status (loud at the call site), never a silent success:
//   - "captured": every closure repo was a git work-tree and got a recoverable SHA;
//   - "degraded": there was a closure but capture couldn't fully run (no git, a
//     non-work-tree root, a git failure) — the run is NOT reproducible from a code SHA;
//   - "none": no first-party closure to capture (e.g. no-sensor steps + spec not in git).
func captureRunCode(closureByRepo map[string][]string, primaryRoot, specPath, ref string) (commit string, d []record.CodeRef, status string) {
	addSpecToClosure(closureByRepo, specPath) // the run-spec is a first-party input the trace never sees
	total := 0
	for _, ps := range closureByRepo {
		total += len(ps)
	}
	if total == 0 {
		return "", nil, "none"
	}
	if _, err := exec.LookPath("git"); err != nil {
		return "", nil, "degraded"
	}
	repos := make([]string, 0, len(closureByRepo))
	for r := range closureByRepo {
		repos = append(repos, r)
	}
	sort.Strings(repos)
	commits := map[string]string{}
	captured, skipped := 0, 0
	for _, repo := range repos {
		if _, err := gitOut(repo, "rev-parse", "--is-inside-work-tree"); err != nil {
			skipped++ // that root isn't a git work-tree — a real reproducibility gap
			continue
		}
		c, manifest, err := captureClosure(repo, closureByRepo[repo], ref)
		if err != nil {
			skipped++
			continue
		}
		d = append(d, manifest...)
		commits[repo] = c
		captured++
	}
	if captured == 0 {
		return "", nil, "degraded"
	}
	commit = commits[primaryRoot]
	if commit == "" {
		for _, repo := range repos {
			if c, ok := commits[repo]; ok {
				commit = c
				break
			}
		}
	}
	status = "captured"
	if skipped > 0 {
		status = "degraded" // some closure repo couldn't be captured — partial durability
	}
	return commit, d, status
}

// captureSweepCode captures the sweep's code closure + spec ONCE (per-shape-run: the
// closure is the same across points) to `refs/metis/sweeps/<shapeRunID>` and backfills
// every point-record's CodeManifest with the D + commit + capture status.
func captureSweepCode(o runOpts, man sweepManifest) error {
	closureByRepo := sweepClosure(o.expPath, man)
	primary := cacheProjectRoot(o.stepPath, filepath.Dir(o.expPath))
	commit, d, status := captureRunCode(closureByRepo, primary, o.expPath, "refs/metis/sweeps/"+man.ShapeRunID)
	warnOnUncaptured(o.out, status, "sweep "+man.ShapeRunID)
	for _, p := range man.Points {
		if err := backfillCodeManifest(o.expPath, p.RunID, d, commit, status); err != nil {
			return err
		}
	}
	return nil
}

// captureSingleRun captures ONE run's code closure + spec to `refs/metis/runs/<runID>`
// and backfills its record — the metis#14 single-run path (the sweep loop suppresses this
// via runOpts.inSweep, capturing once per shape-run instead of per point).
func captureSingleRun(o runOpts, runID string) error {
	runDir := filepath.Join(filepath.Dir(o.expPath), "runs", runID)
	closureByRepo := closureFromRunDir(runDir)
	primary := cacheProjectRoot(o.stepPath, filepath.Dir(o.expPath))
	commit, d, status := captureRunCode(closureByRepo, primary, o.expPath, "refs/metis/runs/"+runID)
	warnOnUncaptured(o.out, status, "run "+runID)
	return backfillCodeManifest(o.expPath, runID, d, commit, status)
}

// warnOnUncaptured emits a LOUD one-line note when a run/sweep could not be durably
// captured — reproducibility is a promise; a broken one must be visible, not silent (#14).
func warnOnUncaptured(out io.Writer, status, what string) {
	if out == nil {
		return
	}
	switch status {
	case "degraded":
		fmt.Fprintf(out, "metis: warning: code capture DEGRADED for %s — the run is not reproducible from a code SHA (no git work-tree, or a git failure)\n", what)
	case "none":
		fmt.Fprintf(out, "metis: note: no first-party code closure captured for %s (steps read no traced code) — not code-SHA reproducible\n", what)
	}
}

// closureFromRunDir collects, PER REPO ROOT, the union of first-party read paths across a
// single run's step reads.json (metis#11 multi-root: grouped by their repo so each is
// captured in the right repo). Root keys are symlink-resolved for a stable identity.
func closureFromRunDir(runDir string) map[string][]string {
	sets := map[string]map[string]bool{}
	stepDirs, _ := filepath.Glob(filepath.Join(runDir, "*"))
	for _, sd := range stepDirs {
		rs, err := loadReadSet(sd)
		if err != nil {
			continue
		}
		for repo, paths := range rs.Roots {
			key := resolveRoot(repo)
			if sets[key] == nil {
				sets[key] = map[string]bool{}
			}
			for _, r := range paths {
				sets[key][r] = true
			}
		}
	}
	return sortedSets(sets)
}

// sweepClosure is the union of every sweep point's run closure (one code closure across
// the shape-run — the same code decides all points).
func sweepClosure(expPath string, man sweepManifest) map[string][]string {
	dir := filepath.Dir(expPath)
	sets := map[string]map[string]bool{}
	for _, p := range man.Points {
		for repo, paths := range closureFromRunDir(filepath.Join(dir, "runs", p.RunID)) {
			if sets[repo] == nil {
				sets[repo] = map[string]bool{}
			}
			for _, r := range paths {
				sets[repo][r] = true
			}
		}
	}
	return sortedSets(sets)
}

// addSpecToClosure adds the experiment `.md` (a first-party input the Python read-set
// never sees — the Go runner parses it) to its own repo's closure, so its bytes are
// hashed + captured. Best-effort: a spec outside a git repo is skipped. Merges into an
// existing same-repo key (symlink-resolved) so the spec lands beside the code, not a dupe.
func addSpecToClosure(closureByRepo map[string][]string, specPath string) {
	abs, err := filepath.Abs(specPath)
	if err != nil {
		return
	}
	if _, err := os.Stat(abs); err != nil {
		return // no spec file on disk — best-effort skip (don't abort the code closure)
	}
	// Symlink-resolve the spec path so Rel against git's (realpath) toplevel is clean —
	// git returns /private/var/... while filepath.Abs keeps the /var symlink on macOS, and
	// a mixed-form Rel yields a broken ../.. path that git hash-object rejects.
	if r, err := filepath.EvalSymlinks(abs); err == nil {
		abs = r
	}
	root, err := gitOut(filepath.Dir(abs), "rev-parse", "--show-toplevel")
	if err != nil {
		return // spec not in a git repo (or no git) — best-effort skip
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return
	}
	key := resolveRoot(root)
	for existing := range closureByRepo {
		if resolveRoot(existing) == key {
			key = existing // reuse the code's repo key
			break
		}
	}
	for _, p := range closureByRepo[key] {
		if p == rel {
			return // already present
		}
	}
	closureByRepo[key] = append(closureByRepo[key], rel)
}

// shapeBlobHash returns the shape .md's git blob-hash — its content identity, computed
// PRE-run so PointAddress can content-address the intent (metis#27). Reuses gitBlobHashes
// over the spec's repo-relative path (symlink-resolved like addSpecToClosure, so Rel
// against git's realpath toplevel is clean). Returns an error when the spec isn't in a git
// work-tree — the caller falls back (a no-git run keeps a timestamp dir + an empty
// shape-blob in its address).
func shapeBlobHash(specPath string) (string, error) {
	abs, err := filepath.Abs(specPath)
	if err != nil {
		return "", err
	}
	if r, err := filepath.EvalSymlinks(abs); err == nil {
		abs = r
	}
	root, err := gitOut(filepath.Dir(abs), "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}
	hashes, err := gitBlobHashes(root, []string{rel})
	if err != nil {
		return "", err
	}
	return string(hashes[rel]), nil
}

// resolveRoot symlink-resolves a repo root for a stable map key (macOS /var → /private/var).
func resolveRoot(root string) string {
	if r, err := filepath.EvalSymlinks(root); err == nil {
		return r
	}
	return root
}

// sortedSets flattens a set-of-paths map into sorted slices.
func sortedSets(sets map[string]map[string]bool) map[string][]string {
	out := make(map[string][]string, len(sets))
	for repo, set := range sets {
		ps := make([]string, 0, len(set))
		for p := range set {
			ps = append(ps, p)
		}
		sort.Strings(ps)
		out[repo] = ps
	}
	return out
}

// backfillCodeManifest updates a run/point record's CodeManifest with the captured code
// closure (D) + the durable commit SHA + the capture status — the record's #3 slots #2/#8
// fill. A "captured" commit overrides the coarse HEAD SHA; a non-captured status is
// recorded honestly (so the record itself carries the reproducibility gap, #14).
func backfillCodeManifest(expPath, runID string, d []record.CodeRef, commit, status string) error {
	recPath := filepath.Join(filepath.Dir(expPath), "runs", runID, "record.json")
	b, err := os.ReadFile(recPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var rec record.RunRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return err
	}
	for i := range rec.Steps {
		rec.Steps[i].Code.D = d
		if commit != "" {
			rec.Steps[i].Code.Commit = commit
		}
		rec.Steps[i].Code.CaptureStatus = status
	}
	// The realized code identity (metis#27): the ONE post-capture site where the run's D
	// closure exists + the record is re-written. Two runs of the same config with different
	// code get different fingerprints → the ledger keeps both. (buildRecord runs before D
	// exists, so it can't set this.) A hash error (non-canonicalizable closure — unreachable
	// for a string manifest) is surfaced rather than silently dropped.
	fp, err := record.CodeFingerprint(d)
	if err != nil {
		return err
	}
	rec.CodeFingerprint = fp
	return writeRecordJSON(filepath.Join(filepath.Dir(expPath), "runs", runID), rec)
}

// isPathDirty reports whether the working-tree blob (workHash) differs from HEAD's
// version of the path — i.e. the file is edited or untracked (no HEAD version).
func isPathDirty(root, path, workHash string) (bool, error) {
	headHash, err := gitOut(root, "rev-parse", "HEAD:"+path)
	if err != nil {
		// Not in HEAD (untracked / new) → dirty. (rev-parse fails for an unknown path.)
		return true, nil
	}
	return headHash != workHash, nil
}

// gitOutEnv is gitOut with extra env (for GIT_INDEX_FILE during the capture tree build).
func gitOutEnv(dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}
