from __future__ import annotations

import json
import runpy
from pathlib import Path

import pytest
import yaml

from developer.automation import agent_context_migration
from developer.automation.agent_context_migration import (
    RETIRED_PROFILE_ROOTS,
    verify,
    verify_current_profile_selectors,
    verify_operation_implementation_refs,
)
from ptsip.local_profile_catalog import default_catalog_payload
from ptsip.repository.namespace import default_repository_index_payload


ROOT = Path(__file__).resolve().parents[2]


def test_current_agent_context_candidate_passes_m5_verification() -> None:
    result = verify("M5", ROOT)

    assert result["status"] == "PASS"
    assert all(check["status"] == "PASS" for check in result["checks"])


def _write_profile(path: Path, payload: object | None = None) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    if payload is None:
        payload = {"components": [{"id": "current", "include": ["src/agent_contracts/**"]}]}
    path.write_text(yaml.safe_dump(payload), encoding="utf-8")


def _repo(tmp_path: Path) -> Path:
    (tmp_path / "pyproject.toml").write_text("[project]\nname='synthetic'\n", encoding="utf-8")
    for relative in ("developer/state/index.yaml", "developer/state/repository-state-index.schema.json"):
        destination = tmp_path / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes((ROOT / relative).read_bytes())
    (tmp_path / ".ptsip/profiles").mkdir(parents=True)
    (tmp_path / ".ptsip/index.json").write_text(
        json.dumps(default_repository_index_payload()), encoding="utf-8"
    )
    (tmp_path / ".ptsip/profiles/index.json").write_text(
        json.dumps(default_catalog_payload()), encoding="utf-8"
    )
    _write_profile(tmp_path / "developer/profiles/ptsip-repository.yaml")
    _write_profile(tmp_path / ".ptsip/profiles/main.ptsip.yaml")
    return tmp_path


@pytest.mark.parametrize("retired_root", RETIRED_PROFILE_ROOTS)
@pytest.mark.parametrize("field", ("include", "analysis_inputs", "associated_artifacts"))
def test_retired_selectors_fail_even_when_the_files_reappear(
    tmp_path: Path, retired_root: str, field: str
) -> None:
    root = _repo(tmp_path)
    selector = retired_root if retired_root.endswith((".md", ".yaml")) else retired_root + "/**"
    retired_path = root / retired_root
    if retired_root.endswith((".md", ".yaml")):
        retired_path.write_text("returned legacy content\n", encoding="utf-8")
    else:
        retired_path.mkdir(parents=True)
        (retired_path / "returned.yaml").write_text("legacy: true\n", encoding="utf-8")
    profile = {"components": [{"id": "current", "include": ["src/agent_contracts/**"]}]}
    if field == "associated_artifacts":
        profile[field] = [{"id": "legacy", "include": [selector]}]
        expected_item = "associated_artifacts[legacy].include"
    else:
        profile["components"][0][field] = [selector]
        expected_item = f"components[current].{field}"
    _write_profile(root / "developer/profiles/ptsip-repository.yaml", profile)

    result = verify_current_profile_selectors(root)

    assert result["status"] == "FAIL"
    assert result["detail"]["retired_selectors"] == [{
        "profile": "developer/profiles/ptsip-repository.yaml",
        "item": expected_item,
        "selector": selector,
        "retired_root": retired_root,
    }]


def test_non_default_catalog_profile_is_revalidated(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    catalog = default_catalog_payload()
    catalog["profiles"].append({"id": "review", "resource": "review.ptsip.yaml"})
    (root / ".ptsip/profiles/index.json").write_text(json.dumps(catalog), encoding="utf-8")
    _write_profile(root / ".ptsip/profiles/review.ptsip.yaml", {
        "components": [{"id": "legacy", "include": ["spec/**"]}],
    })

    result = verify_current_profile_selectors(root)

    assert result["status"] == "FAIL"
    assert result["detail"]["retired_selectors"][0]["profile"] == ".ptsip/profiles/review.ptsip.yaml"


def test_current_selectors_preserve_history_and_prose(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    (root / "history.md").write_text("Retired spec/** and MEMORY.md\n", encoding="utf-8")
    _write_profile(root / "developer/profiles/ptsip-repository.yaml", {
        "components": [{
            "id": "current", "include": ["src/agent_contracts/**"],
            "purpose": "Replaces spec/** and MEMORY.md.",
            "analysis_inputs": ["developer/planning/**", ".ptsip/context/**"],
        }],
    })

    assert verify_current_profile_selectors(root)["status"] == "PASS"


@pytest.mark.parametrize("selector", ("./spec/**", "spec\\**", "./docs/planning/**"))
def test_selector_normalization_cannot_bypass_retirement(tmp_path: Path, selector: str) -> None:
    root = _repo(tmp_path)
    _write_profile(root / ".ptsip/profiles/main.ptsip.yaml", {
        "components": [{"id": "legacy", "include": [selector]}],
    })

    result = verify_current_profile_selectors(root)

    assert result["status"] == "FAIL"
    assert result["detail"]["retired_selectors"][0]["selector"] == selector


@pytest.mark.parametrize("relative", (
    "developer/profiles/ptsip-repository.yaml",
    ".ptsip/index.json",
    ".ptsip/profiles/index.json",
    ".ptsip/profiles/main.ptsip.yaml",
))
def test_missing_profile_control_contract_fails_closed(tmp_path: Path, relative: str) -> None:
    root = _repo(tmp_path)
    (root / relative).unlink()

    result = verify_current_profile_selectors(root)

    assert result["status"] == "FAIL"
    assert result["detail"]["errors"]


def test_missing_non_default_profile_fails_closed(tmp_path: Path) -> None:
    root = _repo(tmp_path)
    catalog = default_catalog_payload()
    catalog["profiles"].append({"id": "missing", "resource": "missing.ptsip.yaml"})
    (root / ".ptsip/profiles/index.json").write_text(json.dumps(catalog), encoding="utf-8")

    result = verify_current_profile_selectors(root)

    assert result["status"] == "FAIL"
    assert any("missing.ptsip.yaml" in error for error in result["detail"]["errors"])


@pytest.mark.parametrize("payload", (
    [], {}, {"components": "src/**"},
    {"components": [{}]},
    {"components": [{"id": "current", "include": "src/**"}]},
    {"components": [{"id": "current", "include": ["src/**"], "analysis_inputs": [None]}]},
    {"components": [{"id": "current", "include": ["src/**"]}], "associated_artifacts": {}},
))
def test_malformed_profile_selector_contract_fails_closed(tmp_path: Path, payload: object) -> None:
    root = _repo(tmp_path)
    _write_profile(root / ".ptsip/profiles/main.ptsip.yaml", payload)

    result = verify_current_profile_selectors(root)

    assert result["status"] == "FAIL"
    assert result["detail"]["errors"]


@pytest.mark.parametrize("contract", ("namespace", "catalog"))
def test_malformed_namespace_or_catalog_fails_closed(tmp_path: Path, contract: str) -> None:
    root = _repo(tmp_path)
    if contract == "namespace":
        relative = ".ptsip/index.json"
        payload = default_repository_index_payload()
        payload["namespaces"]["profiles"]["index"] = "../outside.json"
    else:
        relative = ".ptsip/profiles/index.json"
        payload = default_catalog_payload()
        payload["profiles"].append({"id": "outside", "resource": "../outside.ptsip.yaml"})
    (root / relative).write_text(json.dumps(payload), encoding="utf-8")

    result = verify_current_profile_selectors(root)

    assert result["status"] == "FAIL"
    assert result["detail"]["errors"]


def test_auto_verification_revalidates_current_repository_profiles() -> None:
    result = verify("AUTO", ROOT)
    check = next(item for item in result["checks"] if item["id"] == "CURRENT_PROJECT_PROFILE_SELECTORS_REVALIDATED")

    assert result["stage"] == "M8"
    assert check["status"] == "PASS", check["detail"]


def test_auto_verification_fails_on_retired_profile_selector(monkeypatch: pytest.MonkeyPatch) -> None:
    read_yaml = agent_context_migration._yaml

    def profile_with_retired_selector(base: Path, relative: str) -> dict[str, object]:
        payload = read_yaml(base, relative)
        if relative == "developer/profiles/ptsip-repository.yaml":
            payload["components"][0]["analysis_inputs"] = ["docs/planning/**"]
        return payload

    monkeypatch.setattr(agent_context_migration, "_yaml", profile_with_retired_selector)

    result = verify("AUTO", ROOT)
    check = next(item for item in result["checks"] if item["id"] == "CURRENT_PROJECT_PROFILE_SELECTORS_REVALIDATED")

    assert result["stage"] == "M8"
    assert result["status"] == "FAIL"
    assert check["status"] == "FAIL"
    assert check["detail"]["retired_selectors"][0]["selector"] == "docs/planning/**"


def _operation_repository(tmp_path: Path) -> Path:
    (tmp_path / "pyproject.toml").write_text("[project]\nname='fixture'\n", encoding="utf-8")
    contract_root = tmp_path / "src/ptsip/agent_contracts"
    (contract_root / "operations").mkdir(parents=True)
    (contract_root / "index.yaml").write_text(yaml.safe_dump({
        "operations": [{"id": "migrate-profile", "ref": "operations/migrate-profile.yaml"}],
    }), encoding="utf-8")
    _write_operation(tmp_path, ["src/ptsip/migration/analyzer.py"])
    implementation = tmp_path / "src/ptsip/migration/analyzer.py"
    implementation.parent.mkdir(parents=True)
    # Existence checks must never execute the referenced implementation.
    implementation.write_text("raise RuntimeError('must not be imported')\n", encoding="utf-8")
    return tmp_path


def _write_operation(root: Path, refs: object) -> None:
    (root / "src/ptsip/agent_contracts/operations/migrate-profile.yaml").write_text(
        yaml.safe_dump({"operation_id": "PTSIP-OP-MIGRATE-PROFILE-001", "implementation_refs": refs}),
        encoding="utf-8",
    )


def test_current_repository_operation_implementations_exist() -> None:
    result = verify_operation_implementation_refs(ROOT)

    assert result["status"] == "PASS", result["detail"]
    assert result["detail"]["checked"]
    assert result["detail"]["errors"] == []


def test_operation_refs_and_contract_edits_select_repository_validation() -> None:
    resolver = runpy.run_path(str(ROOT / ".github/scripts/resolve_test_modes.py"))
    profile = yaml.safe_load((ROOT / "developer/profiles/ptsip-repository.yaml").read_text())
    registry = yaml.safe_load((ROOT / ".github/test_modes.yaml").read_text())
    refs = verify_operation_implementation_refs(ROOT)["detail"]["checked"]
    paths = {item["ref"] for item in refs}
    paths.add("src/ptsip/agent_contracts/operations/migrate-profile.yaml")

    for path in paths:
        modes, _ = resolver["resolve_automatic_selection"](registry, profile, [path])
        assert "repository-architecture" in {mode["id"] for mode in modes}, path


def test_operation_ref_check_reports_deleted_and_renamed_implementation(tmp_path: Path) -> None:
    root = _operation_repository(tmp_path)
    implementation = root / "src/ptsip/migration/analyzer.py"
    implementation.rename(implementation.with_name("analyzer_new.py"))

    result = verify_operation_implementation_refs(root)

    assert result["status"] == "FAIL"
    error = result["detail"]["errors"][0]
    assert "operations/migrate-profile.yaml" in error
    assert "PTSIP-OP-MIGRATE-PROFILE-001" in error
    assert "src/ptsip/migration/analyzer.py" in error

    _write_operation(root, ["src/ptsip/migration/analyzer_new.py"])
    assert verify_operation_implementation_refs(root)["status"] == "PASS"


def test_relocated_migration_sources_select_binding_and_repository_validation() -> None:
    resolver = runpy.run_path(str(ROOT / ".github/scripts/resolve_test_modes.py"))
    profile = yaml.safe_load((ROOT / "developer/profiles/ptsip-repository.yaml").read_text())
    registry = yaml.safe_load((ROOT / ".github/test_modes.yaml").read_text())
    sources = sorted((ROOT / "src/ptsip/migration").rglob("*.py"))
    assert sources

    for source in sources:
        path = source.relative_to(ROOT).as_posix()
        modes, _ = resolver["resolve_automatic_selection"](registry, profile, [path])
        selected = {mode["id"] for mode in modes}
        assert {"agent-contract-plane", "repository-architecture"} <= selected, path


@pytest.mark.parametrize("reference", (
    "../outside.py", "/tmp/outside.py", "C:/outside.py", "C:outside.py",
    "src\\ptsip\\migration\\analyzer.py", "src/ptsip/migration/*.py", "https://example.com/code.py",
))
def test_operation_implementation_refs_reject_unsafe_paths(tmp_path: Path, reference: str) -> None:
    root = _operation_repository(tmp_path)
    _write_operation(root, [reference])

    result = verify_operation_implementation_refs(root)

    assert result["status"] == "FAIL"
    assert "unsafe ref" in result["detail"]["errors"][0]


@pytest.mark.parametrize("refs", (None, [], "src/ptsip/migration/analyzer.py", [None], [""]))
def test_malformed_implementation_refs_fail_closed(tmp_path: Path, refs: object) -> None:
    root = _operation_repository(tmp_path)
    _write_operation(root, refs)

    result = verify_operation_implementation_refs(root)

    assert result["status"] == "FAIL"
    assert result["detail"]["errors"]


def test_implementation_ref_must_name_a_file(tmp_path: Path) -> None:
    root = _operation_repository(tmp_path)
    _write_operation(root, ["src/ptsip/migration"])

    assert verify_operation_implementation_refs(root)["status"] == "FAIL"


def test_implementation_symlink_cannot_escape_repository(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    root.mkdir()
    _operation_repository(root)
    outside = tmp_path / "outside.py"
    outside.write_text("pass\n", encoding="utf-8")
    link = root / "src/ptsip/migration/outside.py"
    try:
        link.symlink_to(outside)
    except (OSError, NotImplementedError):
        pytest.skip("symlink creation is unavailable in this environment")
    _write_operation(root, ["src/ptsip/migration/outside.py"])

    assert verify_operation_implementation_refs(root)["status"] == "FAIL"


def test_operation_index_cannot_follow_external_symlink(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    root.mkdir()
    _operation_repository(root)
    outside = tmp_path / "outside.yaml"
    outside.write_text("operations: []\n", encoding="utf-8")
    index = root / "src/ptsip/agent_contracts/index.yaml"
    index.unlink()
    try:
        index.symlink_to(outside)
    except (OSError, NotImplementedError):
        pytest.skip("symlink creation is unavailable in this environment")

    result = verify_operation_implementation_refs(root)

    assert result["status"] == "FAIL"
    assert result["detail"]["checked"] == []


@pytest.mark.parametrize("operations", ([], None, [{"ref": "../outside.yaml"}], [{"ref": None}]))
def test_malformed_operation_index_fails_closed(tmp_path: Path, operations: object) -> None:
    root = _operation_repository(tmp_path)
    (root / "src/ptsip/agent_contracts/index.yaml").write_text(
        yaml.safe_dump({"operations": operations}), encoding="utf-8"
    )

    assert verify_operation_implementation_refs(root)["status"] == "FAIL"


def test_unindexed_operation_cannot_escape_implementation_validation(tmp_path: Path) -> None:
    root = _operation_repository(tmp_path)
    (root / "src/ptsip/agent_contracts/operations/unindexed.yaml").write_text(
        "implementation_refs: [missing.py]\n", encoding="utf-8"
    )

    result = verify_operation_implementation_refs(root)

    assert result["status"] == "FAIL"
    assert "unindexed.yaml" in result["detail"]["errors"][0]


def test_auto_verification_fails_when_implementation_ref_is_missing(monkeypatch: pytest.MonkeyPatch) -> None:
    read_yaml = agent_context_migration._yaml

    def operation_with_missing_implementation(base: Path, relative: str) -> dict[str, object]:
        payload = read_yaml(base, relative)
        if relative == "src/ptsip/agent_contracts/operations/migrate-profile.yaml":
            payload["implementation_refs"] = ["src/ptsip/migration/deleted_implementation.py"]
        return payload

    monkeypatch.setattr(agent_context_migration, "_yaml", operation_with_missing_implementation)

    result = verify("AUTO", ROOT)
    check = next(item for item in result["checks"] if item["id"] == "OPERATION_IMPLEMENTATION_REFS_EXIST")

    assert result["status"] == "FAIL"
    assert check["status"] == "FAIL"
    assert "deleted_implementation.py" in check["detail"]["errors"][0]
