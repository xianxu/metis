"""metis.trace — the read-sensor launcher for metis#2 step caching.

`python -m metis.trace <step-module>` installs a `sys.addaudithook` that records
file `open`s, runs the step module as `__main__`, and on exit snapshots
`sys.modules` for the first-party code closure — writing the read-set **D** to
`reads.json` in the step dir. The Go runner (cmd/metis) turns those paths into
`(path, git-blob-hash)` pairs for the cache key; re-hashing them on a later run is
what decides HIT vs MISS (an edited code file → its hash moves → MISS).

D is **first-party code + config under the project root only**:
  - keep: project files that are not under the venv / site-packages / __pycache__
    / .git / the run dir (step outputs);
  - collapse any site-packages read → a single `used_site_packages` flag (the Go
    side folds the uv.lock digest into Code.Deps, not D);
  - upstream artifacts + the dataset are NOT in D — they are class-1 keyed via the
    upstream output-hashes in K_pre. Crucially, everything under METIS_RUN_DIR (a
    step's own outputs + the upstream artifacts it reads) is excluded — else those
    change every run and every step would MISS forever.

Only reads are recorded (write-mode opens are filtered). The sensor's own module
(`metis.trace`, `metis.__init__`) appears in the sys.modules snapshot, so it lands in
every step's D — editing the sensor cold-busts the whole pool (over-invalidation,
safe direction).

Honest limit (design): `sys.addaudithook` sees Python-level `open`/`import`; a
C-extension `fopen` (pandas/pyarrow parquet reads) bypasses it — but those are the
class-1 *data* reads, not first-party code. The *code* closure is captured robustly
via the `sys.modules` snapshot regardless of import caching. The airtight version is
a syscall-trace sensor swap; D's definition is unchanged.
"""

from __future__ import annotations

import json
import os
import runpy
import sys
import sysconfig

_EXCLUDE_PARTS = (".venv", "site-packages", "__pycache__", ".git")


def _stdlib_prefixes() -> tuple[str, ...]:
    """Absolute dir prefixes of the Python install / stdlib. Reads under these are NOT
    first-party even when they happen to sit under a git-tracked HOME (e.g. a uv-managed
    interpreter at ~/.local/share/uv/python/…, when ~ is a repo) — the multi-root walk
    would otherwise mis-root the whole stdlib. site-packages/.venv are handled separately
    (they set the deps flag); this is the stdlib/install itself."""
    cands = {sys.base_prefix, sys.prefix, os.path.dirname(os.__file__)}
    for key in ("stdlib", "platstdlib"):
        p = sysconfig.get_paths().get(key)
        if p:
            cands.add(p)
    return tuple(sorted(os.path.abspath(p) + os.sep for p in cands if p))


_STDLIB_PREFIXES = _stdlib_prefixes()

# _roots maps a first-party repo root → the set of repo-relative code paths read from it
# (metis#11: multi-root — a step's closure can span the metis repo AND a consumer repo like
# kbench). _root_cache memoizes the walk-up from a dir to its containing repo root.
_roots: dict[str, set[str]] = {}
_root_cache: dict[str, "str | None"] = {}
_used_site_packages = False


def _repo_root(path: str) -> "str | None":
    """The nearest ancestor dir of `path` containing a `.git` marker — a DIR (normal repo)
    OR a FILE (a linked worktree / submodule uses a `.git` file). None if under
    site-packages/.venv or no repo is found (stdlib / temp). Cached per-directory."""
    ap = os.path.abspath(path)
    if "site-packages" in ap or ".venv" in ap:
        return None
    chain: list[str] = []
    cur = os.path.dirname(ap)
    root: "str | None" = None
    while True:
        if cur in _root_cache:
            root = _root_cache[cur]
            break
        chain.append(cur)
        if os.path.exists(os.path.join(cur, ".git")):
            root = cur
            break
        parent = os.path.dirname(cur)
        if parent == cur:  # filesystem root, no repo
            root = None
            break
        cur = parent
    for c in chain:  # every dir between `path` and its root shares that root
        _root_cache[c] = root
    return root


def _classify(path: str) -> None:
    """Record a read path: first-party → into D under its repo root; site-packages → the
    deps flag; stdlib/temp/another-non-repo → dropped."""
    global _used_site_packages
    if not isinstance(path, str) or not path:
        return
    ap = os.path.abspath(path)
    if "site-packages" in ap or ".venv" in ap:
        _used_site_packages = True
        return
    if ap.startswith(_STDLIB_PREFIXES):
        return  # stdlib / Python install — not first-party (even under a git-repo HOME)
    root = _repo_root(ap)
    if root is None:
        return  # stdlib / system / temp / not under any repo → not first-party
    rel = os.path.relpath(ap, root)
    if any(part in _EXCLUDE_PARTS for part in rel.split(os.sep)):
        return
    # skip the run dir (a step's own outputs + upstream artifacts live under runs/)
    run_dir = os.environ.get("METIS_RUN_DIR")
    if run_dir and ap.startswith(os.path.abspath(run_dir) + os.sep):
        return
    _roots.setdefault(root, set()).add(rel)


def _audit(event: str, args) -> None:
    if event != "open" or not args:
        return
    # Reads only: the "open" audit event fires for writes too, but a write is not an
    # input. Skip write modes ('w'/'a'/'x'/'+') and O_WRONLY/O_RDWR opens. (Metis
    # writes land under the excluded run dir anyway, but filter here for correctness.)
    mode = args[1] if len(args) > 1 else None
    if isinstance(mode, str) and any(c in mode for c in "wax+"):
        return
    flags = args[2] if len(args) > 2 else 0
    if isinstance(flags, int) and (flags & (os.O_WRONLY | os.O_RDWR)):
        return
    _classify(args[0])


def _snapshot_modules() -> None:
    """Fold the loaded first-party module files into D — robust to import caching
    (a module imported before the hook installed still shows up here)."""
    for mod in list(sys.modules.values()):
        f = getattr(mod, "__file__", None)
        if isinstance(f, str) and f:
            _classify(f)


def _write_reads() -> None:
    _snapshot_modules()
    step_dir = os.environ.get("METIS_STEP_DIR", os.getcwd())
    payload = {
        # v2 (metis#11): first-party code grouped by repo root, so a consumer repo's code
        # is captured alongside metis's. The Go side hashes each root's paths in that repo.
        "roots": {root: sorted(paths) for root, paths in sorted(_roots.items())},
        "used_site_packages": _used_site_packages,
    }
    with open(os.path.join(step_dir, "reads.json"), "w") as fh:
        json.dump(payload, fh, indent=2)


def main() -> None:
    if len(sys.argv) < 2:
        print("usage: python -m metis.trace <module> [args...]", file=sys.stderr)
        raise SystemExit(2)
    target = sys.argv[1]
    sys.argv = sys.argv[1:]  # present the target's own argv to it
    sys.addaudithook(_audit)
    try:
        runpy.run_module(target, run_name="__main__", alter_sys=True)
    finally:
        _write_reads()  # even on a step error — record what it read before failing


if __name__ == "__main__":
    main()
