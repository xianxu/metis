package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xianxu/metis/pkg/record"
)

// readSet is the metis.trace read-sensor's reads.json output (v2, metis#11): the
// first-party code+config read-set D grouped BY repo root (`roots`), so a consumer
// repo's code (kbench) is captured alongside metis's, and whether the step touched
// site-packages (→ the uv.lock digest folds into D per-root, below).
type readSet struct {
	Roots            map[string][]string `json:"roots"`
	UsedSitePackages bool                `json:"used_site_packages"`
}

// loadReadSet reads a step's reads.json (written by the Python sensor). An absent
// file is an empty read-set, not an error: a step that ran without the sensor is
// treated as having no first-party code deps (a K_pre-only cache entry), which is
// the safe direction (more likely to MISS, never a false HIT).
func loadReadSet(stepDir string) (readSet, error) {
	b, err := os.ReadFile(filepath.Join(stepDir, "reads.json"))
	if errors.Is(err, os.ErrNotExist) {
		return readSet{}, nil
	}
	if err != nil {
		return readSet{}, err
	}
	var rs readSet
	if err := json.Unmarshal(b, &rs); err != nil {
		return readSet{}, fmt.Errorf("parse reads.json: %w", err)
	}
	// Guard the lockstep/false-HIT direction (metis#11): a legacy v1 reads.json
	// ({project_root, reads}) silently unmarshals to a nil Roots → empty D → a vacuous
	// K_pre-only HIT (worse than the cross-repo miss we're fixing). Detect the format
	// mismatch and fail LOUD rather than serve a stale HIT. (An absent or genuinely-empty
	// reads.json is still a legitimate empty read-set — the safe K_pre-only entry.)
	if rs.Roots == nil {
		var legacy struct {
			ProjectRoot string   `json:"project_root"`
			Reads       []string `json:"reads"`
		}
		_ = json.Unmarshal(b, &legacy)
		if legacy.ProjectRoot != "" || len(legacy.Reads) > 0 {
			return readSet{}, fmt.Errorf("reads.json at %s is legacy v1 (project_root/reads); this metis expects v2 (roots) — wipe .metis-cache and re-run", stepDir)
		}
	}
	return rs, nil
}

// gitBlobHashes returns git's blob hash for each path (relative to dir) in ONE
// `git hash-object` call. git's blob-hash IS the content-hash (the record's content
// store is git); it hashes the working-tree bytes — no commit needed, so a dirty or
// untracked file still hashes and the cache validates correctly.
func gitBlobHashes(dir string, paths []string) (map[string]record.Hash, error) {
	if len(paths) == 0 {
		return map[string]record.Hash{}, nil
	}
	out, err := gitOut(dir, append([]string{"hash-object", "--"}, paths...)...)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != len(paths) {
		return nil, fmt.Errorf("git hash-object: got %d hashes for %d paths", len(lines), len(paths))
	}
	m := make(map[string]record.Hash, len(paths))
	for i, p := range paths {
		m[p] = record.Hash(lines[i])
	}
	return m, nil
}

// buildD turns the sensor's per-repo first-party read paths into the read-set D =
// [(repo, path, blob-hash)], repo-qualified and sorted (metis#11: D can span the metis
// repo AND a consumer repo). Each (repo, path) is hashed via the injected hasher (the
// runner closes it over per-repo batch-computed gitBlobHashes maps). Pure over the
// hasher — unit-tested with no git.
func buildD(roots map[string][]string, blobHash func(repo, path string) (record.Hash, error)) ([]record.CodeRef, error) {
	repos := make([]string, 0, len(roots))
	for r := range roots {
		repos = append(repos, r)
	}
	sort.Strings(repos)
	d := []record.CodeRef{}
	for _, repo := range repos {
		paths := append([]string(nil), roots[repo]...)
		sort.Strings(paths)
		for _, p := range paths {
			h, err := blobHash(repo, p)
			if err != nil {
				return nil, fmt.Errorf("blob-hash %s:%s: %w", repo, p, err)
			}
			d = append(d, record.CodeRef{Repo: repo, Path: p, BlobHash: h})
		}
	}
	return d, nil
}
