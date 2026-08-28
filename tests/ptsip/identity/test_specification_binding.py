from __future__ import annotations

import pytest

from ptsip.profile_identity import (
    PP_1_01,
    ProjectProfileOperation,
    require_current_project_profile_support,
)
from ptsip.specification_binding import (
    SPECIFICATION_036,
    SPECIFICATION_037,
    SPECIFICATION_SOURCE,
    SpecificationBinding,
    SpecificationBindingError,
    SpecificationOperation,
    current_target_specification_binding,
    require_current_specification_support,
)


def test_specification_binding_round_trips_exact_identity() -> None:
    binding = SpecificationBinding.from_mapping(
        {
            "family": "0.3.7-draft",
            "source": SPECIFICATION_SOURCE,
            "revision": SPECIFICATION_037.revision,
        }
    )

    assert binding.as_dict() == {
        "family": "0.3.7-draft",
        "source": SPECIFICATION_SOURCE,
        "revision": SPECIFICATION_037.revision,
    }


@pytest.mark.parametrize(
    ("payload", "code"),
    [
        ({"source": SPECIFICATION_SOURCE, "revision": SPECIFICATION_037.revision}, "SPEC_BINDING_FIELD_MISSING"),
        (
            {
                "family": "0.3.7-draft",
                "source": SPECIFICATION_SOURCE,
                "revision": SPECIFICATION_037.revision,
                "version": "pp.1.01",
            },
            "SPEC_BINDING_FIELD_UNEXPECTED",
        ),
        (
            {
                "family": " 0.3.7-draft",
                "source": SPECIFICATION_SOURCE,
                "revision": SPECIFICATION_037.revision,
            },
            "SPEC_BINDING_FAMILY_INVALID",
        ),
        (
            {
                "family": "0.3.7-draft",
                "source": "not-a-uri",
                "revision": SPECIFICATION_037.revision,
            },
            "SPEC_BINDING_SOURCE_MALFORMED",
        ),
        (
            {
                "family": "0.3.7-draft",
                "source": SPECIFICATION_SOURCE,
                "revision": "mutable",
            },
            "SPEC_BINDING_REVISION_MALFORMED",
        ),
    ],
)
def test_specification_binding_rejects_malformed_identity(payload: object, code: str) -> None:
    with pytest.raises(SpecificationBindingError) as caught:
        SpecificationBinding.from_mapping(payload)

    assert caught.value.code == code


def test_specification_registry_supports_current_and_deliberate_historical_binding() -> None:
    current = require_current_specification_support(
        SPECIFICATION_037,
        SpecificationOperation.VALIDATE,
    )
    historical = require_current_specification_support(
        SPECIFICATION_036,
        SpecificationOperation.VALIDATE,
    )

    assert current.binding == SPECIFICATION_037
    assert historical.binding == SPECIFICATION_036


def test_historical_binding_is_not_target_creation_authority() -> None:
    with pytest.raises(SpecificationBindingError) as caught:
        require_current_specification_support(
            SPECIFICATION_036,
            SpecificationOperation.CREATE_TARGET,
        )

    assert caught.value.code == "SPEC_BINDING_UNSUPPORTED"


def test_unknown_revision_fails_closed_without_historical_migration_lookup() -> None:
    unsupported = SpecificationBinding(
        family=SPECIFICATION_037.family,
        source=SPECIFICATION_037.source,
        revision="0" * 40,
    )

    with pytest.raises(SpecificationBindingError) as caught:
        require_current_specification_support(
            unsupported,
            SpecificationOperation.VALIDATE,
        )

    assert caught.value.code == "SPEC_BINDING_UNSUPPORTED"


def test_pp_and_specification_capabilities_are_independent_authorities() -> None:
    pp_support = require_current_project_profile_support(
        PP_1_01,
        ProjectProfileOperation.VALIDATE,
    )
    specification_support = require_current_specification_support(
        SPECIFICATION_037,
        SpecificationOperation.VALIDATE,
    )

    assert pp_support.contract == PP_1_01
    assert specification_support.binding == SPECIFICATION_037
    assert current_target_specification_binding() == SPECIFICATION_037
