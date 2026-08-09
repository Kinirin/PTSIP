from __future__ import annotations

import json
import hashlib
from dataclasses import asdict, dataclass
from importlib.resources import files
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

from .repository.discover import RepositoryInfo
from .repository.snapshot import capture_snapshot


@dataclass(frozen=True)
class ArtifactEvidenceDocument:
    source_path: str
    source_sha256: str
    binding_path: str | None
    binding_valid: bool
    payload: dict[str, object]

    def as_dict(self) -> dict[str, object]:
        return {
            "source_path": self.source_path,
            "source_sha256": self.source_sha256,
            "binding_path": self.binding_path,
            "binding_valid": self.binding_valid,
            "payload": self.payload,
        }


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


def _binding_issue(
    artifact_path: Path,
    digest: str,
    repository: RepositoryInfo,
) -> tuple[str | None, str | None]:
    binding_path = Path(str(artifact_path) + ".binding.json")
    if not binding_path.is_file():
        return str(binding_path), "Artifact evidence has no revision-binding sidecar."
    try:
        binding = yaml.safe_load(binding_path.read_text(encoding="utf-8-sig"))
    except (OSError, UnicodeError, yaml.YAMLError) as exc:
        return str(binding_path), f"Unable to parse artifact evidence binding: {exc}"
    if not isinstance(binding, dict) or binding.get("format") != "ptsip-artifact-evidence-binding/v1":
        return str(binding_path), "Artifact evidence binding format must be 'ptsip-artifact-evidence-binding/v1'."
    subject = binding.get("subject")
    if not isinstance(subject, dict):
        return str(binding_path), "Artifact evidence binding requires a subject object."
    expected_repositories = {repository.root}
    if repository.remote and repository.remote.repository:
        expected_repositories.add(repository.remote.repository)
    if subject.get("repository") not in expected_repositories:
        return str(binding_path), "Artifact evidence binding repository does not match the evaluated Consumer Repository."
    if not repository.commit or subject.get("revision") != repository.commit:
        return str(binding_path), "Artifact evidence binding revision does not match the evaluated repository revision."
    if binding.get("artifact_sha256") != digest:
        return str(binding_path), "Artifact evidence binding SHA-256 does not match the imported document."
    snapshot_fingerprint = capture_snapshot(repository.root).tracked_content_fingerprint
    if subject.get("tracked_content_sha256") != snapshot_fingerprint:
        return str(binding_path), "Artifact evidence binding tracked-content fingerprint does not match the evaluated repository snapshot."
    return str(binding_path), None


def load_artifact_evidence(
    paths: list[str | Path] | tuple[str | Path, ...] | None,
    repository: RepositoryInfo | None = None,
) -> ArtifactEvidenceLoad:
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
            raw = path.read_bytes()
            digest = hashlib.sha256(raw).hexdigest()
            payload = yaml.safe_load(raw.decode("utf-8-sig"))
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
        binding_path: str | None = None
        binding_valid = repository is None
        if repository is not None:
            binding_path, binding_error = _binding_issue(path, digest, repository)
            binding_valid = binding_error is None
            if binding_error is not None:
                issues.append(ArtifactEvidenceIssue(binding_path or source, binding_error))
        documents.append(ArtifactEvidenceDocument(source, digest, binding_path, binding_valid, payload))

    return ArtifactEvidenceLoad(tuple(documents), tuple(issues))
