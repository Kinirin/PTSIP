from __future__ import annotations

import argparse
import subprocess
from pathlib import Path
from typing import Sequence


HOOKS_PATH = ".githooks"
PRE_COMMIT_HOOK = ".githooks/pre-commit"


class DeveloperSetupError(RuntimeError):
    pass


def _git(root: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        cwd=root,
        check=False,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def install_developer_hooks(root: str | Path = ".") -> Path:
    repo = Path(root).resolve()
    hook = repo / PRE_COMMIT_HOOK
    if not hook.is_file():
        raise DeveloperSetupError(
            f"Canonical pre-commit hook is missing: {PRE_COMMIT_HOOK}"
        )

    inside = _git(repo, "rev-parse", "--is-inside-work-tree")
    if inside.returncode != 0 or inside.stdout.strip().lower() != "true":
        raise DeveloperSetupError(f"Not a Git working tree: {repo}")

    configured = _git(repo, "config", "--local", "core.hooksPath", HOOKS_PATH)
    if configured.returncode != 0:
        raise DeveloperSetupError(
            configured.stderr.strip() or "Unable to configure core.hooksPath."
        )

    verify = _git(repo, "config", "--local", "--get", "core.hooksPath")
    if verify.returncode != 0 or verify.stdout.strip() != HOOKS_PATH:
        raise DeveloperSetupError(
            "core.hooksPath did not resolve to the canonical .githooks directory."
        )
    return hook


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Install PTSIP repository-local developer hook activation."
    )
    parser.add_argument("--repository", default=".")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        hook = install_developer_hooks(args.repository)
    except DeveloperSetupError as exc:
        print(f"PTSIP developer setup: FAIL: {exc}")
        return 2
    print("PTSIP developer setup: PASS")
    print(f"core.hooksPath: {HOOKS_PATH}")
    print(f"pre-commit: {hook}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
