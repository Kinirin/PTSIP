from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Iterable, Mapping

import yaml

from developer.automation.agent_instruction_classifier import LEVEL1, load_trial_policy
from developer.automation.agent_instruction_materializer import DEFAULT_OUTPUT_ROOT





class AgentInstructionEntryResolverError(RuntimeError):
    pass



def _routing(root: Path) -> tuple[dict[str, tuple[str, ...]], tuple[str, ...]]:
    trial = load_trial_policy(root)
    entry = trial.get("level_1_entry_resolution")
    if not isinstance(entry, Mapping):
        raise AgentInstructionEntryResolverError("level_1_entry_resolution policy missing")
    raw_routes = entry.get("default_routes")
    widen = entry.get("explicit_widen_allowed")
    if not isinstance(raw_routes, Mapping) or not isinstance(widen, list):
        raise AgentInstructionEntryResolverError("Level 1 entry routing policy is invalid")
    routes: dict[str, tuple[str, ...]] = {}
    for operation, raw_labels in raw_routes.items():
        if not isinstance(operation, str) or not isinstance(raw_labels, list):
            raise AgentInstructionEntryResolverError("Level 1 route is invalid")
        labels = tuple(raw_labels)
        if not labels or any(label not in LEVEL1 for label in labels):
            raise AgentInstructionEntryResolverError(f"invalid Level 1 route: {operation}")
        routes[operation] = labels
    if any(not isinstance(label, str) or label not in LEVEL1 for label in widen):
        raise AgentInstructionEntryResolverError("explicit widen vocabulary is invalid")
    return routes, tuple(widen)

def _load(path: Path, label: str) -> Mapping[str, object]:
    if not path.is_file():
        raise AgentInstructionEntryResolverError(f"{label} missing: {path}")
    value = yaml.safe_load(path.read_text(encoding="utf-8"))
    if not isinstance(value, Mapping):
        raise AgentInstructionEntryResolverError(f"{label} must be a mapping")
    return value


def _load_json(path: Path, label: str) -> Mapping[str, object]:
    if not path.is_file():
        raise AgentInstructionEntryResolverError(f"{label} missing: {path}")
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise AgentInstructionEntryResolverError(f"{label} invalid JSON") from exc
    if not isinstance(value, Mapping):
        raise AgentInstructionEntryResolverError(f"{label} must be an object")
    return value


def _ref(root: Path, base: Path, value: object, label: str) -> Path:
    if not isinstance(value, str) or not value:
        raise AgentInstructionEntryResolverError(f"{label} must be a path")
    path = (base / value).resolve()
    try:
        path.relative_to(root)
    except ValueError as exc:
        raise AgentInstructionEntryResolverError(f"{label} escapes repository") from exc
    return path


def _ids(root: Path, agent: Path, index: Mapping[str, object], label: str) -> tuple[str, ...]:
    refs = index.get("level_1_projections")
    if not isinstance(refs, Mapping):
        raise AgentInstructionEntryResolverError("level_1_projections missing")
    payload = _load(_ref(root, agent, refs.get(label), label), label)
    if payload.get("level_1") != label:
        raise AgentInstructionEntryResolverError(f"projection mismatch: {label}")
    ids = payload.get("atom_ids")
    if not isinstance(ids, list) or any(not isinstance(x, str) for x in ids):
        raise AgentInstructionEntryResolverError(f"invalid atom ids: {label}")
    return tuple(ids)


def _compact(atom: Mapping[str, object]) -> dict[str, object]:
    atom_id = atom.get("atom_id")
    labels = atom.get("level_1")
    text = atom.get("instruction_text", atom.get("normalized_text"))
    if not isinstance(atom_id, str) or not isinstance(labels, list) or not isinstance(text, str):
        raise AgentInstructionEntryResolverError("invalid registry atom")
    return {"atom_id": atom_id, "level_1": list(labels), "text": text.rstrip("\r\n")}


def _resolve_progressive(
    root: Path,
    agent: Path,
    index: Mapping[str, object],
    operation: str,
    labels: list[str],
) -> dict[str, object]:
    progressive = index.get("progressive_reasoning")
    if (
        not isinstance(progressive, Mapping)
        or progressive.get("highest_materialized_level") != 1
        or progressive.get("per_atom_advancement") is not True
    ):
        raise AgentInstructionEntryResolverError("progressive Level 1 state is invalid")

    stage = _load_json(
        _ref(root, agent, progressive.get("level_1_ref"), "level_1_ref"),
        "Level 1 stage",
    )
    unresolved_doc = _load_json(
        _ref(
            root,
            agent,
            progressive.get("level_1_unresolved_ref"),
            "level_1_unresolved_ref",
        ),
        "Level 1 unresolved",
    )
    if stage.get("level") != 1 or unresolved_doc.get("level") != 1:
        raise AgentInstructionEntryResolverError("Level 1 staged artifact mismatch")

    raw_pass = stage.get("pass")
    raw_unresolved = unresolved_doc.get("items")
    if not isinstance(raw_pass, list) or not isinstance(raw_unresolved, list):
        raise AgentInstructionEntryResolverError("Level 1 staged collections invalid")

    instructions: list[dict[str, object]] = []
    seen: set[str] = set()
    for raw in raw_pass:
        if not isinstance(raw, Mapping):
            raise AgentInstructionEntryResolverError("invalid Level 1 pass item")
        atom_id = raw.get("atom_id")
        header = raw.get("pass_header")
        if not isinstance(atom_id, str) or not isinstance(header, Mapping):
            raise AgentInstructionEntryResolverError("invalid Level 1 pass identity")
        item_labels = header.get("labels")
        if not isinstance(item_labels, list) or any(x not in LEVEL1 for x in item_labels):
            raise AgentInstructionEntryResolverError(f"invalid pass labels: {atom_id}")
        if not set(item_labels).intersection(labels) or atom_id in seen:
            continue
        machine = raw.get("machine")
        residual = raw.get("natural_residual")
        if not isinstance(machine, Mapping) or not isinstance(residual, list):
            raise AgentInstructionEntryResolverError(f"invalid staged payload: {atom_id}")
        instructions.append(
            {
                "atom_id": atom_id,
                "pass_header": dict(header),
                "machine": dict(machine),
                "natural_residual": list(residual),
            }
        )
        seen.add(atom_id)

    unresolved: list[dict[str, object]] = []
    for raw in raw_unresolved:
        if (
            not isinstance(raw, Mapping)
            or not isinstance(raw.get("atom_id"), str)
            or not isinstance(raw.get("natural_language"), str)
        ):
            raise AgentInstructionEntryResolverError("invalid Level 1 unresolved item")
        unresolved.append(
            {
                "atom_id": raw["atom_id"],
                "status": "UNRESOLVED",
                "heading_path": list(raw.get("heading_path", [])),
                "natural_language": raw["natural_language"],
            }
        )

    return {
        "schema_version": "ptsip-agent-instruction-entry-resolution/v2",
        "stage": "PROGRESSIVE_LEVEL_1",
        "operation": operation,
        "selected_level_1": labels,
        "scope_filtering": "NOT_AVAILABLE_AT_LEVEL_1",
        "level_2_used": False,
        "previous_level_rerun": False,
        "unresolved_policy": "ALWAYS_INCLUDE_UNTIL_CLASSIFIED",
        "instruction_count": len(instructions),
        "unresolved_count": len(unresolved),
        "instructions": instructions,
        "unresolved": unresolved,
    }


def resolve_entry(
    repository: str | Path,
    *,
    operation: str,
    include: Iterable[str] = (),
    agent_root: Path = DEFAULT_OUTPUT_ROOT,
) -> dict[str, object]:
    root = Path(repository).resolve()
    operation = operation.upper()
    routes, widen_allowed = _routing(root)
    if operation not in routes:
        raise AgentInstructionEntryResolverError(f"unsupported operation: {operation}")
    agent = (root / agent_root).resolve()
    try:
        agent.relative_to(root)
    except ValueError as exc:
        raise AgentInstructionEntryResolverError("agent root escapes repository") from exc

    labels = list(routes[operation])
    for raw in include:
        label = raw.upper()
        if label not in widen_allowed:
            raise AgentInstructionEntryResolverError(f"unsupported Level 1 widen: {label}")
        if label not in labels:
            labels.append(label)

    index = _load(agent / "index.yaml", ".agent/index.yaml")
    if isinstance(index.get("progressive_reasoning"), Mapping):
        return _resolve_progressive(root, agent, index, operation, labels)

    registry = _load(_ref(root, agent, index.get("registry_ref"), "registry_ref"), "registry")
    if index.get("level_2_materialized") is not False or registry.get("level_2_materialized") is not False:
        raise AgentInstructionEntryResolverError("Level 2 routing is not active")

    raw_atoms = registry.get("atoms")
    if not isinstance(raw_atoms, list):
        raise AgentInstructionEntryResolverError("registry.atoms must be a list")
    ordered: list[Mapping[str, object]] = []
    by_id: dict[str, Mapping[str, object]] = {}
    for raw in raw_atoms:
        if not isinstance(raw, Mapping) or not isinstance(raw.get("atom_id"), str):
            raise AgentInstructionEntryResolverError("invalid registry atom")
        atom_id = raw["atom_id"]
        if atom_id in by_id:
            raise AgentInstructionEntryResolverError(f"duplicate atom id: {atom_id}")
        ordered.append(raw)
        by_id[atom_id] = raw

    selected: set[str] = set()
    for label in labels:
        selected.update(_ids(root, agent, index, label))

    unresolved_path = _ref(root, agent, index.get("unresolved_ref"), "unresolved_ref")
    unresolved_projection = _load(unresolved_path, "unresolved")
    if unresolved_projection.get("routing_state") != "UNRESOLVED":
        raise AgentInstructionEntryResolverError("unresolved routing state mismatch")
    unresolved_ids = unresolved_projection.get("atom_ids")
    if not isinstance(unresolved_ids, list) or any(not isinstance(x, str) for x in unresolved_ids):
        raise AgentInstructionEntryResolverError("invalid unresolved atom ids")

    unknown = (selected | set(unresolved_ids)) - set(by_id)
    if unknown:
        raise AgentInstructionEntryResolverError("unknown atom ids: " + ", ".join(sorted(unknown)))

    instructions = [_compact(atom) for atom in ordered if atom["atom_id"] in selected]
    unresolved = []
    for atom_id in unresolved_ids:
        item = _compact(by_id[atom_id])
        source = by_id[atom_id].get("source")
        item["heading_path"] = list(source.get("heading_path", [])) if isinstance(source, Mapping) else []
        unresolved.append(item)

    return {
        "schema_version": "ptsip-agent-instruction-entry-resolution/v1",
        "stage": "LEGACY_LEVEL_1",
        "operation": operation,
        "selected_level_1": labels,
        "scope_filtering": "NOT_AVAILABLE_AT_LEVEL_1",
        "level_2_used": False,
        "unresolved_policy": "ALWAYS_INCLUDE_UNTIL_CLASSIFIED",
        "instruction_count": len(instructions),
        "unresolved_count": len(unresolved),
        "instructions": instructions,
        "unresolved": unresolved,
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Resolve Level 1 repository agent instructions")
    parser.add_argument("operation")
    parser.add_argument("--include", action="append", default=[])
    parser.add_argument("--repository", default=".")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args(argv)
    try:
        result = resolve_entry(args.repository, operation=args.operation, include=args.include)
    except (AgentInstructionEntryResolverError, OSError, yaml.YAMLError) as exc:
        print(f"agent-instruction-entry-resolver: {exc}")
        return 2
    if args.json:
        print(json.dumps(result, ensure_ascii=True, indent=2))
        return 0
    print(f"Agent entry {result['operation']}: {result['instruction_count']} managed, {result['unresolved_count']} unresolved")
    print("Level 1:", ", ".join(result["selected_level_1"]))
    for item in result["instructions"]:
        if result["stage"] == "PROGRESSIVE_LEVEL_1":
            machine = json.dumps(item["machine"], ensure_ascii=True, separators=(",", ":"))
            residual = item["natural_residual"]
            suffix = f" residual={json.dumps(residual, ensure_ascii=True)}" if residual else ""
            print(f"[{item['atom_id']}|PASS] machine={machine}{suffix}")
        else:
            print(f"[{item['atom_id']}|{','.join(item['level_1'])}] {item['text']}")
    for item in result["unresolved"]:
        if result["stage"] == "PROGRESSIVE_LEVEL_1":
            heading = " > ".join(item["heading_path"])
            suffix = f"|{heading}" if heading else ""
            print(f"[{item['atom_id']}|UNRESOLVED{suffix}] {item['natural_language']}")
        else:
            heading = " > ".join(item["heading_path"])
            suffix = f"|{heading}" if heading else ""
            print(f"[{item['atom_id']}|UNRESOLVED{suffix}] {item['text']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
