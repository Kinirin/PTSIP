from __future__ import annotations

import re
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
    "spec/PTSIP-DRAFT-PROFILE-TRANSITION.md",
)
CANONICAL_MACHINE_READABLE_PATHS = (
    "schemas/ptsip-profile.schema.json",
    "schemas/ptsip-profile-pp-1.01.schema.json",
    "schemas/ptsip-artifact-evidence.schema.json",
    "schemas/ptsip-agent-classification.schema.json",
    "schemas/ptsip-diagnostic.schema.json",
    "schemas/ptsip-normalized-evidence.schema.json",
    "registry/ptsip-registry.yaml",
)
EMBEDDED_MACHINE_READABLE_PATHS = (
    "src/ptsip/specdata/ptsip-profile.schema.json",
    "src/ptsip/specdata/ptsip-profile-pp-1.01.schema.json",
    "src/ptsip/specdata/ptsip-artifact-evidence.schema.json",
    "src/ptsip/specdata/ptsip-agent-classification.schema.json",
    "src/ptsip/specdata/ptsip-diagnostic.schema.json",
    "src/ptsip/specdata/ptsip-normalized-evidence.schema.json",
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

    source_root = str(ROOT / "src")
    if source_root not in sys.path:
        sys.path.insert(0, source_root)

    from ptsip.constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION, TOOL_VERSION
    from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION
    from ptsip.spec_identity import current_spec_identity
    from ptsip.specification_binding import current_target_specification_binding

    tool_version = TOOL_VERSION
    pp_version = CURRENT_PROJECT_PROFILE_VERSION
    spec_version = SPEC_VERSION
    spec_revision = SPEC_REVISION
    spec_source = SPEC_SOURCE
    current_binding = current_target_specification_binding()
    runtime_identity = current_spec_identity()

    if package_version != tool_version:
        errors.append(
            f"Package version {package_version!r} does not match Tool runtime {tool_version!r}."
        )
    if pp_version != "pp.1.01":
        errors.append(f"Current Project Profile contract is {pp_version!r}; expected 'pp.1.01'.")
    if (
        current_binding.family != spec_version
        or current_binding.source != spec_source
        or current_binding.revision != spec_revision
    ):
        errors.append(
            "Current Specification capability binding does not match Tool constants."
        )
    if (
        runtime_identity.tool_version != tool_version
        or runtime_identity.project_profile_contract_version != pp_version
        or runtime_identity.version != spec_version
        or runtime_identity.source != spec_source
        or runtime_identity.revision != spec_revision
    ):
        errors.append("Runtime spec identity does not expose the independent release identities.")

    if not re.fullmatch(r"[0-9a-f]{40}", spec_revision):
        errors.append("SPEC_REVISION must be a full 40-character lowercase Git commit SHA.")

    profile = yaml.safe_load((ROOT / "ptsip.yaml").read_text(encoding="utf-8"))
    profile_ptsip = profile.get("ptsip", {})
    profile_spec = profile_ptsip.get("specification", {})

    if profile_ptsip.get("version") != pp_version:
        errors.append("ptsip.yaml ptsip.version does not match the current PP contract.")
    if profile_spec.get("family") != spec_version:
        errors.append("ptsip.yaml specification.family does not match SPEC_VERSION.")
    if profile_spec.get("revision") != spec_revision:
        errors.append("ptsip.yaml specification.revision does not match SPEC_REVISION.")
    if profile_spec.get("source") != spec_source:
        errors.append("ptsip.yaml specification.source does not match SPEC_SOURCE.")

    maintained_profiles = (
        "profiles/example.ptsip.yaml",
        "profiles/hybrid-python-package.ptsip.yaml",
        "profiles/template-python-package.ptsip.yaml",
    )
    for relative_path in maintained_profiles:
        maintained = yaml.safe_load((ROOT / relative_path).read_text(encoding="utf-8"))
        maintained_ptsip = maintained.get("ptsip", {})
        maintained_spec = maintained_ptsip.get("specification", {})
        if maintained_ptsip.get("version") != pp_version or maintained_spec != {
            "family": spec_version,
            "source": spec_source,
            "revision": spec_revision,
        }:
            errors.append(f"Maintained profile {relative_path!r} has a stale release binding.")

    spec_note = ROOT / "releasenote" / "specification" / f"{spec_version}.md"
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

    tool_note = ROOT / "releasenote" / "tool" / f"{package_version}.md"
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
            if (
                pp_version not in tool_note_text
                or spec_version not in tool_note_text
                or spec_revision not in tool_note_text
            ):
                errors.append(
                    "Tool release note does not record the exact Tool/PP/Specification binding."
                )

    pp_note = ROOT / "releasenote" / "project-profile" / f"{pp_version}.md"
    if not pp_note.is_file():
        errors.append(f"Missing Project Profile release note: {pp_note.relative_to(ROOT)}")
    else:
        pp_note_text = pp_note.read_text(encoding="utf-8")
        if pp_version not in pp_note_text or spec_revision not in pp_note_text:
            errors.append("Project Profile release note does not record the current binding.")

    release_index = ROOT / "releasenote" / "README.md"
    if not release_index.is_file():
        errors.append("Missing release-note index: releasenote/README.md")
    else:
        release_index_text = release_index.read_text(encoding="utf-8")
        tool_index_marker = f"tool/{package_version}.md"
        pp_index_marker = f"project-profile/{pp_version}.md"
        spec_index_marker = f"specification/{spec_version}.md"
        if tool_index_marker not in release_index_text:
            errors.append(
                f"Release-note index does not recognize Tool {package_version}."
            )
        if pp_index_marker not in release_index_text:
            errors.append(
                f"Release-note index does not recognize Project Profile {pp_version}."
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
    print(f"Tool version: {tool_version}")
    print(f"Project Profile: {pp_version}")
    print(f"Specification family: {spec_version}")
    print(f"Specification revision: {spec_revision}")
    print(f"Tool note: releasenote/tool/{package_version}.md")
    print(f"Project Profile note: releasenote/project-profile/{pp_version}.md")
    print(f"Specification note: releasenote/specification/{spec_version}.md")
    print(f"Release-bound Specification assets: {len(RELEASE_BOUND_SPEC_PATHS)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
