from __future__ import annotations

import argparse
import json
import re
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Iterable, Mapping

import yaml

POLICY_PATH = Path("developer/policy/MPD-0010.yaml")
POLICY_SECTION = "agent_instruction_entry_taxonomy_trial"
RESULT_SCHEMA = "ptsip-agent-instruction-classification-trial/v1"
LEVEL1 = ("APPLICABILITY", "RULE", "ACTION", "EVIDENCE", "OTHER")
LEVEL2 = {
    "APPLICABILITY": ("PATH", "TASK", "ENVIRONMENT", "VERSION", "VCS_CONTEXT"),
    "RULE": ("MUTATION", "DEPENDENCY", "COMPATIBILITY", "STYLE", "SECURITY", "RESOURCE"),
    "ACTION": ("COMMAND", "READ", "WRITE", "GENERATE", "BUILD", "INSTALL", "PUBLISH"),
    "EVIDENCE": ("TEST", "BUILD_RESULT", "STATIC_CHECK", "ARTIFACT", "REVIEW", "STATUS"),
    "OTHER": ("CONTEXT", "GOAL"),
}
INHERITABLE = frozenset({"APPLICABILITY", "RULE", "ACTION", "EVIDENCE"})

HEADING = re.compile(r"^(#{1,6})\s+(.*\S)\s*$")
LIST_ITEM = re.compile(r"^(?P<indent>[ \t]*)(?:[-+*]|\d+[.)])\s+(?P<text>\S.*)$")
CONDITION = re.compile(r"^(?:before|after|when|whenever|if|unless|while|during|for\b|on\b)|\b(?:only when|appl(?:y|ies) to|working on|changes? to)\b", re.I)
PATH_SCOPE = re.compile(r"\b(?:under|within|inside|outside|across|in)\s+`?[^\s`]+(?:/|\\)[^\s`]*`?", re.I)
RULE = re.compile(r"\b(?:must(?:\s+not)?|shall(?:\s+not)?|should(?:\s+not)?|do not|don't|never|always|only|cannot|can't|may not|required|forbidden|prohibited|avoid|prefer|ensure)\b", re.I)
ACTION = re.compile(r"(?:^|[,;:]\s+)(?:run|use|read|load|resolve|check|prepare|select|inspect|create|follow|record|evaluate|compare|apply|verify|build|publish|deploy|invoke|write|edit|generate|remove|update|add|install|format|lint|test|execute|commit|push|open|review|confirm|report|stop|keep|preserve|reject)\b", re.I)
COMMAND = re.compile(r"(?:^|[`\s])(?:python(?:\s+-m)?|pytest|git|pip|uv|npm|pnpm|yarn|cargo|go\s+(?:test|build)|mvn|gradle|make|cmake|ruff|mypy|pyright|twine|ptsip)(?:\s|`|$)", re.I)
EVIDENCE = re.compile(r"\b(?:verify|verification|validate|validation|evidence|test|tests|pytest|lint|typecheck|type check|artifact|review|status|pass|passed|fail|failed|success|exact[- ]sha|clean status)\b", re.I)
STATUS = re.compile(r"\b(?:status|pass|passed|fail|failed|success|successful|exact[- ]sha|commit status|clean status)\b", re.I)
GOAL = re.compile(r"\b(?:the\s+)?(?:goal|objective|purpose|intent)\s+(?:of\s+[^.]{1,80}\s+)?is\s+|\b(?:this|the\s+project|the\s+system|the\s+component)\s+(?:exists|is\s+designed|is\s+intended)\s+to\s+|\bwe\s+aim\s+to\s+", re.I)
CONTEXT = re.compile(r"\b(?:is|are|means|refers to|consists of|contains|includes|uses|belongs to|lives under|is based on|is built with)\b", re.I)
KEY_VALUE = re.compile(r"^[^:]{1,80}:\s+\S")
MAPPING = re.compile(r"\s(?:--|—|->|→)\s")
LANG_VERSION = re.compile(r"^(?:Python|Node(?:\.js)?|Java|Go|Rust|R)\s+\d+(?:\.\d+){0,3}\+?$", re.I)
URL = re.compile(r"^https?://\S+$", re.I)


class AgentInstructionClassifierError(RuntimeError):
    pass


@dataclass(frozen=True)
class InstructionAtom:
    atom_id: str
    kind: str
    text: str
    line_start: int
    line_end: int
    heading_path: tuple[str, ...]
    parent_atom_id: str | None


@dataclass(frozen=True)
class ClassifiedAtom:
    atom_id: str
    kind: str
    text: str
    line_start: int
    line_end: int
    heading_path: tuple[str, ...]
    parent_atom_id: str | None
    direct_level1: tuple[str, ...]
    inherited_level1: tuple[str, ...]
    level1: tuple[str, ...]
    level2: Mapping[str, tuple[str, ...]]
    unresolved: bool


def _mapping(value: object, label: str) -> Mapping[str, object]:
    if not isinstance(value, Mapping):
        raise AgentInstructionClassifierError(f"{label} must be a mapping")
    return value


def repository_root(start: Path) -> Path:
    current = start.resolve()
    if current.is_file():
        current = current.parent
    for candidate in (current, *current.parents):
        if (candidate / POLICY_PATH).is_file():
            return candidate
    raise AgentInstructionClassifierError("unable to locate repository root")


def load_trial_policy(root: Path) -> Mapping[str, object]:
    document = _mapping(yaml.safe_load((root / POLICY_PATH).read_text(encoding="utf-8")), str(POLICY_PATH))
    rules = _mapping(document.get("rules"), "MPD-0010.rules")
    trial = _mapping(rules.get(POLICY_SECTION), f"MPD-0010.rules.{POLICY_SECTION}")
    if trial.get("level_1_vocabulary") != list(LEVEL1):
        raise AgentInstructionClassifierError("Level 1 vocabulary mismatch")
    level2 = _mapping(trial.get("level_2_vocabulary"), "level_2_vocabulary")
    for parent, expected in LEVEL2.items():
        if level2.get(parent) != list(expected):
            raise AgentInstructionClassifierError(f"Level 2 vocabulary mismatch for {parent}")
    if trial.get("unresolved_is_namespace") is not False:
        raise AgentInstructionClassifierError("UNRESOLVED must not be a namespace")
    return trial


def _append(atoms: list[InstructionAtom], kind: str, text: str, start: int, end: int, headings: list[str], parent: int | None) -> int:
    text = " ".join(text.split())
    if not text:
        return -1
    atoms.append(InstructionAtom(f"A{len(atoms)+1:04d}", kind, text, start, end, tuple(headings), atoms[parent].atom_id if parent is not None else None))
    return len(atoms) - 1


def parse_markdown(text: str) -> tuple[InstructionAtom, ...]:
    atoms: list[InstructionAtom] = []
    headings: list[str] = []
    paragraph: list[str] = []
    paragraph_start = 0
    list_stack: list[tuple[int, int]] = []
    pending_intro: int | None = None
    active_intro: int | None = None
    code: list[str] = []
    code_start = 0
    code_parent: int | None = None
    in_code = False

    def flush(end: int) -> None:
        nonlocal paragraph, paragraph_start, pending_intro
        if not paragraph:
            return
        idx = _append(atoms, "paragraph", " ".join(paragraph), paragraph_start, end, headings, None)
        paragraph = []
        paragraph_start = 0
        pending_intro = idx if idx >= 0 and atoms[idx].text.endswith(":") else None

    lines = text.splitlines()
    for no, raw in enumerate(lines, 1):
        stripped = raw.strip()
        if stripped.startswith("```"):
            if in_code:
                _append(atoms, "code_block", "\n".join(code), code_start, max(code_start, no - 1), headings, code_parent)
                code, code_parent, in_code = [], None, False
            else:
                flush(no - 1)
                in_code, code_start, code_parent = True, no + 1, pending_intro
                pending_intro, list_stack, active_intro = None, [], None
            continue
        if in_code:
            if stripped:
                code.append(stripped)
            continue
        heading = HEADING.match(stripped)
        if heading:
            flush(no - 1)
            level = len(heading.group(1))
            headings[:] = headings[: level - 1]
            headings.append(heading.group(2).strip())
            list_stack, pending_intro, active_intro = [], None, None
            continue
        if not stripped:
            flush(no - 1)
            if list_stack:
                list_stack, active_intro = [], None
            continue
        item = LIST_ITEM.match(raw)
        if item:
            flush(no - 1)
            indent = len(item.group("indent").expandtabs(4))
            if not list_stack:
                active_intro, pending_intro = pending_intro, None
            while list_stack and list_stack[-1][0] >= indent:
                list_stack.pop()
            parent = list_stack[-1][1] if list_stack else active_intro
            idx = _append(atoms, "list_item", item.group("text"), no, no, headings, parent)
            if idx >= 0:
                list_stack.append((indent, idx))
            continue
        if not paragraph:
            pending_intro, active_intro, list_stack = None, None, []
            paragraph_start = no
        paragraph.append(stripped)
    if in_code and code:
        _append(atoms, "code_block", "\n".join(code), code_start, len(lines), headings, code_parent)
    flush(len(lines))
    return tuple(atoms)


def _direct(atom: InstructionAtom) -> tuple[str, ...]:
    text = atom.text
    found: set[str] = set()
    if CONDITION.search(text) or PATH_SCOPE.search(text):
        found.add("APPLICABILITY")
    if RULE.search(text):
        found.add("RULE")
    if ACTION.search(text) or (atom.kind == "code_block" and COMMAND.search(text)):
        found.add("ACTION")
    if EVIDENCE.search(text) and ("ACTION" in found or STATUS.search(text) or re.search(r"\b(?:evidence|verification|validation|review)\b", text, re.I)):
        found.add("EVIDENCE")
    if GOAL.search(text) or CONTEXT.search(text) or KEY_VALUE.search(text) or MAPPING.search(text) or URL.search(text) or LANG_VERSION.search(text):
        found.add("OTHER")
    return tuple(x for x in LEVEL1 if x in found)


def _clause(text: str, verbs: str) -> bool:
    return re.search(rf"(?:^|[,;:]\s+)(?:{verbs})\b", text, re.I) is not None


def _level2(atom: InstructionAtom, parents: Iterable[str]) -> dict[str, tuple[str, ...]]:
    text = atom.text
    parents = set(parents)
    out: dict[str, tuple[str, ...]] = {}
    if "APPLICABILITY" in parents:
        values = {
            name for name, hit in {
                "PATH": bool(PATH_SCOPE.search(text) or re.search(r"\b(?:path|file|directory|directories)\b", text, re.I)),
                "TASK": bool(re.search(r"\b(?:task|work|workflow|session|implementation|release|migration|review|documentation|verification)\b", text, re.I)),
                "ENVIRONMENT": bool(re.search(r"\b(?:windows|linux|macos|CI|runner|shell|powershell|bash|environment|venv)\b", text, re.I)),
                "VERSION": bool(re.search(r"\b(?:version|tool|python|node(?:\.js)?|java|go|rust)\s+v?\d+(?:\.\d+){0,3}\b", text, re.I)),
                "VCS_CONTEXT": bool(re.search(r"\b(?:git|branch|HEAD|commit|tag|checkout|merge|rebase|stash|origin/[^\s`]+)\b", text, re.I)),
            }.items() if hit
        }
        out["APPLICABILITY"] = tuple(x for x in LEVEL2["APPLICABILITY"] if x in values)
    if "RULE" in parents:
        patterns = {
            "MUTATION": r"\b(?:edit|write|change|modify|move|relocate|delete|remove|revert|reset|clean|stash|overwrite|create)\b",
            "DEPENDENCY": r"\b(?:dependency|dependencies|import|imports|depends on|dependency direction)\b",
            "COMPATIBILITY": r"\b(?:compatibility|compatible|backward|forward|supported versions?|version support)\b",
            "STYLE": r"\b(?:style|naming|formatting|format|line length|comments?|docstrings?)\b",
            "SECURITY": r"\b(?:security|secret|credential|password|token|authentication|authorization|permission|privacy)\b",
            "RESOURCE": r"\b(?:cpu|memory|disk|network|time budget|token budget|credit cost|resource)\b",
        }
        values = {name for name, pattern in patterns.items() if re.search(pattern, text, re.I)}
        out["RULE"] = tuple(x for x in LEVEL2["RULE"] if x in values)
    if "ACTION" in parents:
        values: set[str] = set()
        if COMMAND.search(text) or atom.kind == "code_block": values.add("COMMAND")
        if _clause(text, r"read|load|inspect|open"): values.add("READ")
        if _clause(text, r"write|edit|update|change|modify|remove|delete|move|relocate|create"): values.add("WRITE")
        if _clause(text, r"generate|scaffold|materialize|render"): values.add("GENERATE")
        if _clause(text, r"build|compile|package"): values.add("BUILD")
        if _clause(text, r"install") or re.search(r"(?:^|[,;:]\s+)(?:pip|npm|pnpm|yarn)\s+(?:install|add)\b", text, re.I): values.add("INSTALL")
        if _clause(text, r"publish|release|deploy|upload|promote"): values.add("PUBLISH")
        out["ACTION"] = tuple(x for x in LEVEL2["ACTION"] if x in values)
    if "EVIDENCE" in parents:
        patterns = {
            "TEST": r"\b(?:test|tests|testing|pytest|regression|smoke)\b",
            "BUILD_RESULT": r"\b(?:build result|built distribution|build output|build succeeded|build failed|distribution)\b",
            "STATIC_CHECK": r"\b(?:lint|static check|type check|typecheck|ruff|mypy|pyright|format check)\b",
            "ARTIFACT": r"\b(?:artifact|wheel|sdist|package output|generated output|distribution file)\b",
            "REVIEW": r"\b(?:review|final diff|approval|approved)\b",
            "STATUS": STATUS.pattern,
        }
        values = {name for name, pattern in patterns.items() if re.search(pattern, text, re.I)}
        out["EVIDENCE"] = tuple(x for x in LEVEL2["EVIDENCE"] if x in values)
    if "OTHER" in parents:
        values: set[str] = set()
        if GOAL.search(text): values.add("GOAL")
        if CONTEXT.search(text) or KEY_VALUE.search(text) or MAPPING.search(text) or URL.search(text) or LANG_VERSION.search(text): values.add("CONTEXT")
        out["OTHER"] = tuple(x for x in LEVEL2["OTHER"] if x in values)
    return out


def classify_atoms(atoms: Iterable[InstructionAtom]) -> tuple[ClassifiedAtom, ...]:
    atoms = tuple(atoms)
    by_id = {a.atom_id: a for a in atoms}
    direct = {a.atom_id: _direct(a) for a in atoms}
    result: list[ClassifiedAtom] = []
    for atom in atoms:
        inherited: set[str] = set()
        parent_id, seen = atom.parent_atom_id, set()
        while parent_id and parent_id not in seen:
            seen.add(parent_id)
            parent = by_id[parent_id]
            if parent.text.endswith(":"):
                inherited.update(set(direct[parent_id]) & INHERITABLE)
            parent_id = parent.parent_atom_id
        effective = tuple(x for x in LEVEL1 if x in set(direct[atom.atom_id]) | inherited)
        result.append(ClassifiedAtom(atom.atom_id, atom.kind, atom.text, atom.line_start, atom.line_end, atom.heading_path, atom.parent_atom_id, direct[atom.atom_id], tuple(x for x in LEVEL1 if x in inherited), effective, _level2(atom, effective), not effective))
    return tuple(result)


def classify_markdown(text: str) -> tuple[ClassifiedAtom, ...]:
    return classify_atoms(parse_markdown(text))


def classify_file(path: Path, root: Path) -> dict[str, object]:
    source = path.resolve() if path.is_absolute() else (root / path).resolve()
    try:
        relative = source.relative_to(root)
    except ValueError as exc:
        raise AgentInstructionClassifierError("source path must remain inside repository") from exc
    if not source.is_file():
        raise AgentInstructionClassifierError(f"source file does not exist: {relative.as_posix()}")
    load_trial_policy(root)
    atoms = classify_markdown(source.read_text(encoding="utf-8"))
    unresolved = sum(a.unresolved for a in atoms)
    return {
        "schema_version": RESULT_SCHEMA,
        "policy_ref": f"MPD-0010#{POLICY_SECTION}",
        "source": relative.as_posix(),
        "level_1_vocabulary": list(LEVEL1),
        "level_2_vocabulary": {k: list(v) for k, v in LEVEL2.items()},
        "summary": {
            "atom_count": len(atoms),
            "classified_count": len(atoms) - unresolved,
            "unresolved_count": unresolved,
            "level_1_counts": {label: sum(label in a.level1 for a in atoms) for label in LEVEL1},
        },
        "atoms": [asdict(a) for a in atoms],
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Experimental deterministic AGENTS.md classifier")
    parser.add_argument("source", nargs="?", default="AGENTS.md")
    parser.add_argument("--repository", default=".")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args(argv)
    try:
        root = repository_root(Path(args.repository))
        result = classify_file(Path(args.source), root)
    except (AgentInstructionClassifierError, OSError, yaml.YAMLError) as exc:
        print(f"agent-instruction-classifier: {exc}")
        return 2
    if args.json:
        print(json.dumps(result, ensure_ascii=False, indent=2))
    else:
        s = result["summary"]
        print(f"Agent instruction trial: {s['classified_count']}/{s['atom_count']} classified, {s['unresolved_count']} unresolved")
        print("Level 1:", ", ".join(result["level_1_vocabulary"]))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
