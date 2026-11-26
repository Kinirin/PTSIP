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

## Direct user approval

The developer CLI verifies detached Ed25519 approval evidence using the admitted
user's deployment-pinned public key. The trusted deployment sets the linker
variables `main.policyUpdateApprovalPublicKey` (standard base64 of the 32-byte
public key) and `main.policyUpdateApprovalReplayDirectory` (an absolute,
operator-managed directory outside the worktree) with `go build -ldflags` and
`-X`. The directory must already exist and be protected from modification by
request submitters. Keep the signing key with the admitted user. A build
without these settings reports `POLICY_UPDATE_APPROVAL_PROVIDER_UNAVAILABLE`.

From the repository root, the read-only preview derives the binding for the
owner to review alongside the request:

```powershell
go -C .ptsip/engine/developer run . --root ../../.. --validator policy-rules --operation update --input update-request.json --dry-run
```

The owner signs the exact UTF-8 JSON payload containing
`format: "ptsip-direct-policy-update-approval/v1"`, UTC `issued_at`,
`expires_at`, and the complete preview `approval` object. Set the envelope's
`payload` to the standard base64 of those JSON bytes and its `signature` to
the standard base64 of the detached Ed25519 signature. Use a unique
`approval_ref` for each direct approval. Supply the envelope to the deployed
CLI with `--approval-evidence` and submit the same request and repository
root used in the preview:

```powershell
ptsip-developer --root . --validator policy-rules --operation update --input update-request.json --approval-evidence C:/private/approval.json
```

The verifier checks the signer, validity window, repository root and exact
request binding before consuming the approval reference. Consumption is
persistent even if a later check or write fails. Retrying a consumed request
requires new direct approval. The existing atomic update and readback checks
run after approval. A previously mismatched source requires a separate
reconciliation decision.

From the repository root, run:

```powershell
go -C developer/tests test -mod=readonly ./policy/contract/... -count=1
go -C src/ptsip/common_contract test -mod=readonly ./source ./cmd/ptsip-common-contract -count=1
```
