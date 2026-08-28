from __future__ import annotations

from pathlib import Path
from types import SimpleNamespace

import pytest

from ptsip.migration import ExecutionStateError
from ptsip.migration.execution_apply import _apply_delta_batch


def _verified() -> SimpleNamespace:
    final_point = SimpleNamespace(
        path="ptsip_0.3.7.yaml",
        draft_version="0.3.7-draft",
        specification_revision="rev7",
    )
    return SimpleNamespace(
        authorized=SimpleNamespace(
            bound=SimpleNamespace(
                plan=SimpleNamespace(final_point=final_point),
            )
        )
    )


def _seed(*, mode: str = "explicit") -> dict[str, object]:
    return {
        "ptsip": {
            "version": "0.3.7-draft",
            "specification": {
                "source": "https://github.com/Kinirin/PTSIP",
                "revision": "rev7",
            },
        },
        "responsibility_map": {"mode": mode},
    }


def test_planned_seed_rejects_hybrid_declaration_authority_before_write(tmp_path: Path) -> None:
    target = tmp_path / "ptsip_0.3.7.yaml"
    seed = _seed(mode="hybrid")
    seed["responsibility_map"] = {
        "mode": "hybrid",
        "template": {"id": "python-package-library", "revision": "sha256:fixture"},
    }

    with pytest.raises(ExecutionStateError, match="explicit Responsibility Map targets only"):
        _apply_delta_batch(target, (), _verified(), seed_payload=seed)

    assert not target.exists()


def test_planned_seed_rejects_unaccepted_target_entities_before_write(tmp_path: Path) -> None:
    target = tmp_path / "ptsip_0.3.7.yaml"
    seed = _seed()
    seed["components"] = [
        {
            "id": "unaccepted",
            "classification": "PRODUCT",
            "include": ["hidden/**"],
        }
    ]

    with pytest.raises(ExecutionStateError, match="must not contain target entities"):
        _apply_delta_batch(target, (), _verified(), seed_payload=seed)

    assert not target.exists()
