from __future__ import annotations

import hashlib
import json
import re
from copy import deepcopy
from pathlib import Path
from typing import Mapping

import yaml
from jsonschema import Draft202012Validator, ValidationError

from .model import (
    CheckStatus,
    EligibilityCheck,
    EligibilityResult,
    EligibilityStatus,
    GovernanceAuthorityError,
    LifecycleState,
    ProjectAuthorityRecord,
    RepositoryBinding,
    SolveSubject,
    SubjectMatchKind,
)
from .subject_matching import match_subject_binding


_CHECK_IDS = (
    "CURRENT_ADR_IS_SELECTED",
    "AUTHORITY_ROLE_IS_RESOLVED",
    "AUTHORITY_ROLE_IS_PROJECT_AUTHORITY_ELIGIBLE",
    "AUTHORITY_CONTRACT_IS_VALID",
    "AUTHORITY_CONTRACT_LIFECYCLE_IS_CURRENTLY_USABLE",
    "TOOL_SUPPORT_IS_SUFFICIENT",
    "REPOSITORY_BINDING_MATCHES",
    "SUBJECT_BINDING_MATCHES",
    "CURRENT_APPLICABILITY_IS_APPLICABLE",
)
_SHA40 = re.compile(r"^[0-9a-f]{40}$")


def _canonical_json(value: object) -> str:
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False)


def _stable_digest(value: object) -> str:
    return hashlib.sha256(_canonical_json(value).encode("utf-8")).hexdigest()


def _mapping(value: object, *, code: str, label: str) -> Mapping[str, object]:
    if not isinstance(value, Mapping):
        raise GovernanceAuthorityError(code, f"{label} must be a mapping.", value)
    return value


class AuthorityCatalog:
    """Load shipped Support Feature authority contracts only.

    Developer policy, owner grants, planning state, and legacy decisions are not
    product-runtime authority inputs.
    """

    INDEX = "src/ptsip/specdata/support-policy-index.yaml"
    AUTHORITY_SCHEMA_REGISTRY = "src/ptsip/specdata/ptsip-support-authority-schema-registry.yaml"
    AUTHORITY_ROLE_REGISTRY = "src/ptsip/specdata/ptsip-support-authority-role-registry.yaml"
    AUTHORITY_SUBJECT_REGISTRY = "src/ptsip/specdata/ptsip-support-authority-subject-registry.yaml"

    SUPPORT_POLICY_SCHEMA = "src/ptsip/specdata/ptsip-support-feature-policy.schema.json"
    SUPPORT_INDEX_SCHEMA = "src/ptsip/specdata/ptsip-support-feature-policy-index.schema.json"
    AUTHORITY_SEMANTICS_SCHEMA = "src/ptsip/specdata/ptsip-support-authority-semantics.schema.json"
    AUTHORITY_ROLE_SCHEMA = "src/ptsip/specdata/ptsip-support-authority-role.schema.json"
    AUTHORITY_SUBJECT_SCHEMA = "src/ptsip/specdata/ptsip-support-subject-binding.schema.json"
    PROJECT_AUTHORITY_RECORD_SCHEMA = "src/ptsip/specdata/ptsip-support-project-authority-record.schema.json"
    ELIGIBILITY_RESULT_SCHEMA = "src/ptsip/specdata/ptsip-support-authority-eligibility-result.schema.json"

    def __init__(self, repository_root: str | Path) -> None:
        self.root = Path(repository_root).resolve()
        self.index = self._load_yaml(self.INDEX)
        self.authority_schema_registry = self._load_yaml(self.AUTHORITY_SCHEMA_REGISTRY)
        self.role_registry = self._load_yaml(self.AUTHORITY_ROLE_REGISTRY)
        self.subject_registry = self._load_yaml(self.AUTHORITY_SUBJECT_REGISTRY)
        self.policy_schema = self._load_json(self.SUPPORT_POLICY_SCHEMA)
        self.index_schema = self._load_json(self.SUPPORT_INDEX_SCHEMA)
        self.semantics_schema = self._load_json(self.AUTHORITY_SEMANTICS_SCHEMA)
        self.role_schema = self._load_json(self.AUTHORITY_ROLE_SCHEMA)
        self.subject_schema = self._load_json(self.AUTHORITY_SUBJECT_SCHEMA)
        self.project_authority_record_schema = self._load_json(self.PROJECT_AUTHORITY_RECORD_SCHEMA)
        self.eligibility_result_schema = self._load_json(self.ELIGIBILITY_RESULT_SCHEMA)
        self._validate_catalog_assets()

    def _path(self, relative: str) -> Path:
        path = Path(relative)
        if path.is_absolute() or ".." in path.parts:
            raise GovernanceAuthorityError("GOVERNANCE_PATH_INVALID", "support governance paths must be repository-relative and traversal-free.", relative)
        resolved = (self.root / path).resolve()
        if self.root not in resolved.parents and resolved != self.root:
            raise GovernanceAuthorityError("GOVERNANCE_PATH_INVALID", "support governance path escaped repository root.", relative)
        return resolved

    def _load_yaml(self, relative: str) -> dict[str, object]:
        path = self._path(relative)
        if not path.is_file():
            raise GovernanceAuthorityError("GOVERNANCE_ASSET_MISSING", f"missing support governance asset: {relative}", relative)
        value = yaml.safe_load(path.read_text(encoding="utf-8"))
        if not isinstance(value, dict):
            raise GovernanceAuthorityError("GOVERNANCE_ASSET_INVALID", f"support governance asset must contain a mapping: {relative}", value)
        return value

    def _load_json(self, relative: str) -> dict[str, object]:
        path = self._path(relative)
        if not path.is_file():
            raise GovernanceAuthorityError("GOVERNANCE_SCHEMA_MISSING", f"missing support governance schema: {relative}", relative)
        value = json.loads(path.read_text(encoding="utf-8"))
        if not isinstance(value, dict):
            raise GovernanceAuthorityError("GOVERNANCE_SCHEMA_INVALID", f"support governance schema must contain a mapping: {relative}", value)
        return value

    def _validate_catalog_assets(self) -> None:
        for schema in (
            self.policy_schema, self.index_schema, self.semantics_schema,
            self.role_schema, self.subject_schema,
            self.project_authority_record_schema, self.eligibility_result_schema,
        ):
            Draft202012Validator.check_schema(schema)
        Draft202012Validator(self.index_schema).validate(self.index)
        roles = self.role_registry.get("policy_roles")
        if not isinstance(roles, list):
            raise GovernanceAuthorityError("AUTHORITY_ROLE_REGISTRY_INVALID", "support role registry requires policy_roles.", roles)
        tokens = self.role_registry.get("effect_vocabulary", {})
        if not isinstance(tokens, Mapping) or tokens.get("count") != len(tokens.get("tokens", [])):
            raise GovernanceAuthorityError("AUTHORITY_ROLE_REGISTRY_INVALID", "support effect vocabulary count mismatch.", tokens)

    @property
    def current_routes(self) -> Mapping[str, object]:
        policies = self.index.get("policies")
        if not isinstance(policies, list):
            raise GovernanceAuthorityError("SUPPORT_POLICY_INDEX_INVALID", "support policy index policies must be a list.", policies)
        routes: dict[str, object] = {}
        for item in policies:
            route = _mapping(item, code="SUPPORT_POLICY_INDEX_INVALID", label="support policy route")
            policy_id = route.get("id")
            if not isinstance(policy_id, str) or policy_id in routes:
                raise GovernanceAuthorityError("SUPPORT_POLICY_INDEX_INVALID", "support policy ids must be unique strings.", policy_id)
            routes[policy_id] = route
        return routes

    def load_current_record(self, policy_id: str) -> tuple[str, Mapping[str, object], Mapping[str, object]]:
        route = _mapping(self.current_routes.get(policy_id), code="CURRENT_SUPPORT_POLICY_NOT_SELECTED", label=f"route for {policy_id}")
        path = route.get("path")
        if not isinstance(path, str):
            raise GovernanceAuthorityError("SUPPORT_POLICY_INDEX_INVALID", "support policy route requires a path.", route)
        record = self._load_yaml(path)
        return path, route, record

    def iter_current_records(self) -> tuple[tuple[str, str, Mapping[str, object], Mapping[str, object]], ...]:
        items = []
        for policy_id in self.current_routes:
            path, route, record = self.load_current_record(policy_id)
            items.append((policy_id, path, route, record))
        return tuple(items)

    def _validate_current_selection(self, policy_id: str, source_ref: str, route: Mapping[str, object], record: Mapping[str, object]) -> None:
        policy = _mapping(record.get("policy"), code="MALFORMED_AUTHORITY_RECORD", label="policy")
        if policy.get("id") != policy_id or route.get("id") != policy_id or route.get("path") != source_ref:
            raise GovernanceAuthorityError("CURRENT_SUPPORT_POLICY_SELECTION_MISMATCH", "support policy record does not exactly match support-policy-index.", {"policy_id":policy_id,"source_ref":source_ref})

    def _role_for_policy(self, policy_id: str) -> Mapping[str, object]:
        roles = self.role_registry.get("policy_roles", [])
        found = [item for item in roles if isinstance(item, Mapping) and item.get("policy_id") == policy_id]
        if len(found) != 1:
            raise GovernanceAuthorityError("AUTHORITY_ROLE_UNRESOLVED", "support policy must have exactly one role registry entry.", policy_id)
        return found[0]

    def _validate_role(self, record: Mapping[str, object]) -> Mapping[str, object]:
        policy = _mapping(record.get("policy"), code="MALFORMED_AUTHORITY_RECORD", label="policy")
        role = dict(self._role_for_policy(str(policy.get("id"))))
        role.pop("policy_id", None)
        try:
            Draft202012Validator(self.role_schema).validate(role)
        except ValidationError as exc:
            raise GovernanceAuthorityError("AUTHORITY_ROLE_INVALID", exc.message, role) from exc
        allowed = set(self.role_registry["effect_vocabulary"]["tokens"])
        if any(item not in allowed for item in role["effects"]):
            raise GovernanceAuthorityError("AUTHORITY_ROLE_UNREGISTERED_EFFECT", "support authority role contains an unregistered effect token.", role["effects"])
        return role

    def _validate_contract_and_semantics(self, record: Mapping[str, object]) -> Mapping[str, object]:
        policy = _mapping(record.get("policy"), code="MALFORMED_AUTHORITY_RECORD", label="policy")
        policy_id = str(policy.get("id"))
        entries = self.authority_schema_registry.get("entries", [])
        found = [item for item in entries if isinstance(item, Mapping) and item.get("policy_id") == policy_id]
        if len(found) != 1:
            raise GovernanceAuthorityError("UNSUPPORTED_AUTHORITY_TYPE", "support policy has no exact schema registry entry.", policy_id)
        entry = found[0]
        contract = _mapping(record.get("authority_contract"), code="MALFORMED_AUTHORITY_RECORD", label="authority_contract")
        if (
            contract.get("authority_type") != entry.get("authority_type")
            or contract.get("schema_id") != entry.get("schema_id")
            or contract.get("schema_version") != entry.get("schema_version")
        ):
            raise GovernanceAuthorityError("AUTHORITY_CONTRACT_MISMATCH", "support authority contract does not match its registry entry.", contract)
        definition = self.semantics_schema.get("$defs", {}).get(entry.get("schema_definition"))
        if not isinstance(definition, Mapping):
            raise GovernanceAuthorityError("UNSUPPORTED_AUTHORITY_SCHEMA_VERSION", "support semantic schema definition is unavailable.", entry.get("schema_definition"))
        try:
            Draft202012Validator(definition).validate(record["authority_semantics"])
        except ValidationError as exc:
            raise GovernanceAuthorityError("AUTHORITY_SEMANTICS_INVALID", exc.message, record["authority_semantics"]) from exc
        return entry

    def lifecycle_state(self, record: Mapping[str, object]) -> LifecycleState:
        policy = _mapping(record.get("policy"), code="MALFORMED_AUTHORITY_RECORD", label="policy")
        state = policy.get("status")
        try:
            return LifecycleState(state)
        except (ValueError, TypeError) as exc:
            raise GovernanceAuthorityError("AUTHORITY_LIFECYCLE_UNREGISTERED", "support policy status is not a registered lifecycle state.", state) from exc

    def lifecycle_is_eligible(self, state: LifecycleState) -> bool:
        lifecycle = _mapping(self.authority_schema_registry.get("lifecycle_policy"), code="AUTHORITY_LIFECYCLE_POLICY_INVALID", label="lifecycle_policy")
        return state.value in lifecycle.get("project_authority_eligible_states", [])

    def binding_for_policy(self, policy_id: str, solve_subject: SolveSubject) -> dict[str, object]:
        return {
            "authority_domain": "PROJECT_GOVERNANCE",
            "repository_binding": solve_subject.repository_binding.as_dict(),
            "subject_type": "GOVERNANCE_TOPIC",
            "subject_identity": {"scheme": "SUPPORT_POLICY_ID", "value": policy_id},
        }

    def validate_binding_registration(self, binding: Mapping[str, object]) -> None:
        try:
            Draft202012Validator(self.subject_schema).validate(binding)
        except ValidationError as exc:
            raise GovernanceAuthorityError("AUTHORITY_BINDING_INVALID", exc.message, binding) from exc
        identity = _mapping(binding["subject_identity"], code="AUTHORITY_BINDING_INVALID", label="subject_identity")
        schemes = _mapping(self.subject_registry.get("subject_identity_schemes"), code="AUTHORITY_SUBJECT_REGISTRY_INVALID", label="subject_identity_schemes")
        entry = schemes.get(identity.get("scheme"))
        if not isinstance(entry, Mapping) or identity.get("value") not in entry.get("registered_values", []):
            raise GovernanceAuthorityError("UNREGISTERED_AUTHORITY_SUBJECT_IDENTITY", "support policy identity is not registered.", identity)
        repository = _mapping(binding["repository_binding"], code="AUTHORITY_BINDING_INVALID", label="repository_binding")
        repository_schemes = _mapping(self.subject_registry.get("repository_identity_schemes"), code="AUTHORITY_SUBJECT_REGISTRY_INVALID", label="repository_identity_schemes")
        if repository.get("scheme") not in repository_schemes:
            raise GovernanceAuthorityError("UNSUPPORTED_REPOSITORY_IDENTITY_SCHEME", "repository identity scheme is not supported.", repository.get("scheme"))

    def validate_current_corpus(self) -> tuple[str, ...]:
        validated = []
        for policy_id, source_ref, route, record in self.iter_current_records():
            self._validate_current_selection(policy_id, source_ref, route, record)
            Draft202012Validator(self.policy_schema).validate(record)
            self._validate_role(record)
            self._validate_contract_and_semantics(record)
            if policy_id not in self.subject_registry["subject_identity_schemes"]["SUPPORT_POLICY_ID"]["registered_values"]:
                raise GovernanceAuthorityError("UNREGISTERED_AUTHORITY_SUBJECT_IDENTITY", "support policy id is absent from subject registry.", policy_id)
            validated.append(policy_id)
        return tuple(validated)


class ProjectAuthorityRuntime:
    """Fresh-solve Support Feature authority evaluator and projector."""

    def __init__(self, repository_root: str | Path) -> None:
        self.catalog = AuthorityCatalog(repository_root)

    def _blank_checks(self) -> dict[str, EligibilityCheck]:
        return {check_id: EligibilityCheck(check_id, CheckStatus.NOT_EVALUATED) for check_id in _CHECK_IDS}

    def _result(self, *, authority_id: str, source_ref: str, status: EligibilityStatus, checks: Mapping[str, EligibilityCheck], lifecycle_state: LifecycleState | None = None, subject_match: SubjectMatchKind | None = None, diagnostics: tuple[str, ...] = ()) -> EligibilityResult:
        result = EligibilityResult(authority_id=authority_id,source_ref=source_ref,status=status,lifecycle_state=lifecycle_state,subject_match=subject_match,checks=tuple(checks[item] for item in _CHECK_IDS),diagnostics=diagnostics)
        Draft202012Validator(self.catalog.eligibility_result_schema).validate(result.as_dict())
        return result

    def evaluate_policy(self, policy_id: str, solve_subject: SolveSubject, *, source_revision: str) -> EligibilityResult:
        if not _SHA40.fullmatch(source_revision):
            raise GovernanceAuthorityError("AUTHORITY_SOURCE_REVISION_INVALID", "source_revision must be a full lowercase 40-hex Git revision.", source_revision)
        source_ref, route, record = self.catalog.load_current_record(policy_id)
        authority_id = str(_mapping(record.get("policy"), code="MALFORMED_AUTHORITY_RECORD", label="policy").get("id", "UNKNOWN"))
        checks = self._blank_checks()
        try:
            self.catalog._validate_current_selection(policy_id, source_ref, route, record)
            checks["CURRENT_ADR_IS_SELECTED"] = EligibilityCheck("CURRENT_ADR_IS_SELECTED", CheckStatus.PASS)
        except GovernanceAuthorityError as exc:
            checks["CURRENT_ADR_IS_SELECTED"] = EligibilityCheck("CURRENT_ADR_IS_SELECTED", CheckStatus.FAIL, exc.code)
            return self._result(authority_id=authority_id,source_ref=source_ref,status=EligibilityStatus.MALFORMED_AUTHORITY_RECORD,checks=checks,diagnostics=(exc.code,))

        try:
            role = self.catalog._validate_role(record)
        except GovernanceAuthorityError as exc:
            checks["AUTHORITY_ROLE_IS_RESOLVED"] = EligibilityCheck("AUTHORITY_ROLE_IS_RESOLVED", CheckStatus.FAIL, exc.code)
            return self._result(authority_id=authority_id,source_ref=source_ref,status=EligibilityStatus.MALFORMED_AUTHORITY_RECORD,checks=checks,diagnostics=(exc.code,))
        if role["resolution_state"] not in {"DECLARED","DETERMINISTICALLY_DERIVED"}:
            diagnostic = "AUTHORITY_ROLE_ADVISORY_ONLY" if role["resolution_state"] == "ADVISORY_CANDIDATE" else "AUTHORITY_ROLE_UNRESOLVED"
            checks["AUTHORITY_ROLE_IS_RESOLVED"] = EligibilityCheck("AUTHORITY_ROLE_IS_RESOLVED", CheckStatus.FAIL, diagnostic)
            return self._result(authority_id=authority_id,source_ref=source_ref,status=EligibilityStatus.CURRENTLY_INELIGIBLE,checks=checks,diagnostics=(diagnostic,))
        checks["AUTHORITY_ROLE_IS_RESOLVED"] = EligibilityCheck("AUTHORITY_ROLE_IS_RESOLVED", CheckStatus.PASS)
        if role["projection_role"] != "PROJECT_ARCHITECTURE_AUTHORITY":
            checks["AUTHORITY_ROLE_IS_PROJECT_AUTHORITY_ELIGIBLE"] = EligibilityCheck("AUTHORITY_ROLE_IS_PROJECT_AUTHORITY_ELIGIBLE",CheckStatus.FAIL,"AUTHORITY_ROLE_NOT_PROJECT_AUTHORITY_ELIGIBLE")
            return self._result(authority_id=authority_id,source_ref=source_ref,status=EligibilityStatus.CURRENTLY_INELIGIBLE,checks=checks,diagnostics=("AUTHORITY_ROLE_NOT_PROJECT_AUTHORITY_ELIGIBLE",))
        checks["AUTHORITY_ROLE_IS_PROJECT_AUTHORITY_ELIGIBLE"] = EligibilityCheck("AUTHORITY_ROLE_IS_PROJECT_AUTHORITY_ELIGIBLE", CheckStatus.PASS)

        try:
            self.catalog._validate_contract_and_semantics(record)
        except GovernanceAuthorityError as exc:
            checks["AUTHORITY_CONTRACT_IS_VALID"] = EligibilityCheck("AUTHORITY_CONTRACT_IS_VALID", CheckStatus.FAIL, exc.code)
            status = EligibilityStatus.TOOL_CAPABILITY_GAP if exc.code in {"UNSUPPORTED_AUTHORITY_TYPE","UNSUPPORTED_AUTHORITY_SCHEMA_VERSION"} else EligibilityStatus.MALFORMED_AUTHORITY_RECORD
            return self._result(authority_id=authority_id,source_ref=source_ref,status=status,checks=checks,diagnostics=(exc.code,))
        checks["AUTHORITY_CONTRACT_IS_VALID"] = EligibilityCheck("AUTHORITY_CONTRACT_IS_VALID", CheckStatus.PASS)

        try:
            lifecycle_state = self.catalog.lifecycle_state(record)
        except GovernanceAuthorityError as exc:
            checks["AUTHORITY_CONTRACT_LIFECYCLE_IS_CURRENTLY_USABLE"] = EligibilityCheck("AUTHORITY_CONTRACT_LIFECYCLE_IS_CURRENTLY_USABLE",CheckStatus.FAIL,exc.code)
            return self._result(authority_id=authority_id,source_ref=source_ref,status=EligibilityStatus.MALFORMED_AUTHORITY_RECORD,checks=checks,diagnostics=(exc.code,))
        if not self.catalog.lifecycle_is_eligible(lifecycle_state):
            diagnostic=f"AUTHORITY_LIFECYCLE_{lifecycle_state.value}"
            checks["AUTHORITY_CONTRACT_LIFECYCLE_IS_CURRENTLY_USABLE"] = EligibilityCheck("AUTHORITY_CONTRACT_LIFECYCLE_IS_CURRENTLY_USABLE",CheckStatus.FAIL,diagnostic)
            return self._result(authority_id=authority_id,source_ref=source_ref,status=EligibilityStatus.CURRENTLY_INELIGIBLE,lifecycle_state=lifecycle_state,checks=checks,diagnostics=(diagnostic,))
        checks["AUTHORITY_CONTRACT_LIFECYCLE_IS_CURRENTLY_USABLE"] = EligibilityCheck("AUTHORITY_CONTRACT_LIFECYCLE_IS_CURRENTLY_USABLE", CheckStatus.PASS)
        checks["TOOL_SUPPORT_IS_SUFFICIENT"] = EligibilityCheck("TOOL_SUPPORT_IS_SUFFICIENT", CheckStatus.PASS)

        binding = self.catalog.binding_for_policy(policy_id, solve_subject)
        try:
            self.catalog.validate_binding_registration(binding)
        except GovernanceAuthorityError as exc:
            checks["REPOSITORY_BINDING_MATCHES"] = EligibilityCheck("REPOSITORY_BINDING_MATCHES", CheckStatus.FAIL, exc.code)
            status = EligibilityStatus.TOOL_CAPABILITY_GAP if exc.code.startswith("UNSUPPORTED_") or exc.code.startswith("UNREGISTERED_") else EligibilityStatus.MALFORMED_AUTHORITY_RECORD
            return self._result(authority_id=authority_id,source_ref=source_ref,status=status,lifecycle_state=lifecycle_state,checks=checks,diagnostics=(exc.code,))
        checks["REPOSITORY_BINDING_MATCHES"] = EligibilityCheck("REPOSITORY_BINDING_MATCHES", CheckStatus.PASS)

        try:
            subject_match = match_subject_binding(binding, solve_subject, self.catalog.subject_registry)
        except GovernanceAuthorityError as exc:
            checks["SUBJECT_BINDING_MATCHES"] = EligibilityCheck("SUBJECT_BINDING_MATCHES", CheckStatus.FAIL, exc.code)
            return self._result(authority_id=authority_id,source_ref=source_ref,status=EligibilityStatus.MALFORMED_AUTHORITY_RECORD,lifecycle_state=lifecycle_state,subject_match=SubjectMatchKind.INVALID_BINDING,checks=checks,diagnostics=(exc.code,))
        if subject_match is SubjectMatchKind.NO_MATCH:
            checks["SUBJECT_BINDING_MATCHES"] = EligibilityCheck("SUBJECT_BINDING_MATCHES",CheckStatus.FAIL,"SUBJECT_BINDING_NOT_APPLICABLE")
            checks["CURRENT_APPLICABILITY_IS_APPLICABLE"] = EligibilityCheck("CURRENT_APPLICABILITY_IS_APPLICABLE",CheckStatus.FAIL,"NOT_APPLICABLE")
            return self._result(authority_id=authority_id,source_ref=source_ref,status=EligibilityStatus.CURRENTLY_INELIGIBLE,lifecycle_state=lifecycle_state,subject_match=subject_match,checks=checks,diagnostics=("SUBJECT_BINDING_NOT_APPLICABLE",))
        checks["SUBJECT_BINDING_MATCHES"] = EligibilityCheck("SUBJECT_BINDING_MATCHES", CheckStatus.PASS)
        checks["CURRENT_APPLICABILITY_IS_APPLICABLE"] = EligibilityCheck("CURRENT_APPLICABILITY_IS_APPLICABLE", CheckStatus.PASS)
        return self._result(authority_id=authority_id,source_ref=source_ref,status=EligibilityStatus.CURRENTLY_ELIGIBLE,lifecycle_state=lifecycle_state,subject_match=subject_match,checks=checks)

    def evaluate_current_authorities(self, solve_subject: SolveSubject, *, source_revision: str) -> tuple[EligibilityResult, ...]:
        return tuple(self.evaluate_policy(policy_id, solve_subject, source_revision=source_revision) for policy_id in self.catalog.current_routes)

    def project_policy(self, policy_id: str, solve_subject: SolveSubject, *, source_revision: str) -> ProjectAuthorityRecord | None:
        result = self.evaluate_policy(policy_id, solve_subject, source_revision=source_revision)
        if not result.currently_eligible:
            return None
        source_ref, _, record = self.catalog.load_current_record(policy_id)
        role = self.catalog._validate_role(record)
        binding = self.catalog.binding_for_policy(policy_id, solve_subject)
        project_record = ProjectAuthorityRecord(
            authority_id=result.authority_id,
            authority_contract=deepcopy(record["authority_contract"]),
            authority_semantics=deepcopy(record["authority_semantics"]),
            authority_role=deepcopy(role),
            authority_provenance={"source_type":"SUPPORT_FEATURE_POLICY","source_ref":source_ref,"source_revision":source_revision,"source_digest":_stable_digest(record)},
            subject_binding=deepcopy(binding),
        )
        schema=deepcopy(self.catalog.project_authority_record_schema)
        schema["properties"]["authority_role"]=deepcopy(self.catalog.role_schema)
        schema["properties"]["subject_binding"]=deepcopy(self.catalog.subject_schema)
        Draft202012Validator(schema).validate(project_record.as_dict())
        return project_record

    def project_current_authorities(self, solve_subject: SolveSubject, *, source_revision: str) -> tuple[ProjectAuthorityRecord, ...]:
        records=[]
        for policy_id in self.catalog.current_routes:
            projected=self.project_policy(policy_id,solve_subject,source_revision=source_revision)
            if projected is not None:
                records.append(projected)
        return tuple(records)
