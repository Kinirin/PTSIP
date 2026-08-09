from __future__ import annotations

import argparse
import json
import sys

from .clarification.generator import analyze_clarifications
from .clarification.i18n import resolve_language
from .clarification.render import render_console
from .clarification.transports.github_issue import publish as publish_github_issues
from .constants import TOOL_VERSION
from .doctor import doctor
from .inspection.components import discover_component_candidates
from .inspection.dependencies import scan_dependency_edges
from .inspection.inventory import collect_inventory
from .pilot.runner import run_pilot
from .repository.discover import discover_repository
from .repository.snapshot import capture_snapshot, compare_snapshots
from .spec_identity import current_spec_identity
from .validation.profile import validate_profile


def _emit(payload: object, as_json: bool) -> None:
    if as_json:
        print(json.dumps(payload, indent=2, ensure_ascii=False))
        return
    if isinstance(payload, dict):
        for key, value in payload.items():
            print(f"{key}: {value}")
    else:
        print(payload)


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="ptsip", description="Reference development tooling for PTSIP")
    parser.add_argument("--version", action="version", version=f"PTSIP Tool {TOOL_VERSION}")
    sub = parser.add_subparsers(dest="command", required=True)

    p_spec = sub.add_parser("spec", help="Show the canonical specification binding embedded in this tool")
    p_spec.add_argument("--json", action="store_true")

    p_doctor = sub.add_parser("doctor", help="Check the local environment without modifying the target repository")
    p_doctor.add_argument("path", nargs="?", default=".")
    p_doctor.add_argument("--json", action="store_true")

    p_inspect = sub.add_parser("inspect", help="Read-only repository evidence collection")
    p_inspect.add_argument("path", nargs="?", default=".")
    p_inspect.add_argument("--json", action="store_true")

    p_pilot = sub.add_parser("pilot", help="Run a read-only PTSIP pilot and store the report outside the repository by default")
    p_pilot.add_argument("path", nargs="?", default=".")
    p_pilot.add_argument("--report", help="Explicit report destination; may be inside the repository only when the user chooses it")
    p_pilot.add_argument("--json", action="store_true")

    p_validate = sub.add_parser("validate", help="Validate an existing PTSIP project profile")
    p_validate.add_argument("path", nargs="?", default=".")
    p_validate.add_argument("--profile", help="Explicit project-profile path")
    p_validate.add_argument("--json", action="store_true")

    p_clarify = sub.add_parser(
        "clarify",
        help="Generate deterministic human clarification requests instead of speculatively inferring missing architectural intent",
    )
    p_clarify.add_argument("path", nargs="?", default=".")
    p_clarify.add_argument("--json", action="store_true")
    p_clarify.add_argument("--lang", choices=("en", "ko"), help="Question language; otherwise PTSIP_LANG, OS locale, then English")
    p_clarify.add_argument("--component", action="append", help="Limit clarification to a detected component candidate ID; repeatable")
    p_clarify.add_argument("--publish", choices=("github-issue",), help="Explicitly publish clarification requests to an external transport")
    p_clarify.add_argument("--repo", help="Override the detected GitHub origin using owner/repository; requires --publish github-issue")
    return parser


def main(argv: list[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        if args.command == "spec":
            _emit(current_spec_identity().as_dict(), args.json)
            return 0
        if args.command == "doctor":
            result = doctor(args.path)
            _emit(result, args.json)
            return 0 if result["python_ok"] and result["target_exists"] else 2
        if args.command == "inspect":
            repo = discover_repository(args.path)
            before = capture_snapshot(repo.root)
            inventory = collect_inventory(repo.root)
            dependencies = scan_dependency_edges(repo.root)
            candidates = discover_component_candidates(repo.root, inventory, dependencies)
            after = capture_snapshot(repo.root)
            comparison = compare_snapshots(before, after)
            payload = {
                "repository": repo.as_dict(),
                "snapshot": {
                    "before": before.as_dict(),
                    "after": after.as_dict(),
                    "comparison": comparison.as_dict(),
                },
                "non_intrusion": {
                    "status": "VERIFIED_NO_OBSERVED_CHANGE" if comparison.stable else "CHANGE_OBSERVED_DURING_ANALYSIS",
                    "analysis_read_only_by_design": True,
                },
                "inventory": inventory.as_dict(),
                "components": {
                    "status": "CANDIDATES_ONLY",
                    "candidates": [candidate.as_dict() for candidate in candidates],
                },
                "dependencies": dependencies.as_dict(),
            }
            _emit(payload, args.json)
            return 0 if comparison.stable else 4
        if args.command == "pilot":
            result = run_pilot(args.path, args.report)
            _emit({**result.report, "report_path": str(result.report_path)}, args.json)
            stable = bool(result.report["snapshot"]["comparison"]["stable"])
            return 0 if stable else 4
        if args.command == "validate":
            repo = discover_repository(args.path)
            result = validate_profile(repo.root, args.profile)
            _emit(result.as_dict(), args.json)
            return 0 if result.valid else 3
        if args.command == "clarify":
            if args.repo and args.publish != "github-issue":
                raise ValueError("--repo requires --publish github-issue")
            language = resolve_language(args.lang)
            analysis = analyze_clarifications(args.path, args.component)
            payload = analysis.as_dict(language)
            publications = ()
            if args.publish == "github-issue":
                if not analysis.comparison.stable:
                    raise RuntimeError("Repository state changed during clarification analysis; refusing to publish questions from invalidated evidence.")
                publications = publish_github_issues(
                    repository_root=analysis.repository.root,
                    remote=analysis.repository.remote,
                    repository_revision=analysis.repository.commit,
                    requests=analysis.requests,
                    language=language,
                    repo_override=args.repo,
                )
                payload["publication"] = {
                    "transport": "github-issue",
                    "results": [item.as_dict() for item in publications],
                }
            if args.json:
                _emit(payload, True)
            else:
                print(render_console(analysis.requests, language))
                for item in publications:
                    print(f"github_issue[{item.status}]: {item.issue_url}")
            return 0 if analysis.comparison.stable else 4
    except (FileNotFoundError, PermissionError, OSError, RuntimeError, ValueError) as exc:
        print(f"PTSIP error: {exc}", file=sys.stderr)
        return 2
    return 2
