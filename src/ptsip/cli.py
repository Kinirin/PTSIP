from __future__ import annotations

import argparse
import json
import sys

from .adoption import apply_adoption, prepare_adoption
from .app.client import ControlPlaneClient
from .app.github_authority import CoordinationUnavailable, GithubControlPlaneClient
from .app.github_reconciliation import run_github_gate
from .app.local_client import LocalControlPlaneClient
from .clarification.generator import analyze_clarifications
from .clarification.i18n import resolve_language
from .clarification.render import render_console, render_issue
from .clarification.resolution import (
    DecisionAnswer,
    prepare_local_profile,
    validate_answer,
    write_prepared_local_profile,
)
from .clarification.resolution.model import CLASSIFICATIONS, LIFECYCLE_OWNERS
from .clarification.transports.github_issue import publish as publish_github_issues
from .conformance_engine import evaluate_conformance
from .constants import TOOL_VERSION
from .doctor import doctor
from .inspection.components import discover_component_candidates
from .inspection.dependencies_030 import scan_dependency_edges
from .inspection.inventory import collect_inventory
from .pilot.runner import run_pilot
from .repository.discover import RepositoryInfo, discover_repository
from .repository.snapshot import capture_snapshot, compare_snapshots
from .spec_identity import current_spec_identity
from .storage.local_state import repository_fingerprint
from .topology import migrate_topology
from .validation.profile import validate_profile

DecisionClient = ControlPlaneClient | LocalControlPlaneClient | GithubControlPlaneClient


def _configure_console_encoding() -> None:
    for stream in (sys.stdout, sys.stderr):
        reconfigure = getattr(stream, "reconfigure", None)
        if callable(reconfigure):
            reconfigure(errors="backslashreplace")


def _emit(payload: object, as_json: bool) -> None:
    if as_json:
        print(json.dumps(payload, indent=2, ensure_ascii=False))
        return
    if isinstance(payload, dict):
        for key, value in payload.items():
            print(f"{key}: {value}")
    else:
        print(payload)


def _yes_no(value: str) -> bool:
    return value.casefold() == "yes"


def _github_repository(repo: RepositoryInfo) -> str | None:
    if repo.remote and repo.remote.provider == "github" and repo.remote.repository:
        return repo.remote.repository
    return None


def _decision_repository(repo: RepositoryInfo) -> str:
    github = _github_repository(repo)
    if github:
        return github
    return f"local:{repository_fingerprint(repo.root)}"


def _coordination_backend(repo: RepositoryInfo, coordination: str | None) -> str:
    if coordination == "local":
        return "LOCAL"
    if coordination == "github":
        if not _github_repository(repo):
            raise RuntimeError("GitHub coordination requires a GitHub origin using owner/repository identity")
        return "GITHUB"
    return "GITHUB" if _github_repository(repo) else "LOCAL"


def _decision_backend(
    repo: RepositoryInfo,
    explicit_url: str | None,
    coordination: str | None,
) -> str:
    if explicit_url and coordination:
        raise ValueError("--control-plane and --coordination are mutually exclusive")
    if explicit_url:
        if not _github_repository(repo):
            raise RuntimeError("remote HTTP control plane requires a GitHub origin")
        return "REMOTE"
    return _coordination_backend(repo, coordination)


def _decision_client(
    repo: RepositoryInfo,
    explicit_url: str | None,
    coordination: str | None,
) -> tuple[str, DecisionClient]:
    backend = _decision_backend(repo, explicit_url, coordination)
    if backend == "REMOTE":
        assert explicit_url is not None
        return backend, ControlPlaneClient(explicit_url)
    if backend == "GITHUB":
        repository = _github_repository(repo)
        assert repository is not None
        return backend, GithubControlPlaneClient(repository)
    return backend, LocalControlPlaneClient(repo.root)


def _answer_from_mapping(payload: dict[str, object]) -> DecisionAnswer:
    answer = DecisionAnswer(
        classification=str(payload.get("classification", "")),
        purpose=str(payload.get("purpose", "")),
        shipped=bool(payload.get("shipped")),
        runtime_required=bool(payload.get("runtime_required")),
        lifecycle_owner=str(payload.get("lifecycle_owner", "")),
        executable=bool(payload.get("executable")),
    )
    validation = validate_answer(answer)
    if not validation.valid:
        raise ValueError("Authoritative decision answer is invalid: " + "; ".join(validation.errors))
    return answer


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

    p_conform = sub.add_parser(
        "conform",
        help="Evaluate Enforced PTSIP conformance from declared architecture and observed evidence",
    )
    p_conform.add_argument("path", nargs="?", default=".")
    p_conform.add_argument("--profile", help="Explicit project-profile path")
    p_conform.add_argument(
        "--artifact-evidence",
        action="append",
        help="Explicit ptsip-artifact-evidence/v1 JSON/YAML input; repeatable and read-only",
    )
    p_conform.add_argument(
        "--agent-decision",
        action="append",
        help="Explicit ptsip-agent-classification decision input; repeatable review evidence that never overrides the profile",
    )
    p_conform.add_argument(
        "--external-evidence",
        action="append",
        help="Explicit ptsip-external-evidence/v1 JSON/YAML dependency evidence; repeatable and subject/revision checked",
    )
    p_conform.add_argument("--json", action="store_true")

    p_clarify = sub.add_parser(
        "clarify",
        help="Generate deterministic human clarification requests instead of speculatively inferring missing architectural intent",
    )
    p_clarify.add_argument("path", nargs="?", default=".")
    p_clarify.add_argument("--profile", help="Explicit project-profile path")
    p_clarify.add_argument("--json", action="store_true")
    p_clarify.add_argument("--lang", choices=("en", "ko"), help="Question language; otherwise PTSIP_LANG, OS locale, then English")
    p_clarify.add_argument("--component", action="append", help="Limit clarification to a detected component candidate ID; repeatable")
    p_clarify.add_argument("--publish", choices=("github-issue",), help="Manual/offline fallback: explicitly publish clarification requests")
    p_clarify.add_argument("--repo", help="Override the detected GitHub origin using owner/repository; requires --publish github-issue")

    p_adopt = sub.add_parser(
        "adopt",
        help="Explicitly establish or extend project-owned PTSIP architecture from a discovered component candidate",
    )
    p_adopt.add_argument("path", nargs="?", default=".")
    p_adopt.add_argument("--profile", help="Explicit project-profile path; defaults to repository-root ptsip.yaml")
    p_adopt.add_argument("--component", required=True, help="Detected component candidate ID")
    p_adopt.add_argument("--classification", required=True, choices=CLASSIFICATIONS)
    p_adopt.add_argument("--purpose", required=True)
    p_adopt.add_argument("--shipped", required=True, choices=("yes", "no"))
    p_adopt.add_argument("--runtime-required", required=True, choices=("yes", "no"))
    p_adopt.add_argument(
        "--lifecycle-owner",
        required=True,
        choices=LIFECYCLE_OWNERS,
        help="Transitional compatibility fact; canonical Tool 0.3.6 profiles persist classification as lifecycle authority.",
    )
    p_adopt.add_argument("--executable", required=True, choices=("yes", "no"))
    p_adopt.add_argument(
        "--coordination",
        choices=("local", "github"),
        help="Decision coordination backend; default is GitHub for GitHub repositories and local otherwise",
    )
    p_adopt.add_argument("--actor", default="project-owner-session", help="Audit actor label for GitHub-coordinated adoption")
    p_adopt.add_argument("--apply", action="store_true", help="Apply the reviewed adoption plan; default is dry-run")
    p_adopt.add_argument("--json", action="store_true")

    p_gate = sub.add_parser(
        "gate",
        help="Poll/register architecture decisions only when an active coding-agent task requires them",
    )
    p_gate.add_argument("path", nargs="?", default=".")
    p_gate.add_argument("--profile", help="Explicit project-profile path; defaults to repository-root ptsip.yaml")
    p_gate.add_argument("--component", action="append", help="Limit the gate to a detected component candidate ID; repeatable")
    p_gate.add_argument("--lang", choices=("en", "ko"), help="Issue language; otherwise PTSIP_LANG, OS locale, then English")
    p_gate.add_argument(
        "--coordination",
        choices=("local", "github"),
        help="Decision coordination backend; default is GitHub for GitHub repositories and local otherwise",
    )
    p_gate.add_argument(
        "--control-plane",
        help="Optional hosted HTTP PTSIP control-plane override; mutually exclusive with --coordination",
    )
    p_gate.add_argument("--json", action="store_true")

    p_resolve = sub.add_parser(
        "resolve",
        help="Explicitly resolve a pending PTSIP decision from an active user/coding-agent session and apply the profile locally",
    )
    p_resolve.add_argument("path", nargs="?", default=".")
    p_resolve.add_argument("--profile", help="Explicit project-profile path; defaults to repository-root ptsip.yaml")
    p_resolve.add_argument("--decision", required=True, help="Decision ID returned by ptsip gate")
    p_resolve.add_argument("--classification", required=True, choices=CLASSIFICATIONS)
    p_resolve.add_argument("--purpose", required=True)
    p_resolve.add_argument("--shipped", required=True, choices=("yes", "no"))
    p_resolve.add_argument("--runtime-required", required=True, choices=("yes", "no"))
    p_resolve.add_argument(
        "--lifecycle-owner",
        required=True,
        choices=LIFECYCLE_OWNERS,
        help="Transitional compatibility fact; canonical Tool 0.3.6 profiles persist classification as lifecycle authority.",
    )
    p_resolve.add_argument("--executable", required=True, choices=("yes", "no"))
    p_resolve.add_argument("--actor", default="coding-agent-session", help="Audit actor label; no free-form inference is performed")
    p_resolve.add_argument(
        "--coordination",
        choices=("local", "github"),
        help="Decision coordination backend; default is GitHub for GitHub repositories and local otherwise",
    )
    p_resolve.add_argument(
        "--control-plane",
        help="Optional hosted HTTP PTSIP control-plane override; mutually exclusive with --coordination",
    )
    p_resolve.add_argument("--json", action="store_true")

    p_topology = sub.add_parser(
        "topology",
        help="Plan or explicitly apply a repository component-root migration without changing architecture classification",
    )
    p_topology.add_argument("path", nargs="?", default=".")
    p_topology.add_argument("--profile", help="Explicit project-profile path")
    p_topology.add_argument("--component", help="Component ID; required for component-based profiles")
    p_topology.add_argument("--from", dest="from_root", required=True, help="Existing repository-relative component root")
    p_topology.add_argument("--to", dest="to_root", required=True, help="Target repository-relative component root")
    p_topology.add_argument("--apply", action="store_true", help="Apply the reviewed migration plan; default is dry-run")
    p_topology.add_argument("--json", action="store_true")
    return parser


def main(argv: list[str] | None = None) -> int:
    _configure_console_encoding()
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
        if args.command == "conform":
            result = evaluate_conformance(
                args.path,
                args.profile,
                args.artifact_evidence,
                args.agent_decision,
                args.external_evidence,
            )
            _emit(result.report, args.json)
            if result.outcome == "CONFORMANT":
                return 0
            if result.outcome == "NON_CONFORMANT":
                return 5
            return 6
        if args.command == "topology":
            repo = discover_repository(args.path)
            result = migrate_topology(
                repo.root,
                args.profile,
                args.from_root,
                args.to_root,
                args.component,
                apply=args.apply,
            )
            _emit(result, args.json)
            return 0
        if args.command == "clarify":
            if args.repo and args.publish != "github-issue":
                raise ValueError("--repo requires --publish github-issue")
            language = resolve_language(args.lang)
            analysis = analyze_clarifications(args.path, args.component, args.profile)
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
        if args.command == "adopt":
            answer = DecisionAnswer(
                classification=args.classification,
                purpose=args.purpose.strip(),
                shipped=_yes_no(args.shipped),
                runtime_required=_yes_no(args.runtime_required),
                lifecycle_owner=args.lifecycle_owner,
                executable=_yes_no(args.executable),
            )
            preparation = prepare_adoption(args.path, args.component, answer, args.profile)
            backend = _coordination_backend(preparation.repository, args.coordination)
            payload = preparation.as_dict(apply=args.apply, backend=backend)
            if preparation.status not in {"ADOPTION_PLAN", "ALREADY_DECLARED"}:
                _emit(payload, args.json)
                return 8
            if not args.apply:
                _emit(payload, args.json)
                return 0

            authority: dict[str, object] | None = None
            if backend == "GITHUB":
                repo = preparation.repository
                if not repo.commit or not repo.branch:
                    raise RuntimeError("GitHub-coordinated ptsip adopt requires a checked-out Git branch and commit")
                repository = _github_repository(repo)
                assert repository is not None
                client = GithubControlPlaneClient(repository)
                candidate = preparation.candidate
                assert candidate is not None
                request_payload = (
                    preparation.request.as_dict()
                    if preparation.request is not None
                    else {
                        "component_id": candidate.id,
                        "include": list(candidate.include),
                        "anchors": list(candidate.anchors),
                        "evidence_ids": list(candidate.evidence_ids),
                        "missing_fields": [],
                        "reason_codes": [],
                        "status": "INCOMPLETE",
                    }
                )
                gate_payload = {
                    "id": preparation.request.id if preparation.request is not None else f"adopt-{candidate.id}",
                    "repository": repository,
                    "branch": repo.branch,
                    "subject_revision": repo.commit,
                    "component_id": preparation.request.component_id if preparation.request is not None else candidate.id,
                    "request": request_payload,
                }
                if preparation.status == "ALREADY_DECLARED":
                    gated = client.peek({"repository": repository, "request": request_payload})
                    if gated.get("status") == "NO_AUTHORITY_DECISION":
                        _emit(payload, args.json)
                        return 0
                else:
                    gated = client.gate(gate_payload)

                decision = gated.get("decision")
                if not isinstance(decision, dict):
                    raise RuntimeError("GitHub authority returned no adoption decision record")
                decision_id = str(decision.get("id", ""))
                if not decision_id:
                    raise RuntimeError("GitHub authority returned an adoption decision without an ID")
                gate_status = str(gated.get("status", ""))
                if gate_status == "DECISION_REQUIRED":
                    resolved_response = client.resolve(
                        {
                            "decision_id": decision_id,
                            "answer": answer.as_dict(),
                            "source": "PROJECT_ADOPTION",
                            "actor": args.actor,
                        }
                    )
                    resolved_decision = resolved_response.get("decision")
                    if not isinstance(resolved_decision, dict):
                        raise RuntimeError("GitHub authority returned no resolved adoption decision")
                    if resolved_response.get("status") == "ALREADY_RESOLVED":
                        if resolved_decision.get("answer") != answer.as_dict():
                            payload["status"] = "ALREADY_RESOLVED"
                            payload["message"] = "A different authoritative architecture decision won the GitHub coordination race."
                            payload["authority"] = resolved_response
                            _emit(payload, args.json)
                            return 9
                    elif resolved_response.get("status") != "RESOLVED" or not resolved_response.get("accepted"):
                        payload["status"] = str(resolved_response.get("status") or "CONFLICT")
                        payload["authority"] = resolved_response
                        _emit(payload, args.json)
                        return 8
                    authority = resolved_response
                elif gate_status == "RESOLVED_APPLICATION_REQUIRED":
                    if decision.get("answer") != answer.as_dict():
                        payload["status"] = "ALREADY_RESOLVED"
                        payload["message"] = "GitHub authority already contains a different architecture decision for this component scope."
                        payload["authority"] = gated
                        _emit(payload, args.json)
                        return 9
                    authority = gated
                else:
                    payload["status"] = gate_status or "CONFLICT"
                    payload["authority"] = gated
                    _emit(payload, args.json)
                    return 8

            status, profile_path, message = apply_adoption(preparation)
            payload["status"] = status
            payload["profile"] = {
                "path": profile_path,
                "projected_valid": status in {"ADOPTED", "ALREADY_DECLARED"},
            }
            payload["message"] = message
            if authority is not None:
                payload["authority"] = authority
            _emit(payload, args.json)
            return 0 if status in {"ADOPTED", "ALREADY_DECLARED"} else 8
        if args.command == "gate":
            language = resolve_language(args.lang)
            analysis = analyze_clarifications(args.path, args.component, args.profile)
            if not analysis.comparison.stable:
                raise RuntimeError("Repository state changed during decision-gate analysis; retry against a stable snapshot.")
            repo = analysis.repository
            backend = _decision_backend(repo, args.control_plane, args.coordination)

            if backend == "GITHUB":
                backend, client = _decision_client(repo, args.control_plane, args.coordination)
                payload, exit_code = run_github_gate(
                    analysis,
                    client,  # type: ignore[arg-type]
                    args.component,
                    args.profile,
                    language,
                )
                _emit(payload, args.json)
                return exit_code

            if not analysis.requests:
                payload = {
                    "status": "NO_DECISION_REQUIRED",
                    "backend": backend,
                    "repository": repo.as_dict(),
                    "decisions": [],
                }
                _emit(payload, args.json)
                return 0
            if not repo.commit or not repo.branch:
                raise RuntimeError("ptsip gate requires a checked-out Git branch and commit")
            backend, client = _decision_client(repo, args.control_plane, args.coordination)
            decision_repository = _decision_repository(repo)
            decisions: list[dict[str, object]] = []
            blocked = False
            errored = False
            baseline = analysis.after
            for request in analysis.requests:
                title, body = render_issue(request, language, repo.commit)
                response = client.gate(
                    {
                        "id": request.id,
                        "repository": decision_repository,
                        "branch": repo.branch,
                        "subject_revision": repo.commit,
                        "component_id": request.component_id,
                        "request": request.as_dict(),
                        "issue": {"title": title, "body": body},
                    }
                )
                status = str(response.get("status", ""))
                if backend == "GITHUB" and status == "RESOLVED_APPLICATION_REQUIRED":
                    decision = response.get("decision")
                    if not isinstance(decision, dict) or not isinstance(decision.get("answer"), dict):
                        response = {**response, "status": "CONFLICT", "message": "Resolved GitHub authority record has no valid answer."}
                        status = "CONFLICT"
                    else:
                        current = capture_snapshot(repo.root)
                        if not compare_snapshots(baseline, current).stable:
                            response = {**response, "status": "STALE", "message": "Repository changed before authoritative profile reconciliation."}
                            status = "STALE"
                        else:
                            try:
                                answer = _answer_from_mapping(decision["answer"])
                                prepared = prepare_local_profile(
                                    repo.root,
                                    request.component_id,
                                    list(request.include),
                                    answer,
                                    args.profile,
                                )
                                profile = write_prepared_local_profile(prepared)
                                application = client.application(
                                    {
                                        "decision_id": str(decision.get("id", "")),
                                        "status": "LOCAL_APPLIED",
                                        "applied_revision": repo.commit,
                                    }
                                )
                                response = {
                                    **response,
                                    "status": "RESOLVED",
                                    "reconciliation": {
                                        "status": "LOCAL_APPLIED",
                                        "profile_path": str(profile),
                                        "application": application,
                                    },
                                }
                                status = "RESOLVED"
                                baseline = capture_snapshot(repo.root)
                            except (FileNotFoundError, OSError, RuntimeError, ValueError) as exc:
                                response = {**response, "status": "CONFLICT", "message": str(exc)}
                                status = "CONFLICT"
                decisions.append(response)
                if status == "DECISION_REQUIRED":
                    blocked = True
                elif status in {"STALE", "CONFLICT", "INVALID", "RESOLVED_APPLICATION_REQUIRED"}:
                    errored = True
            overall = "DECISION_REQUIRED" if blocked else ("DECISION_ERROR" if errored else "RESOLVED")
            _emit(
                {
                    "status": overall,
                    "backend": backend,
                    "repository": repo.as_dict(),
                    "decisions": decisions,
                },
                args.json,
            )
            if blocked:
                return 7
            if errored:
                return 8
            return 0
        if args.command == "resolve":
            repo = discover_repository(args.path)
            if not repo.commit or not repo.branch:
                raise RuntimeError("ptsip resolve requires a checked-out Git branch and commit")
            backend, client = _decision_client(repo, args.control_plane, args.coordination)
            answer = DecisionAnswer(
                classification=args.classification,
                purpose=args.purpose.strip(),
                shipped=_yes_no(args.shipped),
                runtime_required=_yes_no(args.runtime_required),
                lifecycle_owner=args.lifecycle_owner,
                executable=_yes_no(args.executable),
            )
            validation = validate_answer(answer)
            if not validation.valid:
                _emit({"status": "CONFLICT", "validation": validation.as_dict()}, args.json)
                return 8

            lookup = client.decision({"decision_id": args.decision})
            decision = lookup.get("decision")
            if not isinstance(decision, dict):
                raise RuntimeError("Decision backend returned no decision record")
            if str(decision.get("repository", "")) != _decision_repository(repo):
                raise RuntimeError("Decision repository does not match the active local repository identity")
            if backend != "GITHUB":
                if str(decision.get("branch", "")) != repo.branch:
                    raise RuntimeError("Decision branch does not match the checked-out local branch; run ptsip gate first")
                if str(decision.get("subject_revision", "")) != repo.commit:
                    _emit(
                        {
                            "status": "STALE_REQUIRES_GATE",
                            "message": "Decision target revision differs from the active repository. Run ptsip gate for the affected component before retrying the same authoritative decision.",
                            "decision": decision,
                        },
                        args.json,
                    )
                    return 8

            existing_status = str(decision.get("status", ""))
            stored_answer = decision.get("answer")
            application_status = str(decision.get("application_status", ""))
            already_resolved_same = False
            if existing_status == "RESOLVED":
                if stored_answer != answer.as_dict():
                    _emit(
                        {
                            "status": "ALREADY_RESOLVED",
                            "accepted": False,
                            "message": "The authoritative decision already exists with different facts; this later answer cannot replace it.",
                            "decision": decision,
                        },
                        args.json,
                    )
                    return 9
                if backend != "GITHUB" and application_status in {"APPLIED", "LOCAL_APPLIED"}:
                    _emit({"status": "ALREADY_APPLIED", "decision": decision}, args.json)
                    return 0
                already_resolved_same = True
            elif existing_status != "PENDING":
                _emit({"status": existing_status or "DECISION_ERROR", "decision": decision}, args.json)
                return 8

            request = decision.get("request")
            if not isinstance(request, dict):
                raise RuntimeError("Decision record has no request payload")
            include = request.get("include")
            component_id = str(decision.get("component_id", ""))
            if not isinstance(include, list) or not component_id:
                raise RuntimeError("Decision request has no component include selectors")

            try:
                prepared = prepare_local_profile(
                    repo.root,
                    component_id,
                    [str(item) for item in include],
                    answer,
                    args.profile,
                )
            except (ValueError, RuntimeError) as exc:
                _emit({"status": "CONFLICT", "message": str(exc), "decision": decision}, args.json)
                return 8

            if already_resolved_same and backend == "GITHUB":
                resolved = decision
            else:
                response = client.resolve(
                    {
                        "decision_id": args.decision,
                        "answer": answer.as_dict(),
                        "actor": args.actor,
                    }
                )
                resolved_raw = response.get("decision")
                if response.get("status") == "ALREADY_RESOLVED" and isinstance(resolved_raw, dict):
                    if resolved_raw.get("answer") == answer.as_dict() and backend == "GITHUB":
                        resolved = resolved_raw
                    else:
                        _emit(response, args.json)
                        return 9
                elif response.get("status") != "RESOLVED" or not response.get("accepted"):
                    _emit(response, args.json)
                    return 9
                else:
                    if not isinstance(resolved_raw, dict):
                        raise RuntimeError("Decision backend returned no resolved decision record")
                    resolved = resolved_raw

            if backend != "GITHUB" and str(resolved.get("subject_revision", "")) != repo.commit:
                client.application({"decision_id": args.decision, "status": "STALE"})
                _emit({"status": "STALE_REQUIRES_GATE", "decision": resolved}, args.json)
                return 8

            try:
                profile = write_prepared_local_profile(prepared)
            except Exception:
                client.application({"decision_id": args.decision, "status": "FAILED"})
                raise
            application = client.application(
                {
                    "decision_id": args.decision,
                    "status": "LOCAL_APPLIED",
                    "applied_revision": repo.commit,
                }
            )
            _emit(
                {
                    "status": "RESOLVED",
                    "backend": backend,
                    "decision": resolved,
                    "profile_path": str(profile),
                    "application": application,
                },
                args.json,
            )
            return 0
    except CoordinationUnavailable as exc:
        if getattr(args, "json", False):
            _emit({"status": "COORDINATION_UNAVAILABLE", "message": str(exc)}, True)
        else:
            print(f"PTSIP coordination error: {exc}", file=sys.stderr)
        return 8
    except (FileNotFoundError, KeyError, PermissionError, OSError, RuntimeError, ValueError) as exc:
        print(f"PTSIP error: {exc}", file=sys.stderr)
        return 2
    return 2
