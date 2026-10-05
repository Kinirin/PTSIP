from __future__ import annotations

from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
WORKFLOW = ROOT.parent / ".github" / "workflows" / "tooling-test.yml"


def _workflow_text() -> str:
    return WORKFLOW.read_text(encoding="utf-8")


def test_static_validation_is_not_pinned_to_one_development_branch() -> None:
    text = _workflow_text()

    assert (
        "(github.event_name == 'push' && github.event.deleted == false) || "
        "(github.event_name == 'workflow_dispatch'"
    ) in text
    assert "github.ref_name == 'dev/0.3.8'" not in text


def test_wu02_lane_validation_uses_validator_owned_control_branch() -> None:
    text = _workflow_text()

    assert (
        'from developer.automation.wu02_lane_validator import CONTROL_BRANCH; '
        'print(CONTROL_BRANCH)'
    ) in text
    assert "steps.wu02-route.outputs.required == 'true'" in text
    assert (
        'python -m developer.automation.wu02_lane_validator '
        '--execution-branch "$env:PTSIP_EXECUTION_BRANCH"'
    ) in text


def test_release_push_verification_is_not_version_branch_pinned() -> None:
    text = _workflow_text()

    assert "startsWith(github.event.head_commit.message, 'release:')" in text
    assert "github.ref_name == 'dev/0.3.8'" not in text
