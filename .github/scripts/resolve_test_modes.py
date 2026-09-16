from __future__ import annotations

import argparse
import json
import re
import runpy
import subprocess
import sys
from pathlib import Path, PurePosixPath
from typing import Any, Iterable

import yaml


VALIDATOR_PATH = Path(__file__).with_name("validate_test_modes.py")
VALIDATE_REGISTRY = runpy.run_path(str(VALIDATOR_PATH))["validate_registry"]
SELF_PROFILE_PATH = "developer/profiles/ptsip-repository.yaml"


class TestModeSelectionError(ValueError):
    """Stable fail-closed error for Test Mode selection."""


def normalize_repo_path(value: str) -> str:
    if not isinstance(value, str) or not value.strip():
        raise TestModeSelectionError("repository path must be a non-empty string")

    normalized = value.strip().replace("\\", "/")
    while normalized.startswith("./"):
        normalized = normalized[2:]

    if re.match(r"^[A-Za-z]:/", normalized):
        raise TestModeSelectionError(f"repository path must be relative: {value!r}")

    path = PurePosixPath(normalized)
    if path.is_absolute() or normalized == "." or ".." in path.parts:
        raise TestModeSelectionError(
            f"repository path escapes repository root: {value!r}"
        )
    return path.as_posix()


def normalize_changed_files(values: Iterable[str]) -> list[str]:
    return sorted({normalize_repo_path(value) for value in values})


def _glob_regex(pattern: str) -> re.Pattern[str]:
    pattern = normalize_repo_path(pattern)
    output = ["^"]
    index = 0
    while index < len(pattern):
        if pattern[index : index + 3] == "**/":
            output.append("(?:.*/)?")
            index += 3
        elif pattern[index : index + 2] == "**":
            output.append(".*")
            index += 2
        elif pattern[index] == "*":
            output.append("[^/]*")
            index += 1
        elif pattern[index] == "?":
            output.append("[^/]")
            index += 1
        else:
            output.append(re.escape(pattern[index]))
            index += 1
    output.append("$")
    return re.compile("".join(output))


def matches_pattern(path: str, pattern: str) -> bool:
    return bool(_glob_regex(pattern).match(normalize_repo_path(path)))


def _load_mapping(path: Path, *, label: str) -> dict[str, Any]:
    payload = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
    if not isinstance(payload, dict):
        raise TestModeSelectionError(f"{label} root must be a mapping")
    return payload


def load_valid_registry(
    registry_path: Path,
    profile_path: Path,
    repo_root: Path,
) -> tuple[dict[str, Any], dict[str, Any]]:
    errors = VALIDATE_REGISTRY(registry_path, profile_path, repo_root)
    if errors:
        raise TestModeSelectionError(
            "invalid Test Mode Registry: " + "; ".join(errors)
        )
    registry = _load_mapping(registry_path, label="Test Mode Registry")
    profile = _load_mapping(profile_path, label="Project Profile")
    return registry, profile


def _modes(registry: dict[str, Any]) -> list[dict[str, Any]]:
    modes = registry.get("modes")
    if not isinstance(modes, list) or not all(
        isinstance(mode, dict) for mode in modes
    ):
        raise TestModeSelectionError(
            "Test Mode Registry modes must contain mappings"
        )
    return modes


def _components(profile: dict[str, Any]) -> list[dict[str, Any]]:
    components = profile.get("components")
    if not isinstance(components, list) or not all(
        isinstance(component, dict) for component in components
    ):
        raise TestModeSelectionError(
            "Project Profile components must contain mappings"
        )
    return components


def _component_index(profile: dict[str, Any]) -> dict[str, dict[str, Any]]:
    result: dict[str, dict[str, Any]] = {}
    for component in _components(profile):
        component_id = component.get("id")
        if not isinstance(component_id, str) or not component_id:
            raise TestModeSelectionError(
                "Project Profile component has invalid id"
            )
        if component_id in result:
            raise TestModeSelectionError(
                f"Project Profile duplicate component id: {component_id}"
            )
        result[component_id] = component
    return result


def _patterns(component: dict[str, Any], field: str) -> list[str]:
    value = component.get(field, [])
    if not isinstance(value, list) or not all(
        isinstance(item, str) for item in value
    ):
        raise TestModeSelectionError(
            f"component {component.get('id')!r} has invalid {field}"
        )
    return list(value)


def _mode_matches_path(
    mode: dict[str, Any],
    components: dict[str, dict[str, Any]],
    path: str,
) -> bool:
    component_ref = mode.get("component_ref")
    component = components.get(str(component_ref))
    if component is None:
        raise TestModeSelectionError(
            f"Test Mode references missing component: {component_ref!r}"
        )
    patterns = _patterns(component, "analysis_inputs") + _patterns(
        component, "include"
    )
    return any(matches_pattern(path, pattern) for pattern in patterns)


def _path_has_declared_owner(
    profile: dict[str, Any],
    path: str,
) -> bool:
    for component in _components(profile):
        if any(
            matches_pattern(path, pattern)
            for pattern in _patterns(component, "include")
        ):
            return True

    associated = profile.get("associated_artifacts", [])
    if not isinstance(associated, list):
        raise TestModeSelectionError(
            "Project Profile associated_artifacts must be a list"
        )
    for artifact in associated:
        if not isinstance(artifact, dict):
            raise TestModeSelectionError(
                "Project Profile associated_artifacts must contain mappings"
            )
        include = artifact.get("include", [])
        if not isinstance(include, list) or not all(
            isinstance(pattern, str) for pattern in include
        ):
            raise TestModeSelectionError(
                f"associated artifact {artifact.get('id')!r} has invalid include"
            )
        if any(matches_pattern(path, pattern) for pattern in include):
            return True

    return False


def resolve_automatic_selection(
    registry: dict[str, Any],
    profile: dict[str, Any],
    changed_files: Iterable[str],
) -> tuple[list[dict[str, Any]], list[str]]:
    changed = normalize_changed_files(changed_files)
    if not changed:
        raise TestModeSelectionError(
            "automatic Test Mode resolution received no changed files"
        )

    modes = _modes(registry)
    components = _component_index(profile)
    selected: list[dict[str, Any]] = []
    no_verification_required: list[str] = []
    unmapped: list[str] = []

    for path in changed:
        if not _path_has_declared_owner(profile, path):
            unmapped.append(path)
            continue

        matching_modes = [
            mode
            for mode in modes
            if _mode_matches_path(mode, components, path)
        ]
        if matching_modes:
            for mode in matching_modes:
                if mode not in selected:
                    selected.append(mode)
            continue

        no_verification_required.append(path)

    if unmapped:
        raise TestModeSelectionError(
            "unmapped changed paths: " + ", ".join(unmapped)
        )

    return selected, no_verification_required


def select_automatic_modes(
    registry: dict[str, Any],
    profile: dict[str, Any],
    changed_files: Iterable[str],
) -> list[dict[str, Any]]:
    selected, _ = resolve_automatic_selection(
        registry, profile, changed_files
    )
    return selected


def select_manual_modes(
    registry: dict[str, Any],
    requested_mode: str,
) -> list[dict[str, Any]]:
    requested = (requested_mode or "").strip()
    if not requested:
        raise TestModeSelectionError("requested Test Mode must not be empty")

    selected = [
        mode for mode in _modes(registry) if mode.get("id") == requested
    ]
    if not selected:
        raise TestModeSelectionError(
            f"unknown requested Test Mode: {requested}"
        )
    return selected


def build_execution_plan(
    selected_modes: Iterable[dict[str, Any]],
) -> list[dict[str, Any]]:
    plan: list[dict[str, Any]] = []
    for mode in selected_modes:
        execution = mode.get("execution")
        if not isinstance(execution, dict):
            raise TestModeSelectionError(
                f"mode {mode.get('id')!r} has invalid execution declaration"
            )
        targets = execution.get("pytest")
        if not isinstance(targets, list) or not all(
            isinstance(target, str) for target in targets
        ):
            raise TestModeSelectionError(
                f"mode {mode.get('id')!r} has invalid pytest targets"
            )
        plan.append(
            {
                "id": mode["id"],
                "component_ref": mode["component_ref"],
                "pytest": list(targets),
            }
        )
    return plan


def changed_files_from_git(
    repo_root: Path,
    base: str,
    head: str,
) -> list[str]:
    base = (base or "").strip()
    head = (head or "HEAD").strip()

    if not base:
        parent = subprocess.run(
            ["git", "rev-parse", f"{head}^"],
            cwd=repo_root,
            check=False,
            capture_output=True,
            text=True,
        )
        if parent.returncode == 0 and parent.stdout.strip():
            base = parent.stdout.strip()

    if base:
        command = ["git", "diff", "--name-only", base, head]
    else:
        command = [
            "git",
            "diff-tree",
            "--root",
            "--no-commit-id",
            "--name-only",
            "-r",
            head,
        ]

    result = subprocess.run(
        command,
        cwd=repo_root,
        check=True,
        capture_output=True,
        text=True,
    )
    return normalize_changed_files(
        line for line in result.stdout.splitlines() if line.strip()
    )


def _result_payload(
    selected: list[dict[str, Any]],
    changed_files: list[str],
    no_verification_required: list[str] | None = None,
) -> dict[str, Any]:
    plan = build_execution_plan(selected)
    return {
        "resolution": (
            "SELECTED" if plan else "NO_TEST_MODE_REQUIRED"
        ),
        "changed_files": changed_files,
        "selected_ids": [item["id"] for item in plan],
        "no_verification_required": no_verification_required or [],
        "plan": plan,
    }


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description=(
            "Resolve repository Test Modes from the canonical Project Profile"
        )
    )
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--registry", default=".github/test_modes.yaml")
    parser.add_argument("--profile", default=SELF_PROFILE_PATH)

    subparsers = parser.add_subparsers(dest="command", required=True)

    automatic = subparsers.add_parser("automatic")
    automatic.add_argument("--base", default="")
    automatic.add_argument("--head", default="HEAD")
    automatic.add_argument("--changed-file", action="append", default=[])

    manual = subparsers.add_parser("manual")
    manual.add_argument("--mode", required=True)
    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    repo_root = Path(args.repo_root).resolve()
    registry_path = repo_root / args.registry
    profile_path = repo_root / args.profile

    try:
        registry, profile = load_valid_registry(
            registry_path, profile_path, repo_root
        )
        if args.command == "automatic":
            changed = (
                normalize_changed_files(args.changed_file)
                if args.changed_file
                else changed_files_from_git(repo_root, args.base, args.head)
            )
            selected, no_verification_required = (
                resolve_automatic_selection(
                    registry, profile, changed
                )
            )
        else:
            changed = []
            no_verification_required = []
            selected = select_manual_modes(registry, args.mode)

        print(
            json.dumps(
                _result_payload(
                    selected,
                    changed,
                    no_verification_required,
                ),
                separators=(",", ":"),
            )
        )
        return 0
    except (
        OSError,
        subprocess.CalledProcessError,
        TestModeSelectionError,
        yaml.YAMLError,
    ) as exc:
        print(f"Test Mode resolver error: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
