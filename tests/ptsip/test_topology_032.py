from __future__ import annotations

import json
import subprocess
from pathlib import Path

import yaml

from ptsip.clarification.resolution import (
    DecisionAnswer,
    prepare_local_profile,
    write_prepared_local_profile,
)
from ptsip.cli import main
from ptsip.constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION
from ptsip.topology import migrate_topology


LEGACY_SPEC_REVISION = "b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e"


def _run(root: Path, *args: str) -> str:
    proc = subprocess.run(
        list(args),
        cwd=root,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if proc.returncode != 0:
        raise AssertionError(proc.stderr or proc.stdout)
    return proc.stdout.strip()


def _profile(selector: str = "old/**", classification: str = "DEVELOPMENT_TOOLING") -> str:
    payload = {
        "ptsip": {
            "version": CURRENT_PROJECT_PROFILE_VERSION,
            "specification": {
                "family": SPEC_VERSION,
                "source": SPEC_SOURCE,
                "revision": SPEC_REVISION,
            },
        },
        "responsibility_map": {"mode": "explicit"},
        "components": [
            {
                "id": "sdk-tools",
                "classification": classification,
                "include": [selector],
                "purpose": "Repository SDK tooling",
                "shipped": False,
                "runtime_required": False,
                "executable": True,
            }
        ],
        "policies": {
            "product_to_nonproduct_runtime_dependency": "deny",
            "nonproduct_in_product_package": "deny",
            "independent_build_resolution": "required",
        },
    }
    return yaml.safe_dump(payload, sort_keys=False)


def _legacy_boundary_profile() -> str:
    payload = {
        "ptsip": {
            "version": "0.3.4-draft",
            "specification": {
                "source": "https://github.com/Kinirin/PTSIP",
                "revision": LEGACY_SPEC_REVISION,
            },
        },
        "boundaries": {
            "product": {"roots": ["product"]},
            "toolchain": {"roots": ["old"]},
        },
        "policies": {
            "product_to_toolchain_runtime_dependency": "deny",
            "toolchain_in_product_package": "deny",
            "independent_build_resolution": "required",
        },
    }
    return yaml.safe_dump(payload, sort_keys=False)


def _git_repo(tmp_path: Path) -> tuple[Path, Path]:
    root = tmp_path / "repo"
    root.mkdir()
    (root / "old").mkdir()
    (root / "old" / "main.py").write_text("print('tool')\n", encoding="utf-8")
    profile = root / "docs" / "sdk" / "simple-connection.ptsip.yaml"
    profile.parent.mkdir(parents=True)
    profile.write_text(_profile(), encoding="utf-8")
    (root / "pyproject.toml").write_text(
        "[tool.example]\ncomponent-root = 'old'\n",
        encoding="utf-8",
    )
    _run(root, "git", "init")
    _run(root, "git", "config", "user.email", "ptsip@example.invalid")
    _run(root, "git", "config", "user.name", "PTSIP Test")
    _run(root, "git", "add", ".")
    _run(root, "git", "commit", "-m", "fixture")
    return root, profile


def test_resolution_projection_respects_explicit_profile_path(tmp_path: Path):
    root = tmp_path / "repo"
    root.mkdir()
    (root / "tools").mkdir()
    (root / "tools" / "generate.py").write_text("print('generate')\n", encoding="utf-8")
    profile = root / "docs" / "sdk" / "project.ptsip.yaml"
    profile.parent.mkdir(parents=True)
    answer = DecisionAnswer(
        classification="DEVELOPMENT_TOOLING",
        purpose="Repository migration tooling",
        shipped=False,
        runtime_required=False,
        executable=True,
    )

    prepared = prepare_local_profile(root, "tools", ["tools/**"], answer, profile)
    written = write_prepared_local_profile(prepared)

    assert written == profile.resolve()
    assert profile.is_file()
    assert not (root / "ptsip.yaml").exists()
    payload = yaml.safe_load(profile.read_text(encoding="utf-8"))
    component = payload["components"][0]
    assert component["classification"] == "DEVELOPMENT_TOOLING"
    assert component["runtime_required"] is False
    assert "lifecycle_owner" not in component


def test_topology_dry_run_reports_impacts_without_mutation(tmp_path: Path):
    root, profile = _git_repo(tmp_path)

    result = migrate_topology(root, profile, "old", "sdk/tooling", "sdk-tools", apply=False)

    assert result["status"] == "PLAN"
    assert result["applied"] is False
    assert result["classification"]["preserved"] is True
    assert result["migration"]["classification"] == "DEVELOPMENT_TOOLING"
    assert result["profile_changes"][0]["before"] == "old/**"
    assert result["profile_changes"][0]["after"] == "sdk/tooling/**"
    assert result["reference_impacts"]["BUILD"]
    assert result["automatic_reference_rewrite"] is False
    assert result["reference_analysis"]["dependency_scan_complete"] is True
    assert (root / "old" / "main.py").is_file()
    assert not (root / "sdk" / "tooling").exists()
    assert "old/**" in profile.read_text(encoding="utf-8")
    assert _run(root, "git", "status", "--porcelain") == ""


def test_topology_dependency_edges_find_import_without_root_literal(tmp_path: Path):
    root = tmp_path / "repo"
    root.mkdir()
    (root / "src" / "sdk").mkdir(parents=True)
    (root / "src" / "sdk" / "__init__.py").write_text("", encoding="utf-8")
    (root / "src" / "sdk" / "main.py").write_text("VALUE = 1\n", encoding="utf-8")
    (root / "consumer.py").write_text("import sdk.main\n", encoding="utf-8")
    payload = yaml.safe_load(_profile("src/sdk/**"))
    payload["components"].append(
        {
            "id": "consumer",
            "classification": "PRODUCT",
            "include": ["consumer.py"],
            "purpose": "Product consumer",
            "shipped": True,
            "runtime_required": True,
            "executable": True,
        }
    )
    profile = root / "ptsip.yaml"
    profile.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")
    _run(root, "git", "init")
    _run(root, "git", "config", "user.email", "ptsip@example.invalid")
    _run(root, "git", "config", "user.name", "PTSIP Test")
    _run(root, "git", "add", ".")
    _run(root, "git", "commit", "-m", "fixture")

    result = migrate_topology(root, profile, "src/sdk", "src/tooling/sdk", "sdk-tools", apply=False)

    imports = result["reference_impacts"]["IMPORT"]
    assert any(
        item["path"] == "consumer.py" and "IMPORTS sdk.main -> src/sdk/main.py" in item["text"]
        for item in imports
    )
    assert result["reference_analysis"]["dependency_edge_count"] >= 1


def test_topology_cli_uses_explicit_profile_path(tmp_path: Path, capsys):
    root, profile = _git_repo(tmp_path)

    assert (
        main(
            [
                "topology",
                str(root),
                "--profile",
                str(profile),
                "--component",
                "sdk-tools",
                "--from",
                "old",
                "--to",
                "sdk/tooling",
                "--json",
            ]
        )
        == 0
    )
    payload = json.loads(capsys.readouterr().out)
    assert payload["status"] == "PLAN"
    assert payload["profile_path"] == str(profile.resolve())
    assert payload["classification"]["preserved"] is True


def test_topology_apply_moves_root_and_preserves_classification(tmp_path: Path):
    root, profile = _git_repo(tmp_path)
    (root / "pyproject.toml").write_text("[tool.example]\n", encoding="utf-8")
    _run(root, "git", "add", "pyproject.toml")
    _run(root, "git", "commit", "-m", "remove reference")

    result = migrate_topology(root, profile, "old", "sdk/tooling", "sdk-tools", apply=True)

    assert result["status"] == "APPLIED"
    assert result["move_method"] == "git-mv"
    assert result["git_index_updated"] is True
    assert not (root / "old").exists()
    assert (root / "sdk" / "tooling" / "main.py").is_file()
    payload = yaml.safe_load(profile.read_text(encoding="utf-8"))
    component = payload["components"][0]
    assert component["classification"] == "DEVELOPMENT_TOOLING"
    assert component["include"] == ["sdk/tooling/**"]
    assert component["runtime_required"] is False
    assert "lifecycle_owner" not in component
    staged = _run(root, "git", "diff", "--cached", "--name-status")
    assert "docs/sdk/simple-connection.ptsip.yaml" in staged
    assert "sdk/tooling/main.py" in staged


def test_topology_legacy_boundary_profile_is_rejected_by_canonical_036_topology(tmp_path: Path):
    root = tmp_path / "repo"
    root.mkdir()
    (root / "product").mkdir()
    (root / "product" / "app.py").write_text("VALUE = 1\n", encoding="utf-8")
    (root / "old").mkdir()
    (root / "old" / "tool.py").write_text("VALUE = 2\n", encoding="utf-8")
    profile = root / "ptsip.yaml"
    legacy_profile = _legacy_boundary_profile()
    profile.write_text(legacy_profile, encoding="utf-8")
    _run(root, "git", "init")
    _run(root, "git", "config", "user.email", "ptsip@example.invalid")
    _run(root, "git", "config", "user.name", "PTSIP Test")
    _run(root, "git", "add", ".")
    _run(root, "git", "commit", "-m", "fixture")

    try:
        migrate_topology(root, profile, "old", "devtools", apply=False)
    except ValueError as exc:
        message = str(exc)
        assert "Current PTSIP profile is invalid" in message
        assert "responsibility_map" in message
        assert "0.3.6-draft" in message
    else:
        raise AssertionError("canonical Tool 0.3.6 topology must reject a legacy boundary profile")

    assert profile.read_text(encoding="utf-8") == legacy_profile
    assert (root / "old" / "tool.py").is_file()
    assert not (root / "devtools").exists()


def test_topology_apply_requires_clean_git_state(tmp_path: Path):
    root, profile = _git_repo(tmp_path)
    (root / "old" / "main.py").write_text("print('changed')\n", encoding="utf-8")

    try:
        migrate_topology(root, profile, "old", "sdk/tooling", "sdk-tools", apply=True)
    except RuntimeError as exc:
        assert "clean Git working tree" in str(exc)
    else:
        raise AssertionError("dirty repository must block topology --apply")


def test_topology_rejects_classification_ambiguous_component_selection(tmp_path: Path):
    root, profile = _git_repo(tmp_path)

    try:
        migrate_topology(root, profile, "old", "sdk/tooling", None, apply=False)
    except ValueError as exc:
        assert "--component is required" in str(exc)
    else:
        raise AssertionError("component ownership migration must identify its component")


def test_topology_requires_profile_inside_repository(tmp_path: Path):
    root, _profile_path = _git_repo(tmp_path)
    outside = tmp_path / "outside.ptsip.yaml"
    outside.write_text(_profile(), encoding="utf-8")

    try:
        migrate_topology(root, outside, "old", "sdk/tooling", "sdk-tools", apply=False)
    except ValueError as exc:
        assert "profile to be inside the repository" in str(exc)
    else:
        raise AssertionError("topology migration must not rewrite an external profile")
