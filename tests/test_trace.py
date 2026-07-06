"""Tests for metis.trace's multi-root read-set (metis#11).

The sensor must capture first-party code from EVERY repo a step touches (metis + a
consumer repo like kbench), not just the metis repo — so editing a consumer step's
own code changes D and busts the metis#2 cache.
"""

import os
import subprocess

import pytest

import metis.trace as trace


def _git_init(d):
    subprocess.run(["git", "init", "-q", d], check=True)


@pytest.fixture(autouse=True)
def _reset_sensor():
    """Each test starts from a clean sensor state (module globals) + a fresh root cache."""
    trace._roots = {}
    trace._used_site_packages = False
    trace._repo_root.cache_clear() if hasattr(trace._repo_root, "cache_clear") else None
    trace._root_cache = {}
    yield


def test_classify_groups_reads_by_repo_root(tmp_path):
    """A read under repo A and a read under repo B land under their OWN roots — the
    multi-root guarantee (was: only the metis repo's reads were kept)."""
    a = tmp_path / "repoA"
    b = tmp_path / "repoB"
    (a / "pkg").mkdir(parents=True)
    (b / "pkg").mkdir(parents=True)
    _git_init(str(a))
    _git_init(str(b))
    fa = a / "pkg" / "mod_a.py"
    fb = b / "pkg" / "mod_b.py"
    fa.write_text("# a\n")
    fb.write_text("# b\n")

    trace._classify(str(fa))
    trace._classify(str(fb))

    assert str(a) in trace._roots and str(b) in trace._roots, trace._roots
    assert trace._roots[str(a)] == {"pkg/mod_a.py"}
    assert trace._roots[str(b)] == {"pkg/mod_b.py"}


def test_repo_root_finds_git_dir_and_git_file(tmp_path):
    """`.git` can be a DIR (normal repo) or a FILE (linked worktree/submodule) — both
    mark a repo root."""
    normal = tmp_path / "normal"
    (normal / "sub").mkdir(parents=True)
    _git_init(str(normal))
    assert trace._repo_root(str(normal / "sub" / "x.py")) == str(normal)

    worktree = tmp_path / "worktree"
    (worktree / "sub").mkdir(parents=True)
    (worktree / ".git").write_text("gitdir: /somewhere/.git/worktrees/wt\n")  # a .git FILE
    assert trace._repo_root(str(worktree / "sub" / "y.py")) == str(worktree)


def test_site_packages_sets_flag_not_a_read(tmp_path):
    trace._classify("/anywhere/.venv/lib/python3.12/site-packages/pandas/__init__.py")
    assert trace._used_site_packages is True
    assert trace._roots == {}


def test_non_repo_path_is_dropped(tmp_path):
    """A file under no git repo (stdlib/temp) is not first-party — dropped."""
    loose = tmp_path / "loose.py"  # tmp_path itself is not a git repo
    loose.write_text("x\n")
    trace._classify(str(loose))
    assert trace._roots == {}


def test_run_dir_excluded(tmp_path, monkeypatch):
    """Everything under METIS_RUN_DIR (a step's own outputs + upstream artifacts) is
    excluded even inside a repo — else outputs change every run and every step MISSes."""
    repo = tmp_path / "repo"
    (repo / "runs" / "r1" / "step").mkdir(parents=True)
    _git_init(str(repo))
    monkeypatch.setenv("METIS_RUN_DIR", str(repo / "runs" / "r1"))
    out = repo / "runs" / "r1" / "step" / "out.json"
    out.write_text("{}\n")
    trace._classify(str(out))
    assert trace._roots == {}


def test_write_reads_emits_roots_map(tmp_path, monkeypatch):
    """reads.json v2: a `roots` map (repo-root → sorted rel paths) + used_site_packages."""
    repo = tmp_path / "repo"
    (repo / "pkg").mkdir(parents=True)
    _git_init(str(repo))
    f = repo / "pkg" / "m.py"
    f.write_text("x\n")
    trace._classify(str(f))
    step_dir = tmp_path / "stepdir"
    step_dir.mkdir()
    monkeypatch.setenv("METIS_STEP_DIR", str(step_dir))
    trace._write_reads()

    import json

    payload = json.loads((step_dir / "reads.json").read_text())
    # v2 format: a `roots` map, no v1 `reads`/`project_root`. (`_write_reads` also snapshots
    # metis's own loaded modules — so other roots may appear; assert OUR repo is grouped right.)
    assert "roots" in payload and "reads" not in payload and "project_root" not in payload
    assert payload["roots"].get(str(repo)) == ["pkg/m.py"], payload["roots"]
    assert payload["used_site_packages"] in (True, False)
