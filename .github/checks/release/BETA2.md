# Bounded PTSIP 0.3.8.b4 hotfix release

The Project Owner requested this emergency release on 2026-10-18. The isolated
`codex/validation-b4` branch starts from published b3 commit b4588071. It must not
merge to main. The legacy filename and controller function names remain routing
compatibility details, not policy or version authority.

`beta2-authorization.json` records the current bounded instruction. The retired
Developer Policy Resolver stays retired; software checks are not automated policy
PASS. The historical Profile schema loading dependency from the b3 branch remains
limited to existing Profile verification. No successor Family validator is created.

`PTSIP==0.3.8b4` contains Tool T.0.3.8.b4, Profile pp.1.02 and all seven native
engines in each platform wheel. Validation is 0.3.2 with capability contract 3.1.0;
Context/Projection remain 0.2.1 and the other engines remain 0.3.1. Schema-reference
support and limits are documented in docs/validation-schema-references.md.

`tooling-test.yml` verifies all nine components on Windows and Linux, then installs
the complete platform wheels. The Linux join also rebuilds and installs the complete
sdist. `tooling-release.yml` requires the same exact-source successful run, publishes
one source-SHA GitHub Release, and uploads only the two complete PTSIP wheels and
one PTSIP sdist to PyPI. Intermediate engine/Tool artifacts are retained as GitHub
build evidence. There is no independent-engine PyPI upload job.

The publisher uses the existing pypi Environment token secret ENTIRE_ACCOUNT;
secret values are never read or printed. Immutable public PyPI hashes are checked
before record commits. Record commits target codex/validation-b4 explicitly.
Historic release assets and already published versions remain immutable.

The original b3 single-project migration exception is historical and applies only
to its exact approved source/version. It does not authorize replacing a published
Tool or engine at the same version in b4. The frozen b3 GHCR path is unchanged and
is not a b4 publication destination.
