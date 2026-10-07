from __future__ import annotations

from dataclasses import dataclass

from ptsip.migration.planning.convergence.final_state import FinalPointReference as LegacyFinalPointReference

@dataclass(frozen=True)
class DirectFinalPointReference(LegacyFinalPointReference):
    """PP-native view of the shared Final Point reference.

    The WU-01~07 sequential implementation historically named this identity
    ``draft_version``.  Direct latest-target convergence uses an independent
    Project Profile contract such as ``pp.1.01``; its serialized/current API
    therefore exposes ``profile_contract`` instead of leaking the historical
    Tool-numbered draft vocabulary.
    """

    @property
    def profile_contract(self) -> str:
        return self.draft_version

    def as_dict(self) -> dict[str, object]:
        payload = super().as_dict()
        payload["profile_contract"] = payload.pop("draft_version")
        return payload

