from __future__ import annotations

import fnmatch
from dataclasses import asdict, dataclass
from pathlib import Path

import yaml

from .repository.snapshot import repository_files
from .validation.components import ComponentPartition


_CANONICAL_CLASSIFICATIONS = {
    "PRODUCT",
    "DEVELOPMENT_TOOLING",
    "DELIVERY",
    "OPERATIONS",
    "NEUTRAL_CONTRACT",
}


@dataclass(frozen=True)
class ReleaseWorkflowEvidence:
    path: str
    release_like: bool
    trigger_paths: tuple[str, ...]
    scoped_classifications: tuple[str, ...]
    scope_complete: bool
    reason: str | None = None

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class LifecycleEvaluationResult:
    status: str
    reason: str | None
    workflows: tuple[ReleaseWorkflowEvidence, ...]
    blocking_gaps: tuple[dict[str, object], ...]
    observations: tuple[str, ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "status": self.status,
            "reason": self.reason,
            "workflows": [item.as_dict() for item in self.workflows],
            "blocking_gaps": list(self.blocking_gaps),
            "observations": list(self.observations),
        }


def _gap(gap_id: str, message: str, rule_ids: tuple[str, ...]) -> dict[str, object]:
    return {
        "id": f"lifecycle:{gap_id}",
        "blocking": True,
        "rule_ids": list(rule_ids),
        "evidence_ids": [f"lifecycle:{gap_id}"],
        "message": message,
    }


def _workflow_on(payload: dict[str, object]) -> object:
    if "on" in payload:
        return payload.get("on")
    return payload.get(True)


def _release_like(path: str, payload: dict[str, object]) -> bool:
    tokens: list[str] = [path.lower()]
    name = payload.get("name")
    if isinstance(name, str):
        tokens.append(name.lower())
    jobs = payload.get("jobs", {})
    if isinstance(jobs, dict):
        for job_name, job in jobs.items():
            tokens.append(str(job_name).lower())
            if not isinstance(job, dict):
                continue
            steps = job.get("steps", [])
            if not isinstance(steps, list):
                continue
            for step in steps:
                if not isinstance(step, dict):
                    continue
                for key in ("name", "run", "uses"):
                    value = step.get(key)
                    if isinstance(value, str):
                        tokens.append(value.lower())
    trigger = _workflow_on(payload)
    if isinstance(trigger, str):
        tokens.append(trigger.lower())
    elif isinstance(trigger, list):
        tokens.extend(str(item).lower() for item in trigger)
    elif isinstance(trigger, dict):
        tokens.extend(str(item).lower() for item in trigger.keys())
    text = " ".join(tokens)
    return any(
        word in text
        for word in (
            "release",
            "publish",
            "deploy",
            "distribution",
            "twine upload",
            "npm publish",
        )
    )


def _trigger_paths(payload: dict[str, object]) -> tuple[str, ...]:
    trigger = _workflow_on(payload)
    if not isinstance(trigger, dict):
        return ()
    collected: list[str] = []
    for event in ("push", "pull_request", "workflow_run"):
        config = trigger.get(event)
        if not isinstance(config, dict):
            continue
        paths = config.get("paths")
        if isinstance(paths, list):
            collected.extend(str(item) for item in paths if isinstance(item, str) and item.strip())
    return tuple(dict.fromkeys(collected))


def _normalize_pattern(pattern: str) -> str:
    normalized = pattern.replace("\\", "/")
    while normalized.startswith("./"):
        normalized = normalized[2:]
    return normalized


def _path_matches(path: str, pattern: str) -> bool:
    normalized = _normalize_pattern(pattern)
    candidate = path.replace("\\", "/")
    if fnmatch.fnmatchcase(candidate, normalized):
        return True
    if normalized.endswith("/**"):
        return candidate.startswith(normalized[:-3].rstrip("/") + "/")
    return False


def _scope_from_paths(
    trigger_paths: tuple[str, ...],
    owners: dict[str, str],
    classifications: dict[str, str],
) -> tuple[tuple[str, ...], bool]:
    if not trigger_paths:
        return (), False
    scope: set[str] = set()
    matched = False
    positive_patterns = [item for item in trigger_paths if not item.startswith("!")]
    negative_patterns = [item[1:] for item in trigger_paths if item.startswith("!") and len(item) > 1]
    if not positive_patterns:
        return (), False
    for path, component_id in owners.items():
        positive = any(_path_matches(path, pattern) for pattern in positive_patterns)
        excluded = any(_path_matches(path, pattern) for pattern in negative_patterns)
        if positive and not excluded:
            matched = True
            classification = classifications.get(component_id)
            if classification in _CANONICAL_CLASSIFICATIONS:
                scope.add(classification)
    return tuple(sorted(scope)), matched


def evaluate_lifecycle_evidence(
    repository_root: str | Path,
    components: list[dict[str, object]],
    partition: ComponentPartition,
) -> LifecycleEvaluationResult:
    root = Path(repository_root).resolve()
    owners = {assignment.path: assignment.component_id for assignment in partition.assignments}
    classifications = {
        str(component.get("id")): str(component.get("classification"))
        for component in components
        if component.get("id") and component.get("classification")
    }
    gaps: list[dict[str, object]] = []
    observations: list[str] = []

    relevant = [
        item
        for item in components
        if str(item.get("classification")) in _CANONICAL_CLASSIFICATIONS
    ]

    # Enforced lifecycle-independence evaluation needs explicit ownership facts.
    # These are evidence metadata, not a second classification authority.
    for field, rule_id in (
        ("release_owner", "PTSIP-LCY-001"),
        ("compatibility_owner", "PTSIP-LCY-002"),
    ):
        missing = [
            str(item.get("id"))
            for item in relevant
            if not isinstance(item.get(field), str) or not str(item.get(field)).strip()
        ]
        if missing:
            gaps.append(
                _gap(
                    f"{field}-missing",
                    (
                        f"Lifecycle evidence is missing {field} for component(s): "
                        f"{', '.join(sorted(missing))}. The field is not required for generic profile "
                        "validity, but this strict lifecycle evaluator cannot establish governability without it."
                    ),
                    (rule_id, "PTSIP-EVD-003"),
                )
            )

    _mode, paths, discovery_errors = repository_files(root)
    for index, error in enumerate(discovery_errors):
        gaps.append(
            _gap(
                f"repository-scan-{index}",
                f"Lifecycle workflow discovery was incomplete: {error}",
                ("PTSIP-LCY-001", "PTSIP-EVD-003"),
            )
        )

    workflows: list[ReleaseWorkflowEvidence] = []
    for rel in paths:
        suffix = Path(rel).suffix.lower()
        if not rel.startswith(".github/workflows/") or suffix not in {".yaml", ".yml"}:
            continue
        try:
            payload = yaml.safe_load((root / rel).read_text(encoding="utf-8-sig")) or {}
        except (OSError, UnicodeError, yaml.YAMLError) as exc:
            gaps.append(
                _gap(
                    f"workflow-parse:{rel}",
                    f"Unable to parse lifecycle workflow {rel!r}: {exc}",
                    ("PTSIP-LCY-001", "PTSIP-EVD-003"),
                )
            )
            continue
        if not isinstance(payload, dict):
            gaps.append(
                _gap(
                    f"workflow-shape:{rel}",
                    f"Lifecycle workflow {rel!r} root is not a mapping.",
                    ("PTSIP-LCY-001", "PTSIP-EVD-003"),
                )
            )
            continue
        if not _release_like(rel, payload):
            continue

        trigger_paths = _trigger_paths(payload)
        scope, complete = _scope_from_paths(trigger_paths, owners, classifications)
        owner_component = owners.get(rel)
        owner_classification = classifications.get(owner_component) if owner_component else None
        reason: str | None = None
        if not trigger_paths:
            reason = (
                "Release-like workflow has no positive path scope. This is not a classification failure: "
                "workflow triggers are evidence, not lifecycle authority."
            )
        elif not complete:
            reason = (
                "Release-like workflow path filters did not resolve completely to tracked component ownership. "
                "Do not infer lifecycle ownership from the trigger alone."
            )

        workflows.append(
            ReleaseWorkflowEvidence(rel, True, trigger_paths, scope, complete, reason)
        )

        if owner_classification == "DELIVERY":
            observations.append(
                f"{rel}: release-like workflow is declared under DELIVERY; trigger scope={scope or ('UNSCOPED',)}."
            )
        elif owner_classification is not None:
            observations.append(
                f"{rel}: release-like workflow is declared under {owner_classification}. The activity name does not override "
                "its governing lifecycle obligation; review only if repository evidence contradicts the declaration."
            )
        else:
            observations.append(
                f"{rel}: release-like workflow has no resolved component owner; trigger/workflow naming is insufficient "
                "to assign DELIVERY automatically."
            )
        if reason:
            observations.append(f"{rel}: {reason}")

    delivery_components = sorted(
        str(item.get("id"))
        for item in relevant
        if str(item.get("classification")) == "DELIVERY"
    )
    operations_components = sorted(
        str(item.get("id"))
        for item in relevant
        if str(item.get("classification")) == "OPERATIONS"
    )
    if delivery_components:
        observations.append("Declared DELIVERY responsibility: " + ", ".join(delivery_components))
    if operations_components:
        observations.append("Declared OPERATIONS responsibility: " + ", ".join(operations_components))

    status = "RAN" if not gaps else "BLOCKED"
    reason = None if not gaps else "LIFECYCLE_EVIDENCE_INCOMPLETE"
    return LifecycleEvaluationResult(
        status,
        reason,
        tuple(workflows),
        tuple(gaps),
        tuple(observations),
    )
