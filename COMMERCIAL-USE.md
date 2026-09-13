# PTSIP Commercial Use and Cooperation Procedure

This document defines the current Official Cooperation Channels and submission mechanics for PTSIP Community Reciprocity License 1.0.

**Effective Time:** 2026-09-15T00:00:00-07:00 (Pacific Time in effect on 2026-09-15), equivalent to 2026-09-15T07:00:00Z.

It does not add substantive duties beyond LICENSE. If this document conflicts with LICENSE, LICENSE controls.

## Standing Commercial Authorization

No individual approval response is required.

Standing Commercial Authorization becomes available when a Commercial User:

1. accepts PTSIP Community Reciprocity License 1.0;
2. expressly accepts both the applicable PTSIP License and the Commercial Cooperation Duty;
3. identifies the PTSIP Version or Revision actually used; and
4. submits a valid Commercial Use Registration within 90 calendar days of first Commercial Use; and
5. continues to comply with the applicable attribution and cooperation duties.

PTSIP does not require a default monetary license fee.

## Official Cooperation Channels

### Public registration, reports, and non-confidential findings

Use the PTSIP GitHub Issues page:

https://github.com/Kinirin/PTSIP/issues

Use one of these title prefixes:

- [License][Commercial Registration]
- [License][Commercial Report]
- [License][Final Commercial Report]
- [License][Research Report]
- [License][Research Summary]
- [License][Material Finding]

The GitHub issue timestamp is sufficient evidence of submission.

### PTSIP source changes

Submit source changes through a pull request to:

https://github.com/Kinirin/PTSIP

A pull request, patch link, or equivalent source-form submission may satisfy the PTSIP-modification return requirement when it contains enough information to review and reproduce the PTSIP-specific change.

### Security-sensitive or confidential findings

Do not place credentials, personal data, customer secrets, or other protected information in a public issue.

If the repository exposes a private vulnerability-reporting or other private project channel, use that channel.

If no private channel is available, submit a redacted public Material Finding containing only enough non-confidential information to identify that a finding exists and to request a private follow-up channel. The License never requires public disclosure of protected information.

## Commercial Use Registration template

Title:

[License][Commercial Registration] <organization or project name>

Body:

### User
Individual or legal entity:

### Commercial activity
Product, service, system, business activity, or internal commercial function:

### First Commercial Use date
YYYY-MM-DD:

### PTSIP License version — REQUIRED
PTSIP Community Reciprocity License version:

### PTSIP version / immutable revision — REQUIRED
Provide at least one precise identifier. Include both when reasonably available.
Tool/release version:
Specification family:
Immutable Specification revision:
Git commit or other immutable revision:

### Primary environment
Relevant coding agents, CI, operating environment, platforms, or integrations:

### Responsible contact
GitHub account or other accountable project contact:

### Attribution location
Where PTSIP attribution is or will be provided:

### License and Cooperation Duty acceptance — REQUIRED
By submitting this Commercial Use Registration, the registering individual or Legal Entity expressly accepts:

1. the PTSIP Community Reciprocity License version identified in this registration; and
2. the Commercial Cooperation Duty, including the applicable attribution, reporting, Material Finding, PTSIP-modification return, cure, suspension, and reinstatement obligations.

We acknowledge that the Commercial Cooperation Duty is a material and essential condition of Standing Commercial Authorization and is not merely a voluntary recommendation.

Submission of this registration constitutes affirmative acceptance of these terms for the Commercial Use identified above.

## Annual Commercial Cooperation Report template

Title:

[License][Commercial Report] <organization or project name> <reporting period>

Body:

### Reporting period
Start:
End:

### PTSIP License version — REQUIRED

### PTSIP versions / immutable revisions used — REQUIRED
Provide at least one precise identifier for each materially used PTSIP line; include both version and immutable revision when reasonably available.

### Usage scope
Where and how PTSIP was used.

### Relevant environments and integrations
Coding agents, CI, platforms, tools, or workflows.

### Compatibility findings
Successful compatibility, incompatibilities, and required workarounds.

### Policy and architecture findings
Ambiguous, incomplete, unexpectedly strict, unexpectedly permissive, or difficult-to-apply PTSIP policy behavior.

### Verification and failure findings
Known false positives, false negatives, incorrect decisions, failure patterns, or bypasses.

### Improvements
Concrete suggestions, missing capabilities, usability problems, or interoperability improvements.

### PTSIP modifications
Describe PTSIP-specific modifications and link the corresponding PR, patch, or technical contribution. State "none" only if none were made.

### Operational observations
Shareable aggregated findings that may help improve PTSIP. This may include agent behavior, compatibility matrices, evaluation results, or measured context, reasoning, token, credit, or development-cost observations.

### Protected information
Identify any categories omitted because they could not lawfully or reasonably be disclosed. Do not include the protected information itself.

### Attestation
To the best of our knowledge, this report is a truthful account of material PTSIP-related knowledge reasonably obtained during the reporting period.

## Final Commercial Cooperation Report

Use the annual report template and add:

### Commercial Use end date

### Reason use ended

### Remaining deployed or maintained PTSIP-related material
Explain whether any PTSIP or PTSIP-Derived Material continues to materially enable the commercial activity.

### Outstanding obligations
List any Material Finding or PTSIP modification submission that is still being completed.

## Research Cooperation Report template

Title:

[License][Research Report] <organization or project> <reporting period>

Body:

### Research purpose

### Research period

### PTSIP License version — REQUIRED

### PTSIP versions / immutable revisions — REQUIRED
Provide at least one precise identifier; include both when reasonably available.

### Research environment and integrations

### Compatibility observations

### Policy or architecture observations

### Failures or unexpected behavior

### PTSIP modifications

### Improvement suggestions

### Publications or presentations
Links if public and available.

### Protected information
Identify only the category of information omitted; do not disclose protected content.

## Research Cooperation Summary template

Title:

[License][Research Summary] <organization or project>

Body:

### Research purpose and outcome

### PTSIP License version — REQUIRED

### PTSIP versions / immutable revisions — REQUIRED
Provide at least one precise identifier; include both when reasonably available.

### Duration of material PTSIP use

### Most useful PTSIP behavior

### Most limiting or unclear PTSIP behavior

### Compatibility findings

### Material findings already reported
Links:

### PTSIP modifications or improvements returned
Links:

### Publication, presentation, or report
Link if public and available.

## Material Finding template

Title:

[License][Material Finding] <short description>

Body:

### PTSIP License version — REQUIRED

### PTSIP version / immutable revision — REQUIRED
Provide at least one precise identifier; include both when reasonably available.

### Finding category
Policy ambiguity / false positive / false negative / compatibility / agent behavior / conformance / security / other

### Summary

### Expected behavior

### Observed behavior

### Reproduction information
Provide only non-confidential information.

### Impact

### Workaround, if known

### Related PTSIP modification, if any

## Compliance timing principle

The compliance periods in the PTSIP License are intended to encourage voluntary cooperation and reasonable correction, not to create traps for users acting in good faith.

A missed ordinary deadline begins the cure process defined in LICENSE rather than being treated as a reason for immediate punitive action.

## Reporting anniversary

The normal commercial reporting anniversary is based on the first Commercial Use date stated in the registration.

Example:

First Commercial Use: 2026-10-15
First annual reporting period ends: 2027-10-14
First report due: 2028-01-12

A correction, late report, suspension, or automatic reinstatement does not reset that anniversary unless the applicable License or a later written agreement expressly says otherwise.

## Updating a registration

A new registration is not required for every internal change.

Update or add a registration when Commercial Use changes materially, such as:

- a materially different product or service;
- a materially different legal entity;
- a move from research-only use into Commercial Use;
- resumed Commercial Use after a real end of use;
- a materially different commercial use category not reasonably covered by the existing record.

Minor product renames, team changes, routine version upgrades, or ordinary technical refactoring do not by themselves require a new registration.

## No profit statement required

PTSIP does not ask Commercial Users to prove profit, revenue, or PTSIP-attributable financial return.

The duty follows Commercial Use, not profitability.

A report may state that a metric was not measured. That does not remove the obligation to provide other reasonably available PTSIP-related knowledge.
