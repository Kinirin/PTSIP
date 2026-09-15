from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

from .model import GovernanceAuthorityError


@dataclass(frozen=True)
class SupportAssetLayout:
    policy: Path
    schemas: Path
    registries: Path
    source: str


def _is_complete(layout: SupportAssetLayout) -> bool:
    return (
        (layout.policy / "index.yaml").is_file()
        and layout.schemas.is_dir()
        and layout.registries.is_dir()
    )


def resolve_support_asset_layout(repository_root: str | Path) -> SupportAssetLayout:
    """Resolve shipped Support assets or the canonical repository source."""

    package_root = Path(__file__).resolve().parents[1]
    shipped_root = package_root / "support"
    shipped = SupportAssetLayout(
        policy=shipped_root / "policy",
        schemas=shipped_root / "schemas",
        registries=shipped_root / "registries",
        source="SHIPPED_PROJECTION",
    )
    if _is_complete(shipped):
        return shipped

    repository = Path(repository_root).resolve()
    canonical_root = repository / "docs" / "Support_policy" / "policy"
    canonical = SupportAssetLayout(
        policy=canonical_root,
        schemas=canonical_root / "schemas",
        registries=canonical_root / "registries",
        source="CANONICAL_REPOSITORY_SOURCE",
    )
    if _is_complete(canonical):
        return canonical

    raise GovernanceAuthorityError(
        "SUPPORT_ASSET_LAYOUT_MISSING",
        "neither shipped ptsip/support assets nor canonical docs/Support_policy/policy assets are available.",
        {"shipped": str(shipped_root), "canonical": str(canonical_root)},
    )


def resolve_support_asset(
    layout: SupportAssetLayout,
    category: str,
    relative: str,
) -> Path:
    bases = {
        "policy": layout.policy,
        "schemas": layout.schemas,
        "registries": layout.registries,
    }
    base = bases.get(category)
    if base is None:
        raise GovernanceAuthorityError(
            "SUPPORT_ASSET_CATEGORY_INVALID",
            "support asset category is not registered.",
            category,
        )

    path = Path(relative)
    if path.is_absolute() or ".." in path.parts:
        raise GovernanceAuthorityError(
            "GOVERNANCE_PATH_INVALID",
            "support governance paths must be relative and traversal-free.",
            relative,
        )

    resolved = (base / path).resolve()
    resolved_base = base.resolve()
    if resolved_base not in resolved.parents and resolved != resolved_base:
        raise GovernanceAuthorityError(
            "GOVERNANCE_PATH_INVALID",
            "support governance path escaped its declared support boundary.",
            relative,
        )
    return resolved
