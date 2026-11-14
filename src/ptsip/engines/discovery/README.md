# Installed official engine diagnostics

`ptsip engines compatibility --json` delegates to the Rust Discovery operation
`discovery.check-environment`. It diagnoses the official engines registered in the selected Python
installation. The diagnostic reads the declared package, metadata, and scripts
locations.

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
compatibility. In particular, a package with independent engines can have verified
installation integrity while its overall compatibility remains `UNVERIFIED`.
Known incompatibility affects the report without hiding independent engine rows.

CLI exit 0 means the diagnostic's declared scope was verified; exit 4 reports
unverified or incompatible results, including unavailable native diagnostics.
An older installed Discovery binary produces a structured unavailable result.

The CLI adapter locates its own distribution and scripts directory and checks
the Discovery artifact hash before starting it. Rust reads the bundle origins,
METADATA and installed contracts, validates identities and current file hashes,
then rechecks the observed sources before returning. JSON is UTF-8 on every host.

The report covers declared compatibility evidence and records execution
authorization as false.

## Plan discovery

Run the Discovery binary with one `discovery.planning` JSON request on stdin.
Provide the owner root, the exact catalog reference (`id`, `version`, `path`,
`sha256`) and the plan ID. Within the selected catalog, a plan ID must resolve
to one version. The engine verifies the selected plan's identity and exact
source bytes and returns its reference without interpreting its contents.
