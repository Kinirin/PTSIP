from __future__ import annotations

import re
import runpy
import subprocess
import sys
import tomllib
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parents[2]
REQUIRED_SPEC_PATHS = (
    "spec/PTSIP-SPEC.md",
    "spec/PTSIP-CONFORMANCE.md",
    "spec/PTSIP-TERMINOLOGY.md",
    "spec/PTSIP-GOVERNANCE.md",
    "spec/PTSIP-RESPONSIBILITY-MAP.md",
)


def _git(*args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        cwd=ROOT,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )


def main() -> int:
    errors: list[str] = []

    with (ROOT / "pyproject.toml").open("rb") as handle:
        package_version = str(tomllib.load(handle)["project"]["version"])

    constants = runpy.run_path(str(ROOT / "src/ptsip/constants.py"))
    spec_version = str(constants["SPEC_VERSION"])
    spec_revision = str(constants["SPEC_REVISION"])
    spec_source = str(constants["SPEC_SOURCE"])

    expected_spec_version = f"{package_version}-draft"
    if spec_version != expected_spec_version:
        errors.append(
            f"SPEC_VERSION is {spec_version!r}; Tool {package_version} requires "
            f"{expected_spec_version!r}."
        )

    if not re.fullmatch(r"[0-9a-f]{40}", spec_revision):
        errors.append("SPEC_REVISION must be a full 40-character lowercase Git commit SHA.")

    profile = yaml.safe_load((ROOT / "ptsip.yaml").read_text(encoding="utf-8"))
    profile_ptsip = profile.get("ptsip", {})
    profile_spec = profile_ptsip.get("specification", {})

    if profile_ptsip.get("version") != spec_version:
        errors.append("ptsip.yaml ptsip.version does not match SPEC_VERSION.")
    if profile_spec.get("revision") != spec_revision:
        errors.append("ptsip.yaml specification.revision does not match SPEC_REVISION.")
    if profile_spec.get("source") != spec_source:
        errors.append("ptsip.yaml specification.source does not match SPEC_SOURCE.")

    spec_note = ROOT / "releasenote" / f"spec-{spec_version}.md"
    if not spec_note.is_file() or not spec_note.read_text(encoding="utf-8").strip():
        errors.append(
            f"Missing or empty Specification release note: {spec_note.relative_to(ROOT)}"
        )

    if re.fullmatch(r"[0-9a-f]{40}", spec_revision):
        revision_check = _git("rev-parse", "--verify", f"{spec_revision}^{{commit}}")
        if revision_check.returncode != 0:
            errors.append(f"SPEC_REVISION does not resolve to a Git commit: {spec_revision}")
        else:
            ancestor_check = _git("merge-base", "--is-ancestor", spec_revision, "HEAD")
            if ancestor_check.returncode != 0:
                errors.append(
                    f"SPEC_REVISION {spec_revision} is not an ancestor of release source HEAD."
                )

            for path in REQUIRED_SPEC_PATHS:
                path_check = _git("cat-file", "-e", f"{spec_revision}:{path}")
                if path_check.returncode != 0:
                    errors.append(
                        f"Canonical Specification path {path!r} is absent at SPEC_REVISION."
                    )

    if errors:
        print("PTSIP release Specification contract: FAIL", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print("PTSIP release Specification contract: PASS")
    print(f"Tool version: {package_version}")
    print(f"Specification: {spec_version}")
    print(f"Specification revision: {spec_revision}")
    print(f"Specification note: releasenote/spec-{spec_version}.md")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
