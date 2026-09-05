from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path
from typing import Any

import yaml


REGISTRY_SCHEMA_VERSION = "ptsip-p03-authority-role-provisional-dimensions/v1"
MATRIX_SCHEMA_VERSION = "ptsip-p03-authority-role-matrix/v1"
DEFAULT_REGISTRY = "planning/0.4.0/WU-02/p03-authority-role-provisional-dimensions.yaml"
DEFAULT_OUTPUT = "planning/0.4.0/WU-02/p03-authority-role-matrix.generated.yaml"
_DIMENSION_ID = re.compile(r"^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$")
_ADR_ID = re.compile(r"^ADR-([0-9]{4})$")
_MISSING = object()

_ALLOWED_OPERATORS = {
    "PRESENT",
    "NON_EMPTY",
    "EQUALS",
    "NOT_EQUALS",
    "CONTAINS",
    "CONTAINS_ALL",
}
_EXPRESSION_BRANCH_KEYS = {"all", "any", "not"}
_CONDITION_KEYS = {"path", "operator", "value"}


class AnalysisError(ValueError):
    pass


def _load_yaml(path: Path, *, label: str) -> dict[str, Any]:
    try:
        payload = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
    except FileNotFoundError as exc:
        raise AnalysisError(f"{label} not found: {path}") from exc
    except yaml.YAMLError as exc:
        raise AnalysisError(f"{label} is not valid YAML: {path}: {exc}") from exc
    if not isinstance(payload, dict):
        raise AnalysisError(f"{label} root must be a mapping: {path}")
    return payload


def _adr_number(value: object, *, label: str) -> int:
    if not isinstance(value, str):
        raise AnalysisError(f"{label} must be an ADR id")
    match = _ADR_ID.fullmatch(value)
    if match is None:
        raise AnalysisError(f"{label} must match ADR-NNNN: {value!r}")
    return int(match.group(1))


def _resolve_path(record: dict[str, Any], path: object) -> object:
    if not isinstance(path, str) or not path or path.startswith(".") or path.endswith("."):
        raise AnalysisError(f"predicate path must be a canonical dotted string: {path!r}")
    current: object = record
    for part in path.split("."):
        if not part:
            raise AnalysisError(f"predicate path contains an empty segment: {path!r}")
        if not isinstance(current, dict) or part not in current:
            return _MISSING
        current = current[part]
    return current


def _is_non_empty(value: object) -> bool:
    if value is _MISSING or value is None:
        return False
    if isinstance(value, str):
        return bool(value)
    if isinstance(value, (list, tuple, dict, set)):
        return bool(value)
    return True


def _evaluate_condition(condition: dict[str, Any], record: dict[str, Any]) -> bool:
    unknown = set(condition) - _CONDITION_KEYS
    if unknown:
        raise AnalysisError(f"predicate contains unsupported fields: {sorted(unknown)}")
    if "path" not in condition or "operator" not in condition:
        raise AnalysisError("predicate requires path and operator")

    operator = condition["operator"]
    if operator not in _ALLOWED_OPERATORS:
        raise AnalysisError(f"unsupported predicate operator: {operator!r}")

    actual = _resolve_path(record, condition["path"])

    if operator == "PRESENT":
        if "value" in condition:
            raise AnalysisError("PRESENT must not define value")
        return actual is not _MISSING
    if operator == "NON_EMPTY":
        if "value" in condition:
            raise AnalysisError("NON_EMPTY must not define value")
        return _is_non_empty(actual)

    if "value" not in condition:
        raise AnalysisError(f"{operator} requires value")
    expected = condition["value"]

    if operator == "EQUALS":
        return actual is not _MISSING and actual == expected
    if operator == "NOT_EQUALS":
        return actual is not _MISSING and actual != expected
    if operator == "CONTAINS":
        if actual is _MISSING or not isinstance(actual, (list, tuple, set, str)):
            return False
        return expected in actual
    if operator == "CONTAINS_ALL":
        if actual is _MISSING or not isinstance(actual, (list, tuple, set)):
            return False
        if not isinstance(expected, list):
            raise AnalysisError("CONTAINS_ALL value must be a list")
        return all(item in actual for item in expected)

    raise AssertionError(operator)


def _evaluate_expression(expression: object, record: dict[str, Any]) -> bool:
    if not isinstance(expression, dict) or not expression:
        raise AnalysisError("dimension expression must be a non-empty mapping")

    branch_keys = set(expression) & _EXPRESSION_BRANCH_KEYS
    if branch_keys:
        if len(branch_keys) != 1 or set(expression) != branch_keys:
            raise AnalysisError("expression branch must contain exactly one of all/any/not")

        branch = next(iter(branch_keys))
        value = expression[branch]
        if branch in {"all", "any"}:
            if not isinstance(value, list) or not value:
                raise AnalysisError(f"{branch} expression must be a non-empty list")
            results = [_evaluate_expression(item, record) for item in value]
            return all(results) if branch == "all" else any(results)
        return not _evaluate_expression(value, record)

    return _evaluate_condition(expression, record)


def _load_registry(path: Path) -> dict[str, Any]:
    registry = _load_yaml(path, label="P03 provisional dimension registry")
    if registry.get("schema_version") != REGISTRY_SCHEMA_VERSION:
        raise AnalysisError(
            f"registry schema_version must be {REGISTRY_SCHEMA_VERSION!r}"
        )

    analysis = registry.get("analysis")
    if not isinstance(analysis, dict):
        raise AnalysisError("registry analysis must be a mapping")
    required_analysis = {
        "index_path",
        "first_adr",
        "last_adr",
        "reviewed_through",
        "vocabulary_registration",
        "runtime_authority",
    }
    if set(analysis) != required_analysis:
        raise AnalysisError(
            f"registry analysis keys must be exactly {sorted(required_analysis)}"
        )
    if analysis["vocabulary_registration"] is not False:
        raise AnalysisError("vocabulary_registration must remain false during corpus analysis")
    if analysis["runtime_authority"] != "NONE":
        raise AnalysisError("provisional analysis registry must declare runtime_authority: NONE")

    first = _adr_number(analysis["first_adr"], label="analysis.first_adr")
    last = _adr_number(analysis["last_adr"], label="analysis.last_adr")
    reviewed = _adr_number(analysis["reviewed_through"], label="analysis.reviewed_through")
    if not first <= reviewed <= last:
        raise AnalysisError("reviewed_through must be within first_adr..last_adr")

    dimensions = registry.get("dimensions")
    if not isinstance(dimensions, dict) or not dimensions:
        raise AnalysisError("registry dimensions must be a non-empty mapping")
    for dimension_id, definition in dimensions.items():
        if not isinstance(dimension_id, str) or _DIMENSION_ID.fullmatch(dimension_id) is None:
            raise AnalysisError(f"invalid provisional dimension id: {dimension_id!r}")
        if not isinstance(definition, dict):
            raise AnalysisError(f"dimension {dimension_id} must be a mapping")
        if set(definition) != {"status", "introduced_by", "expression"}:
            raise AnalysisError(
                f"dimension {dimension_id} must contain status, introduced_by, expression only"
            )
        if definition["status"] != "PROVISIONAL":
            raise AnalysisError(f"dimension {dimension_id} status must be PROVISIONAL")
        _adr_number(definition["introduced_by"], label=f"{dimension_id}.introduced_by")
        _evaluate_expression(definition["expression"], {})

    return registry


def _index_routes(index: dict[str, Any]) -> dict[str, str]:
    topics = index.get("topics")
    if not isinstance(topics, dict):
        raise AnalysisError("decisions INDEX topics must be a mapping")

    routes: dict[str, str] = {}
    for topic_id, route in topics.items():
        if not isinstance(route, dict):
            raise AnalysisError(f"INDEX topic {topic_id!r} route must be a mapping")
        adr_id = route.get("current_adr")
        path = route.get("path")
        _adr_number(adr_id, label=f"INDEX {topic_id}.current_adr")
        if not isinstance(path, str) or not path.endswith(".yaml"):
            raise AnalysisError(f"INDEX {topic_id}.path must be a YAML path")
        if adr_id in routes:
            raise AnalysisError(f"INDEX contains duplicate current ADR id: {adr_id}")
        routes[adr_id] = path
    return routes


def _reviewed_ids(analysis: dict[str, Any]) -> list[str]:
    first = _adr_number(analysis["first_adr"], label="analysis.first_adr")
    reviewed = _adr_number(analysis["reviewed_through"], label="analysis.reviewed_through")
    return [f"ADR-{number:04d}" for number in range(first, reviewed + 1)]


def build_matrix(repo_root: Path, registry_path: Path) -> dict[str, Any]:
    registry = _load_registry(registry_path)
    analysis = registry["analysis"]
    index_path = repo_root / str(analysis["index_path"])
    index = _load_yaml(index_path, label="ADR INDEX")
    routes = _index_routes(index)

    dimensions: dict[str, dict[str, Any]] = registry["dimensions"]
    dimension_ids = list(dimensions)
    rows: dict[str, Any] = {}

    for adr_id in _reviewed_ids(analysis):
        route_path = routes.get(adr_id)
        if route_path is None:
            raise AnalysisError(f"INDEX has no current route for reviewed ADR {adr_id}")
        record_path = repo_root / route_path
        record = _load_yaml(record_path, label=adr_id)
        decision = record.get("decision")
        if not isinstance(decision, dict) or decision.get("id") != adr_id:
            raise AnalysisError(f"{route_path} does not contain decision.id {adr_id}")

        effects = {
            dimension_id: bool(
                _evaluate_expression(definition["expression"], record)
            )
            for dimension_id, definition in dimensions.items()
        }
        rows[adr_id] = {
            "source_path": route_path,
            "role_effect_analysis": effects,
        }

    payload = {
        "schema_version": MATRIX_SCHEMA_VERSION,
        "source": {
            "dimension_registry": registry_path.relative_to(repo_root).as_posix()
            if registry_path.is_relative_to(repo_root)
            else str(registry_path),
            "index_path": str(analysis["index_path"]),
            "reviewed_through": analysis["reviewed_through"],
            "vocabulary_registration": False,
            "runtime_authority": "NONE",
        },
        "dimensions": dimension_ids,
        "rows": rows,
        "validation": {
            "rectangular": True,
            "reviewed_row_count": len(rows),
            "dimension_count": len(dimension_ids),
        },
    }
    validate_matrix(payload)
    return payload


def validate_matrix(payload: dict[str, Any]) -> None:
    if payload.get("schema_version") != MATRIX_SCHEMA_VERSION:
        raise AnalysisError(f"matrix schema_version must be {MATRIX_SCHEMA_VERSION!r}")

    dimensions = payload.get("dimensions")
    rows = payload.get("rows")
    if not isinstance(dimensions, list) or not dimensions:
        raise AnalysisError("matrix dimensions must be a non-empty list")
    if len(set(dimensions)) != len(dimensions):
        raise AnalysisError("matrix dimensions must be unique")
    if not all(isinstance(item, str) and _DIMENSION_ID.fullmatch(item) for item in dimensions):
        raise AnalysisError("matrix dimensions contain an invalid id")
    if not isinstance(rows, dict) or not rows:
        raise AnalysisError("matrix rows must be a non-empty mapping")

    expected = set(dimensions)
    for adr_id, row in rows.items():
        _adr_number(adr_id, label="matrix row id")
        if not isinstance(row, dict):
            raise AnalysisError(f"matrix row {adr_id} must be a mapping")
        effects = row.get("role_effect_analysis")
        if not isinstance(effects, dict):
            raise AnalysisError(f"matrix row {adr_id} role_effect_analysis must be a mapping")
        actual = set(effects)
        if actual != expected:
            missing = sorted(expected - actual)
            extra = sorted(actual - expected)
            raise AnalysisError(
                f"matrix row {adr_id} is not rectangular; missing={missing} extra={extra}"
            )
        for dimension_id, value in effects.items():
            if type(value) is not bool:
                raise AnalysisError(
                    f"matrix row {adr_id}.{dimension_id} must be exact boolean"
                )

    validation = payload.get("validation")
    if not isinstance(validation, dict):
        raise AnalysisError("matrix validation must be a mapping")
    if validation.get("rectangular") is not True:
        raise AnalysisError("matrix validation.rectangular must be true")
    if validation.get("reviewed_row_count") != len(rows):
        raise AnalysisError("matrix reviewed_row_count is stale")
    if validation.get("dimension_count") != len(dimensions):
        raise AnalysisError("matrix dimension_count is stale")


def _dump_yaml(payload: dict[str, Any]) -> str:
    return yaml.safe_dump(
        payload,
        sort_keys=False,
        allow_unicode=True,
        default_flow_style=False,
    )


def _write_matrix(output_path: Path, payload: dict[str, Any]) -> None:
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(_dump_yaml(payload), encoding="utf-8")


def _check_matrix(output_path: Path, expected: dict[str, Any]) -> None:
    current = _load_yaml(output_path, label="generated P03 authority role matrix")
    validate_matrix(current)
    if current != expected:
        raise AnalysisError(
            "generated P03 authority role matrix is stale; run with --write to regenerate all reviewed ADR rows"
        )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Deterministically evaluate provisional P03 Authority Role dimensions"
    )
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--registry", default=DEFAULT_REGISTRY)
    parser.add_argument("--output", default=DEFAULT_OUTPUT)
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--write", action="store_true")
    mode.add_argument("--check", action="store_true")
    args = parser.parse_args(argv)

    repo_root = Path(args.repo_root).resolve()
    registry_path = Path(args.registry)
    if not registry_path.is_absolute():
        registry_path = repo_root / registry_path
    output_path = Path(args.output)
    if not output_path.is_absolute():
        output_path = repo_root / output_path

    try:
        payload = build_matrix(repo_root, registry_path)
        if args.write:
            _write_matrix(output_path, payload)
            action = "wrote"
        else:
            _check_matrix(output_path, payload)
            action = "validated"
    except (AnalysisError, OSError) as exc:
        print(f"P03 authority-role matrix error: {exc}", file=sys.stderr)
        return 1

    print(
        f"P03 authority-role matrix {action}: "
        f"{payload['validation']['reviewed_row_count']} reviewed ADR(s), "
        f"{payload['validation']['dimension_count']} provisional dimension(s)"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
