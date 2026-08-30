from __future__ import annotations

import runpy
import tomllib
from pathlib import Path

from ptsip.constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION, TOOL_VERSION
from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION


ROOT = Path(__file__).resolve().parents[2]
EXPECTED_SPEC_REVISION = "3c47816770d194ae42f98faedc911d980db0e62a"
HISTORICAL_036_REVISION = "d6995ed232e845b88d8235b851e80ab54b7804ea"


def test_tool_037_package_runtime_pp_and_spec_binding_match() -> None:
    payload = tomllib.loads((ROOT / "pyproject.toml").read_text(encoding="utf-8"))
    assert payload["project"]["version"] == "0.3.7"
    assert TOOL_VERSION == "0.3.7"
    assert CURRENT_PROJECT_PROFILE_VERSION == "pp.1.01"
    assert SPEC_VERSION == "0.3.7-draft"
    assert SPEC_SOURCE == "https://github.com/Kinirin/PTSIP"
    assert SPEC_REVISION == EXPECTED_SPEC_REVISION


def test_release_workflow_derives_tool_tag_from_package_version() -> None:
    workflow = (ROOT / ".github" / "workflows" / "tooling-release.yml").read_text(encoding="utf-8")
    assert '$expectedTag = "tool-v$packageVersion"' in workflow
    assert "py -3.14" in workflow
    assert "actions/setup-python@" not in workflow
    assert "python -m build" in workflow
    assert "python -m twine check $distFiles" in workflow
    assert "Version:\\s*0\\.3\\.7" in workflow
    assert "ptsip-profile-pp-1.01.schema.json" in workflow
    assert "ptsip-normalized-evidence.schema.json" in workflow
    assert "Verify publication Product Artifact evidence and exact snapshot binding" in workflow
    assert "ptsip-artifact-evidence/v1" in workflow
    assert "ptsip-artifact-evidence-binding/v1" in workflow
    assert "producer_component = 'repository-release-automation'" in workflow
    assert "artifact_snapshot_binding" in workflow
    assert "PTSIP-PKG-001" in workflow
    assert "$conformExit -notin @(0, 6)" in workflow
    assert "--force-reinstall --no-deps" in workflow
    assert "pypa/gh-action-pypi-publish@release/v1" in workflow


def test_routine_ci_supports_selective_modes_and_preserves_full_exact_sha() -> None:
    workflow = (ROOT / ".github" / "workflows" / "tooling-test.yml").read_text(encoding="utf-8")
    assert "workflow_dispatch:" in workflow
    assert "inputs:" in workflow
    assert "scope:" in workflow
    assert "default: selective" in workflow
    assert "mode:" in workflow
    assert "ptsip-migration" in workflow
    assert "ptsip-evidence" in workflow
    assert "ptsip-source-compat" in workflow
    assert "vpms" in workflow
    assert "ref: ${{ github.sha }}" in workflow
    assert "py -3.14" in workflow
    assert "actions/setup-python@" not in workflow
    assert "Resolve and run selected Test Modes" in workflow
    assert "resolve_test_modes.py manual --mode $env:TEST_MODE" in workflow
    assert "& python -m pytest -q @targets" in workflow
    assert (
        "      - name: Resolve and run selected Test Modes\n"
        "        if: ${{ inputs.scope == 'selective' }}"
    ) in workflow
    assert (
        "      - name: Run complete repository regression\n"
        "        if: ${{ inputs.scope == 'full' }}"
    ) in workflow
    assert "python -m pytest -q" in workflow
    assert "python -m build" in workflow
    assert "python -m twine check $distFiles" in workflow
    assert "python .github/scripts/verify_release_contract.py" in workflow
    assert "Version:\\s*0\\.3\\.7" in workflow
    assert "ptsip-profile-pp-1.01.schema.json" in workflow
    assert "Verify Product Artifact evidence and exact snapshot binding" in workflow
    assert "ptsip-artifact-evidence/v1" in workflow
    assert "ptsip-artifact-evidence-binding/v1" in workflow
    assert "--artifact-evidence" in workflow
    assert "artifact_snapshot_binding" in workflow
    assert "wheel-sha256:" in workflow
    assert "PTSIP-PKG-001" in workflow
    assert "$conformExit -notin @(0, 6)" in workflow
    assert 'Write-Host "Artifact-aware conformance outcome: $($report.outcome)"\n          exit 0' in workflow
    assert "--force-reinstall --no-deps" in workflow
    assert (
        "      - name: Record successful exact-SHA tooling verification\n"
        "        if: ${{ inputs.scope == 'full' }}"
    ) in workflow
    assert 'context = "self-hosted/tooling-test"' in workflow
    assert "ptsip --version" in workflow
    assert "ptsip spec" in workflow
    assert "ptsip conform --help" in workflow


def test_release_preparation_derives_identity_without_manual_inputs() -> None:
    workflow = (ROOT / ".github" / "workflows" / "release.yml").read_text(encoding="utf-8")
    assert "workflow_dispatch:" in workflow
    assert "inputs:" not in workflow
    assert "ref: ${{ github.sha }}" in workflow
    assert 'DISPATCHED_REF: ${{ github.ref }}' in workflow
    assert 'refs/heads/main' in workflow
    assert 'self-hosted/tooling-test' in workflow
    assert "Reconfirm candidate remains current main" in workflow
    assert 'target_commitish = $env:SOURCE_SHA' in workflow
    assert 'draft = $true' in workflow
    assert 'origin/main moved to $mainSha after tooling verification' in workflow
    assert "py -3.14" in workflow
    assert "actions/setup-python@" not in workflow
    assert '$note = "releasenote/tool/$packageVersion.md"' in workflow


def test_release_package_contains_bound_machine_readable_contracts() -> None:
    specdata = ROOT / "src" / "ptsip" / "specdata"
    for name in (
        "ptsip-profile.schema.json",
        "ptsip-profile-pp-1.01.schema.json",
        "ptsip-registry.yaml",
        "ptsip-artifact-evidence.schema.json",
        "ptsip-agent-classification.schema.json",
        "ptsip-diagnostic.schema.json",
        "ptsip-normalized-evidence.schema.json",
    ):
        assert (specdata / name).is_file()


def test_canonical_and_embedded_machine_readable_contracts_are_identical() -> None:
    pairs = (
        ("schemas/ptsip-profile.schema.json", "src/ptsip/specdata/ptsip-profile.schema.json"),
        ("schemas/ptsip-profile-pp-1.01.schema.json", "src/ptsip/specdata/ptsip-profile-pp-1.01.schema.json"),
        ("registry/ptsip-registry.yaml", "src/ptsip/specdata/ptsip-registry.yaml"),
        ("schemas/ptsip-artifact-evidence.schema.json", "src/ptsip/specdata/ptsip-artifact-evidence.schema.json"),
        ("schemas/ptsip-agent-classification.schema.json", "src/ptsip/specdata/ptsip-agent-classification.schema.json"),
        ("schemas/ptsip-diagnostic.schema.json", "src/ptsip/specdata/ptsip-diagnostic.schema.json"),
        ("schemas/ptsip-normalized-evidence.schema.json", "src/ptsip/specdata/ptsip-normalized-evidence.schema.json"),
    )
    for canonical, embedded in pairs:
        assert (ROOT / canonical).read_bytes() == (ROOT / embedded).read_bytes(), canonical


def _release_contract_namespace() -> dict[str, object]:
    return runpy.run_path(str(ROOT / ".github" / "scripts" / "verify_release_contract.py"))


def test_release_contract_requires_full_037_normative_snapshot() -> None:
    release_contract = (ROOT / ".github" / "scripts" / "verify_release_contract.py").read_text(encoding="utf-8")
    for path in (
        "spec/PTSIP-SPEC.md",
        "spec/PTSIP-CONFORMANCE.md",
        "spec/PTSIP-TERMINOLOGY.md",
        "spec/PTSIP-GOVERNANCE.md",
        "spec/PTSIP-RESPONSIBILITY-MAP.md",
        "spec/PTSIP-DRAFT-PROFILE-TRANSITION.md",
        "schemas/ptsip-profile.schema.json",
        "schemas/ptsip-profile-pp-1.01.schema.json",
        "schemas/ptsip-artifact-evidence.schema.json",
        "schemas/ptsip-agent-classification.schema.json",
        "schemas/ptsip-diagnostic.schema.json",
        "schemas/ptsip-normalized-evidence.schema.json",
        "registry/ptsip-registry.yaml",
        "src/ptsip/specdata/ptsip-profile.schema.json",
        "src/ptsip/specdata/ptsip-profile-pp-1.01.schema.json",
        "src/ptsip/specdata/ptsip-artifact-evidence.schema.json",
        "src/ptsip/specdata/ptsip-agent-classification.schema.json",
        "src/ptsip/specdata/ptsip-diagnostic.schema.json",
        "src/ptsip/specdata/ptsip-normalized-evidence.schema.json",
        "src/ptsip/specdata/ptsip-registry.yaml",
    ):
        assert path in release_contract

    assert 'head_object != bound_object' in release_contract
    assert 'canonical_object != embedded_object' in release_contract

    spec = (ROOT / "spec" / "PTSIP-SPEC.md").read_text(encoding="utf-8")
    map_spec = (ROOT / "spec" / "PTSIP-RESPONSIBILITY-MAP.md").read_text(encoding="utf-8")
    transition_spec = (ROOT / "spec" / "PTSIP-DRAFT-PROFILE-TRANSITION.md").read_text(
        encoding="utf-8"
    )
    spec_note = (
        ROOT / "releasenote" / "specification" / "0.3.7-draft.md"
    ).read_text(encoding="utf-8")
    assert "0.3.6-draft" in spec
    assert "DEVELOPMENT_TOOLING" in spec
    assert "DELIVERY" in spec
    assert "OPERATIONS" in spec
    assert "PTSIP-RMAP-012" in map_spec
    assert "Explicit Specification binding and capability authority" in transition_spec
    assert EXPECTED_SPEC_REVISION in spec_note

    assert 'expected_spec_version = f"{package_version}-draft"' not in release_contract
    assert 'profile_ptsip.get("version") != spec_version' not in release_contract


def test_release_documents_record_current_tool_and_spec_binding() -> None:
    tool_note = (ROOT / "releasenote" / "tool" / "0.3.7.md").read_text(encoding="utf-8")
    pp_note = (ROOT / "releasenote" / "project-profile" / "pp.1.01.md").read_text(
        encoding="utf-8"
    )
    release_index = (ROOT / "releasenote" / "README.md").read_text(encoding="utf-8")
    assert "0.3.7" in tool_note
    assert "pp.1.01" in tool_note
    assert "0.3.7-draft" in tool_note
    assert EXPECTED_SPEC_REVISION in tool_note
    assert "\n## " in tool_note
    assert EXPECTED_SPEC_REVISION in pp_note
    assert "tool/0.3.7.md" in release_index
    assert "project-profile/pp.1.01.md" in release_index
    assert "specification/0.3.7-draft.md" in release_index


def test_historical_036_operational_context_remains_exact() -> None:
    stale_active_marker = "WU-04G  clarification/adoption integration               ACTIVE"
    for path in ("MEMORY.md", "AGENTS.md"):
        text = (ROOT / path).read_text(encoding="utf-8")
        assert "WU-07" in text, path
        assert "Release Contract Strengthening" in text, path
        assert HISTORICAL_036_REVISION in text, path
        assert stale_active_marker not in text, path


def test_release_contract_accepts_current_exact_bound_assets() -> None:
    namespace = _release_contract_namespace()
    bound_paths = namespace["RELEASE_BOUND_SPEC_PATHS"]
    assert isinstance(bound_paths, tuple)
    assert len(bound_paths) == 20
    main = namespace["main"]
    assert callable(main)
    assert main() == 0


def test_release_contract_rejects_bound_asset_blob_drift() -> None:
    namespace = _release_contract_namespace()
    main = namespace["main"]
    assert callable(main)
    globals_dict = main.__globals__
    original_object_id = globals_dict["_git_object_id"]

    def fake_object_id(revision: str, path: str) -> str | None:
        if path == "spec/PTSIP-SPEC.md":
            return "head-drift" if revision == "HEAD" else "bound-object"
        return original_object_id(revision, path)

    globals_dict["_git_object_id"] = fake_object_id
    assert main() == 1
