from __future__ import annotations

import copy
import tempfile
from dataclasses import dataclass
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

from ...constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ...validation.profile import _root_overlap_errors, _schema, validate_profile
from .model import DecisionAnswer

DEFAULT_POLICIES: dict[str, str] = {
    "product_to_toolchain_runtime_dependency": "deny",
    "toolchain_in_product_package": "deny",
    "independent_build_resolution": "required",
}


@dataclass(frozen=True)
class PreparedLocalProfile:
    path: Path
    content: str
    expected_source: str | None


def _base_profile() -> dict[str, object]:
    return {
        "ptsip": {
            "version": SPEC_VERSION,
            "specification": {
                "source": SPEC_SOURCE,
                "revision": SPEC_REVISION,
            },
        },
        "components": [],
        "policies": dict(DEFAULT_POLICIES),
    }


def _boundary_root(selector: str) -> str:
    normalized = selector.replace("\\", "/").strip().strip("/")
    if normalized.endswith("/**"):
        normalized = normalized[:-3].rstrip("/")
    if not normalized or any(char in normalized for char in "*?[]"):
        raise ValueError(f"Boundary profile cannot safely represent selector {selector!r}")
    return normalized


def _set_missing_or_require_equal(component: dict[str, object], key: str, expected: object) -> None:
    if key not in component or component[key] in (None, ""):
        component[key] = expected
        return
    if component[key] != expected:
        raise ValueError(
            f"Existing component {component.get('id')!r} declares {key}={component[key]!r}, "
            f"which conflicts with the resolved decision value {expected!r}"
        )


def project_payload(
    existing: dict[str, object] | None,
    component_id: str,
    include: tuple[str, ...] | list[str],
    answer: DecisionAnswer,
) -> dict[str, object]:
    payload = copy.deepcopy(existing) if existing is not None else _base_profile()
    if not isinstance(payload, dict):
        raise ValueError("PTSIP profile root must be a mapping")

    if "boundaries" in payload and "components" not in payload:
        boundaries = payload.get("boundaries")
        if not isinstance(boundaries, dict):
            raise ValueError("boundaries must be a mapping")
        key = {
            "PRODUCT": "product",
            "TOOLCHAIN": "toolchain",
            "NEUTRAL_CONTRACT": "neutral_contract",
        }[answer.classification]
        target = boundaries.setdefault(key, {"roots": []})
        if not isinstance(target, dict):
            raise ValueError(f"boundaries.{key} must be a mapping")
        roots = target.setdefault("roots", [])
        if not isinstance(roots, list):
            raise ValueError(f"boundaries.{key}.roots must be a list")
        for selector in include:
            root = _boundary_root(str(selector))
            if root not in roots:
                roots.append(root)
        return payload

    components = payload.setdefault("components", [])
    if not isinstance(components, list):
        raise ValueError("components must be a list")
    payload.pop("boundaries", None)

    component: dict[str, object] | None = None
    for item in components:
        if isinstance(item, dict) and str(item.get("id")) == component_id:
            component = item
            break
    if component is None:
        component = {"id": component_id, "include": [str(item) for item in include]}
        components.append(component)
    elif "include" not in component:
        component["include"] = [str(item) for item in include]

    # Clarification fills missing declaration facts. It must not silently
    # reclassify or rewrite facts the profile already declares. Existing
    # include selectors are preserved so resolving one detected candidate does
    # not accidentally narrow a broader project-owned component boundary.
    _set_missing_or_require_equal(component, "classification", answer.classification)
    _set_missing_or_require_equal(component, "purpose", answer.purpose)
    _set_missing_or_require_equal(component, "shipped", answer.shipped)
    _set_missing_or_require_equal(component, "executable", answer.executable)
    _set_missing_or_require_equal(component, "release_owner", answer.lifecycle_owner)
    return payload


def validate_projected_payload(payload: dict[str, object]) -> tuple[str, ...]:
    errors = [
        f"{'.'.join(str(part) for part in item.absolute_path) or '<root>'}: {item.message}"
        for item in sorted(Draft202012Validator(_schema()).iter_errors(payload), key=lambda value: list(value.absolute_path))
    ]
    errors.extend(_root_overlap_errors(payload))
    return tuple(errors)


def dump_payload(payload: dict[str, object]) -> str:
    return yaml.safe_dump(payload, sort_keys=False, allow_unicode=True)


def load_profile_text(text: str | None) -> dict[str, object] | None:
    if text is None:
        return None
    payload = yaml.safe_load(text)
    if not isinstance(payload, dict):
        raise ValueError("PTSIP profile root must be a mapping")
    return payload


def prepare_local_profile(
    repository_root: str | Path,
    component_id: str,
    include: tuple[str, ...] | list[str],
    answer: DecisionAnswer,
) -> PreparedLocalProfile:
    root = Path(repository_root).resolve()
    profile = root / "ptsip.yaml"
    expected_source = profile.read_text(encoding="utf-8-sig") if profile.is_file() else None
    existing = load_profile_text(expected_source)
    projected = project_payload(existing, component_id, include, answer)
    content = dump_payload(projected)

    with tempfile.NamedTemporaryFile("w", suffix=".yaml", dir=root, delete=False, encoding="utf-8", newline="\n") as handle:
        temp = Path(handle.name)
        handle.write(content)
    try:
        result = validate_profile(root, temp)
        if not result.valid:
            raise ValueError("Projected PTSIP profile is invalid: " + "; ".join(result.errors))
    finally:
        if temp.exists():
            temp.unlink()
    return PreparedLocalProfile(profile, content, expected_source)


def write_prepared_local_profile(prepared: PreparedLocalProfile) -> Path:
    current_source = prepared.path.read_text(encoding="utf-8-sig") if prepared.path.is_file() else None
    if current_source != prepared.expected_source:
        raise RuntimeError("ptsip.yaml changed after decision projection validation; refusing to overwrite concurrent changes")
    with tempfile.NamedTemporaryFile(
        "w",
        suffix=".yaml",
        dir=prepared.path.parent,
        delete=False,
        encoding="utf-8",
        newline="\n",
    ) as handle:
        temp = Path(handle.name)
        handle.write(prepared.content)
    try:
        temp.replace(prepared.path)
    finally:
        if temp.exists():
            temp.unlink()
    return prepared.path


def apply_local_profile(
    repository_root: str | Path,
    component_id: str,
    include: tuple[str, ...] | list[str],
    answer: DecisionAnswer,
) -> Path:
    return write_prepared_local_profile(prepare_local_profile(repository_root, component_id, include, answer))
