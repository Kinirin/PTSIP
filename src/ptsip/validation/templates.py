from __future__ import annotations

import copy
import hashlib
import json
from dataclasses import dataclass
from types import MappingProxyType
from typing import Mapping


CATALOG_FORMAT = "ptsip-responsibility-template-catalog/v1"

PROJECT_EXPLICIT = "PROJECT_EXPLICIT"
TEMPLATE = "TEMPLATE"
PROJECT_OVERRIDE = "PROJECT_OVERRIDE"
PROJECT_EXTENSION = "PROJECT_EXTENSION"
PROJECT_REMOVAL = "PROJECT_REMOVAL"

_PROVENANCE_VALUES = frozenset(
    {
        PROJECT_EXPLICIT,
        TEMPLATE,
        PROJECT_OVERRIDE,
        PROJECT_EXTENSION,
        PROJECT_REMOVAL,
    }
)
_STABLE_ID_COLLECTIONS = ("components", "associated_artifacts", "relationships")
_SET_VALUED_FIELDS = frozenset(
    {
        "roles",
        "include",
        "exclude",
        "manifests",
        "consumers",
        "analysis_inputs",
    }
)
_OPTIONAL_SET_FIELDS_BY_COLLECTION = {
    "components": ("roles", "exclude", "manifests", "consumers", "analysis_inputs"),
    "associated_artifacts": ("exclude",),
    "relationships": (),
}


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
class ResolutionProvenance:
    """Derived declaration origin metadata for one resolved profile.

    Provenance is explanatory runtime metadata. It is not lifecycle ownership,
    evidence provenance, or canonical Project Profile state.
    """

    components: Mapping[str, str]
    associated_artifacts: Mapping[str, str]
    relationships: Mapping[str, str]
    removals: Mapping[str, tuple[str, ...]]

    def __post_init__(self) -> None:
        component_origins = dict(self.components)
        artifact_origins = dict(self.associated_artifacts)
        relationship_origins = dict(self.relationships)
        removal_rows = {key: tuple(value) for key, value in dict(self.removals).items()}

        for label, origins in (
            ("component", component_origins),
            ("associated-artifact", artifact_origins),
            ("relationship", relationship_origins),
        ):
            invalid = sorted(set(origins.values()) - _PROVENANCE_VALUES)
            if invalid:
                raise TemplateMaterializationError(
                    f"Unknown {label} provenance value(s): {', '.join(invalid)}."
                )

        invalid_removal_groups = sorted(set(removal_rows) - set(_STABLE_ID_COLLECTIONS))
        if invalid_removal_groups:
            raise TemplateMaterializationError(
                "Unknown removal provenance collection(s): "
                + ", ".join(invalid_removal_groups)
                + "."
            )

        object.__setattr__(self, "components", MappingProxyType(component_origins))
        object.__setattr__(self, "associated_artifacts", MappingProxyType(artifact_origins))
        object.__setattr__(self, "relationships", MappingProxyType(relationship_origins))
        object.__setattr__(self, "removals", MappingProxyType(removal_rows))

    def as_dict(self) -> dict[str, object]:
        return {
            "components": dict(self.components),
            "associated_artifacts": dict(self.associated_artifacts),
            "relationships": dict(self.relationships),
            "removals": {key: list(value) for key, value in self.removals.items()},
        }


@dataclass(frozen=True)
class ResolvedProfile:
    """Source-preserving runtime view of one resolved Responsibility Map."""

    source_payload: dict[str, object]
    effective_payload: dict[str, object]
    source_mode: str
    template_id: str | None
    template_revision: str | None
    effective_map_digest: str
    provenance: ResolutionProvenance

    def __post_init__(self) -> None:
        object.__setattr__(self, "source_payload", copy.deepcopy(self.source_payload))
        object.__setattr__(self, "effective_payload", copy.deepcopy(self.effective_payload))

    @property
    def payload(self) -> dict[str, object]:
        """Current internal compatibility accessor for the effective payload.

        New resolved-view consumers should use ``effective_payload`` explicitly.
        The original project declaration is always available as ``source_payload``.
        """

        return self.effective_payload

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
            "effective_map_digest": self.effective_map_digest,
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


def _canonical_json_sort_key(value: object) -> str:
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False)


def _normalize_generic(value: object, *, field_name: str | None = None) -> object:
    if isinstance(value, Mapping):
        normalized: dict[str, object] = {}
        for key, item in value.items():
            if not isinstance(key, str):
                raise TemplateMaterializationError(
                    "Effective-map digest requires string mapping keys."
                )
            normalized[key] = _normalize_generic(item, field_name=key)
        return normalized
    if isinstance(value, list):
        normalized_items = [_normalize_generic(item) for item in value]
        if field_name in _SET_VALUED_FIELDS:
            return sorted(normalized_items, key=_canonical_json_sort_key)
        return normalized_items
    if isinstance(value, tuple):
        normalized_items = [_normalize_generic(item) for item in value]
        if field_name in _SET_VALUED_FIELDS:
            return sorted(normalized_items, key=_canonical_json_sort_key)
        return normalized_items
    return value


def _normalize_stable_id_collection(
    value: object,
    *,
    collection_name: str,
) -> list[dict[str, object]]:
    if value is None:
        return []
    if not isinstance(value, list):
        raise TemplateMaterializationError(
            f"Effective-map {collection_name} must be a list for digest calculation."
        )

    normalized: list[dict[str, object]] = []
    seen: set[str] = set()
    for item in value:
        if not isinstance(item, Mapping):
            raise TemplateMaterializationError(
                f"Effective-map {collection_name} entries must be mappings for digest calculation."
            )
        item_id = item.get("id")
        if not isinstance(item_id, str) or not item_id:
            raise TemplateMaterializationError(
                f"Effective-map {collection_name} entry requires a non-empty id for digest calculation."
            )
        if item_id in seen:
            raise TemplateMaterializationError(
                f"Effective-map {collection_name} IDs are not unique: {item_id}."
            )
        seen.add(item_id)

        row = copy.deepcopy(dict(item))
        for field in _OPTIONAL_SET_FIELDS_BY_COLLECTION[collection_name]:
            row.setdefault(field, [])
        normalized_row = _normalize_generic(row)
        if not isinstance(normalized_row, dict):
            raise AssertionError("Normalized stable-ID entry must remain a mapping.")
        normalized.append(normalized_row)

    normalized.sort(key=lambda row: str(row["id"]))
    return normalized


def _normalize_dependency_policy(value: object) -> dict[str, object]:
    if not isinstance(value, Mapping):
        raise TemplateMaterializationError(
            "Effective-map component_dependency_policy must be a mapping for digest calculation."
        )

    policy = copy.deepcopy(dict(value))
    for key in ("allow", "deny"):
        raw_relations = policy.get(key, [])
        if not isinstance(raw_relations, list):
            raise TemplateMaterializationError(
                f"component_dependency_policy.{key} must be a list for digest calculation."
            )
        relations: list[dict[str, object]] = []
        for relation in raw_relations:
            if not isinstance(relation, Mapping):
                raise TemplateMaterializationError(
                    f"component_dependency_policy.{key} entries must be mappings."
                )
            source = relation.get("from")
            target = relation.get("to")
            if not isinstance(source, str) or not source or not isinstance(target, str) or not target:
                raise TemplateMaterializationError(
                    f"component_dependency_policy.{key} entries require non-empty from/to IDs."
                )
            normalized_relation = _normalize_generic(copy.deepcopy(dict(relation)))
            if not isinstance(normalized_relation, dict):
                raise AssertionError("Normalized dependency-policy relation must remain a mapping.")
            relations.append(normalized_relation)
        relations.sort(key=lambda item: (str(item["from"]), str(item["to"])))
        policy[key] = relations

    normalized_policy = _normalize_generic(policy)
    if not isinstance(normalized_policy, dict):
        raise AssertionError("Normalized dependency policy must remain a mapping.")
    return normalized_policy


def effective_map_semantics(payload: Mapping[str, object]) -> dict[str, object]:
    """Return the canonical semantic object used for effective-map identity.

    Source mode, template identity, Specification binding, provenance, and
    serialization-only details are intentionally excluded.
    """

    semantic: dict[str, object] = {
        "components": _normalize_stable_id_collection(
            payload.get("components"), collection_name="components"
        ),
        "associated_artifacts": _normalize_stable_id_collection(
            payload.get("associated_artifacts"), collection_name="associated_artifacts"
        ),
        "relationships": _normalize_stable_id_collection(
            payload.get("relationships"), collection_name="relationships"
        ),
    }

    if "component_dependency_policy" in payload:
        semantic["component_dependency_policy"] = _normalize_dependency_policy(
            payload["component_dependency_policy"]
        )

    policies = payload.get("policies")
    if not isinstance(policies, Mapping):
        raise TemplateMaterializationError(
            "Effective-map policies must be a mapping for digest calculation."
        )
    semantic["policies"] = _normalize_generic(copy.deepcopy(dict(policies)))
    return semantic


def calculate_effective_map_digest(payload: Mapping[str, object]) -> str:
    """Return deterministic SHA-256 identity for effective architecture semantics."""

    encoded = json.dumps(
        effective_map_semantics(payload),
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


def _mapping_items(value: object, *, label: str) -> list[dict[str, object]]:
    if value is None:
        return []
    if not isinstance(value, list):
        raise TemplateMaterializationError(f"{label} collection must be a list.")
    items: list[dict[str, object]] = []
    for item in value:
        if not isinstance(item, Mapping):
            raise TemplateMaterializationError(f"{label} entry must be a mapping.")
        items.append(copy.deepcopy(dict(item)))
    return items


def _removal_ids(value: object, *, label: str) -> list[str]:
    if value is None:
        return []
    if not isinstance(value, list):
        raise TemplateMaterializationError(f"{label} removals must be a list.")
    removals: list[str] = []
    for item in value:
        if not isinstance(item, str) or not item:
            raise TemplateMaterializationError(f"{label} removal IDs must be non-empty strings.")
        removals.append(item)
    if len(removals) != len(set(removals)):
        raise TemplateMaterializationError(f"{label} removal IDs are not unique.")
    return removals


def _merge_id_collection(
    base: object,
    replacements: object,
    removals: object,
    *,
    label: str,
) -> tuple[list[dict[str, object]], dict[str, str], tuple[str, ...]]:
    base_items = _mapping_items(base, label=f"Template {label}")
    replacement_items = _mapping_items(replacements, label=f"Override {label}")
    removal_ids = _removal_ids(removals, label=label)

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
    origins: dict[str, str] = {}
    for item in base_items:
        item_id = _id(item, label)
        if item_id in removal_ids:
            continue
        replacement = replacement_by_id.pop(item_id, None)
        if replacement is not None:
            result.append(copy.deepcopy(replacement))
            origins[item_id] = PROJECT_OVERRIDE
        else:
            result.append(copy.deepcopy(item))
            origins[item_id] = TEMPLATE

    for item in replacement_items:
        item_id = _id(item, f"override {label}")
        if item_id in replacement_by_id:
            result.append(copy.deepcopy(item))
            origins[item_id] = PROJECT_EXTENSION
            replacement_by_id.pop(item_id, None)

    return result, origins, tuple(sorted(removal_ids))


def _origin_map(value: object, *, label: str, origin: str) -> dict[str, str]:
    items = _mapping_items(value, label=label)
    ids = [_id(item, label) for item in items]
    if len(ids) != len(set(ids)):
        raise TemplateMaterializationError(f"{label} IDs are not unique.")
    return {item_id: origin for item_id in ids}


def _resolved_profile(
    *,
    source_payload: dict[str, object],
    effective_payload: dict[str, object],
    source_mode: str,
    template_id: str | None,
    template_revision: str | None,
    provenance: ResolutionProvenance,
) -> ResolvedProfile:
    return ResolvedProfile(
        source_payload=source_payload,
        effective_payload=effective_payload,
        source_mode=source_mode,
        template_id=template_id,
        template_revision=template_revision,
        effective_map_digest=calculate_effective_map_digest(effective_payload),
        provenance=provenance,
    )


def materialize_profile(payload: Mapping[str, object]) -> ResolvedProfile:
    """Resolve explicit/template/hybrid input into one source-preserving view.

    Materialization is read-only: the caller payload is never mutated. Hybrid
    overrides replace matching stable IDs in place, remove only known template
    IDs, and append genuinely new project-owned IDs in declaration order.
    """

    source = copy.deepcopy(dict(payload))
    map_meta = source.get("responsibility_map")
    if not isinstance(map_meta, Mapping):
        raise TemplateMaterializationError("Profile responsibility_map must be a mapping.")

    mode = map_meta.get("mode")
    if mode == "explicit":
        effective = copy.deepcopy(source)
        provenance = ResolutionProvenance(
            components=_origin_map(
                effective.get("components"), label="component", origin=PROJECT_EXPLICIT
            ),
            associated_artifacts=_origin_map(
                effective.get("associated_artifacts"),
                label="associated-artifact",
                origin=PROJECT_EXPLICIT,
            ),
            relationships=_origin_map(
                effective.get("relationships"), label="relationship", origin=PROJECT_EXPLICIT
            ),
            removals={
                "components": (),
                "associated_artifacts": (),
                "relationships": (),
            },
        )
        return _resolved_profile(
            source_payload=source,
            effective_payload=effective,
            source_mode="explicit",
            template_id=None,
            template_revision=None,
            provenance=provenance,
        )
    if mode not in {"template", "hybrid"}:
        raise TemplateMaterializationError(f"Unsupported Responsibility Map mode: {mode!r}.")

    template_ref = map_meta.get("template")
    if not isinstance(template_ref, Mapping):
        raise TemplateMaterializationError("Template-backed map requires a template reference.")
    definition = resolve_template(template_ref)
    effective_map = copy.deepcopy(definition.map_payload)

    component_origins = _origin_map(
        effective_map.get("components"), label="component", origin=TEMPLATE
    )
    artifact_origins = _origin_map(
        effective_map.get("associated_artifacts"), label="associated-artifact", origin=TEMPLATE
    )
    relationship_origins = _origin_map(
        effective_map.get("relationships"), label="relationship", origin=TEMPLATE
    )
    removal_provenance: dict[str, tuple[str, ...]] = {
        "components": (),
        "associated_artifacts": (),
        "relationships": (),
    }

    if mode == "hybrid":
        overrides = map_meta.get("overrides")
        if not isinstance(overrides, Mapping):
            raise TemplateMaterializationError("Hybrid map requires an overrides mapping.")

        components, component_origins, component_removals = _merge_id_collection(
            effective_map.get("components", []),
            overrides.get("components", []),
            overrides.get("remove_component_ids", []),
            label="component",
        )
        artifacts, artifact_origins, artifact_removals = _merge_id_collection(
            effective_map.get("associated_artifacts", []),
            overrides.get("associated_artifacts", []),
            overrides.get("remove_associated_artifact_ids", []),
            label="associated-artifact",
        )
        relationships, relationship_origins, relationship_removals = _merge_id_collection(
            effective_map.get("relationships", []),
            overrides.get("relationships", []),
            overrides.get("remove_relationship_ids", []),
            label="relationship",
        )
        effective_map["components"] = components
        effective_map["associated_artifacts"] = artifacts
        effective_map["relationships"] = relationships
        removal_provenance = {
            "components": component_removals,
            "associated_artifacts": artifact_removals,
            "relationships": relationship_removals,
        }

    effective = copy.deepcopy(source)
    for key in (
        "components",
        "associated_artifacts",
        "relationships",
        "component_dependency_policy",
    ):
        effective.pop(key, None)
        if key in effective_map:
            effective[key] = copy.deepcopy(effective_map[key])
    effective["responsibility_map"] = {"mode": "explicit"}

    provenance = ResolutionProvenance(
        components=component_origins,
        associated_artifacts=artifact_origins,
        relationships=relationship_origins,
        removals=removal_provenance,
    )
    return _resolved_profile(
        source_payload=source,
        effective_payload=effective,
        source_mode=str(mode),
        template_id=definition.id,
        template_revision=definition.revision,
        provenance=provenance,
    )
