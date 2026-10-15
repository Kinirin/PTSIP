# Temporary PTSIP 0.3.8b3 container delivery

The owner authorized this temporary delivery on 2026-10-15 on `dev/0.3.8b2`.
It must not merge to main. The image contains the already published complete
Package: Tool T.0.3.8.b3, Profile pp.1.02 and seven independent native engines.
Context and Projection are 0.2.1; Storage, Validation, Discovery, Planning and
Execution are 0.3.1. Profile history is excluded.

The application source is `916333f9aa0f9589272911818d8352fb0cae09e1`, tested by
tooling-test run 38677942975. The original shipping manifest SHA256 and all eight
input wheel hashes are checked. The container recipe has its own source SHA and
verification run. The published application artifacts are not rebuilt or changed.

This recipe is a bounded projection of one existing published composition. It
does not define a canonical current-version manifest or implement the pending
successor INFO policy. The language-neutral reconciliation recorded at 740d1a65
is preserved. Existing pyproject metadata is consumed only inside the already
built wheel, not as the authority for this container delivery. Retired Developer
policies and automated validators remain retired.

The registered `release-container` mode in `.github/test_modes.yaml` is executed
only by `tooling-test.yml`. It builds a non-root Linux amd64 image on Ubuntu
24.04 (glibc 2.39), checks the Tool, Profile catalog and all seven installed native
binary hashes, runs pip check, and saves that exact tested image. The compressed
image and receipt are immutable Actions artifacts.

`tooling-release.yml` with destination `ghcr` discovers the successful exact-recipe
verification automatically, authenticates with repository-scoped GITHUB_TOKEN,
loads and hashes the admitted image, and pushes it without rebuilding or testing.
Existing version tags that refer to another image are rejected. Publication
evidence is added to the original GitHub Release under a new recipe-specific
filename. Both PyPI upload jobs are explicitly on hold.

Registry target: `ghcr.io/kinirin/ptsip:0.3.8b3`. Aliases `0.3.8.b3` and
`0.3.8.f87bfe` identify the same container. Windows requires a Linux-container
runtime; this is not a Windows-native container or an ARM64 image. The image
includes Python 3.12 and dependencies, so users do not need Python on the host.
Building the image downloads third-party dependencies; it does not publish PTSIP
or engines to PyPI. Installation does not grant engine execution authority.

After publication, verify the exact digest from the publication receipt, then use:

```sh
docker run --rm ghcr.io/kinirin/ptsip:0.3.8b3 --version
docker run --rm ghcr.io/kinirin/ptsip:0.3.8b3 --help
```

For repository operations, mount the desired repository at `/workspace` and use
the normal PTSIP command arguments. Linux bind mounts may require `--user` to
match the host repository owner. GHCR first-publication visibility is checked
separately; public pull access is not implied by a successful authenticated push.
