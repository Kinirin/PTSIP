from __future__ import annotations

from copy import deepcopy

from .constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from .profile_identity import (
    CURRENT_PROJECT_PROFILE_VERSION,
    DEVELOPER_BASELINE_USER_REVISION,
    ProjectProfileVersion,
)


def project_profile_ptsip_metadata(
    version: str,
    *,
    role: str = "PROJECT",
) -> dict[str, object]:
    parsed = ProjectProfileVersion.parse(version, require_canonical=True)
    if parsed >= ProjectProfileVersion(1, 2):
        return {
            "version": parsed.canonical,
            "revision": DEVELOPER_BASELINE_USER_REVISION.canonical,
            "profile_role": role,
            "specification": {
                "source": SPEC_SOURCE,
                "revision": SPEC_REVISION,
            },
        }
    return {
        "version": parsed.canonical,
        "specification": {
            "family": SPEC_VERSION,
            "source": SPEC_SOURCE,
            "revision": SPEC_REVISION,
        },
    }


def current_project_profile_ptsip_metadata(
    *,
    role: str = "PROJECT",
) -> dict[str, object]:
    return deepcopy(
        project_profile_ptsip_metadata(
            CURRENT_PROJECT_PROFILE_VERSION,
            role=role,
        )
    )
