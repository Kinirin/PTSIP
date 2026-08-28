from __future__ import annotations

import hashlib
from pathlib import Path

import pytest

from ptsip.evidence.contract import (
    EvidenceAssertion,
    EvidenceChannel,
    EvidenceChannelStatus,
    EvidenceEvaluationContext,
    EvidenceOrigin,
    EvidenceRecord,
    EvidenceRecordStatus,
    NormalizedEvidenceSet,
    SnapshotBinding,
    SourceGenerationBinding as EvidenceSourceGenerationBinding,
)
from ptsip.migration import (
    ArchitectureFindingKind,
    LifecycleFindingKind,
    TargetArchitectureState,
    TargetCompatibility,
    TargetComponent,
    analyze_source_migration,
    target_state_from_mapping,
)
from ptsip.repository.snapshot import capture_snapshot
from ptsip.source_compat.model import (
    CompatibilitySourceProfile,
    SourceBoundary,
    SourceComponent,
    SourceDeclarationScope,
    SourceFamily,
    SourceGenerationBinding,
    SourceLocation,
    SourceRelationship,
    V034SourceSemantics,
    V036SourceSemantics,
    freeze_json,
)


def _repo(tmp_path: Path, files: dict[str, str]) -> Path:
    (tmp_path / "ptsip.yaml").write_text("source-profile", encoding="utf-8")
    for rel, content in files.items():
        path = tmp_path / rel
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")
    return tmp_path


def _generation(root: Path, version: str = "0.3.6-draft") -> SourceGenerationBinding:
    digest = hashlib.sha256((root / "ptsip.yaml").read_bytes()).hexdigest()
    return SourceGenerationBinding(
        profile_path="ptsip.yaml",
        declared_version=version,
        specification_revision="revision",
        specification_source="https://github.com/Kinirin/PTSIP",
        content_sha256=digest,
        temporary=False,
    )


def _evidence(
    root: Path,
    generation: SourceGenerationBinding,
    records: tuple[EvidenceRecord, ...] = (),
    channels: tuple[EvidenceChannel, ...] | None = None,
) -> NormalizedEvidenceSet:
    snapshot = capture_snapshot(root)
    return NormalizedEvidenceSet(
        context=EvidenceEvaluationContext(
            evaluation_id="evaluation",
            snapshot=SnapshotBinding(
                repository_root=str(root.resolve()),
                revision=snapshot.head,
                status_fingerprint=snapshot.status_fingerprint,
                tracked_content_fingerprint=snapshot.tracked_content_fingerprint,
            ),
            source_generation=EvidenceSourceGenerationBinding(
                profile_path=generation.profile_path,
                version=generation.declared_version,
                specification_revision=generation.specification_revision,
                content_sha256=generation.content_sha256,
            ),
        ),
        records=records,
        channels=channels
        or (EvidenceChannel("candidate", EvidenceChannelStatus.PRODUCED, len(records)),),
    )


def _component(
    component_id: str,
    classification: str,
    include: tuple[str, ...],
    *,
    pointer: str = "/components/0",
) -> SourceComponent:
    return SourceComponent(
        id=component_id,
        source_classification=classification,
        include=include,
        exclude=(),
        purpose="test",
        scope=SourceDeclarationScope.TOP_LEVEL,
        location=SourceLocation("ptsip.yaml", pointer),
    )


def _v036(
    root: Path,
    components: tuple[SourceComponent, ...],
    *,
    relationships: tuple[SourceRelationship, ...] = (),
) -> CompatibilitySourceProfile:
    return CompatibilitySourceProfile(
        family=SourceFamily.TOOL_036_PROFILE,
        generation=_generation(root),
        components=components,
        associated_artifacts=(),
        relationships=relationships,
        policies=(),
        family_semantics=V036SourceSemantics("explicit"),
        raw_payload=freeze_json({}),
    )


def test_obligation_taxonomy_separates_required_removal_and_async(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"src/a.py": "a", "extra.txt": "x"})
    profile = _v036(root, (_component("core", "PRODUCT", ("src/**", "gone/**")),))
    analysis = analyze_source_migration(root, profile, _evidence(root, profile.generation))

    assert [item.path for item in analysis.required] == ["src/a.py"]
    assert [item.selector for item in analysis.removals] == ["gone/**"]
    assert {item.path for item in analysis.async_targets} == {"extra.txt", "ptsip.yaml"}


def test_required_is_unresolved_without_accepted_target_state(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"src/a.py": "a"})
    profile = _v036(root, (_component("core", "PRODUCT", ("src/**",)),))
    analysis = analyze_source_migration(root, profile, _evidence(root, profile.generation))

    assert analysis.required[0].target_status == TargetCompatibility.NOT_EVALUATED
    assert not analysis.required[0].resolved
    assert analysis.completion.required_unresolved == 1


def test_historical_toolchain_never_auto_maps_to_development_tooling(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"tools/x.py": "x"})
    generation = _generation(root, "0.3.4-draft")
    profile = CompatibilitySourceProfile(
        family=SourceFamily.TOOL_035_PROFILE,
        generation=generation,
        components=(_component("tools", "TOOLCHAIN", ("tools/**",)),),
        associated_artifacts=(),
        relationships=(),
        policies=(),
        family_semantics=V034SourceSemantics("COMPONENTS"),
        raw_payload=freeze_json({}),
    )
    target = TargetArchitectureState(
        "0.3.7-draft",
        "target-revision",
        (TargetComponent("tools", "DEVELOPMENT_TOOLING", ("tools/**",)),),
    )
    analysis = analyze_source_migration(root, profile, _evidence(root, generation), target_state=target)

    assert analysis.required[0].target_status == TargetCompatibility.TARGET_REVIEW_REQUIRED
    assert not analysis.required[0].resolved
    assert LifecycleFindingKind.HISTORICAL_TOOLCHAIN_AMBIGUITY in {item.kind for item in analysis.lifecycle_findings}


def test_exact_target_semantics_resolve_required_element(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"src/a.py": "a"})
    profile = _v036(root, (_component("core", "PRODUCT", ("src/**",)),))
    target = TargetArchitectureState(
        "0.3.7-draft",
        "target-revision",
        (TargetComponent("core", "PRODUCT", ("src/**",)),),
    )
    analysis = analyze_source_migration(root, profile, _evidence(root, profile.generation), target_state=target)

    assert analysis.required[0].target_status == TargetCompatibility.ALREADY_SATISFIED
    assert analysis.required[0].resolved
    assert analysis.completion.complete


def test_semantically_compatible_target_id_can_resolve_required_element(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"src/a.py": "a"})
    profile = _v036(root, (_component("core", "PRODUCT", ("src/**",)),))
    target = TargetArchitectureState(
        "0.3.7-draft",
        "target-revision",
        (TargetComponent("renamed", "PRODUCT", ("src/**",)),),
    )
    analysis = analyze_source_migration(root, profile, _evidence(root, profile.generation), target_state=target)

    assert analysis.required[0].target_status == TargetCompatibility.COMPATIBLE_TARGET_STATE
    assert analysis.required[0].resolved


def test_lifecycle_difference_is_conflict_not_silent_conversion(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"src/a.py": "a"})
    profile = _v036(root, (_component("core", "PRODUCT", ("src/**",)),))
    target = TargetArchitectureState(
        "0.3.7-draft",
        "target-revision",
        (TargetComponent("core", "DELIVERY", ("src/**",)),),
    )
    analysis = analyze_source_migration(root, profile, _evidence(root, profile.generation), target_state=target)

    assert analysis.required[0].target_status == TargetCompatibility.CONFLICTING_TARGET_STATE
    assert not analysis.required[0].resolved
    assert LifecycleFindingKind.POSSIBLE_LIFECYCLE_SEPARATION in {item.kind for item in analysis.lifecycle_findings}


def test_equal_specificity_source_coverage_fails_closed(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"src/a.py": "a"})
    profile = _v036(
        root,
        (
            _component("one", "PRODUCT", ("src/**",), pointer="/components/0"),
            _component("two", "PRODUCT", ("src/**",), pointer="/components/1"),
        ),
    )
    analysis = analyze_source_migration(root, profile, _evidence(root, profile.generation))

    assert len(analysis.ambiguous) == 1
    assert not analysis.valid
    assert not analysis.required


def test_conflicting_evidence_is_attached_without_changing_obligation_category(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"src/a.py": "a"})
    profile = _v036(root, (_component("core", "PRODUCT", ("src/**",)),))
    origin = EvidenceOrigin(
        provenance="OBSERVED",
        adapter="test",
        evidence_id="edge",
        source_path="src/a.py",
    )
    record = EvidenceRecord(
        semantic_id="evidence:conflict",
        subject="path:src/a.py",
        predicate="dependency",
        qualifiers={},
        assertions=(
            EvidenceAssertion("assertion:a", {"value": 1}, (origin,)),
            EvidenceAssertion("assertion:b", {"value": 2}, (origin,)),
        ),
        status=EvidenceRecordStatus.CONFLICT,
    )
    analysis = analyze_source_migration(root, profile, _evidence(root, profile.generation, (record,)))

    assert analysis.required[0].category.value == "REQUIRED"
    assert analysis.required[0].evidence.conflict_ids == ("evidence:conflict",)
    assert ArchitectureFindingKind.EVIDENCE_CONFLICT in {item.kind for item in analysis.architecture_findings}


def test_evidence_source_generation_mismatch_invalidates_analysis(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"src/a.py": "a"})
    profile = _v036(root, (_component("core", "PRODUCT", ("src/**",)),))
    evidence = _evidence(root, profile.generation)
    evidence = NormalizedEvidenceSet(
        context=EvidenceEvaluationContext(
            evaluation_id=evidence.context.evaluation_id,
            snapshot=evidence.context.snapshot,
            source_generation=EvidenceSourceGenerationBinding(
                profile_path="ptsip.yaml",
                version="0.3.4-draft",
                specification_revision=profile.generation.specification_revision,
                content_sha256=profile.generation.content_sha256,
            ),
        ),
        records=evidence.records,
        channels=evidence.channels,
    )

    analysis = analyze_source_migration(root, profile, evidence)

    assert not analysis.valid
    assert "EVIDENCE_SOURCE_CONTEXT_MISMATCH" in {item.code for item in analysis.issues}


def test_missing_source_relationship_is_reviewable_finding(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"src/a.py": "a"})
    relationship = SourceRelationship(
        id="core-reads-contract",
        source="core",
        target="contract",
        relationship_type="READS",
        scope=SourceDeclarationScope.TOP_LEVEL,
        location=SourceLocation("ptsip.yaml", "/relationships/0"),
    )
    profile = _v036(root, (_component("core", "PRODUCT", ("src/**",)),), relationships=(relationship,))
    target = TargetArchitectureState(
        "0.3.7-draft",
        "target-revision",
        (TargetComponent("core", "PRODUCT", ("src/**",)),),
    )
    analysis = analyze_source_migration(root, profile, _evidence(root, profile.generation), target_state=target)

    assert ArchitectureFindingKind.MISSING_RELATIONSHIP in {item.kind for item in analysis.architecture_findings}


def test_legacy_boundary_roots_project_to_required_elements(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"product/a.py": "a"})
    generation = _generation(root, "0.3.4-draft")
    profile = CompatibilitySourceProfile(
        family=SourceFamily.TOOL_035_PROFILE,
        generation=generation,
        components=(),
        associated_artifacts=(),
        relationships=(),
        policies=(),
        family_semantics=V034SourceSemantics(
            "BOUNDARIES",
            (SourceBoundary("PRODUCT", ("product",), SourceLocation("ptsip.yaml", "/boundaries/product")),),
        ),
        raw_payload=freeze_json({}),
    )
    analysis = analyze_source_migration(root, profile, _evidence(root, generation))

    assert [item.path for item in analysis.required] == ["product/a.py"]
    assert analysis.required[0].source_classification == "PRODUCT"


def test_failed_or_not_analyzed_evidence_channels_remain_incomplete_not_false(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"src/a.py": "a"})
    profile = _v036(root, (_component("core", "PRODUCT", ("src/**",)),))
    evidence = _evidence(
        root,
        profile.generation,
        channels=(
            EvidenceChannel("dependency", EvidenceChannelStatus.FAILED, 0, "failed"),
            EvidenceChannel("artifact", EvidenceChannelStatus.NOT_ANALYZED, 0),
        ),
    )
    analysis = analyze_source_migration(root, profile, evidence)

    assert analysis.required[0].evidence.incomplete_channels == ("artifact", "dependency")
    assert ArchitectureFindingKind.EVIDENCE_INCOMPLETE in {item.kind for item in analysis.architecture_findings}


def test_async_work_never_compensates_for_unresolved_required_work(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"src/a.py": "a", "extra.txt": "x"})
    profile = _v036(root, (_component("core", "PRODUCT", ("src/**",)),))
    analysis = analyze_source_migration(root, profile, _evidence(root, profile.generation))

    assert analysis.completion.async_count >= 1
    assert analysis.completion.required_unresolved == 1
    assert not analysis.completion.complete


def test_source_binding_change_after_wu04_invalidates_analysis(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"src/a.py": "a"})
    profile = _v036(root, (_component("core", "PRODUCT", ("src/**",)),))
    evidence = _evidence(root, profile.generation)
    (root / "ptsip.yaml").write_text("changed-after-read", encoding="utf-8")

    analysis = analyze_source_migration(root, profile, evidence)

    assert not analysis.valid
    assert "SOURCE_CONTENT_STALE" in {item.code for item in analysis.issues}


def test_target_mapping_reader_accepts_explicit_only() -> None:
    explicit = target_state_from_mapping(
        {
            "ptsip": {"version": "0.3.7-draft", "specification": {"revision": "target-revision"}},
            "responsibility_map": {"mode": "explicit"},
            "components": [{"id": "core", "classification": "PRODUCT", "include": ["src/**"]}],
        }
    )
    assert explicit.components[0].id == "core"

    with pytest.raises(ValueError, match="explicit target state only"):
        target_state_from_mapping(
            {
                "ptsip": {"version": "0.3.7-draft", "specification": {"revision": "target-revision"}},
                "responsibility_map": {"mode": "hybrid"},
            }
        )


def test_template_source_is_materialized_only_in_source_projection_stage(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"src/a.py": "a", "tests/test_a.py": "t"})
    generation = _generation(root)
    payload = {
        "ptsip": {
            "version": "0.3.6-draft",
            "specification": {"source": "https://github.com/Kinirin/PTSIP", "revision": "revision"},
        },
        "responsibility_map": {
            "mode": "template",
            "template": {
                "id": "python-package-library",
                "revision": "sha256:409acd1cd9907a60761a3cf26a051185d40b5e926e6952131b641b10bccc5c9b",
            },
        },
        "policies": {
            "product_to_nonproduct_runtime_dependency": "deny",
            "nonproduct_in_product_package": "deny",
            "independent_build_resolution": "required",
        },
    }
    profile = CompatibilitySourceProfile(
        family=SourceFamily.TOOL_036_PROFILE,
        generation=generation,
        components=(),
        associated_artifacts=(),
        relationships=(),
        policies=(),
        family_semantics=V036SourceSemantics(
            responsibility_map_mode="template",
            template_id="python-package-library",
            template_revision="sha256:409acd1cd9907a60761a3cf26a051185d40b5e926e6952131b641b10bccc5c9b",
        ),
        raw_payload=freeze_json(payload),
    )

    analysis = analyze_source_migration(root, profile, _evidence(root, generation))

    assert {item.source_declaration_id for item in analysis.required} == {"package", "package-tests"}


def test_equivalent_inputs_produce_same_deterministic_digest(tmp_path: Path) -> None:
    root = _repo(tmp_path, {"src/b.py": "b", "src/a.py": "a"})
    profile = _v036(root, (_component("core", "PRODUCT", ("src/**",)),))
    evidence = _evidence(root, profile.generation)

    first = analyze_source_migration(root, profile, evidence)
    second = analyze_source_migration(root, profile, evidence)

    assert first.deterministic_digest == second.deterministic_digest
