from __future__ import annotations

from pathlib import Path
from types import SimpleNamespace

import pytest
import yaml

from ptsip.migration.identity_rewrite import (
    IdentityRewriteAuthorization,
    IdentityRewriteError,
    authorize_identity_rewrite,
    build_identity_rewrite_plan,
    execute_identity_rewrite,
)
from ptsip.profile_compatibility import V036_REVISION
from ptsip.repository.profile_convergence import discover_direct_profile_convergence
from ptsip.validation.profile import validate_profile


def _payload() -> dict[str, object]:
    return {
        "ptsip": {
            "version": "0.3.6-draft",
            "specification": {
                "source": "https://github.com/Kinirin/PTSIP",
                "revision": V036_REVISION,
            },
        },
        "responsibility_map": {"mode": "explicit"},
        "components": [
            {
                "id": "core",
                "classification": "PRODUCT",
                "roles": ["IMPLEMENTATION"],
                "include": ["src/**"],
                "purpose": "runtime",
                "shipped": True,
                "runtime_required": True,
            }
        ],
        "associated_artifacts": [],
        "relationships": [],
        "policies": {
            "product_to_nonproduct_runtime_dependency": "deny",
            "nonproduct_in_product_package": "deny",
            "independent_build_resolution": "required",
            "shared_executable_cross_lifecycle": "deny",
            "neutral_contract_sharing": "allow",
        },
    }


def _prepare(root: Path):
    (root / "src").mkdir()
    (root / "src/a.py").write_text("a", encoding="utf-8")
    profile = root / "ptsip.yaml"
    profile.write_text(
        "# preserved header\n" + yaml.safe_dump(_payload(), sort_keys=False),
        encoding="utf-8",
    )
    discovery = discover_direct_profile_convergence(root)
    assert discovery.valid and discovery.state is not None
    return profile, discovery.state


def test_identity_rewrite_changes_only_contract_identity_and_validates(tmp_path: Path) -> None:
    profile, state = _prepare(tmp_path)
    before = profile.read_bytes()
    plan = build_identity_rewrite_plan(state)
    authorization = authorize_identity_rewrite(plan, authority_revision="owner-decision:1")

    result = execute_identity_rewrite(tmp_path, plan, authorization)

    after = profile.read_bytes()
    expected = before.replace(b"version: 0.3.6-draft", b"version: pp.1.01", 1)
    line_ending = b"\r\n" if b"\r\n" in before else b"\n"
    expected = expected.replace(
        b"  specification:" + line_ending,
        b"  specification:" + line_ending + b"    family: 0.3.6-draft" + line_ending,
        1,
    )
    assert after == expected
    assert result.source_declared_version == "0.3.6-draft"
    assert result.target_contract == "pp.1.01"
    assert result.specification_family == "0.3.6-draft"
    assert result.before_sha256 != result.after_sha256
    rewritten = yaml.safe_load(after)
    assert rewritten["ptsip"]["specification"] == {
        "family": "0.3.6-draft",
        "source": "https://github.com/Kinirin/PTSIP",
        "revision": V036_REVISION,
    }
    for key in (
        "responsibility_map",
        "components",
        "associated_artifacts",
        "relationships",
        "policies",
    ):
        assert rewritten[key] == _payload()[key]
    validation = validate_profile(tmp_path)
    assert validation.valid


def test_identity_rewrite_requires_exact_authorized_plan(tmp_path: Path) -> None:
    _profile, state = _prepare(tmp_path)
    plan = build_identity_rewrite_plan(state)
    stale = IdentityRewriteAuthorization(
        plan_digest="0" * 64,
        authority_revision="owner-decision:1",
        authorization_id="stale",
    )

    with pytest.raises(IdentityRewriteError) as exc_info:
        execute_identity_rewrite(tmp_path, plan, stale)

    assert exc_info.value.code == "PP_IDENTITY_REWRITE_AUTHORIZATION_STALE"


def test_identity_rewrite_fails_closed_if_repository_changes_after_plan(tmp_path: Path) -> None:
    _profile, state = _prepare(tmp_path)
    plan = build_identity_rewrite_plan(state)
    authorization = authorize_identity_rewrite(plan, authority_revision="owner-decision:1")
    (tmp_path / "unexpected.txt").write_text("changed", encoding="utf-8")

    with pytest.raises(IdentityRewriteError) as exc_info:
        execute_identity_rewrite(tmp_path, plan, authorization)

    assert exc_info.value.code == "PP_IDENTITY_REWRITE_STALE_REPOSITORY"


def test_identity_rewrite_rolls_back_original_bytes_on_post_validation_failure(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    profile, state = _prepare(tmp_path)
    before = profile.read_bytes()
    plan = build_identity_rewrite_plan(state)
    authorization = authorize_identity_rewrite(plan, authority_revision="owner-decision:1")
    monkeypatch.setattr(
        "ptsip.migration.identity_rewrite.validate_profile",
        lambda *_args, **_kwargs: SimpleNamespace(
            valid=False,
            errors=["forced post-write failure"],
            warnings=[],
        ),
    )

    with pytest.raises(IdentityRewriteError) as exc_info:
        execute_identity_rewrite(tmp_path, plan, authorization)

    assert exc_info.value.code == "PP_IDENTITY_REWRITE_POST_VALIDATION_FAILED"
    assert profile.read_bytes() == before


def test_identity_rewrite_authority_revision_must_be_explicit(tmp_path: Path) -> None:
    _profile, state = _prepare(tmp_path)
    plan = build_identity_rewrite_plan(state)

    with pytest.raises(IdentityRewriteError) as exc_info:
        authorize_identity_rewrite(plan, authority_revision="  ")

    assert exc_info.value.code == "PP_IDENTITY_REWRITE_AUTHORITY_REQUIRED"
