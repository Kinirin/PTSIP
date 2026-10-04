from __future__ import annotations

import json
import os
from pathlib import Path
from typing import Mapping

from ptsip.migration.execution.guard.repository_snapshot import RepositorySnapshotExpectation
from ptsip.migration.execution.state.phase import ExecutionPhase
from ptsip.migration.execution.ledger.record import CheckpointRecord, LedgerIntegrityError
from ptsip.migration.proposal.semantic_identity import semantic_digest

def default_ledger_root(repository_root: str | Path) -> Path:
    root = Path(repository_root).expanduser().resolve()
    git_dir = root / ".git"
    if git_dir.is_dir():
        return git_dir / "ptsip" / "migration"
    repository_id = semantic_digest(str(root))[:24]
    return Path.home() / ".ptsip" / "state" / repository_id / "migration"

class CheckpointLedger:
    """Append-only, digest-chained checkpoint directory outside observed repository content."""

    def __init__(self, root: str | Path, plan_digest: str) -> None:
        if not plan_digest.strip():
            raise ValueError("Checkpoint ledger requires a plan digest.")
        self.root = Path(root).expanduser().resolve() / plan_digest
        self.plan_digest = plan_digest

    @classmethod
    def for_repository(cls, repository_root: str | Path, plan_digest: str) -> "CheckpointLedger":
        return cls(default_ledger_root(repository_root), plan_digest)

    def _files(self) -> list[Path]:
        if not self.root.exists():
            return []
        return sorted(
            (item for item in self.root.iterdir() if item.is_file() and item.name.endswith(".json")),
            key=lambda item: item.name,
        )

    def read_all(self) -> tuple[CheckpointRecord, ...]:
        rows: list[CheckpointRecord] = []
        expected_previous: str | None = None
        for expected_sequence, path in enumerate(self._files(), start=1):
            try:
                raw = json.loads(path.read_text(encoding="utf-8"))
            except (OSError, UnicodeError, json.JSONDecodeError) as exc:
                raise LedgerIntegrityError(f"Unable to read checkpoint {path.name}: {exc}") from exc
            if not isinstance(raw, Mapping):
                raise LedgerIntegrityError(f"Checkpoint {path.name} is not an object.")
            row = CheckpointRecord.from_mapping(raw)
            expected_name = f"{expected_sequence:06d}-{row.digest[:16]}.json"
            if row.sequence != expected_sequence or path.name != expected_name:
                raise LedgerIntegrityError("Checkpoint sequence/name is not contiguous and canonical.")
            if row.plan_digest != self.plan_digest:
                raise LedgerIntegrityError("Checkpoint belongs to a different execution plan.")
            if row.previous_digest != expected_previous:
                raise LedgerIntegrityError("Checkpoint digest chain is broken.")
            rows.append(row)
            expected_previous = row.digest
        return tuple(rows)

    def latest(self) -> CheckpointRecord | None:
        rows = self.read_all()
        return rows[-1] if rows else None

    def append(
        self,
        *,
        phase: ExecutionPhase,
        source_path: str | None = None,
        source_sha256: str | None = None,
        final_point_before_sha256: str | None = None,
        final_point_after_sha256: str | None = None,
        analysis_digest: str | None = None,
        decision_ids: tuple[str, ...] = (),
        repository_snapshot: RepositorySnapshotExpectation | None = None,
        payload: object = None,
    ) -> CheckpointRecord:
        rows = self.read_all()
        previous = rows[-1] if rows else None
        row = CheckpointRecord.build(
            sequence=len(rows) + 1,
            phase=phase,
            plan_digest=self.plan_digest,
            source_path=source_path,
            source_sha256=source_sha256,
            final_point_before_sha256=final_point_before_sha256,
            final_point_after_sha256=final_point_after_sha256,
            analysis_digest=analysis_digest,
            decision_ids=decision_ids,
            repository_snapshot=repository_snapshot,
            payload=payload,
            previous_digest=previous.digest if previous else None,
        )
        self.root.mkdir(parents=True, exist_ok=True)
        target = self.root / f"{row.sequence:06d}-{row.digest[:16]}.json"
        if target.exists():
            raise LedgerIntegrityError(f"Checkpoint target already exists: {target.name}")
        temporary = self.root / f".{target.name}.tmp-{os.getpid()}"
        encoded = json.dumps(row.as_dict(), indent=2, sort_keys=True, ensure_ascii=False) + "\n"
        try:
            with temporary.open("x", encoding="utf-8", newline="\n") as handle:
                handle.write(encoded)
                handle.flush()
                os.fsync(handle.fileno())
            os.replace(temporary, target)
        finally:
            if temporary.exists():
                temporary.unlink(missing_ok=True)
        return row

