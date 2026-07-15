"""metis#44 — fork-server tests. Drive the REAL server as a subprocess (real forks, no
mocks): the protocol, per-child env authority, failure surfacing, concurrency, and the
reads.json/used_site_packages contract. Toy step modules live in a tmp GIT repo on
PYTHONPATH so the trace sensor classifies them first-party (D)."""

from __future__ import annotations

import json
import os
import subprocess
import sys
import textwrap
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
