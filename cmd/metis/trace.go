package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xianxu/metis/pkg/record"
)

// readSet is the metis.trace read-sensor's reads.json output: the project root the
// paths are relative to, the first-party code+config read-set D, and whether the
// step touched site-packages (→ the uv.lock digest folds into Code.Deps, not D).
type readSet struct {
	ProjectRoot      string   `json:"project_root"`
	Reads            []string `json:"reads"`
	UsedSitePackages bool     `json:"used_site_packages"`
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

// buildD turns the sensor's first-party read paths into the read-set D =
// [(path, blob-hash)], hashing each via the injected hasher (the runner closes it
// over a batch-computed gitBlobHashes map). Pure over the hasher — unit-tested with
// no git.
func buildD(reads []string, blobHash func(path string) (record.Hash, error)) ([]record.CodeRef, error) {
	d := make([]record.CodeRef, 0, len(reads))
	for _, p := range reads {
		h, err := blobHash(p)
		if err != nil {
			return nil, fmt.Errorf("blob-hash %s: %w", p, err)
		}
		d = append(d, record.CodeRef{Path: p, BlobHash: h})
	}
	return d, nil
}
