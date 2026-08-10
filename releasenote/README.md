# PTSIP Reference Tool Release Notes

This directory is the canonical repository history for versioned **Reference Tool** release notes.

Policy:

- generate new Tool release notes from Git history with `git-cliff` and `cliff.toml`;
- use `.github/workflows/release.yml` only when a Tool release is intentionally being prepared;
- require the requested version to match `pyproject.toml`;
- run the Tool test/CLI smoke gate before creating a release draft;
- commit `releasenote/<version>.md` to `main` before the release tag is created;
- use the same Markdown file as the GitHub Release body;
- treat a note as immutable once its `tool-v<version>` tag exists;
- publish the prepared GitHub draft manually so the existing `tooling-release.yml` Trusted Publishing workflow can publish to PyPI.

`TOOLING-CHANGELOG.md` is legacy pre-migration history and should receive no new release entries. `CHANGELOG.md` remains the independently versioned **PTSIP Specification** change history and is not the PyPI Tool release log.

Only actually published historical Tool releases are backfilled here. Source-only or unreleased Tool versions are intentionally not represented as releases.
