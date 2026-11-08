"""Local read-only policy applicability snapshot and deterministic triage.

This is a developer convenience entry, NOT a registered Task Engine operation
or authority evaluator. Outputs are local non-normative artifacts.
"""
from __future__ import annotations

import argparse
import json
from collections import defaultdict
from pathlib import Path
import subprocess
import sys
import tempfile


ASSESSOR = Path(".ptsip/engine/developer/policy/applicability")
GO_MODULE = Path(".ptsip/engine/developer")


def call(args: list[str], cwd: Path, stdout_file: Path, error_file: Path) -> None:
    with stdout_file.open("w", encoding="utf-8") as stdout, error_file.open(
        "w", encoding="utf-8"
    ) as stderr:
        result = subprocess.run(
            args, cwd=cwd, stdout=stdout, stderr=stderr, check=False
        )
    if result.returncode != 0:
        raise RuntimeError(
            f"COMMAND_FAILED exit={result.returncode}; stderr={error_file}; "
            f"stdout={stdout_file}"
        )


def load(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


def triage(snapshot: dict, assessment: dict) -> dict:
    if snapshot.get("source_head") != assessment.get("source_head"):
        raise ValueError("SOURCE_HEAD_MISMATCH")
    owners = {
        item["policy_id"]: item["plane"] for item in snapshot["policies"]
    }
    groups: dict[tuple[str, str, str, str], dict] = {}
    for row in assessment["decisions"]:
        policy_id = row["policy_id"]
        if policy_id not in owners:
            raise ValueError(f"UNKNOWN_POLICY_ID: {policy_id}")
        key = (
            owners[policy_id], row["contract_kind"], row["state"], row["reason"]
        )
        group = groups.setdefault(
            key,
            {
                "plane": key[0],
                "contract_kind": key[1],
                "state": key[2],
                "reason": key[3],
                "count": 0,
                "examples": [],
            },
        )
        group["count"] += 1
        if len(group["examples"]) < 3:
            group["examples"].append(
                {
                    "policy_id": policy_id,
                    "rule_id": row["rule_id"],
                    "policy_path": row["policy_path"],
                    "source_pointer": row["source_pointer"],
                }
            )
    if sum(group["count"] for group in groups.values()) != assessment["binding_count"]:
        raise ValueError("BINDING_COUNT_MISMATCH")
    by_state: dict[str, int] = defaultdict(int)
    for group in groups.values():
        by_state[group["state"]] += group["count"]
    if dict(by_state) != assessment["counts"]:
        raise ValueError("STATE_COUNT_MISMATCH")
    return {
        "format": "ptsip-applicability-triage/v1",
        "source_head": assessment["source_head"],
        "evidence_role": "NON_NORMATIVE_DERIVED",
        "policy_mutation_authorized": False,
        "evaluation_authorized": False,
        "execution_authorized": False,
        "source_assessment": "assessment.json",
        "binding_count": assessment["binding_count"],
        "counts": assessment["counts"],
        "groups": [groups[key] for key in sorted(groups)],
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--skip-tests", action="store_true", help="run only the snapshot and triage"
    )
    args = parser.parse_args()

    script_dir = Path(__file__).resolve().parent
    repo = subprocess.run(
        ["git", "-C", str(script_dir), "rev-parse", "--show-toplevel"],
        text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True,
    ).stdout.strip()
    root = Path(repo).resolve()
    head = subprocess.run(
        ["git", "rev-parse", "HEAD"],
        cwd=root, text=True, stdout=subprocess.PIPE,
        stderr=subprocess.PIPE, check=True,
    ).stdout.strip()
    if len(head) != 40 or not all(c in "0123456789abcdef" for c in head):
        raise ValueError("INVALID_GIT_HEAD")

    output_dir = root / ".agent" / "policy-applicability" / head
    output_dir.mkdir(parents=True, exist_ok=True)
    log_dir = Path(tempfile.gettempdir()) / "ptsip-policy-applicability" / head
    log_dir.mkdir(parents=True, exist_ok=True)

    snapshot_file = output_dir / "snapshot.json"
    assessment_file = output_dir / "assessment.json"
    summary_file = output_dir / "summary.json"
    triage_file = output_dir / "triage.json"

    # Snapshot adapter itself rejects uncommitted canonical source changes.
    call(
        [sys.executable, str(root / ASSESSOR / "input.py"),
         "--root", str(root)],
        root, snapshot_file, log_dir / "snapshot.stderr.log",
    )
    call(
        [
            "go", "-C", str(root / GO_MODULE), "run",
            "./policy/applicability/cmd/policy-applicability",
            "--input", str(snapshot_file),
            "--output", str(assessment_file),
            "--summary",
        ],
        root, summary_file, log_dir / "assess.stderr.log",
    )

    snapshot, assessment, summary = map(
        load, (snapshot_file, assessment_file, summary_file)
    )
    if head != snapshot.get("source_head") or head != assessment.get("source_head"):
        raise ValueError("SOURCE_HEAD_CHANGED")
    if summary.get("binding_count") != assessment.get("binding_count"):
        raise ValueError("SUMMARY_BINDING_COUNT_MISMATCH")
    if summary.get("counts") != assessment.get("counts"):
        raise ValueError("SUMMARY_STATE_COUNTS_MISMATCH")
    triage_file.write_text(
        json.dumps(triage(snapshot, assessment), indent=2, ensure_ascii=False) + "\n",
        encoding="utf-8",
    )

    if not args.skip_tests:
        call(
            ["go", "-C", str(root / GO_MODULE), "test",
             "./policy/applicability", "-count=1"],
            root, log_dir / "go-test.stdout.log", log_dir / "go-test.stderr.log",
        )
        call(
            [sys.executable, "-m", "pytest", "-q",
             "src/tests/ptsip/contracts/test_applicability_snapshot.py",
             "src/tests/ptsip/contracts/test_unified_policy.py"],
            root, log_dir / "pytest.stdout.log", log_dir / "pytest.stderr.log",
        )

    print(json.dumps({
        "status": "PASS",
        "source_head": head,
        "artifact_directory": str(output_dir),
        "log_directory": str(log_dir),
        "policy_count": assessment["policy_count"],
        "rule_count": assessment["rule_count"],
        "binding_count": assessment["binding_count"],
        "counts": assessment["counts"],
        "tests": "SKIPPED" if args.skip_tests else "PASSED",
        "policy_mutation_authorized": False,
    }, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, KeyError, RuntimeError, subprocess.CalledProcessError) as exc:
        print(f"APPLICABILITY_LOCAL_FAILED: {exc}", file=sys.stderr)
        raise SystemExit(2) from exc
