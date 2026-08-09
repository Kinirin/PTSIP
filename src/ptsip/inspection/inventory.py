from __future__ import annotations

import ast
from collections import Counter
from dataclasses import asdict, dataclass
from pathlib import Path

EXCLUDED_DIRS = {
    ".git", ".hg", ".svn", ".venv", "venv", "env", "node_modules", "dist", "build",
    "__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache", ".tox", ".nox",
}
MANIFEST_NAMES = {
    "pyproject.toml", "setup.py", "setup.cfg", "package.json", "Cargo.toml", "go.mod",
    "pom.xml", "build.gradle", "build.gradle.kts", "requirements.txt",
}
SCHEMA_SUFFIXES = {".json", ".yaml", ".yml", ".proto", ".avsc", ".xsd"}


@dataclass(frozen=True)
class Inventory:
    files: int
    directories: int
    python_modules: int
    python_packages: int
    python_imports: int
    manifests: list[str]
    requirements_files: list[str]
    schema_candidates: list[str]
    test_roots: list[str]
    tool_like_roots: list[str]
    sdk_like_roots: list[str]
    top_level_extensions: dict[str, int]

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


def _iter_files(root: Path):
    stack = [root]
    while stack:
        current = stack.pop()
        try:
            entries = list(current.iterdir())
        except (OSError, PermissionError):
            continue
        for entry in entries:
            if entry.is_dir():
                if entry.name in EXCLUDED_DIRS:
                    continue
                yield ("dir", entry)
                stack.append(entry)
            elif entry.is_file():
                yield ("file", entry)


def _relative(root: Path, path: Path) -> str:
    return path.relative_to(root).as_posix()


def _count_imports(path: Path) -> int:
    try:
        tree = ast.parse(path.read_text(encoding="utf-8", errors="replace"))
    except (OSError, SyntaxError, UnicodeError):
        return 0
    return sum(isinstance(node, (ast.Import, ast.ImportFrom)) for node in ast.walk(tree))


def collect_inventory(root: str | Path) -> Inventory:
    root = Path(root).resolve()
    files = 0
    directories = 0
    python_modules = 0
    python_packages: set[str] = set()
    python_imports = 0
    manifests: list[str] = []
    requirements: list[str] = []
    schemas: list[str] = []
    extensions: Counter[str] = Counter()

    for kind, path in _iter_files(root):
        rel = _relative(root, path)
        if kind == "dir":
            directories += 1
            continue
        files += 1
        suffix = path.suffix.lower() or "<none>"
        extensions[suffix] += 1
        if path.suffix.lower() == ".py":
            python_modules += 1
            python_imports += _count_imports(path)
            if path.name == "__init__.py":
                python_packages.add(str(path.parent))
        if path.name in MANIFEST_NAMES or path.suffix.lower() in {".csproj", ".sln"}:
            manifests.append(rel)
        if path.name.startswith("requirements") and path.suffix.lower() in {".txt", ".in"}:
            requirements.append(rel)
        lower_parts = {part.lower() for part in path.parts}
        if path.suffix.lower() in SCHEMA_SUFFIXES and (
            "schema" in path.stem.lower() or "schemas" in lower_parts or "contracts" in lower_parts
        ):
            schemas.append(rel)

    top_dirs = [p for p in root.iterdir() if p.is_dir() and p.name not in EXCLUDED_DIRS]
    test_roots = sorted(p.name for p in top_dirs if p.name.lower() in {"test", "tests", "testing"})
    tool_roots = sorted(p.name for p in top_dirs if p.name.lower() in {"tool", "tools", "devtools", "scripts", "build", "ci"})
    sdk_roots = sorted(p.name for p in top_dirs if "sdk" in p.name.lower() or p.name.lower() in {"src", "lib", "libs", "packages"})

    return Inventory(
        files=files,
        directories=directories,
        python_modules=python_modules,
        python_packages=len(python_packages),
        python_imports=python_imports,
        manifests=sorted(manifests),
        requirements_files=sorted(set(requirements)),
        schema_candidates=sorted(schemas),
        test_roots=test_roots,
        tool_like_roots=tool_roots,
        sdk_like_roots=sdk_roots,
        top_level_extensions=dict(extensions.most_common(15)),
    )
