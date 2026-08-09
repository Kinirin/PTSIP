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
                        f"boundaries: {plane}:{normalized!r} overlaps {previous_plane}:{previous_root!r}; use components for nested ownership"
                    )
            seen.append((str(plane), normalized))
    return errors


def _component_policy_errors(payload: dict[str, object], component_ids: set[str]) -> list[str]:
    policy = payload.get("component_dependency_policy")
    if not isinstance(policy, dict):
        return []
    errors: list[str] = []
    for relation_kind in ("allow", "deny"):
        relations = policy.get(relation_kind, [])
        if not isinstance(relations, list):
            continue
        for index, relation in enumerate(relations):
            if not isinstance(relation, dict):
                continue
            for endpoint in ("from", "to"):
                value = relation.get(endpoint)
                if isinstance(value, str) and value not in component_ids:
                    errors.append(
                        f"component_dependency_policy.{relation_kind}.{index}.{endpoint}: component {value!r} is not declared"
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
            warnings.append("Specification binding has no immutable revision; reproducibility is weaker for a draft specification.")
        elif SPEC_REVISION != "UNRELEASED" and revision != SPEC_REVISION:
            errors.append(
                f"Profile revision {revision!r} is not supported by tooling snapshot {SPEC_REVISION!r}."
            )

        errors.extend(_root_overlap_errors(payload))

        components = payload.get("components")
        if isinstance(components, list):
            component_ids = [str(item.get("id")) for item in components if isinstance(item, dict)]
            duplicates = sorted({item for item in component_ids if component_ids.count(item) > 1})
            if duplicates:
                errors.append(f"components: duplicate component ids: {', '.join(duplicates)}")
            component_set = set(component_ids)
            errors.extend(_component_policy_errors(payload, component_set))
            partition = partition_components(repository_root, [item for item in components if isinstance(item, dict)])
            details["component_partition"] = partition.as_dict()
            if partition.conflicts:
                errors.append(f"components: {len(partition.conflicts)} tracked path(s) have equal-specificity ownership conflicts")
            if partition.unmatched_selectors:
                errors.append(
                    "components: include selector(s) matched no tracked files: " + ", ".join(partition.unmatched_selectors[:20])
                )
            if partition.scan_errors:
                warnings.append(f"Component partition scan was incomplete: {len(partition.scan_errors)} repository scan error(s).")
            if partition.unassigned_files:
                warnings.append(
                    f"{len(partition.unassigned_files)} tracked file(s) are outside declared component selectors; this is not automatically a PTSIP violation."
                )
        else:
            warnings.append(
                "Profile uses boundary roots only. This shorthand cannot express nested/file-level component ownership; use components when precise partitioning is required."
            )

    return ValidationResult(str(profile), not errors, errors, warnings, details or None)
