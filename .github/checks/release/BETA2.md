# Bounded PTSIP 0.3.8b2 release

The Project Owner authorized this release on 2026-10-15. Work is isolated on
`dev/0.3.8b2`; this branch must not merge to `main`. Current policy retirement
and Family reconstruction records are retained. The owner subsequently authorized
restoring the historical common schema loading dependency for existing Profile
verification on this branch only. It is copied from 781a0a46764734a98707662af1a3a2fbd6a88cf7;
retired policy statuses and the disabled Developer Policy Resolver are preserved.
Existing read-only Profile registry and PP commit/range/release checks use a
branch-scoped adapter instead of resolving a retired command owner. Profile
reconciliation, policy commands and other mutations are excluded from that adapter.
No successor Family policy, schema or validator is introduced.

`beta2-authorization.json` records the bounded release instruction, not a
successor policy or an automated policy-validation PASS. Release code does not
invoke the retired Developer Policy Resolver. Software verification still binds
the exact source SHA, complete selection, two required hosts and file digests.

`PTSIP==0.3.8b2` is the complete pip Package. It contains the Tool code and public
Profile catalog, and requires all seven independent engine wheels with exact
versions. Context/Projection are 0.2.0; Storage/Validation/Discovery/Planning/
Execution are 0.3.0. Native executables install into the environment's scripts
directory. Profile history is excluded. Installation grants no runtime authority.

The Tool display identity is `T.0.3.8.b2`, with numeric distribution projection
`0.3.8b2`. The Package's opaque composition identity remains `0.3.8.<SHA6>`;
the owner explicitly selected `0.3.8b2` for this PyPI publication. Independent
future version design is not decided by this bounded release.

`tooling-test.yml` runs the registered mode, 9 components x 2 hosts and two
complete Package pip-install joins. `tooling-release.yml` consumes the successful
exact-SHA run, ships the admitted original files and checks public PyPI hashes.
Publication never runs tests. The owner confirmed the new `pypi` Environment
Secret named `ENTIRE_ACCOUNT`; release use is restricted to `PTSIP` and the seven
registered engine projects. The old `PYPI_API_TOKEN` is not used for this release.
No secret value is read, stored or printed by the controller.

Record commits target `dev/0.3.8b2` explicitly. Source branch guards reject a
changed branch head before publication. No workflow command pushes to `main`.

PyPI rejects ZIP streaming data descriptors. Shipping normalizes only ZIP local
headers while preserving every member byte, engine binary, RECORD and original
source. The new wheel hash and original wheel hash are linked in the transport
receipt; historical GitHub files are not overwritten. Complete Package upload
waits for every engine project upload so an incomplete PyPI dependency set is
never advertised as an installable Package. New-project HTTP 429 remains an
external service limit and must not be bypassed or treated as publication PASS.
