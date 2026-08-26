from __future__ import annotations

import json
from dataclasses import dataclass
from importlib.resources import files
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

from ..constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ..profile_identity import (
    ProjectProfileIdentityError,
    ProjectProfileOperation,
    ProjectProfileVersion,
    require_current_project_profile_support,
)
from .components import partition_components
from .templates import ResolvedProfile, TemplateMaterializationError, materialize_profile


@dataclass(frozen=True)
class ValidationResult:
    profile_path: str | None
    valid: bool
    errors: list[str]
    warnings: list[str]
    details: dict[str, object] | None = None
    resolved_profile: ResolvedProfile | None = None

    def as_dict(self) -> dict[str, object]:
        return {
            "profile_path": self.profile_path,
            "valid": self.valid,
            "errors": self.errors,
            "warnings": self.warnings,
            "details": self.details,
        }


def _schema() -> dict[str, object]:
    schema_path = files("ptsip").joinpath("specdata/ptsip-profile.schema.json")
    return json.loads(schema_path.read_text(encoding="utf-8"))


def _schema_errors(payload: dict[str, object], *, prefix: str = "") -> list[str]:
    validator = Draft202012Validator(_schema())
    result: list[str] = []
    for err in sorted(validator.iter_errors(payload), key=lambda item: list(item.absolute_path)):
        path = ".".join(str(part) for part in err.absolute_path) or "<root>"
        if prefix:
            path = f"{prefix}.{path}" if path != "<root>" else f"{prefix}.<root>"
        result.append(f"{path}: {err.message}")
    return result


def find_profile(repository_root: str | Path, explicit: str | Path | None = None) -> Path | None:
    if explicit:
        candidate = Path(explicit).expanduser().resolve()
        return candidate if candidate.is_file() else None
    candidate = Path(repository_root).resolve() / "ptsip.yaml"
    return candidate if candidate.is_file() else None


def _root_overlap_errors(payload: dict[str, object]) -> list[str]:
    # Retained as a compatibility helper for callers that still import it.
    # Canonical 0.3.6 profiles no longer use boundary-root shorthand.
    boundaries = payload.get("boundaries")
    if not isinstance(boundaries, dict):
        return []
    seen: list[tuple[str, str]] = []
    errors: list[str] = []
    for plane, item in boundaries.items():
        if not isinstance(item, dict):
            continue
        for root in item.get("roots", []):
            normalized = str(root).replace("\\", "/").strip("/")
            for previous_plane, previous_root in seen:
                if (
                    normalized == previous_root
                    or normalized.startswith(previous_root + "/")
                    or previous_root.startswith(normalized + "/")
                ):
                    errors.append(
                        f"boundaries: {plane}:{normalized!r} overlaps {previous_plane}:{previous_root!r}; "
                        "migrate legacy boundary roots before canonical 0.3.6 validation"
                    )
            seen.append((str(plane), normalized))
    return errors


def _component_policy_errors(payload: dict[str, object], component_ids: set[str]) -> list[str]:
    policy = payload.get("component_dependency_policy")
    if not isinstance(policy, dict):
        return []
    errors: list[str] = []
    pairs_by_kind: dict[str, set[tuple[str, str]]] = {"allow": set(), "deny": set()}
    for relation_kind in ("allow", "deny"):
        relations = policy.get(relation_kind, [])
        if not isinstance(relations, list):
            continue
        for index, relation in enumerate(relations):
            if not isinstance(relation, dict):
                continue
            source = relation.get("from")
            target = relation.get("to")
            for endpoint, value in (("from", source), ("to", target)):
                if isinstance(value, str) and value not in component_ids:
                    errors.append(
                        f"component_dependency_policy.{relation_kind}.{index}.{endpoint}: "
                        f"component {value!r} is not declared"
                    )
            if isinstance(source, str) and isinstance(target, str):
                pair = (source, target)
                if pair in pairs_by_kind[relation_kind]:
                    errors.append(
                        f"component_dependency_policy.{relation_kind}: duplicate relation "
                        f"{source!r} -> {target!r}"
                    )
                pairs_by_kind[relation_kind].add(pair)

    conflicting = sorted(pairs_by_kind["allow"] & pairs_by_kind["deny"])
    for source, target in conflicting:
        errors.append(
            f"component_dependency_policy: relation {source!r} -> {target!r} appears in both allow and deny"
        )
    return errors


def _duplicate_ids(items: object, label: str) -> list[str]:
    if not isinstance(items, list):
        return []
    ids = [str(item.get("id")) for item in items if isinstance(item, dict) and item.get("id")]
    duplicates = sorted({item for item in ids if ids.count(item) > 1})
    if not duplicates:
        return []
    return [f"{label}: duplicate ids: {', '.join(duplicates)}"]


def _source_responsibility_map_errors(payload: dict[str, object]) -> list[str]:
    """Validate source-only declaration mechanics before materialization."""

    map_meta = payload.get("responsibility_map")
    if not isinstance(map_meta, dict) or map_meta.get("mode") != "hybrid":
        return []

    overrides = map_meta.get("overrides")
    if not isinstance(overrides, dict):
        return []

    errors: list[str] = []
    errors.extend(_duplicate_ids(overrides.get("components"), "responsibility_map.overrides.components"))
    errors.extend(
        _duplicate_ids(
            overrides.get("associated_artifacts"),
            "responsibility_map.overrides.associated_artifacts",
        )
    )
    errors.extend(
        _duplicate_ids(overrides.get("relationships"), "responsibility_map.overrides.relationships")
    )

    component_override_ids = {
        str(item.get("id"))
        for item in overrides.get("components", [])
        if isinstance(item, dict) and item.get("id")
    }
    removed_component_ids = {
        str(item) for item in overrides.get("remove_component_ids", []) if isinstance(item, str)
    }
    conflict = sorted(component_override_ids & removed_component_ids)
    if conflict:
        errors.append(
            "responsibility_map.overrides: component IDs cannot be both overridden and removed: "
            + ", ".join(conflict)
        )

    artifact_override_ids = {
        str(item.get("id"))
        for item in overrides.get("associated_artifacts", [])
        if isinstance(item, dict) and item.get("id")
    }
    removed_artifact_ids = {
        str(item)
        for item in overrides.get("remove_associated_artifact_ids", [])
        if isinstance(item, str)
    }
    conflict = sorted(artifact_override_ids & removed_artifact_ids)
    if conflict:
        errors.append(
            "responsibility_map.overrides: associated-artifact IDs cannot be both overridden and removed: "
            + ", ".join(conflict)
        )

    relationship_override_ids = {
        str(item.get("id"))
        for item in overrides.get("relationships", [])
        if isinstance(item, dict) and item.get("id")
    }
    removed_relationship_ids = {
        str(item)
        for item in overrides.get("remove_relationship_ids", [])
        if isinstance(item, str)
    }
    conflict = sorted(relationship_override_ids & removed_relationship_ids)
    if conflict:
        errors.append(
            "responsibility_map.overrides: relationship IDs cannot be both overridden and removed: "
            + ", ".join(conflict)
        )

    return errors


def _responsibility_map_errors(payload: dict[str, object]) -> list[str]:
    """Validate one materialized explicit-form effective Responsibility Map."""

    map_meta = payload.get("responsibility_map")
    if not isinstance(map_meta, dict):
        return ["responsibility_map: effective map metadata is missing"]
    if map_meta.get("mode") != "explicit":
        return ["responsibility_map: effective map must resolve to explicit mode"]

    components = payload.get("components")
    artifacts = payload.get("associated_artifacts", [])
    relationships = payload.get("relationships", [])
    errors: list[str] = []

    errors.extend(_duplicate_ids(components, "components"))
    errors.extend(_duplicate_ids(artifacts, "associated_artifacts"))
    errors.extend(_duplicate_ids(relationships, "relationships"))

    component_ids = {
        str(item.get("id"))
        for item in components or []
        if isinstance(item, dict) and item.get("id")
    } if isinstance(components, list) else set()
    artifact_ids = {
        str(item.get("id"))
        for item in artifacts
        if isinstance(item, dict) and item.get("id")
    } if isinstance(artifacts, list) else set()

    collisions = sorted(component_ids & artifact_ids)
    if collisions:
        errors.append(
            "responsibility_map: component and associated-artifact IDs share one endpoint namespace; "
            f"collisions: {', '.join(collisions)}"
        )

    endpoint_ids = component_ids | artifact_ids
    relation_pairs: set[tuple[str, str, str]] = set()
    anchor_connections: set[str] = set()
    if isinstance(relationships, list):
        for index, relation in enumerate(relationships):
            if not isinstance(relation, dict):
                continue
            source = relation.get("from")
            target = relation.get("to")
            relation_type = relation.get("type")
            for endpoint, value in (("from", source), ("to", target)):
                if isinstance(value, str) and value not in endpoint_ids:
                    errors.append(
                        f"relationships.{index}.{endpoint}: endpoint {value!r} is not declared"
                    )
            if isinstance(source, str) and isinstance(target, str) and isinstance(relation_type, str):
                key = (source, target, relation_type)
                if key in relation_pairs:
                    errors.append(
                        f"relationships: duplicate semantic edge {source!r} --{relation_type}--> {target!r}"
                    )
                relation_pairs.add(key)

    if isinstance(artifacts, list):
        for index, artifact in enumerate(artifacts):
            if not isinstance(artifact, dict):
                continue
            artifact_id = artifact.get("id")
            anchor = artifact.get("anchor")
            if isinstance(anchor, str) and anchor not in component_ids:
                errors.append(
                    f"associated_artifacts.{index}.anchor: anchor {anchor!r} is not a declared component"
                )
            if isinstance(artifact_id, str) and isinstance(anchor, str):
                for source, target, _kind in relation_pairs:
                    if {source, target} == {artifact_id, anchor}:
                        anchor_connections.add(artifact_id)
                        break
        for artifact_id in sorted(artifact_ids - anchor_connections):
            errors.append(
                f"associated_artifacts: {artifact_id!r} has no typed relationship connecting it to its anchor"
            )

    errors.extend(_component_policy_errors(payload, component_ids))
    return errors


def _effective_partition_details(
    repository_root: Path,
    payload: dict[str, object],
    *,
    errors: list[str],
    warnings: list[str],
    details: dict[str, object],
) -> None:
    components = payload.get("components")
    if not isinstance(components, list):
        return

    component_items = [item for item in components if isinstance(item, dict)]
    partition = partition_components(repository_root, component_items)
    details["component_partition"] = partition.as_dict()
    if partition.conflicts:
        errors.append(
            f"components: {len(partition.conflicts)} tracked path(s) have equal-specificity ownership conflicts"
        )
    if partition.unmatched_selectors:
        errors.append(
            "components: include selector(s) matched no tracked files: "
            + ", ".join(partition.unmatched_selectors[:20])
        )
    if partition.scan_errors:
        warnings.append(
            f"Component partition scan was incomplete: {len(partition.scan_errors)} repository scan error(s)."
        )

    artifact_items = [
        item
        for item in payload.get("associated_artifacts", [])
        if isinstance(item, dict)
    ] if isinstance(payload.get("associated_artifacts", []), list) else []
    artifact_paths: set[str] = set()
    if artifact_items:
        artifact_partition = partition_components(repository_root, artifact_items)
        details["associated_artifact_partition"] = artifact_partition.as_dict()
        if artifact_partition.conflicts:
            errors.append(
                "associated_artifacts: "
                f"{len(artifact_partition.conflicts)} tracked path(s) have equal-specificity scope conflicts"
            )
        if artifact_partition.unmatched_selectors:
            errors.append(
                "associated_artifacts: include selector(s) matched no tracked files: "
                + ", ".join(artifact_partition.unmatched_selectors[:20])
            )
        artifact_paths = {item.path for item in artifact_partition.assignments}

    component_paths = {item.path for item in partition.assignments}
    overlapping_scope = sorted(component_paths & artifact_paths)
    if overlapping_scope:
        errors.append(
            "responsibility_map: tracked path(s) cannot simultaneously be classified component content "
            "and associated-artifact content: " + ", ".join(overlapping_scope[:20])
        )

    map_unassigned = sorted(set(partition.unassigned_files) - artifact_paths)
    details["responsibility_map_coverage"] = {
        "component_path_count": len(component_paths),
        "associated_artifact_path_count": len(artifact_paths),
        "unassigned_files": map_unassigned,
        "unassigned_count": len(map_unassigned),
    }
    if map_unassigned:
        warnings.append(
            f"{len(map_unassigned)} tracked file(s) are outside declared component and associated-artifact selectors; "
            "this is not automatically a PTSIP violation."
        )


def _profile_contract_identity_errors(
    payload: dict[str, object],
    *,
    details: dict[str, object],
) -> list[str]:
    ptsip = payload.get("ptsip")
    if not isinstance(ptsip, dict):
        return []
    declared = ptsip.get("version")
    if declared == SPEC_VERSION:
        details["project_profile_contract_identity"] = {
            "kind": "HISTORICAL_LABEL",
            "declared": SPEC_VERSION,
            "canonical_pp_mapping": None,
        }
        return []
    if not isinstance(declared, str) or not declared.startswith("pp."):
        return []
    try:
        version = ProjectProfileVersion.parse(declared, require_canonical=True)
        support = require_current_project_profile_support(version, ProjectProfileOperation.VALIDATE)
    except ProjectProfileIdentityError as exc:
        return [f"ptsip.version [{exc.code}]: {exc}"]
    details["project_profile_contract_identity"] = {
        "kind": "PP_CONTRACT",
        "declared": declared,
        "canonical": version.canonical,
        "compatibility_tool_target": support.tool_version,
    }
    return []


def validate_profile(repository_root: str | Path, explicit: str | Path | None = None) -> ValidationResult:
    repository_root = Path(repository_root).resolve()
    profile = find_profile(repository_root, explicit)
    if profile is None:
        return ValidationResult(
            profile_path=None,
            valid=False,
            errors=["No PTSIP project profile found."],
            warnings=["Read-only inspection and pilot commands do not require a profile."],
        )
    try:
        payload = yaml.safe_load(profile.read_text(encoding="utf-8-sig"))
    except Exception as exc:
        return ValidationResult(str(profile), False, [f"Unable to parse profile: {exc}"], [])
    if not isinstance(payload, dict):
        return ValidationResult(str(profile), False, ["<root>: profile must be a mapping"], [])

    warnings: list[str] = []
    details: dict[str, object] = {}
    errors = _profile_contract_identity_errors(payload, details=details)
    errors.extend(_schema_errors(payload))

    if not errors:
        binding = payload["ptsip"]["specification"]
        if binding["source"] != SPEC_SOURCE:
            errors.append("Profile Specification source binding is not supported by this tooling build.")
        revision = binding.get("revision")
        if not revision:
            warnings.append(
                "Specification binding has no immutable revision; reproducibility is weaker for a draft specification."
            )
        elif SPEC_REVISION != "UNRELEASED" and revision != SPEC_REVISION:
            errors.append(
                f"Profile revision {revision!r} is not supported by tooling snapshot {SPEC_REVISION!r}."
            )

        errors.extend(_source_responsibility_map_errors(payload))

    resolved = None
    if not errors:
        try:
            resolved = materialize_profile(payload)
        except TemplateMaterializationError as exc:
            errors.append(f"responsibility_map: materialization failed: {exc}")

    if resolved is not None:
        effective = resolved.effective_payload
        details["resolution"] = resolved.identity()
        details["resolution_provenance"] = resolved.provenance.as_dict()

        effective_schema_errors = _schema_errors(effective, prefix="effective")
        errors.extend(effective_schema_errors)
        if not effective_schema_errors:
            errors.extend(_responsibility_map_errors(effective))
            _effective_partition_details(
                repository_root,
                effective,
                errors=errors,
                warnings=warnings,
                details=details,
            )

    return ValidationResult(
        str(profile),
        not errors,
        errors,
        warnings,
        details or None,
        resolved,
    )
