# Installed official engine diagnostics

`ptsip engines compatibility --json` delegates to the Rust Discovery operation
`discovery.check-environment`. It diagnoses the seven official engines in the
same selected Python installation. It does not search PATH, probe external
engines, install software, invoke providers, or activate engines.

The report separates native artifact integrity, capability contract agreement,
stdio transport agreement, and source-bound declared dependency evidence.
It includes the current operation catalog, required inputs, schemas and request
templates so a coding agent can discover how to use installed capabilities.
Templates contain placeholders and are not executable requests; the operation's
input and authority requirements still apply. Each operation has
`runtime_probed: false`: catalog presence is not a successful runtime probe.

`VERIFIED` describes checks within the stated declared-relationship scope.
`UNVERIFIED` means the required relationship or current evidence is unavailable.
`INCOMPATIBLE` reports an invalid installed component or a known dependency
contract version mismatch. Empty dependency lists do not prove pairwise
compatibility. In particular, the current seven-engine package can have verified
installation integrity while its overall compatibility remains `UNVERIFIED`.
Known incompatibility affects the report without hiding independent engine rows.

CLI exit 0 means the diagnostic's declared scope was verified; exit 4 reports
unverified or incompatible results, including unavailable native diagnostics.
`ptsip engines status --json` retains its existing installation-only behavior.
An older installed Discovery binary produces a structured unavailable result.

The CLI adapter locates its own distribution and scripts directory and checks
the Discovery artifact hash before starting it. Rust reads the bundle origins,
METADATA and installed contracts, validates identities and current file hashes,
then rechecks the observed sources before returning. JSON is UTF-8 on every host.

Local source agreement is not independent release attestation, project
conformance, operational readiness, or an execution permission grant. Planning
and Storage keep their existing responsibilities. No Bootstrap Integrated Engine
schema, AGENTS.md generation, legacy API retirement, or publication is added.
