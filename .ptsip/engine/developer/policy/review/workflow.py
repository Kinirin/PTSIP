"""GitHub Actions transport for the existing read-only Go review helper.

This adapter selects literal arguments and packages diagnostics. Policy routing,
section extraction and field inventory remain in the existing Go implementation.
"""

import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import subprocess
import sys
import uuid


def string_list(raw, label, required=False):
    values = json.loads(raw)
    if not isinstance(values, list) or (required and not values):
        raise ValueError(f"{label}_MUST_BE_A_JSON_ARRAY")
    if len(values) > 64:
        raise ValueError(f"{label}_EXCEEDS_64_ITEMS")
    if any(not isinstance(value, str) or not value for value in values):
        raise ValueError(f"{label}_MUST_CONTAIN_NONEMPTY_STRINGS")
    if len(set(values)) != len(values):
        raise ValueError(f"{label}_CONTAINS_DUPLICATES")
    return values


def local_path(value):
    path = PurePosixPath(value)
    if (not value or path.is_absolute() or str(path) != value
            or any(part in {".", ".."} for part in path.parts)
            or any(character in value for character in "\\:\x00\r\n")):
        raise ValueError("EXPECTED_CANONICAL_REPOSITORY_RELATIVE_PATH")
    return path


def command_for(root, targets, sources):
    command = ["go", "-C", str(root / ".ptsip/engine/developer"), "run",
               "./policy/review/cmd/policy-review", "--root", str(root)]
    for target in targets:
        path = local_path(target)
        if not target.startswith("developer/policy/") or path.suffix not in {".yaml", ".yml"}:
            raise ValueError("EXACT_DEVELOPER_POLICY_TARGET_REQUIRED")
        command.extend(["--target", target])
    for source in sources:
        path, separator, pointer = source.partition("#")
        local_path(path)
        if not separator or not pointer.startswith("/"):
            raise ValueError("EXPLICIT_SOURCE_SECTION_REQUIRED")
        command.extend(["--source", source])
    return command


def write_json(path, value):
    # ASCII JSON escapes preserve Unicode data through Windows and CI consoles.
    path.write_text(json.dumps(value, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")


def run(environ, invoke=subprocess.run):
    root = Path(environ["GITHUB_WORKSPACE"]).resolve()
    output = Path(environ["REVIEW_OUTPUT_DIRECTORY"]).resolve()
    if output == root or root in output.parents:
        raise ValueError("REVIEW_OUTPUT_MUST_BE_OUTSIDE_CHECKOUT")
    output.mkdir(parents=True, exist_ok=False)
    metadata = {"format": "ptsip-policy-review-actions/v1", "status": "FAIL",
                "schema_validation": "NOT_PERFORMED", "policy_conformance": "NOT_ASSERTED",
                "execution_authorized": False, "mutation_authorized": False,
                "repository": environ.get("GITHUB_REPOSITORY", ""),
                "run_id": environ.get("GITHUB_RUN_ID", ""),
                "run_attempt": environ.get("GITHUB_RUN_ATTEMPT", ""),
                "artifact_name": "policy-review-" + uuid.uuid4().hex,
                "workflow_ref": environ.get("GITHUB_WORKFLOW_REF", "")}
    code = 2
    try:
        targets = string_list(environ.get("REVIEW_TARGETS", "[]"), "TARGETS", required=True)
        sources = string_list(environ.get("REVIEW_SOURCES", "[]"), "SOURCES")
        expected = environ.get("REVIEW_EXPECTED_SHA", "")
        if expected and not re.fullmatch(r"[a-fA-F0-9]{40}", expected):
            raise ValueError("EXPECTED_SHA_MUST_BE_40_HEX_CHARACTERS")
        command = command_for(root, targets, sources)
        write_json(output / "request.json", {"targets": targets, "sources": sources, "expected_sha": expected})
        checkout = invoke(["git", "rev-parse", "HEAD"], cwd=root, capture_output=True, check=True)
        sha = checkout.stdout.decode("ascii").strip()
        if not re.fullmatch(r"[a-f0-9]{40}", sha):
            raise ValueError("INVALID_CHECKOUT_SHA")
        metadata["source_sha"] = sha
        if expected and expected.lower() != sha:
            raise ValueError("CHECKOUT_SHA_MISMATCH")
        if environ.get("GITHUB_SHA") and environ["GITHUB_SHA"].lower() != sha:
            raise ValueError("CHECKOUT_DOES_NOT_MATCH_WORKFLOW_EVENT")
        status = invoke(["git", "status", "--porcelain"], cwd=root, capture_output=True, check=True)
        metadata["worktree_dirty"] = bool(status.stdout.strip())
        if environ.get("GITHUB_ACTIONS") == "true" and metadata["worktree_dirty"]:
            raise ValueError("ACTIONS_CHECKOUT_MUST_BE_CLEAN")
        child_env = dict(environ, PTSIP_PYTHON=sys.executable)
        completed = invoke(command, cwd=root, env=child_env, capture_output=True, timeout=300, check=False)
        (output / "review.json").write_bytes(completed.stdout)
        (output / "stderr.log").write_bytes(completed.stderr)
        metadata["helper_exit_code"] = completed.returncode
        if completed.returncode:
            raise ValueError("POLICY_REVIEW_HELPER_FAILED")
        packet = json.loads(completed.stdout.decode("utf-8"))
        if packet.get("format") != "ptsip-non-normative-policy-review/v1":
            raise ValueError("UNEXPECTED_REVIEW_PACKET")
        if (packet.get("status") != "NON_NORMATIVE_REVIEW"
                or packet.get("schema_validation") != "NOT_PERFORMED"
                or packet.get("mutation_authorized") is not False
                or packet.get("execution_authorized", False) is not False):
            raise ValueError("REVIEW_AUTHORITY_BOUNDARY_CHANGED")
        unresolved = sum(bool(target.get("selection_problem")) for target in packet["targets"])
        findings = sum(len(target["field_findings"]) for target in packet["targets"])
        metadata["counts"] = {"targets": len(packet["targets"]), "sources": len(packet["sources"]),
                              "unresolved_selections": unresolved, "field_findings": findings,
                              "captured_files": len(packet["file_sha256"])}
        metadata["status"] = "REVIEW_REQUIRED" if unresolved or findings else "NON_NORMATIVE_REVIEW"
        metadata["review_sha256"] = hashlib.sha256(completed.stdout).hexdigest()
        code = 0
    except Exception as error:
        metadata["problem"] = str(error)
    write_json(output / "run.json", metadata)
    if not (output / "review.json").exists():
        write_json(output / "review.json", {"status": "FAIL", "problem": metadata.get("problem", "UNKNOWN"),
                                             "mutation_authorized": False})
    if environ.get("GITHUB_OUTPUT"):
        with Path(environ["GITHUB_OUTPUT"]).open("a", encoding="utf-8") as handle:
            handle.write(f"review_status={metadata['status']}\nsource_sha={metadata.get('source_sha', '')}\n")
            handle.write(f"artifact_name={metadata['artifact_name']}\n")
    if environ.get("GITHUB_STEP_SUMMARY"):
        with Path(environ["GITHUB_STEP_SUMMARY"]).open("a", encoding="utf-8") as handle:
            handle.write("## Policy review helper\n\n")
            handle.write(f"Result: **{metadata['status']}**\n\n")
            handle.write(f"Checkout SHA: `{metadata.get('source_sha', 'UNRESOLVED')}`\n\n")
            for name, value in metadata.get("counts", {}).items():
                handle.write(f"- {name}: {value}\n")
            handle.write("\nSchema validation was not performed. This result grants no policy approval or execution authority.\n")
    print(json.dumps({"status": metadata["status"], "source_sha": metadata.get("source_sha", ""),
                      "problem": metadata.get("problem")}, ensure_ascii=True))
    return code


if __name__ == "__main__":
    sys.exit(run(os.environ))
