"""Side-effect-free explicit selection bound to one validated registry snapshot."""
from __future__ import annotations

from collections import Counter
from dataclasses import dataclass
from typing import Mapping
from weakref import WeakKeyDictionary

from .. import contract_runtime as contracts
from ..domain.snapshot import _snapshot_data


@dataclass(frozen=True)
class SelectionDiagnostic:
    code: str
    location: str
    reference: str | None = None

    def as_dict(self):
        result = {"code": self.code, "location": self.location}
        if self.reference is not None:
            result["reference"] = self.reference
        return result


@dataclass(frozen=True)
class _SelectionData:
    state: str
    snapshot: object
    cases: tuple
    diagnostics: tuple[SelectionDiagnostic, ...]


_SELECTIONS: WeakKeyDictionary = WeakKeyDictionary()


class SelectionResult:
    __slots__ = ("__weakref__",)

    def __init__(self):
        raise TypeError("Use resolve_selection; direct construction is not a validated handoff.")

    @property
    def state(self):
        return _selection_data(self).state

    @property
    def case_ids(self):
        return tuple(case.id for case in _selection_data(self).cases)

    @property
    def diagnostics(self):
        return _selection_data(self).diagnostics

    @property
    def ok(self):
        return self.state == "RESOLVED"

    def as_dict(self):
        return {"state": self.state, "case_ids": list(self.case_ids),
                "diagnostics": [item.as_dict() for item in self.diagnostics]}


def _selection_data(result) -> _SelectionData:
    if type(result) is not SelectionResult or result not in _SELECTIONS:
        raise ValueError("UNVALIDATED_SELECTION")
    return _SELECTIONS[result]


def _selected_cases(snapshot, result):
    _snapshot_data(snapshot)
    data = _selection_data(result)
    if data.state != "RESOLVED" or data.diagnostics or not data.cases:
        raise ValueError("UNRESOLVED_SELECTION")
    if data.snapshot is not snapshot:
        raise ValueError("REGISTRY_SNAPSHOT_MISMATCH")
    return data.cases


def _result(snapshot, cases=(), diagnostics=()):
    ordered = tuple(sorted(diagnostics, key=lambda d: (d.location, d.code, d.reference or "")))
    result = object.__new__(SelectionResult)
    _SELECTIONS[result] = _SelectionData(
        "REJECTED" if ordered else "RESOLVED", snapshot,
        () if ordered else tuple(sorted(cases, key=lambda case: case.id)), ordered,
    )
    return result


def resolve_selection(snapshot, request: object, *, rules: Mapping | None = None) -> SelectionResult:
    contract = contracts.require_active_contract("selection")
    contracts.require_active_contract("protocol")
    semantics = contract["semantics"]
    expected = {"ordering": "CASE_ID_ASCENDING", "invalid_request": "REJECT_REQUEST",
                "failure_atomicity": "ALL_OR_NOTHING", "execution_side_effects": False,
                "registry_handoff": "VALIDATED_IMMUTABLE_SNAPSHOT", "rule_kind": "CASE_ID_SET",
                "case_policy_is_selection_rule": False}
    if any(semantics.get(key) != value for key, value in expected.items()):
        raise contracts.ContractUnavailable("UNSUPPORTED_SELECTION_CONTRACT")
    try:
        data = _snapshot_data(snapshot)
    except ValueError:
        return _result(None, diagnostics=(SelectionDiagnostic("UNVALIDATED_REGISTRY", "$.registry"),))
    if contracts.document_errors("request", request):
        code, reference = "INVALID_REQUEST", None
        if isinstance(request, dict) and request.get("kind") == "CASE_IDS":
            ids = request.get("case_ids")
            if isinstance(ids, list) and all(isinstance(value, str) and value and not any(c.isspace() for c in value) for value in ids):
                duplicates = sorted(value for value, count in Counter(ids).items() if count > 1)
                if duplicates:
                    code, reference = "DUPLICATE_CASE_ID", duplicates[0]
        return _result(snapshot, diagnostics=(SelectionDiagnostic(code, "$", reference),))
    if request["kind"] == "RULE_REF":
        identity = request["rule_ref"]
        raw = rules.get(identity) if isinstance(rules, Mapping) else None
        if not isinstance(raw, dict) or raw.get("id") != identity:
            return _result(snapshot, diagnostics=(SelectionDiagnostic("UNRESOLVED_SELECTION_RULE", "$.rule_ref", identity),))
        if contracts.document_errors("rule", raw):
            return _result(snapshot, diagnostics=(SelectionDiagnostic("UNSUPPORTED_SELECTION_RULE", "$.rule_ref", identity),))
        identities = tuple(raw["case_ids"])
    else:
        identities = tuple(request["case_ids"])
    cases, diagnostics = [], []
    for index, identity in enumerate(identities):
        case = data.case_index.get(identity)
        if case is None:
            diagnostics.append(SelectionDiagnostic("UNKNOWN_CASE", f"$.case_ids[{index}]", identity))
        else:
            cases.append(case)
    return _result(snapshot, cases, diagnostics)
