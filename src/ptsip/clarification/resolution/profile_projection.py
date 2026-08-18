from __future__ import annotations

import copy
import tempfile
from dataclasses import dataclass
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

from ...constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ...validation.profile import _schema, validate_profile
from .model import DecisionAnswer

DEFAULT_POLICIES: dict[str, str] = {
    "product_to_nonproduct_runtime_dependency": "deny",
    "nonproduct_in_product_package": "deny",
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
        "responsibility_map": {"mode": "explicit"},
        "components": [],
        "policies": dict(DEFAULT_POLICIES),
    }


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

    if "boundaries" in payload:
        raise ValueError(
            "Canonical Tool 0.3.6 adoption requires Responsibility Map v2 component declarations. "
            "Legacy boundary-root profiles must be handled by the Tool 0.3.5 migration path."
        )

    map_meta = payload.setdefault("responsibility_map", {"mode": "explicit"})
    if not isinstance(map_meta, dict) or map_meta.get("mode") != "explicit":
        raise ValueError(
            "Direct local adoption currently projects only explicit Responsibility Maps; "
            "template/hybrid projection requires the template materialization path."
        )

    components = payload.setdefault("components", [])
    if not isinstance(components, list):
        raise ValueError("components must be a list")

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

    # Canonical 0.3.6 persists classification as the sole lifecycle-ownership
    # authority. DecisionAnswer.lifecycle_owner is validated as a compatibility
    # input but is deliberately not serialized into the new Project Profile.
    _set_missing_or_require_equal(component, "classification", answer.classification)
    _set_missing_or_require_equal(component, "purpose", answer.purpose)
    _set_missing_or_require_equal(component, "shipped", answer.shipped)
    _set_missing_or_require_equal(component, "runtime_required", answer.runtime_required)
    _set_missing_or_require_equal(component, "executable", answer.executable)
    component.pop("lifecycle_owner", None)
    return payload


def validate_projected_payload(payload: dict[str, object]) -> tuple[str, ...]:
    return tuple(
        f"{'.'.join(str(part) for part in item.absolute_path) or '<root>'}: {item.message}"
        for item in sorted(
            Draft202012Validator(_schema()).iter_errors(payload),
            key=lambda value: list(value.absolute_path),
        )
    )


def dump_payload(payload: dict[str, object]) -> str:
    return yaml.safe_dump(payload, sort_keys=False, allow_unicode=True)


def load_profile_text(text: str | None) -> dict[str, object] | None:
    if text is None:
        return None
    payload = yaml.safe_load(text)
    if not isinstance(payload, dict):
        raise ValueError("PTSIP profile root must be a mapping")
    return payload


def _local_profile_path(repository_root: Path, explicit: str | Path | None) -> Path:
    if explicit is None:
        return repository_root / "ptsip.yaml"
    profile = Path(explicit).expanduser().resolve()
    if not profile.parent.is_dir():
        raise FileNotFoundError(f"PTSIP profile parent directory does not exist: {profile.parent}")
    return profile


def prepare_local_profile(
    repository_root: str | Path,
    component_id: str,
    include: tuple[str, ...] | list[str],
    answer: DecisionAnswer,
    profile_path: str | Path | None = None,
) -> PreparedLocalProfile:
    root = Path(repository_root).resolve()
    profile = _local_profile_path(root, profile_path)
    expected_source = profile.read_text(encoding="utf-8-sig") if profile.is_file() else None
    existing = load_profile_text(expected_source)
    projected = project_payload(existing, component_id, include, answer)
    content = dump_payload(projected)

    with tempfile.NamedTemporaryFile(
        "w",
        suffix=".yaml",
        dir=profile.parent,
        delete=False,
        encoding="utf-8",
        newline="\n",
    ) as handle:
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
        raise RuntimeError(
            f"{prepared.path} changed after decision projection validation; refusing to overwrite concurrent changes"
        )
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
    profile_path: str | Path | None = None,
) -> Path:
    return write_prepared_local_profile(
        prepare_local_profile(repository_root, component_id, include, answer, profile_path)
    )
