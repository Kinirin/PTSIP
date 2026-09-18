from __future__ import annotations

import argparse
from pathlib import Path

import yaml

from developer.automation.agent_instruction_progressive import (
    ProgressiveReasoningError,
    check as check_progressive,
    migrate_level1,
)


class AgentInstructionActivationError(RuntimeError):
    pass


def activate(repository: str | Path) -> dict[str, object]:
    root = Path(repository).resolve()
    try:
        return migrate_level1(root)
    except (ProgressiveReasoningError, OSError, yaml.YAMLError) as exc:
        raise AgentInstructionActivationError(str(exc)) from exc


def check_activation(repository: str | Path) -> tuple[str, ...]:
    return check_progressive(Path(repository).resolve())


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Activate compact progressive repository agent instructions"
    )
    parser.add_argument("command", choices=("activate", "check"))
    parser.add_argument("--repository", default=".")
    args = parser.parse_args(argv)

    try:
        if args.command == "activate":
            result = activate(args.repository)
            print(
                f"Agent instruction Level 1 activated: {result['pass_count']} pass, "
                f"{result['unresolved_count']} unresolved; "
                f"integration={result.get('integration', 'UNKNOWN')}"
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
    except (
        AgentInstructionActivationError,
        ProgressiveReasoningError,
        OSError,
        yaml.YAMLError,
    ) as exc:
        print(f"agent-instruction-activation: {exc}")
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
