from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path, PurePosixPath
from typing import Any

import yaml


REGISTRY_VERSION = 1
_MODE_ID = re.compile(r"^[a-z0-9][a-z0-9-]*$")
_ROOT_KEYS = {"version", "modes"}
_MODE_KEYS = {"id", "component_ref", "execution", "watch"}
_EXECUTION_KEYS = {"pytest"}
_ARCHITECTURE_KEYS = {
    "classification",
    "roles",
    "purpose",
    "shipped",
    "runtime_required",
    "executable",
    "release_owner",
    "compatibility_owner",
    "analysis_inputs",
}
_GLOB_CHARS = set("*?[]")


def _load_yaml(path: Path, *, label: str) -> tuple[dict[str, Any] | None, list[str]]:
    try:
        payload = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
    except FileNotFoundError:
        return None, [f"{label} not found: {path}"]
    except yaml.YAMLError as exc:
        return None, [f"{label} is not valid YAML: {exc}"]

    if not isinstance(payload, dict):
        return None, [f"{label} root must be a mapping"]
    return payload, []


def _validate_relative_posix_path(value: object, *, label: str, allow_glob: bool) -> list[str]:
    if not isinstance(value, str) or not value.strip():
        return [f"{label} must be a non-empty string"]
    if "\\" in value:
        return [f"{label} must use repository-relative POSIX separators: {value!r}"]

    path = PurePosixPath(value)
    if path.is_absolute() or ".." in path.parts:
        return [f"{label} must stay inside the repository: {value!r}"]
    if not allow_glob and any(char in value for char in _GLOB_CHARS):
        return [f"{label} must be an exact file or directory path, not a glob: {value!r}"]
    return []


def _component_index(profile: dict[str, Any]) -> tuple[dict[str, dict[str, Any]], list[str]]:
    components = profile.get("components")
    if not isinstance(components, list):
        return {}, ["Project Profile components must be a list"]

    index: dict[str, dict[str, Any]] = {}
    errors: list[str] = []
    for position, component in enumerate(components):
        if not isinstance(component, dict):
            errors.append(f"Project Profile component[{position}] must be a mapping")
            continue
        component_id = component.get("id")
        if not isinstance(component_id, str) or not component_id:
            errors.append(f"Project Profile component[{position}] has no valid id")
            continue
        if component_id in index:
            errors.append(f"Project Profile contains duplicate component id: {component_id}")
            continue
        index[component_id] = component
    return index, errors


def validate_registry(registry_path: Path, profile_path: Path, repo_root: Path) -> list[str]:
    registry, errors = _load_yaml(registry_path, label="Test Mode Registry")
    if registry is None:
        return errors

    profile, profile_errors = _load_yaml(profile_path, label="Project Profile")
    errors.extend(profile_errors)
    if profile is None:
        return errors

    unknown_root = sorted(set(registry) - _ROOT_KEYS)
    if unknown_root:
        errors.append(f"Test Mode Registry contains unsupported root fields: {unknown_root}")

    if registry.get("version") != REGISTRY_VERSION:
        errors.append(f"Test Mode Registry version must be {REGISTRY_VERSION}")

    modes = registry.get("modes")
    if not isinstance(modes, list):
        errors.append("Test Mode Registry modes must be a list")
        return errors

    components, component_errors = _component_index(profile)
    errors.extend(component_errors)

    seen_ids: set[str] = set()
    seen_pytest_targets: dict[str, str] = {}
    for position, mode in enumerate(modes):
        prefix = f"mode[{position}]"
        if not isinstance(mode, dict):
            errors.append(f"{prefix} must be a mapping")
            continue

        architecture_fields = sorted(set(mode) & _ARCHITECTURE_KEYS)
        if architecture_fields:
            errors.append(
                f"{prefix} duplicates architecture authority fields {architecture_fields}; "
                "use component_ref instead"
            )

        unknown_mode = sorted(set(mode) - _MODE_KEYS)
        if unknown_mode:
            errors.append(f"{prefix} contains unsupported fields: {unknown_mode}")

        missing = sorted(_MODE_KEYS - set(mode))
        if missing:
            errors.append(f"{prefix} is missing required fields: {missing}")

        mode_id = mode.get("id")
        if not isinstance(mode_id, str) or not _MODE_ID.fullmatch(mode_id):
            errors.append(f"{prefix}.id must match {_MODE_ID.pattern}")
        elif mode_id in seen_ids:
            errors.append(f"duplicate Test Mode id: {mode_id}")
        else:
            seen_ids.add(mode_id)

        component_ref = mode.get("component_ref")
        if not isinstance(component_ref, str) or not component_ref:
            errors.append(f"{prefix}.component_ref must be a non-empty string")
        else:
            component = components.get(component_ref)
            if component is None:
                errors.append(f"{prefix}.component_ref does not exist in ptsip.yaml: {component_ref}")
            else:
                roles = component.get("roles", [])
                if not isinstance(roles, list) or "VERIFICATION" not in roles:
                    errors.append(
                        f"{prefix}.component_ref must reference a VERIFICATION component: {component_ref}"
                    )

        execution = mode.get("execution")
        if not isinstance(execution, dict):
            errors.append(f"{prefix}.execution must be a mapping")
        else:
            unknown_execution = sorted(set(execution) - _EXECUTION_KEYS)
            if unknown_execution:
                errors.append(f"{prefix}.execution contains unsupported fields: {unknown_execution}")

            pytest_targets = execution.get("pytest")
            if not isinstance(pytest_targets, list) or not pytest_targets:
                errors.append(f"{prefix}.execution.pytest must be a non-empty list")
            else:
                for target_position, target in enumerate(pytest_targets):
                    label = f"{prefix}.execution.pytest[{target_position}]"
                    path_errors = _validate_relative_posix_path(target, label=label, allow_glob=False)
                    errors.extend(path_errors)
                    if not path_errors and isinstance(target, str):
                        owner = seen_pytest_targets.get(target)
                        current_owner = mode_id if isinstance(mode_id, str) else prefix
                        if owner is not None:
                            errors.append(
                                f"{label} duplicates pytest target {target!r} already owned by mode {owner!r}"
                            )
                        else:
                            seen_pytest_targets[target] = current_owner

                        parts = PurePosixPath(target).parts
                        if not repo_root.joinpath(*parts).exists():
                            errors.append(f"{label} does not exist in the repository: {target}")

        watch = mode.get("watch")
        if not isinstance(watch, list) or not watch:
            errors.append(f"{prefix}.watch must be a non-empty list")
        else:
            for watch_position, pattern in enumerate(watch):
                errors.extend(
                    _validate_relative_posix_path(
                        pattern,
                        label=f"{prefix}.watch[{watch_position}]",
                        allow_glob=True,
                    )
                )

    return errors


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Validate the repository Test Mode Registry v1")
    parser.add_argument("--registry", default=".github/test_modes.yaml")
    parser.add_argument("--profile", default="ptsip.yaml")
    parser.add_argument("--repo-root", default=".")
    args = parser.parse_args(argv)

    repo_root = Path(args.repo_root).resolve()
    registry_path = repo_root / args.registry
    profile_path = repo_root / args.profile
    errors = validate_registry(registry_path, profile_path, repo_root)

    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1

    payload = yaml.safe_load(registry_path.read_text(encoding="utf-8-sig"))
    modes = payload.get("modes", []) if isinstance(payload, dict) else []
    print(f"Test Mode Registry v{REGISTRY_VERSION} valid: {len(modes)} mode(s)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
