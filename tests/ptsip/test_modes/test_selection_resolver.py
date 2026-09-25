from __future__ import annotations

import runpy
import subprocess
from pathlib import Path

import pytest
import yaml


REPO_ROOT = Path(__file__).resolve().parents[3]
REGISTRY_PATH = REPO_ROOT / ".github" / "test_modes.yaml"
PROFILE_PATH = REPO_ROOT / "developer" / "profiles" / "ptsip-repository.yaml"
RESOLVER_PATH = REPO_ROOT / ".github" / "scripts" / "resolve_test_modes.py"
RESOLVER = runpy.run_path(str(RESOLVER_PATH))

MATCHES_PATTERN = RESOLVER["matches_pattern"]
NORMALIZE_REPO_PATH = RESOLVER["normalize_repo_path"]
SELECT_AUTOMATIC = RESOLVER["select_automatic_modes"]
RESOLVE_AUTOMATIC = RESOLVER["resolve_automatic_selection"]
SELECT_MANUAL = RESOLVER["select_manual_modes"]
BUILD_PLAN = RESOLVER["build_execution_plan"]
CHANGED_FILES_FROM_GIT = RESOLVER["changed_files_from_git"]
SELECTION_ERROR = RESOLVER["TestModeSelectionError"]

EXPECTED_MODE_IDS = [
    "ptsip-core",
    "ptsip-evidence",
    "ptsip-source-compat",
    "ptsip-migration",
    "ptsip-remediation",
    "vpms",
    "ptsip-contract",
    "repository-architecture",
    "repository-release",
    "test-mode-control-plane",
]


def _yaml(path: Path) -> dict[str, object]:
    payload = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
    assert isinstance(payload, dict)
    return payload


def _registry() -> dict[str, object]:
    return _yaml(REGISTRY_PATH)


def _profile() -> dict[str, object]:
    return _yaml(PROFILE_PATH)


def _ids(selected: list[dict[str, object]]) -> list[str]:
    return [str(mode["id"]) for mode in selected]


def test_repo_path_normalization_accepts_windows_separators() -> None:
    assert (
        NORMALIZE_REPO_PATH(r"src\ptsip\evidence\contract.py")
        == "src/ptsip/evidence/contract.py"
    )


def test_profile_pattern_matching_respects_recursive_repository_globs() -> None:
    assert MATCHES_PATTERN(
        "src/ptsip/evidence/contract.py",
        "src/ptsip/evidence/**",
    )
    assert not MATCHES_PATTERN(
        "src/ptsip/migration/model.py",
        "src/ptsip/evidence/**",
    )


def test_evidence_change_selects_declared_dependents() -> None:
    selected = SELECT_AUTOMATIC(
        _registry(),
        _profile(),
        ["src/ptsip/evidence/contract.py"],
    )
    assert _ids(selected) == [
        "ptsip-evidence",
        "ptsip-migration",
        "ptsip-remediation",
        "vpms",
    ]


def test_source_compat_change_selects_declared_dependents() -> None:
    selected = SELECT_AUTOMATIC(
        _registry(),
        _profile(),
        ["src/ptsip/source_compat/reader.py"],
    )
    assert _ids(selected) == [
        "ptsip-source-compat",
        "ptsip-migration",
        "vpms",
    ]


def test_migration_change_selects_migration_and_current_vpms_boundary() -> None:
    selected = SELECT_AUTOMATIC(
        _registry(),
        _profile(),
        ["src/ptsip/migration/model.py"],
    )
    assert _ids(selected) == ["ptsip-migration", "vpms"]


def test_support_policy_change_selects_all_declared_support_verifiers() -> None:
    selected = SELECT_AUTOMATIC(
        _registry(),
        _profile(),
        ["docs/Support_policy/policy/index.yaml"],
    )
    assert _ids(selected) == [
        "ptsip-core",
        "ptsip-contract",
        "repository-architecture",
        "repository-release",
    ]


def test_governance_change_selects_declared_cross_boundary_verifiers() -> None:
    selected = SELECT_AUTOMATIC(
        _registry(),
        _profile(),
        ["src/ptsip/governance/authority.py"],
    )
    assert _ids(selected) == [
        "ptsip-core",
        "vpms",
        "ptsip-contract",
        "repository-architecture",
    ]


def test_vpms_change_selects_only_vpms() -> None:
    selected = SELECT_AUTOMATIC(
        _registry(),
        _profile(),
        ["src/vpms/model.py"],
    )
    assert _ids(selected) == ["vpms"]


def test_test_change_selects_only_declared_verification_owner() -> None:
    selected = SELECT_AUTOMATIC(
        _registry(),
        _profile(),
        ["tests/ptsip/evidence/test_normalization_037.py"],
    )
    assert _ids(selected) == ["ptsip-evidence"]


def test_agent_contract_source_change_selects_contract_verification() -> None:
    selected = SELECT_AUTOMATIC(
        _registry(),
        _profile(),
        ["src/ptsip/agent_contracts/operations/conform.yaml"],
    )
    assert _ids(selected) == ["vpms", "ptsip-contract", "repository-release"]


def test_agent_contract_test_change_selects_contract_verification() -> None:
    selected = SELECT_AUTOMATIC(
        _registry(),
        _profile(),
        ["tests/ptsip/agent_contracts/test_operation_loading.py"],
    )
    assert _ids(selected) == ["ptsip-contract"]


def test_pre_commit_hook_change_selects_architecture_verification() -> None:
    selected = SELECT_AUTOMATIC(
        _registry(),
        _profile(),
        [".githooks/pre-commit"],
    )
    assert _ids(selected) == ["repository-architecture"]


def test_distribution_contract_validator_change_selects_release_verification() -> None:
    selected = SELECT_AUTOMATIC(
        _registry(),
        _profile(),
        [".github/scripts/verify_distribution_contracts.py"],
    )
    assert _ids(selected) == ["repository-release"]


def test_shared_ptsip_conftest_change_fans_out_to_ptsip_test_modes() -> None:
    selected = SELECT_AUTOMATIC(
        _registry(),
        _profile(),
        ["tests/ptsip/conftest.py"],
    )
    assert _ids(selected) == [
        "ptsip-core",
        "ptsip-evidence",
        "ptsip-source-compat",
        "ptsip-migration",
        "ptsip-remediation",
        "ptsip-contract",
        "repository-architecture",
        "repository-release",
        "test-mode-control-plane",
    ]


def test_test_mode_control_plane_change_does_not_expand_to_all_modes() -> None:
    selected = SELECT_AUTOMATIC(
        _registry(),
        _profile(),
        [".github/scripts/resolve_test_modes.py"],
    )
    assert _ids(selected) == ["test-mode-control-plane"]


def test_tooling_workflow_change_selects_release_and_control_plane() -> None:
    selected = SELECT_AUTOMATIC(
        _registry(),
        _profile(),
        [".github/workflows/tooling-test.yml"],
    )
    assert _ids(selected) == [
        "repository-release",
        "test-mode-control-plane",
    ]


def test_declared_documentation_change_can_require_no_test_mode() -> None:
    selected, no_verification = RESOLVE_AUTOMATIC(
        _registry(),
        _profile(),
        ["README.md"],
    )
    assert _ids(selected) == []
    assert no_verification == ["README.md"]


def test_unmapped_change_fails_closed() -> None:
    with pytest.raises(SELECTION_ERROR, match="unmapped changed paths"):
        RESOLVE_AUTOMATIC(
            _registry(),
            _profile(),
            ["unregistered-area/file.txt"],
        )


def test_broad_dependent_analysis_input_cannot_create_ownership() -> None:
    with pytest.raises(SELECTION_ERROR, match="unmapped changed paths"):
        RESOLVE_AUTOMATIC(
            _registry(),
            _profile(),
            ["src/ptsip/unregistered_subsystem/new_file.py"],
        )


def test_manual_specific_mode_selects_only_requested_mode() -> None:
    selected = SELECT_MANUAL(_registry(), "ptsip-evidence")
    assert _ids(selected) == ["ptsip-evidence"]


def test_manual_all_is_not_a_special_escape_hatch() -> None:
    with pytest.raises(SELECTION_ERROR, match="unknown requested Test Mode"):
        SELECT_MANUAL(_registry(), "all")


def test_manual_unknown_mode_fails_closed() -> None:
    with pytest.raises(SELECTION_ERROR, match="unknown requested Test Mode"):
        SELECT_MANUAL(_registry(), "missing-mode")


def test_execution_plan_contains_execution_identity_not_architecture_authority() -> None:
    selected = SELECT_AUTOMATIC(
        _registry(),
        _profile(),
        ["src/ptsip/evidence/contract.py"],
    )
    plan = BUILD_PLAN(selected)

    assert [item["id"] for item in plan] == [
        "ptsip-evidence",
        "ptsip-migration",
        "ptsip-remediation",
        "vpms",
    ]
    assert all("classification" not in item for item in plan)
    assert all("roles" not in item for item in plan)
    assert all("purpose" not in item for item in plan)



def _git(repo_root: Path, *args: str) -> str:
    result = subprocess.run(
        ["git", *args],
        cwd=repo_root,
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout.strip()


def test_automatic_git_diff_defaults_to_immediate_parent(tmp_path: Path) -> None:
    _git(tmp_path, "init")
    _git(tmp_path, "checkout", "-b", "main")
    _git(tmp_path, "config", "user.email", "tests@example.invalid")
    _git(tmp_path, "config", "user.name", "PTSIP Tests")

    (tmp_path / "base.txt").write_text("base\n", encoding="utf-8")
    _git(tmp_path, "add", "base.txt")
    _git(tmp_path, "commit", "-m", "base")

    _git(tmp_path, "checkout", "-b", "feature")
    (tmp_path / "first.txt").write_text("first\n", encoding="utf-8")
    _git(tmp_path, "add", "first.txt")
    _git(tmp_path, "commit", "-m", "first")
    (tmp_path / "second.txt").write_text("second\n", encoding="utf-8")
    _git(tmp_path, "add", "second.txt")
    _git(tmp_path, "commit", "-m", "second")

    assert CHANGED_FILES_FROM_GIT(tmp_path, "", "HEAD") == ["second.txt"]


def test_automatic_git_diff_honors_explicit_verified_base(tmp_path: Path) -> None:
    _git(tmp_path, "init")
    _git(tmp_path, "checkout", "-b", "main")
    _git(tmp_path, "config", "user.email", "tests@example.invalid")
    _git(tmp_path, "config", "user.name", "PTSIP Tests")

    (tmp_path / "base.txt").write_text("base\n", encoding="utf-8")
    _git(tmp_path, "add", "base.txt")
    _git(tmp_path, "commit", "-m", "base")
    base_sha = _git(tmp_path, "rev-parse", "HEAD")

    (tmp_path / "first.txt").write_text("first\n", encoding="utf-8")
    _git(tmp_path, "add", "first.txt")
    _git(tmp_path, "commit", "-m", "first")
    (tmp_path / "second.txt").write_text("second\n", encoding="utf-8")
    _git(tmp_path, "add", "second.txt")
    _git(tmp_path, "commit", "-m", "second")

    assert CHANGED_FILES_FROM_GIT(tmp_path, base_sha, "HEAD") == [
        "first.txt",
        "second.txt",
    ]
