package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/xianxu/metis/pkg/cache"
	"github.com/xianxu/metis/pkg/cas"
	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/record"
)

// depsLockFile is the dependency lockfile folded into D when a step uses site-packages
// — a dep upgrade changes its git-blob-hash and invalidates the cache.
const depsLockFile = "uv.lock"

// outputManifest is a step's cached output: the metrics it emitted plus its artifact
// files as (runDir-relative path, CAS content-hash) pairs. Stored as a small JSON
// blob in the CAS; a HIT reads it back, rewrites the files from the CAS into the step
// dir (so downstream steps read them), and returns the metrics — no subprocess.
type outputManifest struct {
	Metrics   map[string]float64 `json:"metrics,omitempty"`
	Artifacts []record.FileHash  `json:"artifacts"`
}

// cachingExecutor decorates a StepExecutor with the metis#2 validating-trace cache.
// Before running a step it computes K_pre (from the step's config, the run seed, and
// the upstream steps' output-hashes accumulated in topo order), looks up the cache
// index, and on a HIT (the stored read-set D re-hashes clean) materializes the cached
// output from the CAS and SKIPS the subprocess. On a MISS it runs, records D (from the
// sensor's reads.json) + stores the output, and writes the index entry. Runner.Run
// executes in topo order, so a step's upstream output-hashes are ready when reached.
type cachingExecutor struct {
	inner       experiment.StepExecutor
	store       cas.Store
	indexDir    string
	projectRoot string // where D paths resolve + git hash-object runs (the metis code root)
	seed        int
	out         io.Writer

	outputs map[string]record.Hash // step-id → output-hash, accumulated across the run
}

func newCachingExecutor(inner experiment.StepExecutor, store cas.Store, cacheDir, projectRoot string, seed int, out io.Writer) *cachingExecutor {
	return &cachingExecutor{
		inner:       inner,
		store:       store,
		indexDir:    filepath.Join(cacheDir, "index"),
		projectRoot: projectRoot,
		seed:        seed,
		out:         out,
		outputs:     map[string]record.Hash{},
	}
}

func (c *cachingExecutor) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	kpre, err := c.kpre(step)
	if err != nil {
		return experiment.StepResult{}, err
	}
	stepDir := filepath.Join(runDir, step.ID)

	entry, ok, err := c.lookup(kpre)
	if err != nil {
		return experiment.StepResult{}, err
	}
	if ok && c.isHit(step, entry) {
		res, err := c.materialize(entry, stepDir, runDir)
		if err != nil {
			return experiment.StepResult{}, err
		}
		fmt.Fprintf(c.out, "⚡ step %s (cache hit)\n", step.ID)
		if err := c.recordOutput(step.ID, res.Artifacts, runDir); err != nil {
			return experiment.StepResult{}, err
		}
		return res, nil
	}

	// MISS — run, then record D + store the output + write the index entry.
	res, err := c.inner.Execute(step, runDir)
	if err != nil {
		return res, err
	}
	if err := c.recordMiss(kpre, res, stepDir, runDir); err != nil {
		return res, err
	}
	if err := c.recordOutput(step.ID, res.Artifacts, runDir); err != nil {
		return res, err
	}
	return res, nil
}

// kpre builds the ex-ante cache key from the step's config, the run seed, and the
// upstream steps' output-hashes accumulated so far (Kpre sorts them internally).
func (c *cachingExecutor) kpre(step experiment.Step) (cache.Hash, error) {
	upstream := make([]record.Hash, 0, len(step.Needs))
	for _, need := range step.Needs {
		if h, ok := c.outputs[need]; ok {
			upstream = append(upstream, h)
		}
	}
	return cache.Kpre(record.StepRecord{
		StepID:   step.ID,
		Uses:     step.Uses,
		With:     step.With,
		Upstream: upstream,
	}, c.seed)
}

func (c *cachingExecutor) lookup(kpre cache.Hash) (cache.Entry, bool, error) {
	b, err := os.ReadFile(filepath.Join(c.indexDir, string(kpre)+".json"))
	if os.IsNotExist(err) {
		return cache.Entry{}, false, nil
	}
	if err != nil {
		return cache.Entry{}, false, err
	}
	e, err := cache.DecodeEntry(b)
	if err != nil {
		return cache.Entry{}, false, err
	}
	return e, true, nil
}

// isHit decides whether an index entry is a HIT. An immutable-leaf step (a conscious
// pin that its external source is frozen) hits on the K_pre match alone; every other
// step re-hashes its stored read-set D via git and hits only if all files are
// unchanged. A hasher failure is treated as a MISS (safe: recompute, never stale).
func (c *cachingExecutor) isHit(step experiment.Step, entry cache.Entry) bool {
	if isImmutableLeaf(step) {
		return true
	}
	paths := make([]string, len(entry.D))
	for i, ref := range entry.D {
		paths[i] = ref.Path
	}
	hashes, err := gitBlobHashes(c.projectRoot, paths)
	if err != nil {
		return false
	}
	return cache.Validate(entry.D, func(p string) (record.Hash, error) {
		h, ok := hashes[p]
		if !ok {
			return "", fmt.Errorf("no hash for %s", p)
		}
		return h, nil
	})
}

// materialize reconstructs a cached step: fetch its output manifest from the CAS,
// rewrite each artifact from the CAS into the run dir (so downstream steps read them),
// and return the metrics + artifact paths — no subprocess.
func (c *cachingExecutor) materialize(entry cache.Entry, stepDir, runDir string) (experiment.StepResult, error) {
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		return experiment.StepResult{}, err
	}
	mb, err := c.store.Get(entry.Output)
	if err != nil {
		return experiment.StepResult{}, fmt.Errorf("cache: get output manifest: %w", err)
	}
	var man outputManifest
	if err := json.Unmarshal(mb, &man); err != nil {
		return experiment.StepResult{}, fmt.Errorf("cache: parse output manifest: %w", err)
	}
	arts := make([]string, 0, len(man.Artifacts))
	for _, a := range man.Artifacts {
		b, err := c.store.Get(a.Hash)
		if err != nil {
			return experiment.StepResult{}, fmt.Errorf("cache: get artifact %s: %w", a.Path, err)
		}
		dst := filepath.Join(runDir, filepath.FromSlash(a.Path))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return experiment.StepResult{}, err
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			return experiment.StepResult{}, err
		}
		arts = append(arts, a.Path)
	}
	return experiment.StepResult{Metrics: man.Metrics, Artifacts: arts}, nil
}

// recordMiss stores a freshly-run step's output in the CAS + writes the index entry:
// build D from the sensor's reads.json, CAS-store each artifact into an output
// manifest, and index the manifest hash under K_pre.
func (c *cachingExecutor) recordMiss(kpre cache.Hash, res experiment.StepResult, stepDir, runDir string) error {
	rs, err := loadReadSet(stepDir)
	if err != nil {
		return err
	}
	root := rs.ProjectRoot
	if root == "" {
		root = c.projectRoot
	}
	// Fold uv.lock into D when the step touched site-packages, so a dependency
	// upgrade (a new uv.lock) invalidates the cache — otherwise a pandas/sklearn bump
	// would false-HIT and serve output computed against the old deps.
	paths := rs.Reads
	if rs.UsedSitePackages {
		if _, err := os.Stat(filepath.Join(root, depsLockFile)); err == nil {
			paths = append(append([]string(nil), rs.Reads...), depsLockFile)
		}
	}
	hashes, err := gitBlobHashes(root, paths)
	if err != nil {
		return err
	}
	d, err := buildD(paths, func(p string) (record.Hash, error) {
		h, ok := hashes[p]
		if !ok {
			return "", fmt.Errorf("no hash for %s", p)
		}
		return h, nil
	})
	if err != nil {
		return err
	}

	man := outputManifest{Metrics: res.Metrics}
	for _, rel := range res.Artifacts {
		b, err := os.ReadFile(filepath.Join(runDir, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("cache: read artifact %s: %w", rel, err)
		}
		h, err := c.store.Put(b)
		if err != nil {
			return err
		}
		man.Artifacts = append(man.Artifacts, record.FileHash{Path: rel, Hash: h})
	}
	sort.Slice(man.Artifacts, func(i, j int) bool { return man.Artifacts[i].Path < man.Artifacts[j].Path })
	mb, err := json.Marshal(man)
	if err != nil {
		return err
	}
	outHash, err := c.store.Put(mb)
	if err != nil {
		return err
	}
	return c.writeEntry(cache.Entry{Kpre: kpre, D: d, Output: outHash})
}

func (c *cachingExecutor) writeEntry(e cache.Entry) error {
	if err := os.MkdirAll(c.indexDir, 0o755); err != nil {
		return err
	}
	b, err := cache.EncodeEntry(e)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.indexDir, string(e.Kpre)+".json"), b, 0o644)
}

// recordOutput computes and remembers a step's output-hash (from its artifacts) so
// downstream steps' K_pre can reference it — the same OutputHash used in the record.
func (c *cachingExecutor) recordOutput(stepID string, artifacts []string, runDir string) error {
	fhs, err := hashArtifacts(runDir, artifacts)
	if err != nil {
		return err
	}
	c.outputs[stepID] = record.OutputHash(fhs)
	return nil
}

// isImmutableLeaf reports whether a step is marked as a pinned external leaf
// (`with: {cache: {leaf: immutable}}`) — cached on the K_pre match alone (fetch once,
// don't re-observe), the v1 external-fetch policy (metis#2's leaf policy). A conscious
// soundness bet the author makes for an impure fetch whose source is frozen/versioned.
func isImmutableLeaf(step experiment.Step) bool {
	c, ok := step.With["cache"].(map[string]any)
	if !ok {
		return false
	}
	return c["leaf"] == "immutable"
}
