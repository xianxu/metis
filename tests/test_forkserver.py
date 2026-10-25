"""metis#44 — fork-server tests. Drive the REAL server as a subprocess (real forks, no
mocks): the protocol, per-child env authority, failure surfacing, concurrency, and the
reads.json/used_site_packages contract. Toy step modules live in a tmp GIT repo on
PYTHONPATH so the trace sensor classifies them first-party (D)."""

from __future__ import annotations

import json
import os
import select
import signal
import subprocess
import sys
import textwrap
import threading
import time
from pathlib import Path

import pytest

REPO = Path(__file__).resolve().parents[1]


@pytest.fixture()
def server(tmp_path):
    """A running forkserver whose PYTHONPATH carries tmp_path (toy modules importable) and
    whose preload is disabled (fast start; preload value is covered by the ready line test)."""
    env = dict(os.environ)
    env["PYTHONPATH"] = str(tmp_path)
    env["METIS_FORKSERVER_PRELOAD"] = ""
    proc = subprocess.Popen(
        [sys.executable, "-m", "metis.forkserver"],
        stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        cwd=REPO, env=env, text=True,
    )
    ready = json.loads(proc.stdout.readline())
    assert ready["ready"] is True
    yield proc
    if proc.poll() is None:
        proc.stdin.close()
        proc.wait(timeout=30)


def _toy(tmp_path: Path, name: str, body: str) -> None:
    (tmp_path / f"{name}.py").write_text(textwrap.dedent(body))


def _request(proc, req: dict) -> None:
    proc.stdin.write(json.dumps(req) + "\n")
    proc.stdin.flush()


def _responses(proc, n: int) -> dict[int, dict]:
    got = {}
    while len(got) < n:
        line = proc.stdout.readline()
        assert line, "server stdout closed before all responses arrived"
        resp = json.loads(line)
        got[resp["id"]] = resp
    return got


def _mkstep(tmp_path: Path, i: int) -> Path:
    d = tmp_path / f"step{i}"
    d.mkdir()
    return d


def test_round_trip_env_cwd_metrics(server, tmp_path):
    """The child runs in the request cwd with the request env and its artifacts land where a
    legacy subprocess step's would."""
    _toy(tmp_path, "toy_ok", """
        import json, os
        json.dump({"fold_score": 0.5}, open("metrics.json", "w"))
        open("saw_env.txt", "w").write(os.environ.get("METIS_STEP_ID", "MISSING"))
    """)
    step = _mkstep(tmp_path, 1)
    _request(server, {"id": 1, "module": "toy_ok", "cwd": str(step),
                      "env": {"METIS_STEP_DIR": str(step), "METIS_STEP_ID": "toy"}})
    resp = _responses(server, 1)[1]
    assert resp["exit"] == 0, resp
    assert json.load(open(step / "metrics.json")) == {"fold_score": 0.5}
    assert (step / "saw_env.txt").read_text() == "toy"


def test_env_is_authoritative_no_bleed_between_requests(server, tmp_path):
    """Request 2 must NOT see request 1's METIS_READ_ROOT — the child scrubs METIS_* before
    applying its own env (the seal's absence must be genuine absence, as in exec.go)."""
    _toy(tmp_path, "toy_env", """
        import os
        open("root.txt", "w").write(os.environ.get("METIS_READ_ROOT", "ABSENT"))
    """)
    s1, s2 = _mkstep(tmp_path, 1), _mkstep(tmp_path, 2)
    _request(server, {"id": 1, "module": "toy_env", "cwd": str(s1),
                      "env": {"METIS_STEP_DIR": str(s1), "METIS_READ_ROOT": "/sealed"}})
    assert _responses(server, 1)[1]["exit"] == 0
    _request(server, {"id": 2, "module": "toy_env", "cwd": str(s2),
                      "env": {"METIS_STEP_DIR": str(s2)}})
    assert _responses(server, 1)[2]["exit"] == 0
    assert (s1 / "root.txt").read_text() == "/sealed"
    assert (s2 / "root.txt").read_text() == "ABSENT"


def test_failure_surfaces_traceback_and_exit_code(server, tmp_path):
    _toy(tmp_path, "toy_boom", """
        raise RuntimeError("kaboom-marker")
    """)
    step = _mkstep(tmp_path, 1)
    _request(server, {"id": 5, "module": "toy_boom", "cwd": str(step),
                      "env": {"METIS_STEP_DIR": str(step)}})
    resp = _responses(server, 1)[5]
    assert resp["exit"] != 0
    assert "kaboom-marker" in resp["output"] and "Traceback" in resp["output"]


def test_deliberate_sys_exit_code_passes_through(server, tmp_path):
    _toy(tmp_path, "toy_exit3", """
        raise SystemExit(3)
    """)
    step = _mkstep(tmp_path, 1)
    _request(server, {"id": 9, "module": "toy_exit3", "cwd": str(step),
                      "env": {"METIS_STEP_DIR": str(step)}})
    assert _responses(server, 1)[9]["exit"] == 3


def test_concurrent_requests_match_ids(server, tmp_path):
    """Overlapping children: responses route by id regardless of completion order."""
    _toy(tmp_path, "toy_sleep", """
        import json, os, sys, time
        time.sleep(float(os.environ["METIS_TOY_SLEEP"]))
        json.dump({"n": float(os.environ["METIS_TOY_SLEEP"])}, open("metrics.json", "w"))
    """)
    steps = {i: _mkstep(tmp_path, i) for i in (1, 2, 3, 4)}
    delays = {1: 0.4, 2: 0.05, 3: 0.2, 4: 0.0}
    for i, d in delays.items():
        _request(server, {"id": i, "module": "toy_sleep", "cwd": str(steps[i]),
                          "env": {"METIS_STEP_DIR": str(steps[i]),
                                  "METIS_TOY_SLEEP": str(d)}})
    got = _responses(server, 4)
    for i, d in delays.items():
        assert got[i]["exit"] == 0
        assert json.load(open(steps[i] / "metrics.json")) == {"n": d}


def test_reads_json_written_with_forced_site_packages(server, tmp_path):
    """Each child writes its own reads.json via run_traced, with the toy module in D (the
    tmp repo is a git root) and used_site_packages FORCED true (the metis#44 contract: a
    warm child never observes the site-packages reads that normally set the flag)."""
    subprocess.run(["git", "init", "-q", str(tmp_path)], check=True)
    _toy(tmp_path, "toy_traced", """
        import json
        json.dump({"ok": 1}, open("metrics.json", "w"))
    """)
    step = _mkstep(tmp_path, 1)
    _request(server, {"id": 1, "module": "toy_traced", "cwd": str(step),
                      "env": {"METIS_STEP_DIR": str(step)}})
    assert _responses(server, 1)[1]["exit"] == 0
    reads = json.load(open(step / "reads.json"))
    assert reads["used_site_packages"] is True
    root_paths = reads["roots"].get(str(tmp_path.resolve()), [])
    assert "toy_traced.py" in root_paths


def test_eof_drains_in_flight_children(server, tmp_path):
    """Closing stdin while a child runs: the server waits for it and still emits the
    response before exiting (the shutdown contract Go relies on)."""
    _toy(tmp_path, "toy_slow", """
        import json, time
        time.sleep(0.5)
        json.dump({"ok": 1}, open("metrics.json", "w"))
    """)
    step = _mkstep(tmp_path, 1)
    _request(server, {"id": 1, "module": "toy_slow", "cwd": str(step),
                      "env": {"METIS_STEP_DIR": str(step)}})
    server.stdin.close()
    resp = _responses(server, 1)[1]
    assert resp["exit"] == 0
    assert server.wait(timeout=30) == 0
    assert json.load(open(step / "metrics.json")) == {"ok": 1}


# ── metis#72: live capped log, process groups, cancel, and the server-owned join ──────────────

MARKER = "\n[test: truncated]\n"


def _gate(tmp_path: Path) -> Path:
    g = tmp_path / "gate"
    os.mkfifo(g)
    return g


def _open_gate(g: Path) -> None:
    with open(g, "w") as fh:
        fh.write("go\n")


def _wait_path(p: Path, timeout: float = 10.0) -> None:
    deadline = time.monotonic() + timeout
    while not p.exists():
        assert time.monotonic() < deadline, f"timed out waiting for {p}"
        time.sleep(0.01)


def _response_within(proc, timeout: float):
    """The next response line, or None if none arrives within timeout."""
    r, _, _ = select.select([proc.stdout], [], [], timeout)
    if not r:
        return None
    return json.loads(proc.stdout.readline())


def _lifecycle_req(i, module, step, **extra):
    req = {"id": i, "module": module, "cwd": str(step), "env": {"METIS_STEP_DIR": str(step)},
           "log_path": str(step / "step.log"), "log_cap": 1 << 20, "log_marker": MARKER,
           "grace": 0.2, "join_bound": 0.3}
    req["env"].update(extra.pop("env", {}))
    req.update(extra)
    return req


GATED = """
    import os, sys
    print("early", flush=True)
    open("ready", "w").close()
    open(os.environ["TOY_GATE"]).read()
    print("late", flush=True)
"""


def test_log_streams_before_exit_and_child_has_own_group(server, tmp_path):
    _toy(tmp_path, "toy_gated", GATED + """
    open("pgid", "w").write("%d %d" % (os.getpgid(0), os.getpid()))
    """)
    step, gate = _mkstep(tmp_path, 1), _gate(tmp_path)
    _request(server, _lifecycle_req(1, "toy_gated", step, env={"TOY_GATE": str(gate)}))
    _wait_path(step / "ready")
    assert (step / "step.log").read_text() == "early\n"  # visible while the child still runs
    _open_gate(gate)
    resp = _responses(server, 1)[1]
    assert resp["exit"] == 0 and resp["joined"] is True and resp["log_truncated"] is False, resp
    assert (step / "step.log").read_text() == "early\nlate\n"
    assert "late" in resp["output"]
    pgid, pid = (step / "pgid").read_text().split()
    assert pgid == pid, "a forked step must lead its own process group"


def test_cancel_term(server, tmp_path):
    _toy(tmp_path, "toy_gated", GATED)
    step, gate = _mkstep(tmp_path, 1), _gate(tmp_path)
    _request(server, _lifecycle_req(1, "toy_gated", step, env={"TOY_GATE": str(gate)}))
    _wait_path(step / "ready")
    _request(server, {"cancel": 1, "signal": "TERM"})
    resp = _responses(server, 1)[1]
    assert resp["exit"] == -signal.SIGTERM and resp["joined"] is True, resp


def test_cancel_kill_when_term_ignored(server, tmp_path):
    _toy(tmp_path, "toy_stubborn", """
    import signal, time
    signal.signal(signal.SIGTERM, signal.SIG_IGN)
    open("ready", "w").close()
    while True:
        time.sleep(0.05)
    """)
    step = _mkstep(tmp_path, 1)
    _request(server, _lifecycle_req(1, "toy_stubborn", step))
    _wait_path(step / "ready")
    _request(server, {"cancel": 1, "signal": "TERM"})
    assert _response_within(server, 0.5) is None, "TERM is ignored: no response yet"
    _request(server, {"cancel": 1, "signal": "KILL"})
    resp = _responses(server, 1)[1]
    assert resp["exit"] == -signal.SIGKILL, resp


def test_cancel_after_reap_is_noop(server, tmp_path):
    _toy(tmp_path, "toy_quick", """
    print("done")
    """)
    s1, s2 = _mkstep(tmp_path, 1), _mkstep(tmp_path, 2)
    _request(server, _lifecycle_req(1, "toy_quick", s1))
    assert _responses(server, 1)[1]["exit"] == 0
    _request(server, {"cancel": 1, "signal": "KILL"})  # reaped: must signal nothing
    _request(server, _lifecycle_req(2, "toy_quick", s2))
    assert _responses(server, 1)[2]["exit"] == 0, "the server survives a stale cancel"


def test_output_is_tail_and_log_is_capped(server, tmp_path):
    _toy(tmp_path, "toy_chatty", """
    import sys
    for i in range(20000):
        sys.stdout.write("line %06d\\n" % i)
    """)
    step = _mkstep(tmp_path, 1)
    _request(server, _lifecycle_req(1, "toy_chatty", step, log_cap=1000))
    resp = _responses(server, 1)[1]
    assert resp["exit"] == 0 and resp["log_truncated"] is True, resp
    log = (step / "step.log").read_text()
    assert log.startswith("line 000000\n") and log.endswith(MARKER) and len(log) <= 1000 + len(MARKER)
    assert resp["output"].endswith("line 019999\n") and len(resp["output"]) <= 64 * 1024


def test_escaped_holder_is_unjoined_within_bound(server, tmp_path):
    _toy(tmp_path, "toy_escaper", """
    import os, subprocess, sys, time
    subprocess.Popen([sys.executable, "-c",
        "import os, sys, time; os.setsid(); open(sys.argv[1], 'w').write(str(os.getpid())); time.sleep(30)",
        "orphan.pid"])
    while not os.path.exists("orphan.pid") or not open("orphan.pid").read():
        time.sleep(0.01)
    """)
    step = _mkstep(tmp_path, 1)
    try:
        start = time.monotonic()
        _request(server, _lifecycle_req(1, "toy_escaper", step))
        resp = _response_within(server, 5.0)
        assert resp is not None, "the server must answer within its join bounds"
        assert resp["exit"] == 0 and resp["joined"] is False, resp
        assert time.monotonic() - start < 3.0
        # runtime is fork to exit; the bounded join wait that followed is not compute
        assert 0 <= resp["runtime"] < 0.3 + 0.2 + 0.3, resp
    finally:
        pid = int((step / "orphan.pid").read_text())
        os.kill(pid, signal.SIGKILL)


def test_in_group_lingerer_is_stopped(server, tmp_path):
    _toy(tmp_path, "toy_lingerer", """
    import subprocess
    p = subprocess.Popen(["sleep", "30"])
    open("lingerer.pid", "w").write(str(p.pid))
    """)
    step = _mkstep(tmp_path, 1)
    _request(server, _lifecycle_req(1, "toy_lingerer", step))
    resp = _response_within(server, 5.0)
    assert resp is not None and resp["exit"] == 0 and resp["joined"] is True, resp
    pid = int((step / "lingerer.pid").read_text())
    deadline = time.monotonic() + 5
    while True:
        try:
            os.kill(pid, 0)
        except ProcessLookupError:
            break
        if time.monotonic() > deadline:
            os.kill(pid, signal.SIGKILL)
            pytest.fail("the in-group lingerer survived")
        time.sleep(0.01)


def test_ready_line_declares_protocol(tmp_path):
    env = dict(os.environ, METIS_FORKSERVER_PRELOAD="")
    proc = subprocess.Popen([sys.executable, "-m", "metis.forkserver"], stdin=subprocess.PIPE,
                            stdout=subprocess.PIPE, cwd=REPO, env=env, text=True)
    try:
        assert json.loads(proc.stdout.readline())["protocol"] == 2
    finally:
        proc.stdin.close()
        proc.wait(timeout=30)


def test_unopenable_log_never_launches(server, tmp_path):
    """A log the server cannot open refuses before fork: no child, an explicit not-launched answer."""
    _toy(tmp_path, "toy_marker", """
    open("launched", "w").close()
    """)
    step = _mkstep(tmp_path, 1)
    _request(server, _lifecycle_req(1, "toy_marker", step, log_path=str(step)))  # a directory
    resp = _responses(server, 1)[1]
    assert resp["launched"] is False and resp["exit_known"] is False and "error" in resp, resp
    assert not (step / "launched").exists()


class _FailingSink:
    def write(self, b):
        raise OSError(28, "No space left on device")

    def flush(self):
        pass

    def close(self):
        pass


def test_log_sink_error_keeps_draining_and_reports(tmp_path):
    """A sink error never stops consumption (the child is never SIGPIPEd): the tail keeps
    filling, the first error is kept, and EOF on the owned pipe is still what confirms the join."""
    from metis import forkserver as fs

    log = fs._Log(str(tmp_path / "step.log"), 1 << 20, MARKER)
    log.fh.close()
    log.fh = _FailingSink()
    r, w = os.pipe()
    os.write(w, b"a" * 1000 + b"\nfinished-cleanly\n")
    os.close(w)
    wake_r, wake_w = os.pipe()
    try:
        assert fs._pump(r, log, wake_r) is True, "EOF observed: the join is confirmed"
    finally:
        for fd in (wake_r, wake_w):
            os.close(fd)
    assert "No space left" in log.error
    assert log.tail.endswith(b"finished-cleanly\n")


def test_pump_stopped_by_wake_is_not_eof(tmp_path):
    """Stopping the reader (a holder outside the group keeps the pipe) is not a join."""
    from metis import forkserver as fs

    log = fs._Log(None, None, None)
    r, w = os.pipe()  # the writer stays open: no EOF will come
    wake_r, wake_w = os.pipe()
    os.write(wake_w, b"x")
    try:
        assert fs._pump(r, log, wake_r) is False
    finally:
        for fd in (w, wake_r, wake_w):
            os.close(fd)


def _fork_quiet_child(behavior: str) -> tuple[int, int]:
    """Fork a child (stdout and stderr to /dev/null, so its pipe EOF arrives at once) that either
    exits 3 after a moment ("exits") or ignores TERM and loops ("stubborn"). Returns (pid, r)."""
    r, w = os.pipe()
    pid = os.fork()
    if pid == 0:  # never returns: no pytest code runs in the child
        try:
            os.setpgid(0, 0)
            devnull = os.open(os.devnull, os.O_WRONLY)
            os.dup2(devnull, 1)
            os.dup2(devnull, 2)
            os.close(r)
            os.close(w)
            if behavior == "exits":
                time.sleep(0.2)
                os._exit(3)
            signal.signal(signal.SIGTERM, signal.SIG_IGN)
            while True:
                time.sleep(0.05)
        finally:
            os._exit(1)
    os.close(w)
    return pid, r


def _run_waiter(fs, pid, r):
    import io as _io

    lock = threading.Lock()
    out = _io.StringIO()
    fs._children[1] = pid
    t = threading.Thread(target=fs._wait, args=({"id": 1, "grace": 0.2, "join_bound": 0.2}, pid, r,
                                                fs._Log(None, None, None), lock, out), daemon=True)
    t.start()
    return t, lock, out


def _check_answered_and_reaped(fs, t, lock, out, pid):
    time.sleep(0.1)
    assert lock.acquire(timeout=0.5), "the waiter must never block while holding the lock"
    lock.release()
    t.join(timeout=5)
    assert not t.is_alive(), "the waiter must finish within its bounds"
    resp = json.loads(out.getvalue())
    with pytest.raises(ChildProcessError):
        os.waitpid(pid, os.WNOHANG)  # already reaped by the waiter
    assert 1 not in fs._children
    return resp


def _cleanup(fs, pid):
    try:
        os.kill(pid, signal.SIGKILL)
        os.waitpid(pid, 0)
    except (ProcessLookupError, ChildProcessError):
        pass
    fs._children.pop(1, None)


def _boom(*_a, **_k):
    raise RuntimeError("injected waiter failure")


# The child of these in-process forks makes only raw os calls and exits; it takes no Python lock,
# so forking from pytest's multi-threaded process is safe here.
@pytest.mark.filterwarnings("ignore:This process .* is multi-threaded:DeprecationWarning")
def test_waiter_failure_after_exit_answers_unjoined_and_reaps(monkeypatch):
    """BR-10: a waiter failure after the child's exit still reaps it and answers, unjoined, with
    the real exit code (not an invented one)."""
    from metis import forkserver as fs

    monkeypatch.setattr(fs, "_drain_join", _boom)  # before the fork: a failed patch leaks nothing
    pid, r = _fork_quiet_child("exits")
    try:
        t, lock, out = _run_waiter(fs, pid, r)
        resp = _check_answered_and_reaped(fs, t, lock, out, pid)
        assert resp["joined"] is False and "injected waiter failure" in resp["error"], resp
        assert resp["exit_known"] is True and resp["exit"] == 3, resp
    finally:
        _cleanup(fs, pid)


# The child of these in-process forks makes only raw os calls and exits; it takes no Python lock,
# so forking from pytest's multi-threaded process is safe here.
@pytest.mark.filterwarnings("ignore:This process .* is multi-threaded:DeprecationWarning")
def test_exit_watcher_failure_falls_back_and_still_reaps(monkeypatch):
    """BR-10: when exit observation itself fails, a non-blocking reap under the lock takes over:
    a cancel still ends the child, which is reaped and answered (unjoined, error reported)."""
    from metis import forkserver as fs

    monkeypatch.setattr(fs, "_wait_exit_noreap", _boom)
    pid, r = _fork_quiet_child("stubborn")
    try:
        t, lock, out = _run_waiter(fs, pid, r)
        time.sleep(0.2)
        assert lock.acquire(timeout=1), "the waiter must not hold the lock while the child lives"
        try:
            fs._signal_group(1, signal.SIGKILL)  # what a cancel request does
        finally:
            lock.release()
        resp = _check_answered_and_reaped(fs, t, lock, out, pid)
        assert resp["exit_known"] is True and resp["exit"] == -signal.SIGKILL, resp
        assert resp["joined"] is False and "injected waiter failure" in resp["error"], resp
    finally:
        _cleanup(fs, pid)
