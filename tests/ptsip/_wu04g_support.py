"""Compatibility shim for historical WU-04G test helpers.

New and migrated tests should import from ``_test_support``. This module remains only
while legacy root-level suites are being normalized during repository verification
infrastructure cleanup.
"""

from _test_support import (  # noqa: F401
    PYTHON_PACKAGE_TEMPLATE_ID,
    PYTHON_PACKAGE_TEMPLATE_REVISION,
    associated_artifact_payload,
    canonical_v2_answer,
    clarification_answer_text,
    clone_repo,
    commit_all,
    component_payload,
    explicit_profile_payload,
    git,
    hybrid_profile_payload,
    init_git_repo,
    legacy_v1_answer,
    policy_payload,
    template_profile_payload,
    write_profile,
    write_text,
)
