from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import yaml


def repository_root(start: str | Path | None = None) -> Path:
    current = Path(start or __file__).resolve()
    if current.is_file():
        current = current.parent
    for candidate in (current, *current.parents):
        if (candidate / "pyproject.toml").is_file():
            return candidate
    raise RuntimeError("Unable to locate PTSIP repository root.")


def load_yaml(path: str | Path, *, root: str | Path | None = None) -> dict[str, Any]:
    base = repository_root(root)
    candidate = Path(path)
    if not candidate.is_absolute():
        candidate = base / candidate
    value = yaml.safe_load(candidate.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{candidate}: expected a YAML mapping")
    return value


def load_json(path: str | Path, *, root: str | Path | None = None) -> dict[str, Any]:
    base = repository_root(root)
    candidate = Path(path)
    if not candidate.is_absolute():
        candidate = base / candidate
    value = json.loads(candidate.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{candidate}: expected a JSON object")
    return value
