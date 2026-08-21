from __future__ import annotations

from vpms.integration.ptsip_bridge import metadata_from_effective_map


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
