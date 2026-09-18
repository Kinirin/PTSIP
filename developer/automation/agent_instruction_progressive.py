from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from typing import Mapping

import yaml

from developer.automation.agent_instruction_classifier import LEVEL1
from developer.automation.agent_instruction_materializer import DEFAULT_OUTPUT_ROOT

LEVEL1_STAGE_REF = "stages/level1.json"
LEVEL1_UNRESOLVED_REF = "unresolved/level1.json"
SCHEMA = "ptsip-agent-progressive-reasoning/v1"

TRIGGER = re.compile(r"^\s*(before|after|when|whenever|if|unless|while|during|for|on)\b", re.I)
MODAL_PATTERNS = (
    ("PROHIBITION", re.compile(r"\b(?:must\s+not|shall\s+not|do\s+not|don't|never|cannot|can't|may\s+not|forbidden|prohibited)\b", re.I)),
    ("REQUIREMENT", re.compile(r"\b(?:must|shall|required|always|ensure)\b", re.I)),
    ("PREFERENCE", re.compile(r"\b(?:should|prefer|avoid)\b", re.I)),
    ("PERMISSION", re.compile(r"\b(?:may|can|allowed)\b", re.I)),
)
ACTION_WORDS = (
    "resolve", "read", "load", "scan", "run", "use", "check", "prepare", "select", "inspect", "create",
    "follow", "record", "evaluate", "compare", "apply", "verify", "build", "publish", "deploy", "invoke",
    "write", "edit", "generate", "remove", "update", "add", "install", "format", "lint", "test", "execute",
    "commit", "push", "open", "review", "confirm", "report", "stop", "keep", "preserve", "reject", "infer",
    "choose", "enter", "replace", "derive", "describe", "claim", "combine", "discover", "re-run", "rerun",
)
ACTION_RE = re.compile(r"\b(" + "|".join(re.escape(x) for x in sorted(ACTION_WORDS, key=len, reverse=True)) + r")\b", re.I)
BACKTICK = re.compile(r"\x60([^\x60]+)\x60")
COMMAND = re.compile(r"\b(?:python(?:\s+-m)?|pytest|git|pip|uv|npm|pnpm|yarn|cargo|mvn|gradle|make|cmake|ruff|mypy|pyright|twine)\b", re.I)


class ProgressiveReasoningError(RuntimeError):
    pass


def bootstrap_text() -> str:
    return """# AGENTS.md

This repository uses progressive repository-local agent instruction reasoning under .agent/.

Before repository work, classify the current operation as READ, MODIFY, PLAN, VERIFY, or RELEASE, then resolve the bounded instruction set mechanically:

    python -m developer.automation.agent_instruction_entry_resolver <READ|MODIFY|PLAN|VERIFY|RELEASE>

The resolver consumes the deepest proven machine stage for each atom plus only the natural-language residual that has not yet been mechanized. It must not re-run a previous passed classification level.

Level 1 UNRESOLVED items remain natural language at Level 1 until positively resolved or the Level 1 taxonomy is extended. They are not coerced to OTHER. Level 2 may advance only the Level 1 PASS subset; Level 1 UNRESOLVED items do not block unrelated PASS atoms and are not consumed by Level 2.

OTHER is intentionally excluded from normal Level 1 entry. Widen only when context or goal material is required:

    python -m developer.automation.agent_instruction_entry_resolver <OPERATION> --include OTHER

Do not use provenance/history as a reasoning input. Re-run the entry resolver whenever the operation changes. Resolver failure is fail-closed for repository instruction loading.
"""


def _load_yaml(path: Path, label: str) -> dict[str, object]:
    if not path.is_file():
        raise ProgressiveReasoningError(f"{label} missing: {path}")
    value = yaml.safe_load(path.read_text(encoding="utf-8"))
    if not isinstance(value, Mapping):
        raise ProgressiveReasoningError(f"{label} must be a mapping")
    return dict(value)


def _dump_yaml(value: Mapping[str, object]) -> str:
    return yaml.safe_dump(dict(value), sort_keys=False, allow_unicode=True, width=120)


def _dump_json(value: Mapping[str, object]) -> str:
    return json.dumps(dict(value), ensure_ascii=False, indent=2) + "\n"


def _atom_text(atom: Mapping[str, object]) -> str:
    for key in ("instruction_text", "normalized_text"):
        value = atom.get(key)
        if isinstance(value, str) and value.strip():
            return " ".join(value.split())
    source = atom.get("source")
    if isinstance(source, Mapping):
        value = source.get("raw_excerpt")
        if isinstance(value, str) and value.strip():
            return " ".join(value.split())
    raise ProgressiveReasoningError(f"atom {atom.get('atom_id')} has no natural-language source")


def _special_operations(text: str) -> list[str]:
    lowered = text.lower()
    if (
        "before broadly reading" in lowered
        and "resolve" in lowered
        and "policy" in lowered
        and "context mechanically" in lowered
    ):
        return ["AUTOMATED_REASONING_DURING_REPOSITORY_DOCUMENT_RETRIEVAL"]
    return []


def _mechanize(text: str, labels: list[str]) -> tuple[dict[str, object], list[dict[str, str]]]:
    normalized = " ".join(text.split())
    machine: dict[str, object] = {"level_1_labels": list(labels)}
    residual_work = normalized

    special = _special_operations(normalized)
    if special:
        machine["operations"] = special
        return machine, []

    trigger = TRIGGER.search(normalized)
    if trigger:
        machine["trigger"] = trigger.group(1).upper()
        residual_work = TRIGGER.sub("", residual_work, count=1).strip(" ,:;-")

    for name, pattern in MODAL_PATTERNS:
        if pattern.search(normalized):
            machine["modality"] = name
            residual_work = pattern.sub("", residual_work)
            break

    refs: list[str] = []
    for value in BACKTICK.findall(normalized):
        if value not in refs:
            refs.append(value)
    if refs:
        machine["references"] = refs
        for ref in refs:
            residual_work = residual_work.replace(f"\x60{ref}\x60", " ")

    operations: list[str] = []
    for match in ACTION_RE.finditer(normalized):
        op = match.group(1).replace("-", "_").upper()
        if op == "RERUN":
            op = "RE_RUN"
        if op not in operations:
            operations.append(op)
    if operations:
        machine["operations"] = operations
        residual_work = ACTION_RE.sub("", residual_work)

    if COMMAND.search(normalized):
        machine["command_semantics"] = "COMMAND_OR_TOOL_INVOCATION"

    residual_work = re.sub(r"\s+", " ", residual_work)
    residual_work = re.sub(r"\s+([,.;:])", r"\1", residual_work).strip(" ,:;-.")
    residual: list[dict[str, str]] = []
    if residual_work:
        residual.append({"role": "UNMECHANIZED_ARGUMENT", "text": residual_work})
    return machine, residual


def build_level1(root: Path) -> tuple[dict[str, object], dict[str, object], dict[str, object]]:
    agent = root / DEFAULT_OUTPUT_ROOT
    index = _load_yaml(agent / "index.yaml", "index")
    registry = _load_yaml(agent / "registry.yaml", "registry")
    atoms = registry.get("atoms")
    if not isinstance(atoms, list):
        raise ProgressiveReasoningError("registry.atoms must be a list")

    passed: list[dict[str, object]] = []
    unresolved: list[dict[str, object]] = []
    for raw in atoms:
        if not isinstance(raw, Mapping):
            raise ProgressiveReasoningError("invalid registry atom")
        atom = dict(raw)
        atom_id = atom.get("atom_id")
        labels = atom.get("level_1")
        if not isinstance(atom_id, str) or not isinstance(labels, list):
            raise ProgressiveReasoningError("invalid registry atom identity")
        text = _atom_text(atom)
        if atom.get("unresolved") is True or not labels:
            source = atom.get("source")
            heading = list(source.get("heading_path", [])) if isinstance(source, Mapping) else []
            unresolved.append(
                {
                    "atom_id": atom_id,
                    "status": "UNRESOLVED",
                    "heading_path": heading,
                    "natural_language": text,
                }
            )
            continue
        if any(label not in LEVEL1 for label in labels):
            raise ProgressiveReasoningError(f"invalid Level 1 label on {atom_id}")
        machine, residual = _mechanize(text, list(labels))
        passed.append(
            {
                "atom_id": atom_id,
                "pass_header": {"level": 1, "status": "PASS", "labels": list(labels)},
                "machine": machine,
                "natural_residual": residual,
            }
        )

    stage = {
        "schema_version": SCHEMA,
        "level": 1,
        "status": "PASS_WITH_UNRESOLVED" if unresolved else "PASS",
        "input_contract": "SOURCE_NATURAL_LANGUAGE",
        "next_level_input_contract": "CURRENT_LEVEL_PASS_SUBSET_ONLY",
        "next_level_payload": "PASS_HEADER_PLUS_MACHINE_FIELDS_PLUS_NATURAL_RESIDUAL",
        "rerun_previous_level_for_next_level": False,
        "unresolved_advances_to_next_level": False,
        "unresolved_blocks_passed_atoms": False,
        "pass_count": len(passed),
        "unresolved_count": len(unresolved),
        "pass": passed,
    }
    unresolved_doc = {
        "schema_version": SCHEMA,
        "level": 1,
        "routing_state": "UNRESOLVED",
        "count": len(unresolved),
        "items": unresolved,
    }
    return stage, unresolved_doc, {"index": index, "registry": registry}


def migrate_level1(repository: str | Path) -> dict[str, object]:
    root = Path(repository).resolve()
    agent = root / DEFAULT_OUTPUT_ROOT
    stage, unresolved, state = build_level1(root)
    index = state["index"]
    registry = state["registry"]

    stage_path = agent / LEVEL1_STAGE_REF
    unresolved_path = agent / LEVEL1_UNRESOLVED_REF
    stage_path.parent.mkdir(parents=True, exist_ok=True)
    unresolved_path.parent.mkdir(parents=True, exist_ok=True)
    stage_path.write_text(_dump_json(stage), encoding="utf-8", newline="\n")
    unresolved_path.write_text(_dump_json(unresolved), encoding="utf-8", newline="\n")

    atoms = registry.get("atoms")
    assert isinstance(atoms, list)
    for raw in atoms:
        if not isinstance(raw, dict):
            continue
        raw.pop("instruction_text", None)
        raw.pop("normalized_text", None)
        source = raw.get("source")
        if isinstance(source, dict):
            source.pop("raw_excerpt", None)
    registry.pop("provenance_ref", None)
    registry["management_mode"] = "PROGRESSIVE_LEVEL_1"
    registry["reasoning_payload"] = "STAGED_ONLY"

    index.pop("provenance_ref", None)
    index["management_mode"] = "PROGRESSIVE_LEVEL_1"
    index["progressive_reasoning"] = {
        "highest_materialized_level": 1,
        "per_atom_advancement": True,
        "level_1_ref": LEVEL1_STAGE_REF,
        "level_1_unresolved_ref": LEVEL1_UNRESOLVED_REF,
        "previous_level_rerun_forbidden": True,
        "provenance_is_reasoning_input": False,
        "next_level_candidate_count": stage["pass_count"],
        "unresolved_blocks_next_level_candidates": False,
    }
    index["authority_refs"] = {
        "level_1_pass": LEVEL1_STAGE_REF,
        "level_1_unresolved": LEVEL1_UNRESOLVED_REF,
    }
    index.pop("authority_ref", None)

    (agent / "registry.yaml").write_text(_dump_yaml(registry), encoding="utf-8", newline="\n")
    (agent / "index.yaml").write_text(_dump_yaml(index), encoding="utf-8", newline="\n")

    provenance = agent / "provenance" / "AGENTS.pre-level1.md"
    if provenance.is_file():
        provenance.unlink()
        try:
            provenance.parent.rmdir()
        except OSError:
            pass

    (root / "AGENTS.md").write_text(
        bootstrap_text(),
        encoding="utf-8",
        newline="\n",
    )

    return {
        "level": 1,
        "pass_count": stage["pass_count"],
        "unresolved_count": stage["unresolved_count"],
        "next_level_candidate_count": stage["pass_count"],
        "unresolved_blocks_next_level_candidates": False,
    }


def check(repository: str | Path) -> tuple[str, ...]:
    root = Path(repository).resolve()
    agent = root / DEFAULT_OUTPUT_ROOT
    errors: list[str] = []
    index = _load_yaml(agent / "index.yaml", "index")
    registry = _load_yaml(agent / "registry.yaml", "registry")
    progressive = index.get("progressive_reasoning")
    if (
        not isinstance(progressive, Mapping)
        or progressive.get("highest_materialized_level") != 1
        or progressive.get("per_atom_advancement") is not True
    ):
        return ("LEVEL_1_PROGRESSIVE_MODE_NOT_ACTIVE",)

    stage_path = agent / str(progressive.get("level_1_ref", ""))
    unresolved_path = agent / str(progressive.get("level_1_unresolved_ref", ""))
    try:
        stage = json.loads(stage_path.read_text(encoding="utf-8"))
        unresolved = json.loads(unresolved_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return ("LEVEL_1_STAGE_INVALID",)

    if stage.get("level") != 1 or unresolved.get("level") != 1:
        errors.append("LEVEL_1_STAGE_MISMATCH")
    if stage.get("pass_count") != len(stage.get("pass", [])):
        errors.append("LEVEL_1_PASS_COUNT_MISMATCH")
    if unresolved.get("count") != len(unresolved.get("items", [])):
        errors.append("LEVEL_1_UNRESOLVED_COUNT_MISMATCH")
    if progressive.get("next_level_candidate_count") != stage.get("pass_count"):
        errors.append("NEXT_LEVEL_CANDIDATE_COUNT_MISMATCH")
    if progressive.get("unresolved_blocks_next_level_candidates") is not False:
        errors.append("UNRESOLVED_MUST_NOT_BLOCK_PASSED_ATOMS")

    atoms = registry.get("atoms")
    if isinstance(atoms, list):
        for atom in atoms:
            if not isinstance(atom, Mapping):
                continue
            source = atom.get("source")
            if atom.get("instruction_text") is not None or atom.get("normalized_text") is not None:
                errors.append("REGISTRY_CONTAINS_NATURAL_LANGUAGE_PAYLOAD")
                break
            if isinstance(source, Mapping) and source.get("raw_excerpt") is not None:
                errors.append("REGISTRY_CONTAINS_NATURAL_LANGUAGE_PAYLOAD")
                break

    if (agent / "provenance" / "AGENTS.pre-level1.md").exists():
        errors.append("PROVENANCE_MD_MUST_NOT_BE_REASONING_SURFACE")
    agents = root / "AGENTS.md"
    if not agents.is_file() or agents.read_text(encoding="utf-8") != bootstrap_text():
        errors.append("AGENTS_BOOTSTRAP_STALE")
    return tuple(errors)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Progressive agent instruction reasoning stages")
    parser.add_argument("command", choices=("migrate-level1", "check"))
    parser.add_argument("--repository", default=".")
    args = parser.parse_args(argv)
    try:
        if args.command == "migrate-level1":
            result = migrate_level1(args.repository)
            print(
                f"Progressive Level 1: {result['pass_count']} pass, "
                f"{result['unresolved_count']} unresolved; "
                f"{result['next_level_candidate_count']} Level 2 candidate(s)"
            )
            return 0
        errors = check(args.repository)
        if errors:
            print("Progressive Level 1: STALE")
            for error in errors:
                print(error)
            return 1
        print("Progressive Level 1: CURRENT")
        return 0
    except (ProgressiveReasoningError, OSError, yaml.YAMLError) as exc:
        print(f"agent-instruction-progressive: {exc}")
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
