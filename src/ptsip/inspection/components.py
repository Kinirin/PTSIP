from __future__ import annotations

import re
from dataclasses import asdict, dataclass
from pathlib import Path

from ..model import DecisionStatus
from .dependencies import DependencyScan
from .inventory import Inventory


@dataclass(frozen=True)
class ComponentCandidate:
    id: str
    include: tuple[str, ...]
    anchors: tuple[str, ...]
    evidence_ids: tuple[str, ...]
    decision_status: DecisionStatus = DecisionStatus.UNKNOWN
    classification: None = None

    def as_dict(self) -> dict[str, object]:
        payload = asdict(self)
        payload["decision_status"] = self.decision_status.value
        return payload


def _slug(value: str) -> str:
    text = re.sub(r"[^a-zA-Z0-9]+", "-", value).strip("-").lower()
    return text or "repository-root"


def discover_component_candidates(root: str | Path, inventory: Inventory, dependencies: DependencyScan) -> list[ComponentCandidate]:
    root = Path(root).resolve()
    raw: dict[str, dict[str, set[str]]] = {}

    def add(selector: str, anchor: str, evidence_id: str) -> None:
        item = raw.setdefault(selector, {"anchors": set(), "evidence_ids": set()})
        item["anchors"].add(anchor)
        item["evidence_ids"].add(evidence_id)

    for manifest in inventory.manifests:
        parent = Path(manifest).parent.as_posix()
        selector = manifest if parent in {".", ""} else f"{parent}/**"
        add(selector, "manifest", f"manifest:{manifest}")

    schema_parents: dict[str, int] = {}
    for schema in inventory.schema_candidates:
        parent = Path(schema).parent.as_posix()
        schema_parents[parent] = schema_parents.get(parent, 0) + 1
    for parent, count in schema_parents.items():
        selector = f"{parent}/**" if parent not in {".", ""} else "*.schema.*"
        add(selector, "contract/schema-group", f"schema-group:{parent}:{count}")

    for name in inventory.tool_like_roots:
        add(f"{name}/**", "top-level-tool-root", f"root:{name}")
    for name in inventory.test_roots:
        add(f"{name}/**", "top-level-test-root", f"root:{name}")

    for edge in dependencies.edges:
        if edge.adapter == "github-actions" and edge.resolved_path:
            add(edge.resolved_path, "ci-invoked-script", edge.evidence_id)

    candidates: list[ComponentCandidate] = []
    for selector, details in sorted(raw.items()):
        selector_path = selector.removesuffix("/**")
        candidates.append(
            ComponentCandidate(
                id=_slug(selector_path),
                include=(selector,),
                anchors=tuple(sorted(details["anchors"])),
                evidence_ids=tuple(sorted(details["evidence_ids"])),
            )
        )
    return candidates
