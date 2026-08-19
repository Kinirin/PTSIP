from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path

from .source_adapters import JAVASCRIPT_TYPESCRIPT_SOURCE_SUFFIXES


_JS_EXTENSIONS = tuple(sorted(JAVASCRIPT_TYPESCRIPT_SOURCE_SUFFIXES))
_CONFIG_NAMES = ("tsconfig.json", "jsconfig.json")


@dataclass(frozen=True)
class TypeScriptAliasResolution:
    matched: bool
    resolved_path: str | None
    note: str | None = None


@dataclass(frozen=True)
class _AliasRule:
    pattern: str
    targets: tuple[str, ...]
    base_directory: Path


def _strip_jsonc_comments(source: str) -> str:
    output: list[str] = []
    index = 0
    in_string = False
    escaped = False
    while index < len(source):
        char = source[index]
        following = source[index + 1] if index + 1 < len(source) else ""
        if in_string:
            output.append(char)
            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == '"':
                in_string = False
            index += 1
            continue
        if char == '"':
            in_string = True
            output.append(char)
            index += 1
            continue
        if char == "/" and following == "/":
            index += 2
            while index < len(source) and source[index] not in "\r\n":
                index += 1
            continue
        if char == "/" and following == "*":
            index += 2
            while index + 1 < len(source) and source[index : index + 2] != "*/":
                if source[index] in "\r\n":
                    output.append(source[index])
                index += 1
            index = min(index + 2, len(source))
            continue
        output.append(char)
        index += 1
    return "".join(output)


def _strip_trailing_commas(source: str) -> str:
    output: list[str] = []
    index = 0
    in_string = False
    escaped = False
    while index < len(source):
        char = source[index]
        if in_string:
            output.append(char)
            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == '"':
                in_string = False
            index += 1
            continue
        if char == '"':
            in_string = True
            output.append(char)
            index += 1
            continue
        if char == ",":
            probe = index + 1
            while probe < len(source) and source[probe].isspace():
                probe += 1
            if probe < len(source) and source[probe] in "]}":
                index += 1
                continue
        output.append(char)
        index += 1
    return "".join(output)


def _read_jsonc(path: Path) -> dict[str, object] | None:
    try:
        source = path.read_text(encoding="utf-8-sig")
        payload = json.loads(_strip_trailing_commas(_strip_jsonc_comments(source)))
    except (OSError, UnicodeError, json.JSONDecodeError):
        return None
    return payload if isinstance(payload, dict) else None


def _within_root(root: Path, path: Path) -> bool:
    try:
        return path.is_relative_to(root)
    except (OSError, ValueError):
        return False


def _resolve_config_extends(root: Path, config_path: Path, value: object) -> Path | None:
    if not isinstance(value, str) or not value.startswith("."):
        return None
    candidate = (config_path.parent / value).resolve()
    if candidate.suffix.lower() != ".json":
        candidate = Path(str(candidate) + ".json")
    if not _within_root(root, candidate) or not candidate.is_file():
        return None
    return candidate


def _rules_from_payload(config_path: Path, payload: dict[str, object]) -> tuple[_AliasRule, ...] | None:
    compiler_options = payload.get("compilerOptions")
    if not isinstance(compiler_options, dict) or "paths" not in compiler_options:
        return None
    paths = compiler_options.get("paths")
    if not isinstance(paths, dict):
        return tuple()
    base_url = compiler_options.get("baseUrl")
    if isinstance(base_url, str):
        base_directory = (config_path.parent / base_url).resolve()
    else:
        base_directory = config_path.parent.resolve()
    rules: list[_AliasRule] = []
    for pattern, raw_targets in paths.items():
        if not isinstance(pattern, str) or not isinstance(raw_targets, list):
            continue
        targets = tuple(item for item in raw_targets if isinstance(item, str))
        if targets:
            rules.append(_AliasRule(pattern=pattern, targets=targets, base_directory=base_directory))
    return tuple(rules)


def _load_alias_rules(root: Path, config_path: Path, stack: tuple[Path, ...] = ()) -> tuple[_AliasRule, ...]:
    config_path = config_path.resolve()
    if config_path in stack or not _within_root(root, config_path):
        return tuple()
    payload = _read_jsonc(config_path)
    if payload is None:
        return tuple()
    parent_rules: tuple[_AliasRule, ...] = tuple()
    parent = _resolve_config_extends(root, config_path, payload.get("extends"))
    if parent is not None:
        parent_rules = _load_alias_rules(root, parent, (*stack, config_path))
    local_rules = _rules_from_payload(config_path, payload)
    return parent_rules if local_rules is None else local_rules


def _nearest_config(root: Path, source_rel: str) -> Path | None:
    root = root.resolve()
    current = (root / source_rel).resolve().parent
    if not _within_root(root, current):
        return None
    while True:
        for name in _CONFIG_NAMES:
            candidate = current / name
            if candidate.is_file():
                return candidate
        if current == root:
            break
        current = current.parent
    return None


def _match_alias(pattern: str, specifier: str) -> tuple[str | None, tuple[int, int]] | None:
    wildcard_count = pattern.count("*")
    if wildcard_count == 0:
        return (None, (1, len(pattern))) if pattern == specifier else None
    if wildcard_count != 1:
        return None
    prefix, suffix = pattern.split("*", 1)
    if not specifier.startswith(prefix) or not specifier.endswith(suffix):
        return None
    if len(specifier) < len(prefix) + len(suffix):
        return None
    capture_end = len(specifier) - len(suffix) if suffix else len(specifier)
    capture = specifier[len(prefix) : capture_end]
    return capture, (0, len(prefix) + len(suffix))


def _resolve_candidate(root: Path, base_directory: Path, target_text: str) -> str | None:
    candidate = Path(target_text)
    target = candidate.resolve() if candidate.is_absolute() else (base_directory / candidate).resolve()
    if not _within_root(root, target):
        return None
    candidates: list[Path] = []
    if target.is_file():
        candidates.append(target)
    else:
        candidates.extend(Path(str(target) + extension) for extension in _JS_EXTENSIONS)
        candidates.extend(target / f"index{extension}" for extension in _JS_EXTENSIONS)
    for item in candidates:
        if item.is_file() and _within_root(root, item.resolve()):
            return item.resolve().relative_to(root).as_posix()
    return None


def resolve_typescript_path_alias(root: Path, source_rel: str, specifier: str) -> TypeScriptAliasResolution:
    root = root.resolve()
    config = _nearest_config(root, source_rel)
    if config is None:
        return TypeScriptAliasResolution(False, None)
    rules = _load_alias_rules(root, config)
    matches: list[tuple[tuple[int, int], _AliasRule, str | None]] = []
    for rule in rules:
        match = _match_alias(rule.pattern, specifier)
        if match is None:
            continue
        capture, score = match
        matches.append((score, rule, capture))
    if not matches:
        return TypeScriptAliasResolution(False, None)
    best_score = max(item[0] for item in matches)
    best = [item for item in matches if item[0] == best_score]
    resolved: set[str] = set()
    for _score, rule, capture in best:
        for target_pattern in rule.targets:
            target = target_pattern.replace("*", capture or "", 1) if "*" in target_pattern else target_pattern
            found = _resolve_candidate(root, rule.base_directory, target)
            if found is not None:
                resolved.add(found)
    if len(resolved) == 1:
        path = next(iter(resolved))
        return TypeScriptAliasResolution(True, path, f"Target resolves through {config.name} compilerOptions.paths")
    if len(resolved) > 1:
        return TypeScriptAliasResolution(True, None, "TypeScript path alias resolves to multiple repository targets")
    return TypeScriptAliasResolution(True, None, "TypeScript path alias matched but no repository target could be resolved")
