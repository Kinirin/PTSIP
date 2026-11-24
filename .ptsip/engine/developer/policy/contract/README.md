# Developer policy contracts

`entry` resolves the declared policy kind and Family through the registered
contract graph. `check` performs read-only verification. `authoring` creates
new policy files with exclusive creation. Responsibility-Authority and Plan
status retain their own verifier routes.

`update.Apply` updates values in existing Unified Policy content items. Its
request names the target file, the SHA-256 of its current bytes, the approval
reference, and the replacement value for each approved item. The authorizer
receives the selected identity, repository, target, base and proposed byte
digests, canonical source digests, and changed item IDs. A trusted authorizer
verifies this exact change with the policy owner before accepting it.

The standalone developer CLI exposes `policy-rules --operation update` and
currently returns `POLICY_UPDATE_APPROVAL_PROVIDER_UNAVAILABLE`. An owner
integration can supply the trusted authorizer through `update.Run` or
`update.Apply`. A previously mismatched source requires a separate
reconciliation decision.

From the repository root, run:

```powershell
go -C developer/tests test -mod=readonly ./policy/contract/... -count=1
go -C src/ptsip/common_contract test -mod=readonly ./source ./cmd/ptsip-common-contract -count=1
```
