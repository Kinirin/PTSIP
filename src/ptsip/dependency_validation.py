"""Advisory validation selection from declared ownership and observed imports."""
from __future__ import annotations

from pathlib import Path
import re
import tomllib

from .validation.components import partition_components, selector_matches_path
from .validation.profile import validate_profile
from .repository.snapshot import capture_snapshot, compare_snapshots


def validation_plan(root, report, *, changed_paths=()):
    before = capture_snapshot(root)
    expected = report.get("snapshot", {}).get("after", {})
    if any(before.as_dict()[key] != expected.get(key) for key in (
            "head", "status_fingerprint", "tracked_content_fingerprint")):
        return {"status": "BLOCKED", "reason": "STALE_ANALYSIS", "steps": []}
    result = _validation_plan(root, report, changed_paths=changed_paths)
    if not compare_snapshots(before, capture_snapshot(root)).stable:
        return {"status": "BLOCKED", "reason": "SNAPSHOT_CHANGED", "steps": []}
    return result


def _validation_plan(root, report, *, changed_paths=()):
    root = Path(root).resolve()
    validation = validate_profile(root, report.get("profile_path"))
    if not validation.valid or not validation.resolved_profile or report["status"] != "RAN":
        return {"status": "BLOCKED", "reason": "VALID_PROFILE_AND_SNAPSHOT_REQUIRED", "steps": []}
    payload = validation.resolved_profile.effective_payload
    components = {item["id"]: item for item in payload["components"]}
    partition = partition_components(root, list(components.values()))
    owners = {item.path: item.component_id for item in partition.assignments}
    changed = sorted(set(str(path).replace("\\", "/").removeprefix("./") for path in changed_paths))
    if any(path not in owners for path in changed):
        return {"status": "BLOCKED", "reason": "CHANGED_PATH_OWNERSHIP_REQUIRED", "steps": []}
    affected = {owners[path] for path in changed}
    if not changed:
        affected = {item["component"]["id"] for item in report["items"]
                    if item["actionability"] != "AUTO_RESOLVED"}
    links = {(item["from"], item["to"]) for item in payload.get("relationships", [])
             if item.get("type") == "VERIFIES"}
    selected = {key for key, meta in components.items() if "VERIFICATION" in meta.get("roles", [])
                and (key in affected or any((key, target) in links for target in affected)
                     or any(selector_matches_path(path, pattern) for path in changed
                            for pattern in meta.get("analysis_inputs", [])))}
    tests = sorted(path for path, owner in owners.items() if owner in selected
                   and Path(path).name.startswith("test_") and path.endswith(".py"))
    runner_evidence = []
    for item in report["items"]:
        edge = item["dependency"]
        if edge["source"] in tests and edge["target"] == "pytest":
            runner_evidence.append(item["evidence_id"])
    try:
        project = tomllib.loads((root / "pyproject.toml").read_text(encoding="utf-8-sig")).get("project", {})
        if not isinstance(project, dict):
            raise ValueError("Project metadata must be a table")
        requirements = project.get("dependencies", [])
        optional = project.get("optional-dependencies", {})
        if not isinstance(requirements, list) or not isinstance(optional, dict):
            raise ValueError("Dependency metadata has an invalid shape")
        requirements = requirements + [value for group in optional.values() if isinstance(group, list) for value in group]
        if any(isinstance(value, str) and re.match(r"^pytest(?:\s|[<>=!~\[;@]|$)", value.strip())
               for value in requirements):
            runner_evidence.append("manifest:pyproject.toml:pytest")
    except (OSError, ValueError, TypeError):
        pass
    if not tests or not runner_evidence:
        return {"status": "REVIEW_REQUIRED", "reason": "VERIFICATION_EXECUTION_EVIDENCE_REQUIRED",
                "verification_components": sorted(selected), "steps": [], "authority": "ADVISORY_ONLY"}
    graph = {}
    for item in report["items"]:
        edge = item["dependency"]
        if edge.get("resolved_path"):
            graph.setdefault(edge["source"], set()).add(edge["resolved_path"])
    def reaches_change(path):
        visited, pending = set(), [path]
        while pending:
            current = pending.pop()
            if current in changed:
                return True
            if current not in visited:
                visited.add(current)
                pending.extend(graph.get(current, ()))
        return False
    focused = [path for path in tests if reaches_change(path)]
    steps = []
    if focused:
        steps.append({"scope": "focused", "command": ["python", "-m", "pytest", *focused, "-q"],
                      "run_when": "focused_change", "basis": "OBSERVED_IMPORT_REACHABILITY_OR_CHANGED_TEST"})
    # Only collapse to a directory if every tracked test in it belongs to the
    # selected verification components; never broaden to unrelated tests.
    parents = sorted({str(Path(path).parent).replace("\\", "/") for path in tests})
    targets = []
    for parent in parents:
        peers = [path for path in owners if path.startswith(parent + "/")
                 and Path(path).name.startswith("test_") and path.endswith(".py")]
        targets.extend([parent] if all(path in tests for path in peers)
                       else [path for path in tests if str(Path(path).parent).replace("\\", "/") == parent])
    steps.append({"scope": "component", "command": ["python", "-m", "pytest", *targets, "-q", "--maxfail=10"],
                  "run_when": "batch_complete", "basis": "DECLARED_VERIFICATION_OWNERSHIP"})
    return {"format": "ptsip-dependency-validation-plan/v1", "status": "PROPOSED",
            "changed_paths": changed, "verification_components": sorted(selected),
            "runner_evidence": sorted(runner_evidence), "steps": steps,
            "authority": "ADVISORY_ONLY", "executed": False,
            "checkpoint": "Run the repository-declared full regression and strict conform at a WU checkpoint"}
