from __future__ import annotations

import json
from dataclasses import dataclass
from importlib.resources import files
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

from ..constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION


@dataclass(frozen=True)
class ValidationResult:
    profile_path: str | None
    valid: bool
    errors: list[str]
    warnings: list[str]

    def as_dict(self) -> dict[str, object]:
        return {
            "profile_path": self.profile_path,
            "valid": self.valid,
            "errors": self.errors,
            "warnings": self.warnings,
        }


def _schema() -> dict[str, object]:
    schema_path = files("ptsip").joinpath("specdata/ptsip-profile.schema.json")
    return json.loads(schema_path.read_text(encoding="utf-8"))


def find_profile(repository_root: str | Path, explicit: str | Path | None = None) -> Path | None:
    if explicit:
        candidate = Path(explicit).expanduser().resolve()
        return candidate if candidate.is_file() else None
    candidate = Path(repository_root).resolve() / "ptsip.yaml"
    return candidate if candidate.is_file() else None


def validate_profile(repository_root: str | Path, explicit: str | Path | None = None) -> ValidationResult:
    profile = find_profile(repository_root, explicit)
    if profile is None:
        return ValidationResult(
            profile_path=None,
            valid=False,
            errors=["No PTSIP project profile found."],
            warnings=["Read-only inspection and pilot commands do not require a profile."],
        )
    try:
        payload = yaml.safe_load(profile.read_text(encoding="utf-8"))
    except Exception as exc:
        return ValidationResult(str(profile), False, [f"Unable to parse profile: {exc}"], [])

    validator = Draft202012Validator(_schema())
    errors = [
        f"{'.'.join(str(p) for p in err.absolute_path) or '<root>'}: {err.message}"
        for err in sorted(validator.iter_errors(payload), key=lambda e: list(e.absolute_path))
    ]
    warnings: list[str] = []
    if not errors:
        binding = payload["ptsip"]["specification"]
        if binding["source"] != SPEC_SOURCE or payload["ptsip"]["version"] != SPEC_VERSION:
            errors.append("Profile specification binding is not supported by this tooling build.")
        revision = binding.get("revision")
        if not revision:
            warnings.append("Specification binding has no immutable revision; reproducibility is weaker.")
        elif revision != SPEC_REVISION:
            warnings.append(
                f"Profile revision {revision!r} differs from tooling snapshot {SPEC_REVISION!r}."
            )
    return ValidationResult(str(profile), not errors, errors, warnings)
