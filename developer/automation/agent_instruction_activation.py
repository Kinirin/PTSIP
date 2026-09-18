from __future__ import annotations

import argparse
import hashlib
from pathlib import Path
from typing import Mapping

import yaml

from developer.automation.agent_instruction_classifier import LEVEL1
from developer.automation.agent_instruction_materializer import DEFAULT_OUTPUT_ROOT, DEFAULT_SOURCE, check_materialization

MODE = "MANAGED_LEVEL_1"
PROVENANCE_REF = "provenance/AGENTS.pre-level1.md"


class AgentInstructionActivationError(RuntimeError):
    pass


def _load(path: Path, label: str) -> dict[str, object]:
    if not path.is_file():
        raise AgentInstructionActivationError(f"{label} missing: {path}")
    value = yaml.safe_load(path.read_text(encoding="utf-8"))
    if not isinstance(value, Mapping):
        raise AgentInstructionActivationError(f"{label} must be a mapping")
    return dict(value)


def _dump(value: Mapping[str, object]) -> str:
    return yaml.safe_dump(dict(value), sort_keys=False, allow_unicode=True, width=120)


def _sha(value: str) -> str:
    return hashlib.sha256(value.encode("utf-8")).hexdigest()


def bootstrap_text() -> str:
    return """# AGENTS.md

This repository uses the repository-local Level 1 agent instruction surface under .agent/.

Before repository work, classify the current operation as READ, MODIFY, PLAN, VERIFY, or RELEASE, then resolve the bounded instruction set mechanically:

    python -m developer.automation.agent_instruction_entry_resolver <READ|MODIFY|PLAN|VERIFY|RELEASE>

Use both the returned managed Level 1 instructions and the returned UNRESOLVED instructions. UNRESOLVED remains an unclassified routing state and must not be coerced to OTHER.

OTHER is intentionally excluded from normal Level 1 entry. Widen only when context or goal material is required:

    python -m developer.automation.agent_instruction_entry_resolver <OPERATION> --include OTHER

Re-run the resolver whenever the operation changes. Do not scan .agent/registry.yaml or .agent/level1/ as the normal entry path. Resolver failure is fail-closed for repository instruction loading. Level 2 routing is not active.
"""


def activate(repository: str | Path) -> dict[str, object]:
    root = Path(repository).resolve()
    source = root / DEFAULT_SOURCE
    agent = (root / DEFAULT_OUTPUT_ROOT).resolve()
    if not source.is_file():
        raise AgentInstructionActivationError("AGENTS.md is missing")
    stale = check_materialization(DEFAULT_SOURCE, root=root, output_root=DEFAULT_OUTPUT_ROOT)
    if stale:
        raise AgentInstructionActivationError("materialization is stale: " + ", ".join(stale))

    index_path = agent / "index.yaml"
    registry_path = agent / "registry.yaml"
    index = _load(index_path, "index")
    registry = _load(registry_path, "registry")
    if index.get("management_mode") == MODE:
        raise AgentInstructionActivationError("Level 1 management is already active")
    if index.get("level_2_materialized") is not False or registry.get("level_2_materialized") is not False:
        raise AgentInstructionActivationError("Level 2 must remain unmaterialized")

    text = source.read_text(encoding="utf-8")
    migration_source = registry.pop("source", None)
    if not isinstance(migration_source, Mapping) or migration_source.get("sha256") != _sha(text):
        raise AgentInstructionActivationError("registry source digest does not match AGENTS.md")

    atoms = registry.get("atoms")
    if not isinstance(atoms, list):
        raise AgentInstructionActivationError("registry.atoms must be a list")
    managed = unresolved = 0
    for atom in atoms:
        if not isinstance(atom, dict):
            raise AgentInstructionActivationError("invalid registry atom")
        source_info = atom.get("source")
        if not isinstance(source_info, Mapping) or not isinstance(source_info.get("raw_excerpt"), str):
            raise AgentInstructionActivationError("source.raw_excerpt is required")
        atom["instruction_text"] = source_info["raw_excerpt"]
        if atom.get("unresolved") is True:
            atom["management_state"] = "UNRESOLVED"
            unresolved += 1
        else:
            labels = atom.get("level_1")
            if not isinstance(labels, list) or not labels:
                raise AgentInstructionActivationError("classified atom has no Level 1 label")
            atom["management_state"] = "LEVEL_1_MANAGED"
            managed += 1

    registry["management_mode"] = MODE
    registry["authority_mode"] = "REGISTRY_ATOM_INSTRUCTION_TEXT"
    registry["migration_source"] = dict(migration_source)
    registry["provenance_ref"] = PROVENANCE_REF
    index["management_mode"] = MODE
    index["authority_ref"] = "registry.yaml#atoms[].instruction_text"
    index["provenance_ref"] = PROVENANCE_REF
    index["entry_resolver"] = {
        "command": "python -m developer.automation.agent_instruction_entry_resolver <READ|MODIFY|PLAN|VERIFY|RELEASE>",
        "level_2_used": False,
        "unresolved_policy": "ALWAYS_INCLUDE_UNTIL_CLASSIFIED",
        "other_default": "EXPLICIT_WIDEN_ONLY",
    }

    provenance = agent / PROVENANCE_REF
    provenance.parent.mkdir(parents=True, exist_ok=True)
    provenance.write_text(text, encoding="utf-8", newline="\n")
    registry_path.write_text(_dump(registry), encoding="utf-8", newline="\n")
    index_path.write_text(_dump(index), encoding="utf-8", newline="\n")
    source.write_text(bootstrap_text(), encoding="utf-8", newline="\n")
    return {"managed": managed, "unresolved": unresolved}


def check_activation(repository: str | Path) -> tuple[str, ...]:
    root = Path(repository).resolve()
    agent = root / DEFAULT_OUTPUT_ROOT
    errors: list[str] = []
    index = _load(agent / "index.yaml", "index")
    registry = _load(agent / "registry.yaml", "registry")
    if index.get("management_mode") != MODE or registry.get("management_mode") != MODE:
        errors.append("MANAGEMENT_MODE_NOT_ACTIVE")
    if index.get("level_2_materialized") is not False or registry.get("level_2_materialized") is not False:
        errors.append("LEVEL_2_MUST_NOT_BE_MATERIALIZED")
    if (root / DEFAULT_SOURCE).read_text(encoding="utf-8") != bootstrap_text():
        errors.append("AGENTS_BOOTSTRAP_STALE")

    migration = registry.get("migration_source")
    provenance = agent / str(index.get("provenance_ref", ""))
    if not isinstance(migration, Mapping) or not provenance.is_file() or migration.get("sha256") != _sha(provenance.read_text(encoding="utf-8")):
        errors.append("PROVENANCE_INVALID")

    atoms = registry.get("atoms")
    if not isinstance(atoms, list):
        return tuple(errors + ["REGISTRY_ATOMS_INVALID"])
    by_id = {a.get("atom_id"): a for a in atoms if isinstance(a, Mapping) and isinstance(a.get("atom_id"), str)}
    refs = index.get("level_1_projections")
    if not isinstance(refs, Mapping):
        errors.append("LEVEL_1_PROJECTIONS_INVALID")
    else:
        for label in LEVEL1:
            payload = _load(agent / str(refs.get(label, "")), f"projection {label}")
            for atom_id in payload.get("atom_ids", []):
                atom = by_id.get(atom_id)
                if atom is None or label not in atom.get("level_1", []) or atom.get("unresolved") is True:
                    errors.append(f"PROJECTION_MISMATCH:{label}:{atom_id}")
    unresolved = _load(agent / str(index.get("unresolved_ref", "")), "unresolved")
    for atom_id in unresolved.get("atom_ids", []):
        atom = by_id.get(atom_id)
        if atom is None or atom.get("unresolved") is not True:
            errors.append(f"UNRESOLVED_MISMATCH:{atom_id}")
    return tuple(errors)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Activate Level 1 repository agent instruction management")
    parser.add_argument("command", choices=("activate", "check"))
    parser.add_argument("--repository", default=".")
    args = parser.parse_args(argv)
    try:
        if args.command == "activate":
            result = activate(args.repository)
            print(f"Agent instruction Level 1 activated: {result['managed']} managed, {result['unresolved']} unresolved; Level 2 not materialized")
            return 0
        errors = check_activation(args.repository)
        if errors:
            print("Agent instruction Level 1 activation: STALE")
            for error in errors:
                print(error)
            return 1
        print("Agent instruction Level 1 activation: CURRENT")
        return 0
    except (AgentInstructionActivationError, OSError, yaml.YAMLError) as exc:
        print(f"agent-instruction-activation: {exc}")
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
