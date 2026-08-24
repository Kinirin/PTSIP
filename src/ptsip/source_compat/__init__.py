from .integrity import source_integrity_issues
from .model import (
    CompatibilitySourceProfile,
    SourceAssociatedArtifact,
    SourceBoundary,
    SourceComponent,
    SourceDeclarationScope,
    SourceFamily,
    SourceGenerationBinding,
    SourceLocation,
    SourcePolicy,
    SourceReadIssue,
    SourceReadResult,
    SourceRelationship,
    V034SourceSemantics,
    V036SourceSemantics,
)
from .reader import (
    V034_REVISION,
    V036_REVISION,
    read_source_profile as _read_source_profile,
    supported_source_families,
    validate_source_read_binding,
)


def read_source_profile(repository_root, generation) -> SourceReadResult:
    result = _read_source_profile(repository_root, generation)
    if result.profile is None or result.issues:
        return result
    issues = source_integrity_issues(result.profile)
    return SourceReadResult(None, issues) if issues else result


__all__ = [
    "CompatibilitySourceProfile",
    "SourceAssociatedArtifact",
    "SourceBoundary",
    "SourceComponent",
    "SourceDeclarationScope",
    "SourceFamily",
    "SourceGenerationBinding",
    "SourceLocation",
    "SourcePolicy",
    "SourceReadIssue",
    "SourceReadResult",
    "SourceRelationship",
    "V034SourceSemantics",
    "V036SourceSemantics",
    "V034_REVISION",
    "V036_REVISION",
    "read_source_profile",
    "source_integrity_issues",
    "supported_source_families",
    "validate_source_read_binding",
]
