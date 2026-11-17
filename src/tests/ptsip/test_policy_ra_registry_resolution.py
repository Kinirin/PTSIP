"""Policy v4 registry, user contracts, and forward reference conformance."""

import json
from concurrent.futures import ThreadPoolExecutor
from importlib.resources import files

import pytest

from ptsip.user_contracts import validate_policy_v4, validate_user_contracts
from ptsip.user_contracts._schema import DIALECT, catalog


RA_ID = "R-ID-" + "A" * 32


def _root(project):
    return project / ".ptsip" / "contract" / "user"


def _write(project, name, schema):
    root = _root(project)
    root.mkdir(parents=True, exist_ok=True)
    (root / name).write_text(json.dumps(schema), encoding="utf-8")


def _by_name(results):
    return {result.name: result for result in results}


@pytest.fixture
def policy():
    return {
        "Policy_Rules": {
            "rules": [{"id": "r.first", "content": {"content_id": ["field-name"]}, "RA_id": RA_ID}],
            "evaluation": {"state": "requested"},
        },
        "Policy_Content": {"items": [{"id": "field-name", "value": {"kind": "note"}}]},
    }


def test_registry_locates_only_new_top_level_schemas():
    config = json.loads(files("ptsip").joinpath("schemas", "config.json").read_text(encoding="utf-8"))
    schemas, resources, registry = catalog()
    assert set(schemas) == {"ptsip.policy.v4", "RA_list", "RAC_list", "AEC_list"}
    assert set(config) == {"Policy", "plan", "storage"}
    assert len(resources) == 8
    for identifier in resources:
        assert registry.resolver().lookup(identifier).contents["$id"] == identifier


def test_rule_id_and_content_id_use_independent_patterns(tmp_path, policy):
    _write(tmp_path, "rules_patterns.json", {"$schema": DIALECT, "pattern": r"^r\."})
    _write(tmp_path, "policy_content_id.json", {"$schema": DIALECT, "pattern": r"^[a-z-]+$"})
    _write(tmp_path, "evaluation_state.json", {"$schema": DIALECT, "const": "requested"})
    results = _by_name(validate_user_contracts(policy, tmp_path))
    assert {result.status for result in results.values()} == {"VALID"}

    policy["Policy_Rules"]["rules"][0]["content"]["content_id"] = ["r.first"]
    results = _by_name(validate_user_contracts(policy, tmp_path))
    assert results["rules_patterns.json"].status == "VALID"
    assert results["policy_content_id.json"].status == "INVALID"
    assert results["policy_content_id.json"].issues[0].path == (
        "Policy_Rules", "rules", 0, "content", "content_id", 0
    )


def test_only_missing_used_contract_is_bootstrapped(tmp_path):
    policy = {"Policy_Rules": {"evaluation": {"state": False}}}
    assert validate_user_contracts(policy, tmp_path)[0].status == "UNCONFIGURED"
    assert not _root(tmp_path).exists()

    results = validate_user_contracts(policy, tmp_path, bootstrap=True)
    assert [result.status for result in results] == ["UNCONFIGURED"]
    names = [path.name for path in _root(tmp_path).iterdir()]
    assert names == ["evaluation_state.json"]
    assert json.loads((_root(tmp_path) / names[0]).read_bytes()) == {"$schema": DIALECT}
    assert validate_user_contracts({}, tmp_path, bootstrap=True) == ()


def test_bootstrap_is_idempotent_when_requested_concurrently(tmp_path, policy):
    with ThreadPoolExecutor(max_workers=4) as pool:
        results = list(pool.map(lambda _: validate_user_contracts(policy, tmp_path, bootstrap=True), range(12)))
    expected = {"evaluation_state.json", "policy_content_id.json", "rules_patterns.json"}
    assert {path.name for path in _root(tmp_path).iterdir()} == expected
    for result in results:
        assert {item.name for item in result} == expected
        assert {item.status for item in result} == {"UNCONFIGURED"}


@pytest.mark.parametrize(
    ("payload", "status", "code"),
    [
        (b"", "INVALID", "CONTRACT_SYNTAX"),
        (b"[", "INVALID", "CONTRACT_SYNTAX"),
        (b'{"enum":[1],"enum":[2]}', "INVALID", "CONTRACT_SYNTAX"),
        (b'{"pattern":4}', "INVALID", "CONTRACT_SCHEMA"),
        (b'{"properties":[]}', "INVALID", "CONTRACT_SCHEMA"),
        (b'{"pattern":"a{99999999999999999999999}"}', "INVALID", "CONTRACT_SCHEMA"),
        (b'{"$schema":"https://json-schema.org/draft-07/schema"}', "UNSUPPORTED", "CONTRACT_DIALECT"),
        (
            b'{"$defs":{"legacy":{"$id":"urn:legacy","$schema":"http://json-schema.org/draft-07/schema#"}}}',
            "UNSUPPORTED", "CONTRACT_DIALECT",
        ),
    ],
)
def test_preserve_and_report_existing_bad_contracts(tmp_path, policy, payload, status, code):
    root = _root(tmp_path)
    root.mkdir(parents=True)
    target = root / "evaluation_state.json"
    target.write_bytes(payload)
    result = _by_name(validate_user_contracts(policy, tmp_path, bootstrap=True))[target.name]
    assert target.read_bytes() == payload
    assert result.status == status
    assert result.issues[0].code == code


def test_custom_bindings_use_explicit_paths_and_same_directory_resources(tmp_path, policy):
    _write(tmp_path, "unspecified.json", {"const": 8})
    _write(tmp_path, "wrong-target.json", {"x-ptsip-target": "Policy_Identity.missing", "const": 8})
    _write(tmp_path, "vocabulary.json", {"$defs": {"state": {"enum": ["requested"]}}})
    _write(tmp_path, "evaluation_state.json", {"$ref": "vocabulary.json#/$defs/state"})
    _write(tmp_path, "require-rules.json", {"x-ptsip-target": "Policy_Rules", "required": ["rules"]})
    results = _by_name(validate_user_contracts(policy, tmp_path))
    assert results["evaluation_state.json"].status == "VALID"
    assert results["require-rules.json"].status == "VALID"
    for name in ("unspecified.json", "wrong-target.json", "vocabulary.json"):
        assert results[name].status == "UNBOUND"


@pytest.mark.parametrize("reference", ["missing.json", "../outside.json", "https://example.test/contract.json"])
def test_unknown_or_external_resource_is_reported(tmp_path, policy, reference):
    _write(tmp_path, "evaluation_state.json", {"$ref": reference})
    (_root(tmp_path).parent / "outside.json").write_text('{"const":"requested"}', encoding="utf-8")
    result = _by_name(validate_user_contracts(policy, tmp_path))["evaluation_state.json"]
    assert result.status == "INVALID"
    assert result.issues[0].code == "CONTRACT_REFERENCE"


def test_unvisited_schema_branch_still_requires_a_local_reference(tmp_path, policy):
    _write(tmp_path, "evaluation_state.json", {"anyOf": [True, {"$ref": "missing.json"}]})
    result = _by_name(validate_user_contracts(policy, tmp_path))["evaluation_state.json"]
    assert result.status == "INVALID"
    assert result.issues[0].code == "CONTRACT_REFERENCE"


def test_relative_reference_resolves_from_declared_canonical_id(tmp_path, policy):
    parent = "https://schema.example.test/contracts/"
    _write(tmp_path, "evaluation_state.json", {"$id": parent + "state", "$ref": "vocabulary"})
    _write(tmp_path, "vocabulary.json", {"$id": parent + "vocabulary", "const": "requested"})
    result = _by_name(validate_user_contracts(policy, tmp_path))["evaluation_state.json"]
    assert result.status == "VALID"


def test_duplicate_embedded_resource_ids_are_ambiguous(tmp_path, policy):
    _write(tmp_path, "evaluation_state.json", {"$ref": "urn:test:shared"})
    for name, state in (("vocabulary-one.json", "requested"), ("vocabulary-two.json", "other")):
        _write(tmp_path, name, {"$defs": {"state": {"$id": "urn:test:shared", "const": state}}})
    results = _by_name(validate_user_contracts(policy, tmp_path))
    assert len(results) == 5
    for name in ("evaluation_state.json", "vocabulary-one.json", "vocabulary-two.json"):
        assert results[name].status == "INVALID"
        assert results[name].issues[0].code == "CONTRACT_REFERENCE"


def test_custom_contract_can_target_fields_inside_policy_schema_tuples(tmp_path):
    custom = {
        "x-ptsip-target": "Policy_Lifecycle.evaluation[].last_changed_date",
        "pattern": "^2026",
    }
    _write(tmp_path, "date-boundary.json", custom)
    document = {"Policy_Lifecycle": {"evaluation": [{"last_changed_date": "20251117"}, {}]}}
    result, = validate_user_contracts(document, tmp_path)
    assert result.status == "INVALID"
    assert result.issues[0].path == ("Policy_Lifecycle", "evaluation", 0, "last_changed_date")


def test_rule_uses_an_exact_ra_id_and_returns_immutable_projected_usage(policy):
    assert not validate_policy_v4(policy).valid
    matching = {"RA_list": {"entries": [{"RA_id": RA_ID}]}}
    other = {"RA_list": {"entries": [{"RA_id": "R-ID-" + "B" * 32}]}}
    result = validate_policy_v4(policy, ra=matching)
    assert result.valid
    assert result.ra_usage == ((RA_ID, (("Policy_Rules", "rules", 0, "RA_id"),)),)
    assert set(matching["RA_list"]["entries"][0]) == {"RA_id"}
    missing = validate_policy_v4(policy, ra=other)
    assert missing.ra_usage == ()
    assert missing.issues[0].path == ("Policy_Rules", "rules", 0, "RA_id")


def test_content_must_resolve_uniquely_in_its_own_document(policy):
    del policy["Policy_Rules"]["rules"][0]["RA_id"]
    assert validate_policy_v4(policy).valid
    policy["Policy_Content"]["items"].append({"id": "field-name"})
    assert {issue.code for issue in validate_policy_v4(policy).issues} == {"DUPLICATE_ID", "UNKNOWN_ID"}
    policy["Policy_Content"]["items"] = [{"id": "Field-name"}]
    assert [issue.code for issue in validate_policy_v4(policy).issues] == ["UNKNOWN_ID"]
