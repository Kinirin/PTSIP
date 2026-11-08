import importlib.util
from pathlib import Path
import subprocess

import pytest


def test_source_snapshot_rejects_uncommitted_assessment_inputs(tmp_path):
    root = Path(__file__).resolve().parents[4]
    script = root / ".ptsip/engine/developer/policy/applicability/input.py"
    spec = importlib.util.spec_from_file_location("policy_applicability_input", script)
    adapter = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(adapter)

    def git(*args):
        return subprocess.check_output(["git", *args], cwd=tmp_path, text=True).strip()

    git("init", "--quiet")
    policy = tmp_path / "src/policy/index.yaml"
    policy.parent.mkdir(parents=True)
    policy.write_text("policies: []\n", encoding="utf-8")
    git("add", ".")
    git("-c", "user.name=Test", "-c", "user.email=test@example.invalid",
        "-c", "core.hooksPath=.git/no-hooks", "commit", "--quiet", "-m", "fixture")
    head = git("rev-parse", "HEAD")

    # Local analysis artifacts do not change the source snapshot.
    (tmp_path / ".agent").mkdir()
    (tmp_path / ".agent" / "candidate.jsonl").write_text("{}\n", encoding="utf-8")
    adapter.require_source_snapshot(tmp_path, head)
    with pytest.raises(ValueError, match="SOURCE_HEAD_CHANGED_DURING_ASSESSMENT"):
        adapter.require_source_snapshot(tmp_path, "0" * 40)

    policy.write_text("policies: [uncommitted]\n", encoding="utf-8")
    with pytest.raises(ValueError, match="SOURCE_SNAPSHOT_DIRTY.*src/policy/index.yaml"):
        adapter.require_source_snapshot(tmp_path, head)
    git("add", "src/policy/index.yaml")
    with pytest.raises(ValueError, match="SOURCE_SNAPSHOT_DIRTY"):
        adapter.require_source_snapshot(tmp_path, head)
