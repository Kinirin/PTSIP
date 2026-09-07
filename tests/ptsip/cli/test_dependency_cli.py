import json
from pathlib import Path

import pytest

from ptsip.cli import main
from ptsip.dependency_review import build_review_pack, write_review_pack
from ptsip.repository.discover import discover_repository
from ptsip.repository.snapshot import capture_snapshot


def report_for(root):
    root.mkdir(exist_ok=True)
    (root / "example.py").write_text("try:\n    import optional\nexcept ImportError:\n    optional = None\n", encoding="utf-8")
    snapshot = capture_snapshot(root).as_dict()
    items = []
    for index, state in enumerate(["AUTO_RESOLVED", "REPOSITORY_DEFECT", "RESOLVER_LIMITATION"] + ["REVIEW_REQUIRED"] * 10):
        items.append({"evidence_id": f"python:{index:02d}", "actionability": state,
                      "reason": "GUARDED", "dependency": {"source": "example.py", "target": "optional", "line": 2},
                      "component": {"id": "app", "classification": "PRODUCT"},
                      "declaration": {"found": False}, "usage": {"guarded": True}})
    return {"format": "ptsip-dependency-analysis/v1", "status": "RAN", "repository": discover_repository(root).as_dict(),
            "items": items, "summary": {"observed_edges": 13, "AUTO_RESOLVED": 1, "REPOSITORY_DEFECT": 1,
                                        "RESOLVER_LIMITATION": 1, "REVIEW_REQUIRED": 10},
            "snapshot": {"after": snapshot}}


def test_review_pack_only_escalates_bounded_review_required(tmp_path):
    report = report_for(tmp_path / "consumer")
    pack = build_review_pack(report)
    assert pack["summary"] == {"review_required_total": 10, "selected_for_review": 8, "deferred": 2, "ai_reviewed_items": 0}
    assert all(item["actionability"] == "REVIEW_REQUIRED" for item in pack["reviews"])
    assert all(len(json.dumps(item, ensure_ascii=False, sort_keys=True, indent=2).encode("utf-8")) <= 12000
               for item in pack["reviews"])
    assert build_review_pack(report) == pack
    assert build_review_pack(report, max_items=0)["summary"]["deferred"] == 10


def test_review_pack_byte_budget_and_stale_evidence_fail_closed(tmp_path):
    report = report_for(tmp_path / "consumer")
    for item in report["items"]:
        item["reason"] = "한글" * 500
    assert build_review_pack(report, max_context_bytes_per_item=100)["summary"]["selected_for_review"] == 0
    (tmp_path / "consumer" / "example.py").write_text("changed", encoding="utf-8")
    with pytest.raises(RuntimeError, match="changed"):
        write_review_pack(report, tmp_path / "stale.json")
    assert not (tmp_path / "stale.json").exists()


def test_cli_preserves_full_json_and_concise_human_output(tmp_path, monkeypatch, capsys):
    report = report_for(tmp_path / "consumer")
    monkeypatch.setattr("ptsip.cli.analyze_dependencies", lambda *args, **kwargs: report)
    assert main(["dependency", "analyze", str(tmp_path)]) == 0
    output = capsys.readouterr().out
    assert "observed" in output and "python:03" not in output
    assert len(output.splitlines()) < 15
    assert main(["dependency", "analyze", str(tmp_path), "--json"]) == 0
    assert json.loads(capsys.readouterr().out) == json.loads(json.dumps(report))
    path = tmp_path / "pack.json"
    assert main(["dependency", "review-pack", str(tmp_path), "--output", str(path), "--max-items", "2", "--json"]) == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["summary"]["selected_for_review"] == 2
    assert path.is_file()


def test_default_state_non_intrusion_and_output_clobber_protection(tmp_path, monkeypatch):
    root = tmp_path / "consumer"
    report = report_for(root)
    monkeypatch.setenv("PTSIP_HOME", str(root / ".ptsip"))
    with pytest.raises(ValueError, match="outside"):
        write_review_pack(report)
    assert not (root / ".ptsip").exists()
    output = tmp_path / "occupied.json"
    output.write_text("existing", encoding="utf-8")
    with pytest.raises(FileExistsError):
        write_review_pack(report, output)
    assert output.read_text(encoding="utf-8") == "existing"
