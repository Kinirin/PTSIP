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
| `0.3.2` | Historical migration source | [`0.3.2.md`](0.3.2.md) |
| `0.3.3` | Permanent source-only implementation version | [`0.3.3.md`](0.3.3.md) |
| `0.3.4` | Published historical Tool release; stale pre-publication repository note removed | GitHub Release history |
| `0.3.5` | **Published; first VPMS-capable Tool release** | [`0.3.5.md`](0.3.5.md) |
| `0.3.6` | **Release candidate; WU-07 release preparation in progress** | [`0.3.6.md`](0.3.6.md) |

Versions that never represented a real PTSIP Tool source/release state are not fabricated merely to make the sequence contiguous.

### Tool 0.3.5 publication note

`tool-v0.3.5` was published from commit `79bc4c2daf695e8462a02f2a7c4b1bb1a88846e1` on 2026-08-17. The `tooling-release` workflow run `32011194245` completed both build and PyPI Trusted Publishing successfully.

The tagged `releasenote/0.3.5.md` was finalized immediately before publication and its status header therefore records the draft/publication boundary as it existed at that immutable release commit. The version document is not rewritten after publication merely to change an operational status line. Current publication state is recorded in this index and `STATUS.md`.

### Tool 0.3.6 pre-publication note

Tool `0.3.6` remains unpublished while WU-07 performs final Specification freeze and release-contract strengthening. Its current release note is a reviewed pre-publication source document and is not a published release record until the exact-main release gate completes and `tool-v0.3.6` is actually published.

Current development binding:

```text
Specification 0.3.6-draft
SPEC_REVISION d6995ed232e845b88d8235b851e80ab54b7804ea
```

## Specification history

| Family | State | Document |
| --- | --- | --- |
| `0.1.0-draft` | Historical initial public draft | [`spec-0.1.0-draft.md`](spec-0.1.0-draft.md) |
| `0.2.0-draft` | Historical experimental draft family | [`spec-0.2.0-draft.md`](spec-0.2.0-draft.md) |
| `0.3.4-draft` | Published Tool `0.3.5` baseline family | [`spec-0.3.4-draft.md`](spec-0.3.4-draft.md) |
| `0.3.6-draft` | **Current active Tool `0.3.6` development family** | [`spec-0.3.6-draft.md`](spec-0.3.6-draft.md) |

Draft family labels are mutable; their exact normative identity is the immutable Git revision recorded by the relevant Tool, release, or evaluation.

## Tool release policy

- use `.github/workflows/release.yml` only for explicit release preparation, never on ordinary pushes;
- require the requested version to match `pyproject.toml`;
- run the Tool test/CLI smoke gate before creating a release draft;
- allow a source/unreleased version document to exist before release so important migration history is not hidden in a monolithic changelog;
- `git-cliff` may generate the first release-note draft from final Git history, but generated commit lists are a starting point rather than a substitute for explaining architectural or user-visible changes;
- review and, when necessary, rewrite the generated draft before publication so newly introduced subsystems, compatibility boundaries, migration meaning, and verification evidence are described explicitly;
- commit the final `releasenote/<version>.md` to `main` before the release tag is published;
- use the same reviewed Markdown file as the GitHub Release body at the publication boundary;
- treat the tagged version document as immutable once its `tool-v<version>` release is published;
- record post-publication operational status corrections in `releasenote/README.md` and `STATUS.md` rather than rewriting the tagged version document;
- publish the prepared GitHub draft manually so the existing `tooling-release.yml` Trusted Publishing workflow can publish to PyPI.

## Completed planning retention

After a Tool version is published and its durable release note exists, completed version-specific planning files may be removed from the active `planning/` directory. Their implementation history remains recoverable from Git history and the corresponding immutable release point. Active future planning remains in `planning/`.

## Specification note policy

Specification notes are curated directly because Specification commits are deliberately excluded from Tool `git-cliff` output. Update the applicable `spec-<family>.md` document whenever the normative draft family evolves, and bind exact normative claims to immutable Git revisions rather than relying on the family label alone.
