from __future__ import annotations

import ast
from dataclasses import FrozenInstanceError
from pathlib import Path

import pytest

from vpms.domain.model import (
    FormulaRef,
    PolicyRef,
    RunnerRef,
    TargetRef,
    VariablesRef,
    VerificationCase,
    VerificationPurpose,
)
from vpms.integration.ptsip_bridge import (
    PtsipMetadataError,
    PtsipTargetMetadata,
    load_ptsip_metadata,
    resolve_target_metadata,
)


_REPO_ROOT = Path(__file__).resolve().parents[3]


def _write_profile(tmp_path: Path, content: str) -> Path:
    path = tmp_path / "ptsip.yaml"
    path.write_text(content, encoding="utf-8")
    return path


def test_bridge_reads_only_minimal_target_metadata_deterministically(tmp_path: Path) -> None:
    profile = _write_profile(
        tmp_path,
        """
components:
  - id: z-tool
    classification: TOOLCHAIN
    purpose: ignored-by-vpms-bridge
    shipped: false
  - id: a-product
    classification: PRODUCT
    runtime_required: true
""".strip(),
    )

    snapshot = load_ptsip_metadata(profile)

    assert snapshot.as_dict() == {
        "targets": [
            {"component_id": "a-product", "classification": "PRODUCT"},
            {"component_id": "z-tool", "classification": "TOOLCHAIN"},
        ]
    }


def test_target_resolution_uses_component_identity_only(tmp_path: Path) -> None:
    profile = _write_profile(
        tmp_path,
        """
components:
  - id: product-api
    classification: PRODUCT
""".strip(),
    )
    snapshot = load_ptsip_metadata(profile)

    assert resolve_target_metadata(
        TargetRef(component_id="product-api"), snapshot
    ) == PtsipTargetMetadata(component_id="product-api", classification="PRODUCT")
    assert resolve_target_metadata(TargetRef(component_id="missing"), snapshot) is None


def test_ptsip_classification_does_not_determine_vpms_purpose(tmp_path: Path) -> None:
    profile = _write_profile(
        tmp_path,
        """
components:
  - id: verifier-sdk
    classification: TOOLCHAIN
""".strip(),
    )
    snapshot = load_ptsip_metadata(profile)
    case = VerificationCase(
        id="product.behavior",
        purpose=VerificationPurpose.PRODUCT,
        target=TargetRef(component_id="verifier-sdk"),
        formula=FormulaRef(ref="formula"),
        variables=VariablesRef(ref="variables"),
        policy=PolicyRef(ref="policy"),
        runner=RunnerRef(ref="runner"),
    )

    metadata = resolve_target_metadata(case.target, snapshot)

    assert metadata is not None
    assert metadata.classification == "TOOLCHAIN"
    assert case.purpose is VerificationPurpose.PRODUCT


def test_bridge_preserves_forward_compatible_ptsip_classification_text(tmp_path: Path) -> None:
    profile = _write_profile(
        tmp_path,
        """
components:
  - id: shared-contract
    classification: NEUTRAL_CONTRACT
""".strip(),
    )

    metadata = resolve_target_metadata(
        TargetRef(component_id="shared-contract"),
        load_ptsip_metadata(profile),
    )

    assert metadata is not None
    assert metadata.classification == "NEUTRAL_CONTRACT"


def test_bridge_is_read_only_and_returns_immutable_metadata(tmp_path: Path) -> None:
    profile = _write_profile(
        tmp_path,
        """
components:
  - id: product-api
    classification: PRODUCT
""".strip(),
    )
    before = profile.read_bytes()

    snapshot = load_ptsip_metadata(profile)
    target = snapshot.targets[0]

    assert profile.read_bytes() == before
    with pytest.raises(FrozenInstanceError):
        target.classification = "TOOLCHAIN"  # type: ignore[misc]


def test_bridge_has_no_ptsip_runtime_import_or_write_api() -> None:
    bridge_path = _REPO_ROOT / "src" / "vpms" / "integration" / "ptsip_bridge.py"
    tree = ast.parse(bridge_path.read_text(encoding="utf-8"))
    imports = []

    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            imports.extend(alias.name for alias in node.names)
        elif isinstance(node, ast.ImportFrom) and node.module:
            imports.append(node.module)

    public_functions = [
        node.name
        for node in tree.body
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef))
        and not node.name.startswith("_")
    ]

    assert not any(name == "ptsip" or name.startswith("ptsip.") for name in imports)
    assert public_functions == [
        "metadata_from_effective_map",
        "load_ptsip_metadata",
        "resolve_target_metadata",
    ]


@pytest.mark.parametrize(
    ("content", "message"),
    [
        ("[]", "root must be a mapping"),
        ("components: {}", "components must be a list"),
        ("components:\n  - bad", "must be a mapping"),
        ("components:\n  - id: ''\n    classification: PRODUCT", "invalid id"),
        ("components:\n  - id: a\n    classification: ''", "invalid classification"),
        (
            "components:\n  - id: a\n    classification: PRODUCT\n"
            "  - id: a\n    classification: TOOLCHAIN",
            "Duplicate PTSIP component id",
        ),
    ],
)
def test_malformed_minimal_contract_fails_explicitly(
    tmp_path: Path, content: str, message: str
) -> None:
    profile = _write_profile(tmp_path, content)

    with pytest.raises(PtsipMetadataError, match=message):
        load_ptsip_metadata(profile)


def test_profile_without_components_is_valid_empty_metadata_snapshot(tmp_path: Path) -> None:
    profile = _write_profile(
        tmp_path,
        """
ptsip:
  version: 0.3.4-draft
boundaries:
  PRODUCT:
    roots: [product]
""".strip(),
    )

    assert load_ptsip_metadata(profile).targets == ()
