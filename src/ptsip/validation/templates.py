from __future__ import annotations

import copy
import hashlib
import json
from dataclasses import dataclass
from typing import Mapping


CATALOG_FORMAT = "ptsip-responsibility-template-catalog/v1"


class TemplateMaterializationError(ValueError):
    """Raised when a template-backed Responsibility Map cannot be resolved safely."""


@dataclass(frozen=True)
class TemplateDefinition:
    id: str
    revision: str
    description: str
    map_payload: dict[str, object]

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "revision": self.revision,
            "description": self.description,
            "map": copy.deepcopy(self.map_payload),
        }


@dataclass(frozen=True)
class MaterializedProfile:
    payload: dict[str, object]
    source_mode: str
    template_id: str | None
    template_revision: str | None

    @property
    def template_bound(self) -> bool:
        return self.template_id is not None and self.template_revision is not None

    def identity(self) -> dict[str, object]:
        return {
            "source_mode": self.source_mode,
            "materialized": True,
            "template": (
                {"id": self.template_id, "revision": self.template_revision}
                if self.template_bound
                else None
            ),
        }


def calculate_template_revision(map_payload: Mapping[str, object]) -> str:
    """Return the immutable semantic revision for one template map.

    Revisions are SHA-256 digests of canonical JSON for the template's map
    payload only. Changing component, selector, role, relationship, artifact,
    or dependency-policy semantics therefore changes the required revision.
    """

    encoded = json.dumps(
        map_payload,
        sort_keys=True,
        separators=(",", ":"),
        ensure_ascii=False,
    ).encode("utf-8")
    return "sha256:" + hashlib.sha256(encoded).hexdigest()


_TEMPLATE_ROWS: tuple[dict[str, object], ...] = (
    {
        "id": "python-package-library",
        "revision": "sha256:409acd1cd9907a60761a3cf26a051185d40b5e926e6952131b641b10bccc5c9b",
        "description": "Python package/library with Product implementation and Product-owned verification.",
        "map": {
            "components": [
                {
                    "id": "package",
                    "classification": "PRODUCT",
                    "roles": ["IMPLEMENTATION"],
                    "include": ["src/**"],
                    "purpose": "python_package_library_implementation",
                    "shipped": True,
                    "runtime_required": True,
                    "executable": True,
                },
                {
                    "id": "package-tests",
                    "classification": "PRODUCT",
                    "roles": ["VERIFICATION"],
                    "include": ["tests/**"],
                    "purpose": "product_owned_package_verification",
                    "shipped": False,
                    "runtime_required": False,
                    "executable": True,
                },
            ],
            "relationships": [
                {
                    "id": "package-tests-verify-package",
                    "from": "package-tests",
                    "to": "package",
                    "type": "VERIFIES",
                }
            ],
        },
    },
    {
        "id": "python-cli-application",
        "revision": "sha256:fce170879bb5afcbd768ce473e6782ce05b50655eb93c7d84c4464e3d079b572",
        "description": "Python CLI/application with Product implementation and Product-owned verification.",
        "map": {
            "components": [
                {
                    "id": "application",
                    "classification": "PRODUCT",
                    "roles": ["IMPLEMENTATION"],
                    "include": ["src/**"],
                    "purpose": "python_cli_application_implementation",
                    "shipped": True,
                    "runtime_required": True,
                    "executable": True,
                },
                {
                    "id": "application-tests",
                    "classification": "PRODUCT",
                    "roles": ["VERIFICATION"],
                    "include": ["tests/**"],
                    "purpose": "product_owned_application_verification",
                    "shipped": False,
                    "runtime_required": False,
                    "executable": True,
                },
            ],
            "relationships": [
                {
                    "id": "application-tests-verify-application",
                    "from": "application-tests",
                    "to": "application",
                    "type": "VERIFIES",
                }
            ],
        },
    },
    {
        "id": "mixed-product-development-delivery",
        "revision": "sha256:ae45cb5674c0e9e87259ca629317d8c47420d55a1609b7b065ec6b1b2d515692",
        "description": "Mixed-lifecycle repository with Product implementation, reusable development verification, and release delivery automation.",
        "map": {
            "components": [
                {
                    "id": "product",
                    "classification": "PRODUCT",
                    "roles": ["IMPLEMENTATION"],
                    "include": ["src/**"],
                    "purpose": "product_implementation",
                    "shipped": True,
                    "runtime_required": True,
                    "executable": True,
                },
                {
                    "id": "development-verification",
                    "classification": "DEVELOPMENT_TOOLING",
                    "roles": ["VERIFICATION", "AUTOMATION"],
                    "include": ["tests/**"],
                    "purpose": "reusable_development_verification",
                    "shipped": False,
                    "runtime_required": False,
                    "executable": True,
                },
                {
                    "id": "delivery",
                    "classification": "DELIVERY",
                    "roles": ["AUTOMATION"],
                    "include": [".github/workflows/release.yml"],
                    "purpose": "release_delivery_automation",
                    "shipped": False,
                    "runtime_required": False,
                    "executable": True,
                },
            ],
            "relationships": [
                {
                    "id": "development-verification-verifies-product",
                    "from": "development-verification",
                    "to": "product",
                    "type": "VERIFIES",
                },
                {
                    "id": "delivery-builds-product",
                    "from": "delivery",
                    "to": "product",
                    "type": "BUILDS",
                },
                {
                    "id": "delivery-publishes-product",
                    "from": "delivery",
                    "to": "product",
                    "type": "PUBLISHES",
                },
            ],
        },
    },
)


def template_catalog() -> tuple[TemplateDefinition, ...]:
    """Return the closed Tool 0.3.6 template catalog after integrity checks."""

    definitions: list[TemplateDefinition] = []
    seen: set[tuple[str, str]] = set()
    for row in _TEMPLATE_ROWS:
        template_id = str(row.get("id", ""))
        revision = str(row.get("revision", ""))
        description = str(row.get("description", ""))
        map_payload = row.get("map")
        if not template_id or not revision or not isinstance(map_payload, Mapping):
            raise RuntimeError("PTSIP template catalog contains an invalid definition.")
        expected = calculate_template_revision(map_payload)
        if revision != expected:
            raise RuntimeError(
                f"PTSIP template {template_id!r} declares revision {revision!r}, "
                f"but its semantic content requires {expected!r}."
            )
        key = (template_id, revision)
        if key in seen:
            raise RuntimeError(f"Duplicate PTSIP template identity: {template_id}@{revision}")
        seen.add(key)
        definitions.append(
            TemplateDefinition(
                id=template_id,
                revision=revision,
                description=description,
                map_payload=copy.deepcopy(dict(map_payload)),
            )
        )
    return tuple(definitions)


def resolve_template(template_ref: Mapping[str, object]) -> TemplateDefinition:
    template_id = template_ref.get("id")
    revision = template_ref.get("revision")
    if not isinstance(template_id, str) or not template_id:
        raise TemplateMaterializationError("Template reference requires a non-empty id.")
    if not isinstance(revision, str) or not revision:
        raise TemplateMaterializationError("Template reference requires a non-empty revision.")

    catalog = template_catalog()
    for definition in catalog:
        if definition.id == template_id and definition.revision == revision:
            return definition

    available = [definition.revision for definition in catalog if definition.id == template_id]
    if available:
        raise TemplateMaterializationError(
            f"Unknown revision {revision!r} for PTSIP template {template_id!r}; "
            f"available revision(s): {', '.join(available)}."
        )
    raise TemplateMaterializationError(f"Unknown PTSIP template id: {template_id!r}.")


def _id(value: object, label: str) -> str:
    if not isinstance(value, Mapping):
        raise TemplateMaterializationError(f"{label} entry must be a mapping.")
    item_id = value.get("id")
    if not isinstance(item_id, str) or not item_id:
        raise TemplateMaterializationError(f"{label} entry requires a non-empty id.")
    return item_id


def _merge_id_collection(
    base: object,
    replacements: object,
    removals: object,
    *,
    label: str,
) -> list[dict[str, object]]:
    base_items = [copy.deepcopy(dict(item)) for item in base or [] if isinstance(item, Mapping)]
    replacement_items = [
        copy.deepcopy(dict(item))
        for item in replacements or []
        if isinstance(item, Mapping)
    ]
    removal_ids = [str(item) for item in removals or [] if isinstance(item, str)]

    base_ids = [_id(item, label) for item in base_items]
    if len(base_ids) != len(set(base_ids)):
        raise TemplateMaterializationError(f"Template {label} IDs are not unique.")

    replacement_ids = [_id(item, f"override {label}") for item in replacement_items]
    if len(replacement_ids) != len(set(replacement_ids)):
        raise TemplateMaterializationError(f"Override {label} IDs are not unique.")

    unknown_removals = sorted(set(removal_ids) - set(base_ids))
    if unknown_removals:
        raise TemplateMaterializationError(
            f"Hybrid override removes unknown {label} ID(s): {', '.join(unknown_removals)}."
        )
    conflicting = sorted(set(replacement_ids) & set(removal_ids))
    if conflicting:
        raise TemplateMaterializationError(
            f"Hybrid override cannot replace and remove the same {label} ID(s): {', '.join(conflicting)}."
        )

    replacement_by_id = {
        _id(item, f"override {label}"): item for item in replacement_items
    }
    result: list[dict[str, object]] = []
    for item in base_items:
        item_id = _id(item, label)
        if item_id in removal_ids:
            continue
        result.append(copy.deepcopy(replacement_by_id.pop(item_id, item)))

    for item in replacement_items:
        item_id = _id(item, f"override {label}")
        if item_id in replacement_by_id:
            result.append(copy.deepcopy(item))
            replacement_by_id.pop(item_id, None)
    return result


def materialize_profile(payload: Mapping[str, object]) -> MaterializedProfile:
    """Resolve explicit/template/hybrid input into one deterministic explicit map.

    Materialization is read-only: the source payload is never mutated. Hybrid
    overrides replace matching stable IDs in place, remove only known template
    IDs, and append genuinely new project-owned IDs in declaration order.
    """

    source = copy.deepcopy(dict(payload))
    map_meta = source.get("responsibility_map")
    if not isinstance(map_meta, Mapping):
        raise TemplateMaterializationError("Profile responsibility_map must be a mapping.")

    mode = map_meta.get("mode")
    if mode == "explicit":
        return MaterializedProfile(
            payload=source,
            source_mode="explicit",
            template_id=None,
            template_revision=None,
        )
    if mode not in {"template", "hybrid"}:
        raise TemplateMaterializationError(f"Unsupported Responsibility Map mode: {mode!r}.")

    template_ref = map_meta.get("template")
    if not isinstance(template_ref, Mapping):
        raise TemplateMaterializationError("Template-backed map requires a template reference.")
    definition = resolve_template(template_ref)
    effective_map = copy.deepcopy(definition.map_payload)

    if mode == "hybrid":
        overrides = map_meta.get("overrides")
        if not isinstance(overrides, Mapping):
            raise TemplateMaterializationError("Hybrid map requires an overrides mapping.")

        effective_map["components"] = _merge_id_collection(
            effective_map.get("components", []),
            overrides.get("components", []),
            overrides.get("remove_component_ids", []),
            label="component",
        )
        effective_map["associated_artifacts"] = _merge_id_collection(
            effective_map.get("associated_artifacts", []),
            overrides.get("associated_artifacts", []),
            overrides.get("remove_associated_artifact_ids", []),
            label="associated-artifact",
        )
        effective_map["relationships"] = _merge_id_collection(
            effective_map.get("relationships", []),
            overrides.get("relationships", []),
            overrides.get("remove_relationship_ids", []),
            label="relationship",
        )

    materialized = source
    for key in ("components", "associated_artifacts", "relationships", "component_dependency_policy"):
        materialized.pop(key, None)
        if key in effective_map:
            materialized[key] = copy.deepcopy(effective_map[key])
    materialized["responsibility_map"] = {"mode": "explicit"}

    return MaterializedProfile(
        payload=materialized,
        source_mode=str(mode),
        template_id=definition.id,
        template_revision=definition.revision,
    )
