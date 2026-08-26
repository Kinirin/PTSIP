from __future__ import annotations

import copy
import tempfile
from dataclasses import dataclass
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

from ...constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ...validation.profile import _schema, validate_profile
from ...validation.templates import materialize_profile
from .model import DecisionAnswer

DEFAULT_POLICIES: dict[str, str] = {
    "product_to_nonproduct_runtime_dependency": "deny",
    "nonproduct_in_product_package": "deny",
    "independent_build_resolution": "required",
}

_DECISION_FIELDS = (
    "classification",
    "purpose",
    "shipped",
    "runtime_required",
    "executable",
)


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


def _decision_value(answer: DecisionAnswer, key: str) -> object:
    return getattr(answer, key)


def _apply_answer(
    component: dict[str, object],
    answer: DecisionAnswer,
    *,
    require_existing_equal: bool,
) -> None:
    for key in _DECISION_FIELDS:
        expected = _decision_value(answer, key)
        if require_existing_equal:
            _set_missing_or_require_equal(component, key, expected)
        else:
            component[key] = expected
    component.pop("lifecycle_owner", None)


def _decision_matches(component: dict[str, object], answer: DecisionAnswer) -> bool:
    return all(component.get(key) == _decision_value(answer, key) for key in _DECISION_FIELDS)


def _find_component(items: object, component_id: str, *, label: str) -> dict[str, object] | None:
    if not isinstance(items, list):
        raise ValueError(f"{label} must be a list")
    for item in items:
        if isinstance(item, dict) and str(item.get("id")) == component_id:
            return item
    return None


def _new_component(
    component_id: str,
    include: tuple[str, ...] | list[str],
    answer: DecisionAnswer,
) -> dict[str, object]:
    component: dict[str, object] = {
        "id": component_id,
        "include": [str(item) for item in include],
    }
    _apply_answer(component, answer, require_existing_equal=False)
    return component


def _project_explicit(
    payload: dict[str, object],
    component_id: str,
    include: tuple[str, ...] | list[str],
    answer: DecisionAnswer,
) -> dict[str, object]:
    components = payload.setdefault("components", [])
    component = _find_component(components, component_id, label="components")
    if component is None:
        assert isinstance(components, list)
        components.append(_new_component(component_id, include, answer))
        return payload

    if "include" not in component:
        component["include"] = [str(item) for item in include]
    _apply_answer(component, answer, require_existing_equal=True)
    return payload


def _project_template_backed(
    payload: dict[str, object],
    map_meta: dict[str, object],
    component_id: str,
    include: tuple[str, ...] | list[str],
    answer: DecisionAnswer,
) -> dict[str, object]:
    # Materialization is read-only and is used here only to understand the
    # currently selected immutable template/hybrid declaration.  It does not
    # authorize a write or select architecture from repository evidence.
    resolved = materialize_profile(payload)
    effective_components = resolved.effective_payload.get("components", [])
    effective_component = _find_component(
        effective_components,
        component_id,
        label="effective components",
    )

    mode = map_meta.get("mode")
    if mode == "hybrid":
        overrides = map_meta.get("overrides")
        if not isinstance(overrides, dict):
            raise ValueError("Hybrid Responsibility Map requires an overrides mapping")
        removal_ids = overrides.get("remove_component_ids", [])
        if not isinstance(removal_ids, list):
            raise ValueError("remove_component_ids must be a list")
        if component_id in removal_ids:
            raise ValueError(
                f"Existing hybrid project declaration removes component {component_id!r}; "
                "the accepted decision cannot silently reverse that project-owned removal"
            )
        project_components = overrides.get("components", [])
        project_component = _find_component(
            project_components,
            component_id,
            label="responsibility_map.overrides.components",
        )
        if project_component is not None:
            if "include" not in project_component:
                project_component["include"] = [str(item) for item in include]
            _apply_answer(project_component, answer, require_existing_equal=True)
            return payload
    elif mode == "template":
        overrides = None
        project_components = None
    else:  # pragma: no cover - caller guards the supported source modes
        raise ValueError(f"Unsupported Responsibility Map mode: {mode!r}")

    # If the immutable template already represents the accepted canonical
    # decision, source mode must remain unchanged.  This prevents automatic
    # template -> hybrid conversion and also avoids adding empty override
    # collections to an existing hybrid source when no project delta is needed.
    if effective_component is not None and _decision_matches(effective_component, answer):
        return payload

    if mode == "template":
        template_ref = map_meta.get("template")
        if not isinstance(template_ref, dict):
            raise ValueError("Template Responsibility Map requires an immutable template reference")
        map_meta["mode"] = "hybrid"
        overrides = {}
        map_meta["overrides"] = overrides
        project_components = []
        overrides["components"] = project_components
    else:
        assert isinstance(overrides, dict)
        if "components" not in overrides:
            project_components = []
            overrides["components"] = project_components

    assert isinstance(overrides, dict)
    assert isinstance(project_components, list)

    if effective_component is None:
        project_components.append(_new_component(component_id, include, answer))
        return payload

    # A differing accepted decision may replace one template-owned stable-ID
    # entity.  Preserve all template fields not covered by the decision (for
    # example roles/selectors) and change only the five canonical decision
    # fields.  This is a lossless whole-entity representation of the accepted
    # delta, not materialize-to-explicit writeback.
    replacement = copy.deepcopy(effective_component)
    _apply_answer(replacement, answer, require_existing_equal=False)
    project_components.append(replacement)
    return payload


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
    if not isinstance(map_meta, dict):
        raise ValueError("responsibility_map must be a mapping")

    mode = map_meta.get("mode")
    if mode == "explicit":
        return _project_explicit(payload, component_id, include, answer)
    if mode in {"template", "hybrid"}:
        return _project_template_backed(payload, map_meta, component_id, include, answer)
    raise ValueError(f"Unsupported Responsibility Map mode: {mode!r}")


def validate_projected_payload(payload: dict[str, object]) -> tuple[str, ...]:
    return tuple(
        f"{'.'.join(str(part) for part in item.absolute_path) or '<root>'}: {item.message}"
        for item in sorted(
            Draft202012Validator(_schema(payload)).iter_errors(payload),
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
