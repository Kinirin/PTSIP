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
CANONICAL_MACHINE_READABLE_PATHS = (
    "schemas/ptsip-profile.schema.json",
    "schemas/ptsip-artifact-evidence.schema.json",
    "schemas/ptsip-agent-classification.schema.json",
    "schemas/ptsip-diagnostic.schema.json",
    "registry/ptsip-registry.yaml",
)
EMBEDDED_MACHINE_READABLE_PATHS = (
    "src/ptsip/specdata/ptsip-profile.schema.json",
    "src/ptsip/specdata/ptsip-artifact-evidence.schema.json",
    "src/ptsip/specdata/ptsip-agent-classification.schema.json",
    "src/ptsip/specdata/ptsip-diagnostic.schema.json",
    "src/ptsip/specdata/ptsip-registry.yaml",
)
RELEASE_BOUND_SPEC_PATHS = (
    *REQUIRED_SPEC_PATHS,
    *CANONICAL_MACHINE_READABLE_PATHS,
    *EMBEDDED_MACHINE_READABLE_PATHS,
)
CANONICAL_EMBEDDED_PAIRS = tuple(
    zip(CANONICAL_MACHINE_READABLE_PATHS, EMBEDDED_MACHINE_READABLE_PATHS)
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


def _git_object_id(revision: str, path: str) -> str | None:
    result = _git("rev-parse", "--verify", f"{revision}:{path}")
    if result.returncode != 0:
        return None
    object_id = result.stdout.strip()
    return object_id or None


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
    spec_note_text = ""
    if not spec_note.is_file():
        errors.append(f"Missing Specification release note: {spec_note.relative_to(ROOT)}")
    else:
        spec_note_text = spec_note.read_text(encoding="utf-8")
        if not spec_note_text.strip():
            errors.append(f"Empty Specification release note: {spec_note.relative_to(ROOT)}")
        elif spec_revision not in spec_note_text:
            errors.append(
                "Specification release note does not record the exact bound SPEC_REVISION."
            )

    tool_note = ROOT / "releasenote" / f"{package_version}.md"
    tool_note_text = ""
    if not tool_note.is_file():
        errors.append(f"Missing Tool release note: {tool_note.relative_to(ROOT)}")
    else:
        tool_note_text = tool_note.read_text(encoding="utf-8")
        if not tool_note_text.strip():
            errors.append(f"Empty Tool release note: {tool_note.relative_to(ROOT)}")
        else:
            if not re.search(r"(?m)^##\s+\S", tool_note_text):
                errors.append(
                    f"Tool release note has no categorized sections: {tool_note.relative_to(ROOT)}"
                )
            if package_version not in tool_note_text:
                errors.append("Tool release note does not record the package version.")
            if spec_version not in tool_note_text or spec_revision not in tool_note_text:
                errors.append(
                    "Tool release note does not record the exact Tool/Specification release binding."
                )

    release_index = ROOT / "releasenote" / "README.md"
    if not release_index.is_file():
        errors.append("Missing release-note index: releasenote/README.md")
    else:
        release_index_text = release_index.read_text(encoding="utf-8")
        tool_index_marker = f"| `{package_version}` |"
        spec_index_marker = f"| `{spec_version}` |"
        if tool_index_marker not in release_index_text:
            errors.append(
                f"Release-note index does not recognize Tool {package_version}."
            )
        if spec_index_marker not in release_index_text:
            errors.append(
                f"Release-note index does not recognize Specification {spec_version}."
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

            for path in RELEASE_BOUND_SPEC_PATHS:
                bound_object = _git_object_id(spec_revision, path)
                if bound_object is None:
                    errors.append(
                        f"Release-bound Specification asset {path!r} is absent at SPEC_REVISION."
                    )
                    continue

                head_object = _git_object_id("HEAD", path)
                if head_object is None:
                    errors.append(
                        f"Release-bound Specification asset {path!r} is absent from release source HEAD."
                    )
                    continue

                if head_object != bound_object:
                    errors.append(
                        f"Release-bound Specification asset {path!r} differs from SPEC_REVISION "
                        f"{spec_revision}."
                    )

            for canonical, embedded in CANONICAL_EMBEDDED_PAIRS:
                canonical_object = _git_object_id("HEAD", canonical)
                embedded_object = _git_object_id("HEAD", embedded)
                if canonical_object is None or embedded_object is None:
                    continue
                if canonical_object != embedded_object:
                    errors.append(
                        f"Embedded Specification asset {embedded!r} differs from canonical "
                        f"asset {canonical!r} at release source HEAD."
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
    print(f"Tool note: releasenote/{package_version}.md")
    print(f"Specification note: releasenote/spec-{spec_version}.md")
    print(f"Release-bound Specification assets: {len(RELEASE_BOUND_SPEC_PATHS)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
