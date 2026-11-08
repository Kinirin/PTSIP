"""Regression coverage for validation of materialized policy files."""

import importlib.util
import json
from pathlib import Path
import subprocess

import pytest

from ptsip.policy_contracts import unified


ROOT = Path(__file__).resolve().parents[4]
TARGET = "developer/policy/ARCH/MPD-ARCH-0999.yaml"
SECOND = "developer/policy/ARCH/MPD-ARCH-0998.yaml"


def git(root, *args):
    return subprocess.run(["git", *args], cwd=root, check=True, capture_output=True).stdout


def put(root, path, value):
    destination = root / path
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_text(json.dumps(value), encoding="utf-8")
    return destination


@pytest.fixture
def migration():
    path = ROOT / ".ptsip/engine/developer/policy/migrate_unified.py"
    spec = importlib.util.spec_from_file_location("migration_current_test", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


@pytest.fixture
def repository(tmp_path):
    git(tmp_path, "init", "-q")
    git(tmp_path, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
        "commit", "--allow-empty", "-qm", "Fixture baseline")
    for path in (
        ".ptsip/index.json", ".ptsip/contracts/index.json",
        ".ptsip/contracts/policy/boundary.json", unified.INDEX, unified.SCHEMA,
        unified.AUTHORING_BASIS, unified.EVALUATION_REGISTRY, unified.EVALUATION_INTERFACE,
    ):
        target = tmp_path / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes((ROOT / path).read_bytes())
    for path in ("developer/policy/index.yaml", "src/policy/index.yaml"):
        put(tmp_path, path, {"policies": []})
    return tmp_path


def source():
    return {
        "Policy_Identity": {
            "id": "MPD-ARCH-0999",
            "policy_kind": "PTSIP_DEVELOPER_POLICY",
            "root_family": "ARCH",
            "title": "Current coverage fixture",
        },
        "Policy_Lifecycle": {"status": "DRAFT"},
        "Policy_Rules": {"Definition": [{"id": "original-rule", "rule": "PRESERVE"}]},
    }


def candidate(migration, repository, path=TARGET, registration="UNINDEXED"):
    head = git(repository, "rev-parse", "HEAD").decode().strip()
    return migration.normalize(repository, path, source(), registration, head, unified.VERSION)


def test_migration_preview_and_current_validation_have_distinct_contracts(migration, repository):
    put(repository, TARGET, source())
    git(repository, "add", "--", TARGET)
    preview = migration.run(repository, apply=False)
    assert preview["summary"]["changed_documents"] == 1
    with pytest.raises(ValueError, match="CURRENT_POLICY_NOT_UNIFIED"):
        migration.run(repository, apply=False, verify_current=True)


def test_current_validation_reads_untracked_unindexed_files_without_writes(migration, repository):
    path = put(repository, TARGET, candidate(migration, repository))
    receipt = put(repository, migration.INVENTORY, {"role": "historical migration receipt"})
    before = (path.read_bytes(), receipt.read_bytes())
    report = migration.run(repository, apply=False, verify_current=True)
    assert report["summary"]["UNINDEXED"] == 1
    assert report["summary"]["changed_documents"] == 0
    assert (path.read_bytes(), receipt.read_bytes()) == before
    with pytest.raises(ValueError, match="CURRENT_POLICY_VERIFICATION_IS_READ_ONLY"):
        migration.run(repository, apply=True, verify_current=True)
    put(repository, TARGET, source())
    with pytest.raises(ValueError, match="CURRENT_POLICY_NOT_UNIFIED"):
        migration.run(repository, apply=False, verify_current=True)


def test_current_validation_rejects_registration_and_source_path_drift(migration, repository):
    put(repository, TARGET, candidate(migration, repository, registration="INDEXED"))
    with pytest.raises(ValueError, match="CANDIDATE_SOURCE_REGISTRATION_MISMATCH"):
        migration.run(repository, apply=False, verify_current=True)
    put(repository, TARGET, candidate(migration, repository, path=SECOND))
    with pytest.raises(ValueError, match="CANDIDATE_SOURCE_PATH_MISMATCH"):
        migration.run(repository, apply=False, verify_current=True)


def test_current_validation_rejects_duplicate_policy_ids(migration, repository):
    put(repository, TARGET, candidate(migration, repository))
    put(repository, SECOND, candidate(migration, repository, path=SECOND))
    with pytest.raises(ValueError, match="DUPLICATE_CURRENT_POLICY_ID"):
        migration.run(repository, apply=False, verify_current=True)


def test_current_validation_rejects_cross_plane_index_route(migration, repository):
    path = "src/policy/ARCH/SFP-ARCH-0999.yaml"
    put(repository, "developer/policy/index.yaml", {"policies": [{"path": path, "id": "SFP-ARCH-0999"}]})
    with pytest.raises(ValueError, match="POLICY_INDEX_PLANE_MISMATCH"):
        migration.run(repository, apply=False, verify_current=True)
