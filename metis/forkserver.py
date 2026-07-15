"""metis.forkserver — the warm leaf fork-server (metis#44).

`python -m metis.forkserver` pre-imports the heavy third-party libraries once, then serves
step-execution requests as JSON lines on stdin. Per request it **forks** (main thread only):
the child scrubs `METIS_*` from its env, applies the request's env verbatim, chdirs to the
step dir, redirects stdout+stderr onto a per-request pipe, and runs the step through the SAME
`metis.trace.run_traced` machinery as the legacy `python -m metis.trace <mod>` subprocess —
then `os._exit`s. Every step therefore keeps today's per-process semantics (crash isolation,
its own reads.json, an authoritative env) while skipping the ~1s interpreter+import tax that
metis#44 measured on every one of a sweep's ~5k leaf spawns.

Protocol (JSONL over the server's stdin/stdout; stderr is free-form diagnostics):
  →  {"id": 7, "module": "kbench.titanic.features", "cwd": "<stepDir>", "env": {"METIS_*": …}}
  ←  {"ready": true}                        (once, after preload — requests may be sent before)
  ←  {"id": 7, "exit": 0, "output": "<combined child stdout+stderr, tail-capped>"}
stdin EOF ⇒ drain in-flight children, exit 0.

Design constraints (the metis#44 plan pins these):
  - Preload THIRD-PARTY ONLY. The cache read-set D is derived from the child's sys.modules
    snapshot (first-party filter): preloaded first-party would widen every step's D
    (over-invalidation), a delta-rule would under-capture it (stale cache hits). Third-party
    is excluded from D by design, so warming it is free. Override the preload list with
    `METIS_FORKSERVER_PRELOAD` (comma-separated; empty string = no preload — tests).
  - Only the MAIN thread forks; waiter threads do IO only. The server never executes a
    BLAS/OpenMP region (imports only), so each child initializes its own threadpools.
  - The child forces `used_site_packages` (see run_traced) — it inherits the imports and
    would never observe the site-packages reads that normally set the flag.
POSIX-only by construction (os.fork).
"""

from __future__ import annotations

import importlib
import json
import os
import sys
import threading
import traceback

_DEFAULT_PRELOAD = "numpy,pandas,sklearn,pyarrow"
_OUTPUT_CAP = 200_000  # keep the tail; matches CombinedOutput's role (error context only)


def _preload() -> list[str]:
    """Import the heavy libraries the leaves share. A missing one is fine (a layer's venv
    may not carry it) — the child pays its own import then, exactly like today."""
    loaded = []
    names = os.environ.get("METIS_FORKSERVER_PRELOAD", _DEFAULT_PRELOAD)
    for name in filter(None, (n.strip() for n in names.split(","))):
        try:
            importlib.import_module(name)
            loaded.append(name)
        except ImportError:
            print(f"forkserver: preload {name!r} not importable (skipped)", file=sys.stderr)
    return loaded


def _child(req: dict) -> "None":  # never returns — os._exit
    code = 1
    try:
        # The request env is AUTHORITATIVE for METIS_*: scrub first so an absent key (e.g.
        # READ_ROOT on an unconfined run) is genuinely absent, mirroring exec.go's strip.
        for k in [k for k in os.environ if k.startswith("METIS_")]:
            del os.environ[k]
        os.environ.update(req.get("env") or {})
        os.chdir(req["cwd"])
        from metis.trace import run_traced  # already imported in the server; cheap here

        run_traced(req["module"], force_site_packages=True)
        code = 0
    except SystemExit as e:  # a step's deliberate exit code passes through
        c = e.code
        code = c if isinstance(c, int) else (0 if c is None else 1)
    except BaseException:
        traceback.print_exc()
        code = 1
    finally:
        try:
            sys.stdout.flush()
            sys.stderr.flush()
        except Exception:
            pass
        os._exit(code)


def _wait(req_id, pid: int, rfd: int, lock: threading.Lock, out) -> None:
    """Waiter thread: drain the child's combined-output pipe to EOF, reap it, emit the
    response line. IO only — never forks, never touches os.environ."""
    chunks = []
    with os.fdopen(rfd, "rb") as fh:
        while True:
            b = fh.read(65536)
            if not b:
                break
            chunks.append(b)
    _, status = os.waitpid(pid, 0)
    exit_code = os.waitstatus_to_exitcode(status)
    output = b"".join(chunks).decode("utf-8", "replace")
    if len(output) > _OUTPUT_CAP:
        output = output[-_OUTPUT_CAP:]
    with lock:
        out.write(json.dumps({"id": req_id, "exit": exit_code, "output": output}) + "\n")
        out.flush()


def serve() -> None:
    out = sys.stdout
    lock = threading.Lock()
    loaded = _preload()
    with lock:
        out.write(json.dumps({"ready": True, "preloaded": loaded}) + "\n")
        out.flush()
    waiters: list[threading.Thread] = []
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        req = json.loads(line)  # a malformed line is a protocol bug — fail loud, not skip
        r, w = os.pipe()
        pid = os.fork()
        if pid == 0:  # ---- child
            os.close(r)
            os.dup2(w, 1)
            os.dup2(w, 2)
            os.close(w)
            _child(req)  # never returns
        # ---- parent
        os.close(w)
        t = threading.Thread(target=_wait, args=(req["id"], pid, r, lock, out), daemon=True)
        t.start()
        waiters.append(t)
    for t in waiters:  # stdin EOF: drain in-flight children before exiting
        t.join()


if __name__ == "__main__":
    serve()
