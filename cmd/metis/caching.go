package main

import (
	"encoding/json"
	"errors"
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

// sortedUpstream collects a step's upstream terms — one per resolved `need` present in
// the map — and returns them sorted (so a derived key is invariant to needs-declaration
// order). ONE collection primitive, two callers passing DIFFERENT maps by design (metis#24):
//   - the record-provenance assembler (buildRecord) passes upstream OUTPUT-hashes;
//   - the cachingExecutor's K_pre passes upstream K_pres (input identities).
// The wiring is identical; only the term's meaning differs — so after #24 the executor's
// key and the record's Upstream deliberately DIVERGE (input-addressed vs output-addressed).
func sortedUpstream(needs []string, terms map[string]record.Hash) []record.Hash {
	up := make([]record.Hash, 0, len(needs))
	for _, need := range needs {
		if h, ok := terms[need]; ok {
			up = append(up, h)
		}
	}
	sort.Slice(up, func(i, j int) bool { return up[i] < up[j] })
	return up
}

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
// Before running a step it computes K_pre (from the step's config, the run seed, and the
// upstream steps' K_pres — input identities, metis#24 — accumulated in topo order), looks
// up the cache index, and on a HIT (the stored transitive-D re-hashes clean) materializes
// the cached output from the CAS and SKIPS the subprocess. On a MISS it runs, records D
// (from the sensor's reads.json), folds the transitive-D snapshot, stores the output, and
// writes the index entry. Runner.Run executes in topo order, so a step's upstream K_pres
// and transitive-D closures are ready when reached.
//
// The two accumulators are the metis#24 mechanism: `kpres` makes the key input-addressed
// (invariant to upstream OUTPUT non-determinism); `transitiveD` restores the upstream-CODE
// invalidation that the dropped output-hash term used to carry (each step stores its own
// transitive closure of read-sets; isHit re-hashes it). Both are per-point (a fresh
// executor per point-run — run.go), populated in topo order and repopulated from the stored
// entry on a HIT so a downstream step in the same run still sees an upstream HIT's closure.
type cachingExecutor struct {
	inner    experiment.StepExecutor
	store    cas.Store
	indexDir string
	seed     int
	out      io.Writer

	kpres       map[string]cache.Hash        // step-id → K_pre (input identity), for downstream keys
	transitiveD map[string][]record.CodeRef  // step-id → transitive read-set closure snapshot
}

// newCachingExecutor: D paths are now repo-qualified (metis#11) — each ref carries its own
// repo root, so there's no single project-root to thread here; store + validate hash each
// ref in its own repo.
func newCachingExecutor(inner experiment.StepExecutor, store cas.Store, cacheDir string, seed int, out io.Writer) *cachingExecutor {
	return &cachingExecutor{
		inner:       inner,
		store:       store,
		indexDir:    filepath.Join(cacheDir, "index"),
		seed:        seed,
		out:         out,
		kpres:       map[string]cache.Hash{},
		transitiveD: map[string][]record.CodeRef{},
	}
}

func (c *cachingExecutor) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	kpre, err := c.kpre(step)
	if err != nil {
		return experiment.StepResult{}, err
	}
	// Remember this step's K_pre unconditionally (identical on HIT + MISS — it's the
	// input identity, computed here before the lookup), so a downstream step's key can
	// reference it. Set here (not in a HIT-only branch) so the MISS path can't forget it.
	c.kpres[step.ID] = kpre
	stepDir := filepath.Join(runDir, step.ID)

	entry, ok, err := c.lookup(kpre)
	if err != nil {
		return experiment.StepResult{}, err
	}
	if ok && c.isHit(step, entry) {
		res, err := c.materialize(entry, stepDir, runDir)
		switch {
		case err == nil:
			fmt.Fprintf(c.out, "⚡ step %s (cache hit)\n", step.ID)
			// Repopulate this step's transitive-D closure from the STORED snapshot so a
			// downstream step in the same run still folds an upstream HIT's closure into
			// its own snapshot (load-bearing — a dropped repopulation only surfaces one
			// edit later; see TestCachingExecutor_HitFeedsDownstreamClosure).
			c.transitiveD[step.ID] = entry.TransitiveD
			return res, nil
		case errors.Is(err, cas.ErrNotFound) || errors.Is(err, cas.ErrCorrupt):
			// The wipeable-cache contract (pkg/cas): a missing/corrupt/evicted output
			// blob is NOT a failure — the index hit but the bytes are gone (a partial
			// `rm -rf cas/`, corruption, or LRU eviction). Fall through to recompute.
			fmt.Fprintf(c.out, "… step %s (index hit but output wiped — recomputing)\n", step.ID)
		default:
			return experiment.StepResult{}, err // a real IO error — propagate
		}
	}

	// MISS (or a recoverable materialize failure) — run, then record D + fold the
	// transitive-D snapshot + store the output + write the index entry.
	res, err := c.inner.Execute(step, runDir)
	if err != nil {
		return res, err
	}
	if err := c.recordMiss(step, kpre, res, stepDir, runDir); err != nil {
		return res, err
	}
	return res, nil
}

// kpre builds the ex-ante cache key from the step's config, the run seed, and the
// upstream steps' K_pres accumulated so far (input-addressed, metis#24 — Kpre sorts them
// internally). A step reached in topo order has every upstream's K_pre in c.kpres.
func (c *cachingExecutor) kpre(step experiment.Step) (cache.Hash, error) {
	return cache.Kpre(record.StepRecord{
		StepID:   step.ID,
		Uses:     step.Uses,
		With:     step.With,
		Upstream: sortedUpstream(step.Needs, c.kpres),
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
// step re-hashes its stored TRANSITIVE-D closure via git and hits only if every file —
// its own AND every transitively-upstream read — is unchanged (metis#24: the input-
// addressed K_pre no longer carries upstream-code edits, so the closure snapshot does).
// A hasher failure is treated as a MISS (safe: recompute, never stale).
func (c *cachingExecutor) isHit(step experiment.Step, entry cache.Entry) bool {
	if isImmutableLeaf(step) {
		return true
	}
	// Migration guard (metis#24): the K_pre-term change orphans all non-root entries (an
	// implicit cache-version bump), but a surviving root entry — or any legacy entry —
	// carries no TransitiveD (nil, absent in its JSON). A #24 entry ALWAYS stores a non-nil
	// TransitiveD (MergeTransitiveD never returns nil). So a nil snapshot means "written by
	// pre-#24 code" → MISS, so a nil can never vacuously HIT and serve an unvalidated output.
	if entry.TransitiveD == nil {
		return false
	}
	// Re-hash the stored closure per-repo (metis#11): group the refs by their repo root, hash
	// each repo's paths in ITS repo (git -C repo), then validate. Symmetric with the store
	// side (recordMiss folds + stores the SAME closure) — store and HIT-check can't disagree.
	hashesByRepo, err := hashDByRepo(entry.TransitiveD)
	if err != nil {
		return false
	}
	return cache.Validate(entry.TransitiveD, func(ref record.CodeRef) (record.Hash, error) {
		h, ok := hashesByRepo[ref.Repo][ref.Path]
		if !ok {
			return "", fmt.Errorf("no hash for %s:%s", ref.Repo, ref.Path)
		}
		return h, nil
	})
}

// hashDByRepo groups D refs by repo root and git-blob-hashes each repo's paths in that
// repo — the shared per-repo hasher for both the store (recordMiss) and validate (isHit)
// sides. A ref with an empty repo root is a legacy (pre-#11) single-root entry: we reject
// it with an error (→ MISS) rather than hashing it. NOTE: `git -C "" hash-object` does NOT
// fail — `-C ""` is a no-op and git resolves the path against the CURRENT WORKING DIR, so
// relying on "git fails" would make the HIT/MISS cwd-dependent. The explicit guard keeps
// validation cwd-independent + symmetric with loadReadSet's loud rejection of legacy reads.json.
func hashDByRepo(d []record.CodeRef) (map[string]map[string]record.Hash, error) {
	byRepo := map[string][]string{}
	for _, ref := range d {
		if ref.Repo == "" {
			return nil, fmt.Errorf("legacy read-set entry (empty repo root) — treating as MISS; metis#11 requires a repo-qualified D")
		}
		byRepo[ref.Repo] = append(byRepo[ref.Repo], ref.Path)
	}
	out := make(map[string]map[string]record.Hash, len(byRepo))
	for repo, paths := range byRepo {
		h, err := gitBlobHashes(repo, paths)
		if err != nil {
			return nil, err
		}
		out[repo] = h
	}
	return out, nil
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
// build D from the sensor's reads.json, fold the transitive-D snapshot (this step's D
// unioned with each upstream's stored closure — the metis#24 soundness snapshot), CAS-store
// each artifact into an output manifest, and index the manifest hash under K_pre.
func (c *cachingExecutor) recordMiss(step experiment.Step, kpre cache.Hash, res experiment.StepResult, stepDir, runDir string) error {
	rs, err := loadReadSet(stepDir)
	if err != nil {
		return err
	}
	// Build the per-repo path set (metis#11: D can span metis + a consumer repo). Fold
	// uv.lock into each root that HAS one when the step touched site-packages, so a
	// dependency upgrade (a new uv.lock → its blob-hash moves) invalidates the cache —
	// otherwise a pandas/sklearn bump would false-HIT against the old deps.
	roots := map[string][]string{}
	for repo, paths := range rs.Roots {
		p := append([]string(nil), paths...)
		if rs.UsedSitePackages {
			if _, err := os.Stat(filepath.Join(repo, depsLockFile)); err == nil {
				p = append(p, depsLockFile)
			}
		}
		roots[repo] = p
	}
	// Hash each repo's paths in ITS repo — the SAME grouping isHit re-hashes against, so
	// store and a later HIT-check can never disagree on where a D path lives.
	hashesByRepo := map[string]map[string]record.Hash{}
	for repo, paths := range roots {
		h, err := gitBlobHashes(repo, paths)
		if err != nil {
			return err
		}
		hashesByRepo[repo] = h
	}
	d, err := buildD(roots, func(repo, path string) (record.Hash, error) {
		h, ok := hashesByRepo[repo][path]
		if !ok {
			return "", fmt.Errorf("no hash for %s:%s", repo, path)
		}
		return h, nil
	})
	if err != nil {
		return err
	}

	// Fold the transitive-D snapshot: this step's own D unioned with each upstream's
	// already-computed closure (topo order guarantees they're populated — set on their
	// own MISS or repopulated on their HIT). Stored in THIS step's entry so isHit
	// validates the closure against the current tree (restoring upstream-code invalidation
	// the input-addressed K_pre drops). MergeTransitiveD returns a non-nil slice even when
	// empty, so a #24 entry's TransitiveD is never nil (distinguishing it from a legacy one).
	upstream := make([][]record.CodeRef, 0, len(step.Needs))
	for _, need := range step.Needs {
		upstream = append(upstream, c.transitiveD[need])
	}
	td := cache.MergeTransitiveD(d, upstream...)
	c.transitiveD[step.ID] = td

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
	return c.writeEntry(cache.Entry{Kpre: kpre, D: d, TransitiveD: td, Output: outHash})
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
