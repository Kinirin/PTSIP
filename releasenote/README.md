# PTSIP Release Notes

This directory is the canonical repository history for three independently versioned authorities:

```text
PTSIP Reference Tool
Project Profile Contract
PTSIP Specification family / immutable revision
```

Root-level `CHANGELOG.md` and `TOOLING-CHANGELOG.md` are no longer maintained. Version history is written here so each authority has an explicit record instead of accumulating unrelated histories into large changelog files.

## Namespace policy

ADR-0021 adopts a forward-looking namespace split while preserving historical release-note paths.

New-generation records use:

```text
releasenote/tool/<tool-version>.md
releasenote/project-profile/<pp-contract-version>.md
releasenote/specification/<spec-family>.md
```

Examples:

```text
releasenote/tool/0.3.7.md
releasenote/project-profile/pp.1.01.md
releasenote/specification/0.3.7-draft.md
```

Published or historical flat files already stored directly under `releasenote/` remain in place. They are not bulk-moved merely to make the directory visually uniform or to rewrite old identity conventions.

Namespace indexes:

- [`tool/README.md`](tool/README.md)
- [`project-profile/README.md`](project-profile/README.md)
- [`specification/README.md`](specification/README.md)

## Project Profile contract history

| Contract | State | Document |
| --- | --- | --- |
| `pp.1.01` | **WU-09 identity implementation / pre-adoption** | [`project-profile/pp.1.01.md`](project-profile/pp.1.01.md) |

### `0.3.6-draft -> pp.1.01` compatibility notice

ADR-0021 classifies the transition from historical Project Profile identity `0.3.6-draft` to `pp.1.01` as:

```text
IDENTITY_ONLY
```

For this bridge there is no Project Profile contract-content delta in `components`, `relationships`, `associated_artifacts`, `policies`, Responsibility Map semantics, or lifecycle classifications.

An otherwise valid `0.3.6-draft` Project Profile does not require architecture redesign or lifecycle reclassification merely because the canonical Project Profile contract identity becomes `pp.1.01`. Post-rewrite identity/schema validation is still required. See [`project-profile/pp.1.01.md`](project-profile/pp.1.01.md) for the user-facing migration-continuity notice.

Historical `0.3.6-draft` records remain historical facts; they are not rewritten as though they were originally published under the `pp.*` namespace.

## Historical Reference Tool history

The following Tool records retain their original flat paths.

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
| `0.3.6` | **Development complete; pre-publication release candidate** | [`0.3.6.md`](0.3.6.md) |

Versions that never represented a real PTSIP Tool source/release state are not fabricated merely to make the sequence contiguous.

### Tool 0.3.5 publication note

`tool-v0.3.5` was published from commit `79bc4c2daf695e8462a02f2a7c4b1bb1a88846e1` on 2026-08-17. The `tooling-release` workflow run `32011194245` completed both build and PyPI Trusted Publishing successfully.

The tagged `releasenote/0.3.5.md` was finalized immediately before publication and its status header therefore records the draft/publication boundary as it existed at that immutable release commit. The version document is not rewritten after publication merely to change an operational status line. Current publication state is recorded in this index and `STATUS.md`.

### Tool 0.3.6 pre-publication note

Tool `0.3.6` development work is complete. WU-00 through WU-07 are closed, and WU-07 completed **Strategy B — Release Contract Strengthening**. The reviewed Tool release note remains a pre-publication source document until the exact-main release and publication boundary succeeds.

Current binding:

```text
Specification 0.3.6-draft
SPEC_REVISION d6995ed232e845b88d8235b851e80ab54b7804ea
```

Final development-branch exact verification authority:

```text
source SHA:       452d0f8b0c78bdebb180ceb2b9994485f59eb43a
workflow run/job: 32640319047 / 97196299107
Python:           3.14.6
pytest:           331 passed / 0 failed
profile coverage: unassigned_count=0
Product Artifact: PASS / exact snapshot binding
PTSIP-PKG-001:    0 definite violations
status:           self-hosted/tooling-test = success
```

Documentation descendants after that run record closure and do not replace the exact verification authority.

## Historical Specification history

The following Specification records retain their original flat `spec-<family>.md` paths.

| Family | State | Document |
| --- | --- | --- |
| `0.1.0-draft` | Historical initial public draft | [`spec-0.1.0-draft.md`](spec-0.1.0-draft.md) |
| `0.2.0-draft` | Historical experimental draft family | [`spec-0.2.0-draft.md`](spec-0.2.0-draft.md) |
| `0.3.4-draft` | Published Tool `0.3.5` baseline family | [`spec-0.3.4-draft.md`](spec-0.3.4-draft.md) |
| `0.3.6-draft` | Tool `0.3.6` bound release-candidate family | [`spec-0.3.6-draft.md`](spec-0.3.6-draft.md) |
| `0.3.7-draft` | **Active Tool `0.3.7` development family; immutable activation snapshot `b648d9e026f502b14481ba2d0606d9acc88a31fc`** | [`spec-0.3.7-draft.md`](spec-0.3.7-draft.md) |

The existing `spec-0.3.7-draft.md` remains at its historical flat path. Future Specification-note records use the `specification/` namespace; ADR-0021 does not require retroactive relocation.

Draft family labels are not the immutable normative identity by themselves; exact normative claims are bound to the immutable Git revision recorded by the relevant Tool, release, or evaluation.

## Tool release policy

- use `.github/workflows/release.yml` only for explicit release preparation, never on ordinary pushes;
- require the requested version to match `pyproject.toml`;
- run the Tool test/CLI smoke gate before creating a release draft;
- use `releasenote/tool/<version>.md` for new-generation Tool release records;
- allow a source/unreleased version document to exist before release so important migration history is not hidden in a monolithic changelog;
- `git-cliff` may generate the first release-note draft from final Git history, but generated commit lists are a starting point rather than a substitute for explaining architectural or user-visible changes;
- review and, when necessary, rewrite the generated draft before publication so newly introduced subsystems, compatibility boundaries, migration meaning, and verification evidence are described explicitly;
- commit the final new-generation Tool release note to `main` before the release tag is published;
- use the same reviewed Markdown file as the GitHub Release body at the publication boundary;
- treat the tagged version document as immutable once its `tool-v<version>` release is published;
- record post-publication operational status corrections in `releasenote/README.md` and `STATUS.md` rather than rewriting the tagged version document;
- publish the prepared GitHub draft manually so the existing `tooling-release.yml` Trusted Publishing workflow can publish to PyPI.

## Project Profile note policy

Project Profile contract notes use `releasenote/project-profile/<pp-version>.md` for new-generation records.

A PP note must distinguish:

```text
contract semantic change
identity-only transition
one project's instance revision
```

and must not imply that a Tool package version bump authorizes PP migration. Identity-only bridges must state explicitly whether users need semantic architecture review.

## Completed planning retention

After a Tool version is published and its durable release note exists, completed version-specific planning files may be removed from the active `planning/` directory. Their implementation history remains recoverable from Git history and the corresponding immutable release point. Active future planning remains in `planning/`.

Tool `0.3.6.1` planning is cancelled. Active migration development planning is now Tool `0.3.7` under `planning/0.3.7.md` and `planning/0.3.7/*`.

## Specification note policy

New-generation Specification notes use `releasenote/specification/<family>.md`. Historical flat `spec-<family>.md` records remain where they are.

Specification notes are curated directly because Specification commits are deliberately excluded from Tool `git-cliff` output. Update the applicable Specification note whenever the normative draft family evolves, and bind exact normative claims to immutable Git revisions rather than relying on the family label alone.
