from __future__ import annotations

from ptsip.cli import _parser


def test_conform_exposes_explicit_review_evidence_inputs() -> None:
    parser = _parser()
    args = parser.parse_args(
        [
            "conform",
            ".",
            "--artifact-evidence",
            "artifact.json",
            "--agent-decision",
            "decision.json",
            "--external-evidence",
            "external.json",
            "--json",
        ]
    )
    assert args.command == "conform"
    assert args.artifact_evidence == ["artifact.json"]
    assert args.agent_decision == ["decision.json"]
    assert args.external_evidence == ["external.json"]
    assert args.json is True
