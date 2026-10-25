"""The experiment merge-checks refuse when the vocabulary tool is missing (metis#70).

A missing tool must never read as a schema verdict: in CI, command-not-found
once surfaced as "valid-baseline.md should PASS but was rejected".
"""

import pathlib
import subprocess

import pytest

ROOT = pathlib.Path(__file__).resolve().parent.parent
CHECKS = ROOT / "scripts" / "merge-checks.d"


@pytest.mark.parametrize(
    "check, args",
    [
        ("experiment-schema-selftest.sh", []),
        ("experiment-validate.sh", ["HEAD", "HEAD"]),
    ],
)
def test_missing_vocabulary_is_not_a_verdict(check, args):
    run = subprocess.run(
        ["bash", str(CHECKS / check), *args],
        cwd=ROOT,
        env={"PATH": "/usr/bin:/bin", "VOCAB": "metis-no-such-vocabulary"},
        capture_output=True,
        text=True,
    )
    assert run.returncode == 2, run.stdout + run.stderr
    assert "vocabulary tool 'metis-no-such-vocabulary' not found" in run.stderr
    assert "rejected" not in run.stdout


def test_vocab_helper_is_not_run_as_a_check():
    # run-merge-checks runs every executable in merge-checks.d; the helper is sourced only.
    helper = CHECKS / "_vocab.bash"
    assert helper.is_file()
    assert not helper.stat().st_mode & 0o111
