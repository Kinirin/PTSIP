import json
from importlib.resources import files

import pytest
from jsonschema import Draft202012Validator
from referencing import Registry, Resource

from ptsip.user_contracts import validate_policy_v4, validate_user_contracts


LIFECYCLE_STATES = (
    "DRAFT.Default",
    "DRAFT.Reviewed",
    "DRAFT.Denied",
    "DRAFT.Pass",
    "APPROVED.Default",
    "APPROVED.Applying",
    "APPROVED.Application_Problems",
    "APPROVED.Unresolved",
    "APPROVED.Activate",
    "APPROVED.provisional",
    "RETIRED.Default",
    "RETIRED.Removable",
    "RETIRED.Waiting",
)
USER_CONTRACT_DIR = ".ptsip/contract/user"
JSON_SCHEMA_DIALECT = "https://json-schema.org/draft/2020-12/schema"


def write_user_contract(root, filename, target, schema):
    directory = root / USER_CONTRACT_DIR
    directory.mkdir(parents=True, exist_ok=True)
    payload = {"$schema": JSON_SCHEMA_DIALECT, "x-ptsip-target": target, **schema}
    (directory / filename).write_text(json.dumps(payload), encoding="utf-8")


def test_packaged_policy_schema_composes_exactly_four_product_contracts():
    package_dir = files("ptsip").joinpath("schemas", "policy")
    component_paths = {
        "Policy_Identity": ("identity", "PTSIP_policy_identity.schema.json"),
        "Policy_Lifecycle": ("lifecycle", "PTSIP_policy_lifecycle.schema.json"),
        "Policy_Rules": ("rules", "PTSIP_policy_rules.schema.json"),
        "Policy_Content": ("content", "PTSIP_policy_content.schema.json"),
    }
    root = json.loads(package_dir.joinpath("public-structure-policy.schema.json").read_text(encoding="utf-8"))
    components = {
        field: json.loads(package_dir.joinpath(*path).read_text(encoding="utf-8"))
        for field, path in component_paths.items()
    }
    assert root["properties"] == {
        field: {"$ref": schema["$id"]} for field, schema in components.items()
    }
    assert len({root["$id"], *(schema["$id"] for schema in components.values())}) == 5
    assert components["Policy_Lifecycle"]["properties"]["status"]["enum"] == list(LIFECYCLE_STATES)

    for schema in (root, *components.values()):
        Draft202012Validator.check_schema(schema)
    registry = Registry().with_resources(
        (schema["$id"], Resource.from_contents(schema)) for schema in components.values()
    )
    Draft202012Validator(root, registry=registry).validate({field: {} for field in components})


@pytest.mark.parametrize("status", LIFECYCLE_STATES)
def test_policy_v4_accepts_each_approved_lifecycle_state(status):
    report = validate_policy_v4({"Policy_Lifecycle": {"status": status}})
    assert report.valid, report.issues


@pytest.mark.parametrize("description_entry", [{}, {"description": ""}, {"description": "a" * 200}])
def test_lifecycle_evaluation_has_an_optional_description_in_the_second_slot(description_entry):
    evaluation = [{"last_changed_date": "20261117"}, description_entry]
    report = validate_policy_v4({"Policy_Lifecycle": {"evaluation": evaluation}})
    assert report.valid, report.issues


@pytest.mark.parametrize(
    "payload",
    [
        {"Policy_Authority": {}},
        {"Policy_Evaluation": {}},
        {"Policy_Relationships": {}},
        {"Policy_Migration": {}},
        {"Policy_Identity": {"root_family": "ARCH"}},
        {"Policy_Identity": {"policy_kind": "local"}},
        {"Policy_Identity": {"revision": 2}},
        {"Policy_Lifecycle": {"status": "APPROVED.Provisional"}},
        {"Policy_Lifecycle": {"evaluation": [{"last_changed_date": "20261117"}]}},
        {"Policy_Lifecycle": {"evaluation": [{"description": ""}, {"last_changed_date": "20261117"}]}},
        {"Policy_Lifecycle": {"evaluation": [{"last_changed_date": "2026-11-17"}, {}]}},
        {"Policy_Lifecycle": {"evaluation": [{"last_changed_date": "20261301"}, {}]}},
        {"Policy_Lifecycle": {"evaluation": [{"last_changed_date": "20261117"}, {"description": "a" * 201}]}},
        {"Policy_Lifecycle": {"evaluation": [{"last_changed_date": "20261117"}, {}, {}]}},
        {"Policy_Lifecycle": {"evaluation": [{"last_changed_date": "20261117", "approval_ref": "x"}, {}]}},
        {"Policy_Rules": {"rules": [{"id_RA": "R-ID-" + "A" * 32}]}},
        {"Policy_Rules": {"rules": [{"RA_id": "R-ID-" + "a" * 32}]}},
        {"Policy_Rules": {"evaluation": {"source_refs": ["source"]}}},
        {"Policy_Rules": {"evaluation": {"evidence_refs": ["item"]}}},
        {"Policy_Rules": {"evaluation": {"evaluated_at": "20261117"}}},
        {"Policy_Rules": {"evaluation": {"evaluation_scope": "all"}}},
        {"Policy_Content": {"source_document": {}}},
        {"Policy_Content": {"items": [{"source_pointer": "/source"}]}},
        {"Policy_Content": {"items": [{"path": "item.txt"}]}},
        {"Policy_Content": {"items": [{"sha256": "a" * 64}]}},
        {"Policy_Content": {"items": [{"revision_ref": "previous"}]}},
    ],
)
def test_policy_v4_rejects_removed_fields_and_invalid_approved_shapes(payload):
    assert not validate_policy_v4(payload).valid


def test_content_fields_have_no_default_requiredness_or_fixed_id_pattern(tmp_path):
    policy = {"Policy_Content": {"items": [{}, {"id": "project / label"}, {"value": [None, {"mixed": True}]}]}}
    report = validate_policy_v4(policy, project_root=tmp_path, bootstrap=True)
    assert report.valid, report.issues


@pytest.mark.parametrize(
    "version",
    [
        "1.2.3",
        "ver1.2.3",
        "v1.2.3alpha",
        "Ver.1.2.3.beta",
        "V.1.2.3-Hotfix",
        "1.2.3-rc+4.5",
        "1.2.3.4",
        "1.2.3.4.5",
    ],
)
def test_policy_v4_accepts_official_version_without_user_contract(version):
    report = validate_policy_v4({"Policy_Identity": {"version": version}})
    assert report.valid, report.issues


@pytest.mark.parametrize("version", ["1.2", "1.2.3-Beta", "1.2.3-unknown", "1.2.3\n", 123])
def test_policy_v4_rejects_unrecognized_version_without_user_contract(version):
    assert not validate_policy_v4({"Policy_Identity": {"version": version}}).valid


def test_omitted_version_does_not_require_or_bootstrap_a_version_contract(tmp_path):
    report = validate_policy_v4({"Policy_Identity": {"title": "Example"}}, project_root=tmp_path, bootstrap=True)
    assert report.valid, report.issues
    assert not (tmp_path / ".ptsip").exists()


def test_policy_v4_version_accepts_official_or_explicit_user_contract(tmp_path):
    target = "Policy_Identity.version"
    custom = {"release_train": "winter"}
    write_user_contract(tmp_path, "version-for-my-project.json", target, {"const": custom})

    for version in ("v1.2.3", custom):
        report = validate_policy_v4({"Policy_Identity": {"version": version}}, project_root=tmp_path)
        assert report.valid, report.issues
    invalid = validate_policy_v4({"Policy_Identity": {"version": {"release_train": "summer"}}}, project_root=tmp_path)
    assert not invalid.valid
    assert any(issue.path == ("Policy_Identity", "version") for issue in invalid.issues)


def test_category_uses_user_selected_physical_fields_and_requiredness(tmp_path):
    target = "Policy_Identity.Policy_Category"
    write_user_contract(tmp_path, "project-categories.json", target, {
        "x-ptsip-category-roles": {
            "CATEGORY_NAMESPACE": "team",
            "NAMESPACE_DEFINITION": "scope_summary",
        },
        "type": "object",
        "properties": {
            "team": {"type": "string", "description": "Category namespace"},
            "scope_summary": {"type": "string", "description": "Namespace definition"},
        },
        "additionalProperties": False,
    })

    category = {"team": "release", "scope_summary": "Artifact preparation"}
    report = validate_policy_v4({"Policy_Identity": {"Policy_Category": category}}, project_root=tmp_path)
    assert report.valid, report.issues
    assert any(item.targets == (target,) and item.status == "VALID" for item in report.user_contracts)

    report = validate_policy_v4(
        {"Policy_Identity": {"Policy_Category": {"team": "release"}}}, project_root=tmp_path
    )
    assert not report.valid
    assert [(issue.code, issue.path) for issue in report.issues] == [
        ("CATEGORY_ROLE", ("Policy_Identity", "Policy_Category")),
    ]


def test_category_without_user_contract_is_reported_without_autogeneration(tmp_path):
    policy = {"Policy_Identity": {"Policy_Category": {"team": "release", "scope_summary": "Artifact preparation"}}}
    result, = validate_user_contracts(policy, tmp_path, bootstrap=True)
    assert result.targets == ("Policy_Identity.Policy_Category",)
    assert result.status == "UNCONFIGURED"
    assert not (tmp_path / ".ptsip").exists()


def test_user_contract_bootstrap_cannot_write_in_development_checkout(tmp_path):
    checkout = tmp_path / "source"
    product = checkout / "src" / "ptsip"
    (product / "schemas").mkdir(parents=True)
    (product / "CLI.go").touch()
    (product / "schemas" / "config.json").touch()
    project = checkout / "contained-project"
    project.mkdir()
    policy = {"Policy_Content": {"items": [{"id": "a"}]}}

    with pytest.raises(ValueError, match="outside the PTSIP development checkout"):
        validate_user_contracts(policy, project, bootstrap=True)
    assert not (project / ".ptsip").exists()
    assert not (checkout / ".ptsip").exists()
