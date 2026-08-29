from __future__ import annotations

import subprocess
from pathlib import Path

import yaml

from ptsip.clarification.generator import analyze_clarifications
from ptsip.constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION


def _git(repo: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


def _repo(tmp_path: Path) -> Path:
    repo = tmp_path / "repo"
    repo.mkdir(parents=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")
    (repo / "tools").mkdir()
    (repo / "tools" / "generate.py").write_text("print('generate')\n", encoding="utf-8")
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")
    return repo


def _write_valid_profile(profile: Path) -> None:
    payload = {
        "ptsip": {
            "version": CURRENT_PROJECT_PROFILE_VERSION,
            "specification": {
                "family": SPEC_VERSION,
                "source": SPEC_SOURCE,
                "revision": SPEC_REVISION,
            },
        },
        "responsibility_map": {"mode": "explicit"},
        "components": [
            {
                "id": "tools",
                "classification": "DEVELOPMENT_TOOLING",
                "include": ["tools/**"],
                "purpose": "Repository-local generation tooling",
                "shipped": False,
                "runtime_required": False,
                "executable": True,
            }
        ],
        "policies": {
            "product_to_nonproduct_runtime_dependency": "deny",
            "nonproduct_in_product_package": "deny",
            "independent_build_resolution": "required",
        },
    }
    profile.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")


def test_invalid_profile_reports_non_authoritative_recovery_and_blocks_questions(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    profile = repo / "ptsip.yaml"
    profile.write_text("ptsip: [invalid\n", encoding="utf-8")

    analysis = analyze_clarifications(repo, ["tools"])
    payload = analysis.as_dict("en")
    recovery = payload["profile"]["recovery"]

    assert analysis.status == "PROFILE_INVALID"
    assert analysis.requests == ()
    assert payload["profile"]["failure_stage"] == "PARSE"
    assert recovery["status"] == "PROJECT_CORRECTION_REQUIRED"
    assert recovery["authoritative"] is False
    assert recovery["raw_profile_fallback"] is False
    assert recovery["reuse_partial_effective_state"] is False
    assert recovery["retry_requires_fresh_repository_snapshot"] is True


def test_corrected_profile_retry_uses_fresh_effective_profile(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    profile = repo / "ptsip.yaml"
    profile.write_text("ptsip: [invalid\n", encoding="utf-8")

    first = analyze_clarifications(repo, ["tools"])
    assert first.status == "PROFILE_INVALID"

    _write_valid_profile(profile)
    second = analyze_clarifications(repo, ["tools"])

    assert second.status == "NO_CLARIFICATION_REQUIRED"
    assert second.requests == ()
    assert second.profile_parse_error is None
    assert second.profile_failure_stage is None
    assert second.profile_recovery is None
