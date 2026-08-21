from __future__ import annotations

import ast
from pathlib import Path

from vpms.integration.ptsip_bridge import metadata_from_effective_map


_REPO_ROOT = Path(__file__).resolve().parents[3]


def test_explicit_effective_map_projects_vpms_target_metadata() -> None:
    effective_payload = {
        "components": [
            {
                "id": "z-verifier",
                "classification": "DEVELOPMENT_TOOLING",
                "purpose": "verification implementation",
                "roles": ["VERIFICATION"],
                "shipped": False,
            },
            {
                "id": "a-product",
                "classification": "PRODUCT",
                "runtime_required": True,
            },
        ],
        "associated_artifacts": [],
        "relationships": [],
        "component_dependency_policy": {},
        "policies": {},
    }

    snapshot = metadata_from_effective_map(effective_payload)

    assert snapshot.as_dict() == {
        "targets": [
            {"component_id": "a-product", "classification": "PRODUCT"},
            {
                "component_id": "z-verifier",
                "classification": "DEVELOPMENT_TOOLING",
            },
        ]
    }


def test_vpms_effective_map_bridge_does_not_import_ptsip_runtime() -> None:
    bridge_path = _REPO_ROOT / "src" / "vpms" / "integration" / "ptsip_bridge.py"
    tree = ast.parse(bridge_path.read_text(encoding="utf-8"), filename=str(bridge_path))
    imports: list[str] = []

    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            imports.extend(alias.name for alias in node.names)
        elif isinstance(node, ast.ImportFrom) and node.module:
            imports.append(node.module)

    assert not any(name == "ptsip" or name.startswith("ptsip.") for name in imports)
