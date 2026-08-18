from __future__ import annotations

import json
from dataclasses import dataclass
from importlib.resources import files
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

from ..constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from .components import partition_components


@dataclass(frozen=True)
class ValidationResult:
    profile_path: str | None
    valid: bool
    errors: list[str]
    warnings: list[str]
    details: dict[str, object] | None = None

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


def _responsibility_map_errors(payload: dict[str, object]) -> list[str]:
    map_meta = payload.get("responsibility_map")
    if not isinstance(map_meta, dict):
        return []
    mode = map_meta.get("mode")
    errors: list[str] = []

    if mode == "explicit":
        components = payload.get("components")
        artifacts = payload.get("associated_artifacts", [])
        relationships = payload.get("relationships", [])

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

    if mode == "hybrid":
        overrides = map_meta.get("overrides")
        if not isinstance(overrides, dict):
            return errors
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

    validator = Draft202012Validator(_schema())
    errors = [
        f"{'.'.join(str(p) for p in err.absolute_path) or '<root>'}: {err.message}"
        for err in sorted(validator.iter_errors(payload), key=lambda e: list(e.absolute_path))
    ]
    warnings: list[str] = []
    details: dict[str, object] = {}

    if not errors:
        binding = payload["ptsip"]["specification"]
        if binding["source"] != SPEC_SOURCE or payload["ptsip"]["version"] != SPEC_VERSION:
            errors.append("Profile specification binding is not supported by this tooling build.")
        revision = binding.get("revision")
        if not revision:
            warnings.append(
                "Specification binding has no immutable revision; reproducibility is weaker for a draft specification."
            )
        elif SPEC_REVISION != "UNRELEASED" and revision != SPEC_REVISION:
            errors.append(
                f"Profile revision {revision!r} is not supported by tooling snapshot {SPEC_REVISION!r}."
            )

        errors.extend(_responsibility_map_errors(payload))

        map_meta = payload.get("responsibility_map")
        mode = map_meta.get("mode") if isinstance(map_meta, dict) else None
        components = payload.get("components")
        if mode == "explicit" and isinstance(components, list):
            partition = partition_components(
                repository_root,
                [item for item in components if isinstance(item, dict)],
            )
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
            if partition.unassigned_files:
                warnings.append(
                    f"{len(partition.unassigned_files)} tracked file(s) are outside declared component selectors; "
                    "this is not automatically a PTSIP violation."
                )
        elif mode in {"template", "hybrid"}:
            warnings.append(
                "Template-backed Responsibility Map is structurally valid but requires the version-bound template "
                "catalog/materialization layer before component-level conformance can be evaluated."
            )

    return ValidationResult(str(profile), not errors, errors, warnings, details or None)
