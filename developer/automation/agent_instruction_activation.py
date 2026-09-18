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
    bootstrap_text,
    check as check_progressive,
    migrate_level1,
)


class AgentInstructionActivationError(RuntimeError):
    pass


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
            print(
                f"Agent instruction Level 1 activated: {result['pass_count']} pass, "
                f"{result['unresolved_count']} unresolved; "
                f"{result['next_level_candidate_count']} Level 2 candidate(s)"
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
