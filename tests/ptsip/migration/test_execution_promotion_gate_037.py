from __future__ import annotations

from pathlib import Path
from types import SimpleNamespace

import pytest

from ptsip.migration import (
    CanonicalSourceComplete,
    CheckpointLedger,
    ExecutionStateError,
    prepare_promotion,
)


def test_promotion_is_blocked_while_participating_temporary_source_remains(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    root.mkdir()
    (root / "ptsip_0.3.6.yaml").write_text("temporary source\n", encoding="utf-8")

    temporary_source = SimpleNamespace(source_path="ptsip_0.3.6.yaml")
    canonical_source = SimpleNamespace(source_path="ptsip.yaml")
    bound = SimpleNamespace(
        sources=(temporary_source, canonical_source),
        plan=SimpleNamespace(final_point=SimpleNamespace(path="ptsip_0.4.0.yaml")),
    )
    verified = SimpleNamespace(
        source=canonical_source,
        source_index=1,
        authorized=SimpleNamespace(bound=bound),
    )
    completed = SimpleNamespace(
        reanalyzed=SimpleNamespace(
            applied=SimpleNamespace(verified=verified),
        )
    )
    canonical = CanonicalSourceComplete(completed)
    ledger = CheckpointLedger(tmp_path / "ledger", "promotion-gate-fixture")

    with pytest.raises(ExecutionStateError, match="Participating temporary source still exists"):
        prepare_promotion(root, canonical, ledger)
