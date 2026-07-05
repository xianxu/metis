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
    upstream output-hashes in K_pre.

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

# project root = parent of the metis package (…/metis/trace.py → …/)
_PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
_EXCLUDE_PARTS = (".venv", "site-packages", "__pycache__", ".git")

_reads: set[str] = set()
_used_site_packages = False


def _classify(path: str) -> None:
    """Record a read path: first-party → into D; site-packages → the deps flag."""
    global _used_site_packages
    if not isinstance(path, str) or not path:
        return
    ap = os.path.abspath(path)
    if "site-packages" in ap or ".venv" in ap:
        _used_site_packages = True
        return
    if not ap.startswith(_PROJECT_ROOT + os.sep):
        return  # stdlib / system / temp / another repo → not first-party
    rel = os.path.relpath(ap, _PROJECT_ROOT)
    if any(part in _EXCLUDE_PARTS for part in rel.split(os.sep)):
        return
    # skip the run dir (a step's own outputs + upstream artifacts live under runs/)
    run_dir = os.environ.get("METIS_RUN_DIR")
    if run_dir and ap.startswith(os.path.abspath(run_dir) + os.sep):
        return
    _reads.add(rel)


def _audit(event: str, args) -> None:
    if event == "open" and args:
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
        "project_root": _PROJECT_ROOT,
        "reads": sorted(_reads),
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
