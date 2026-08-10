from __future__ import annotations

import os
import shutil
import subprocess
import tempfile
from dataclasses import asdict, dataclass
from pathlib import Path, PurePosixPath

import yaml

from .clarification.resolution import validate_projected_payload
from .repository.discover import discover_repository
from .repository.snapshot import repository_files
from .validation.profile import find_profile, validate_profile


_PATH_FIELDS = ("include", "exclude", "manifests", "analysis_inputs")
_SOURCE_SUFFIXES = {
    ".py",
    ".pyi",
    ".js",
    ".jsx",
    ".mjs",
    ".cjs",
    ".ts",
    ".tsx",
    ".mts",
    ".cts",
    ".go",
    ".cs",
    ".java",
    ".kt",
    ".kts",
    ".rs",
    ".rb",
    ".php",
    ".swift",
    ".scala",
    ".c",
    ".cc",
    ".cpp",
    ".h",
    ".hpp",
}
_BUILD_NAMES = {
    "pyproject.toml",
    "setup.py",
    "setup.cfg",
    "requirements.txt",
    "package.json",
    "package-lock.json",
    "pnpm-lock.yaml",
    "yarn.lock",
    "tsconfig.json",
    "go.mod",
    "go.sum",
    "cargo.toml",
    "cargo.lock",
    "makefile",
    "cmakelists.txt",
    "dockerfile",
    "build.gradle",
    "build.gradle.kts",
    "pom.xml",
}
_DOC_SUFFIXES = {".md", ".mdx", ".rst", ".adoc", ".txt"}
_CLASSIFICATION_BY_BOUNDARY = {
    "product": "PRODUCT",
    "toolchain": "TOOLCHAIN",
    "neutral_contract": "NEUTRAL_CONTRACT",
}


@dataclass(frozen=True)
class ReferenceImpact:
    category: str
    path: str
    line: int
    text: str

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class ProfileChange:
    location: str
    before: str
    after: str

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


@dataclass(frozen=True)
class TopologyMigrationPlan:
    repository_root: str
    profile_path: str
    from_root: str
    to_root: str
    component_id: str | None
    classification: str
    ownership_mode: str
    classification_before: dict[str, str]
    classification_after: dict[str, str]
    source_files: tuple[str, ...]
    profile_changes: tuple[ProfileChange, ...]
    reference_impacts: tuple[ReferenceImpact, ...]
    projected_profile: str

    def as_dict(self) -> dict[str, object]:
        impacts: dict[str, list[dict[str, object]]] = {
            "PROFILE": [],
            "IMPORT": [],
            "BUILD": [],
            "CI": [],
            "DOCUMENTATION": [],
            "OTHER": [],
        }
        for item in self.reference_impacts:
            impacts.setdefault(item.category, []).append(item.as_dict())
        manual_count = sum(
            1 for item in self.reference_impacts if item.category != "PROFILE"
        )
        return {
            "repository_root": self.repository_root,
            "profile_path": self.profile_path,
            "migration": {
                "from": self.from_root,
                "to": self.to_root,
                "component_id": self.component_id,
                "classification": self.classification,
                "ownership_mode": self.ownership_mode,
            },
            "classification": {
                "before": self.classification_before,
                "after": self.classification_after,
                "preserved": self.classification_before == self.classification_after,
            },
            "source_files": list(self.source_files),
            "profile_changes": [item.as_dict() for item in self.profile_changes],
            "reference_impacts": impacts,
            "manual_reference_updates_required": manual_count,
            "automatic_actions": [
                "move repository root",
                "rewrite project-profile path declarations",
                "verify architecture classification is unchanged",
                "validate the projected profile after the move",
            ],
            "automatic_reference_rewrite": False,
        }


def _normalize_root(value: str) -> str:
    text = value.replace("\\", "/").strip()
    while text.startswith("./"):
        text = text[2:]
    text = text.strip("/")
    if not text or text == ".":
        raise ValueError("Topology root must be a non-empty repository-relative path")
    if any(char in text for char in "*?[]"):
        raise ValueError("Topology roots must not contain glob syntax")
    path = PurePosixPath(text)
    if path.is_absolute() or ".." in path.parts:
        raise ValueError("Topology roots must stay within the repository")
    return path.as_posix()


def _normalize_selector(value: str) -> str:
    text = value.replace("\\", "/").strip()
    while text.startswith("./"):
        text = text[2:]
    return text.strip("/")


def _inside(root: str, candidate: str) -> bool:
    return candidate == root or candidate.startswith(root + "/")


def _within_repository(root: Path, candidate: Path) -> bool:
    try:
        candidate.relative_to(root)
    except ValueError:
        return False
    return True


def _check_topology_path(root: Path, relative: str, *, must_exist: bool) -> Path:
    candidate = root / relative
    if must_exist:
        if not candidate.exists():
            raise FileNotFoundError(f"Topology source root does not exist: {candidate}")
        if candidate.is_symlink():
            raise ValueError("Topology source root must not be a symbolic link")
        if not candidate.is_dir():
            raise ValueError("Topology source root must be a directory")
        if not _within_repository(root, candidate.resolve()):
            raise ValueError("Topology source root resolves outside the repository")
        return candidate

    if candidate.exists():
        raise FileExistsError(f"Topology target root already exists: {candidate}")
    existing_parent = candidate.parent
    while not existing_parent.exists() and existing_parent != root:
        existing_parent = existing_parent.parent
    if not existing_parent.exists() or not _within_repository(root, existing_parent.resolve()):
        raise ValueError("Topology target root would resolve outside the repository")
    return candidate


def _rewrite_reference(value: str, old: str, new: str) -> str:
    normalized = value.replace("\\", "/")
    prefix = ""
    while normalized.startswith("./"):
        prefix += "./"
        normalized = normalized[2:]
    if normalized == old:
        return prefix + new
    if normalized.startswith(old + "/"):
        return prefix + new + normalized[len(old) :]
    return value


def _classification_snapshot(payload: dict[str, object]) -> tuple[str, dict[str, str]]:
    components = payload.get("components")
    if isinstance(components, list):
        result: dict[str, str] = {}
        for item in components:
            if isinstance(item, dict):
                component_id = str(item.get("id", ""))
                classification = str(item.get("classification", ""))
                if component_id:
                    result[component_id] = classification
        return "components", result
    boundaries = payload.get("boundaries")
    if isinstance(boundaries, dict):
        result = {}
        for plane, classification in _CLASSIFICATION_BY_BOUNDARY.items():
            item = boundaries.get(plane)
            if isinstance(item, dict):
                result[plane] = classification
        return "boundaries", result
    raise ValueError("Profile declares neither components nor boundaries")


def _load_profile(profile: Path) -> dict[str, object]:
    payload = yaml.safe_load(profile.read_text(encoding="utf-8-sig"))
    if not isinstance(payload, dict):
        raise ValueError("PTSIP profile root must be a mapping")
    return payload


def _rewrite_profile(
    payload: dict[str, object],
    old: str,
    new: str,
    component_id: str | None,
) -> tuple[dict[str, object], str | None, str, list[ProfileChange]]:
    projected = yaml.safe_load(yaml.safe_dump(payload, sort_keys=False, allow_unicode=True))
    assert isinstance(projected, dict)
    mode, _before = _classification_snapshot(payload)
    changes: list[ProfileChange] = []

    if mode == "components":
        components = projected.get("components")
        assert isinstance(components, list)
        if not component_id:
            raise ValueError("--component is required when the profile uses component ownership")
        selected: dict[str, object] | None = None
        selected_index = -1
        for index, item in enumerate(components):
            if isinstance(item, dict) and str(item.get("id", "")) == component_id:
                selected = item
                selected_index = index
                break
        if selected is None:
            raise ValueError(f"Component {component_id!r} is not declared in the selected profile")
        includes = selected.get("include", [])
        if not isinstance(includes, list) or not any(
            _inside(old, _normalize_selector(str(item))) for item in includes
        ):
            raise ValueError(
                f"Component {component_id!r} has no include selector rooted at {old!r}"
            )
        classification = str(selected.get("classification", ""))

        # Rewrite every component path declaration beneath the moved root. This
        # preserves nested component ownership instead of collapsing it into the
        # explicitly selected parent component.
        for index, item in enumerate(components):
            if not isinstance(item, dict):
                continue
            for field in _PATH_FIELDS:
                values = item.get(field)
                if not isinstance(values, list):
                    continue
                for value_index, raw in enumerate(values):
                    before = str(raw)
                    after = _rewrite_reference(before, old, new)
                    if after == before:
                        continue
                    values[value_index] = after
                    changes.append(
                        ProfileChange(
                            location=f"components[{index}].{field}[{value_index}]",
                            before=before,
                            after=after,
                        )
                    )
        if not any(
            change.location.startswith(f"components[{selected_index}].include[")
            for change in changes
        ):
            raise ValueError(
                f"Component {component_id!r} does not expose a rewritable root selector for {old!r}"
            )
        return projected, component_id, classification, changes

    boundaries = projected.get("boundaries")
    assert isinstance(boundaries, dict)
    owners: list[tuple[str, int]] = []
    for plane, item in boundaries.items():
        if not isinstance(item, dict):
            continue
        roots = item.get("roots")
        if not isinstance(roots, list):
            continue
        for index, raw in enumerate(roots):
            if _normalize_selector(str(raw)) == old:
                owners.append((str(plane), index))
    if len(owners) != 1:
        raise ValueError(
            f"Boundary ownership requires exactly one root equal to {old!r}; found {len(owners)}"
        )
    plane, _owner_index = owners[0]
    classification = _CLASSIFICATION_BY_BOUNDARY.get(plane)
    if classification is None:
        raise ValueError(f"Unsupported boundary plane {plane!r}")
    for current_plane, item in boundaries.items():
        if not isinstance(item, dict):
            continue
        roots = item.get("roots")
        if not isinstance(roots, list):
            continue
        for index, raw in enumerate(roots):
            before = str(raw)
            after = _rewrite_reference(before, old, new)
            if after == before:
                continue
            roots[index] = after
            changes.append(
                ProfileChange(
                    location=f"boundaries.{current_plane}.roots[{index}]",
                    before=before,
                    after=after,
                )
            )
    return projected, None, classification, changes


def _reference_category(path: str, line: str) -> str:
    normalized = path.replace("\\", "/")
    lower_path = normalized.casefold()
    name = PurePosixPath(normalized).name.casefold()
    suffix = PurePosixPath(normalized).suffix.casefold()
    if lower_path.startswith(".github/workflows/"):
        return "CI"
    if name in _BUILD_NAMES or suffix in {
        ".csproj",
        ".fsproj",
        ".vbproj",
        ".sln",
        ".props",
        ".targets",
    }:
        return "BUILD"
    if lower_path.startswith("docs/") or suffix in _DOC_SUFFIXES:
        return "DOCUMENTATION"
    if suffix in _SOURCE_SUFFIXES:
        lowered_line = line.casefold()
        if any(
            token in lowered_line
            for token in ("import ", "from ", "require(", "include", "load", "source")
        ):
            return "IMPORT"
    return "OTHER"


def _scan_references(root: Path, profile: Path, old: str) -> list[ReferenceImpact]:
    _mode, paths, errors = repository_files(root)
    if errors:
        raise RuntimeError("Repository reference scan is incomplete: " + "; ".join(errors))
    results: list[ReferenceImpact] = []
    profile_resolved = profile.resolve()
    for rel in paths:
        path = root / rel
        try:
            if path.is_symlink():
                continue
            category_override = "PROFILE" if path.resolve() == profile_resolved else None
            data = path.read_bytes()
        except OSError:
            continue
        if b"\0" in data:
            continue
        try:
            text = data.decode("utf-8-sig")
        except UnicodeDecodeError:
            continue
        for line_number, line in enumerate(text.splitlines(), 1):
            if old not in line and old.replace("/", "\\") not in line:
                continue
            results.append(
                ReferenceImpact(
                    category=category_override or _reference_category(rel, line),
                    path=rel,
                    line=line_number,
                    text=line.strip()[:240],
                )
            )
    return results


def plan_topology_migration(
    repository_root: str | Path,
    profile_path: str | Path | None,
    from_root: str,
    to_root: str,
    component_id: str | None = None,
) -> TopologyMigrationPlan:
    root = Path(repository_root).resolve()
    old = _normalize_root(from_root)
    new = _normalize_root(to_root)
    if old == new:
        raise ValueError("Topology source and target roots are identical")
    if _inside(old, new) or _inside(new, old):
        raise ValueError("Topology source and target roots must not contain one another")

    source = _check_topology_path(root, old, must_exist=True)
    _check_topology_path(root, new, must_exist=False)

    profile = find_profile(root, profile_path)
    if profile is None:
        raise FileNotFoundError("No PTSIP project profile found for topology migration")
    profile = profile.resolve()
    if not _within_repository(root, profile):
        raise ValueError("Topology migration requires the selected profile to be inside the repository")
    if profile.is_symlink():
        raise ValueError("Topology migration does not rewrite a symbolic-link profile")
    if _within_repository(source.resolve(), profile):
        raise ValueError("The selected PTSIP profile cannot be located inside the root being moved")

    validation = validate_profile(root, profile)
    if not validation.valid:
        raise ValueError("Current PTSIP profile is invalid: " + "; ".join(validation.errors))

    payload = _load_profile(profile)
    ownership_mode, before = _classification_snapshot(payload)
    projected, selected_component, classification, changes = _rewrite_profile(
        payload, old, new, component_id
    )
    after_mode, after = _classification_snapshot(projected)
    if after_mode != ownership_mode or before != after:
        raise RuntimeError("Projected topology migration changes architecture classification")
    projected_errors = validate_projected_payload(projected)
    if projected_errors:
        raise ValueError("Projected PTSIP profile is invalid: " + "; ".join(projected_errors))
    if not changes:
        raise ValueError("Topology migration would not change any profile path declaration")

    _mode, repository_paths, scan_errors = repository_files(root)
    if scan_errors:
        raise RuntimeError("Repository file scan is incomplete: " + "; ".join(scan_errors))
    source_files = tuple(
        path for path in repository_paths if _inside(old, path.replace("\\", "/"))
    )
    if not source_files:
        raise ValueError("Topology source root contains no repository files to migrate")
    references = _scan_references(root, profile, old)
    projected_text = yaml.safe_dump(projected, sort_keys=False, allow_unicode=True)
    return TopologyMigrationPlan(
        repository_root=str(root),
        profile_path=str(profile),
        from_root=old,
        to_root=new,
        component_id=selected_component,
        classification=classification,
        ownership_mode=ownership_mode,
        classification_before=before,
        classification_after=after,
        source_files=source_files,
        profile_changes=tuple(changes),
        reference_impacts=tuple(references),
        projected_profile=projected_text,
    )


def _git(root: Path, *args: str) -> None:
    proc = subprocess.run(
        ["git", "-C", str(root), *args],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        message = proc.stderr.strip() or proc.stdout.strip() or "git command failed"
        raise RuntimeError(message)


def _atomic_write(path: Path, content: str) -> None:
    with tempfile.NamedTemporaryFile(
        "w",
        suffix=path.suffix or ".yaml",
        dir=path.parent,
        delete=False,
        encoding="utf-8",
        newline="\n",
    ) as handle:
        temp = Path(handle.name)
        handle.write(content)
    try:
        temp.replace(path)
    finally:
        if temp.exists():
            temp.unlink()


def _rollback_git(
    root: Path,
    profile: Path,
    from_root: str,
    to_root: str,
    moved: bool,
) -> list[str]:
    errors: list[str] = []
    try:
        profile_rel = profile.relative_to(root).as_posix()
        _git(root, "reset", "--quiet", "HEAD", "--", profile_rel)
    except Exception as exc:
        errors.append(f"profile index reset failed: {exc}")
    if moved:
        try:
            _git(root, "mv", "--", to_root, from_root)
        except Exception as exc:
            errors.append(f"root rollback failed: {exc}")
    return errors


def apply_topology_migration(plan: TopologyMigrationPlan) -> dict[str, object]:
    root = Path(plan.repository_root)
    profile = Path(plan.profile_path)
    source = root / plan.from_root
    target = root / plan.to_root
    repo = discover_repository(root)
    if repo.is_git and repo.dirty:
        raise RuntimeError("Topology --apply requires a clean Git working tree and index")
    original_profile = profile.read_text(encoding="utf-8-sig")
    created_parent = not target.parent.exists()
    target.parent.mkdir(parents=True, exist_ok=True)
    moved = False
    try:
        if repo.is_git:
            _git(root, "mv", "--", plan.from_root, plan.to_root)
            move_method = "git-mv"
        else:
            shutil.move(str(source), str(target))
            move_method = "filesystem-move"
        moved = True

        # Full validation is intentionally deferred until the move has happened:
        # component selectors now point at the target path and repository_files()
        # can observe that target (including the Git index after git mv).
        with tempfile.NamedTemporaryFile(
            "w",
            suffix=".yaml",
            dir=profile.parent,
            delete=False,
            encoding="utf-8",
            newline="\n",
        ) as handle:
            projected_temp = Path(handle.name)
            handle.write(plan.projected_profile)
        try:
            validation = validate_profile(root, projected_temp)
        finally:
            if projected_temp.exists():
                projected_temp.unlink()
        if not validation.valid:
            raise ValueError(
                "Post-move projected PTSIP profile is invalid: " + "; ".join(validation.errors)
            )

        _atomic_write(profile, plan.projected_profile)
        final_payload = _load_profile(profile)
        final_mode, final_classification = _classification_snapshot(final_payload)
        if final_mode != plan.ownership_mode or final_classification != plan.classification_before:
            raise RuntimeError("Applied topology migration changed architecture classification")
        final_validation = validate_profile(root, profile)
        if not final_validation.valid:
            raise ValueError(
                "Applied PTSIP profile is invalid: " + "; ".join(final_validation.errors)
            )
        if repo.is_git:
            profile_rel = profile.relative_to(root).as_posix()
            _git(root, "add", "--", profile_rel)

        payload = plan.as_dict()
        payload.update(
            {
                "status": (
                    "APPLIED_WITH_REFERENCE_REVIEW"
                    if payload["manual_reference_updates_required"]
                    else "APPLIED"
                ),
                "applied": True,
                "move_method": move_method,
                "git_index_updated": bool(repo.is_git),
                "profile_validation": final_validation.as_dict(),
            }
        )
        return payload
    except Exception as exc:
        rollback_errors: list[str] = []
        try:
            if profile.exists() and profile.read_text(encoding="utf-8-sig") != original_profile:
                _atomic_write(profile, original_profile)
        except Exception as rollback_exc:
            rollback_errors.append(f"profile content restore failed: {rollback_exc}")

        if repo.is_git:
            rollback_errors.extend(
                _rollback_git(root, profile, plan.from_root, plan.to_root, moved)
            )
        elif moved:
            try:
                if target.exists():
                    shutil.move(str(target), str(source))
            except Exception as rollback_exc:
                rollback_errors.append(f"root rollback failed: {rollback_exc}")

        if created_parent:
            try:
                os.removedirs(target.parent)
            except OSError:
                pass
        if rollback_errors:
            raise RuntimeError(
                "Topology migration failed and rollback was incomplete: "
                + "; ".join(rollback_errors)
            ) from exc
        raise


def migrate_topology(
    repository_root: str | Path,
    profile_path: str | Path | None,
    from_root: str,
    to_root: str,
    component_id: str | None = None,
    *,
    apply: bool = False,
) -> dict[str, object]:
    plan = plan_topology_migration(
        repository_root,
        profile_path,
        from_root,
        to_root,
        component_id,
    )
    if not apply:
        payload = plan.as_dict()
        payload.update({"status": "PLAN", "applied": False})
        return payload
    return apply_topology_migration(plan)
