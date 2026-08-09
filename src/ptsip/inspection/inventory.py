from __future__ import annotations

import ast
import tokenize
from collections import Counter
from dataclasses import asdict, dataclass
from pathlib import Path

from ..repository.snapshot import repository_files

MANIFEST_NAMES = {
    "pyproject.toml", "setup.py", "setup.cfg", "package.json", "Cargo.toml", "go.mod",
    "pom.xml", "build.gradle", "build.gradle.kts", "requirements.txt",
}
MANIFEST_SUFFIXES = {".csproj", ".sln", ".slnx"}
SCHEMA_SUFFIXES = {".json", ".yaml", ".yml", ".proto", ".avsc", ".xsd"}


@dataclass(frozen=True)
class ScanIssue:
    category: str
    path: str
    message: str

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


@dataclass(frozen=True)
class Inventory:
    scan_mode: str
    expected_files: int
    scanned_files: int
    files: int
    directories: int
    symlinks: int
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
    scan_issues: list[ScanIssue]

    @property
    def coverage_complete(self) -> bool:
        return self.expected_files == self.scanned_files and not self.scan_issues

    def as_dict(self) -> dict[str, object]:
        payload = asdict(self)
        payload["coverage_complete"] = self.coverage_complete
        return payload


def _relative(path: str) -> str:
    return Path(path).as_posix()


def _count_imports(path: Path) -> tuple[int, ScanIssue | None]:
    try:
        with tokenize.open(path) as handle:
            source = handle.read()
    except (OSError, UnicodeError, SyntaxError) as exc:
        return 0, ScanIssue("READ_ERROR", path.as_posix(), str(exc))
    try:
        tree = ast.parse(source, filename=str(path))
    except SyntaxError as exc:
        return 0, ScanIssue("PYTHON_PARSE_ERROR", path.as_posix(), f"line {exc.lineno}: {exc.msg}")
    return sum(isinstance(node, (ast.Import, ast.ImportFrom)) for node in ast.walk(tree)), None


def collect_inventory(root: str | Path) -> Inventory:
    root = Path(root).resolve()
    scan_mode, repository_paths, discovery_errors = repository_files(root)
    issues: list[ScanIssue] = [ScanIssue("DISCOVERY_ERROR", "<repository>", item) for item in discovery_errors]
    files = 0
    symlinks = 0
    python_modules = 0
    python_packages: set[str] = set()
    python_imports = 0
    manifests: list[str] = []
    requirements: list[str] = []
    schemas: list[str] = []
    extensions: Counter[str] = Counter()
    directories: set[str] = set()
    top_dirs: set[str] = set()
    scanned = 0

    for rel_text in repository_paths:
        rel = _relative(rel_text)
        path = root / rel_text
        try:
            if not path.exists() and not path.is_symlink():
                issues.append(ScanIssue("MISSING_DURING_SCAN", rel, "Path disappeared during scan"))
                continue
            if path.is_symlink():
                symlinks += 1
            elif not path.is_file():
                issues.append(ScanIssue("UNSUPPORTED_ENTRY", rel, "Tracked entry is not a regular file or symlink"))
                continue
        except OSError as exc:
            issues.append(ScanIssue("STAT_ERROR", rel, str(exc)))
            continue

        scanned += 1
        files += 1
        parts = Path(rel).parts
        if parts:
            top_dirs.add(parts[0])
        parent = Path(rel).parent
        while parent.as_posix() not in {".", ""}:
            directories.add(parent.as_posix())
            parent = parent.parent

        suffix = path.suffix.lower() or "<none>"
        extensions[suffix] += 1
        if path.suffix.lower() == ".py" and not path.is_symlink():
            python_modules += 1
            import_count, issue = _count_imports(path)
            python_imports += import_count
            if issue:
                issues.append(ScanIssue(issue.category, rel, issue.message))
            if path.name == "__init__.py":
                python_packages.add(Path(rel).parent.as_posix())
        if path.name in MANIFEST_NAMES or path.suffix.lower() in MANIFEST_SUFFIXES:
            manifests.append(rel)
        if path.name.startswith("requirements") and path.suffix.lower() in {".txt", ".in"}:
            requirements.append(rel)
        lower_parts = {part.lower() for part in Path(rel).parts}
        if path.suffix.lower() in SCHEMA_SUFFIXES and (
            "schema" in path.stem.lower() or "schemas" in lower_parts or "contracts" in lower_parts
        ):
            schemas.append(rel)

    test_roots = sorted(name for name in top_dirs if name.lower() in {"test", "tests", "testing"})
    tool_roots = sorted(name for name in top_dirs if name.lower() in {"tool", "tools", "devtools", "scripts", "build", "ci"})
    sdk_roots = sorted(name for name in top_dirs if "sdk" in name.lower() or name.lower() in {"src", "lib", "libs", "packages"})

    return Inventory(
        scan_mode=scan_mode,
        expected_files=len(repository_paths),
        scanned_files=scanned,
        files=files,
        directories=len(directories),
        symlinks=symlinks,
        python_modules=python_modules,
        python_packages=len(python_packages),
        python_imports=python_imports,
        manifests=sorted(set(manifests)),
        requirements_files=sorted(set(requirements)),
        schema_candidates=sorted(set(schemas)),
        test_roots=test_roots,
        tool_like_roots=tool_roots,
        sdk_like_roots=sdk_roots,
        top_level_extensions=dict(extensions.most_common(20)),
        scan_issues=issues,
    )
