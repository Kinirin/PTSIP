from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
FOCUSED = "tests/ptsip/test_clarification_adoption_effective_map_036.py"

EXISTING_G_SCOPE = (
    "tests/ptsip/test_clarification.py",
    "tests/ptsip/test_adoption_033.py",
    "tests/ptsip/test_decision_control_plane.py",
    "tests/ptsip/test_local_control_plane_033.py",
    "tests/ptsip/test_github_authority_033.py",
    "tests/ptsip/test_github_authority_034.py",
    "tests/ptsip/test_repository_self_profile_035.py",
    "tests/ptsip/test_topology_032.py",
)

FILE_TRACKS = {
    "G1": (
        "tests/ptsip/test_clarification.py",
        "tests/ptsip/test_adoption_033.py",
        "tests/ptsip/test_repository_self_profile_035.py",
    ),
    "G2": (
        "tests/ptsip/test_clarification.py",
        "tests/ptsip/test_decision_control_plane.py",
        "tests/ptsip/test_local_control_plane_033.py",
        "tests/ptsip/test_github_authority_033.py",
        "tests/ptsip/test_github_authority_034.py",
        "tests/ptsip/test_topology_032.py",
    ),
    "G3": (
        "tests/ptsip/test_adoption_033.py",
        "tests/ptsip/test_decision_control_plane.py",
        "tests/ptsip/test_topology_032.py",
    ),
    "G4": (
        "tests/ptsip/test_adoption_033.py",
        "tests/ptsip/test_decision_control_plane.py",
        "tests/ptsip/test_local_control_plane_033.py",
        "tests/ptsip/test_github_authority_033.py",
        "tests/ptsip/test_github_authority_034.py",
        "tests/ptsip/test_topology_032.py",
    ),
    "G5": EXISTING_G_SCOPE,
}

FOCUSED_CLASSES = {
    "G1": "TestG1EffectiveReadCoverage",
    "G2": "TestG2DecisionProtocolV2",
    "G3": "TestG3HybridSafeApply",
    "G4": "TestG4ProfilePathControlPlane",
    "G5": "TestG5RecoveryAndIntegration",
}


def _path_part(node_id: str) -> str:
    return node_id.split("::", 1)[0]


def _targets(track: str, mode: str) -> tuple[str, ...]:
    if track == "baseline":
        return EXISTING_G_SCOPE
    focused = (f"{FOCUSED}::{FOCUSED_CLASSES[track]}",)
    if mode == "focused":
        return focused
    if mode == "files":
        return (*FILE_TRACKS[track], *focused)
    return (*EXISTING_G_SCOPE, FOCUSED)


def _validate_targets(track: str, targets: tuple[str, ...]) -> list[str]:
    errors: list[str] = []
    for target in targets:
        path = ROOT / _path_part(target)
        if path.is_file():
            continue
        errors.append(f"missing test target for {track}: {_path_part(target)}")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description="Run an exact WU-04G pytest track set")
    parser.add_argument("track", choices=("baseline", "G1", "G2", "G3", "G4", "G5"))
    parser.add_argument(
        "--mode",
        choices=("focused", "files", "all-g"),
        default="focused",
        help="new focused class only, participating whole files, or complete G migration set",
    )
    parser.add_argument("--durations", type=int, default=0, help="show N slowest tests")
    parser.add_argument("--durations-min", type=float, default=0.05)
    parser.add_argument("--list", action="store_true", help="print pytest command without running it")
    parser.add_argument("pytest_args", nargs=argparse.REMAINDER, help="extra pytest args after --")
    args = parser.parse_args()

    mode = "files" if args.track == "baseline" else args.mode
    targets = _targets(args.track, mode)
    errors = _validate_targets(args.track, targets)
    if errors:
        print("WU-04G track test runner: FAIL", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 2

    durations = args.durations
    if durations == 0 and args.track in {"baseline", "G5"} and mode in {"files", "all-g"}:
        durations = 30

    command = [sys.executable, "-m", "pytest", "-q", *targets]
    if durations > 0:
        command.extend([f"--durations={durations}", f"--durations-min={args.durations_min}"])
    if args.pytest_args:
        extras = args.pytest_args[1:] if args.pytest_args[0] == "--" else args.pytest_args
        command.extend(extras)

    print("WU-04G pytest command:")
    print(" ".join(command))
    if args.list:
        return 0

    result = subprocess.run(command, cwd=ROOT, check=False)
    return result.returncode


if __name__ == "__main__":
    raise SystemExit(main())
