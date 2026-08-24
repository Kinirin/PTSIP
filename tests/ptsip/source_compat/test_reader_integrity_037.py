from __future__ import annotations

import hashlib
from pathlib import Path

import yaml

from ptsip.repository.profile_transition import DraftVersion, ProfileGenerationIdentity
from ptsip.source_compat import V034_REVISION, V036_REVISION, read_source_profile


def _write_profile(root: Path, payload: dict[str, object]) -> ProfileGenerationIdentity:
    raw = yaml.safe_dump(payload, sort_keys=False).encode()
    path = root / "ptsip.yaml"
    path.write_bytes(raw)
    version = str(payload["ptsip"]["version"])
    major, minor, micro = map(int, version.removesuffix("-draft").split("."))
    return ProfileGenerationIdentity(
        path="ptsip.yaml",
        version=DraftVersion(major, minor, micro),
        declared_version=version,
        specification_revision=str(payload["ptsip"]["specification"]["revision"]),
        specification_source=str(payload["ptsip"]["specification"]["source"]),
        content_sha256=hashlib.sha256(raw).hexdigest(),
        temporary=False,
    )


def test_034_overlapping_boundaries_fail_source_integrity(tmp_path: Path) -> None:
    payload = {
        "ptsip": {
            "version": "0.3.4-draft",
            "specification": {
                "source": "https://github.com/Kinirin/PTSIP",
                "revision": V034_REVISION,
            },
        },
        "boundaries": {
            "product": {"roots": ["src"]},
            "toolchain": {"roots": ["src/tools"]},
        },
        "policies": {
            "product_to_toolchain_runtime_dependency": "deny",
            "toolchain_in_product_package": "deny",
            "independent_build_resolution": "required",
        },
    }
    result = read_source_profile(tmp_path, _write_profile(tmp_path, payload))
    assert not result.complete
    assert result.issues[0].code == "SOURCE_BOUNDARY_OVERLAP"


def test_036_hybrid_override_remove_conflict_fails_source_integrity(tmp_path: Path) -> None:
    payload = {
        "ptsip": {
            "version": "0.3.6-draft",
            "specification": {
                "source": "https://github.com/Kinirin/PTSIP",
                "revision": V036_REVISION,
            },
        },
        "responsibility_map": {
            "mode": "hybrid",
            "template": {"id": "base", "revision": "abc"},
            "overrides": {
                "components": [
                    {
                        "id": "same",
                        "classification": "OPERATIONS",
                        "include": ["ops/**"],
                        "purpose": "ops",
                        "shipped": False,
                        "runtime_required": False,
                    }
                ],
                "remove_component_ids": ["same"],
            },
        },
        "policies": {
            "product_to_nonproduct_runtime_dependency": "deny",
            "nonproduct_in_product_package": "deny",
            "independent_build_resolution": "required",
        },
    }
    result = read_source_profile(tmp_path, _write_profile(tmp_path, payload))
    assert not result.complete
    assert result.issues[0].code == "SOURCE_OVERRIDE_REMOVE_CONFLICT"
