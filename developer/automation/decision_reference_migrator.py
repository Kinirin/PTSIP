from __future__ import annotations

import re
import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Mapping

from developer.automation.policy_loader import load_yaml, repository_root


ADR_TOKEN = re.compile(r"\bADR-[0-9]{4}\b")
ADR_PATH = re.compile(r"decisions/(ADR-[0-9]{4})-[A-Za-z0-9_.-]+\.(?:md|yaml)")
ROUTING = "developer/policy/legacy-decision-reference-routing.yaml"


class UnroutedDecisionReferenceError(ValueError):
    pass


@dataclass(frozen=True)
class ProjectedEdge:
    source: str
    relation: str
    target: str
    scope: str | None = None


def load_routing(root: str | Path | None = None) -> dict[str, object]:
    return load_yaml(ROUTING, root=repository_root(root))


def canonical_targets(adr_id: str, *, root: str | Path | None = None) -> tuple[str, ...]:
    routing = load_routing(root)
    route = routing.get("id_routes", {}).get(adr_id)
    if not isinstance(route, Mapping):
        raise UnroutedDecisionReferenceError(f"No canonical target route for {adr_id}")
    targets = route.get("targets")
    if not isinstance(targets, list) or not targets:
        raise UnroutedDecisionReferenceError(f"Invalid canonical target route for {adr_id}")
    return tuple(str(item) for item in targets)


def canonical_policy_path(policy_id: str) -> str:
    if policy_id.startswith("SFP-"):
        return f"src/ptsip/specdata/{policy_id}.yaml"
    if policy_id.startswith("MPD-"):
        return f"developer/policy/{policy_id}.yaml"
    raise UnroutedDecisionReferenceError(f"Unsupported policy ID {policy_id}")


def project_relation(
    source_adr: str,
    relation: str,
    target_adr: str,
    *,
    scope: str | None = None,
    root: str | Path | None = None,
) -> tuple[ProjectedEdge, ...]:
    routing = load_routing(root)
    source_targets = canonical_targets(source_adr, root=root)
    target_targets = canonical_targets(target_adr, root=root)

    if len(source_targets) == 1 and len(target_targets) == 1:
        return (ProjectedEdge(source_targets[0], relation, target_targets[0], scope),)

    for item in routing.get("relation_routes", []):
        if not isinstance(item, Mapping):
            continue
        if (
            item.get("source_adr") == source_adr
            and item.get("relation") == relation
            and item.get("target_adr") == target_adr
            and item.get("scope") == scope
        ):
            edges = item.get("projected_edges", [])
            return tuple(
                ProjectedEdge(str(edge["source"]), relation, str(edge["target"]), scope)
                for edge in edges
                if isinstance(edge, Mapping)
            )
    raise UnroutedDecisionReferenceError(
        f"Split relation requires an explicit route: {source_adr} {relation} {target_adr} scope={scope!r}"
    )


def rewrite_textual_reference(text: str, *, root: str | Path | None = None) -> str:
    routing = load_routing(root)
    routes = routing.get("id_routes", {})

    def path_replacement(match: re.Match[str]) -> str:
        adr_id = match.group(1)
        route = routes.get(adr_id)
        if not isinstance(route, Mapping):
            raise UnroutedDecisionReferenceError(f"No textual route for {adr_id}")
        targets = [str(item) for item in route["targets"]]
        return " + ".join(canonical_policy_path(item) for item in targets)

    def token_replacement(match: re.Match[str]) -> str:
        adr_id = match.group(0)
        route = routes.get(adr_id)
        if not isinstance(route, Mapping):
            raise UnroutedDecisionReferenceError(f"No textual route for {adr_id}")
        return str(route["textual_lineage"])

    text = ADR_PATH.sub(path_replacement, text)
    return ADR_TOKEN.sub(token_replacement, text)


def tracked_files(root: str | Path | None = None) -> tuple[Path, ...]:
    base = repository_root(root)
    result = subprocess.run(
        ["git", "ls-files"],
        cwd=base,
        check=True,
        capture_output=True,
        text=True,
    )
    return tuple(base / line for line in result.stdout.splitlines())


def tracked_textual_reference_files(root: str | Path | None = None) -> tuple[Path, ...]:
    base = repository_root(root)
    allowed = {".md", ".txt", ".rst"}
    return tuple(
        path
        for path in tracked_files(base)
        if path.suffix.lower() in allowed
        and not path.relative_to(base).as_posix().startswith("decisions/")
    )


def scan_textual_references(root: str | Path | None = None) -> dict[str, tuple[str, ...]]:
    base = repository_root(root)
    found: dict[str, tuple[str, ...]] = {}
    for path in tracked_textual_reference_files(base):
        text = path.read_text(encoding="utf-8", errors="strict")
        refs = tuple(sorted(set(ADR_TOKEN.findall(text))))
        if refs:
            found[path.relative_to(base).as_posix()] = refs
    return found


def scan_machine_references(root: str | Path | None = None) -> dict[str, tuple[str, ...]]:
    """Report ADR tokens in tracked non-prose files without rewriting them.

    Machine-bearing formats are intentionally not rewritten by generic textual
    lineage rules because an ADR token may carry authority/dependency meaning.
    Those references must be handled by a dedicated materializer or exact route.
    """

    base = repository_root(root)
    prose = {".md", ".txt", ".rst"}
    skip_prefixes = ("decisions/",)
    found: dict[str, tuple[str, ...]] = {}
    for path in tracked_files(base):
        relative = path.relative_to(base).as_posix()
        if relative.startswith(skip_prefixes) or path.suffix.lower() in prose:
            continue
        try:
            text = path.read_text(encoding="utf-8", errors="strict")
        except (UnicodeDecodeError, OSError):
            continue
        refs = tuple(sorted(set(ADR_TOKEN.findall(text))))
        if refs:
            found[relative] = refs
    return found


def rewrite_textual_files(*, apply: bool, root: str | Path | None = None) -> dict[str, tuple[str, ...]]:
    base = repository_root(root)
    changed: dict[str, tuple[str, ...]] = {}
    for relative, refs in scan_textual_references(base).items():
        path = base / relative
        original = path.read_text(encoding="utf-8")
        rewritten = rewrite_textual_reference(original, root=base)
        if rewritten != original:
            changed[relative] = refs
            if apply:
                path.write_text(rewritten, encoding="utf-8")
    return changed


if __name__ == "__main__":
    pending = rewrite_textual_files(apply=False)
    for path, refs in sorted(pending.items()):
        print(f"TEXT {path}: {', '.join(refs)}")
    machine = scan_machine_references()
    for path, refs in sorted(machine.items()):
        print(f"MACHINE {path}: {', '.join(refs)}")
    print(f"Textual reference files pending deterministic rewrite: {len(pending)}")
    print(f"Machine-bearing files requiring explicit migration review: {len(machine)}")
