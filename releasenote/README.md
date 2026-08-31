# PTSIP Release Notes

This directory is the canonical repository history for three independently versioned authorities:

```text
PTSIP Reference Tool
Project Profile Contract
PTSIP Specification family / immutable revision
```

Root-level `CHANGELOG.md` and `TOOLING-CHANGELOG.md` are no longer maintained. Version history is separated by authority so Tool, Project Profile, and Specification identities are not collapsed into one sequence.

## Namespace policy

The release-note directory is normalized by authority:

```text
releasenote/tool/<tool-version>.md
releasenote/project-profile/<pp-contract-version>.md
releasenote/specification/<spec-note>.md
```

Historical flat Tool and Specification release-note files have been relocated into these authority namespaces. This is a repository-organization change only; it does not rewrite the historical Tool/Specification identity or publication state recorded inside each note.

The path-retention convention originally recorded with ADR-0021 is therefore superseded for repository layout. ADR-0021's substantive identity separation remains: Tool version, Project Profile contract identity, and Specification family/revision remain independent authorities.

Namespace indexes:

- [`tool/README.md`](tool/README.md)
- [`project-profile/README.md`](project-profile/README.md)
- [`specification/README.md`](specification/README.md)

## Project Profile contract history

| Contract | State | Document |
| --- | --- | --- |
| `pp.1.01` | Current Tool 0.3.7 contract / PTSIP repository adopted | [`project-profile/pp.1.01.md`](project-profile/pp.1.01.md) |

### `0.3.6-draft -> pp.1.01` compatibility notice

ADR-0021 classifies the transition from historical Project Profile identity `0.3.6-draft` to `pp.1.01` as:

```text
IDENTITY_ONLY
```

For this bridge there is no Project Profile architecture-semantics delta in `components`, `relationships`, `associated_artifacts`, `policies`, Responsibility Map semantics, or lifecycle classifications. Current serialization materializes the explicit historical Specification family while preserving source/revision.

An otherwise valid `0.3.6-draft` Project Profile does not require architecture redesign or lifecycle reclassification merely because the canonical Project Profile contract identity becomes `pp.1.01`. Post-rewrite identity/schema validation is still required.

Historical `0.3.6-draft` facts remain historical facts; they are not rewritten as though they were originally published under the `pp.*` namespace.

## Current new-generation records

| Authority | Identity | State | Document |
| --- | --- | --- | --- |
| Tool | `0.3.7` | Release candidate; exact-SHA verification and publication pending | [`tool/0.3.7.md`](tool/0.3.7.md) |
| Project Profile | `pp.1.01` | Current contract / repository adopted | [`project-profile/pp.1.01.md`](project-profile/pp.1.01.md) |
| Specification | `0.3.7-draft @ 3c47816770d194ae42f98faedc911d980db0e62a` | WU-12 final normative freeze | [`specification/0.3.7-draft.md`](specification/0.3.7-draft.md) |

## Historical Reference Tool history

All retained Tool notes now live under `releasenote/tool/`.

| Version | State | Document |
| --- | --- | --- |
| `0.1.0a1` | Published prerelease | [`tool/0.1.0a1.md`](tool/0.1.0a1.md) |
| `0.2.0` | Published | [`tool/0.2.0.md`](tool/0.2.0.md) |
| `0.2.2` | Historical source version, not published | [`tool/0.2.2.md`](tool/0.2.2.md) |
| `0.2.3` | Source-only migration, intentionally not published | [`tool/0.2.3.md`](tool/0.2.3.md) |
| `0.3.0` | Published | [`tool/0.3.0.md`](tool/0.3.0.md) |
| `0.3.1` | Published | [`tool/0.3.1.md`](tool/0.3.1.md) |
| `0.3.2` | Historical migration source | [`tool/0.3.2.md`](tool/0.3.2.md) |
| `0.3.3` | Permanent source-only implementation version | [`tool/0.3.3.md`](tool/0.3.3.md) |
| `0.3.4` | Published historical Tool release; stale pre-publication repository note removed | GitHub Release history |
| `0.3.5` | Published; first VPMS-capable Tool release | [`tool/0.3.5.md`](tool/0.3.5.md) |
| `0.3.6` | Development complete; pre-publication release candidate | [`tool/0.3.6.md`](tool/0.3.6.md) |
| `0.3.7` | Current release candidate | [`tool/0.3.7.md`](tool/0.3.7.md) |

Versions that never represented a real PTSIP Tool source/release state are not fabricated merely to make the sequence contiguous.

### Tool 0.3.5 publication note

`tool-v0.3.5` was published from commit `79bc4c2daf695e8462a02f2a7c4b1bb1a88846e1` on 2026-08-17. The `tooling-release` workflow run `32011194245` completed both build and PyPI Trusted Publishing successfully.

The tagged Tool note was finalized immediately before publication. Current operational status corrections belong in this index and `STATUS.md` rather than rewriting an immutable tagged release point.

### Tool 0.3.6 pre-publication note

Tool `0.3.6` development work is complete. WU-00 through WU-07 are closed, and WU-07 completed Strategy B — Release Contract Strengthening. The reviewed Tool release note remains a pre-publication source document until the exact-main release/publication boundary succeeds.

Current historical binding:

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

## Historical Specification history

Historical flat Specification note basenames are preserved under the `specification/` namespace with their original `spec-` prefix. This avoids confusing those historical notes with the current canonical new-generation `specification/0.3.7-draft.md` final-freeze record.

| Family | State | Document |
| --- | --- | --- |
| `0.1.0-draft` | Historical initial public draft | [`specification/spec-0.1.0-draft.md`](specification/spec-0.1.0-draft.md) |
| `0.2.0-draft` | Historical experimental draft family | [`specification/spec-0.2.0-draft.md`](specification/spec-0.2.0-draft.md) |
| `0.3.3-draft` | Historical draft record | [`specification/spec-0.3.3-draft.md`](specification/spec-0.3.3-draft.md) |
| `0.3.4-draft` | Published Tool 0.3.5 baseline family | [`specification/spec-0.3.4-draft.md`](specification/spec-0.3.4-draft.md) |
| `0.3.6-draft` | Tool 0.3.6 bound release-candidate family | [`specification/spec-0.3.6-draft.md`](specification/spec-0.3.6-draft.md) |
| `0.3.7-draft` | Historical initial activation snapshot `b648d9e026f502b14481ba2d0606d9acc88a31fc` | [`specification/spec-0.3.7-draft.md`](specification/spec-0.3.7-draft.md) |
| `0.3.7-draft` | Final WU-12 freeze at `3c47816770d194ae42f98faedc911d980db0e62a` | [`specification/0.3.7-draft.md`](specification/0.3.7-draft.md) |

Draft family labels are not immutable normative identity by themselves. Exact normative claims remain bound to the immutable Git revision recorded by the relevant Tool, release, evaluation, or Specification note.

## Tool release policy

- use `.github/workflows/release.yml` only for explicit release preparation, never on ordinary pushes;
- require the requested version to match `pyproject.toml`;
- run the Tool test/CLI smoke gate before creating a release draft;
- store Tool release records under `releasenote/tool/<version>.md`;
- allow a source/unreleased version document to exist before release so important migration history is not hidden in a monolithic changelog;
- use `git-cliff` only as a starting point when useful; generated commit lists do not replace architectural/user-visible explanation;
- review the release note before publication;
- commit the final Tool release note to `main` before the release tag is published;
- use the same reviewed Markdown as the GitHub Release body at publication;
- treat a tagged version document as immutable after publication;
- record post-publication operational corrections in `releasenote/README.md` and `STATUS.md`;
- publish the prepared GitHub draft manually so `tooling-release.yml` can perform PyPI Trusted Publishing.

## Project Profile note policy

Project Profile contract notes use `releasenote/project-profile/<pp-version>.md`.

A PP note must distinguish:

```text
contract semantic change
identity-only transition
one project's instance revision
```

A Tool package version bump does not authorize PP migration by itself.

## Specification note policy

Specification notes live under `releasenote/specification/`.

Historical relocated records preserve their `spec-<family>.md` basename. Current/new-generation family records use the canonical namespace naming selected for that family and bind normative claims to immutable revisions.

Specification notes are curated directly because Specification commits are deliberately excluded from Tool `git-cliff` output.

## Completed planning retention

Completed version-specific active planning may be removed once its durable implementation/release history is preserved through Git history, ADRs, release notes, and immutable verification/release points.

The active planning namespaces are now:

```text
planning/0.4.0/
    = CORE / RELEASE-BLOCKING

planning/0.4.0-op/
    = OPTIONAL / NON-BLOCKING
```

Completed `planning/0.3.6*` and `planning/0.3.7*` active planning documents have been removed from the working tree. Their historical contents remain recoverable from Git history and their durable release/decision records remain in the repository.
