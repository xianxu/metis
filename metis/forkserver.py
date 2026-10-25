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
  →  {"id": 7, "module": "kbench.titanic.features", "cwd": "<stepDir>", "env": {"METIS_*": …},
      "log_path": "<stepDir>/step.log", "log_cap": N, "log_marker": "…",   (metis#72, optional)
      "grace": s, "join_bound": s}
  →  {"cancel": 7, "signal": "TERM" | "KILL"}   (metis#72: signal request 7's process group)
  ←  {"ready": true}                        (once, after preload — requests may be sent before)
  ←  {"id": 7, "exit": 0, "output": "<combined child stdout+stderr, tail>", "joined": true,
      "log_truncated": false, "runtime": <seconds, fork to exit>}
stdin EOF ⇒ drain in-flight children, exit 0.

Lifecycle (metis#72), mirroring the Go legacy path: each child leads its own process group;
its output streams live into log_path (capped, one marker, the rest drained) while a tail is
kept for the response. The server is the only process that reaps a child, and it reaps only
under the protocol lock while removing the child from its table, so a cancel signals a group
whose leader is unreaped (its pid, hence the group id, still reserved): never a recycled pid.
After the child exits, the join is confirmed when its pipe drains within join_bound; failing
that the server stops what is left of the group (TERM, grace, KILL) and waits join_bound
again. A holder that left the group keeps the pipe: the server stops reading and answers
joined=false. Every response therefore arrives within bounded time of the child's exit.

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
import io
import json
import os
import select
import selectors
import signal
import sys
import threading
import time
import traceback

_DEFAULT_PRELOAD = "numpy,pandas,sklearn,pyarrow"
_TAIL = 64 * 1024  # response output: the log tail (Go's stepLogTail)
PROTOCOL = 2  # metis#72: lifecycle fields and cancel; the Go client refuses an older server
_SIGNALS = {"TERM": signal.SIGTERM, "KILL": signal.SIGKILL}

# id -> pid of every forked child not yet reaped. Guarded by the protocol lock: insertion at
# fork, removal together with the reap, and every cancel signal, so none can interleave.
_children: dict = {}


class _Log:
    """The bounded live log (metis#72; Go's boundedLog): bytes go to log_path until the cap, then
    one marker, the rest drained; a fixed tail is kept for the response. Opening raises (the
    caller refuses the request before fork); a later sink error never raises: consumption goes
    on (a stalled reader would SIGPIPE or block the child) and the first error is kept."""

    def __init__(self, path, cap, marker):
        self.fh = open(path, "wb") if path else None
        self.cap = cap if cap is not None else (1 << 62)
        self.marker = (marker or "\n[step.log truncated]\n").encode()
        self.written = 0
        self.consumed = 0
        self.truncated = False
        self.tail = bytearray()
        self.error = ""

    def _put(self, b: bytes) -> None:
        if self.fh is None or self.error:
            return
        try:
            self.fh.write(b)
            self.fh.flush()
        except OSError as e:
            self.error = f"step.log: {e}"

    def write(self, b: bytes) -> None:
        self.consumed += len(b)
        room = self.cap - self.written
        if room > 0:
            chunk = b[:room]
            self._put(chunk)
            self.written += len(chunk)
        if not self.truncated and self.consumed > self.cap:
            self.truncated = True
            self._put(self.marker)
        self.tail += b[-_TAIL:]
        if len(self.tail) > _TAIL:
            del self.tail[:-_TAIL]

    def close(self) -> None:
        if self.fh:
            try:
                self.fh.close()
            except OSError as e:
                self.error = self.error or f"step.log: {e}"


def _pump(rfd: int, log: _Log, wake_r: int) -> bool:
    """Copy the child's pipe into the log until EOF (returns True: the only evidence of a
    drained join) or until woken (returns False: a holder outside the group keeps the pipe, so
    the server stops reading it). One selector, no polling, no fd-number limit."""
    with selectors.DefaultSelector() as sel:
        sel.register(rfd, selectors.EVENT_READ)
        sel.register(wake_r, selectors.EVENT_READ)
        while True:
            for key, _ in sel.select():
                if key.fd == wake_r:
                    return False
                b = os.read(rfd, 65536)
                if not b:
                    return True
                log.write(b)


def _wait_exit_noreap(pid: int) -> None:
    """Block until the child exits WITHOUT reaping it (its pid and group id stay reserved)."""
    if hasattr(os, "waitid"):
        os.waitid(os.P_PID, pid, os.WEXITED | os.WNOWAIT)
        return
    kq = select.kqueue()  # macOS: no waitid in Python
    try:
        ev = select.kevent(pid, filter=select.KQ_FILTER_PROC,
                           flags=select.KQ_EV_ADD | select.KQ_EV_ONESHOT, fflags=select.KQ_NOTE_EXIT)
        try:
            kq.control([ev], 1, None)
        except ProcessLookupError:
            pass  # already exited (a zombie only this server can reap)
    finally:
        kq.close()


def _signal_group(req_id, sig) -> None:
    """Signal a child's process group iff it is still unreaped. Caller holds the lock."""
    pid = _children.get(req_id)
    if pid is not None:
        try:
            os.killpg(pid, sig)
        except (ProcessLookupError, PermissionError):
            pass  # nothing left to signal (macOS: EPERM when only the exited leader remains)


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


def _respond(lock: threading.Lock, out, resp: dict) -> None:
    with lock:
        out.write(json.dumps(resp) + "\n")
        out.flush()


def _wait(req: dict, pid: int, rfd: int, log: _Log, lock: threading.Lock, out) -> None:
    """Waiter thread: stream the child's output into its log, observe its exit without reaping,
    confirm or give up the join within bounds, then reap under the lock and respond. Every exit
    path, an unexpected error included, stops the child's group if its exit was not observed,
    reaps it and answers; an error is reported with joined=false, never as a clean ending. IO
    only: never forks, never touches os.environ."""
    req_id = req["id"]
    started = time.monotonic()  # the waiter starts right after the fork
    grace = float(req.get("grace", 10.0))
    join = float(req.get("join_bound", 5.0))
    wake_r, wake_w = os.pipe()
    result = {"eof": False}
    done = threading.Event()

    def read() -> None:
        try:
            result["eof"] = _pump(rfd, log, wake_r)
        finally:
            done.set()

    def stop_group(sig) -> None:
        with lock:
            _signal_group(req_id, sig)

    reader = threading.Thread(target=read, daemon=True)
    reader.start()
    runtime = None
    error = ""
    try:
        _wait_exit_noreap(pid)
        runtime = time.monotonic() - started  # launch to exit: the join wait is not compute
        if not done.wait(join):
            stop_group(signal.SIGTERM)
            if not done.wait(grace):
                stop_group(signal.SIGKILL)
                done.wait(join)
    except BaseException:
        error = traceback.format_exc()
        if runtime is None:  # the exit was never observed: end the group before reaping
            stop_group(signal.SIGTERM)
            if not done.wait(grace):
                stop_group(signal.SIGKILL)
    finally:
        if not done.is_set():
            os.write(wake_w, b"x")  # a holder outside the group keeps the pipe: stop reading
        reader.join()
        for fd in (rfd, wake_r, wake_w):
            os.close(fd)
        log.close()
        with lock:
            _children.pop(req_id, None)
            _, status = os.waitpid(pid, 0)
    resp = {"id": req_id, "launched": True, "exit_known": True,
            "exit": os.waitstatus_to_exitcode(status), "output": log.tail.decode("utf-8", "replace"),
            "joined": result["eof"] and not error, "log_truncated": log.truncated,
            "log_error": log.error}
    if runtime is not None:
        resp["runtime"] = runtime
    if error:
        resp["error"] = error
    _respond(lock, out, resp)


def serve() -> None:
    out = sys.stdout
    lock = threading.Lock()
    loaded = _preload()
    with lock:
        out.write(json.dumps({"ready": True, "preloaded": loaded, "protocol": PROTOCOL}) + "\n")
        out.flush()
    waiters: list[threading.Thread] = []
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        req = json.loads(line)  # a malformed line is a protocol bug — fail loud, not skip
        if "cancel" in req:  # metis#72: signal an unreaped child's group; a reaped one is a no-op
            with lock:
                _signal_group(req["cancel"], _SIGNALS[req["signal"]])
            continue
        try:
            log = _Log(req.get("log_path"), req.get("log_cap"), req.get("log_marker"))
        except OSError as e:  # metis#72: refuse before fork, so nothing launches
            _respond(lock, out, {"id": req["id"], "launched": False, "exit_known": False,
                                 "joined": False, "error": f"step.log: {e}", "output": ""})
            continue
        r, w = os.pipe()
        # C1 (close review): fork UNDER the protocol lock. Waiter threads write responses to
        # `out` only while holding `lock`, so holding it across fork() guarantees stdout's
        # internal BufferedWriter/TextIOWrapper locks are FREE in the child — otherwise a
        # fork landing mid-write copies a held io lock into the child (whose owner thread
        # doesn't exist there) and the child deadlocks at its first stdout use, hanging the
        # whole run. The child never touches `lock` itself.
        with lock:
            pid = os.fork()
            if pid:
                # metis#72: the child leads its own group. Set it here too (the child does the
                # same) so a cancel arriving before the child runs still reaches the group.
                try:
                    os.setpgid(pid, pid)
                except OSError:
                    pass
                _children[req["id"]] = pid
        if pid == 0:  # ---- child
            os.setpgid(0, 0)
            os.close(r)
            os.dup2(w, 1)
            os.dup2(w, 2)
            os.close(w)
            # Belt-and-braces for C1: rebind std streams onto fresh objects over the duped
            # fds so NO parent buffer/lock state is reachable in the child at all.
            sys.stdout = io.TextIOWrapper(io.FileIO(1, "wb", closefd=False), line_buffering=True)
            sys.stderr = io.TextIOWrapper(io.FileIO(2, "wb", closefd=False), line_buffering=True)
            # D symmetry: the child inherits metis.forkserver in sys.modules (first-party →
            # it would land in every forkserver-run step's read-set D, making cache keys
            # differ between executors for the same step). The step never imports it — drop
            # it before the trace snapshot.
            sys.modules.pop("metis.forkserver", None)
            _child(req)  # never returns
        # ---- parent
        os.close(w)
        t = threading.Thread(target=_wait, args=(req, pid, r, log, lock, out), daemon=True)
        t.start()
        waiters.append(t)
    for t in waiters:  # stdin EOF: drain in-flight children before exiting
        t.join()


if __name__ == "__main__":
    serve()
