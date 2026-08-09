from __future__ import annotations

from pathlib import Path

from ..repository.snapshot import repository_files
from .dependencies import DependencyScan, DependencyScanIssue, scan_dependency_edges as scan_legacy_dependency_edges
from .javascript import discover_npm_packages, manifest_edges, source_edges


_JS_TS_SUFFIXES = {".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".mts", ".cts"}


def scan_dependency_edges(root: str | Path) -> DependencyScan:
    root = Path(root).resolve()
    legacy = scan_legacy_dependency_edges(root)
    _mode, paths, discovery_errors = repository_files(root)

    edges = list(legacy.edges)
    issues = list(legacy.issues)
    adapters = set(legacy.adapters)
    for error in discovery_errors:
        issue = DependencyScanIssue("repository", "<repository>", error)
        if issue not in issues:
            issues.append(issue)

    by_path, by_name, npm_issues = discover_npm_packages(root, paths)
    if by_path or npm_issues:
        adapters.add("npm-manifest")
    for rel, message in npm_issues:
        issues.append(DependencyScanIssue("npm-manifest", rel, message))

    for package in by_path.values():
        edges.extend(manifest_edges(package, by_name))

    for rel in paths:
        if Path(rel).suffix.lower() not in _JS_TS_SUFFIXES:
            continue
        adapters.add("javascript-typescript")
        found, found_issues = source_edges(root, rel, by_path, by_name)
        edges.extend(found)
        issues.extend(DependencyScanIssue("javascript-typescript", rel, message) for message in found_issues)

    unique_edges = {item.evidence_id: item for item in edges}
    ordered_edges = sorted(
        unique_edges.values(),
        key=lambda item: (item.source, item.line or 0, item.target, item.evidence_id),
    )
    unique_issues = {(item.adapter, item.path, item.message): item for item in issues}
    ordered_issues = [unique_issues[key] for key in sorted(unique_issues)]
    return DependencyScan(tuple(ordered_edges), tuple(ordered_issues), tuple(sorted(adapters)))
