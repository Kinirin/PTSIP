from __future__ import annotations

import json
import subprocess
from pathlib import Path

import yaml

from developer.automation.pp_transition_reconciler import (
    PPTransitionReconcileError,
    reconcile_staged_transition,
)


def _run(root: Path, *args: str) -> str:
    result = subprocess.run(
        ["git", *args],
        cwd=root,
        check=True,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    return result.stdout


def _write(root: Path, path: str, text: str) -> None:
    target = root / path
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(text, encoding="utf-8", newline="\n")


def _schema(version: str, *, extra: str = "") -> str:
    payload = {
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "$id": f"https://raw.githubusercontent.com/Kinirin/PTSIP/main/schemas/ptsip-profile-{version}.schema.json",
        "title": f"PTSIP Project Profile {version}",
        "type": "object",
        "required": ["ptsip", "responsibility_map", "policies"],
        "properties": {
            "ptsip": {
                "type": "object",
                "required": ["version"],
                "properties": {
                    "version": {
                        "type": "string",
                        "const": version,
                        "description": "Canonical Project Profile contract identity.",
                    }
                },
            },
            "responsibility_map": {"type": "object"},
            "policies": {"type": "object"},
        },
    }
    if extra:
        payload["x-semantic-change"] = extra
    return json.dumps(payload, indent=2) + "\n"


def _profile(version: str, purpose: str = "base") -> str:
    return (
        'ptsip:\n'
        f'  version: "{version}"\n'
        'responsibility_map:\n'
        '  mode: explicit\n'
        'components:\n'
        '  - id: product\n'
        '    classification: PRODUCT\n'
        '    include: ["src/**"]\n'
        f'    purpose: {purpose}\n'
        'policies:\n'
        '  product_to_nonproduct_runtime_dependency: deny\n'
        '  nonproduct_in_product_package: deny\n'
        '  independent_build_resolution: required\n'
    )


def _init_repo(root: Path) -> None:
    _run(root, "init")
    _run(root, "config", "user.email", "test@example.com")
    _run(root, "config", "user.name", "PTSIP Test")

    _write(
        root,
        "pyproject.toml",
        """[project]
name = "ptsip-transition-fixture"
version = "0.0.0"
""",
    )

    _write(
        root,
        "profiles/index.yaml",
        """schema_version: ptsip-public-profile-catalog/v1
authority: PTSIP_PUBLIC_PROFILE_CATALOG
root: profiles
profiles:
  - id: example
    resource: example.ptsip.yaml
    contract: pp.1.01
    responsibility_mode: explicit
""",
    )
    _write(root, "profiles/example.ptsip.yaml", _profile("pp.1.01"))
    _write(root, "profiles/history/pp.1.01/example.ptsip.yaml", _profile("pp.1.01"))
    _write(root, "schemas/ptsip-profile-pp-1.01.schema.json", _schema("pp.1.01"))
    _write(root, "src/ptsip/specdata/ptsip-profile-pp-1.01.schema.json", _schema("pp.1.01"))
    _write(
        root,
        "registry/project-profile-contracts.yaml",
        """schema_version: ptsip-project-profile-contract-registry/v1
authority: PTSIP_PROJECT_PROFILE_CONTRACT_IDENTITY
current: pp.1.01
contracts:
  - version: pp.1.01
    lifecycle: CURRENT
    operations: [IDENTIFY, VALIDATE, ANALYZE, CREATE_TARGET]
    schema: schemas/ptsip-profile-pp-1.01.schema.json
    baseline: profiles/history/pp.1.01
transitions: []
""",
    )

    _write(
        root,
        "src/ptsip/specdata/project-profile-contracts.yaml",
        """schema_version: ptsip-project-profile-contract-registry/v1
authority: PTSIP_PROJECT_PROFILE_CONTRACT_IDENTITY
current: pp.1.01
contracts:
  - version: pp.1.01
    lifecycle: CURRENT
    operations: [IDENTIFY, VALIDATE, ANALYZE, CREATE_TARGET]
    schema: schemas/ptsip-profile-pp-1.01.schema.json
    baseline: profiles/history/pp.1.01
transitions: []
""",
    )

    schema_root = Path(__file__).resolve().parents[3] / "developer" / "policy" / "schemas"
    for name in (
        "public-profile-catalog.schema.json",
        "project-profile-contract-registry.schema.json",
    ):
        target = root / "developer" / "policy" / "schemas" / name
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes((schema_root / name).read_bytes())

    _run(root, "add", ".")
    _run(root, "commit", "-m", "baseline")


def test_profile_semantic_change_reconciles_adjacent_minor(tmp_path: Path) -> None:
    _init_repo(tmp_path)
    _write(tmp_path, "profiles/example.ptsip.yaml", _profile("pp.1.01", "changed"))
    _run(tmp_path, "add", "profiles/example.ptsip.yaml")

    preview = reconcile_staged_transition(tmp_path)
    assert preview.status == "RECONCILE"
    assert preview.target == "pp.1.02"

    result = reconcile_staged_transition(tmp_path, apply=True)
    assert result.status == "RECONCILED"

    registry = yaml.safe_load(
        (tmp_path / "registry/project-profile-contracts.yaml").read_text(encoding="utf-8")
    )
    assert registry["current"] == "pp.1.02"
    assert registry["transitions"] == [
        {"from": "pp.1.01", "to": "pp.1.02", "kind": "SEMANTIC_MIGRATION"}
    ]
    embedded_registry = yaml.safe_load(
        (
            tmp_path / "src" / "ptsip" / "specdata" / "project-profile-contracts.yaml"
        ).read_text(encoding="utf-8")
    )
    assert embedded_registry == registry
    assert (
        yaml.safe_load((tmp_path / "profiles/example.ptsip.yaml").read_text(encoding="utf-8"))
        ["ptsip"]["version"]
        == "pp.1.02"
    )
    assert (
        tmp_path / "profiles/history/pp.1.02/example.ptsip.yaml"
    ).read_bytes() == (tmp_path / "profiles/example.ptsip.yaml").read_bytes()

    second = reconcile_staged_transition(tmp_path)
    assert second.status == "ALREADY_RECONCILED"
    assert second.target == "pp.1.02"


def test_schema_change_moves_semantics_to_new_generation_and_restores_old(tmp_path: Path) -> None:
    _init_repo(tmp_path)
    baseline = (tmp_path / "schemas/ptsip-profile-pp-1.01.schema.json").read_bytes()
    _write(
        tmp_path,
        "schemas/ptsip-profile-pp-1.01.schema.json",
        _schema("pp.1.01", extra="changed"),
    )
    _run(tmp_path, "add", "schemas/ptsip-profile-pp-1.01.schema.json")

    reconcile_staged_transition(tmp_path, apply=True)

    assert (tmp_path / "schemas/ptsip-profile-pp-1.01.schema.json").read_bytes() == baseline
    assert not (tmp_path / "schemas/ptsip-profile-pp.1.02.schema.json").exists()
    generated = json.loads(
        (tmp_path / "schemas/ptsip-profile-pp-1.02.schema.json").read_text(encoding="utf-8")
    )
    assert generated["properties"]["ptsip"]["properties"]["version"]["const"] == "pp.1.02"
    assert generated["x-semantic-change"] == "changed"


def test_manual_registry_mutation_fails_closed(tmp_path: Path) -> None:
    _init_repo(tmp_path)
    _write(tmp_path, "profiles/example.ptsip.yaml", _profile("pp.1.01", "changed"))
    registry_path = tmp_path / "registry/project-profile-contracts.yaml"
    registry = registry_path.read_text(encoding="utf-8").replace(
        "current: pp.1.01",
        "current: pp.1.02",
    )
    registry_path.write_text(registry, encoding="utf-8")
    _run(
        tmp_path,
        "add",
        "profiles/example.ptsip.yaml",
        "registry/project-profile-contracts.yaml",
    )

    try:
        reconcile_staged_transition(tmp_path)
    except PPTransitionReconcileError as exc:
        assert exc.code in {
            "PP_TRANSITION_IDENTITY_MISMATCH",
            "MANUAL_PP_TRANSITION_WITHOUT_AUTHORITY",
            "MANUAL_PP_REGISTRY_MUTATION",
            "RECONCILED_TRANSITION_MISSING",
        }
    else:
        raise AssertionError("manual PP registry mutation must fail closed")


def test_unstaged_output_conflict_blocks_apply(tmp_path: Path) -> None:
    _init_repo(tmp_path)
    _write(tmp_path, "profiles/example.ptsip.yaml", _profile("pp.1.01", "changed"))
    _run(tmp_path, "add", "profiles/example.ptsip.yaml")
    with (tmp_path / "profiles/example.ptsip.yaml").open("a", encoding="utf-8") as stream:
        stream.write("# unstaged user edit\n")

    preview = reconcile_staged_transition(tmp_path)
    assert preview.status == "RECONCILE"

    try:
        reconcile_staged_transition(tmp_path, apply=True)
    except PPTransitionReconcileError as exc:
        assert exc.code == "AUTOMATION_WRITE_CONFLICT"
    else:
        raise AssertionError("unstaged automation-output conflict must fail closed")


def test_no_t2_delta_is_no_change(tmp_path: Path) -> None:
    _init_repo(tmp_path)
    result = reconcile_staged_transition(tmp_path)
    assert result.status == "NO_CHANGE"
    assert result.target == "pp.1.01"


def test_historical_baseline_mutation_fails_closed(tmp_path: Path) -> None:
    _init_repo(tmp_path)
    _write(tmp_path, "profiles/example.ptsip.yaml", _profile("pp.1.01", "changed"))
    with (tmp_path / "profiles/history/pp.1.01/example.ptsip.yaml").open(
        "a",
        encoding="utf-8",
    ) as stream:
        stream.write("# forbidden history edit\n")
    _run(
        tmp_path,
        "add",
        "profiles/example.ptsip.yaml",
        "profiles/history/pp.1.01/example.ptsip.yaml",
    )

    try:
        reconcile_staged_transition(tmp_path)
    except PPTransitionReconcileError as exc:
        assert exc.code == "HISTORICAL_BASELINE_MUTATION"
    else:
        raise AssertionError("historical baseline mutation must fail closed")
