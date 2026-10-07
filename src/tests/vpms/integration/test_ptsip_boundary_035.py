from __future__ import annotations

from pathlib import Path

from ptsip.app.store import DecisionStore
from vpms.domain.model import (
    FormulaRef,
    PolicyRef,
    RunnerRef,
    TargetRef,
    VariablesRef,
    VerificationCase,
    VerificationOutcome,
    VerificationPurpose,
)
from vpms.execution.runner import RunnerExecution, run_case
from vpms.integration.ptsip_bridge import load_ptsip_metadata, resolve_target_metadata


_PTSIP_CONFORMANCE_OUTCOMES = frozenset({"CONFORMANT", "NON_CONFORMANT", "INCOMPLETE"})


class _PassExecutor:
    def execute(self, case: VerificationCase) -> RunnerExecution:
        return RunnerExecution(outcome=VerificationOutcome.PASS)


def _case() -> VerificationCase:
    return VerificationCase(
        id="product.boundary-check",
        purpose=VerificationPurpose.PRODUCT,
        target=TargetRef(component_id="verifier-sdk"),
        formula=FormulaRef(ref="boundary.formula"),
        variables=VariablesRef(ref="product.variables"),
        policy=PolicyRef(ref="product.policy"),
        runner=RunnerRef(ref="in-memory.pass"),
    )


def _write_profile(tmp_path: Path) -> Path:
    profile = tmp_path / "ptsip.yaml"
    profile.write_text(
        """
components:
  - id: verifier-sdk
    classification: TOOLCHAIN
    purpose: development_tooling
""".strip(),
        encoding="utf-8",
    )
    return profile


def _seed_authority(store: DecisionStore) -> dict[str, object]:
    record, stale = store.gate(
        {
            "id": "clr-boundary",
            "repository": "example/project",
            "branch": "main",
            "subject_revision": "abc123",
            "component_id": "verifier-sdk",
            "request": {
                "id": "clr-boundary",
                "component_id": "verifier-sdk",
                "status": "INCOMPLETE",
            },
        }
    )
    assert stale == ()
    return record.as_dict()


def test_vpms_execution_does_not_mutate_ptsip_profile_classification_or_authority(
    tmp_path: Path,
) -> None:
    profile = _write_profile(tmp_path)
    profile_before = profile.read_bytes()
    metadata_before = load_ptsip_metadata(profile)
    target_before = resolve_target_metadata(_case().target, metadata_before)
    assert target_before is not None
    assert target_before.classification == "TOOLCHAIN"

    store = DecisionStore(tmp_path / "authority.sqlite3")
    authority_before = _seed_authority(store)

    result = run_case(_case(), _PassExecutor())

    metadata_after = load_ptsip_metadata(profile)
    target_after = resolve_target_metadata(_case().target, metadata_after)
    authority_after = store.get("clr-boundary")

    assert result.outcome is VerificationOutcome.PASS
    assert result.purpose is VerificationPurpose.PRODUCT
    assert result.target.component_id == "verifier-sdk"
    assert target_after == target_before
    assert target_after is not None
    assert target_after.classification == "TOOLCHAIN"
    assert profile.read_bytes() == profile_before
    assert authority_after is not None
    assert authority_after.as_dict() == authority_before


def test_vpms_result_contract_does_not_collapse_into_ptsip_conformance() -> None:
    result = run_case(_case(), _PassExecutor())
    payload = result.as_dict()

    assert result.outcome is VerificationOutcome.PASS
    assert result.outcome.value not in _PTSIP_CONFORMANCE_OUTCOMES
    assert set(VerificationOutcome) == {
        VerificationOutcome.PASS,
        VerificationOutcome.FAIL,
        VerificationOutcome.ERROR,
        VerificationOutcome.SKIPPED,
    }
    assert not _PTSIP_CONFORMANCE_OUTCOMES.intersection(
        outcome.value for outcome in VerificationOutcome
    )
    assert "conformance" not in payload
    assert "ptsip_outcome" not in payload
