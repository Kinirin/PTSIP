from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Mapping

import yaml


INDEX_REF = Path(".agent/index.yaml")
LOCAL_CLI_ONLY = "LOCAL_CLI_ONLY"
MCP_STDIO_READY = "MCP_STDIO_READY"
MCP_ABSENT = "ABSENT"
MCP_READY = "READY"
MCP_IMPLEMENTATION_NOT_AVAILABLE = "NOT_AVAILABLE"
LOCAL_RESOLVER_COMMAND = (
    "python -m developer.automation.agent_instruction_entry_resolver "
    "<READ|MODIFY|PLAN|VERIFY|RELEASE>"
)
STATUS_COMMAND = (
    "python -m developer.automation.agent_integration status --json"
)
INTENDED_MCP_INSTALL_COMMAND = (
    "python -m developer.automation.agent_integration "
    "install-mcp --user-approved"
)


class AgentIntegrationError(RuntimeError):
    pass


def entry_directive(mode: str = LOCAL_CLI_ONLY) -> str:
    if mode not in {LOCAL_CLI_ONLY, MCP_STDIO_READY}:
        raise AgentIntegrationError(f"unsupported integration mode: {mode}")
    mcp_state = MCP_READY if mode == MCP_STDIO_READY else MCP_ABSENT
    return (
        'PTSIP_AGENT_ENTRY version=1 '
        'index=".agent/index.yaml" '
        f'integration="{mode}" '
        f'mcp="{mcp_state}"'
    )


def default_integration_contract() -> dict[str, object]:
    return {
        "mode": LOCAL_CLI_ONLY,
        "local_cli": {
            "state": "READY",
            "resolver_command": LOCAL_RESOLVER_COMMAND,
        },
        "mcp": {
            "state": MCP_ABSENT,
            "optional": True,
            "transport": "STDIO",
            "implementation_state": MCP_IMPLEMENTATION_NOT_AVAILABLE,
            "user_approval_required_before_install": True,
            "automatic_install_without_user_approval": False,
            "offer_to_user_now": False,
            "intended_install_command": INTENDED_MCP_INSTALL_COMMAND,
        },
        "status_command": STATUS_COMMAND,
    }


def _load_index(root: Path) -> dict[str, object]:
    path = root / INDEX_REF
    if not path.is_file():
        raise AgentIntegrationError(f"agent index missing: {path}")
    value = yaml.safe_load(path.read_text(encoding="utf-8"))
    if not isinstance(value, Mapping):
        raise AgentIntegrationError("agent index must be a mapping")
    return dict(value)


def status(repository: str | Path = ".") -> dict[str, object]:
    root = Path(repository).resolve()
    index = _load_index(root)
    integration = index.get("integration")
    if not isinstance(integration, Mapping):
        raise AgentIntegrationError("agent integration contract missing")

    mode = integration.get("mode")
    local_cli = integration.get("local_cli")
    mcp = integration.get("mcp")
    if mode not in {LOCAL_CLI_ONLY, MCP_STDIO_READY}:
        raise AgentIntegrationError("agent integration mode invalid")
    if not isinstance(local_cli, Mapping) or not isinstance(mcp, Mapping):
        raise AgentIntegrationError("agent integration contract invalid")

    implementation_state = mcp.get("implementation_state")
    mcp_state = mcp.get("state")
    offer = (
        mode == LOCAL_CLI_ONLY
        and mcp_state == MCP_ABSENT
        and implementation_state == "READY"
    )
    return {
        "schema_version": "ptsip-agent-integration-status/v1",
        "mode": mode,
        "entry_directive": entry_directive(str(mode)),
        "local_cli": dict(local_cli),
        "mcp": {
            **dict(mcp),
            "offer_to_user_now": offer,
        },
    }


def install_mcp(
    repository: str | Path = ".",
    *,
    user_approved: bool = False,
) -> int:
    current = status(repository)
    mcp = current["mcp"]
    assert isinstance(mcp, dict)
    if mcp.get("implementation_state") != "READY":
        print(
            "PTSIP MCP integration is not available yet; "
            "current mode remains LOCAL_CLI_ONLY"
        )
        return 3
    if not user_approved:
        print("PTSIP MCP installation requires explicit user approval")
        return 4
    raise AgentIntegrationError(
        "MCP implementation is marked READY but no installer is bound"
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Inspect or install repository-local PTSIP agent integration"
    )
    parser.add_argument("command", choices=("status", "install-mcp"))
    parser.add_argument("--repository", default=".")
    parser.add_argument("--json", action="store_true")
    parser.add_argument("--user-approved", action="store_true")
    args = parser.parse_args(argv)

    try:
        if args.command == "install-mcp":
            return install_mcp(
                args.repository,
                user_approved=args.user_approved,
            )
        result = status(args.repository)
    except (AgentIntegrationError, OSError, yaml.YAMLError) as exc:
        print(f"agent-integration: {exc}")
        return 2

    if args.json:
        print(json.dumps(result, ensure_ascii=True, indent=2))
    else:
        mcp = result["mcp"]
        assert isinstance(mcp, dict)
        print(
            f"PTSIP agent integration: {result['mode']}; "
            f"MCP={mcp.get('state')} "
            f"implementation={mcp.get('implementation_state')}"
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
