"""Disposable content-bound scanner evidence in existing external Tool state."""
from __future__ import annotations

import hashlib
import json
import os
from pathlib import Path
import tempfile
import subprocess
import sys

from .constants import TOOL_VERSION
from .storage.local_state import ptsip_home, repository_fingerprint


def digest(value) -> str:
    return hashlib.sha256(json.dumps(value, sort_keys=True, ensure_ascii=True,
                                     separators=(",", ":")).encode()).hexdigest()


def file_hash(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


class EvidenceCache:
    """Revision envelope is separate from reusable per-source semantic keys.

    All callers still capture and compare the complete live snapshot. A changed
    unrelated file/revision can reuse source evidence only when source, manifests,
    ownership, tracked target topology, and executing resolver bytes are equal.
    """

    def __init__(self, root: Path, paths: list[str], *, enabled=True):
        self.root = root.resolve()
        self.stats = {"scope": "PYTHON_SOURCE", "hits": 0, "recomputed": 0,
                      "invalid": 0, "write_failures": 0, "enabled": enabled}
        self.directory = ptsip_home() / "dependency-cache" / repository_fingerprint(root)
        if self.directory.resolve().is_relative_to(self.root):
            self.stats.update(enabled=False, reason="TOOL_STATE_INSIDE_CONSUMER")
        manifest_hashes = {}
        # The legacy resolver also observes root manifests and untracked target
        # existence. Bind those inputs until resolution is tracked-only.
        observed_paths = set(paths)
        observed_paths.update(item.relative_to(root).as_posix() for item in root.glob("requirements*.txt"))
        observed_paths.update(item.relative_to(root).as_posix() for item in root.glob("requirements*.in"))
        if (root / "pyproject.toml").is_file():
            observed_paths.add("pyproject.toml")
        untracked = subprocess.run(["git", "-C", str(root), "ls-files", "--others", "-z"],
                                   capture_output=True, check=False).stdout
        for rel in sorted(observed_paths):
            name = Path(rel).name.lower()
            if (name in {"ptsip.yaml", "pyproject.toml", "setup.cfg", "setup.py"}
                    or "requirements" in name or name == "__init__.py"):
                try:
                    manifest_hashes[rel] = file_hash(root / rel)
                except OSError:
                    self.stats.update(enabled=False, reason="INPUT_UNREADABLE")
        implementation = Path(__file__).parent
        resolver_hashes = {rel: file_hash(implementation / rel) for rel in (
            "dependency_cache.py", "dependency_analysis.py", "dependency_reconciliation.py",
            "inspection/dependencies.py", "inspection/dependencies_030.py", "inspection/python_resolution.py")}
        self.context = digest({"repository": str(self.root), "tool": TOOL_VERSION,
                               "manifests_and_component_declarations": manifest_hashes,
                               "tracked_target_topology": paths, "untracked_target_topology": hashlib.sha256(untracked).hexdigest(),
                               "python_parser": list(sys.version_info[:3]),
                               "platform_module_names": sorted(sys.stdlib_module_names),
                               "resolver": resolver_hashes})

    def get(self, kind: str, rel: str, compute, validate=lambda value: True):
        if not self.stats["enabled"]:
            self.stats["recomputed"] += 1
            return compute()
        try:
            source_hash = file_hash(self.root / rel)
        except OSError:
            self.stats["recomputed"] += 1
            return compute()
        key = digest({"format": "ptsip-source-evidence-cache/v1", "context": self.context,
                      "kind": kind, "path": rel, "source_sha256": source_hash})
        path = self.directory / key[:2] / (key + ".json")
        try:
            envelope = json.loads(path.read_text(encoding="utf-8"))
            value = envelope["value"]
            if envelope["key"] != key or digest(value) != envelope["digest"] or not validate(value):
                raise ValueError("Cache payload is incomplete or corrupt")
            self.stats["hits"] += 1
            return value
        except FileNotFoundError:
            pass
        except (OSError, ValueError, KeyError, TypeError):
            self.stats["invalid"] += 1
        self.stats["recomputed"] += 1
        value = compute()
        # Never persist evidence collected while this exact source changed.
        try:
            if file_hash(self.root / rel) != source_hash or not validate(value):
                return value
            path.parent.mkdir(parents=True, exist_ok=True)
            data = {"key": key, "digest": digest(value), "value": value}
            with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", dir=path.parent,
                                             prefix="pending-", delete=False) as stream:
                temporary = Path(stream.name)
                json.dump(data, stream, ensure_ascii=True, sort_keys=True)
            try:
                os.replace(temporary, path)
            finally:
                temporary.unlink(missing_ok=True)
        except OSError:
            self.stats["write_failures"] += 1
        return value
