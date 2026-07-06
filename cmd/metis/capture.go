package main

import (
	"encoding/json"
	"fmt"
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
// untracked, it commits the closure to a side ref `refs/metis/sweeps/<shapeRunID>`
// (parented on HEAD, so `main` stays clean and GC can't reap the blobs) and returns
// that commit as the durable code SHA; a clean closure returns HEAD (no ref). Recovery
// of even a dirty version = `git checkout <commit>` / `git cat-file blob <hash>`.
func captureClosure(root string, closure []string, shapeRunID string) (commit string, manifest []record.CodeRef, err error) {
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
	tmpIndex := filepath.Join(os.TempDir(), fmt.Sprintf("metis-capture-index-%s", strings.ReplaceAll(shapeRunID, "/", "_")))
	defer os.Remove(tmpIndex)
	env := append(os.Environ(), "GIT_INDEX_FILE="+tmpIndex)
	if _, err := gitOutEnv(root, env, "read-tree", head); err != nil {
		return "", nil, err
	}
	for _, ref := range manifest {
		if _, err := gitOutEnv(root, env, "update-index", "--add", "--cacheinfo", "100644,"+string(ref.BlobHash)+","+ref.Path); err != nil {
			return "", nil, err
		}
	}
	tree, err := gitOutEnv(root, env, "write-tree")
	if err != nil {
		return "", nil, err
	}
	commit, err = gitOut(root, "commit-tree", tree, "-p", head, "-m", "metis: sweep code capture "+shapeRunID)
	if err != nil {
		return "", nil, err
	}
	if _, err := gitOut(root, "update-ref", "refs/metis/sweeps/"+shapeRunID, commit); err != nil {
		return "", nil, err
	}
	return commit, manifest, nil
}

// captureSweepCode captures the sweep's first-party code closure once (per-shape-run
// granularity — the closure is the same code across the sweep's points) and backfills
// each point-record's CodeManifest with the (path, blob-hash) manifest + the captured
// commit SHA. Best-effort: with no git repo (or no closure) it's a no-op — the sweep's
// point-records already ran and remain valid, just without the durable code SHA.
func captureSweepCode(o runOpts, man sweepManifest) error {
	closureByRepo := sweepClosure(o.expPath, man)
	if len(closureByRepo) == 0 {
		return nil // no first-party code closure recorded (e.g. no-sensor test steps)
	}
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}
	// Capture EACH repo's dirty closure to its own side ref (metis#11: the closure can
	// span metis + a consumer repo). The record's D is the repo-qualified union.
	primary := cacheProjectRoot(o.stepPath, filepath.Dir(o.expPath))
	repos := make([]string, 0, len(closureByRepo))
	for r := range closureByRepo {
		repos = append(repos, r)
	}
	sort.Strings(repos)
	var union []record.CodeRef
	commits := map[string]string{}
	for _, repo := range repos {
		if _, err := gitOut(repo, "rev-parse", "--is-inside-work-tree"); err != nil {
			continue // that root isn't a git work-tree — best-effort skip
		}
		commit, manifest, err := captureClosure(repo, closureByRepo[repo], man.ShapeRunID)
		if err != nil {
			return err
		}
		union = append(union, manifest...)
		commits[repo] = commit
	}
	if len(union) == 0 {
		return nil // no captured repo was a git work-tree
	}
	// The record's single Commit is the primary (expPath) repo's; per-repo commits in the
	// record are a metis#14 refinement — the D here is already multi-root/repo-qualified.
	commit := commits[primary]
	if commit == "" {
		for _, repo := range repos {
			if c, ok := commits[repo]; ok {
				commit = c
				break
			}
		}
	}
	for _, p := range man.Points {
		if err := backfillCodeManifest(o.expPath, p.RunID, union, commit); err != nil {
			return err
		}
	}
	return nil
}

// sweepClosure collects, PER REPO ROOT, the union of first-party read paths across the
// sweep's points' step reads.json (metis#11 multi-root: the code files whose bytes decide
// the runs, grouped by their repo so each is captured/hashed in the right repo).
func sweepClosure(expPath string, man sweepManifest) map[string][]string {
	dir := filepath.Dir(expPath)
	sets := map[string]map[string]bool{}
	for _, p := range man.Points {
		stepDirs, _ := filepath.Glob(filepath.Join(dir, "runs", p.RunID, "*"))
		for _, sd := range stepDirs {
			rs, err := loadReadSet(sd)
			if err != nil {
				continue
			}
			for repo, paths := range rs.Roots {
				if sets[repo] == nil {
					sets[repo] = map[string]bool{}
				}
				for _, r := range paths {
					sets[repo][r] = true
				}
			}
		}
	}
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

// backfillCodeManifest updates a point-record's CodeManifest with the captured code
// closure (D) + the durable commit SHA — the record's #3 slots #2/#8 fill.
func backfillCodeManifest(expPath, runID string, d []record.CodeRef, commit string) error {
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
		rec.Steps[i].Code.Commit = commit
	}
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
