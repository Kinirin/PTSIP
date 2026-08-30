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

CONTROL_PLANE_WATCH = (
    ".github/test_modes.yaml",
    ".github/scripts/validate_test_modes.py",
    ".github/scripts/resolve_test_modes.py",
    ".github/workflows/tooling-test.yml",
    "tests/ptsip/test_modes/**",
    "ptsip.yaml",
)


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
        raise TestModeSelectionError(f"repository path escapes repository root: {value!r}")
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


def matches_watch(path: str, pattern: str) -> bool:
    return bool(_glob_regex(pattern).match(normalize_repo_path(path)))


def _load_registry(registry_path: Path) -> dict[str, Any]:
    payload = yaml.safe_load(registry_path.read_text(encoding="utf-8-sig"))
    if not isinstance(payload, dict):
        raise TestModeSelectionError("Test Mode Registry root must be a mapping")
    modes = payload.get("modes")
    if not isinstance(modes, list):
        raise TestModeSelectionError("Test Mode Registry modes must be a list")
    return payload


def load_valid_registry(registry_path: Path, profile_path: Path, repo_root: Path) -> dict[str, Any]:
    errors = VALIDATE_REGISTRY(registry_path, profile_path, repo_root)
    if errors:
        raise TestModeSelectionError("invalid Test Mode Registry: " + "; ".join(errors))
    return _load_registry(registry_path)


def _modes(registry: dict[str, Any]) -> list[dict[str, Any]]:
    modes = registry.get("modes")
    if not isinstance(modes, list) or not all(isinstance(mode, dict) for mode in modes):
        raise TestModeSelectionError("Test Mode Registry modes must contain mappings")
    return modes


def _is_control_plane_change(path: str) -> bool:
    return any(matches_watch(path, pattern) for pattern in CONTROL_PLANE_WATCH)


def select_automatic_modes(
    registry: dict[str, Any],
    changed_files: Iterable[str],
) -> list[dict[str, Any]]:
    changed = normalize_changed_files(changed_files)
    modes = _modes(registry)

    if any(_is_control_plane_change(path) for path in changed):
        return list(modes)

    selected: list[dict[str, Any]] = []
    for mode in modes:
        watch = mode.get("watch")
        if not isinstance(watch, list):
            raise TestModeSelectionError(f"mode {mode.get('id')!r} has invalid watch declaration")
        if any(matches_watch(path, pattern) for path in changed for pattern in watch):
            selected.append(mode)
    return selected


def select_manual_modes(
    registry: dict[str, Any],
    requested_mode: str,
) -> list[dict[str, Any]]:
    modes = _modes(registry)
    requested = (requested_mode or "all").strip()
    if requested == "all":
        return list(modes)

    selected = [mode for mode in modes if mode.get("id") == requested]
    if not selected:
        raise TestModeSelectionError(f"unknown requested Test Mode: {requested}")
    return selected


def build_execution_plan(selected_modes: Iterable[dict[str, Any]]) -> list[dict[str, Any]]:
    plan: list[dict[str, Any]] = []
    for mode in selected_modes:
        execution = mode.get("execution")
        if not isinstance(execution, dict):
            raise TestModeSelectionError(f"mode {mode.get('id')!r} has invalid execution declaration")
        targets = execution.get("pytest")
        if not isinstance(targets, list) or not all(isinstance(target, str) for target in targets):
            raise TestModeSelectionError(f"mode {mode.get('id')!r} has invalid pytest targets")
        plan.append(
            {
                "id": mode["id"],
                "component_ref": mode["component_ref"],
                "pytest": list(targets),
            }
        )
    return plan


def changed_files_from_git(repo_root: Path, base: str, head: str) -> list[str]:
    base = (base or "").strip()
    head = (head or "HEAD").strip()
    zero_sha = bool(base) and set(base) == {"0"}

    if base and not zero_sha:
        command = ["git", "diff", "--name-only", base, head]
    else:
        command = ["git", "diff-tree", "--no-commit-id", "--name-only", "-r", head]

    result = subprocess.run(
        command,
        cwd=repo_root,
        check=True,
        capture_output=True,
        text=True,
    )
    return normalize_changed_files(line for line in result.stdout.splitlines() if line.strip())


def _result_payload(selected: list[dict[str, Any]], changed_files: list[str]) -> dict[str, Any]:
    plan = build_execution_plan(selected)
    return {
        "changed_files": changed_files,
        "selected_ids": [item["id"] for item in plan],
        "plan": plan,
    }


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Resolve repository Test Modes from explicit selection inputs")
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--registry", default=".github/test_modes.yaml")
    parser.add_argument("--profile", default="ptsip.yaml")

    subparsers = parser.add_subparsers(dest="command", required=True)

    automatic = subparsers.add_parser("automatic")
    automatic.add_argument("--base", default="")
    automatic.add_argument("--head", default="HEAD")
    automatic.add_argument("--changed-file", action="append", default=[])

    manual = subparsers.add_parser("manual")
    manual.add_argument("--mode", default="all")
    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    repo_root = Path(args.repo_root).resolve()
    registry_path = repo_root / args.registry
    profile_path = repo_root / args.profile

    try:
        registry = load_valid_registry(registry_path, profile_path, repo_root)
        if args.command == "automatic":
            changed = (
                normalize_changed_files(args.changed_file)
                if args.changed_file
                else changed_files_from_git(repo_root, args.base, args.head)
            )
            selected = select_automatic_modes(registry, changed)
        else:
            changed = []
            selected = select_manual_modes(registry, args.mode)

        print(json.dumps(_result_payload(selected, changed), separators=(",", ":")))
        return 0
    except (OSError, subprocess.CalledProcessError, TestModeSelectionError, yaml.YAMLError) as exc:
        print(f"Test Mode resolver error: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
