# Bounded PTSIP 0.3.8b2 release

The Project Owner authorized this release on 2026-10-15. Work is isolated on
`dev/0.3.8b2`; this branch must not merge to `main`. The current policy retirement
and Family reconstruction records are retained without restoration or amendment.
No new Family policy, policy schema or policy validator is introduced.

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
