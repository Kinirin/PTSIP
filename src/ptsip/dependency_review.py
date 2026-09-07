"""Bounded, read-only evidence packages for unresolved support-contract review."""
from __future__ import annotations

from copy import deepcopy
import hashlib
import json
from pathlib import Path
import tokenize

from .repository.snapshot import capture_snapshot, git_tracked_files
from .storage.local_state import ptsip_home, repository_fingerprint


def _serialized(value: object) -> str:
    return json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2)


def _size(value: object) -> int:
    return len(_serialized(value).encode("utf-8"))


def _snippet(root: Path, path: str, line: int, kind: str) -> dict[str, object] | None:
    candidate = root / path
    if not candidate.resolve().is_relative_to(root) or candidate.is_symlink():
        return None
    try:
        with tokenize.open(candidate) as source:
            lines = source.read().splitlines()
    except (OSError, UnicodeError, SyntaxError):
        return None
    if not 1 <= line <= len(lines):
        return None
    start = max(1, line - 3)
    end = min(len(lines), start + 6)
    selected = "\n".join(lines[start - 1:end])
    return {
        "path": path, "line": line, "start_line": start, "end_line": end,
        "kind": kind, "text": selected[:1200], "truncated": len(selected) > 1200,
    }


def _callers(report: dict, item: dict, max_files: int) -> list[dict]:
    source = item["dependency"]["source"]
    candidates = item.get("callers")
    if candidates is None:
        candidates = [
            {"path": edge["dependency"]["source"], "line": edge["dependency"].get("line"),
             "evidence_id": edge["evidence_id"], "semantics": "OBSERVED_FILE_DEPENDENCY"}
            for edge in report["items"] if edge["dependency"].get("resolved_path") == source
        ]
    selected, files = [], {source}
    for caller in sorted(candidates, key=lambda value: (
        value.get("path", ""), value.get("line") or 0, value.get("evidence_id", "")
    )):
        path = caller.get("path")
        if not path or (path not in files and len(files) >= max_files):
            continue
        files.add(path)
        selected.append({**{key: caller.get(key) for key in ("path", "line", "evidence_id")},
                         "semantics": caller.get("semantics") or caller.get("kind")})
        if len(selected) >= 4:
            break
    return selected


def build_review_pack(
    report: dict, *, max_items: int = 8, max_context_bytes_per_item: int = 12000,
    max_source_files_per_item: int = 4,
) -> dict:
    """Select only REVIEW_REQUIRED items; never invoke an AI or assert authority."""
    if max_items < 0 or max_context_bytes_per_item < 1 or max_source_files_per_item < 1:
        raise ValueError("Review budgets require nonnegative max-items and positive byte/file caps")
    if report.get("status") != "RAN":
        raise ValueError("Review Pack requires stable analysis with a valid selected profile")
    root = Path(report["repository"]["root"]).resolve()
    eligible = sorted((item for item in report["items"] if item["actionability"] == "REVIEW_REQUIRED"),
                      key=lambda item: item["evidence_id"])
    reviews, deferred = [], []
    for item in eligible:
        if len(reviews) >= max_items:
            deferred.append({"evidence_id": item["evidence_id"], "reason": "ITEM_BUDGET"})
            continue
        usage = deepcopy(item.get("usage", {}))
        fallback = usage.pop("fallback", {"detected": False, "semantics": "NO_SYNTAX_EVIDENCE"})
        callers = _callers(report, item, max_source_files_per_item)
        review = {
            "review_id": "review:" + item["evidence_id"], "evidence_id": item["evidence_id"],
            "actionability": "REVIEW_REQUIRED", "reason": item["reason"],
            "dependency": deepcopy(item["dependency"]), "component": deepcopy(item["component"]),
            "target_component": item.get("target_component"),
            "declaration": deepcopy(item.get("declaration", {})), "usage": usage,
            "fallback": fallback, "callers": callers, "snippets": [],
            "question": "What support contract governs this dependency, and is its fallback sufficient for that contract?",
            "authority": "REVIEW_PROPOSAL_ONLY",
        }
        if _size(review) > max_context_bytes_per_item:
            deferred.append({"evidence_id": item["evidence_id"], "reason": "BASE_EVIDENCE_EXCEEDS_BYTE_BUDGET"})
            continue
        source = item["dependency"]["source"]
        references = [(source, item["dependency"].get("line"), "IMPORT")]
        references += [(source, line, "GUARD") for line in usage.get("guard_lines", [])[:4]]
        references += [(value.get("path", source), value.get("line"), "FALLBACK")
                       for value in fallback.get("evidence", [])[:4]]
        references += [(source, line, "USAGE") for line in usage.get("usage_lines", [])[:8]]
        references += [(value["path"], value["line"], "INCOMING_FILE_DEPENDENCY") for value in callers]
        files, seen = set(), set()
        for path, line, kind in references:
            if not isinstance(path, str) or not isinstance(line, int) or (path, line) in seen:
                continue
            if path not in files and len(files) >= max_source_files_per_item:
                continue
            seen.add((path, line))
            snippet = _snippet(root, path, line, kind)
            if snippet is None:
                continue
            review["snippets"].append(snippet)
            if _size(review) > max_context_bytes_per_item:
                review["snippets"].pop()
            else:
                files.add(path)
        if not any(snippet["kind"] == "IMPORT" for snippet in review["snippets"]):
            deferred.append({"evidence_id": item["evidence_id"],
                             "reason": "IMPORT_CONTEXT_UNAVAILABLE_WITHIN_BUDGET"})
            continue
        reviews.append(review)
    return {
        "format": "ptsip-dependency-review-pack/v1", "repository": deepcopy(report["repository"]),
        "component": report.get("component"), "snapshot": deepcopy(report.get("snapshot", {})),
        "summary": {"review_required_total": len(eligible), "selected_for_review": len(reviews),
                    "deferred": len(deferred), "ai_reviewed_items": 0},
        "budget": {"only": ["REVIEW_REQUIRED"], "max_items": max_items,
                   "max_context_bytes_per_item": max_context_bytes_per_item,
                   "max_source_files_per_item": max_source_files_per_item,
                   "byte_measure": "UTF-8 serialized standalone review object, ensure_ascii=false, indent=2, sort_keys=true"},
        "reviews": reviews, "deferred": deferred,
        "authority": "ADVISORY_ONLY", "conformance_evaluated": False,
    }


def write_review_pack(report: dict, output: str | Path | None = None, **budgets) -> tuple[dict, Path]:
    root = Path(report["repository"]["root"]).resolve()
    pack = build_review_pack(report, **budgets)
    observed = capture_snapshot(root).as_dict()
    expected = report.get("snapshot", {}).get("after", {})
    identity = ("is_git", "head", "status_fingerprint", "tracked_content_fingerprint", "tracked_files")
    if not expected or observed["observation_errors"] or any(observed[key] != expected.get(key) for key in identity):
        raise RuntimeError("Repository changed after dependency analysis; regenerate the Review Pack")
    data = json.dumps(pack, ensure_ascii=False, sort_keys=True, separators=(",", ":")) + "\n"
    if output is None:
        destination = (ptsip_home() / "dependency-reviews" / repository_fingerprint(root)
                       / (hashlib.sha256(data.encode("utf-8")).hexdigest()[:20] + ".json")).resolve()
        if destination.is_relative_to(root):
            raise ValueError("Default Review Pack state must stay outside the Consumer Repository")
    else:
        destination = Path(output).expanduser().resolve()
    if destination.suffix.lower() != ".json":
        raise ValueError("Review Pack output must be a JSON report path")
    if destination.is_relative_to(root):
        tracked, errors = git_tracked_files(root)
        if observed["is_git"] and errors:
            raise RuntimeError("Cannot verify Review Pack output against tracked files")
        protected = {str((root / rel).resolve()).casefold() for rel in tracked}
        if str(destination).casefold() in protected:
            raise ValueError("Review Pack output must not overwrite a tracked repository path")
        if any(part in {".git", "src"} for part in destination.relative_to(root).parts):
            raise ValueError("Review Pack output must not be written into source or Git state")
    if destination.exists():
        if destination.read_bytes() != data.encode("utf-8"):
            raise FileExistsError("Review Pack output already exists; choose a new report path")
        return pack, destination
    destination.parent.mkdir(parents=True, exist_ok=True)
    with destination.open("x", encoding="utf-8", newline="\n") as stream:
        stream.write(data)
    return pack, destination
