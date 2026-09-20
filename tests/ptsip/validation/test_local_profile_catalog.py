from __future__ import annotations

import subprocess
from pathlib import Path

import pytest
import yaml

from ptsip.clarification.resolution.model import DecisionAnswer
from ptsip.clarification.resolution.profile_projection import (
    prepare_local_profile,
    write_prepared_local_profile,
)
from ptsip.local_profile_catalog import LocalProfileCatalogError
from ptsip.validation.profile import find_profile, validate_profile


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
    repo.mkdir()
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")
    (repo / "src").mkdir()
    (repo / "src" / "app.py").write_text("VALUE = 1\n", encoding="utf-8")
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")
    return repo


def _answer() -> DecisionAnswer:
    return DecisionAnswer(
        classification="PRODUCT",
        purpose="product_runtime",
        shipped=True,
        runtime_required=True,
        executable=True,
    )


def test_new_adoption_materializes_under_ptsip_profile_catalog(tmp_path: Path) -> None:
    repo = _repo(tmp_path)

    prepared = prepare_local_profile(repo, "product", ["src/**"], _answer())
    assert prepared.path == repo / ".ptsip" / "profiles" / "main.ptsip.yaml"

    written = write_prepared_local_profile(prepared)
    catalog_path = repo / ".ptsip" / "profiles" / "index.yaml"

    assert written == prepared.path
    assert catalog_path.is_file()
    catalog = yaml.safe_load(catalog_path.read_text(encoding="utf-8"))
    assert catalog["schema_version"] == "ptsip-local-profile-catalog/v1"
    assert catalog["default_profile"] == "main"
    assert catalog["profiles"] == [
        {"id": "main", "resource": "main.ptsip.yaml"}
    ]
    assert find_profile(repo) == written

    validation = validate_profile(repo)
    assert validation.valid, validation.errors
    assert validation.profile_path == str(written)


def test_local_catalog_default_profile_wins_over_legacy_root(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    legacy = repo / "ptsip.yaml"
    legacy.write_text("legacy: true\n", encoding="utf-8")

    profile_root = repo / ".ptsip" / "profiles"
    profile_root.mkdir(parents=True)
    local = profile_root / "alternate.ptsip.yaml"
    local.write_text("local: true\n", encoding="utf-8")
    (profile_root / "index.yaml").write_text(
        yaml.safe_dump(
            {
                "schema_version": "ptsip-local-profile-catalog/v1",
                "default_profile": "alternate",
                "profiles": [
                    {"id": "alternate", "resource": "alternate.ptsip.yaml"}
                ],
            },
            sort_keys=False,
        ),
        encoding="utf-8",
    )

    assert find_profile(repo) == local


def test_existing_legacy_profile_is_not_silently_replaced(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    legacy = repo / "ptsip.yaml"
    prepared = prepare_local_profile(repo, "product", ["src/**"], _answer())
    write_prepared_local_profile(prepared)
    original = prepared.path.read_text(encoding="utf-8")

    legacy.write_text(original, encoding="utf-8")
    (repo / ".ptsip").rename(repo / ".ptsip-disabled")

    next_prepared = prepare_local_profile(repo, "product", ["src/**"], _answer())

    assert next_prepared.path == legacy
    assert next_prepared.catalog_path is None


def test_invalid_local_catalog_fails_closed_instead_of_falling_back(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    (repo / "ptsip.yaml").write_text("legacy: true\n", encoding="utf-8")
    profile_root = repo / ".ptsip" / "profiles"
    profile_root.mkdir(parents=True)
    (profile_root / "index.yaml").write_text(
        "schema_version: ptsip-local-profile-catalog/v1\ndefault_profile: missing\nprofiles: []\n",
        encoding="utf-8",
    )

    with pytest.raises(LocalProfileCatalogError):
        find_profile(repo)

    result = validate_profile(repo)
    assert not result.valid
    assert any("Local profile catalog invalid" in error for error in result.errors)
