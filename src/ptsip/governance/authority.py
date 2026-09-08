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
    """Load canonical governance only from decisions/ and schemas/."""

    INDEX = "decisions/INDEX.yaml"
    AUTHORITY_SCHEMA_REGISTRY = "decisions/AUTHORITY-SCHEMA-REGISTRY.yaml"
    AUTHORITY_ROLE_REGISTRY = "decisions/AUTHORITY-ROLE-REGISTRY.yaml"
    AUTHORITY_SUBJECT_REGISTRY = "decisions/AUTHORITY-SUBJECT-REGISTRY.yaml"

    ADR_INDEX_SCHEMA = "schemas/ptsip-adr-index.schema.json"
    ADR_SCHEMA = "schemas/ptsip-adr.schema.json"
    AUTHORITY_SCHEMA_REGISTRY_SCHEMA = "schemas/ptsip-governance-authority-registry.schema.json"
    AUTHORITY_ROLE_SCHEMA = "schemas/ptsip-governance-authority-role.schema.json"
    AUTHORITY_ROLE_REGISTRY_SCHEMA = "schemas/ptsip-governance-authority-role-registry.schema.json"
    AUTHORITY_SUBJECT_SCHEMA = "schemas/ptsip-governance-subject-binding.schema.json"
    AUTHORITY_SUBJECT_REGISTRY_SCHEMA = "schemas/ptsip-governance-authority-subject-registry.schema.json"
    AUTHORITY_SEMANTICS_SCHEMA = "schemas/ptsip-governance-authority-semantics.schema.json"
    PROJECT_AUTHORITY_RECORD_SCHEMA = "schemas/ptsip-project-authority-record.schema.json"
    ELIGIBILITY_RESULT_SCHEMA = "schemas/ptsip-authority-eligibility-result.schema.json"

    def __init__(self, repository_root: str | Path) -> None:
        self.root = Path(repository_root).resolve()
        self.index = self._load_yaml(self.INDEX)
        self.authority_schema_registry = self._load_yaml(self.AUTHORITY_SCHEMA_REGISTRY)
        self.role_registry = self._load_yaml(self.AUTHORITY_ROLE_REGISTRY)
        self.subject_registry = self._load_yaml(self.AUTHORITY_SUBJECT_REGISTRY)
        self.index_schema = self._load_json(self.ADR_INDEX_SCHEMA)
        self.adr_schema = self._load_json(self.ADR_SCHEMA)
        self.authority_schema_registry_schema = self._load_json(self.AUTHORITY_SCHEMA_REGISTRY_SCHEMA)
        self.role_schema = self._load_json(self.AUTHORITY_ROLE_SCHEMA)
        self.role_registry_schema = self._load_json(self.AUTHORITY_ROLE_REGISTRY_SCHEMA)
        self.subject_schema = self._load_json(self.AUTHORITY_SUBJECT_SCHEMA)
        self.subject_registry_schema = self._load_json(self.AUTHORITY_SUBJECT_REGISTRY_SCHEMA)
        self.semantics_schema = self._load_json(self.AUTHORITY_SEMANTICS_SCHEMA)
        self.project_authority_record_schema = self._load_json(self.PROJECT_AUTHORITY_RECORD_SCHEMA)
        self.eligibility_result_schema = self._load_json(self.ELIGIBILITY_RESULT_SCHEMA)
        self._validate_catalog_assets()

    def _path(self, relative: str) -> Path:
        path = Path(relative)
        if path.is_absolute() or ".." in path.parts:
            raise GovernanceAuthorityError(
                "GOVERNANCE_PATH_INVALID",
                "canonical governance paths must be repository-relative and traversal-free.",
                relative,
            )
        resolved = (self.root / path).resolve()
        if self.root not in resolved.parents and resolved != self.root:
            raise GovernanceAuthorityError("GOVERNANCE_PATH_INVALID", "canonical governance path escaped repository root.", relative)
        return resolved

    def _load_yaml(self, relative: str) -> dict[str, object]:
        path = self._path(relative)
        if not path.is_file():
            raise GovernanceAuthorityError("GOVERNANCE_ASSET_MISSING", f"missing governance asset: {relative}", relative)
        value = yaml.safe_load(path.read_text(encoding="utf-8"))
        if not isinstance(value, dict):
            raise GovernanceAuthorityError("GOVERNANCE_ASSET_INVALID", f"governance asset must contain a mapping: {relative}", value)
        return value

    def _load_json(self, relative: str) -> dict[str, object]:
        path = self._path(relative)
        if not path.is_file():
            raise GovernanceAuthorityError("GOVERNANCE_ASSET_MISSING", f"missing governance schema: {relative}", relative)
        value = json.loads(path.read_text(encoding="utf-8"))
        if not isinstance(value, dict):
            raise GovernanceAuthorityError("GOVERNANCE_ASSET_INVALID", f"governance schema must contain a mapping: {relative}", value)
        return value

    def _validate_catalog_assets(self) -> None:
        schemas = (
            self.index_schema,self.adr_schema,self.authority_schema_registry_schema,
            self.role_schema,self.role_registry_schema,self.subject_schema,
            self.subject_registry_schema,self.semantics_schema,
            self.project_authority_record_schema,self.eligibility_result_schema,
        )
        for schema in schemas:
            Draft202012Validator.check_schema(schema)
        Draft202012Validator(self.index_schema).validate(self.index)
        Draft202012Validator(self.authority_schema_registry_schema).validate(self.authority_schema_registry)
        Draft202012Validator(self.role_registry_schema).validate(self.role_registry)
        Draft202012Validator(self.subject_registry_schema).validate(self.subject_registry)
        tokens = self.role_registry["effect_vocabulary"]["tokens"]
        if len(tokens) != 99 or len(set(tokens)) != 99:
            raise GovernanceAuthorityError(
                "AUTHORITY_ROLE_REGISTRY_INVALID",
                "frozen authority effect vocabulary must contain exactly 99 unique tokens.",
                tokens,
            )

    @property
    def current_routes(self) -> Mapping[str, object]:
        return _mapping(self.index.get("topics"), code="ADR_INDEX_INVALID", label="decisions index topics")

    def load_current_record(self, topic_id: str) -> tuple[str, Mapping[str, object], Mapping[str, object]]:
        route = _mapping(self.current_routes.get(topic_id), code="CURRENT_ADR_NOT_SELECTED", label=f"route for {topic_id}")
        path = route.get("path")
        if not isinstance(path, str):
            raise GovernanceAuthorityError("ADR_INDEX_INVALID", "current ADR route requires a path.", route)
        record = self._load_yaml(path)
        return path, route, record

    def iter_current_records(self) -> tuple[tuple[str, str, Mapping[str, object], Mapping[str, object]], ...]:
        items = []
        for topic_id in sorted(self.current_routes):
            path, route, record = self.load_current_record(topic_id)
            items.append((topic_id, path, route, record))
        return tuple(items)

    def _validate_current_selection(self, topic_id: str, source_ref: str, route: Mapping[str, object], record: Mapping[str, object]) -> None:
        decision = _mapping(record.get("decision"), code="MALFORMED_AUTHORITY_RECORD", label="decision")
        if decision.get("topic_id") != topic_id or decision.get("id") != route.get("current_adr") or route.get("path") != source_ref:
            raise GovernanceAuthorityError(
                "CURRENT_ADR_SELECTION_MISMATCH",
                "current ADR record does not exactly match decisions/INDEX.yaml.",
                {"topic_id": topic_id, "source_ref": source_ref, "route": dict(route), "decision": dict(decision)},
            )

    def _validate_role(self, record: Mapping[str, object]) -> Mapping[str, object]:
        role = _mapping(record.get("authority_role"), code="MALFORMED_AUTHORITY_RECORD", label="authority_role")
        try:
            Draft202012Validator(self.role_schema).validate(role)
        except ValidationError as exc:
            raise GovernanceAuthorityError("AUTHORITY_ROLE_INVALID", exc.message, role) from exc
        allowed_effects = set(self.role_registry["effect_vocabulary"]["tokens"])
        effects = role.get("effects")
        if not isinstance(effects, list) or any(item not in allowed_effects for item in effects):
            raise GovernanceAuthorityError("AUTHORITY_ROLE_UNREGISTERED_EFFECT", "authority role contains an unregistered effect token.", effects)
        return role

    def _inline_adr_schema(self) -> dict[str, object]:
        schema = deepcopy(self.adr_schema)
        schema["properties"]["authority_role"] = deepcopy(self.role_schema)
        schema["properties"]["subject_binding"] = deepcopy(self.subject_schema)
        return schema

    def _resolve_contract_entry(self, record: Mapping[str, object]) -> Mapping[str, object]:
        contract = _mapping(record.get("authority_contract"), code="MALFORMED_AUTHORITY_RECORD", label="authority_contract")
        authority_type, schema_id, schema_version = contract.get("authority_type"), contract.get("schema_id"), contract.get("schema_version")
        entries = self.authority_schema_registry["entries"]
        by_type = [entry for entry in entries if entry["authority_type"] == authority_type]
        if not by_type:
            raise GovernanceAuthorityError("UNSUPPORTED_AUTHORITY_TYPE", f"authority type {authority_type!r} is not supported by the current tool registry.", contract)
        by_schema_id = [entry for entry in entries if entry["schema_id"] == schema_id]
        if by_schema_id and all(entry["authority_type"] != authority_type for entry in by_schema_id):
            raise GovernanceAuthorityError("AUTHORITY_CONTRACT_MISMATCH", "schema_id is registered to a different authority_type.", contract)
        same_family = [entry for entry in entries if entry["authority_type"] == authority_type and entry["schema_id"] == schema_id]
        if not same_family:
            raise GovernanceAuthorityError("AUTHORITY_CONTRACT_MISMATCH", "authority_type and schema_id are not a registered combination.", contract)
        exact = [entry for entry in same_family if entry["schema_version"] == schema_version]
        if not exact:
            raise GovernanceAuthorityError("UNSUPPORTED_AUTHORITY_SCHEMA_VERSION", "authority schema version is not supported by the current tool registry.", contract)
        return exact[0]

    def _validate_contract_and_semantics(self, record: Mapping[str, object]) -> Mapping[str, object]:
        try:
            Draft202012Validator(self._inline_adr_schema()).validate(record)
        except ValidationError as exc:
            raise GovernanceAuthorityError("MALFORMED_AUTHORITY_RECORD", exc.message, record) from exc
        entry = self._resolve_contract_entry(record)
        definition = entry["schema_definition"]
        definitions = self.semantics_schema.get("$defs", {})
        if definition not in definitions:
            raise GovernanceAuthorityError("UNSUPPORTED_AUTHORITY_SCHEMA_VERSION", "registered authority schema definition is not implemented by the tool.", definition)
        try:
            Draft202012Validator(definitions[definition]).validate(record["authority_semantics"])
        except ValidationError as exc:
            raise GovernanceAuthorityError("AUTHORITY_SEMANTICS_INVALID", exc.message, record["authority_semantics"]) from exc
        return entry

    def lifecycle_state(self, record: Mapping[str, object]) -> LifecycleState:
        decision = _mapping(record.get("decision"), code="MALFORMED_AUTHORITY_RECORD", label="decision")
        policy = _mapping(self.authority_schema_registry.get("lifecycle_policy"), code="AUTHORITY_LIFECYCLE_POLICY_INVALID", label="lifecycle_policy")
        mapping = _mapping(policy.get("decision_status_mapping"), code="AUTHORITY_LIFECYCLE_POLICY_INVALID", label="decision_status_mapping")
        state = mapping.get(decision.get("status"))
        try:
            return LifecycleState(state)
        except (ValueError, TypeError) as exc:
            raise GovernanceAuthorityError("AUTHORITY_LIFECYCLE_UNREGISTERED", "decision status has no registered authority lifecycle mapping.", decision.get("status")) from exc

    def lifecycle_is_eligible(self, state: LifecycleState) -> bool:
        policy = _mapping(self.authority_schema_registry.get("lifecycle_policy"), code="AUTHORITY_LIFECYCLE_POLICY_INVALID", label="lifecycle_policy")
        states = policy.get("project_authority_eligible_states")
        return isinstance(states, list) and state.value in states

    def validate_binding_registration(self, binding: Mapping[str, object]) -> None:
        try:
            Draft202012Validator(self.subject_schema).validate(binding)
        except ValidationError as exc:
            raise GovernanceAuthorityError("AUTHORITY_BINDING_INVALID", exc.message, binding) from exc
        domains = self.subject_registry.get("authority_domains", [])
        subject_types = self.subject_registry.get("subject_types", [])
        identity_schemes = _mapping(self.subject_registry.get("subject_identity_schemes"), code="AUTHORITY_SUBJECT_REGISTRY_INVALID", label="subject_identity_schemes")
        repository_schemes = _mapping(self.subject_registry.get("repository_identity_schemes"), code="AUTHORITY_SUBJECT_REGISTRY_INVALID", label="repository_identity_schemes")
        if binding["authority_domain"] not in domains:
            raise GovernanceAuthorityError("UNSUPPORTED_AUTHORITY_DOMAIN", "authority_domain is not registered.", binding["authority_domain"])
        if binding["subject_type"] not in subject_types:
            raise GovernanceAuthorityError("UNSUPPORTED_AUTHORITY_SUBJECT_TYPE", "subject_type is not registered.", binding["subject_type"])
        identity = _mapping(binding["subject_identity"], code="AUTHORITY_BINDING_INVALID", label="subject_identity")
        identity_entry = identity_schemes.get(identity.get("scheme"))
        if not isinstance(identity_entry, Mapping):
            raise GovernanceAuthorityError("UNSUPPORTED_AUTHORITY_SUBJECT_IDENTITY_SCHEME", "subject identity scheme is not registered.", identity.get("scheme"))
        if identity.get("value") not in identity_entry.get("registered_values", []):
            raise GovernanceAuthorityError("UNREGISTERED_AUTHORITY_SUBJECT_IDENTITY", "subject identity is not registered.", identity.get("value"))
        repository = _mapping(binding["repository_binding"], code="AUTHORITY_BINDING_INVALID", label="repository_binding")
        if repository.get("scheme") not in repository_schemes:
            raise GovernanceAuthorityError("UNSUPPORTED_REPOSITORY_IDENTITY_SCHEME", "repository identity scheme is not registered.", repository.get("scheme"))
        registered = self.subject_registry.get("current_repository_bindings", [])
        if not any(
            isinstance(item, Mapping)
            and item.get("scheme") == repository.get("scheme")
            and item.get("host") == repository.get("host")
            and item.get("repository_id") == repository.get("repository_id")
            for item in registered
        ):
            raise GovernanceAuthorityError("UNREGISTERED_REPOSITORY_BINDING", "authority repository binding is not registered as current.", repository)

    def validate_current_corpus(self) -> tuple[str, ...]:
        validated = []
        for topic_id, source_ref, route, record in self.iter_current_records():
            self._validate_current_selection(topic_id, source_ref, route, record)
            self._validate_role(record)
            self._validate_contract_and_semantics(record)
            binding = _mapping(record.get("subject_binding"), code="MALFORMED_AUTHORITY_RECORD", label="subject_binding")
            self.validate_binding_registration(binding)
            decision = _mapping(record.get("decision"), code="MALFORMED_AUTHORITY_RECORD", label="decision")
            identity = _mapping(binding.get("subject_identity"), code="AUTHORITY_BINDING_INVALID", label="subject_identity")
            if identity.get("value") != decision.get("topic_id"):
                raise GovernanceAuthorityError("AUTHORITY_SUBJECT_IDENTITY_MISMATCH", "ADR subject_identity must equal decision.topic_id for DECISION_TOPIC_ID.", binding)
            validated.append(str(decision["id"]))
        return tuple(validated)


class ProjectAuthorityRuntime:
    """Fresh-solve eligibility evaluator and ProjectAuthorityRecord projector."""

    def __init__(self, repository_root: str | Path) -> None:
        self.catalog = AuthorityCatalog(repository_root)

    def _blank_checks(self) -> dict[str, EligibilityCheck]:
        return {check_id: EligibilityCheck(check_id, CheckStatus.NOT_EVALUATED) for check_id in _CHECK_IDS}

    def _result(self, *, authority_id: str, source_ref: str, status: EligibilityStatus, checks: Mapping[str, EligibilityCheck], lifecycle_state: LifecycleState | None = None, subject_match: SubjectMatchKind | None = None, diagnostics: tuple[str, ...] = ()) -> EligibilityResult:
        result = EligibilityResult(
            authority_id=authority_id,source_ref=source_ref,status=status,
            lifecycle_state=lifecycle_state,subject_match=subject_match,
            checks=tuple(checks[item] for item in _CHECK_IDS),diagnostics=diagnostics,
        )
        Draft202012Validator(self.catalog.eligibility_result_schema).validate(result.as_dict())
        return result

    def evaluate_topic(self, topic_id: str, solve_subject: SolveSubject, *, source_revision: str) -> EligibilityResult:
        if not _SHA40.fullmatch(source_revision):
            raise GovernanceAuthorityError("AUTHORITY_SOURCE_REVISION_INVALID", "source_revision must be a full lowercase 40-hex Git revision.", source_revision)
        source_ref, route, record = self.catalog.load_current_record(topic_id)
        decision = _mapping(record.get("decision"), code="MALFORMED_AUTHORITY_RECORD", label="decision")
        authority_id = str(decision.get("id", route.get("current_adr", "UNKNOWN")))
        checks = self._blank_checks()

        try:
            self.catalog._validate_current_selection(topic_id, source_ref, route, record)
            checks["CURRENT_ADR_IS_SELECTED"] = EligibilityCheck("CURRENT_ADR_IS_SELECTED", CheckStatus.PASS)
        except GovernanceAuthorityError as exc:
            checks["CURRENT_ADR_IS_SELECTED"] = EligibilityCheck("CURRENT_ADR_IS_SELECTED", CheckStatus.FAIL, exc.code)
            return self._result(authority_id=authority_id,source_ref=source_ref,status=EligibilityStatus.MALFORMED_AUTHORITY_RECORD,checks=checks,diagnostics=(exc.code,))

        try:
            role = self.catalog._validate_role(record)
        except GovernanceAuthorityError as exc:
            checks["AUTHORITY_ROLE_IS_RESOLVED"] = EligibilityCheck("AUTHORITY_ROLE_IS_RESOLVED", CheckStatus.FAIL, exc.code)
            return self._result(authority_id=authority_id,source_ref=source_ref,status=EligibilityStatus.MALFORMED_AUTHORITY_RECORD,checks=checks,diagnostics=(exc.code,))

        resolution_state = role["resolution_state"]
        if resolution_state not in {"DECLARED", "DETERMINISTICALLY_DERIVED"}:
            diagnostic = "AUTHORITY_ROLE_ADVISORY_ONLY" if resolution_state == "ADVISORY_CANDIDATE" else "AUTHORITY_ROLE_UNRESOLVED"
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

        binding = _mapping(record.get("subject_binding"), code="MALFORMED_AUTHORITY_RECORD", label="subject_binding")
        try:
            self.catalog.validate_binding_registration(binding)
        except GovernanceAuthorityError as exc:
            checks["REPOSITORY_BINDING_MATCHES"] = EligibilityCheck("REPOSITORY_BINDING_MATCHES", CheckStatus.FAIL, exc.code)
            status = EligibilityStatus.TOOL_CAPABILITY_GAP if exc.code.startswith("UNSUPPORTED_") or exc.code.startswith("UNREGISTERED_") else EligibilityStatus.MALFORMED_AUTHORITY_RECORD
            return self._result(authority_id=authority_id,source_ref=source_ref,status=status,lifecycle_state=lifecycle_state,checks=checks,diagnostics=(exc.code,))

        authority_repository = RepositoryBinding.from_mapping(_mapping(binding["repository_binding"], code="AUTHORITY_BINDING_INVALID", label="repository_binding"))
        if authority_repository != solve_subject.repository_binding:
            checks["REPOSITORY_BINDING_MATCHES"] = EligibilityCheck("REPOSITORY_BINDING_MATCHES",CheckStatus.FAIL,"REPOSITORY_BINDING_NOT_APPLICABLE")
            return self._result(authority_id=authority_id,source_ref=source_ref,status=EligibilityStatus.CURRENTLY_INELIGIBLE,lifecycle_state=lifecycle_state,subject_match=SubjectMatchKind.NO_MATCH,checks=checks,diagnostics=("REPOSITORY_BINDING_NOT_APPLICABLE",))
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
        return tuple(self.evaluate_topic(topic_id, solve_subject, source_revision=source_revision) for topic_id in sorted(self.catalog.current_routes))

    def project_topic(self, topic_id: str, solve_subject: SolveSubject, *, source_revision: str) -> ProjectAuthorityRecord | None:
        result = self.evaluate_topic(topic_id, solve_subject, source_revision=source_revision)
        if not result.currently_eligible:
            return None
        source_ref, _, record = self.catalog.load_current_record(topic_id)
        project_record = ProjectAuthorityRecord(
            authority_id=result.authority_id,
            authority_contract=deepcopy(record["authority_contract"]),
            authority_semantics=deepcopy(record["authority_semantics"]),
            authority_role=deepcopy(record["authority_role"]),
            authority_provenance={
                "source_type":"MACHINE_ADR",
                "source_ref":source_ref,
                "source_revision":source_revision,
                "source_digest":_stable_digest(record),
            },
            subject_binding=deepcopy(record["subject_binding"]),
        )
        schema=deepcopy(self.catalog.project_authority_record_schema)
        schema["properties"]["authority_role"]=deepcopy(self.catalog.role_schema)
        schema["properties"]["subject_binding"]=deepcopy(self.catalog.subject_schema)
        Draft202012Validator(schema).validate(project_record.as_dict())
        return project_record

    def project_current_authorities(self, solve_subject: SolveSubject, *, source_revision: str) -> tuple[ProjectAuthorityRecord, ...]:
        records=[]
        for topic_id in sorted(self.catalog.current_routes):
            projected=self.project_topic(topic_id,solve_subject,source_revision=source_revision)
            if projected is not None:
                records.append(projected)
        return tuple(records)
