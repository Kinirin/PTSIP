from __future__ import annotations

from pathlib import Path


PYTHON_SOURCE_SUFFIXES = frozenset({".py"})
JAVASCRIPT_TYPESCRIPT_SOURCE_SUFFIXES = frozenset(
    {".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".mts", ".cts"}
)
GO_SOURCE_SUFFIXES = frozenset({".go"})
DOTNET_SOURCE_SUFFIXES = frozenset({".cs"})

SUPPORTED_SOURCE_SUFFIXES = frozenset().union(
    PYTHON_SOURCE_SUFFIXES,
    JAVASCRIPT_TYPESCRIPT_SOURCE_SUFFIXES,
    GO_SOURCE_SUFFIXES,
    DOTNET_SOURCE_SUFFIXES,
)

# These are executable/source ecosystems for which Tool 0.3.0 has no dependency
# adapter.  They are deliberately bounded: documentation, images, and ordinary
# configuration are not source-language coverage blockers merely by extension.
UNSUPPORTED_MANDATORY_SOURCE_SUFFIXES = frozenset(
    {
        ".c",
        ".cc",
        ".cpp",
        ".cxx",
        ".h",
        ".hh",
        ".hpp",
        ".hxx",
        ".java",
        ".kt",
        ".kts",
        ".php",
        ".rb",
        ".rs",
        ".scala",
        ".swift",
    }
)

SUPPORTED_MANIFEST_NAMES = frozenset({"pyproject.toml", "package.json", "go.mod"})
SUPPORTED_MANIFEST_SUFFIXES = frozenset({".csproj"})


def is_supported_source(path: str) -> bool:
    return Path(path).suffix.lower() in SUPPORTED_SOURCE_SUFFIXES


def is_unsupported_mandatory_source(path: str) -> bool:
    return Path(path).suffix.lower() in UNSUPPORTED_MANDATORY_SOURCE_SUFFIXES


def is_supported_manifest(path: str) -> bool:
    candidate = Path(path)
    name = candidate.name.lower()
    requirements = name.startswith("requirements") and candidate.suffix.lower() in {".txt", ".in"}
    return requirements or name in SUPPORTED_MANIFEST_NAMES or candidate.suffix.lower() in SUPPORTED_MANIFEST_SUFFIXES
