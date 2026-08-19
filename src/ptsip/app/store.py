from __future__ import annotations

import json
import sqlite3
import threading
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from ..clarification.resolution import LegacyDecisionAnswerV1, canonicalize_legacy_answer
from ..clarification.resolution.model import CANONICAL_ANSWER_FIELDS, LEGACY_V1_ANSWER_FIELDS
from ..repository.profile_path import normalize_profile_path


@dataclass(frozen=True)
class DecisionRecord:
    id: str
    repository: str
    branch: str
    subject_revision: str
    profile_path: str
    component_id: str
    request: dict[str, object]
    status: str
    answer: dict[str, object] | None
    resolution_source: str | None
    resolved_by: str | None
    application_status: str
    applied_revision: str | None
    issue_number: int | None
    issue_url: str | None

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "repository": self.repository,
            "branch": self.branch,
            "subject_revision": self.subject_revision,
            "profile_path": self.profile_path,
            "component_id": self.component_id,
            "request": self.request,
            "status": self.status,
            "answer": self.answer,
            "resolution_source": self.resolution_source,
            "resolved_by": self.resolved_by,
            "application_status": self.application_status,
            "applied_revision": self.applied_revision,
            "issue_number": self.issue_number,
            "issue_url": self.issue_url,
        }


def _canonical_answer_for_compare(payload: object) -> dict[str, object] | None:
    """Normalize only known persisted answer shapes for retry comparison.

    Existing v1 rows are not rewritten. They are interpreted through the
    explicit compatibility model only long enough to compare with a new v2
    answer. Invalid/unknown historical shapes cannot authorize a retry.
    """

    if not isinstance(payload, dict):
        return None
    if set(payload) == set(CANONICAL_ANSWER_FIELDS):
        return payload
    if set(payload) != set(LEGACY_V1_ANSWER_FIELDS):
        return None
    try:
        classification = payload["classification"]
        purpose = payload["purpose"]
        lifecycle_owner = payload["lifecycle_owner"]
        if not all(isinstance(value, str) and value.strip() for value in (classification, purpose, lifecycle_owner)):
            return None
        for field in ("shipped", "runtime_required", "executable"):
            if not isinstance(payload[field], bool):
                return None
        legacy = LegacyDecisionAnswerV1(
            classification=classification.strip().upper(),
            purpose=purpose.strip(),
            shipped=payload["shipped"],
            runtime_required=payload["runtime_required"],
            lifecycle_owner=lifecycle_owner.strip().upper(),
            executable=payload["executable"],
        )
        return canonicalize_legacy_answer(legacy).as_dict()
    except (KeyError, ValueError):
        return None


class DecisionStore:
    def __init__(self, path: str | Path):
        self.path = Path(path).expanduser().resolve()
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self._lock = threading.RLock()
        self._init()

    def _connect(self) -> sqlite3.Connection:
        conn = sqlite3.connect(self.path, timeout=30)
        conn.row_factory = sqlite3.Row
        return conn

    def _init(self) -> None:
        with self._connect() as conn:
            conn.executescript(
                """
                PRAGMA journal_mode=WAL;
                CREATE TABLE IF NOT EXISTS installations (
                    repository TEXT PRIMARY KEY,
                    installation_id INTEGER NOT NULL
                );
                CREATE TABLE IF NOT EXISTS decisions (
                    id TEXT PRIMARY KEY,
                    repository TEXT NOT NULL,
                    branch TEXT NOT NULL,
                    subject_revision TEXT NOT NULL,
                    profile_path TEXT NOT NULL DEFAULT 'ptsip.yaml',
                    component_id TEXT NOT NULL,
                    request_json TEXT NOT NULL,
                    status TEXT NOT NULL DEFAULT 'PENDING',
                    answer_json TEXT,
                    resolution_source TEXT,
                    resolved_by TEXT,
                    application_status TEXT NOT NULL DEFAULT 'NOT_APPLIED',
                    applied_revision TEXT,
                    issue_number INTEGER,
                    issue_url TEXT,
                    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
                    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
                );
                CREATE INDEX IF NOT EXISTS idx_decision_issue
                    ON decisions(repository, issue_number);
                CREATE INDEX IF NOT EXISTS idx_decision_component
                    ON decisions(repository, component_id, status);
                """
            )
            columns = {str(row[1]) for row in conn.execute("PRAGMA table_info(decisions)").fetchall()}
            if "profile_path" not in columns:
                conn.execute(
                    "ALTER TABLE decisions ADD COLUMN profile_path TEXT NOT NULL DEFAULT 'ptsip.yaml'"
                )

    @staticmethod
    def _record(row: sqlite3.Row) -> DecisionRecord:
        return DecisionRecord(
            id=str(row["id"]),
            repository=str(row["repository"]),
            branch=str(row["branch"]),
            subject_revision=str(row["subject_revision"]),
            profile_path=normalize_profile_path(row["profile_path"]),
            component_id=str(row["component_id"]),
            request=json.loads(str(row["request_json"])),
            status=str(row["status"]),
            answer=json.loads(str(row["answer_json"])) if row["answer_json"] else None,
            resolution_source=str(row["resolution_source"]) if row["resolution_source"] else None,
            resolved_by=str(row["resolved_by"]) if row["resolved_by"] else None,
            application_status=str(row["application_status"]),
            applied_revision=str(row["applied_revision"]) if row["applied_revision"] else None,
            issue_number=int(row["issue_number"]) if row["issue_number"] is not None else None,
            issue_url=str(row["issue_url"]) if row["issue_url"] else None,
        )

    def set_installation(self, repository: str, installation_id: int) -> None:
        with self._lock, self._connect() as conn:
            conn.execute(
                "INSERT INTO installations(repository, installation_id) VALUES (?, ?) "
                "ON CONFLICT(repository) DO UPDATE SET installation_id=excluded.installation_id",
                (repository, installation_id),
            )

    def installation_for(self, repository: str) -> int | None:
        with self._connect() as conn:
            row = conn.execute(
                "SELECT installation_id FROM installations WHERE repository=?", (repository,)
            ).fetchone()
        return int(row[0]) if row else None

    def gate(self, payload: dict[str, Any]) -> tuple[DecisionRecord, tuple[DecisionRecord, ...]]:
        decision_id = str(payload["id"])
        repository = str(payload["repository"])
        branch = str(payload["branch"])
        revision = str(payload["subject_revision"])
        profile_path = normalize_profile_path(payload.get("profile_path"))
        component_id = str(payload["component_id"])
        request = payload["request"]
        if not isinstance(request, dict):
            raise ValueError("request must be an object")
        request_json = json.dumps(request, ensure_ascii=False)
        stale: list[DecisionRecord] = []
        with self._lock, self._connect() as conn:
            conn.execute("BEGIN IMMEDIATE")
            rows = conn.execute(
                "SELECT * FROM decisions WHERE repository=? AND component_id=? AND status='PENDING' AND id<>?",
                (repository, component_id, decision_id),
            ).fetchall()
            for row in rows:
                stale.append(self._record(row))
            conn.execute(
                "UPDATE decisions SET status='STALE', updated_at=CURRENT_TIMESTAMP "
                "WHERE repository=? AND component_id=? AND status='PENDING' AND id<>?",
                (repository, component_id, decision_id),
            )
            conn.execute(
                "INSERT OR IGNORE INTO decisions(id, repository, branch, subject_revision, profile_path, component_id, request_json) "
                "VALUES (?, ?, ?, ?, ?, ?, ?)",
                (decision_id, repository, branch, revision, profile_path, component_id, request_json),
            )
            # A gate call is an active-agent observation, not a timer. When the
            # same semantic decision is still pending or needs application retry,
            # rebind its exact branch/revision/profile target to the snapshot the
            # agent is actually working on. The human answer and first winner are
            # never changed by this rebind.
            conn.execute(
                "UPDATE decisions SET branch=?, subject_revision=?, profile_path=?, request_json=?, updated_at=CURRENT_TIMESTAMP "
                "WHERE id=? AND (status='PENDING' OR (status='RESOLVED' AND application_status IN ('NOT_APPLIED','FAILED','STALE')))",
                (branch, revision, profile_path, request_json, decision_id),
            )
            row = conn.execute("SELECT * FROM decisions WHERE id=?", (decision_id,)).fetchone()
            assert row is not None
            conn.commit()
        return self._record(row), tuple(stale)

    def get(self, decision_id: str) -> DecisionRecord | None:
        with self._connect() as conn:
            row = conn.execute("SELECT * FROM decisions WHERE id=?", (decision_id,)).fetchone()
        return self._record(row) if row else None

    def by_issue(self, repository: str, issue_number: int) -> DecisionRecord | None:
        with self._connect() as conn:
            row = conn.execute(
                "SELECT * FROM decisions WHERE repository=? AND issue_number=? ORDER BY created_at DESC LIMIT 1",
                (repository, issue_number),
            ).fetchone()
        return self._record(row) if row else None

    def attach_issue(self, decision_id: str, issue_number: int, issue_url: str) -> DecisionRecord:
        with self._lock, self._connect() as conn:
            conn.execute(
                "UPDATE decisions SET issue_number=?, issue_url=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
                (issue_number, issue_url, decision_id),
            )
        record = self.get(decision_id)
        assert record is not None
        return record

    def resolve(
        self,
        decision_id: str,
        answer: dict[str, object],
        source: str,
        actor: str,
    ) -> tuple[DecisionRecord, bool]:
        answer_json = json.dumps(answer, ensure_ascii=False)
        with self._lock, self._connect() as conn:
            conn.execute("BEGIN IMMEDIATE")
            changed = conn.execute(
                "UPDATE decisions SET status='RESOLVED', answer_json=?, resolution_source=?, resolved_by=?, "
                "updated_at=CURRENT_TIMESTAMP WHERE id=? AND status='PENDING'",
                (answer_json, source, actor, decision_id),
            ).rowcount
            row = conn.execute("SELECT * FROM decisions WHERE id=?", (decision_id,)).fetchone()
            if row is None:
                conn.rollback()
                raise KeyError(decision_id)
            retry_allowed = False
            if not changed and source == "AGENT_CHAT":
                existing_answer = json.loads(str(row["answer_json"])) if row["answer_json"] else None
                retry_allowed = (
                    str(row["status"]) == "RESOLVED"
                    and str(row["application_status"]) in {"NOT_APPLIED", "FAILED", "STALE"}
                    and _canonical_answer_for_compare(existing_answer) == answer
                )
            conn.commit()
        return self._record(row), bool(changed or retry_allowed)

    def mark_application(self, decision_id: str, status: str, applied_revision: str | None = None) -> DecisionRecord:
        with self._lock, self._connect() as conn:
            conn.execute(
                "UPDATE decisions SET application_status=?, applied_revision=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
                (status, applied_revision, decision_id),
            )
        record = self.get(decision_id)
        if record is None:
            raise KeyError(decision_id)
        return record
