# PTSIP Release Notes

This directory is the canonical repository history for both the independently versioned **PTSIP Reference Tool** and **PTSIP Specification**.

Root-level `CHANGELOG.md` and `TOOLING-CHANGELOG.md` are no longer maintained. Version history is written directly here so each version or Specification family has an explicit document instead of accumulating unrelated histories into large changelog files.

## Naming

### Reference Tool

Tool records use the package version directly:

```text
releasenote/<version>.md
```

Examples: `0.2.0.md`, `0.3.1.md`.

### Specification

Specification records are prefixed because Specification and Tool versions are independent:

```text
releasenote/spec-<family>.md
```

Examples: `spec-0.1.0-draft.md`, `spec-0.2.0-draft.md`.

## Reference Tool history

| Version | State | Document |
| --- | --- | --- |
| `0.1.0a1` | Published prerelease | [`0.1.0a1.md`](0.1.0a1.md) |
| `0.2.0` | Published | [`0.2.0.md`](0.2.0.md) |
| `0.2.2` | Historical source version, not published | [`0.2.2.md`](0.2.2.md) |
| `0.2.3` | Source-only migration, intentionally not published | [`0.2.3.md`](0.2.3.md) |
| `0.3.0` | Published | [`0.3.0.md`](0.3.0.md) |
| `0.3.1` | Published | [`0.3.1.md`](0.3.1.md) |
| `0.3.2` | Migration source, not yet merged/published/tagged | [`0.3.2.md`](0.3.2.md) |
| `0.3.3` | Permanent source-only implementation version | [`0.3.3.md`](0.3.3.md) |
| `0.3.4` | Published GitHub Tool release; stale pre-publication repository note removed | GitHub Release history |
| `0.3.5` | Release draft prepared; publication pending | [`0.3.5.md`](0.3.5.md) |

Versions that never represented a real PTSIP Tool source/release state are not fabricated merely to make the sequence contiguous.

## Specification history

| Family | State | Document |
| --- | --- | --- |
| `0.1.0-draft` | Historical initial public draft | [`spec-0.1.0-draft.md`](spec-0.1.0-draft.md) |
| `0.2.0-draft` | Historical experimental draft family | [`spec-0.2.0-draft.md`](spec-0.2.0-draft.md) |
| `0.3.4-draft` | Current active draft family | [`spec-0.3.4-draft.md`](spec-0.3.4-draft.md) |

Draft family labels are mutable; their exact normative identity is the immutable Git revision recorded by the relevant Tool, release, or evaluation.

## Tool release policy

- use `.github/workflows/release.yml` only for explicit release preparation, never on ordinary pushes;
- require the requested version to match `pyproject.toml`;
- run the Tool test/CLI smoke gate before creating a release draft;
- allow a source/unreleased version document to exist before release so important migration history is not hidden in a monolithic changelog;
- `git-cliff` may generate the first release-note draft from final Git history, but generated commit lists are a starting point rather than a substitute for explaining architectural or user-visible changes;
- review and, when necessary, rewrite the generated draft before publication so newly introduced subsystems, compatibility boundaries, migration meaning, and verification evidence are described explicitly;
- commit the final `releasenote/<version>.md` to `main` before the release tag is published;
- use the same reviewed Markdown file as the GitHub Release body;
- treat the version document as immutable once its `tool-v<version>` tag is published;
- publish the prepared GitHub draft manually so the existing `tooling-release.yml` Trusted Publishing workflow can publish to PyPI.

## Specification note policy

Specification notes are curated directly because Specification commits are deliberately excluded from Tool `git-cliff` output. Update the applicable `spec-<family>.md` document whenever the normative draft family evolves, and bind exact normative claims to immutable Git revisions rather than relying on the family label alone.
