from __future__ import annotations

import json
from pathlib import Path

import yaml

from ptsip.adoption import apply_adoption, prepare_adoption
from ptsip.clarification.resolution import DecisionAnswer
from ptsip.cli import main
from ptsip.constants import SPEC_REVISION
from ptsip.storage.local_state import decision_store_path
from _wu04g_support import commit_all, git, init_git_repo, write_text


def _repo(tmp_path: Path) -> Path:
    repo = init_git_repo(tmp_path / "repo")
    write_text(repo, "tools/generate.py", "print('generate')\n")
    commit_all(repo)
    return repo


def _facts(classification: str) -> tuple[str, str, str, str, str]:
    if classification == "PRODUCT":
        return ("Product runtime component", "yes", "yes", "PRODUCT", "yes")
    if classification == "NEUTRAL_CONTRACT":
        return ("Shared declarative contract", "no", "no", "INDEPENDENT", "no")
    if classification == "DELIVERY":
        return ("Release delivery automation", "no", "no", "DELIVERY", "yes")
    if classification == "OPERATIONS":
        return ("Production maintenance automation", "no", "no", "OPERATIONS", "yes")
    return ("Repository-local generation tooling", "no", "no", "DEVELOPMENT_TOOLING", "yes")


def _adopt_args(
    repo: Path,
    *,
    apply: bool = False,
    profile: Path | None = None,
    classification: str = "DEVELOPMENT_TOOLING",
) -> list[str]:
    purpose, shipped, runtime_required, lifecycle_owner, executable = _facts(classification)
    args = [
        "adopt", str(repo), "--component", "tools",
        "--classification", classification,
        "--purpose", purpose,
        "--shipped", shipped,
        "--runtime-required", runtime_required,
        "--lifecycle-owner", lifecycle_owner,
        "--executable", executable,
        "--coordination", "local",
        "--json",
    ]
    if profile is not None:
        args.extend(["--profile", str(profile)])
    if apply:
        args.append("--apply")
    return args


def _profile_header() -> str:
    return f"""ptsip:
  version: 0.3.6-draft
  specification:
    source: https://github.com/Kinirin/PTSIP
    revision: {SPEC_REVISION}
responsibility_map:
  mode: explicit
"""


def _policies() -> str:
    return """policies:
  product_to_nonproduct_runtime_dependency: deny
  nonproduct_in_product_package: deny
  independent_build_resolution: required
"""


def test_adopt_is_dry_run_by_default_and_apply_persists_canonical_036_declaration(
    tmp_path: Path, monkeypatch, capsys,
) -> None:
    repo = _repo(tmp_path)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))

    before = git(repo, "status", "--porcelain").stdout
    assert before == ""
    assert main(_adopt_args(repo)) == 0
    plan = json.loads(capsys.readouterr().out)
    assert plan["format"] == "ptsip-adoption/v1"
    assert plan["status"] == "ADOPTION_PLAN"
    assert plan["apply"] is False
    assert plan["backend"] == "LOCAL"
    assert not (repo / "ptsip.yaml").exists()
    assert git(repo, "status", "--porcelain").stdout == before

    assert main(_adopt_args(repo, apply=True)) == 0
    adopted = json.loads(capsys.readouterr().out)
    assert adopted["status"] == "ADOPTED"
    profile = repo / "ptsip.yaml"
    document = yaml.safe_load(profile.read_text(encoding="utf-8"))
    assert document["ptsip"]["version"] == "0.3.6-draft"
    assert document["ptsip"]["specification"]["revision"] == SPEC_REVISION
    assert document["responsibility_map"] == {"mode": "explicit"}
    component = next(item for item in document["components"] if item["id"] == "tools")
    assert component == {
        "id": "tools", "include": ["tools/**"], "classification": "DEVELOPMENT_TOOLING",
        "purpose": "Repository-local generation tooling", "shipped": False,
        "runtime_required": False, "executable": True,
    }
    assert "lifecycle_owner" not in component
    assert not decision_store_path(repo).exists()

    assert main(_adopt_args(repo, apply=True)) == 0
    repeated = json.loads(capsys.readouterr().out)
    assert repeated["status"] == "ALREADY_DECLARED"


def test_adopt_does_not_classify_from_tools_directory_name(tmp_path: Path, capsys) -> None:
    for classification in ("PRODUCT", "DEVELOPMENT_TOOLING", "DELIVERY", "OPERATIONS", "NEUTRAL_CONTRACT"):
        repo = _repo(tmp_path / classification.lower())
        assert main(_adopt_args(repo, apply=True, classification=classification)) == 0
        result = json.loads(capsys.readouterr().out)
        assert result["status"] == "ADOPTED"
        profile = yaml.safe_load((repo / "ptsip.yaml").read_text(encoding="utf-8"))
        component = profile["components"][0]
        assert component["classification"] == classification
        assert "lifecycle_owner" not in component


def test_adopt_extends_existing_covering_component_without_creating_duplicate(tmp_path: Path, capsys) -> None:
    repo = _repo(tmp_path)
    profile = repo / "ptsip.yaml"
    profile.write_text(
        _profile_header()
        + """components:
  - id: generator-sdk
    classification: DEVELOPMENT_TOOLING
    include: ["tools/**"]
    purpose: Repository-local generation tooling
"""
        + _policies(), encoding="utf-8",
    )

    assert main(_adopt_args(repo, apply=True)) == 0
    result = json.loads(capsys.readouterr().out)
    assert result["status"] == "ADOPTED"
    document = yaml.safe_load(profile.read_text(encoding="utf-8"))
    assert [item["id"] for item in document["components"]] == ["generator-sdk"]
    component = document["components"][0]
    assert component["classification"] == "DEVELOPMENT_TOOLING"
    assert component["shipped"] is False
    assert component["runtime_required"] is False
    assert component["executable"] is True
    assert "lifecycle_owner" not in component


def test_existing_conflicting_declaration_is_not_overwritten(tmp_path: Path, capsys) -> None:
    repo = _repo(tmp_path)
    profile = repo / "ptsip.yaml"
    profile.write_text(
        _profile_header()
        + """components:
  - id: tools
    classification: PRODUCT
    include: ["tools/**"]
    purpose: Existing product component
"""
        + _policies(), encoding="utf-8",
    )
    before = profile.read_text(encoding="utf-8")
    assert main(_adopt_args(repo, apply=True)) == 8
    result = json.loads(capsys.readouterr().out)
    assert result["status"] == "CONFLICT"
    assert profile.read_text(encoding="utf-8") == before


def test_direct_adoption_refuses_legacy_boundary_profile_and_leaves_it_unchanged(tmp_path: Path, capsys) -> None:
    repo = _repo(tmp_path)
    profile = repo / "ptsip.yaml"
    profile.write_text(
        """ptsip:
  version: 0.3.4-draft
  specification:
    source: https://github.com/Kinirin/PTSIP
    revision: b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e
boundaries:
  product:
    roots: ["product"]
  toolchain:
    roots: ["tools"]
policies:
  product_to_toolchain_runtime_dependency: deny
  toolchain_in_product_package: deny
  independent_build_resolution: required
""", encoding="utf-8",
    )
    before = profile.read_text(encoding="utf-8")
    assert main(_adopt_args(repo, apply=True)) == 8
    result = json.loads(capsys.readouterr().out)
    assert result["status"] in {"CONFLICT", "DECISION_ERROR"}
    assert profile.read_text(encoding="utf-8") == before


def test_adoption_application_refuses_stale_repository_evidence(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    answer = DecisionAnswer(
        classification="DEVELOPMENT_TOOLING", purpose="Repository-local generation tooling",
        shipped=False, runtime_required=False, lifecycle_owner="DEVELOPMENT_TOOLING", executable=True,
    )
    preparation = prepare_adoption(repo, "tools", answer)
    assert preparation.status == "ADOPTION_PLAN"
    write_text(repo, "tools/generate.py", "print('changed')\n")
    status, profile_path, message = apply_adoption(preparation)
    assert status == "STALE_EVIDENCE"
    assert profile_path is not None
    assert message is not None
    assert not (repo / "ptsip.yaml").exists()


def test_adopt_explicit_profile_is_seen_by_clarify_and_gate(tmp_path: Path, monkeypatch, capsys) -> None:
    repo = _repo(tmp_path)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))
    profile = repo / "config" / "ptsip.yaml"

    assert main(_adopt_args(repo, apply=True, profile=profile)) == 0
    capsys.readouterr()
    assert profile.is_file()
    assert not (repo / "ptsip.yaml").exists()

    assert main(["clarify", str(repo), "--component", "tools", "--profile", str(profile), "--json"]) == 0
    clarified = json.loads(capsys.readouterr().out)
    assert clarified["status"] == "NO_CLARIFICATION_REQUIRED"
    assert clarified["clarification_count"] == 0
    assert Path(str(clarified["profile"]["path"])) == profile.resolve()

    assert main([
        "gate", str(repo), "--component", "tools", "--profile", str(profile),
        "--coordination", "local", "--json",
    ]) == 0
    gated = json.loads(capsys.readouterr().out)
    assert gated["status"] == "NO_DECISION_REQUIRED"
    assert gated["backend"] == "LOCAL"
    assert not decision_store_path(repo).exists()


def test_invalid_or_unknown_adoption_never_writes_profile(tmp_path: Path, capsys) -> None:
    repo = _repo(tmp_path)
    invalid = _adopt_args(repo, apply=True)
    invalid[invalid.index("--runtime-required") + 1] = "yes"
    assert main(invalid) == 8
    conflict = json.loads(capsys.readouterr().out)
    assert conflict["status"] == "CONFLICT"
    assert not (repo / "ptsip.yaml").exists()

    unknown = _adopt_args(repo, apply=True)
    unknown[unknown.index("tools")] = "does-not-exist"
    assert main(unknown) == 8
    missing = json.loads(capsys.readouterr().out)
    assert missing["status"] == "UNKNOWN_COMPONENT"
    assert not (repo / "ptsip.yaml").exists()
