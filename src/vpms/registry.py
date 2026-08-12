from __future__ import annotations

from dataclasses import asdict, dataclass
from enum import StrEnum
from typing import Mapping

from .model import (
    FormulaRef,
    PolicyRef,
    RunnerRef,
    TargetRef,
    VariablesRef,
    VerificationCase,
    VerificationPurpose,
)


class RegistryDiagnosticCode(StrEnum):
    MALFORMED_DEFINITIONS = "MALFORMED_DEFINITIONS"
    MALFORMED_CASE = "MALFORMED_CASE"
    MISSING_FIELD = "MISSING_FIELD"
    UNKNOWN_FIELD = "UNKNOWN_FIELD"
    INVALID_FIELD = "INVALID_FIELD"
    UNKNOWN_PURPOSE = "UNKNOWN_PURPOSE"
    DUPLICATE_CASE_ID = "DUPLICATE_CASE_ID"
    UNRESOLVED_REFERENCE = "UNRESOLVED_REFERENCE"


@dataclass(frozen=True)
class RegistryDiagnostic:
    code: RegistryDiagnosticCode
    location: str
    message: str
    case_id: str | None = None
    reference_kind: str | None = None
    reference: str | None = None

    def as_dict(self) -> dict[str, object]:
        payload: dict[str, object] = {
            "format": "vpms-registry-diagnostic/v1",
            "code": self.code.value,
            "location": self.location,
            "message": self.message,
        }
        if self.case_id is not None:
            payload["case_id"] = self.case_id
        if self.reference_kind is not None:
            payload["reference_kind"] = self.reference_kind
        if self.reference is not None:
            payload["reference"] = self.reference
        return payload


@dataclass(frozen=True)
class RegistryReferenceIndex:
    targets: tuple[str, ...] = ()
    formulas: tuple[str, ...] = ()
    variables: tuple[str, ...] = ()
    policies: tuple[str, ...] = ()
    runners: tuple[str, ...] = ()

    def __post_init__(self) -> None:
        for field_name in ("targets", "formulas", "variables", "policies", "runners"):
            values = getattr(self, field_name)
            object.__setattr__(self, field_name, tuple(sorted(set(values))))


@dataclass(frozen=True)
class Registry:
    references: RegistryReferenceIndex
    cases: tuple[VerificationCase, ...]

    def get_case(self, case_id: str) -> VerificationCase | None:
        for case in self.cases:
            if case.id == case_id:
                return case
        return None

    def as_dict(self) -> dict[str, object]:
        return {
            "references": asdict(self.references),
            "cases": [case.as_dict() for case in self.cases],
        }


@dataclass(frozen=True)
class RegistryLoadResult:
    registry: Registry | None
    diagnostics: tuple[RegistryDiagnostic, ...]

    @property
    def ok(self) -> bool:
        return self.registry is not None and not self.diagnostics

    def as_dict(self) -> dict[str, object]:
        return {
            "ok": self.ok,
            "registry": self.registry.as_dict() if self.registry is not None else None,
            "diagnostics": [diagnostic.as_dict() for diagnostic in self.diagnostics],
        }


_REQUIRED_FIELDS = ("id", "purpose", "target", "formula", "variables", "policy", "runner")
_REFERENCE_FIELDS = ("target", "formula", "variables", "policy", "runner")


def _diagnostic_sort_key(diagnostic: RegistryDiagnostic) -> tuple[str, str, str, str, str]:
    return (
        diagnostic.location,
        diagnostic.code.value,
        diagnostic.case_id or "",
        diagnostic.reference_kind or "",
        diagnostic.reference or "",
    )


def _is_identifier(value: object) -> bool:
    return isinstance(value, str) and bool(value) and value.strip() == value


def _available(references: RegistryReferenceIndex, kind: str) -> frozenset[str]:
    if kind == "target":
        return frozenset(references.targets)
    if kind == "formula":
        return frozenset(references.formulas)
    if kind == "variables":
        return frozenset(references.variables)
    if kind == "policy":
        return frozenset(references.policies)
    return frozenset(references.runners)


def load_registry(
    definitions: object,
    *,
    references: RegistryReferenceIndex | None = None,
) -> RegistryLoadResult:
    reference_index = references or RegistryReferenceIndex()
    diagnostics: list[RegistryDiagnostic] = []
    cases: list[VerificationCase] = []
    seen_case_ids: set[str] = set()

    if not isinstance(definitions, list):
        diagnostic = RegistryDiagnostic(
            code=RegistryDiagnosticCode.MALFORMED_DEFINITIONS,
            location="$",
            message="Verification definitions must be a list of case mappings.",
        )
        return RegistryLoadResult(registry=None, diagnostics=(diagnostic,))

    for index, raw_case in enumerate(definitions):
        location = f"$[{index}]"
        if not isinstance(raw_case, Mapping):
            diagnostics.append(
                RegistryDiagnostic(
                    code=RegistryDiagnosticCode.MALFORMED_CASE,
                    location=location,
                    message="Verification Case definition must be a mapping.",
                )
            )
            continue

        raw_keys = {str(key) for key in raw_case.keys()}
        for field in sorted(raw_keys.difference(_REQUIRED_FIELDS)):
            diagnostics.append(
                RegistryDiagnostic(
                    code=RegistryDiagnosticCode.UNKNOWN_FIELD,
                    location=f"{location}.{field}",
                    message=f"Unknown Verification Case field: {field}.",
                )
            )

        missing = [field for field in _REQUIRED_FIELDS if field not in raw_case]
        for field in missing:
            diagnostics.append(
                RegistryDiagnostic(
                    code=RegistryDiagnosticCode.MISSING_FIELD,
                    location=f"{location}.{field}",
                    message=f"Missing required Verification Case field: {field}.",
                )
            )

        valid_values: dict[str, str] = {}
        for field in _REQUIRED_FIELDS:
            if field not in raw_case:
                continue
            value = raw_case[field]
            if not _is_identifier(value):
                diagnostics.append(
                    RegistryDiagnostic(
                        code=RegistryDiagnosticCode.INVALID_FIELD,
                        location=f"{location}.{field}",
                        message=f"Verification Case field {field} must be a non-empty, unpadded string.",
                    )
                )
                continue
            valid_values[field] = value

        case_id = valid_values.get("id")
        if case_id is not None:
            if case_id in seen_case_ids:
                diagnostics.append(
                    RegistryDiagnostic(
                        code=RegistryDiagnosticCode.DUPLICATE_CASE_ID,
                        location=f"{location}.id",
                        message=f"Duplicate Verification Case id: {case_id}.",
                        case_id=case_id,
                    )
                )
            else:
                seen_case_ids.add(case_id)

        purpose: VerificationPurpose | None = None
        purpose_value = valid_values.get("purpose")
        if purpose_value is not None:
            try:
                purpose = VerificationPurpose(purpose_value)
            except ValueError:
                diagnostics.append(
                    RegistryDiagnostic(
                        code=RegistryDiagnosticCode.UNKNOWN_PURPOSE,
                        location=f"{location}.purpose",
                        message=f"Unknown verification purpose: {purpose_value}.",
                        case_id=case_id,
                    )
                )

        unresolved = False
        for kind in _REFERENCE_FIELDS:
            reference = valid_values.get(kind)
            if reference is None:
                continue
            if reference not in _available(reference_index, kind):
                unresolved = True
                diagnostics.append(
                    RegistryDiagnostic(
                        code=RegistryDiagnosticCode.UNRESOLVED_REFERENCE,
                        location=f"{location}.{kind}",
                        message=f"Unresolved {kind} reference: {reference}.",
                        case_id=case_id,
                        reference_kind=kind,
                        reference=reference,
                    )
                )

        current_has_error = any(
            diagnostic.location.startswith(location)
            for diagnostic in diagnostics
        )
        if (
            not missing
            and all(field in valid_values for field in _REQUIRED_FIELDS)
            and case_id is not None
            and purpose is not None
            and not unresolved
            and not current_has_error
        ):
            cases.append(
                VerificationCase(
                    id=case_id,
                    purpose=purpose,
                    target=TargetRef(component_id=valid_values["target"]),
                    formula=FormulaRef(ref=valid_values["formula"]),
                    variables=VariablesRef(ref=valid_values["variables"]),
                    policy=PolicyRef(ref=valid_values["policy"]),
                    runner=RunnerRef(ref=valid_values["runner"]),
                )
            )

    diagnostics.sort(key=_diagnostic_sort_key)
    if diagnostics:
        return RegistryLoadResult(registry=None, diagnostics=tuple(diagnostics))

    cases.sort(key=lambda case: case.id)
    return RegistryLoadResult(
        registry=Registry(references=reference_index, cases=tuple(cases)),
        diagnostics=(),
    )
