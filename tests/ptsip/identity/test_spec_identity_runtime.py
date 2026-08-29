from __future__ import annotations

from ptsip.constants import SPEC_REVISION
from ptsip.spec_identity import current_spec_identity


def test_spec_identity() -> None:
    spec = current_spec_identity()
    assert spec.tool_version == "0.3.7"
    assert spec.version == "0.3.7-draft"
    assert spec.source == "https://github.com/Kinirin/PTSIP"
    assert spec.revision == SPEC_REVISION
