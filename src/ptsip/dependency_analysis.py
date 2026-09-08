"""Non-authoritative, deterministic dependency actionability for bounded review."""
from __future__ import annotations

import ast
from collections import Counter
from enum import StrEnum
from pathlib import Path
import tokenize

from .dependency_reconciliation import (
    _IMPORT_DISTRIBUTION_ALIASES, _normalize_name, _read_declarations,
    reconcile_dependency_evidence, reconcile_dependency_phases,
)
from .conformance import _dependency_coverage_gaps
from .dependency_cache import EvidenceCache
from .inspection.dependencies_030 import scan_dependency_edges
from .inspection.dependencies import _declared_python_dependencies
from .model import EvidenceNodeScope, ResolutionStatus
from .repository.discover import discover_repository
from .repository.snapshot import capture_snapshot, compare_snapshots, repository_files
from .validation.components import partition_components
from .validation.profile import validate_profile
from .validation.rules import evaluate_declared_dependency_boundaries


class Actionability(StrEnum):
    AUTO_RESOLVED = "AUTO_RESOLVED"
    REPOSITORY_DEFECT = "REPOSITORY_DEFECT"
    REVIEW_REQUIRED = "REVIEW_REQUIRED"
    RESOLVER_LIMITATION = "RESOLVER_LIMITATION"


def python_context(root: Path, rel: str) -> dict[int, dict[str, object]]:
    """Collect syntax facts once per file; neither execute nor import consumer code."""
    try:
        with tokenize.open(root / rel) as stream:
            tree = ast.parse(stream.read(), filename=rel)
    except (OSError, UnicodeError, SyntaxError):
        return {}
    parents = {child: node for node in ast.walk(tree) for child in ast.iter_child_nodes(node)}
    import_functions = {alias.asname or alias.name for node in ast.walk(tree)
                        if isinstance(node, ast.ImportFrom) and node.module == "importlib" and not node.level
                        for alias in node.names if alias.name == "import_module"}
    import_functions.add("__import__")
    contexts = {}
    for node in ast.walk(tree):
        dynamic = isinstance(node, ast.Call) and (
            isinstance(node.func, ast.Name) and node.func.id in import_functions
            or isinstance(node.func, ast.Attribute) and node.func.attr == "import_module"
        )
        if not isinstance(node, (ast.Import, ast.ImportFrom)) and not dynamic:
            continue
        ancestors = []
        parent = parents.get(node)
        while parent is not None:
            ancestors.append(parent)
            parent = parents.get(parent)
        guards = [item for item in ancestors if isinstance(item, (ast.Try, ast.TryStar, ast.If))]
        handlers = [handler for item in guards if isinstance(item, (ast.Try, ast.TryStar))
                    for handler in item.handlers]
        names = {alias.asname or alias.name.split(".")[0] for alias in getattr(node, "names", [])}
        usages = sorted({item.lineno for item in ast.walk(tree)
                         if isinstance(item, ast.Name) and isinstance(item.ctx, ast.Load) and item.id in names})
        contexts[node.lineno] = {
            "import_style": "lazy" if any(isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef))
                                           for item in ancestors) else "direct",
            "relative": isinstance(node, ast.ImportFrom) and node.level > 0,
            "guarded": bool(guards),
            "guard_lines": sorted({item.lineno for item in guards})[:4],
            "fallback": {"detected": bool(handlers), "evidence": [
                {"path": rel, "line": handler.lineno, "kind": "EXCEPTION_HANDLER_PRESENT"}
                for handler in handlers[:4]
            ], "semantics": "syntax_presence_only"},
            "usage_lines": usages[:8],
            "usage_locations_total": len(usages),
        }
    return contexts


def classify_dependencies(root, dependencies, components, partition, reconciliation):
    """Route every edge once. Actionability never suppresses conformance evidence."""
    root = Path(root)
    owners = {item.path: item.component_id for item in partition.assignments}
    metadata = {item["id"]: item for item in components}
    reconciled = {item.evidence_id: item.as_dict() for item in reconciliation.resolved_external}
    declarations, declaration_errors = _read_declarations(root, partition)
    native_declarations, native_declaration_issues = _declared_python_dependencies(root)
    declaration_paths = {}
    for declaration in declarations:
        declaration_paths.setdefault(declaration.distribution, set()).add(declaration.path)
    boundary = {evidence: finding for finding in evaluate_declared_dependency_boundaries(
        components, partition, dependencies) for evidence in finding.evidence_ids}
    _, paths, errors = repository_files(root)
    local_names = {Path(path).stem for path in paths if path.endswith(".py")}
    local_names.update(part for path in paths if path.endswith(".py") for part in Path(path).parts[:-1])
    context_cache = EvidenceCache(root, paths)
    contexts = {path: {int(line): item for line, item in context_cache.get(
        "python-context", path, lambda: {str(line): item for line, item in python_context(root, path).items()},
        lambda value: isinstance(value, dict) and all(str(key).isdigit() and isinstance(item, dict)
                                                     for key, item in value.items())).items()} for path in sorted({
        edge.source for edge in dependencies.edges if edge.adapter == "python"})}
    issue_paths = {item.path for item in dependencies.issues}
    incoming = {}
    for edge in dependencies.edges:
        if edge.resolved_path:
            incoming.setdefault(edge.resolved_path, []).append({
                "path": edge.source, "line": edge.line, "evidence_id": edge.evidence_id,
                "kind": "FILE_DEPENDENCY_CALLER",
            })
    items = []
    for edge in dependencies.edges:
        owner = owners.get(edge.source)
        component = metadata.get(owner, {})
        target_owner = owners.get(edge.resolved_path)
        usage = contexts.get(edge.source, {}).get(edge.line, {})
        declaration = reconciled.get(edge.evidence_id)
        import_root = _normalize_name(edge.target.lstrip(".").split(".", 1)[0])
        distribution = _IMPORT_DISTRIBUTION_ALIASES.get(import_root, import_root)
        observed_declaration = (edge.adapter == "python" and edge.resolution == ResolutionStatus.UNRESOLVED
                                and (distribution in declaration_paths or distribution in native_declarations))
        remediation = None
        finding = boundary.get(edge.evidence_id)
        if not owner or errors or partition.scan_errors or edge.source in issue_paths:
            state, reason = Actionability.RESOLVER_LIMITATION, "EVIDENCE_INCOMPLETE"
        elif finding:
            state = Actionability.REPOSITORY_DEFECT if finding.severity == "ERROR" else Actionability.REVIEW_REQUIRED
            reason = finding.rule_id
        elif declaration:
            state, reason = Actionability.AUTO_RESOLVED, declaration["basis"]
        elif edge.resolution == ResolutionStatus.RESOLVED and target_owner:
            state, reason = Actionability.AUTO_RESOLVED, "RESOLVED_OWNED_TARGET"
        elif edge.resolution == ResolutionStatus.EXTERNAL and edge.target_scope in {
            EvidenceNodeScope.EXTERNAL_DEPENDENCY, EvidenceNodeScope.PLATFORM
        }:
            state, reason = Actionability.AUTO_RESOLVED, "RESOLVED_EXTERNAL_OR_PLATFORM"
        elif observed_declaration:
            # Declaration presence prevents an absence-based remediation. It
            # does not establish alias/ownership eligibility or clear a gap.
            state, reason = Actionability.RESOLVER_LIMITATION, "DECLARATION_PRESENT_RESOLUTION_INCOMPLETE"
        elif edge.phase.value == "TEST":
            state, reason = Actionability.RESOLVER_LIMITATION, "VERIFICATION_DEPENDENCY_EVIDENCE_INCOMPLETE"
        elif edge.resolution == ResolutionStatus.DYNAMIC:
            state, reason = Actionability.REVIEW_REQUIRED, "DYNAMIC_TARGET_CONTRACT_REQUIRED"
        elif usage.get("guarded"):
            state, reason = Actionability.REVIEW_REQUIRED, "GUARDED_IMPORT_SUPPORT_CONTRACT_REQUIRED"
        elif (usage.get("relative") or edge.target.startswith((".", "<"))
              or edge.target.split(".")[0] in local_names or edge.resolved_path):
            state, reason = Actionability.RESOLVER_LIMITATION, "LOCAL_TARGET_OR_NAMESPACE_UNRESOLVED"
        elif edge.adapter == "python" and (declaration_errors or native_declaration_issues):
            state, reason = Actionability.RESOLVER_LIMITATION, "DECLARATION_EVIDENCE_INCOMPLETE"
        elif edge.adapter == "python" and component.get("runtime_required") is True:
            state, reason = Actionability.REPOSITORY_DEFECT, "DIRECT_DEPENDENCY_EVIDENCE_MISSING"
            remediation = {
                "kind": "ADD_RUNTIME_REQUIREMENT", "component": owner,
                "import_name": edge.target.split(".")[0], "distribution": None,
                "version": None, "requires_distribution_decision": True,
                "requires_version_decision": True, "automatic_apply": False,
            }
        else:
            state, reason = Actionability.RESOLVER_LIMITATION, "EVIDENCE_INCOMPLETE"
        items.append({
            "evidence_id": edge.evidence_id, "actionability": state.value, "reason": reason,
            "dependency": edge.as_dict(),
            "component": {key: component.get(key) for key in (
                "id", "classification", "roles", "shipped", "runtime_required")},
            "target_component": target_owner,
            "declaration": {"found": bool(declaration) or observed_declaration or (
                                edge.resolution == ResolutionStatus.EXTERNAL
                                and edge.target_scope == EvidenceNodeScope.EXTERNAL_DEPENDENCY),
                            "reconciliation": declaration, "native_basis": edge.note,
                            "unresolved_candidate": {
                                "distribution": distribution,
                                "owned_requirement_paths": sorted(declaration_paths.get(distribution, [])),
                                "native_manifest_match": distribution in native_declarations,
                                "semantics": "DECLARATION_PRESENCE_ONLY_NOT_RESOLUTION_AUTHORITY",
                            } if observed_declaration else None},
            "usage": usage,
            "dynamic_import_kind": edge.note.split(":", 1)[0] if edge.note and "DYNAMIC_IMPORT" in edge.note else None,
            "callers": sorted(incoming.get(edge.source, []), key=lambda item: item["evidence_id"])[:4],
            "callers_total": len(incoming.get(edge.source, [])),
            "remediation_candidate": remediation,
        })
    return sorted(items, key=lambda item: item["evidence_id"])


def analyze_dependencies(path=".", profile_path=None, *, component=None):
    repo = discover_repository(path)
    root = Path(repo.root)
    before = capture_snapshot(root)
    validation = validate_profile(root, profile_path)
    payload = (validation.resolved_profile.effective_payload
               if validation.valid and validation.resolved_profile else {})
    components = payload.get("components", [])
    if component is not None and component not in {item["id"] for item in components}:
        raise ValueError(f"Unknown validated component: {component}")
    partition = partition_components(root, components)
    dependencies = scan_dependency_edges(root)
    dependencies = reconcile_dependency_phases(dependencies, components, partition)
    reconciliation = reconcile_dependency_evidence(root, dependencies, components, partition)
    items = classify_dependencies(root, dependencies, components, partition, reconciliation)
    if component:
        items = [item for item in items if item["component"]["id"] == component]
    selected_ids = {item["evidence_id"] for item in items}
    def blocking_count(reconciled_ids=frozenset()):
        return sum(1 for gap in _dependency_coverage_gaps(
            dependencies, components, partition, reconciled_ids)
            if gap["blocking"] and (not component or selected_ids.intersection(gap["evidence_ids"])))
    after = capture_snapshot(root)
    comparison = compare_snapshots(before, after)
    stable = comparison.stable and validation.valid and not partition.conflicts
    if not stable:
        for item in items:
            item.update(actionability=Actionability.RESOLVER_LIMITATION.value,
                        reason="SNAPSHOT_OR_PROFILE_INVALID", remediation_candidate=None)
    counts = Counter(item["actionability"] for item in items)
    return {
        "format": "ptsip-dependency-analysis/v1", "status": "RAN" if stable else "BLOCKED",
        "repository": repo.as_dict(), "component": component,
        "snapshot": {"before": before.as_dict(), "after": after.as_dict(),
                     "comparison": comparison.as_dict()},
        "summary": {"observed_edges": len(items), **{state.value: counts[state.value] for state in Actionability},
                    "ai_reviewed_items": 0, "machine_classified": len(items),
                    "verification_dependency_edges": sum(item["dependency"]["phase"] == "TEST" for item in items),
                    "blocking_before_reconciliation": blocking_count() if stable else None,
                    "blocking_after_reconciliation": blocking_count(reconciliation.resolved_evidence_ids) if stable else None},
        "items": items, "issues": [issue.as_dict() for issue in dependencies.issues],
        "cache": dependencies.cache,
        "profile_errors": validation.errors,
        "profile_path": validation.profile_path,
        "dependency_reconciliation": reconciliation.as_dict(),
        "authority": "ADVISORY_ONLY", "conformance_evaluated": False,
    }
