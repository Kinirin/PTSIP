from __future__ import annotations

from dataclasses import replace
import subprocess
from pathlib import Path

import pytest

from developer.automation.pp_remote_verify import (
    PPRemoteVerifyError,
    introduced_commits,
    require_comparable_parent_authority,
    verify_commit,
    verify_range,
)
from developer.automation.pp_transition_delta import GitSnapshot, load_authority_state


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


def test_current_head_commit_is_remote_verify_safe() -> None:
    result = verify_commit(ROOT, "HEAD")

    assert result.commit == _git("rev-parse", "HEAD")
    assert result.classification in {
        "NO_T2_AUTHORITY_DELTA",
        "T2_AUTHORITY_DELTA",
        "MERGE_INHERITED_PP_AUTHORITY",
    }


def test_single_commit_range_verifies_every_introduced_commit() -> None:
    parent = _git("rev-parse", "HEAD^")
    head = _git("rev-parse", "HEAD")

    assert introduced_commits(ROOT, parent, head) == (head,)
    results = verify_range(ROOT, base=parent, head=head)
    assert [item.commit for item in results] == [head]


def test_divergent_merge_parent_authority_fails_closed() -> None:
    current = load_authority_state(GitSnapshot(ROOT, "HEAD"))
    divergent = replace(
        current,
        label="synthetic-divergent-parent",
        current="pp.9.99",
    )

    with pytest.raises(PPRemoteVerifyError) as exc_info:
        require_comparable_parent_authority((current, divergent))

    assert exc_info.value.code == "DIVERGENT_PARENT_PP_AUTHORITY"


def test_symbolic_commit_ref_is_normalized_to_sha() -> None:
    result = verify_commit(ROOT, "HEAD")

    assert result.commit == _git("rev-parse", "--verify", "HEAD^{commit}")
    assert len(result.commit) == 40
