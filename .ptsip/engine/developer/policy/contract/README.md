# Developer policy contracts

`entry` resolves the registered policy kind and Family. `check` is read-only.
`authoring` creates new files exclusively; `update` changes existing Unified
Policy content and its canonical source digest atomically. The common source
package remains the sole digest algorithm owner.

## Delegated policy updates

The owner-controlled repository checkout is the trust boundary. Within the
existing WU-15 delegation, the coding agent records the exact update request
in `approval.authorized_update_requests` in the existing WU-15 plan and commits
that record using the already granted authority. This is an exact request
record, not a new user approval or an independent authentication service.
The CLI reads the plan from Git HEAD, not a caller-selected evidence file or
uncommitted plan edits. A request reference or approval boolean by itself
does not grant authority. Do not execute from untrusted checkouts.

The request contains the existing `target`, `baseline_file_sha256`,
`approval_ref` and `values` fields. Reuse the same request for preview and
execution; the committed plan must contain its exact content under the
unique reference. New or out-of-scope policy decisions remain with the owner.

From the repository root:

```powershell
go -C .ptsip/engine/developer run -mod=readonly . --root ../../.. --validator policy-rules --operation update --input update-request.json --dry-run
go -C .ptsip/engine/developer run -mod=readonly . --root ../../.. --validator policy-rules --operation update --input update-request.json
```

The preview is read-only. Execution checks the exact request, repository,
policy identity and baseline, then writes the content and canonical source
digest together and independently verifies readback. A stale baseline is
rejected; no dedicated approval-consumption storage is required. Historical
digest mismatch repair remains separate. No key provisioning or per-request
signature is required.
