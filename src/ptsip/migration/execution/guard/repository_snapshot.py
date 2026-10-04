from __future__ import annotations

from dataclasses import dataclass

from ptsip.repository.snapshot import RepositorySnapshot
from ptsip.migration.analysis.result import MigrationAnalysis

@dataclass(frozen=True)
class RepositorySnapshotExpectation:
    head: str | None
    status_fingerprint: str
    tracked_content_fingerprint: str

    @classmethod
    def from_snapshot(cls, snapshot: RepositorySnapshot) -> "RepositorySnapshotExpectation":
        return cls(snapshot.head, snapshot.status_fingerprint, snapshot.tracked_content_fingerprint)

    def matches(self, snapshot: RepositorySnapshot) -> bool:
        return (
            self.head == snapshot.head
            and self.status_fingerprint == snapshot.status_fingerprint
            and self.tracked_content_fingerprint == snapshot.tracked_content_fingerprint
            and not snapshot.observation_errors
        )

    def as_dict(self) -> dict[str, object]:
        return {
            "head": self.head,
            "status_fingerprint": self.status_fingerprint,
            "tracked_content_fingerprint": self.tracked_content_fingerprint,
        }

def _analysis_snapshot(analysis: MigrationAnalysis) -> RepositorySnapshotExpectation:
    return RepositorySnapshotExpectation(
        analysis.repository_head,
        analysis.repository_status_fingerprint,
        analysis.repository_content_fingerprint,
    )

def _current_snapshot_matches_analysis(analysis: MigrationAnalysis, snapshot) -> bool:
    return (
        analysis.repository_head == snapshot.head
        and analysis.repository_status_fingerprint == snapshot.status_fingerprint
        and analysis.repository_content_fingerprint == snapshot.tracked_content_fingerprint
        and not snapshot.observation_errors
    )

