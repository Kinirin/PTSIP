from __future__ import annotations

import fnmatch
from dataclasses import asdict, dataclass
from pathlib import Path

import yaml

from .repository.snapshot import repository_files
from .validation.components import ComponentPartition


@dataclass(frozen=True)
class ReleaseWorkflowEvidence:
    path: str
    release_like: bool
    trigger_paths: tuple[str, ...]
    scoped_classifications: tuple[str, ...]
    scope_complete: bool
    scope_source: str | None = None
    manual_only: bool = False
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


def _trigger_events(payload: dict[str, object]) -> tuple[str, ...]:
    trigger = _workflow_on(payload)
    if isinstance(trigger, str):
        return (trigger,)
    if isinstance(trigger, list):
        return tuple(dict.fromkeys(str(item) for item in trigger))
    if isinstance(trigger, dict):
        return tuple(dict.fromkeys(str(item) for item in trigger.keys()))
    return ()


def _manual_dispatch_only(payload: dict[str, object]) -> bool:
    return set(_trigger_events(payload)) == {"workflow_dispatch"}


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
    return any(word in text for word in ("release", "publish", "deploy", "distribution", "twine upload", "npm publish"))


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
            if classification in {"PRODUCT", "TOOLCHAIN", "NEUTRAL_CONTRACT"}:
                scope.add(classification)
    return tuple(sorted(scope)), matched


def _artifact_scope_for_manual_workflow(
    workflow_owner: str | None,
    artifact_documents: list[dict[str, object]] | tuple[dict[str, object], ...] | None,
) -> tuple[tuple[str, ...], bool, str | None]:
    if not workflow_owner:
        return (), False, "Manual release workflow is not assigned to a declared producer component."

    classifications: set[str] = set()
    artifact_ids: list[str] = []
    for document in artifact_documents or ():
        if not isinstance(document, dict) or document.get("binding_valid") is not True:
            continue
        payload = document.get("payload")
        if not isinstance(payload, dict):
            continue
        if payload.get("producer_component") != workflow_owner:
            continue
        if payload.get("provenance") != "OBSERVED":
            continue
        contents = payload.get("contents")
        if not isinstance(contents, dict) or contents.get("complete") is not True:
            continue
        classification = payload.get("classification")
        if classification not in {"PRODUCT", "TOOLCHAIN", "NEUTRAL_CONTRACT"}:
            continue
        classifications.add(str(classification))
        artifact_ids.append(str(payload.get("artifact_id", "unknown")))

    if not classifications:
        return (
            (),
            False,
            "Manual release workflow has no revision-bound OBSERVED complete artifact evidence produced by its owning component.",
        )
    if len(classifications) != 1:
        return (
            tuple(sorted(classifications)),
            False,
            "Manual release workflow producer is correlated to artifact evidence from more than one architectural classification.",
        )
    classification = next(iter(classifications))
    return (
        (classification,),
        True,
        "Manual workflow scope is established by revision-bound OBSERVED complete artifact evidence: "
        + ", ".join(sorted(set(artifact_ids))),
    )


def evaluate_lifecycle_evidence(
    repository_root: str | Path,
    components: list[dict[str, object]],
    partition: ComponentPartition,
    artifact_documents: list[dict[str, object]] | tuple[dict[str, object], ...] | None = None,
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

    relevant = [item for item in components if str(item.get("classification")) in {"PRODUCT", "TOOLCHAIN"}]
    for field, rule_id in (("release_owner", "PTSIP-LCY-001"), ("compatibility_owner", "PTSIP-LCY-002")):
        missing = [str(item.get("id")) for item in relevant if not isinstance(item.get(field), str) or not str(item.get(field)).strip()]
        if missing:
            gaps.append(
                _gap(
                    f"{field}-missing",
                    f"Lifecycle evidence is missing {field} for component(s): {', '.join(sorted(missing))}.",
                    (rule_id, "PTSIP-EVD-003"),
                )
            )

    _mode, paths, discovery_errors = repository_files(root)
    for index, error in enumerate(discovery_errors):
        gaps.append(_gap(f"repository-scan-{index}", f"Lifecycle workflow discovery was incomplete: {error}", ("PTSIP-LCY-001", "PTSIP-EVD-003")))

    workflows: list[ReleaseWorkflowEvidence] = []
    for rel in paths:
        suffix = Path(rel).suffix.lower()
        if not rel.startswith(".github/workflows/") or suffix not in {".yaml", ".yml"}:
            continue
        try:
            payload = yaml.safe_load((root / rel).read_text(encoding="utf-8-sig")) or {}
        except (OSError, UnicodeError, yaml.YAMLError) as exc:
            gaps.append(_gap(f"workflow-parse:{rel}", f"Unable to parse lifecycle workflow {rel!r}: {exc}", ("PTSIP-LCY-001", "PTSIP-EVD-003")))
            continue
        if not isinstance(payload, dict):
            gaps.append(_gap(f"workflow-shape:{rel}", f"Lifecycle workflow {rel!r} root is not a mapping.", ("PTSIP-LCY-001", "PTSIP-EVD-003")))
            continue
        if not _release_like(rel, payload):
            continue

        trigger_paths = _trigger_paths(payload)
        manual_only = _manual_dispatch_only(payload)
        scope, complete = _scope_from_paths(trigger_paths, owners, classifications)
        scope_source: str | None = "trigger-paths" if complete else None
        reason: str | None = None

        if not complete and not trigger_paths and manual_only:
            artifact_scope, artifact_complete, artifact_reason = _artifact_scope_for_manual_workflow(
                owners.get(rel),
                artifact_documents,
            )
            scope = artifact_scope
            complete = artifact_complete
            scope_source = "artifact-evidence" if artifact_complete else None
            reason = None if artifact_complete else artifact_reason
            if artifact_complete and artifact_reason:
                observations.append(f"{rel}: {artifact_reason}")
        elif not trigger_paths:
            reason = "Release-like workflow has no positive path scope; trigger alone cannot establish Product/Toolchain release independence."
        elif not complete:
            reason = "Release-like workflow path filters did not resolve to tracked component ownership."

        workflows.append(
            ReleaseWorkflowEvidence(
                path=rel,
                release_like=True,
                trigger_paths=trigger_paths,
                scoped_classifications=scope,
                scope_complete=complete,
                scope_source=scope_source,
                manual_only=manual_only,
                reason=reason,
            )
        )

    product_scoped = [item for item in workflows if item.scope_complete and item.scoped_classifications == ("PRODUCT",)]
    toolchain_scoped = [item for item in workflows if item.scope_complete and item.scoped_classifications == ("TOOLCHAIN",)]
    ambiguous = [item for item in workflows if not item.scope_complete or len(item.scoped_classifications) != 1]

    product_release_required = any(
        str(item.get("classification")) == "PRODUCT" and item.get("shipped") is not False
        for item in relevant
    )
    toolchain_release_required = any(
        str(item.get("classification")) == "TOOLCHAIN" and item.get("shipped") is not False
        for item in relevant
    )
    has_toolchain = any(str(item.get("classification")) == "TOOLCHAIN" for item in relevant)

    if product_release_required and not product_scoped:
        gaps.append(
            _gap(
                "product-release-evidence",
                "No release-like workflow provides Product-only lifecycle evidence from positive path scope or revision-bound observed Product artifact evidence.",
                ("PTSIP-LCY-001", "PTSIP-EVD-003"),
            )
        )
    if toolchain_release_required and not toolchain_scoped:
        gaps.append(
            _gap(
                "toolchain-release-evidence",
                "No release-like workflow provides Toolchain-only lifecycle evidence for a Toolchain component whose shipped state is not explicitly false.",
                ("PTSIP-LCY-001", "PTSIP-EVD-003"),
            )
        )
    elif has_toolchain and not toolchain_scoped:
        observations.append(
            "No Toolchain-only release workflow was required because all declared Toolchain components explicitly set shipped=false."
        )

    for item in ambiguous:
        observations.append(f"{item.path}: {item.reason or 'Release-like workflow spans more than one architectural classification; this is not by itself a lifecycle violation.'}")
        gaps.append(
            _gap(
                f"ambiguous-release-workflow:{item.path}",
                (
                    f"Release-like workflow {item.path!r} cannot be attributed to exactly one architectural plane. "
                    "Its trigger is not itself a violation, but lifecycle independence is not sufficiently evidenced."
                ),
                ("PTSIP-LCY-001", "PTSIP-EVD-003"),
            )
        )

    if product_scoped:
        observations.append("Observed Product-only release workflow scope: " + ", ".join(item.path for item in product_scoped))
    if toolchain_scoped:
        observations.append("Observed Toolchain-only release workflow scope: " + ", ".join(item.path for item in toolchain_scoped))

    status = "RAN" if not gaps else "BLOCKED"
    reason = None if not gaps else "LIFECYCLE_EVIDENCE_INCOMPLETE"
    return LifecycleEvaluationResult(status, reason, tuple(workflows), tuple(gaps), tuple(observations))
