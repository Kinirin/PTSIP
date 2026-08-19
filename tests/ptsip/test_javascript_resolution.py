from __future__ import annotations

import json
from pathlib import Path

from ptsip.inspection.dependencies import DependencyScan
from ptsip.inspection.javascript import discover_npm_packages, source_edges
from ptsip.model import EvidenceNodeScope, ResolutionStatus
from ptsip.validation.components import ComponentAssignment, ComponentPartition
from ptsip.validation.rules import evaluate_declared_dependency_boundaries


def _edge(root: Path, source_rel: str, target: str):
    paths = [path.relative_to(root).as_posix() for path in root.rglob("*") if path.is_file()]
    by_path, by_name, issues = discover_npm_packages(root, paths)
    assert issues == []
    edges, source_issues = source_edges(root, source_rel, by_path, by_name)
    assert source_issues == []
    return next(item for item in edges if item.target == target)


def test_node_namespace_is_platform_dependency(tmp_path: Path) -> None:
    source = tmp_path / "src" / "main.ts"
    source.parent.mkdir(parents=True)
    source.write_text('import { readFile } from "node:fs";\n', encoding="utf-8")

    edge = _edge(tmp_path, "src/main.ts", "node:fs")

    assert edge.resolution == ResolutionStatus.EXTERNAL
    assert edge.target_scope == EvidenceNodeScope.PLATFORM
    assert edge.resolved_path is None
    assert "Node.js built-in module namespace" in str(edge.note)


def test_unknown_bare_package_remains_unresolved(tmp_path: Path) -> None:
    source = tmp_path / "src" / "main.ts"
    source.parent.mkdir(parents=True)
    source.write_text('import value from "not-declared-anywhere";\n', encoding="utf-8")

    edge = _edge(tmp_path, "src/main.ts", "not-declared-anywhere")

    assert edge.resolution == ResolutionStatus.UNRESOLVED
    assert edge.target_scope == EvidenceNodeScope.UNRESOLVED_TARGET


def test_runtime_js_specifier_resolves_typescript_source(tmp_path: Path) -> None:
    source = tmp_path / "src" / "main.ts"
    target = tmp_path / "src" / "reconciliation-control-plane.ts"
    source.parent.mkdir(parents=True)
    source.write_text(
        'const module = await import("./reconciliation-control-plane.js");\n',
        encoding="utf-8",
    )
    target.write_text("export const value = 1;\n", encoding="utf-8")

    edge = _edge(tmp_path, "src/main.ts", "./reconciliation-control-plane.js")

    assert edge.resolution == ResolutionStatus.RESOLVED
    assert edge.target_scope == EvidenceNodeScope.PROJECT_COMPONENT
    assert edge.resolved_path == "src/reconciliation-control-plane.ts"


def test_existing_runtime_js_file_precedes_typescript_source_substitution(tmp_path: Path) -> None:
    source = tmp_path / "src" / "main.ts"
    javascript_target = tmp_path / "src" / "dependency.js"
    typescript_target = tmp_path / "src" / "dependency.ts"
    source.parent.mkdir(parents=True)
    source.write_text('import value from "./dependency.js";\n', encoding="utf-8")
    javascript_target.write_text("export default 'js';\n", encoding="utf-8")
    typescript_target.write_text("export default 'ts';\n", encoding="utf-8")

    edge = _edge(tmp_path, "src/main.ts", "./dependency.js")

    assert edge.resolution == ResolutionStatus.RESOLVED
    assert edge.resolved_path == "src/dependency.js"


def test_tsconfig_wildcard_path_alias_resolves_repository_target(tmp_path: Path) -> None:
    (tmp_path / "tsconfig.json").write_text(
        """{
  // JSONC comments and trailing commas are accepted.
  "compilerOptions": {
    "paths": {
      "@shared/*": ["./src/shared/*"],
    },
  },
}
""",
        encoding="utf-8",
    )
    source = tmp_path / "src" / "main.ts"
    target = tmp_path / "src" / "shared" / "types.ts"
    target.parent.mkdir(parents=True)
    source.write_text('import type { Thing } from "@shared/types";\n', encoding="utf-8")
    target.write_text("export type Thing = string;\n", encoding="utf-8")

    edge = _edge(tmp_path, "src/main.ts", "@shared/types")

    assert edge.resolution == ResolutionStatus.RESOLVED
    assert edge.target_scope == EvidenceNodeScope.PROJECT_COMPONENT
    assert edge.resolved_path == "src/shared/types.ts"


def test_tsconfig_exact_path_alias_resolves_repository_target(tmp_path: Path) -> None:
    (tmp_path / "tsconfig.json").write_text(
        json.dumps({"compilerOptions": {"paths": {"@config": ["./src/config.ts"]}}}),
        encoding="utf-8",
    )
    source = tmp_path / "src" / "main.ts"
    target = tmp_path / "src" / "config.ts"
    source.parent.mkdir(parents=True)
    source.write_text('import config from "@config";\n', encoding="utf-8")
    target.write_text("export default {};\n", encoding="utf-8")

    edge = _edge(tmp_path, "src/main.ts", "@config")

    assert edge.resolution == ResolutionStatus.RESOLVED
    assert edge.resolved_path == "src/config.ts"


def test_local_extends_preserves_parent_path_alias_origin(tmp_path: Path) -> None:
    (tmp_path / "tsconfig.base.json").write_text(
        json.dumps({"compilerOptions": {"paths": {"@shared/*": ["./shared/*"]}}}),
        encoding="utf-8",
    )
    app = tmp_path / "app"
    source = app / "src" / "main.ts"
    target = tmp_path / "shared" / "types.ts"
    source.parent.mkdir(parents=True)
    target.parent.mkdir(parents=True)
    (app / "tsconfig.json").write_text(json.dumps({"extends": "../tsconfig.base.json"}), encoding="utf-8")
    source.write_text('import type { Thing } from "@shared/types";\n', encoding="utf-8")
    target.write_text("export type Thing = string;\n", encoding="utf-8")

    edge = _edge(tmp_path, "app/src/main.ts", "@shared/types")

    assert edge.resolution == ResolutionStatus.RESOLVED
    assert edge.resolved_path == "shared/types.ts"


def test_child_paths_override_inherited_paths(tmp_path: Path) -> None:
    (tmp_path / "tsconfig.base.json").write_text(
        json.dumps({"compilerOptions": {"paths": {"@shared/*": ["./parent/*"]}}}),
        encoding="utf-8",
    )
    app = tmp_path / "app"
    source = app / "src" / "main.ts"
    target = app / "local" / "types.ts"
    source.parent.mkdir(parents=True)
    target.parent.mkdir(parents=True)
    (app / "tsconfig.json").write_text(
        json.dumps(
            {
                "extends": "../tsconfig.base.json",
                "compilerOptions": {"paths": {"@shared/*": ["./local/*"]}},
            }
        ),
        encoding="utf-8",
    )
    source.write_text('import type { Thing } from "@shared/types";\n', encoding="utf-8")
    target.write_text("export type Thing = string;\n", encoding="utf-8")

    edge = _edge(tmp_path, "app/src/main.ts", "@shared/types")

    assert edge.resolution == ResolutionStatus.RESOLVED
    assert edge.resolved_path == "app/local/types.ts"


def test_missing_alias_target_remains_unresolved(tmp_path: Path) -> None:
    (tmp_path / "tsconfig.json").write_text(
        json.dumps({"compilerOptions": {"paths": {"@shared/*": ["./src/shared/*"]}}}),
        encoding="utf-8",
    )
    source = tmp_path / "src" / "main.ts"
    source.parent.mkdir(parents=True)
    source.write_text('import value from "@shared/missing";\n', encoding="utf-8")

    edge = _edge(tmp_path, "src/main.ts", "@shared/missing")

    assert edge.resolution == ResolutionStatus.UNRESOLVED
    assert edge.target_scope == EvidenceNodeScope.UNRESOLVED_TARGET
    assert "alias matched" in str(edge.note)


def test_alias_cannot_escape_repository(tmp_path: Path) -> None:
    external = tmp_path.parent / "outside.ts"
    external.write_text("export default 1;\n", encoding="utf-8")
    (tmp_path / "tsconfig.json").write_text(
        json.dumps({"compilerOptions": {"paths": {"@outside": ["../outside.ts"]}}}),
        encoding="utf-8",
    )
    source = tmp_path / "src" / "main.ts"
    source.parent.mkdir(parents=True)
    source.write_text('import value from "@outside";\n', encoding="utf-8")

    edge = _edge(tmp_path, "src/main.ts", "@outside")

    assert edge.resolution == ResolutionStatus.UNRESOLVED
    assert edge.target_scope == EvidenceNodeScope.UNRESOLVED_TARGET


def test_ambiguous_alias_targets_fail_closed(tmp_path: Path) -> None:
    (tmp_path / "tsconfig.json").write_text(
        json.dumps({"compilerOptions": {"paths": {"@shared/*": ["./src/a/*", "./src/b/*"]}}}),
        encoding="utf-8",
    )
    source = tmp_path / "src" / "main.ts"
    first = tmp_path / "src" / "a" / "types.ts"
    second = tmp_path / "src" / "b" / "types.ts"
    source.parent.mkdir(parents=True)
    first.parent.mkdir(parents=True)
    second.parent.mkdir(parents=True)
    source.write_text('import type { Thing } from "@shared/types";\n', encoding="utf-8")
    first.write_text("export type Thing = string;\n", encoding="utf-8")
    second.write_text("export type Thing = string;\n", encoding="utf-8")

    edge = _edge(tmp_path, "src/main.ts", "@shared/types")

    assert edge.resolution == ResolutionStatus.UNRESOLVED
    assert edge.target_scope == EvidenceNodeScope.UNRESOLVED_TARGET
    assert "multiple repository targets" in str(edge.note)


def test_declared_npm_dependency_behavior_is_unchanged(tmp_path: Path) -> None:
    (tmp_path / "package.json").write_text(
        json.dumps({"dependencies": {"react": "1.0.0"}}),
        encoding="utf-8",
    )
    source = tmp_path / "src" / "main.ts"
    source.parent.mkdir(parents=True)
    source.write_text('import React from "react";\n', encoding="utf-8")

    edge = _edge(tmp_path, "src/main.ts", "react")

    assert edge.resolution == ResolutionStatus.EXTERNAL
    assert edge.target_scope == EvidenceNodeScope.EXTERNAL_DEPENDENCY


def test_alias_cross_component_edge_reaches_boundary_evaluator(tmp_path: Path) -> None:
    (tmp_path / "tsconfig.json").write_text(
        json.dumps({"compilerOptions": {"paths": {"@tools/*": ["./tools/*"]}}}),
        encoding="utf-8",
    )
    product = tmp_path / "product" / "main.ts"
    tool = tmp_path / "tools" / "check.ts"
    product.parent.mkdir(parents=True)
    tool.parent.mkdir(parents=True)
    product.write_text('import { check } from "@tools/check";\n', encoding="utf-8")
    tool.write_text("export const check = 1;\n", encoding="utf-8")

    edge = _edge(tmp_path, "product/main.ts", "@tools/check")
    partition = ComponentPartition(
        assignments=(
            ComponentAssignment("product/main.ts", "product", "product/**"),
            ComponentAssignment("tools/check.ts", "tools", "tools/**"),
        ),
        conflicts=(),
        unmatched_selectors=(),
        unassigned_files=("tsconfig.json",),
        scan_errors=(),
    )
    scan = DependencyScan(edges=(edge,), issues=(), adapters=("javascript-typescript",))
    findings = evaluate_declared_dependency_boundaries(
        [
            {"id": "product", "classification": "PRODUCT"},
            {"id": "tools", "classification": "TOOLCHAIN"},
        ],
        partition,
        scan,
    )

    assert len(findings) == 1
    assert findings[0].rule_id == "PTSIP-DEP-001"
    assert findings[0].severity == "REVIEW"
    assert findings[0].source_component == "product"
    assert findings[0].target_component == "tools"
