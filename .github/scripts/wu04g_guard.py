from __future__ import annotations

import argparse
import ast
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
DEFAULT_BASE = "52a455115d191123504c2fd690ffe499caf0ff6a"
PLAN = ROOT / "planning/0.3.6/WU-04G-clarification-adoption-effective-map.md"
EXPECTED_BRANCH = "tool-0.3.6-lifecycle-ownership"

G_SCOPE_TESTS = {
    "tests/ptsip/test_clarification.py",
    "tests/ptsip/test_adoption_033.py",
    "tests/ptsip/test_decision_control_plane.py",
    "tests/ptsip/test_local_control_plane_033.py",
    "tests/ptsip/test_github_authority_033.py",
    "tests/ptsip/test_github_authority_034.py",
    "tests/ptsip/test_repository_self_profile_035.py",
    "tests/ptsip/test_topology_032.py",
    "tests/ptsip/test_clarification_adoption_effective_map_036.py",
    "tests/ptsip/_wu04g_support.py",
}

EXCLUDED_TESTS = {
    "tests/ptsip/test_conformance_030.py",
    "tests/ptsip/test_conformance_effective_map_036.py",
    "tests/ptsip/test_conformance_engine_030.py",
    "tests/ptsip/test_evidence_correctness_023.py",
    "tests/ptsip/test_merge_gate_followup_030.py",
    "tests/ptsip/test_merge_gate_remediation_030.py",
    "tests/ptsip/test_profile_validation_036.py",
    "tests/ptsip/test_remaining_030.py",
}

AUTHORIZED_G036_TEST = "tests/ptsip/test_clarification_adoption_effective_map_036.py"
AUTHORIZED_SUPPORT = "tests/ptsip/_wu04g_support.py"


def _git(*args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        cwd=ROOT,
        text=True,
        encoding="utf-8",
        errors="replace",
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )


def _require_git_revision(revision: str) -> list[str]:
    result = _git("rev-parse", "--verify", f"{revision}^{{commit}}")
    if result.returncode == 0:
        return []
    return [
        f"base revision {revision!r} is unavailable locally; fetch repository history before auditing"
    ]


def _changed_files(base: str) -> tuple[list[str], list[str]]:
    errors = _require_git_revision(base)
    if errors:
        return [], errors
    commands = (
        ("diff", "--name-only", f"{base}...HEAD"),
        ("diff", "--name-only"),
        ("diff", "--cached", "--name-only"),
        ("ls-files", "--others", "--exclude-standard"),
    )
    files: set[str] = set()
    for command in commands:
        result = _git(*command)
        if result.returncode != 0:
            return [], [result.stderr.strip() or f"git {' '.join(command)} failed"]
        files.update(
            line.strip().replace("\\", "/")
            for line in result.stdout.splitlines()
            if line.strip()
        )
    return sorted(files), []


def _test_count(source: str, label: str) -> tuple[int | None, str | None]:
    try:
        tree = ast.parse(source, filename=label)
    except SyntaxError as exc:
        return None, f"cannot parse {label}: {exc}"
    count = 0
    for node in ast.walk(tree):
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)) and node.name.startswith("test_"):
            count += 1
    return count, None


def _source_at(base: str, path: str) -> str | None:
    result = _git("show", f"{base}:{path}")
    if result.returncode != 0:
        return None
    return result.stdout


def precheck() -> int:
    errors: list[str] = []
    branch = _git("branch", "--show-current")
    head = _git("rev-parse", "HEAD")
    status = _git("status", "--porcelain")

    branch_name = branch.stdout.strip() if branch.returncode == 0 else "<unknown>"
    head_sha = head.stdout.strip() if head.returncode == 0 else "<unknown>"

    if branch_name != EXPECTED_BRANCH:
        errors.append(f"expected branch {EXPECTED_BRANCH!r}, found {branch_name!r}")
    if not PLAN.is_file():
        errors.append(f"missing WU-04G plan: {PLAN.relative_to(ROOT)}")
    else:
        text = PLAN.read_text(encoding="utf-8")
        required_markers = (
            "WU-04G  ACTIVE",
            "WU-04H  LOCKED",
            "WU-04I  LOCKED",
            "D9-B",
            "tests/ptsip/test_clarification.py",
        )
        for marker in required_markers:
            if marker not in text:
                errors.append(f"WU-04G plan is missing expected marker: {marker!r}")

    print(f"WU-04G branch: {branch_name}")
    print(f"WU-04G HEAD:   {head_sha}")
    print(f"Working tree:  {'dirty' if status.stdout.strip() else 'clean'}")

    if errors:
        print("WU-04G precheck: FAIL", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print("WU-04G precheck: PASS")
    return 0


def scope(base: str) -> int:
    changed, errors = _changed_files(base)
    if errors:
        print("WU-04G scope guard: FAIL", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    violations: list[str] = []
    changed_tests = [path for path in changed if path.startswith("tests/")]

    for path in changed_tests:
        if path.startswith("tests/vpms/"):
            violations.append(f"locked H/VPMS test changed: {path}")
            continue
        if path in EXCLUDED_TESTS:
            violations.append(f"completed/deferred test family changed: {path}")
            continue
        if path.startswith("tests/ptsip/") and path not in G_SCOPE_TESTS:
            violations.append(f"test file outside authoritative WU-04G set changed: {path}")

    for path in changed:
        if path.startswith(".github/workflows/"):
            violations.append(f"workflow change is outside D9-B: {path}")
        if (
            path.startswith("tests/ptsip/test_")
            and "_036.py" in path
            and path != AUTHORIZED_G036_TEST
        ):
            violations.append(f"unauthorized new/changed WU-04G 0.3.6 test file: {path}")
        if path.startswith("tests/ptsip/_wu04g") and path != AUTHORIZED_SUPPORT:
            violations.append(f"unauthorized WU-04G support module: {path}")

    print(f"Scope base: {base}")
    print(f"Changed files: {len(changed)}")
    print(f"Changed tests: {len(changed_tests)}")
    for path in changed_tests:
        print(f"  {path}")

    if violations:
        print("WU-04G scope guard: FAIL", file=sys.stderr)
        for violation in violations:
            print(f"- {violation}", file=sys.stderr)
        return 1

    print("WU-04G scope guard: PASS")
    return 0


def report(base: str) -> int:
    changed, errors = _changed_files(base)
    if errors:
        print("WU-04G migration report: FAIL", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print(f"WU-04G migration report from {base} to HEAD")
    print("test file | baseline tests | current tests | delta")
    print("--- | ---: | ---: | ---:")
    parse_errors: list[str] = []

    for path in sorted(G_SCOPE_TESTS):
        if not path.endswith(".py") or path.endswith("_support.py"):
            continue
        baseline_source = _source_at(base, path)
        current_path = ROOT / path
        current_source = current_path.read_text(encoding="utf-8") if current_path.is_file() else None

        base_count = 0
        if baseline_source is not None:
            parsed, error = _test_count(baseline_source, f"{base}:{path}")
            if error:
                parse_errors.append(error)
            elif parsed is not None:
                base_count = parsed

        current_count = 0
        if current_source is not None:
            parsed, error = _test_count(current_source, path)
            if error:
                parse_errors.append(error)
            elif parsed is not None:
                current_count = parsed

        marker = " *" if path in changed else ""
        print(f"{path}{marker} | {base_count} | {current_count} | {current_count - base_count:+d}")

    if parse_errors:
        print("WU-04G migration report: FAIL", file=sys.stderr)
        for error in parse_errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print("\n* changed since base")
    print("Test-count deltas are diagnostic only; semantic replacement mapping remains authoritative.")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description="WU-04G scope and migration guard")
    parser.add_argument("command", choices=("precheck", "scope", "report", "all"))
    parser.add_argument("--base", default=DEFAULT_BASE, help="comparison base commit")
    args = parser.parse_args()

    if args.command == "precheck":
        return precheck()
    if args.command == "scope":
        return scope(args.base)
    if args.command == "report":
        return report(args.base)

    results = (precheck(), scope(args.base), report(args.base))
    return 0 if all(code == 0 for code in results) else 1


if __name__ == "__main__":
    raise SystemExit(main())
