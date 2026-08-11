from __future__ import annotations

import json
import subprocess
from pathlib import Path

import yaml

from ptsip.adoption import apply_adoption, prepare_adoption
from ptsip.clarification.resolution import DecisionAnswer
from ptsip.cli import main
from ptsip.storage.local_state import decision_store_path


SPEC_REVISION = "ccee8cd5e26e92d31a2b93a86157c03d9b796b2c"


def _git(repo: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


def _repo(tmp_path: Path) -> Path:
    repo = tmp_path / "repo"
    repo.mkdir(parents=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")
    (repo / "tools").mkdir()
    (repo / "tools" / "generate.py").write_text("print('generate')\n", encoding="utf-8")
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")
    return repo


def _adopt_args(
    repo: Path,
    *,
    apply: bool = False,
    profile: Path | None = None,
    classification: str = "TOOLCHAIN",
) -> list[str]:
    if classification == "PRODUCT":
        purpose = "Product runtime component"
        shipped = "yes"
        runtime_required = "yes"
        lifecycle_owner = "PRODUCT"
        executable = "yes"
    elif classification == "NEUTRAL_CONTRACT":
        purpose = "Shared declarative contract"
        shipped = "no"
        runtime_required = "no"
        lifecycle_owner = "INDEPENDENT"
        executable = "no"
    else:
        purpose = "Repository-local generation tooling"
        shipped = "no"
        runtime_required = "no"
        lifecycle_owner = "DEVELOPMENT_TOOLING"
        executable = "yes"

    args = [
        "adopt",
        str(repo),
        "--component",
        "tools",
        "--classification",
        classification,
        "--purpose",
        purpose,
        "--shipped",
        shipped,
        "--runtime-required",
        runtime_required,
        "--lifecycle-owner",
        lifecycle_owner,
        "--executable",
        executable,
        "--coordination",
        "local",
        "--json",
    ]
    if profile is not None:
        args.extend(["--profile", str(profile)])
    if apply:
        args.append("--apply")
    return args


def test_adopt_is_dry_run_by_default_and_apply_persists_bound_declaration(
    tmp_path: Path,
    monkeypatch,
    capsys,
) -> None:
    repo = _repo(tmp_path)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))

    before = _git(repo, "status", "--porcelain").stdout
    assert before == ""
    assert main(_adopt_args(repo)) == 0
    plan = json.loads(capsys.readouterr().out)
    assert plan["format"] == "ptsip-adoption/v1"
    assert plan["status"] == "ADOPTION_PLAN"
    assert plan["apply"] is False
    assert plan["backend"] == "LOCAL"
    assert not (repo / "ptsip.yaml").exists()
    assert _git(repo, "status", "--porcelain").stdout == before

    assert main(_adopt_args(repo, apply=True)) == 0
    adopted = json.loads(capsys.readouterr().out)
    assert adopted["status"] == "ADOPTED"
    assert adopted["declaration"]["runtime_required"] is False
    profile = repo / "ptsip.yaml"
    document = yaml.safe_load(profile.read_text(encoding="utf-8"))
    assert document["ptsip"]["version"] == "0.3.4-draft"
    assert document["ptsip"]["specification"]["revision"] == SPEC_REVISION
    component = next(item for item in document["components"] if item["id"] == "tools")
    assert component == {
        "id": "tools",
        "include": ["tools/**"],
        "classification": "TOOLCHAIN",
        "purpose": "Repository-local generation tooling",
        "shipped": False,
        "runtime_required": False,
        "lifecycle_owner": "DEVELOPMENT_TOOLING",
        "executable": True,
    }
    assert "release_owner" not in component
    assert not decision_store_path(repo).exists()

    assert main(_adopt_args(repo, apply=True)) == 0
    repeated = json.loads(capsys.readouterr().out)
    assert repeated["status"] == "ALREADY_DECLARED"


def test_adopt_does_not_classify_from_tools_directory_name(tmp_path: Path, capsys) -> None:
    product_repo = _repo(tmp_path / "product")
    assert main(_adopt_args(product_repo, apply=True, classification="PRODUCT")) == 0
    product_result = json.loads(capsys.readouterr().out)
    assert product_result["status"] == "ADOPTED"
    product_profile = yaml.safe_load((product_repo / "ptsip.yaml").read_text(encoding="utf-8"))
    component = product_profile["components"][0]
    assert component["classification"] == "PRODUCT"
    assert component["runtime_required"] is True
    assert component["lifecycle_owner"] == "PRODUCT"

    neutral_repo = _repo(tmp_path / "neutral")
    assert main(_adopt_args(neutral_repo, apply=True, classification="NEUTRAL_CONTRACT")) == 0
    neutral_result = json.loads(capsys.readouterr().out)
    assert neutral_result["status"] == "ADOPTED"
    neutral_profile = yaml.safe_load((neutral_repo / "ptsip.yaml").read_text(encoding="utf-8"))
    component = neutral_profile["components"][0]
    assert component["classification"] == "NEUTRAL_CONTRACT"
    assert component["executable"] is False
    assert component["runtime_required"] is False
    assert component["lifecycle_owner"] == "INDEPENDENT"
    assert "release_owner" not in component


def test_adopt_extends_existing_covering_component_without_creating_duplicate(
    tmp_path: Path,
    capsys,
) -> None:
    repo = _repo(tmp_path)
    profile = repo / "ptsip.yaml"
    profile.write_text(
        f"""ptsip:\n  version: 0.3.4-draft\n  specification:\n    source: https://github.com/kwaksinwoo01/ptsip\n    revision: {SPEC_REVISION}\ncomponents:\n  - id: generator-sdk\n    classification: TOOLCHAIN\n    include: [\"tools/**\"]\n    purpose: Repository-local generation tooling\npolicies:\n  product_to_toolchain_runtime_dependency: deny\n  toolchain_in_product_package: deny\n  independent_build_resolution: required\n""",
        encoding="utf-8",
    )

    assert main(_adopt_args(repo, apply=True)) == 0
    result = json.loads(capsys.readouterr().out)
    assert result["status"] == "ADOPTED"
    document = yaml.safe_load(profile.read_text(encoding="utf-8"))
    assert [item["id"] for item in document["components"]] == ["generator-sdk"]
    component = document["components"][0]
    assert component["classification"] == "TOOLCHAIN"
    assert component["shipped"] is False
    assert component["runtime_required"] is False
    assert component["lifecycle_owner"] == "DEVELOPMENT_TOOLING"
    assert component["executable"] is True
    assert "release_owner" not in component


def test_existing_conflicting_declaration_is_not_overwritten(tmp_path: Path, capsys) -> None:
    repo = _repo(tmp_path)
    profile = repo / "ptsip.yaml"
    profile.write_text(
        f"""ptsip:\n  version: 0.3.4-draft\n  specification:\n    source: https://github.com/kwaksinwoo01/ptsip\n    revision: {SPEC_REVISION}\ncomponents:\n  - id: tools\n    classification: PRODUCT\n    include: [\"tools/**\"]\n    purpose: Existing product component\npolicies:\n  product_to_toolchain_runtime_dependency: deny\n  toolchain_in_product_package: deny\n  independent_build_resolution: required\n""",
        encoding="utf-8",
    )
    before = profile.read_text(encoding="utf-8")

    assert main(_adopt_args(repo, apply=True)) == 8
    result = json.loads(capsys.readouterr().out)
    assert result["status"] == "CONFLICT"
    assert profile.read_text(encoding="utf-8") == before


def test_structured_adoption_refuses_boundary_shorthand_without_lossless_fact_representation(
    tmp_path: Path,
    capsys,
) -> None:
    repo = _repo(tmp_path)
    profile = repo / "ptsip.yaml"
    profile.write_text(
        f"""ptsip:\n  version: 0.3.4-draft\n  specification:\n    source: https://github.com/kwaksinwoo01/ptsip\n    revision: {SPEC_REVISION}\nboundaries:\n  product:\n    roots: [\"product\"]\n  toolchain:\n    roots: [\"tools\"]\npolicies:\n  product_to_toolchain_runtime_dependency: deny\n  toolchain_in_product_package: deny\n  independent_build_resolution: required\n""",
        encoding="utf-8",
    )
    before = profile.read_text(encoding="utf-8")

    assert main(_adopt_args(repo, apply=True)) == 8
    result = json.loads(capsys.readouterr().out)
    assert result["status"] in {"CONFLICT", "DECISION_ERROR"}
    assert profile.read_text(encoding="utf-8") == before


def test_adoption_application_refuses_stale_repository_evidence(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    answer = DecisionAnswer(
        classification="TOOLCHAIN",
        purpose="Repository-local generation tooling",
        shipped=False,
        runtime_required=False,
        lifecycle_owner="DEVELOPMENT_TOOLING",
        executable=True,
    )
    preparation = prepare_adoption(repo, "tools", answer)
    assert preparation.status == "ADOPTION_PLAN"

    (repo / "tools" / "generate.py").write_text("print('changed')\n", encoding="utf-8")
    status, profile_path, message = apply_adoption(preparation)
    assert status == "STALE_EVIDENCE"
    assert profile_path is not None
    assert message is not None
    assert not (repo / "ptsip.yaml").exists()


def test_adopt_explicit_profile_is_seen_by_clarify_and_gate(
    tmp_path: Path,
    monkeypatch,
    capsys,
) -> None:
    repo = _repo(tmp_path)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))
    config = repo / "config"
    config.mkdir()
    profile = config / "ptsip.yaml"

    assert main(_adopt_args(repo, apply=True, profile=profile)) == 0
    capsys.readouterr()
    assert profile.is_file()
    assert not (repo / "ptsip.yaml").exists()

    assert main(["clarify", str(repo), "--component", "tools", "--profile", str(profile), "--json"]) == 0
    clarified = json.loads(capsys.readouterr().out)
    assert clarified["status"] == "NO_CLARIFICATION_REQUIRED"
    assert clarified["clarification_count"] == 0
    assert Path(str(clarified["profile"]["path"])) == profile.resolve()

    assert main(
        [
            "gate",
            str(repo),
            "--component",
            "tools",
            "--profile",
            str(profile),
            "--coordination",
            "local",
            "--json",
        ]
    ) == 0
    gated = json.loads(capsys.readouterr().out)
    assert gated["status"] == "NO_DECISION_REQUIRED"
    assert gated["backend"] == "LOCAL"
    assert not decision_store_path(repo).exists()


def test_invalid_or_unknown_adoption_never_writes_profile(tmp_path: Path, capsys) -> None:
    repo = _repo(tmp_path)

    invalid = _adopt_args(repo, apply=True)
    runtime_index = invalid.index("--runtime-required") + 1
    invalid[runtime_index] = "yes"
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
