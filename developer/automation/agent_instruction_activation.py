from __future__ import annotations

import argparse
from pathlib import Path

import yaml

from developer.automation.agent_instruction_materializer import (
    DEFAULT_OUTPUT_ROOT,
    DEFAULT_SOURCE,
    check_materialization,
)
from developer.automation.agent_instruction_progressive import (
    check as check_progressive,
    migrate_level1,
)


class AgentInstructionActivationError(RuntimeError):
    pass


def bootstrap_text() -> str:
    return """# AGENTS.md

This repository uses progressive repository-local agent instruction reasoning under .agent/.

Before repository work, classify the current operation as READ, MODIFY, PLAN, VERIFY, or RELEASE, then resolve the bounded instruction set mechanically:

    python -m developer.automation.agent_instruction_entry_resolver <READ|MODIFY|PLAN|VERIFY|RELEASE>

The resolver consumes the deepest active machine stage plus only the natural-language residual that has not yet been mechanized. It must not re-run a previous passed classification level.

Level 1 UNRESOLVED items remain natural language until they are explicitly resolved. They are not coerced to OTHER, and Level 2 automatic classification must not begin while Level 1 unresolved items remain.

OTHER is intentionally excluded from normal Level 1 entry. Widen only when context or goal material is required:

    python -m developer.automation.agent_instruction_entry_resolver <OPERATION> --include OTHER

Do not use provenance/history as a reasoning input. Re-run the entry resolver whenever the operation changes. Resolver failure is fail-closed for repository instruction loading.
"""


def activate(repository: str | Path) -> dict[str, object]:
    root = Path(repository).resolve()
    source = root / DEFAULT_SOURCE
    if not source.is_file():
        raise AgentInstructionActivationError("AGENTS.md is missing")
    stale = check_materialization(
        DEFAULT_SOURCE,
        root=root,
        output_root=DEFAULT_OUTPUT_ROOT,
    )
    if stale:
        raise AgentInstructionActivationError(
            "materialization is stale: " + ", ".join(stale)
        )

    result = migrate_level1(root)
    source.write_text(bootstrap_text(), encoding="utf-8", newline="\n")
    return result


def check_activation(repository: str | Path) -> tuple[str, ...]:
    root = Path(repository).resolve()
    errors = list(check_progressive(root))
    source = root / DEFAULT_SOURCE
    if not source.is_file() or source.read_text(encoding="utf-8") != bootstrap_text():
        errors.append("AGENTS_BOOTSTRAP_STALE")
    return tuple(errors)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Activate progressive repository agent instruction reasoning"
    )
    parser.add_argument("command", choices=("activate", "check"))
    parser.add_argument("--repository", default=".")
    args = parser.parse_args(argv)

    try:
        if args.command == "activate":
            result = activate(args.repository)
            level2 = "allowed" if result["next_level_allowed"] else "blocked"
            print(
                f"Agent instruction Level 1 activated: {result['pass_count']} pass, "
                f"{result['unresolved_count']} unresolved; Level 2 {level2}"
            )
            return 0

        errors = check_activation(args.repository)
        if errors:
            print("Agent instruction progressive activation: STALE")
            for error in errors:
                print(error)
            return 1
        print("Agent instruction progressive activation: CURRENT")
        return 0
    except (AgentInstructionActivationError, OSError, yaml.YAMLError) as exc:
        print(f"agent-instruction-activation: {exc}")
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
