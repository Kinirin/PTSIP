import pytest

from ptsip.user_contracts import validate_rbac


RA_ID = "R-ID-" + "A" * 32
RAC_ID = "ID-RAC-a" + "r" * 15
SECOND_RAC_ID = "ID-RAC-" + "s" * 16
AEC_ID = "ID-AEC-" + "E" * 16
FILE_ID = "profiles-" + "a" * 32


@pytest.fixture
def lists():
    ra = {
        "RA_list": {
            "entries": [
                {
                    "RA_id": RA_ID,
                    "RA_title": "Review a request",
                    "RA_description": "Review assignment and exercise conditions for the request.",
                    "AEC_ids": [{"Exercise review": AEC_ID}],
                    "AEC_name": ["Exercise review"],
                    "RAC_ids": [
                        {"Primary assignment": RAC_ID},
                        {"Secondary assignment": SECOND_RAC_ID},
                    ],
                    "RAC_name": ["Primary assignment", "Secondary assignment"],
                }
            ]
        }
    }
    rac = {
        "file_name": "RAC-list.ptsip.yaml",
        "file_id": FILE_ID,
        "format_version": "v.1",
        "file_title": "Assignment conditions",
        "RAC_list": [
            {
                "RAC_id": RAC_ID,
                "RAC_name": "Primary assignment",
                "RAC_description": "Applies when the initial reviewer is assigned.",
                "value": {"example": ["primary", 1]},
            },
            {
                "RAC_id": SECOND_RAC_ID,
                "RAC_name": "Secondary assignment",
                "RAC_description": "Applies when the backup reviewer is assigned.",
            },
        ],
    }
    aec = {
        "AEC_list": [
            {
                "AEC_id": AEC_ID,
                "AEC_name": "Exercise review",
                "AEC_description": "Applies when the assigned reviewer acts on the request.",
                "value": ["example", {"threshold": None}],
            }
        ],
    }
    return {"ra": ra, "rac": rac, "aec": aec}


def test_accepts_grouped_name_pairs_from_canonical_lists(lists):
    assert validate_rbac(**lists) == ()


def test_names_can_repeat_when_condition_ids_are_distinct(lists):
    lists["rac"]["RAC_list"][1]["RAC_name"] = "Primary assignment"
    entry = lists["ra"]["RA_list"]["entries"][0]
    entry["RAC_ids"][1] = {"Primary assignment": SECOND_RAC_ID}
    entry["RAC_name"][1] = "Primary assignment"
    assert validate_rbac(**lists) == ()


def test_no_conditions_or_relationship_groups_are_mandatory():
    assert validate_rbac(
        ra={"RA_list": {"entries": [{"RA_id": RA_ID}]}},
        rac={"RAC_list": []},
        aec={"AEC_list": []},
    ) == ()


@pytest.mark.parametrize("value", [None, False, 2, "author value", [], {}, {"custom": 1}])
def test_inline_condition_value_has_no_builtin_internal_shape(value):
    assert validate_rbac(
        rac={"RAC_list": [{"RAC_id": RAC_ID, "value": value}]},
        aec={"AEC_list": [{"AEC_id": AEC_ID, "value": value}]},
    ) == ()


@pytest.mark.parametrize(
    "field, value",
    [
        ("file_id", RAC_ID),
        ("file_id", FILE_ID.upper()),
        ("file_id", FILE_ID + "\n"),
        ("File_Identity", {}),
        ("file_kind", "RAC"),
        ("RAC_list", {"entries": []}),
        ("RAC_list", [{"RAC_id": AEC_ID}]),
        ("RAC_list", [{"RAC_id": RAC_ID, "source_ref": "external.json"}]),
    ],
)
def test_rac_rejects_wrong_file_metadata_types_and_unapproved_fields(field, value):
    issues = validate_rbac(rac={field: value})
    assert any(issue.code == "SCHEMA" for issue in issues)


@pytest.mark.parametrize(
    "field, value",
    [
        ("RA_id", RA_ID.lower()),
        ("policy_usage", []),
        ("RA_purpose", "An eighth field"),
        ("RAC_ids", [RAC_ID]),
        ("RAC_ids", [{"Primary assignment": AEC_ID}]),
        ("RAC_ids", [{"Primary assignment": RAC_ID, "Secondary assignment": SECOND_RAC_ID}]),
        ("AEC_name", [{"Exercise review": AEC_ID}]),
    ],
)
def test_ra_rejects_bad_field_shapes_and_extra_fields(lists, field, value):
    entry = lists["ra"]["RA_list"]["entries"][0]
    entry[field] = value
    assert any(issue.code == "SCHEMA" for issue in validate_rbac(**lists))


@pytest.mark.parametrize("list_name", ["rac", "aec"])
def test_ra_requires_reference_target_in_its_own_list(lists, list_name):
    lists[list_name].clear()
    lists[list_name][f"{list_name.upper()}_list"] = []
    assert any(issue.code == "UNKNOWN_ID" for issue in validate_rbac(**lists))


def test_ra_matches_reference_ids_exactly(lists):
    entry = lists["ra"]["RA_list"]["entries"][0]
    changed_id = "ID-RAC-A" + "r" * 15
    entry["RAC_ids"][0] = {"Primary assignment": changed_id}
    assert any(issue.code == "UNKNOWN_ID" for issue in validate_rbac(**lists))


@pytest.mark.parametrize("kind", ["RA", "RAC", "AEC"])
def test_rejects_duplicate_record_ids(lists, kind):
    if kind == "RA":
        entries = lists["ra"]["RA_list"]["entries"]
        different_field = "RA_title"
    else:
        entries = lists[kind.lower()][f"{kind}_list"]
        different_field = f"{kind}_name"
    entries.append({**entries[0], different_field: "Different record with the same ID"})
    assert any(issue.code == "DUPLICATE_ID" for issue in validate_rbac(**lists))


def test_rejects_duplicate_relationship_ids(lists):
    entry = lists["ra"]["RA_list"]["entries"][0]
    entry["RAC_ids"].append({"Primary assignment": RAC_ID})
    entry["RAC_name"].append("Primary assignment")
    assert any(issue.code == "DUPLICATE_ID" for issue in validate_rbac(**lists))


@pytest.mark.parametrize(
    "names",
    [
        [],
        ["Primary assignment"],
        ["Secondary assignment", "Primary assignment"],
        ["Primary assignment", "Unregistered name"],
    ],
)
def test_supplied_display_names_match_the_reference_sequence(lists, names):
    lists["ra"]["RA_list"]["entries"][0]["RAC_name"] = names
    assert any(issue.code == "NAME_PAIRING" for issue in validate_rbac(**lists))


def test_populated_display_names_require_relationship_ids(lists):
    del lists["ra"]["RA_list"]["entries"][0]["AEC_ids"]
    assert any(issue.code == "NAME_PAIRING" for issue in validate_rbac(**lists))


def test_display_name_view_is_optional(lists):
    del lists["ra"]["RA_list"]["entries"][0]["AEC_name"]
    assert validate_rbac(**lists) == ()


def test_condition_owner_is_the_source_of_canonical_name(lists):
    lists["rac"]["RAC_list"][0]["RAC_name"] = "Renamed primary assignment"
    assert any(issue.code == "NAME_MISMATCH" for issue in validate_rbac(**lists))
    entry = lists["ra"]["RA_list"]["entries"][0]
    entry["RAC_ids"][0] = {"Renamed primary assignment": RAC_ID}
    entry["RAC_name"][0] = "Renamed primary assignment"
    assert validate_rbac(**lists) == ()


def test_ra_cannot_supply_a_missing_canonical_name(lists):
    del lists["aec"]["AEC_list"][0]["AEC_name"]
    assert any(issue.code == "NAME_MISMATCH" for issue in validate_rbac(**lists))
