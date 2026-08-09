from __future__ import annotations

import json
from dataclasses import asdict, dataclass
from importlib.resources import files
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator


@dataclass(frozen=True)
class ArtifactEvidenceDocument:
    source_path: str
    payload: dict[str, object]

    def as_dict(self) -> dict[str, object]:
        return {"source_path": self.source_path, "payload": self.payload}


@dataclass(frozen=True)
class ArtifactEvidenceIssue:
    source_path: str
    message: str

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


@dataclass(frozen=True)
class ArtifactEvidenceLoad:
    documents: tuple[ArtifactEvidenceDocument, ...]
    issues: tuple[ArtifactEvidenceIssue, ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "documents": [item.as_dict() for item in self.documents],
            "issues": [item.as_dict() for item in self.issues],
            "document_count": len(self.documents),
            "valid": not self.issues,
        }


def _schema() -> dict[str, object]:
    path = files("ptsip").joinpath("specdata/ptsip-artifact-evidence.schema.json")
    return json.loads(path.read_text(encoding="utf-8"))


def load_artifact_evidence(paths: list[str | Path] | tuple[str | Path, ...] | None) -> ArtifactEvidenceLoad:
    if not paths:
        return ArtifactEvidenceLoad((), ())

    validator = Draft202012Validator(_schema())
    documents: list[ArtifactEvidenceDocument] = []
    issues: list[ArtifactEvidenceIssue] = []
    seen_ids: dict[str, str] = {}

    for raw_path in paths:
        path = Path(raw_path).expanduser().resolve()
        source = str(path)
        if not path.is_file():
            issues.append(ArtifactEvidenceIssue(source, "Artifact evidence file does not exist or is not a file."))
            continue
        try:
            payload = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
        except (OSError, UnicodeError, yaml.YAMLError) as exc:
            issues.append(ArtifactEvidenceIssue(source, f"Unable to parse artifact evidence: {exc}"))
            continue
        if not isinstance(payload, dict):
            issues.append(ArtifactEvidenceIssue(source, "Artifact evidence root must be a mapping/object."))
            continue

        validation_errors = sorted(validator.iter_errors(payload), key=lambda item: list(item.absolute_path))
        if validation_errors:
            for error in validation_errors:
                location = ".".join(str(item) for item in error.absolute_path) or "<root>"
                issues.append(ArtifactEvidenceIssue(source, f"{location}: {error.message}"))
            continue

        artifact_id = str(payload["artifact_id"])
        previous = seen_ids.get(artifact_id)
        if previous is not None:
            issues.append(
                ArtifactEvidenceIssue(
                    source,
                    f"Duplicate artifact_id {artifact_id!r}; already supplied by {previous}.",
                )
            )
            continue
        seen_ids[artifact_id] = source
        documents.append(ArtifactEvidenceDocument(source, payload))

    return ArtifactEvidenceLoad(tuple(documents), tuple(issues))
