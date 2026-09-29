from __future__ import annotations

import subprocess
from pathlib import Path

import pytest
import yaml

from developer.automation.pp.pp_release_verify import (
    PPReleaseVerifyError,
    verify_exact_release_snapshot,
)


ROOT = Path(__file__).resolve().parents[3]


def _git(*args: str) -> str:
    return subprocess.run(
        ["git", *args],
        cwd=ROOT,
        check=True,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    ).stdout.strip()


def test_current_head_is_exact_sha_release_safe() -> None:
    expected = _git("rev-parse", "--verify", "HEAD^{commit}")
    registry = yaml.safe_load(
        (ROOT / "registry" / "project-profile-contracts.yaml").read_text(
            encoding="utf-8"
        )
    )
    current = registry["current"]
    current_contract = next(
        row for row in registry["contracts"] if row["version"] == current
    )

    result = verify_exact_release_snapshot(ROOT, expected_sha="HEAD")

    assert result["status"] == "PASS"
    assert result["source_sha"] == expected
    assert result["project_profile"] == current
    assert result["schema"] == current_contract["schema"]


def test_release_verifier_rejects_non_head_exact_sha() -> None:
    parent = _git("rev-parse", "--verify", "HEAD^")

    with pytest.raises(PPReleaseVerifyError) as exc_info:
        verify_exact_release_snapshot(ROOT, expected_sha=parent)

    assert exc_info.value.code == "RELEASE_EXACT_SHA_MISMATCH"


def test_release_verifier_is_verify_only() -> None:
    source = (
        ROOT / "developer" / "automation" / "pp" / "pp_release_verify.py"
    ).read_text(encoding="utf-8")

    for mutation in (
        '"add"',
        '"commit"',
        '"checkout"',
        '"restore"',
        '"rm"',
    ):
        assert mutation not in source
