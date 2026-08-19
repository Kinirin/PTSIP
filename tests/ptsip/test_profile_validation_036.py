from __future__ import annotations

import copy
import subprocess
from pathlib import Path

import yaml

from ptsip.constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ptsip.validation.profile import validate_profile
from ptsip.validation.templates import template_catalog


POLICIES = {
    "product_to_nonproduct_runtime_dependency": "deny",
    "nonproduct_in_product_package": "deny",
    "independent_build_resolution": "required",
}


def _git(repo: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


def _repo(tmp_path: Path, files: dict[str, str]) -> Path:
    repo = tmp_path / "repo"
    repo.mkdir(parents=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")
    for relative, content in files.items():
        path = repo / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")
    return repo


def _write_profile(repo: Path, payload: dict[str, object]) -> Path:
    profile = repo / "ptsip.yaml"
    profile.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")
    return profile


def _base(mode: str, template_id: str | None = None, revision: str | None = None) -> dict[str, object]:
    responsibility_map: dict[str, object] = {"mode": mode}
    if template_id is not None and revision is not None:
        responsibility_map["template"] = {"id": template_id, "revision": revision}
    return {
        "ptsip": {
            "version": SPEC_VERSION,
            "specification": {
                "source": SPEC_SOURCE,
                "revision": SPEC_REVISION,
            },
        },
        "responsibility_map": responsibility_map,
        "policies": copy.deepcopy(POLICIES),
    }


def test_template_profile_is_fully_validated_against_effective_map(tmp_path: Path) -> None:
    definition = template_catalog()[0]
    repo = _repo(
        tmp_path,
        {
            "src/package.py": "VALUE = 1\n",
            "tests/test_package.py": "def test_value():\n    assert True\n",
        },
    )
    payload = _base("template", definition.id, definition.revision)
    profile = _write_profile(repo, payload)
    before = profile.read_text(encoding="utf-8")

    result = validate_profile(repo)

    assert result.valid, result.errors
    assert not any("materialization" in warning.lower() for warning in result.warnings)
    assert result.details is not None
    resolution = result.details["resolution"]
    assert resolution["source_mode"] == "template"
    assert resolution["template"] == {"id": definition.id, "revision": definition.revision}
    assert str(resolution["effective_map_digest"]).startswith("sha256:")
    partition = result.details["component_partition"]
    owners = {item["path"]: item["component_id"] for item in partition["assignments"]}
    assert owners["src/package.py"] == "package"
    assert owners["tests/test_package.py"] == "package-tests"
    assert result.details["responsibility_map_coverage"]["unassigned_count"] == 0
    assert profile.read_text(encoding="utf-8") == before


def test_hybrid_validation_uses_override_and_removal_effective_map(tmp_path: Path) -> None:
    definition = template_catalog()[0]
    repo = _repo(tmp_path, {"lib/package.py": "VALUE = 1\n"})
    payload = _base("hybrid", definition.id, definition.revision)
    responsibility_map = payload["responsibility_map"]
    assert isinstance(responsibility_map, dict)
    responsibility_map["overrides"] = {
        "components": [
            {
                "id": "package",
                "classification": "PRODUCT",
                "roles": ["IMPLEMENTATION"],
                "include": ["lib/**"],
                "purpose": "custom_package_layout",
                "shipped": True,
                "runtime_required": True,
                "executable": True,
            }
        ],
        "remove_component_ids": ["package-tests"],
        "remove_relationship_ids": ["package-tests-verify-package"],
    }
    _write_profile(repo, payload)

    result = validate_profile(repo)

    assert result.valid, result.errors
    assert result.details is not None
    assert result.details["resolution"]["source_mode"] == "hybrid"
    provenance = result.details["resolution_provenance"]
    assert provenance["components"]["package"] == "PROJECT_OVERRIDE"
    assert provenance["removals"]["components"] == ["package-tests"]
    assert provenance["removals"]["relationships"] == ["package-tests-verify-package"]
    partition = result.details["component_partition"]
    owners = {item["path"]: item["component_id"] for item in partition["assignments"]}
    assert owners == {"lib/package.py": "package"}
    assert result.details["responsibility_map_coverage"]["unassigned_count"] == 0


def test_unknown_template_revision_is_fail_closed_validation_error(tmp_path: Path) -> None:
    definition = template_catalog()[0]
    repo = _repo(tmp_path, {"src/package.py": "VALUE = 1\n"})
    payload = _base("template", definition.id, "sha256:" + "0" * 64)
    _write_profile(repo, payload)

    result = validate_profile(repo)

    assert not result.valid
    assert any("materialization failed" in error for error in result.errors)
    assert any("Unknown revision" in error for error in result.errors)


def test_hybrid_removal_that_leaves_dangling_effective_relationship_fails(tmp_path: Path) -> None:
    definition = template_catalog()[0]
    repo = _repo(tmp_path, {"src/package.py": "VALUE = 1\n"})
    payload = _base("hybrid", definition.id, definition.revision)
    responsibility_map = payload["responsibility_map"]
    assert isinstance(responsibility_map, dict)
    responsibility_map["overrides"] = {
        "remove_component_ids": ["package-tests"],
    }
    _write_profile(repo, payload)

    result = validate_profile(repo)

    assert not result.valid
    assert any(
        "endpoint 'package-tests' is not declared" in error
        for error in result.errors
    )
    assert result.details is not None
    assert result.details["resolution"]["source_mode"] == "hybrid"


def test_equal_explicit_and_template_maps_share_validation_digest_and_partition(tmp_path: Path) -> None:
    definition = template_catalog()[0]
    repo = _repo(
        tmp_path,
        {
            "src/package.py": "VALUE = 1\n",
            "tests/test_package.py": "def test_value():\n    assert True\n",
        },
    )

    template_payload = _base("template", definition.id, definition.revision)
    _write_profile(repo, template_payload)
    template_result = validate_profile(repo)
    assert template_result.valid, template_result.errors
    assert template_result.details is not None

    explicit_payload = _base("explicit")
    explicit_payload.update(copy.deepcopy(definition.map_payload))
    _write_profile(repo, explicit_payload)
    explicit_result = validate_profile(repo)
    assert explicit_result.valid, explicit_result.errors
    assert explicit_result.details is not None

    assert template_result.details["resolution"]["source_mode"] == "template"
    assert explicit_result.details["resolution"]["source_mode"] == "explicit"
    assert (
        template_result.details["resolution"]["effective_map_digest"]
        == explicit_result.details["resolution"]["effective_map_digest"]
    )
    assert (
        template_result.details["component_partition"]
        == explicit_result.details["component_partition"]
    )
    assert (
        template_result.details["responsibility_map_coverage"]
        == explicit_result.details["responsibility_map_coverage"]
    )
