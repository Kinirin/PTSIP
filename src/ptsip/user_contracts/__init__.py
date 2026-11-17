"""Validate the independent, opt-in Policy v4 and user contracts."""

from dataclasses import dataclass

from ._references import policy_references, rbac_index, validate_rbac
from ._schema import POLICY_KEY, VERSION_TARGET, Issue, official_version_valid, structure_issues
from ._validator import ContractResult, validate_user_contracts

__all__ = [
    "ContractResult",
    "Issue",
    "PolicyValidation",
    "validate_policy_v4",
    "validate_rbac",
    "validate_user_contracts",
]


@dataclass(frozen=True)
class PolicyValidation:
    issues: tuple[Issue, ...] = ()
    user_contracts: tuple[ContractResult, ...] = ()
    ra_usage: tuple[tuple[str, tuple[tuple[str | int, ...], ...]], ...] = ()

    @property
    def valid(self):
        """Whether structural and applicable declared constraints passed."""
        return not self.issues


def validate_policy_v4(policy, *, project_root=None, ra=None, rac=None, aec=None, bootstrap=False):
    """Check explicitly supplied documents with the distributed Policy v4 contracts.

    The read-only RA usage view is projected from this policy's forward links.
    User-authored constraints are reported separately from structure and reference
    errors; passing these checks establishes neither user meaning nor authority.
    """
    structure = structure_issues(POLICY_KEY, policy)
    indexed, rbac_issues = rbac_index(ra, rac, aec)
    if structure:
        return PolicyValidation(structure + rbac_issues)
    references, usage = policy_references(policy, indexed)
    contracts = validate_user_contracts(policy, project_root, bootstrap=bootstrap)
    issues = [*rbac_issues, *references]
    for result in contracts:
        if result.targets and VERSION_TARGET not in result.targets and result.status in {
            "INVALID", "UNSUPPORTED", "UNBOUND",
        }:
            issues.extend(result.issues)
    identity = policy.get("Policy_Identity", {})
    if "version" in identity and not any(
        result.status == "VALID" and VERSION_TARGET in result.targets for result in contracts
    ) and not official_version_valid(identity["version"]):
        issues.append(Issue(
            "VERSION", ("Policy_Identity", "version"),
            "Version must match an official format or a configured user version contract",
        ))
    return PolicyValidation(tuple(issues), contracts, usage)
