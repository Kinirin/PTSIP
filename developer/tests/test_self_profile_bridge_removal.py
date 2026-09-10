from __future__ import annotations

import runpy
from pathlib import Path

from ptsip.clarification.generator import analyze_clarifications
from ptsip.validation.profile import validate_profile


REPO_ROOT = Path(__file__).resolve().parents[2]
SELF_PROFILE = REPO_ROOT / "developer" / "profiles" / "ptsip-repository.yaml"
ROOT_BRIDGE = REPO_ROOT / "ptsip.yaml"
TEST_MODE_VALIDATOR = REPO_ROOT / ".github" / "scripts" / "validate_test_modes.py"
TEST_MODE_RESOLVER = REPO_ROOT / ".github" / "scripts" / "resolve_test_modes.py"
TEST_MODE_REGISTRY = REPO_ROOT / ".github" / "test_modes.yaml"
TOOLING_TEST_WORKFLOW = REPO_ROOT / ".github" / "workflows" / "tooling-test.yml"


def test_repository_self_profile_is_explicit_and_root_bridge_is_absent() -> None:
    assert SELF_PROFILE.is_file()
    assert not ROOT_BRIDGE.exists()

    result = validate_profile(REPO_ROOT, SELF_PROFILE)
    assert result.valid, result.errors
    assert result.errors == []
    assert result.warnings == []
    assert result.profile_path == str(SELF_PROFILE)

    analysis = analyze_clarifications(REPO_ROOT, profile_path=SELF_PROFILE)
    assert analysis.status == "NO_CLARIFICATION_REQUIRED"
    assert analysis.requests == ()
    assert analysis.profile_path == str(SELF_PROFILE)


def test_repository_test_mode_control_plane_defaults_to_explicit_self_profile() -> None:
    validator = runpy.run_path(str(TEST_MODE_VALIDATOR))
    resolver = runpy.run_path(str(TEST_MODE_RESOLVER))

    validator_parser = validator["build_parser"]()
    validator_args = validator_parser.parse_args([])
    assert validator_args.profile == "developer/profiles/ptsip-repository.yaml"

    resolver_parser = resolver["build_parser"]()
    resolver_args = resolver_parser.parse_args(["manual"])
    assert resolver_args.profile == "developer/profiles/ptsip-repository.yaml"

    errors = validator["validate_registry"](TEST_MODE_REGISTRY, SELF_PROFILE, REPO_ROOT)
    assert errors == []


def test_current_repository_profile_has_no_root_bridge_dependency() -> None:
    text = SELF_PROFILE.read_text(encoding="utf-8")
    assert '"ptsip.yaml"' not in text
    assert "developer/profiles/ptsip-repository.yaml" in text


def test_full_ci_self_management_commands_select_the_developer_profile() -> None:
    text = TOOLING_TEST_WORKFLOW.read_text(encoding="utf-8")
    profile_arg = "--profile developer/profiles/ptsip-repository.yaml"

    assert f"ptsip validate . {profile_arg} --json" in text
    assert f"ptsip clarify . {profile_arg} --json" in text
    assert f"ptsip gate . {profile_arg} --coordination local --json" in text
    assert f"ptsip conform . {profile_arg} --artifact-evidence $evidencePath --json" in text
