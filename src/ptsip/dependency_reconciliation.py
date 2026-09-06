from __future__ import annotations

import re
from dataclasses import asdict, dataclass
from pathlib import Path

from .inspection.dependencies import DependencyScan
from .model import ResolutionStatus
from .validation.components import ComponentPartition

_REQUIREMENT_NAME_RE = re.compile(r"^\s*([A-Za-z0-9][A-Za-z0-9_.-]*)")
_IMPORT_DISTRIBUTION_ALIASES = {
    "pil": "pillow",
    "yaml": "pyyaml",
}


def _normalize_name(value: str) -> str:
    return re.sub(r"[-.]", "_", value.strip().lower())


def _is_requirement_manifest(path: str) -> bool:
    name = Path(path).name.lower()
    return Path(path).suffix.lower() in {".txt", ".in"} and "requirements" in name


@dataclass(frozen=True)
class DependencyDeclaration:
    component_id: str
    path: str
    distribution: str
    runtime_manifest: bool

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class ReconciledDependency:
    evidence_id: str
    source_component: str
    import_name: str
    distribution: str
    declaration_paths: tuple[str, ...]
    basis: str
    alias_applied: bool

    def as_dict(self) -> dict[str, object]:
        payload = asdict(self)
        payload["declaration_paths"] = list(self.declaration_paths)
        return payload


@dataclass(frozen=True)
class DependencyReconciliation:
    resolved_external: tuple[ReconciledDependency, ...]
    declaration_count: int
    candidate_count: int
    unresolved_candidate_count: int
    issues: tuple[str, ...]
    status: str = "RAN"
    reason: str | None = None

    @classmethod
    def empty(cls) -> "DependencyReconciliation":
        return cls((), 0, 0, 0, (), status="BLOCKED", reason="COMPONENT_OWNERSHIP_REQUIRED")

    @property
    def resolved_evidence_ids(self) -> frozenset[str]:
        return frozenset(item.evidence_id for item in self.resolved_external)

    def as_dict(self) -> dict[str, object]:
        alias_count = sum(1 for item in self.resolved_external if item.alias_applied)
        basis_counts: dict[str, int] = {}
        for item in self.resolved_external:
            basis_counts[item.basis] = basis_counts.get(item.basis, 0) + 1
        return {
            "status": self.status,
            "reason": self.reason,
            "summary": {
                "declaration_count": self.declaration_count,
                "candidate_count": self.candidate_count,
                "resolved_external_count": len(self.resolved_external),
                "unresolved_candidate_count": self.unresolved_candidate_count,
                "alias_resolved_count": alias_count,
                "basis_counts": dict(sorted(basis_counts.items())),
                "issue_count": len(self.issues),
            },
            "resolved_external": [item.as_dict() for item in self.resolved_external],
            "issues": list(self.issues),
        }


def _component_metadata(components: list[dict[str, object]]) -> dict[str, dict[str, object]]:
    return {
        str(item["id"]): item
        for item in components
        if isinstance(item.get("id"), str)
    }


def _read_declarations(
    root: Path,
    partition: ComponentPartition,
) -> tuple[list[DependencyDeclaration], list[str]]:
    declarations: list[DependencyDeclaration] = []
    issues: list[str] = []
    for assignment in partition.assignments:
        rel = assignment.path
        if not _is_requirement_manifest(rel):
            continue
        path = root / rel
        try:
            lines = path.read_text(encoding="utf-8-sig").splitlines()
        except (OSError, UnicodeError) as exc:
            issues.append(f"{rel}: {exc}")
            continue
        runtime_manifest = "runtime" in Path(rel).name.lower()
        for line in lines:
            stripped = line.strip()
            if not stripped or stripped.startswith(
                ("#", "-r", "--requirement", "-c", "--constraint", "-e", "--editable")
            ):
                continue
            match = _REQUIREMENT_NAME_RE.match(stripped)
            if not match:
                continue
            declarations.append(
                DependencyDeclaration(
                    component_id=assignment.component_id,
                    path=rel,
                    distribution=_normalize_name(match.group(1)),
                    runtime_manifest=runtime_manifest,
                )
            )
    return declarations, issues


def _is_product_runtime_component(component: dict[str, object] | None) -> bool:
    return bool(
        component
        and component.get("classification") == "PRODUCT"
        and component.get("runtime_required") is True
    )


def reconcile_dependency_evidence(
    root: str | Path,
    dependencies: DependencyScan,
    components: list[dict[str, object]],
    partition: ComponentPartition,
) -> DependencyReconciliation:
    root = Path(root).resolve()
    owners = {assignment.path: assignment.component_id for assignment in partition.assignments}
    metadata = _component_metadata(components)
    declarations, issues = _read_declarations(root, partition)

    resolved: list[ReconciledDependency] = []
    candidate_count = 0

    for edge in dependencies.edges:
        if edge.adapter != "python" or edge.resolution != ResolutionStatus.UNRESOLVED:
            continue
        if not edge.target or edge.target.startswith(".") or edge.target.startswith("<"):
            continue

        source_component = owners.get(edge.source)
        source_meta = metadata.get(source_component or "")
        if not source_component or source_meta is None:
            continue

        candidate_count += 1
        import_root = _normalize_name(edge.target.split(".", 1)[0])
        distribution = _IMPORT_DISTRIBUTION_ALIASES.get(import_root, import_root)

        same_component = [
            item
            for item in declarations
            if item.component_id == source_component and item.distribution == distribution
        ]
        product_runtime = [
            item
            for item in declarations
            if item.distribution == distribution
            and item.runtime_manifest
            and _is_product_runtime_component(source_meta)
            and _is_product_runtime_component(metadata.get(item.component_id))
        ]

        matches = same_component or product_runtime
        if not matches:
            continue

        resolved.append(
            ReconciledDependency(
                evidence_id=edge.evidence_id,
                source_component=source_component,
                import_name=edge.target,
                distribution=distribution,
                declaration_paths=tuple(sorted({item.path for item in matches})),
                basis=(
                    "SOURCE_COMPONENT_DECLARATION"
                    if same_component
                    else "PRODUCT_RUNTIME_MANIFEST"
                ),
                alias_applied=distribution != import_root,
            )
        )

    resolved.sort(key=lambda item: item.evidence_id)
    return DependencyReconciliation(
        resolved_external=tuple(resolved),
        declaration_count=len(declarations),
        candidate_count=candidate_count,
        unresolved_candidate_count=max(0, candidate_count - len(resolved)),
        issues=tuple(sorted(issues)),
    )
