from __future__ import annotations

import argparse
import subprocess
from pathlib import Path

from developer.automation.policy_loader import load_yaml, repository_root


CONTROL_BRANCH = "dev/0.4.0"
PARENT_PLAN = "docs/planning/0.4.0/WU-02/WU-02.yaml"
LANES = {
    "S1": {
        "id": "WU-02-S1",
        "branch": "dev/0.4.0-WU-02-S1",
        "path": "docs/planning/0.4.0/WU-02/WU-02-S1.yaml",
    },
    "S2": {
        "id": "WU-02-S2",
        "branch": "dev/0.4.0-WU-02-S2",
        "path": "docs/planning/0.4.0/WU-02/WU-02-S2.yaml",
    },
    "S3": {
        "id": "WU-02-S3",
        "branch": "dev/0.4.0-WU-02-S3",
        "path": "docs/planning/0.4.0/WU-02/WU-02-S3.yaml",
    },
}
_REQUIRED_SECTIONS = (
    "record",
    "branch_baseline",
    "responsibility",
    "owned_implementation_surface",
    "session_protocol",
    "execution_state",
    "decisions_made",
    "unresolved_questions",
    "verification_state",
    "integration_handoff",
)


def _git(base: Path, *args: str) -> str:
    result = subprocess.run(
        ["git", *args],
        cwd=base,
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout.strip()


def _current_branch(base: Path) -> str:
    return _git(base, "branch", "--show-current")


def _resolve_control_ref(base: Path) -> str:
    for ref in ("origin/dev/0.4.0", "dev/0.4.0"):
        try:
            _git(base, "rev-parse", "--verify", ref)
        except subprocess.CalledProcessError:
            continue
        return ref
    raise RuntimeError("Cannot resolve dev/0.4.0 or origin/dev/0.4.0.")


def _changed_files(base: Path, control_ref: str) -> tuple[str, ...]:
    output = _git(base, "diff", "--name-only", f"{control_ref}...HEAD")
    return tuple(line for line in output.splitlines() if line)


def validate_lane_documents(root: str | Path | None = None) -> tuple[str, ...]:
    base = repository_root(root)
    errors: list[str] = []

    parent = load_yaml(PARENT_PLAN, root=base)
    declared = {
        entry.get("id"): entry
        for entry in parent.get("parallel_work_lanes", {}).get("canonical_lane_plans", [])
        if isinstance(entry, dict)
    }

    allowed_owners: dict[str, str] = {}
    for lane_name, expected in LANES.items():
        entry = declared.get(expected["id"])
        if not isinstance(entry, dict):
            errors.append(f"{PARENT_PLAN}: missing canonical lane {expected['id']}")
        else:
            for key in ("branch", "path"):
                if entry.get(key) != expected[key]:
                    errors.append(
                        f"{PARENT_PLAN}: {expected['id']} {key} must be {expected[key]!r}"
                    )

        payload = load_yaml(expected["path"], root=base)
        missing = [section for section in _REQUIRED_SECTIONS if section not in payload]
        if missing:
            errors.append(f"{expected['path']}: missing required sections {missing}")
            continue

        record = payload.get("record", {})
        if record.get("id") != expected["id"]:
            errors.append(f"{expected['path']}: record.id must be {expected['id']}")
        if record.get("branch") != expected["branch"]:
            errors.append(f"{expected['path']}: record.branch must be {expected['branch']}")
        if record.get("base_branch") != CONTROL_BRANCH:
            errors.append(f"{expected['path']}: record.base_branch must be {CONTROL_BRANCH}")

        baseline = payload.get("branch_baseline", {})
        if baseline.get("control_plane_branch") != CONTROL_BRANCH:
            errors.append(
                f"{expected['path']}: branch_baseline.control_plane_branch must be {CONTROL_BRANCH}"
            )
        if baseline.get("canonical_plan_source") != "CENTRAL_CONTROL_PLANE":
            errors.append(
                f"{expected['path']}: canonical_plan_source must be CENTRAL_CONTROL_PLANE"
            )

        surface = payload.get("owned_implementation_surface", {})
        allowed_paths = surface.get("allowed_paths", [])
        forbidden_paths = set(surface.get("integration_only_files_forbidden", []))
        if expected["path"] not in allowed_paths:
            errors.append(f"{expected['path']}: lane plan must be in allowed_paths")
        if expected["path"] in forbidden_paths:
            errors.append(f"{expected['path']}: lane plan cannot also be forbidden")

        for owned_path in allowed_paths:
            previous = allowed_owners.get(owned_path)
            if previous is not None and previous != lane_name:
                errors.append(f"{owned_path}: owned by both {previous} and {lane_name}")
            allowed_owners[owned_path] = lane_name

        protocol = payload.get("session_protocol", {})
        required_reads = set(protocol.get("required_read_before_work", []))
        expected_reads = {PARENT_PLAN, *(item["path"] for item in LANES.values())}
        if not expected_reads.issubset(required_reads):
            errors.append(
                f"{expected['path']}: session protocol must read parent and all lane plans"
            )

        execution = payload.get("execution_state", {})
        for key in ("completed", "remaining", "blockers"):
            if not isinstance(execution.get(key), list):
                errors.append(f"{expected['path']}: execution_state.{key} must be a list")

        verification = payload.get("verification_state", {})
        for key in ("required", "passed", "failed"):
            if not isinstance(verification.get(key), list):
                errors.append(f"{expected['path']}: verification_state.{key} must be a list")

        handoff = payload.get("integration_handoff", {})
        for key in ("requests_to_other_lanes", "integration_notes"):
            if not isinstance(handoff.get(key), list):
                errors.append(f"{expected['path']}: integration_handoff.{key} must be a list")

    return tuple(errors)


def validate_current_lane(
    lane: str,
    root: str | Path | None = None,
) -> tuple[str, ...]:
    lane = lane.upper()
    if lane not in LANES:
        return (f"Unknown lane {lane!r}; expected one of {tuple(LANES)}",)

    base = repository_root(root)
    errors = list(validate_lane_documents(base))
    expected = LANES[lane]
    payload = load_yaml(expected["path"], root=base)

    branch = _current_branch(base)
    if branch != expected["branch"]:
        errors.append(
            f"Current branch {branch!r} does not match {lane} branch {expected['branch']!r}"
        )
        return tuple(errors)

    control_ref = _resolve_control_ref(base)
    changed = _changed_files(base, control_ref)
    surface = payload.get("owned_implementation_surface", {})
    allowed = set(surface.get("allowed_paths", []))
    forbidden = set(surface.get("integration_only_files_forbidden", []))

    for path in changed:
        if path in forbidden:
            errors.append(f"{lane}: integration-only or sibling path modified: {path}")
        elif path not in allowed:
            errors.append(
                f"{lane}: path is outside lane ownership; record an integration handoff instead: {path}"
            )

    return tuple(errors)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Validate WU-02 parallel lane planning and branch ownership."
    )
    parser.add_argument(
        "lane",
        nargs="?",
        choices=tuple(LANES),
        help="Validate one leaf branch (S1, S2, or S3). Omit for control-plane planning validation.",
    )
    args = parser.parse_args(argv)

    errors = validate_current_lane(args.lane) if args.lane else validate_lane_documents()
    if errors:
        print("WU-02 lane validation: FAIL")
        for error in errors:
            print(f"- {error}")
        return 1

    print("WU-02 lane validation: PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
