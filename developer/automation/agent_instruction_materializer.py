from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
from typing import Mapping

import yaml

from developer.automation.agent_instruction_classifier import (
    LEVEL1,
    AgentInstructionClassifierError,
    ClassifiedAtom,
    classify_markdown,
    load_trial_policy,
    repository_root,
)


DEFAULT_SOURCE = Path("AGENTS.md")
DEFAULT_OUTPUT_ROOT = Path(".agent")
REGISTRY_NAME = "registry.yaml"
INDEX_NAME = "index.yaml"
UNRESOLVED_NAME = "unresolved.yaml"
LEVEL1_DIR = "level1"
SCHEMA_VERSION = "ptsip-agent-instruction-level1-materialization/v1"
INDEX_SCHEMA_VERSION = "ptsip-agent-instruction-level1-index/v1"
PROJECTION_SCHEMA_VERSION = "ptsip-agent-instruction-level1-projection/v1"


class AgentInstructionMaterializerError(RuntimeError):
    pass


def _safe_repo_path(root: Path, path: Path, *, label: str) -> tuple[str, Path]:
    candidate = path.resolve() if path.is_absolute() else (root / path).resolve()
    try:
        relative = candidate.relative_to(root)
    except ValueError as exc:
        raise AgentInstructionMaterializerError(
            f"{label} must remain inside repository"
        ) from exc
    return relative.as_posix(), candidate


def _source_excerpt(lines: list[str], atom: ClassifiedAtom) -> str:
    start = max(atom.line_start - 1, 0)
    end = min(atom.line_end, len(lines))
    return "".join(lines[start:end])


def _sha256_text(value: str) -> str:
    return hashlib.sha256(value.encode("utf-8")).hexdigest()


def build_materialization(
    source: Path,
    *,
    root: Path,
) -> tuple[
    dict[str, object],
    dict[str, object],
    dict[str, dict[str, object]],
    dict[str, object],
]:
    load_trial_policy(root)
    source_ref, source_path = _safe_repo_path(root, source, label="source")
    if not source_path.is_file():
        raise AgentInstructionMaterializerError(
            f"source file does not exist: {source_ref}"
        )

    source_text = source_path.read_text(encoding="utf-8")
    source_lines = source_text.splitlines(keepends=True)
    atoms = classify_markdown(source_text)

    records: list[dict[str, object]] = []
    for atom in atoms:
        excerpt = _source_excerpt(source_lines, atom)
        records.append(
            {
                "atom_id": atom.atom_id,
                "kind": atom.kind,
                "normalized_text": atom.text,
                "source": {
                    "path": source_ref,
                    "line_start": atom.line_start,
                    "line_end": atom.line_end,
                    "heading_path": list(atom.heading_path),
                    "parent_atom_id": atom.parent_atom_id,
                    "raw_excerpt": excerpt,
                    "raw_excerpt_sha256": _sha256_text(excerpt),
                },
                "level_1": list(atom.level1),
                "unresolved": atom.unresolved,
            }
        )

    classified_count = sum(not atom.unresolved for atom in atoms)
    unresolved_ids = [atom.atom_id for atom in atoms if atom.unresolved]

    registry: dict[str, object] = {
        "schema_version": SCHEMA_VERSION,
        "policy_ref": "MPD-0010#agent_instruction_entry_taxonomy_trial",
        "stage": "LEVEL_1_ONLY",
        "source": {
            "path": source_ref,
            "sha256": _sha256_text(source_text),
        },
        "level_1_vocabulary": list(LEVEL1),
        "level_2_materialized": False,
        "summary": {
            "atom_count": len(atoms),
            "classified_count": classified_count,
            "unresolved_count": len(unresolved_ids),
            "level_1_counts": {
                label: sum(label in atom.level1 for atom in atoms)
                for label in LEVEL1
            },
        },
        "atoms": records,
    }

    projections: dict[str, dict[str, object]] = {}
    projection_refs: dict[str, str] = {}
    for label in LEVEL1:
        projection_path = f"{LEVEL1_DIR}/{label.lower()}.yaml"
        projection_refs[label] = projection_path
        projections[label] = {
            "schema_version": PROJECTION_SCHEMA_VERSION,
            "stage": "LEVEL_1_ONLY",
            "level_1": label,
            "registry_ref": f"../{REGISTRY_NAME}",
            "atom_ids": [
                atom.atom_id for atom in atoms if label in atom.level1
            ],
        }

    unresolved = {
        "schema_version": PROJECTION_SCHEMA_VERSION,
        "stage": "LEVEL_1_ONLY",
        "routing_state": "UNRESOLVED",
        "registry_ref": REGISTRY_NAME,
        "atom_ids": unresolved_ids,
    }

    index = {
        "schema_version": INDEX_SCHEMA_VERSION,
        "stage": "LEVEL_1_ONLY",
        "source_ref": source_ref,
        "registry_ref": REGISTRY_NAME,
        "level_1_projections": projection_refs,
        "unresolved_ref": UNRESOLVED_NAME,
        "level_2_materialized": False,
    }
    return registry, index, projections, unresolved


def _yaml_text(payload: Mapping[str, object]) -> str:
    return yaml.safe_dump(
        dict(payload),
        sort_keys=False,
        allow_unicode=True,
        width=120,
    )


def expected_files(
    source: Path,
    *,
    root: Path,
    output_root: Path = DEFAULT_OUTPUT_ROOT,
) -> dict[Path, str]:
    _, output_path = _safe_repo_path(root, output_root, label="output root")
    registry, index, projections, unresolved = build_materialization(
        source,
        root=root,
    )
    files: dict[Path, str] = {
        output_path / REGISTRY_NAME: _yaml_text(registry),
        output_path / INDEX_NAME: _yaml_text(index),
        output_path / UNRESOLVED_NAME: _yaml_text(unresolved),
    }
    for label, payload in projections.items():
        files[output_path / LEVEL1_DIR / f"{label.lower()}.yaml"] = _yaml_text(
            payload
        )
    return files


def materialize(
    source: Path,
    *,
    root: Path,
    output_root: Path = DEFAULT_OUTPUT_ROOT,
) -> tuple[Path, ...]:
    files = expected_files(source, root=root, output_root=output_root)
    written: list[Path] = []
    for path, text in files.items():
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8", newline="\n")
        written.append(path)
    return tuple(written)


def check_materialization(
    source: Path,
    *,
    root: Path,
    output_root: Path = DEFAULT_OUTPUT_ROOT,
) -> tuple[str, ...]:
    files = expected_files(source, root=root, output_root=output_root)
    errors: list[str] = []
    for path, expected in files.items():
        relative = path.relative_to(root).as_posix()
        if not path.is_file():
            errors.append(f"MISSING:{relative}")
            continue
        actual = path.read_text(encoding="utf-8")
        if actual != expected:
            errors.append(f"STALE:{relative}")
    return tuple(errors)


def _summary(source: Path, *, root: Path) -> Mapping[str, object]:
    registry, _, _, _ = build_materialization(source, root=root)
    summary = registry["summary"]
    assert isinstance(summary, Mapping)
    return summary


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Repository-only Level 1 AGENTS.md materialization trial"
    )
    parser.add_argument(
        "command",
        choices=("status", "materialize", "check"),
        nargs="?",
        default="status",
    )
    parser.add_argument("--source", default=str(DEFAULT_SOURCE))
    parser.add_argument("--repository", default=".")
    parser.add_argument("--output", default=str(DEFAULT_OUTPUT_ROOT))
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args(argv)

    try:
        root = repository_root(Path(args.repository))
        source = Path(args.source)
        output = Path(args.output)
        if args.command == "materialize":
            written = materialize(source, root=root, output_root=output)
            result: Mapping[str, object] = {
                "state": "MATERIALIZED",
                "written": [
                    path.relative_to(root).as_posix() for path in written
                ],
                "summary": dict(_summary(source, root=root)),
                "level_2_materialized": False,
            }
            exit_code = 0
        elif args.command == "check":
            errors = check_materialization(
                source,
                root=root,
                output_root=output,
            )
            result = {
                "state": "CURRENT" if not errors else "STALE",
                "errors": list(errors),
                "summary": dict(_summary(source, root=root)),
                "level_2_materialized": False,
            }
            exit_code = 0 if not errors else 1
        else:
            result = {
                "state": "PREVIEW",
                "summary": dict(_summary(source, root=root)),
                "level_2_materialized": False,
            }
            exit_code = 0
    except (
        AgentInstructionClassifierError,
        AgentInstructionMaterializerError,
        OSError,
        yaml.YAMLError,
    ) as exc:
        print(f"agent-instruction-materializer: {exc}")
        return 2

    if args.json:
        print(json.dumps(result, ensure_ascii=True, indent=2))
    else:
        summary = result["summary"]
        assert isinstance(summary, Mapping)
        print(
            "Agent instruction Level 1 "
            f"{result['state'].lower()}: "
            f"{summary['classified_count']}/{summary['atom_count']} classified, "
            f"{summary['unresolved_count']} unresolved"
        )
        if args.command == "materialize":
            print("Level 2: not materialized")
            print(f"Output: {args.output}")
        elif args.command == "check" and result["state"] == "STALE":
            for error in result["errors"]:
                print(error)
    return exit_code


if __name__ == "__main__":
    raise SystemExit(main())
