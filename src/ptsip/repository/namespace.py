from __future__ import annotations

import json
from importlib.resources import files
from pathlib import Path, PurePosixPath
from typing import Mapping

from jsonschema import Draft202012Validator


PTSIP_REPOSITORY_ROOT = PurePosixPath(".ptsip")
REPOSITORY_INDEX = PTSIP_REPOSITORY_ROOT / "index.json"
REPOSITORY_INDEX_FORMAT = "ptsip-repository-index/v1"
_REPOSITORY_INDEX_SCHEMA = "repository-index.schema.json"


class RepositoryNamespaceError(ValueError):
    """Fail-closed error for the repository-local PTSIP namespace."""


def _schema() -> dict[str, object]:
    resource = files("ptsip").joinpath(
        "repository",
        "schemas",
        _REPOSITORY_INDEX_SCHEMA,
    )
    payload = json.loads(resource.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise RepositoryNamespaceError("Repository index schema must be a JSON object.")
    return payload


def validate_repository_index_payload(payload: object) -> dict[str, object]:
    if not isinstance(payload, Mapping):
        raise RepositoryNamespaceError("PTSIP repository index must be a JSON object.")

    schema = _schema()
    Draft202012Validator.check_schema(schema)
    errors = sorted(
        Draft202012Validator(schema).iter_errors(payload),
        key=lambda item: list(item.absolute_path),
    )
    if errors:
        first = errors[0]
        location = ".".join(str(part) for part in first.absolute_path) or "<root>"
        raise RepositoryNamespaceError(
            f"Invalid {REPOSITORY_INDEX.as_posix()} at {location}: {first.message}"
        )
    return dict(payload)


def load_repository_index(
    repository_root: str | Path,
) -> dict[str, object] | None:
    root = Path(repository_root).resolve()
    index_path = root / REPOSITORY_INDEX.as_posix()
    if not index_path.is_file():
        return None

    try:
        payload = json.loads(index_path.read_text(encoding="utf-8-sig"))
    except Exception as exc:
        raise RepositoryNamespaceError(
            f"Unable to parse {REPOSITORY_INDEX.as_posix()}: {exc}"
        ) from exc
    return validate_repository_index_payload(payload)


def default_repository_index_payload() -> dict[str, object]:
    return {
        "format": REPOSITORY_INDEX_FORMAT,
        "namespaces": {
            "profiles": {"status": "ACTIVE", "index": "profiles/index.json"},
            "context": {"status": "ACTIVE", "root": "context/"},
            "tasks": {"status": "RESERVED", "index": "tasks/index.json"},
            "runtime": {"status": "RESERVED", "root": "runtime/"},
        },
    }


def default_repository_index_text() -> str:
    return json.dumps(
        default_repository_index_payload(),
        indent=2,
        ensure_ascii=False,
    ) + "\n"
