from __future__ import annotations

from .model import CompatibilitySourceProfile, SourceFamily, SourceReadIssue, thaw_json


def _issue(profile: CompatibilitySourceProfile, code: str, message: str, pointer: str | None = None) -> SourceReadIssue:
    return SourceReadIssue(code, message, profile.generation.profile_path, pointer)


def _duplicate_ids(items: object) -> tuple[str, ...]:
    if not isinstance(items, list):
        return ()
    ids = [str(item.get("id")) for item in items if isinstance(item, dict) and item.get("id")]
    return tuple(sorted({item for item in ids if ids.count(item) > 1}))


def _policy_issues(profile: CompatibilitySourceProfile, payload: dict[str, object], component_ids: set[str]) -> list[SourceReadIssue]:
    policy = payload.get("component_dependency_policy")
    if not isinstance(policy, dict):
        return []
    issues: list[SourceReadIssue] = []
    pairs: dict[str, set[tuple[str, str]]] = {"allow": set(), "deny": set()}
    for kind in ("allow", "deny"):
        relations = policy.get(kind, [])
        if not isinstance(relations, list):
            continue
        for index, relation in enumerate(relations):
            if not isinstance(relation, dict):
                continue
            source, target = relation.get("from"), relation.get("to")
            for endpoint, value in (("from", source), ("to", target)):
                if isinstance(value, str) and value not in component_ids:
                    issues.append(_issue(profile, "SOURCE_REFERENCE_ERROR", f"Component dependency policy references undeclared component {value!r}.", f"/component_dependency_policy/{kind}/{index}/{endpoint}"))
            if isinstance(source, str) and isinstance(target, str):
                pair = (source, target)
                if pair in pairs[kind]:
                    issues.append(_issue(profile, "SOURCE_DUPLICATE_RELATION", f"Duplicate component dependency relation {source!r} -> {target!r}.", f"/component_dependency_policy/{kind}/{index}"))
                pairs[kind].add(pair)
    for source, target in sorted(pairs["allow"] & pairs["deny"]):
        issues.append(_issue(profile, "SOURCE_POLICY_CONFLICT", f"Component dependency relation {source!r} -> {target!r} appears in both allow and deny.", "/component_dependency_policy"))
    return issues


def _v034(profile: CompatibilitySourceProfile, payload: dict[str, object]) -> list[SourceReadIssue]:
    issues: list[SourceReadIssue] = []
    boundaries = payload.get("boundaries")
    if isinstance(boundaries, dict):
        seen: list[tuple[str, str]] = []
        for family_name, item in boundaries.items():
            if not isinstance(item, dict):
                continue
            for root in item.get("roots", []):
                normalized = str(root).replace("\\", "/").strip("/")
                for previous_name, previous_root in seen:
                    if normalized == previous_root or normalized.startswith(previous_root + "/") or previous_root.startswith(normalized + "/"):
                        issues.append(_issue(profile, "SOURCE_BOUNDARY_OVERLAP", f"Historical boundary {family_name}:{normalized!r} overlaps {previous_name}:{previous_root!r}.", f"/boundaries/{family_name}"))
                seen.append((str(family_name), normalized))
    components = payload.get("components")
    duplicates = _duplicate_ids(components)
    if duplicates:
        issues.append(_issue(profile, "SOURCE_DUPLICATE_ID", "Duplicate component IDs: " + ", ".join(duplicates), "/components"))
    if isinstance(components, list):
        ids = {str(item.get("id")) for item in components if isinstance(item, dict) and item.get("id")}
        issues.extend(_policy_issues(profile, payload, ids))
    return issues


def _v036(profile: CompatibilitySourceProfile, payload: dict[str, object]) -> list[SourceReadIssue]:
    issues: list[SourceReadIssue] = []
    responsibility_map = payload.get("responsibility_map")
    if not isinstance(responsibility_map, dict):
        return issues
    mode = responsibility_map.get("mode")
    if mode == "hybrid":
        overrides = responsibility_map.get("overrides")
        if not isinstance(overrides, dict):
            return issues
        for label, remove_label in (("components", "remove_component_ids"), ("associated_artifacts", "remove_associated_artifact_ids"), ("relationships", "remove_relationship_ids")):
            duplicates = _duplicate_ids(overrides.get(label))
            if duplicates:
                issues.append(_issue(profile, "SOURCE_DUPLICATE_ID", f"Duplicate {label} override IDs: " + ", ".join(duplicates), f"/responsibility_map/overrides/{label}"))
            declared = {str(item.get("id")) for item in overrides.get(label, []) if isinstance(item, dict) and item.get("id")} if isinstance(overrides.get(label, []), list) else set()
            removed = {str(item) for item in overrides.get(remove_label, []) if isinstance(item, str)}
            conflict = sorted(declared & removed)
            if conflict:
                issues.append(_issue(profile, "SOURCE_OVERRIDE_REMOVE_CONFLICT", f"{label} IDs cannot be both overridden and removed: " + ", ".join(conflict), "/responsibility_map/overrides"))
        return issues
    if mode != "explicit":
        return issues

    components = payload.get("components", [])
    artifacts = payload.get("associated_artifacts", [])
    relationships = payload.get("relationships", [])
    for label, items in (("components", components), ("associated_artifacts", artifacts), ("relationships", relationships)):
        duplicates = _duplicate_ids(items)
        if duplicates:
            issues.append(_issue(profile, "SOURCE_DUPLICATE_ID", f"Duplicate {label} IDs: " + ", ".join(duplicates), f"/{label}"))
    component_ids = {str(item.get("id")) for item in components if isinstance(item, dict) and item.get("id")} if isinstance(components, list) else set()
    artifact_ids = {str(item.get("id")) for item in artifacts if isinstance(item, dict) and item.get("id")} if isinstance(artifacts, list) else set()
    collision = sorted(component_ids & artifact_ids)
    if collision:
        issues.append(_issue(profile, "SOURCE_ENDPOINT_ID_COLLISION", "Component and associated-artifact IDs share one endpoint namespace: " + ", ".join(collision), ""))
    endpoint_ids = component_ids | artifact_ids
    semantic_edges: set[tuple[str, str, str]] = set()
    if isinstance(relationships, list):
        for index, relation in enumerate(relationships):
            if not isinstance(relation, dict):
                continue
            source, target, kind = relation.get("from"), relation.get("to"), relation.get("type")
            for endpoint, value in (("from", source), ("to", target)):
                if isinstance(value, str) and value not in endpoint_ids:
                    issues.append(_issue(profile, "SOURCE_REFERENCE_ERROR", f"Relationship endpoint {value!r} is not declared by the source profile.", f"/relationships/{index}/{endpoint}"))
            if isinstance(source, str) and isinstance(target, str) and isinstance(kind, str):
                edge = (source, target, kind)
                if edge in semantic_edges:
                    issues.append(_issue(profile, "SOURCE_DUPLICATE_RELATION", f"Duplicate semantic relationship {source!r} --{kind}--> {target!r}.", f"/relationships/{index}"))
                semantic_edges.add(edge)
    if isinstance(artifacts, list):
        for index, artifact in enumerate(artifacts):
            if isinstance(artifact, dict) and isinstance(artifact.get("anchor"), str) and artifact["anchor"] not in component_ids:
                issues.append(_issue(profile, "SOURCE_REFERENCE_ERROR", f"Associated-artifact anchor {artifact['anchor']!r} is not a declared component.", f"/associated_artifacts/{index}/anchor"))
    issues.extend(_policy_issues(profile, payload, component_ids))
    return issues


def source_integrity_issues(profile: CompatibilitySourceProfile) -> tuple[SourceReadIssue, ...]:
    payload = thaw_json(profile.raw_payload)
    if not isinstance(payload, dict):
        return (_issue(profile, "SOURCE_SHAPE_ERROR", "Compatibility source payload is not a mapping.", ""),)
    issues = _v034(profile, payload) if profile.family is SourceFamily.TOOL_035_PROFILE else _v036(profile, payload)
    return tuple(issues)
