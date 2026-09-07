"""Tracked Python target resolution. No consumer imports or environment lookup."""
from __future__ import annotations

import ast
from pathlib import Path, PurePosixPath
import tomllib


class PythonResolver:
    def __init__(self, root: Path, paths):
        self.root = root.resolve()
        self.paths = frozenset(path for path in paths if (root / path).is_file()
                               and not (root / path).is_symlink()
                               and (root / path).resolve().is_relative_to(self.root))
        self.roots = {PurePosixPath(".")}
        if any(path.startswith("src/") and path.endswith(".py") for path in self.paths):
            self.roots.add(PurePosixPath("src"))
        for path in sorted(self.paths):
            if PurePosixPath(path).name != "pyproject.toml":
                continue
            try:
                payload = tomllib.loads((root / path).read_text(encoding="utf-8-sig"))
            except (OSError, ValueError):
                continue
            setuptools = payload.get("tool", {}).get("setuptools", {})
            configured = list(setuptools.get("package-dir", {}).values())
            configured += setuptools.get("packages", {}).get("find", {}).get("where", []) if isinstance(setuptools.get("packages"), dict) else []
            for value in configured:
                if not isinstance(value, str):
                    continue
                candidate = (root / PurePosixPath(path).parent / value).resolve()
                if candidate.is_relative_to(self.root):
                    self.roots.add(PurePosixPath(candidate.relative_to(self.root).as_posix()))

    def package_root(self, source: str):
        parent = PurePosixPath(source).parent
        regular = False
        while str(parent / "__init__.py") in self.paths:
            regular = True
            parent = parent.parent
        if regular:
            return parent
        # A namespace context is supported only under an evidenced source root.
        roots = [candidate for candidate in self.roots if PurePosixPath(source).is_relative_to(candidate)
                 and PurePosixPath(source).parent != candidate]
        return max(roots, key=lambda value: len(value.parts)) if roots else None

    def _files(self, base):
        # Do not select the first of module.py and module/__init__.py.
        return {str(path) for path in (base.with_suffix(".py"), base / "__init__.py")
                if str(path) in self.paths}

    def candidates(self, module: str, source: str = ""):
        if not module or module.startswith(".") or not all(part.isidentifier() for part in module.split(".")):
            return set()
        roots = set(self.roots)
        if source:
            package_root = self.package_root(source)
            if package_root is not None:
                roots.add(package_root)
        result = set()
        for root in roots:
            result.update(self._files(root.joinpath(*module.split("."))))
        return result

    def resolve(self, module, source=""):
        candidates = self.candidates(module, source)
        return next(iter(candidates)) if len(candidates) == 1 else None

    def relative(self, source: str, node: ast.ImportFrom):
        root = self.package_root(source)
        parent = PurePosixPath(source).parent
        raw = "." * node.level + (node.module or "")
        if root is None or node.level > len(parent.relative_to(root).parts):
            return [(raw or "<relative>", None)]
        for _ in range(node.level - 1):
            parent = parent.parent
        if node.module:
            base = parent.joinpath(*node.module.split("."))
            target = ".".join(base.relative_to(root).parts)
            candidates = self._files(base)
            return [(target, next(iter(candidates)) if len(candidates) == 1 else None)]
        result = []
        for alias in node.names:
            base = parent.joinpath(*alias.name.split("."))
            candidates = self._files(base) if alias.name != "*" else set()
            target = ".".join(base.relative_to(root).parts)
            if len(candidates) == 1:
                result.append((target, next(iter(candidates))))
            elif not candidates and str(parent / "__init__.py") in self.paths:
                # Importing an attribute of an exact package still loads that
                # package implementation; do not fabricate an attribute module.
                result.append((".".join(parent.relative_to(root).parts), str(parent / "__init__.py")))
            else:
                result.append((target, None))
        return sorted(set(result))


def dynamic_targets(node: ast.Call, ancestors: list[ast.AST]):
    """Literal or lexical finite-loop allowlists only; all other flows stay unknown."""
    if not node.args:
        return "UNBOUNDED_DYNAMIC_IMPORT", []
    value = node.args[0]
    if isinstance(value, ast.Constant) and isinstance(value.value, str):
        return "STATIC_LITERAL_DYNAMIC_IMPORT", [value.value]
    def bounded(expression, name, alternatives):
        if isinstance(expression, ast.Name) and expression.id == name:
            return alternatives
        if isinstance(expression, ast.Constant) and isinstance(expression.value, str):
            return [expression.value]
        if isinstance(expression, ast.BinOp) and isinstance(expression.op, ast.Add):
            left = bounded(expression.left, name, alternatives)
            right = bounded(expression.right, name, alternatives)
            if left and right and len(left) * len(right) <= 32:
                return sorted({a + b for a in left for b in right})
        return []
    for loop in ancestors:
        if isinstance(loop, (ast.FunctionDef, ast.AsyncFunctionDef, ast.Lambda, ast.ClassDef)):
            break
        if not isinstance(loop, ast.For) or not isinstance(loop.target, ast.Name):
            continue
        if not isinstance(loop.iter, (ast.List, ast.Tuple, ast.Set)):
            continue
        alternatives = [item.value for item in loop.iter.elts if isinstance(item, ast.Constant) and isinstance(item.value, str)]
        if not alternatives or len(alternatives) != len(loop.iter.elts) or len(alternatives) > 32:
            continue
        # Reassignment, nested binding, and unknown mutation fail closed.
        if any(isinstance(item, ast.Name) and isinstance(item.ctx, ast.Store)
               and item.id == loop.target.id for statement in loop.body for item in ast.walk(statement)):
            continue
        targets = bounded(value, loop.target.id, alternatives)
        if targets:
            return "BOUNDED_DYNAMIC_IMPORT", targets
    return "UNBOUNDED_DYNAMIC_IMPORT", []
