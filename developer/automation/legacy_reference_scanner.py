from __future__ import annotations

import json
import re
import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Mapping

import yaml
from jsonschema import Draft202012Validator


INVENTORY = "developer/policy/legacy-reference-inventory.yaml"
INVENTORY_SCHEMA = "developer/policy/schemas/legacy-reference-inventory.schema.json"

REFERENCE_RE = re.compile(
    r"decisions/[A-Za-z0-9_.*{}<>/\-]+(?:\.[A-Za-z0-9_.*{}<>/\-]+)?|\bADR-[0-9]{4}\b"
)
SFP_POLICY_RE = re.compile(r"^src/ptsip/specdata/SFP-[0-9]{4}\.yaml$")
MPD_POLICY_RE = re.compile(r"^developer/policy/MPD-000[2-9]\.yaml$")
HISTORICAL_SPEC_NOTE_RE = re.compile(r"^releasenote/specification/spec-.+\.md$")

ACTIVE_PATHS = {
    "ptsip.yaml",
    "developer/profiles/ptsip-repository.yaml",
    "releasenote/project-profile/pp.1.01.md",
}
FROZEN_PROVENANCE_PATHS = {
    "spec/PTSIP-SPEC.md",
    "spec/PTSIP-RESPONSIBILITY-MAP.md",
    "spec/PTSIP-DRAFT-PROFILE-TRANSITION.md",
    "schemas/ptsip-profile-pp-1.01.schema.json",
    "src/ptsip/specdata/ptsip-profile-pp-1.01.schema.json",
}
NON_LEGACY_NAMESPACE_PATHS = {
    "src/ptsip/app/github_authority.py",
    "tests/ptsip/control_plane/test_github_authority_reconciliation.py",
    "releasenote/specification/spec-0.3.3-draft.md",
}
MIGRATION_EXACT_PATHS = {
    "docs/planning/0.4.0/WU-02/WU-02-P01.yaml",
    "schemas/ptsip-adr-index.schema.json",
    "schemas/ptsip-adr.schema.json",
    "schemas/ptsip-adr-template.schema.json",
    "schemas/ptsip-governance-authority-registry.schema.json",
    "schemas/ptsip-governance-authority-role.schema.json",
    "schemas/ptsip-governance-authority-role-registry.schema.json",
    "schemas/ptsip-governance-authority-semantics.schema.json",
    "schemas/ptsip-governance-authority-subject-registry.schema.json",
    "schemas/ptsip-governance-subject-binding.schema.json",
    "schemas/ptsip-project-authority-record.schema.json",
    "schemas/ptsip-authority-eligibility-result.schema.json",
    "schemas/ptsip-authorization-transition.schema.json",
}


@dataclass(frozen=True)
class ReferenceHit:
    path: str
    line: int
    column: int
    reference: str
    classification: str


def _git(root: Path, *args: str) -> str:
    result = subprocess.run(
        ["git", *args], cwd=root, check=True, capture_output=True, text=True
    )
    return result.stdout.strip()


def _load_yaml(root: Path, relative: str) -> dict[str, object]:
    value = yaml.safe_load((root / relative).read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{relative}: expected mapping")
    return value


def _load_json(root: Path, relative: str) -> dict[str, object]:
    value = json.loads((root / relative).read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{relative}: expected mapping")
    return value


def _tracked_files(root: Path) -> tuple[Path, ...]:
    return tuple(root / item for item in _git(root, "ls-files").splitlines())


def _is_non_legacy_namespace(path: str, reference: str) -> bool:
    return (
        path in NON_LEGACY_NAMESPACE_PATHS
        and reference.startswith("decisions/")
        and "ADR-" not in reference
    )


def _classification(path: str, reference: str) -> str:
    if path in ACTIVE_PATHS:
        return "ACTIVE_DEPENDENCY"
    if SFP_POLICY_RE.fullmatch(path) or MPD_POLICY_RE.fullmatch(path):
        return "HISTORICAL_PROVENANCE"
    if path in FROZEN_PROVENANCE_PATHS or HISTORICAL_SPEC_NOTE_RE.fullmatch(path):
        return "HISTORICAL_PROVENANCE"
    if _is_non_legacy_namespace(path, reference):
        return "NON_LEGACY_NAMESPACE"
    if path.startswith("developer/") or path in MIGRATION_EXACT_PATHS:
        return "MIGRATION_ONLY_REFERENCE"
    return "UNCLASSIFIED"


def scan_legacy_references(root: str | Path) -> tuple[ReferenceHit, ...]:
    base = Path(root).resolve()
    hits: list[ReferenceHit] = []
    for path in _tracked_files(base):
        relative = path.relative_to(base).as_posix()
        if relative.startswith("decisions/"):
            continue
        try:
            lines = path.read_text(encoding="utf-8", errors="strict").splitlines()
        except (UnicodeDecodeError, OSError):
            continue
        for line_no, line in enumerate(lines, start=1):
            for match in REFERENCE_RE.finditer(line):
                reference = match.group(0)
                hits.append(
                    ReferenceHit(
                        path=relative,
                        line=line_no,
                        column=match.start() + 1,
                        reference=reference,
                        classification=_classification(relative, reference),
                    )
                )
    return tuple(hits)


def scan_summary(root: str | Path) -> dict[str, object]:
    hits = scan_legacy_references(root)
    counts: dict[str, int] = {}
    files: dict[str, set[str]] = {}
    for hit in hits:
        counts[hit.classification] = counts.get(hit.classification, 0) + 1
        files.setdefault(hit.classification, set()).add(hit.path)
    return {
        "counts": counts,
        "files": {key: sorted(value) for key, value in files.items()},
        "hits": hits,
    }


def validate_legacy_reference_inventory(root: str | Path) -> tuple[str, ...]:
    base = Path(root).resolve()
    errors: list[str] = []
    inventory = _load_yaml(base, INVENTORY)
    schema = _load_json(base, INVENTORY_SCHEMA)
    Draft202012Validator.check_schema(schema)
    for error in Draft202012Validator(schema).iter_errors(inventory):
        errors.append(f"{INVENTORY}: {error.message}")

    basis = inventory["scan_basis"]
    anchor_revision = str(basis["legacy_corpus_revision"])
    expected_tree = str(basis["legacy_decisions_tree_sha"])
    try:
        anchor_tree = _git(base, "rev-parse", f"{anchor_revision}:decisions")
        current_tree = _git(base, "rev-parse", "HEAD:decisions")
    except subprocess.CalledProcessError as exc:
        errors.append(f"legacy decisions tree identity could not be resolved: {exc}")
    else:
        if anchor_tree != expected_tree:
            errors.append("legacy corpus anchor decisions tree does not match registered SHA")
        if current_tree != expected_tree:
            errors.append("current decisions tree changed after classified legacy corpus anchor")

    summary = scan_summary(base)
    hits = summary["hits"]
    unclassified = [item for item in hits if item.classification == "UNCLASSIFIED"]
    if unclassified:
        errors.extend(
            f"unclassified legacy reference: {item.path}:{item.line}:{item.column} {item.reference}"
            for item in unclassified
        )

    active = [item for item in hits if item.classification == "ACTIVE_DEPENDENCY"]
    expected_active = inventory["active_dependencies"]
    if len(active) != expected_active["expected_reference_count"]:
        errors.append(
            f"active legacy reference count is {len(active)}; expected {expected_active['expected_reference_count']}"
        )
    active_files = sorted({item.path for item in active})
    expected_files = sorted(item["path"] for item in expected_active["entries"])
    if active_files != expected_files:
        errors.append(f"active legacy reference files differ: {active_files!r} != {expected_files!r}")
    for entry in expected_active["entries"]:
        path_hits = [item for item in active if item.path == entry["path"]]
        if len(path_hits) != entry["expected_reference_count"]:
            errors.append(
                f"{entry['path']}: active reference count {len(path_hits)} != {entry['expected_reference_count']}"
            )
        actual_refs: dict[str, int] = {}
        for item in path_hits:
            actual_refs[item.reference] = actual_refs.get(item.reference, 0) + 1
        if actual_refs != dict(entry["references"]):
            errors.append(f"{entry['path']}: active reference occurrence map changed")

    provenance = inventory["historical_provenance"]["policy_source_provenance"]
    sfp_files = sorted({item.path for item in hits if item.classification == "HISTORICAL_PROVENANCE" and SFP_POLICY_RE.fullmatch(item.path)})
    mpd_files = sorted({item.path for item in hits if item.classification == "HISTORICAL_PROVENANCE" and MPD_POLICY_RE.fullmatch(item.path)})
    if len(sfp_files) != provenance["sfp_file_count"]:
        errors.append(f"historical SFP provenance file count is {len(sfp_files)}; expected {provenance['sfp_file_count']}")
    if len(mpd_files) != provenance["mpd_file_count"]:
        errors.append(f"historical MPD provenance file count is {len(mpd_files)}; expected {provenance['mpd_file_count']}")

    for dependency in inventory["migration_only_references"]["transitive_dependencies"]:
        consumer = base / dependency["consumer"]
        if not consumer.is_file():
            errors.append(f"migration-only transitive dependency consumer missing: {dependency['consumer']}")

    return tuple(errors)


def removal_blockers(root: str | Path) -> tuple[str, ...]:
    base = Path(root).resolve()
    inventory = _load_yaml(base, INVENTORY)
    summary = scan_summary(base)
    blockers: list[str] = []
    if summary["counts"].get("UNCLASSIFIED", 0):
        blockers.append("UNCLASSIFIED_LEGACY_REFERENCE_NONZERO")
    if summary["counts"].get("ACTIVE_DEPENDENCY", 0):
        blockers.append("ACTIVE_REFERENCE_COUNT_NONZERO")
    if inventory["migration_only_references"]["retirement_required_before_decisions_removal"]:
        blockers.append("MIGRATION_ONLY_REFERENCE_RETIREMENT_PENDING")
    if not inventory["historical_provenance"]["policy_source_provenance"]["source_revision_anchor_materialized_in_policy_records"]:
        blockers.append("HISTORICAL_PROVENANCE_REVISION_ANCHOR_NOT_MATERIALIZED")
    return tuple(sorted(set(blockers)))


if __name__ == "__main__":
    base = Path(__file__).resolve().parents[2]
    failures = validate_legacy_reference_inventory(base)
    summary = scan_summary(base)
    for classification, count in sorted(summary["counts"].items()):
        print(f"{classification}: {count}")
    print("Removal blockers:", ", ".join(removal_blockers(base)) or "NONE")
    if failures:
        raise SystemExit("\n".join(failures))
