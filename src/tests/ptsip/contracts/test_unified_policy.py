import copy
import hashlib
import json
from pathlib import Path

import pytest
import yaml

from ptsip._common_contract import PolicySourceError, request, source_result


ROOT = Path(__file__).resolve().parents[4]
RELATIVE = "CNTR/SFP-CNTR-0009.yaml"


@pytest.fixture
def document():
    return yaml.safe_load((ROOT / "src/policy" / RELATIVE).read_bytes())


def test_go_restores_canonical_source_and_support_projection(document):
    original = source_result("restore-source", "source", document=document)
    canonical = json.dumps(original, ensure_ascii=True, sort_keys=True, separators=(",", ":"), allow_nan=False)
    assert hashlib.sha256(canonical.encode("ascii")).hexdigest() == document["Policy_Content"]["source_document"]["sha256"]
    assert "고객 기본 계약" in original["Policy_Identity"]["title"]
    raw = (ROOT / "src/policy" / RELATIVE).read_text(encoding="utf-8-sig")
    assert source_result("project-support-source", "source", raw=raw, path=RELATIVE) == original
    with pytest.raises(PolicySourceError, match="SUPPORT_PROJECTION_PATH_MISMATCH"):
        source_result("project-support-source", "source", raw=raw, path="CNTR/wrong.yaml")
    report = request("validate-unified", repository_root=str(ROOT), document=document, restore=True)
    assert report["status"] == "VALID"
    assert report["source"] == original
    assert report["scope"] == "STRUCTURE_SOURCE_INTEGRITY_AND_DECLARED_REFERENCES"
    assert report["evaluation_registration"]["status"] == "UNREGISTERED"
    for key in ("execution_authorized", "evaluation_authorized", "mutation_authorized", "publication_authorized"):
        assert report[key] is False


@pytest.mark.parametrize(("change", "problem"), [
    ("digest", "SOURCE_DIGEST_MISMATCH"),
    ("duplicate", "INVALID_OR_DUPLICATE_SOURCE_ITEM"),
    ("overlap", "DUPLICATE_OR_OVERLAPPING_SOURCE_POINTER"),
    ("invalid escape", "INVALID_SOURCE_POINTER"),
])
def test_go_rejects_invalid_source_partitions(document, change, problem):
    content = document["Policy_Content"]
    if change == "digest":
        content["source_document"]["sha256"] = "0" * 64
    else:
        item = copy.deepcopy(content["items"][0])
        if change != "duplicate":
            item["id"] = "SRC-9999"
        if change == "invalid escape":
            item["source_pointer"] = "/~2"
        content["items"].append(item)
    with pytest.raises(PolicySourceError, match=problem):
        source_result("restore-source", "source", document=document)


def test_go_rejects_unregistered_source_item_references(document):
    document["Policy_Evaluation"]["source_refs"] = ["SRC-9999"]
    report = request("validate-unified", repository_root=str(ROOT), document=document)
    assert report["status"] == "FAIL"
    assert "UNRESOLVED_POLICY_SOURCE_ITEM" in report["problem"]
    assert report["execution_authorized"] is False


def test_build_projection_rejects_duplicate_yaml_keys():
    with pytest.raises(PolicySourceError, match="DUPLICATE_MAPPING_KEY"):
        source_result("project-support-source", "source", raw="duplicate: 1\nduplicate: 2\n", path=RELATIVE)


@pytest.mark.parametrize("relative", [
    "ARCH/SFP-ARCH-0001.yaml",
    "ARCH/SFP-ARCH-0004.yaml",
    "CTRL/SFP-CTRL-0002.yaml",
])
def test_migrated_support_policy_lifecycle_matches_source_index_and_contract(relative):
    """A lifecycle transition must preserve canonical source integrity and index equality."""
    target = ROOT / "src/policy" / relative
    policy = yaml.safe_load(target.read_text(encoding="utf-8"))
    index = yaml.safe_load((ROOT / "src/policy/index.yaml").read_text(encoding="utf-8"))
    official_schema = json.loads(
        (ROOT / "src/ptsip/common_contract/schemas/policy_lifecycle.schema.json").read_text(encoding="utf-8")
    )
    index_schema = json.loads(
        (ROOT / "src/policy/schemas/ptsip-support-feature-policy-index.schema.json").read_text(encoding="utf-8")
    )
    status = "APPROVED.Applying"
    assert policy["Policy_Lifecycle"]["status"] == status
    assert status in official_schema["properties"]["status"]["enum"]
    assert "ACTIVE" not in official_schema["properties"]["status"]["enum"]

    entry, = (item for item in index["policies"] if item["id"] == policy["Policy_Identity"]["id"])
    assert entry["path"] == relative
    assert entry["status"] == status
    assert status in index_schema["properties"]["policies"]["items"]["properties"]["status"]["enum"]
    root_family_schema = json.loads(
        (ROOT / "src/policy/schemas/ptsip-support-root-family-policy.schema.json").read_text(encoding="utf-8")
    )
    assert status in root_family_schema["properties"]["policy"]["properties"]["status"]["enum"]
    source_item, = (
        item for item in policy["Policy_Content"]["items"]
        if item["id"] == policy["Policy_Lifecycle"]["source_ref"]
    )
    assert source_item["source_pointer"] == "/policy"
    assert source_item["value"]["status"] == status

    original = source_result("restore-source", "source", document=policy)
    assert original["policy"]["status"] == status
    canonical = json.dumps(original, ensure_ascii=True, sort_keys=True, separators=(",", ":"), allow_nan=False)
    assert hashlib.sha256(canonical.encode("ascii")).hexdigest() == policy["Policy_Content"]["source_document"]["sha256"]


def test_support_index_lifecycle_transition_retains_legacy_other_policy_entries():
    index = yaml.safe_load((ROOT / "src/policy/index.yaml").read_text(encoding="utf-8"))
    index_schema = json.loads(
        (ROOT / "src/policy/schemas/ptsip-support-feature-policy-index.schema.json").read_text(encoding="utf-8")
    )
    official_schema = json.loads(
        (ROOT / "src/ptsip/common_contract/schemas/policy_lifecycle.schema.json").read_text(encoding="utf-8")
    )
    allowed = set(index_schema["properties"]["policies"]["items"]["properties"]["status"]["enum"])
    canonical = set(official_schema["properties"]["status"]["enum"])
    assert canonical <= allowed
    assert {"DRAFT", "ACTIVE", "DEPRECATED", "RETIRED"} <= allowed
    assert all(entry["status"] in allowed for entry in index["policies"])
    assert "APPROVED.NotDeclared" not in allowed


def test_new_lifecycle_is_parsed_without_implicit_eligibility():
    from ptsip.governance.model import LifecycleState

    assert LifecycleState("APPROVED.Applying") is LifecycleState.APPROVED_APPLYING
    authority_registry = yaml.safe_load(
        (ROOT / "src/policy/registries/ptsip-support-authority-schema-registry.yaml").read_text(encoding="utf-8")
    )
    eligible = authority_registry["lifecycle_policy"]["project_authority_eligible_states"]
    assert "APPROVED.Applying" not in eligible
    assert "ACTIVE" in eligible
