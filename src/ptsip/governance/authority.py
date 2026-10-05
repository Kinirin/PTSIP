from __future__ import annotations

import hashlib
import json
import re
from copy import deepcopy
from pathlib import Path
from typing import Callable, Mapping
from functools import lru_cache

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
from .support_assets import resolve_support_asset, resolve_support_asset_layout


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

    INDEX = "index.yaml"
    AUTHORITY_SCHEMA_REGISTRY = "ptsip-support-authority-schema-registry.yaml"
    AUTHORITY_ROLE_REGISTRY = "ptsip-support-authority-role-registry.yaml"
    AUTHORITY_SUBJECT_REGISTRY = "ptsip-support-authority-subject-registry.yaml"

    SUPPORT_POLICY_SCHEMA = "ptsip-support-feature-policy.schema.json"
    SUPPORT_ROOT_FAMILY_POLICY_SCHEMA = "ptsip-support-root-family-policy.schema.json"
    SUPPORT_INDEX_SCHEMA = "ptsip-support-feature-policy-index.schema.json"
    AUTHORITY_SEMANTICS_SCHEMA = "ptsip-support-authority-semantics.schema.json"
    AUTHORITY_ROLE_SCHEMA = "ptsip-support-authority-role.schema.json"
    AUTHORITY_SUBJECT_SCHEMA = "ptsip-support-subject-binding.schema.json"
    PROJECT_AUTHORITY_RECORD_SCHEMA = "ptsip-support-project-authority-record.schema.json"
    ELIGIBILITY_RESULT_SCHEMA = "ptsip-support-authority-eligibility-result.schema.json"

    def __init__(self, repository_root: str | Path) -> None:
        self.root = Path(repository_root).resolve()
        self.assets = resolve_support_asset_layout(self.root)
        self.index = self._load_yaml("policy", self.INDEX)
        self.authority_schema_registry = self._load_yaml("registries", self.AUTHORITY_SCHEMA_REGISTRY)
        self.role_registry = self._load_yaml("registries", self.AUTHORITY_ROLE_REGISTRY)
        self.subject_registry = self._load_yaml("registries", self.AUTHORITY_SUBJECT_REGISTRY)
        self.policy_schema = self._load_json("schemas", self.SUPPORT_POLICY_SCHEMA)
        self.root_family_policy_schema = self._load_json("schemas", self.SUPPORT_ROOT_FAMILY_POLICY_SCHEMA)
        self.index_schema = self._load_json("schemas", self.SUPPORT_INDEX_SCHEMA)
        self.semantics_schema = self._load_json("schemas", self.AUTHORITY_SEMANTICS_SCHEMA)
        self.role_schema = self._load_json("schemas", self.AUTHORITY_ROLE_SCHEMA)
        self.subject_schema = self._load_json("schemas", self.AUTHORITY_SUBJECT_SCHEMA)
        self.project_authority_record_schema = self._load_json("schemas", self.PROJECT_AUTHORITY_RECORD_SCHEMA)
        self.eligibility_result_schema = self._load_json("schemas", self.ELIGIBILITY_RESULT_SCHEMA)
        self._validate_catalog_assets()

    def _load_yaml(self, category: str, relative: str) -> dict[str, object]:
        path = resolve_support_asset(self.assets, category, relative)
        if category == "policy" and relative != self.INDEX:
            try:
                policy_root = resolve_support_asset(self.assets, "policy", self.INDEX).parent
                registry = migration_registry(policy_root, "PTSIP_SUPPORT_FEATURE", registry_path=self.assets.registries / "root-family-migration.json", copy_result=False)
                return read_policy(policy_root, relative, "PTSIP_SUPPORT_FEATURE", registry=registry, module_path=self.assets.registries / "root-family-projection.module.json")
            except (OSError, ValueError, KeyError) as exc:
                raise GovernanceAuthorityError("ROOT_FAMILY_PROJECTION_INVALID", str(exc), relative) from exc
        if not path.is_file():
            raise GovernanceAuthorityError("GOVERNANCE_ASSET_MISSING", f"missing support governance asset: {relative}", relative)
        value = yaml.safe_load(path.read_text(encoding="utf-8"))
        if not isinstance(value, dict):
            raise GovernanceAuthorityError("GOVERNANCE_ASSET_INVALID", f"support governance asset must contain a mapping: {relative}", value)
        return value

    def _load_json(self, category: str, relative: str) -> dict[str, object]:
        path = resolve_support_asset(self.assets, category, relative)
        if not path.is_file():
            raise GovernanceAuthorityError("GOVERNANCE_SCHEMA_MISSING", f"missing support governance schema: {relative}", relative)
        value = json.loads(path.read_text(encoding="utf-8"))
        if not isinstance(value, dict):
            raise GovernanceAuthorityError("GOVERNANCE_SCHEMA_INVALID", f"support governance schema must contain a mapping: {relative}", value)
        return value

    def _validate_catalog_assets(self) -> None:
        for schema in (
            self.policy_schema, self.root_family_policy_schema, self.index_schema, self.semantics_schema,
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
        record = self._load_yaml("policy", path)
        source_ref = f"src/policy/{path}"
        return source_ref, route, record

    def canonical_sources_for_policy(self, policy_id: str) -> list[dict[str, str]]:
        registry = migration_registry(self.assets.policy, "PTSIP_SUPPORT_FEATURE", registry_path=self.assets.registries / "root-family-migration.json")
        source = source_route(registry, policy_id=policy_id)
        if source is None:
            return []
        return [{"policy_id": unit["policy_id"], "path": f"src/policy/{unit['policy_path']}", "section": unit["section"]} for unit in source["units"]]

    def iter_current_records(self) -> tuple[tuple[str, str, Mapping[str, object], Mapping[str, object]], ...]:
        items = []
        for policy_id in self.current_routes:
            path, route, record = self.load_current_record(policy_id)
            items.append((policy_id, path, route, record))
        return tuple(items)

    def resolve_family(self, family: str) -> tuple[tuple[str, Mapping[str, object], Mapping[str, object]], ...]:
        """Resolve only this shipped Support plane's explicitly materialized Family."""
        policy_root = resolve_support_asset(self.assets, "policy", self.INDEX).parent
        registry = migration_registry(policy_root, "PTSIP_SUPPORT_FEATURE", registry_path=self.assets.registries / "root-family-migration.json")
        rows = [{"policy_id": r["id"]} for r in self.index["policies"] if r["id"].startswith(f"SFP-{family}-") and r["path"] == f"{family}/{r['id']}.yaml"]
        if not rows:
            raise GovernanceAuthorityError("SUPPORT_ROOT_FAMILY_UNRESOLVED", "no exact Support Family materialization is registered", family)
        result = []
        for row in rows:
            source_ref, route, record = self.load_current_record(row["policy_id"])
            self._validate_current_selection(row["policy_id"], source_ref, route, record)
            if record.get("policy_class") != "PTSIP_SUPPORT_FEATURE" or record.get("responsibility_family") != family or record["policy"]["status"] != route["status"]:
                raise GovernanceAuthorityError("SUPPORT_ROOT_FAMILY_METADATA_MISMATCH", "Support Family identity/status mismatch", row)
            self._validate_contract_and_semantics(record)
            result.append((source_ref, route, record))
        return tuple(result)

    def _validate_current_selection(self, policy_id: str, source_ref: str, route: Mapping[str, object], record: Mapping[str, object]) -> None:
        policy = _mapping(record.get("policy"), code="MALFORMED_AUTHORITY_RECORD", label="policy")
        selected_path = source_ref.removeprefix("src/policy/")
        if (
            policy.get("id") != policy_id
            or route.get("id") != policy_id
            or route.get("path") != selected_path
        ):
            raise GovernanceAuthorityError("CURRENT_SUPPORT_POLICY_SELECTION_MISMATCH", "support policy record does not exactly match support-policy-index.", {"policy_id":policy_id,"source_ref":source_ref})

    def _policy_schema_for_record(self, record: Mapping[str, object]) -> Mapping[str, object]:
        schema_version = record.get("schema_version")
        if schema_version == "ptsip-support-root-family-policy/v1":
            return self.root_family_policy_schema
        if schema_version == "ptsip-support-feature-policy/v1":
            return self.policy_schema
        raise GovernanceAuthorityError(
            "UNSUPPORTED_SUPPORT_POLICY_SCHEMA",
            "support policy schema_version is not registered.",
            schema_version,
        )

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
        try:
            validate_migration(resolve_support_asset(self.assets, "policy", self.INDEX).parent, "PTSIP_SUPPORT_FEATURE", registry_path=self.assets.registries / "root-family-migration.json", schema_path=self.assets.schemas / "root-family-migration.schema.json")
        except (OSError, ValueError, KeyError, ValidationError) as exc:
            raise GovernanceAuthorityError("ROOT_FAMILY_MIGRATION_INVALID", str(exc), self.INDEX) from exc
        validated = []
        for policy_id, source_ref, route, record in self.iter_current_records():
            self._validate_current_selection(policy_id, source_ref, route, record)
            Draft202012Validator(self._policy_schema_for_record(record)).validate(record)
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
        provenance = {"source_type": "SUPPORT_FEATURE_POLICY", "source_ref": source_ref, "source_revision": source_revision, "source_digest": _stable_digest(record)}
        canonical_sources = self.catalog.canonical_sources_for_policy(policy_id)
        if canonical_sources:
            provenance.update(source_role="REGISTERED_ROOT_FAMILY_PROJECTION", canonical_sources=canonical_sources)
        project_record = ProjectAuthorityRecord(
            authority_id=result.authority_id,
            authority_contract=deepcopy(record["authority_contract"]),
            authority_semantics=deepcopy(record["authority_semantics"]),
            authority_role=deepcopy(role),
            authority_provenance=provenance,
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


REGISTRY = "registries/root-family-migration.json"
SAFE_LOADER = getattr(yaml, "CSafeLoader", yaml.SafeLoader)


@lru_cache(maxsize=128)
def _parse_yaml(data: bytes) -> dict:
    value = yaml.load(data.decode("utf-8"), Loader=SAFE_LOADER)
    if not isinstance(value, dict):
        raise ValueError("ROOT_MIGRATION_RECORD_NOT_MAPPING")
    return value


def load_yaml_mapping(path: Path) -> dict:
    # Cache parsing by exact bytes, never by path, timestamp, or inferred state.
    # Every read observes current bytes, and callers receive an independent mapping.
    return deepcopy(_parse_yaml(path.read_bytes()))


@lru_cache(maxsize=32)
def _parse_json(data: bytes) -> dict:
    return json.loads(data.decode("utf-8"))


def source_digest(data: bytes) -> str:
    """Hash repository text independently of Git's checkout newline conversion."""
    return hashlib.sha256(data.replace(b"\r\n", b"\n")).hexdigest()


def record_digest(value: object) -> str:
    data = json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))
    return hashlib.sha256(data.encode("utf-8")).hexdigest()


def migration_registry(policy_root: Path, policy_class: str, *, registry_path: Path | None = None, copy_result: bool = True) -> dict:
    path = registry_path if registry_path is not None else policy_root / REGISTRY
    if not path.is_file():
        index_path = policy_root / "index.yaml"
        if index_path.is_file() and load_yaml_mapping(index_path).get("migration_registry_ref") == REGISTRY:
            raise ValueError("ROOT_MIGRATION_ADMITTED_REGISTRY_MISSING")
        return {}
    index = load_yaml_mapping(policy_root / "index.yaml")
    if index.get("migration_registry_ref") != REGISTRY:
        raise ValueError("ROOT_MIGRATION_REGISTRY_NOT_ADMITTED")
    value = _parse_json(path.read_bytes())
    if value.get("schema_version") != "ptsip-root-family-migration/v1" or value.get("policy_class") != policy_class:
        raise ValueError("ROOT_MIGRATION_CLASS_OR_VERSION_MISMATCH")
    sources = value.get("sources")
    if not isinstance(sources, list) or len({s["source_policy_id"] for s in sources}) != len(sources):
        raise ValueError("ROOT_MIGRATION_DUPLICATE_OR_INVALID_SOURCE")
    return deepcopy(value) if copy_result else value


def registered_source_path(policy_root: Path, policy_class: str, policy_id: str, expected: str) -> str:
    source = source_route(migration_registry(policy_root, policy_class, copy_result=False), policy_id=policy_id)
    if source is None:
        return expected
    if source["original_path"] != expected:
        raise ValueError("ROOT_MIGRATION_SOURCE_PATH_MISMATCH")
    return source["archive_path"]


def source_route(registry: dict, *, policy_id: str | None = None, path: str | None = None) -> dict | None:
    matches = [s for s in registry.get("sources", []) if
               (policy_id is not None and s["source_policy_id"] == policy_id) or
               (path is not None and path in {s["original_path"], s["archive_path"]})]
    if len(matches) > 1:
        raise ValueError("ROOT_MIGRATION_AMBIGUOUS_SOURCE")
    return matches[0] if matches else None


def safe_policy_path(policy_root: Path, relative: str) -> Path:
    if Path(relative).is_absolute():
        raise ValueError("ROOT_MIGRATION_ABSOLUTE_PATH")
    path = (policy_root / relative).resolve()
    if not path.is_relative_to(policy_root.resolve()):
        raise ValueError("ROOT_MIGRATION_PATH_ESCAPE")
    return path



MODULE = "registries/root-family-projection.module.json"

@lru_cache(maxsize=16)
def _checked_module(data: bytes, schema_data: bytes) -> dict:
    value = _parse_json(data)
    try:
        Draft202012Validator(_parse_json(schema_data)).validate(value)
    except ValidationError as exc:
        raise ValueError("ROOT_MODULE_PROGRAM_SCHEMA_INVALID") from exc
    return value


def projection_module(policy_root: Path, policy_class: str, *, module_path: Path | None = None) -> dict:
    """Load only the class-local, exact-owner-admitted neutral machine program."""
    index = load_yaml_mapping(policy_root / "index.yaml")
    if index.get("projection_module_ref") != MODULE:
        raise ValueError("ROOT_MODULE_NOT_ADMITTED")
    path = module_path if module_path is not None else policy_root / MODULE
    if not path.is_file():
        raise ValueError("ROOT_MODULE_ADMITTED_PROGRAM_MISSING")
    schema_path = path.parent.parent / "schemas/root-family-projection-module.schema.json"
    data = path.read_bytes()
    module = _checked_module(data, schema_path.read_bytes())
    if module["policy_class"] != policy_class:
        raise ValueError("ROOT_MODULE_CLASS_MISMATCH")
    rows = [r for r in index["policies"] if r["id"] == module["owner_policy_ref"]]
    if len(rows) != 1 or rows[0]["status"] != "ACTIVE":
        raise ValueError("ROOT_MODULE_OWNER_UNRESOLVED")
    relative = rows[0]["path"].removeprefix("developer/policy/")
    owner = load_yaml_mapping(safe_policy_path(policy_root, relative))
    rule = owner.get(module["parameters"]["value_field"], {}).get(module["owner_section"], {})
    expected = "developer/policy/" + MODULE if policy_class == "PTSIP_DEVELOPER_POLICY" else MODULE
    if (owner.get("policy_class") != policy_class or owner.get("policy", {}).get("id") != module["owner_policy_ref"]
        or owner["policy"].get("status") != "ACTIVE" or rule.get("module_ref") != expected
        or rule.get("module_id") != module["module_id"] or rule.get("module_sha256") != source_digest(data)
        or rule.get("module_digest_format") != module["module_digest_format"]
        or rule.get("normative_program_language") != module["program_language"]):
        raise ValueError("ROOT_MODULE_OWNER_BINDING_MISMATCH")
    return deepcopy(module)


def project_source(registry: dict, source: dict, read_record: Callable[[str], dict], *, module: dict) -> dict:
    """Interpret the registered neutral program; no language-specific Family rules."""
    env = {"registry": registry, "source": source, "parameters": module["parameters"]}
    records: dict[str, dict] = {}

    def expression(node: dict):
        operator = node["op"]
        if operator == "literal":
            return deepcopy(node["value"])
        if operator == "ref":
            value = env
            for token in node["path"][1:].split("/"):
                value = value.get(token) if isinstance(value, dict) else None
            return value
        args = [expression(a) for a in node["args"]]
        if operator == "eq": return record_digest(args[0]) == record_digest(args[1])
        if operator == "all": return all(a is True for a in args)
        if operator == "not": return args[0] is not True
        if operator == "contains": return args[1] in args[0]
        if operator == "get": return args[0].get(args[1]) if isinstance(args[0], dict) else None
        if operator == "concat": return "".join(args)
        if operator == "starts_with": return isinstance(args[0], str) and args[0].startswith(args[1])
        if operator == "pluck": return [v.get(args[1]) for v in (args[0] or [])]
        if operator == "has_key": return isinstance(args[0], dict) and args[1] in args[0]
        if operator == "pointer_first": return args[0][1:].split("/")[0].replace("~1", "/").replace("~0", "~")
        if operator == "pointer_nonoverlap":
            pointer, seen = args
            return isinstance(pointer, str) and pointer.startswith("/") and not any(pointer == p or pointer.startswith(p + "/") or p.startswith(pointer + "/") for p in seen)
        if operator == "read_record":
            if args[0] not in records: records[args[0]] = read_record(args[0])
            return records[args[0]]
        if operator == "record_digest": return record_digest(args[0])
        raise ValueError("ROOT_MODULE_UNKNOWN_OPERATOR")

    def execute(steps: list):
        for step in steps:
            operator = step["op"]
            if operator == "let": env[step["name"]] = deepcopy(expression(step["value"]))
            elif operator == "assert":
                if not expression(step["predicate"]): raise ValueError(step["error"])
            elif operator == "append": env[step["name"]].append(expression(step["value"]))
            elif operator == "foreach":
                for item in expression(step["items"]):
                    env[step["name"]] = item
                    execute(step["body"])
            elif operator == "set_pointer":
                tokens = [s.replace("~1", "/").replace("~0", "~") for s in expression(step["pointer"])[1:].split("/")]
                target = env[step["name"]]
                for token in tokens[:-1]: target = target.setdefault(token, {})
                target[tokens[-1]] = deepcopy(expression(step["value"]))
            elif operator == "return": return deepcopy(expression(step["value"]))
            else: raise ValueError("ROOT_MODULE_UNKNOWN_INSTRUCTION")
        return None

    try:
        result = execute(module["program"])
    except (TypeError, KeyError, IndexError, AttributeError) as exc:
        raise ValueError("ROOT_MODULE_PROGRAM_INVALID") from exc
    if not isinstance(result, dict): raise ValueError("ROOT_MODULE_RESULT_INVALID")
    return result

def read_policy(policy_root: Path, relative: str, policy_class: str, *, registry: dict | None = None, module_path: Path | None = None) -> dict:
    registry = migration_registry(policy_root, policy_class, copy_result=False) if registry is None else registry
    if registry and registry.get("policy_class") != policy_class:
        raise ValueError("ROOT_MIGRATION_CLASS_OR_VERSION_MISMATCH")
    source = source_route(registry, path=relative)

    def raw(path: str) -> dict:
        return _parse_yaml(safe_policy_path(policy_root, path).read_bytes())

    return project_source(registry, source, raw, module=projection_module(policy_root, policy_class, module_path=module_path)) if source else deepcopy(raw(relative))


def validate_migration(policy_root: Path, policy_class: str, *, registry_path: Path | None = None, schema_path: Path | None = None) -> tuple[str, ...]:
    """Verify admission, immutable sources, complete unit coverage and class isolation."""
    from jsonschema import Draft202012Validator

    registry = migration_registry(policy_root, policy_class, registry_path=registry_path)
    if not registry:
        return ()
    module_path = registry_path.parent / "root-family-projection.module.json" if registry_path is not None else None
    module = projection_module(policy_root, policy_class, module_path=module_path)
    schema = json.loads((schema_path if schema_path is not None else policy_root / "schemas/root-family-migration.schema.json").read_text(encoding="utf-8"))
    Draft202012Validator(schema).validate(registry)
    index = load_yaml_mapping(policy_root / "index.yaml")
    entries = {e["id"]: e for e in index["policies"]}
    if len(entries) != len(index["policies"]):
        raise ValueError("ROOT_MIGRATION_DUPLICATE_INDEX_ID")
    used: set[tuple[str, str]] = set()
    validated = []

    def raw(relative: str) -> dict:
        return _parse_yaml(safe_policy_path(policy_root, relative).read_bytes())

    developer = policy_class == "PTSIP_DEVELOPER_POLICY"
    field = "rules" if developer else "authority_semantics"
    for source in registry["sources"]:
        if safe_policy_path(policy_root, source["original_path"]).exists():
            raise ValueError("ROOT_MIGRATION_SOURCE_REINTRODUCED")
        entry = entries[source["source_policy_id"]]
        indexed = entry["path"].removeprefix("developer/policy/") if developer else entry["path"]
        if indexed != source["archive_path"] or entry.get("authority_role") != "MIGRATION_SOURCE" or entry["status"] != source["source_status"]:
            raise ValueError("ROOT_MIGRATION_SOURCE_CATALOG_MISMATCH")
        original = safe_policy_path(policy_root, source["archive_path"])
        if source_digest(original.read_bytes()) != source["source_sha256"]:
            raise ValueError("ROOT_MIGRATION_ARCHIVED_SOURCE_CHANGED")
        if record_digest(raw(source["archive_path"])) != source["record_sha256"]:
            raise ValueError("ROOT_MIGRATION_SOURCE_RECORD_CHANGED")
        project_source(registry, source, raw, module=module)
        for unit in source["units"]:
            key = (unit["policy_id"], unit["section"])
            if key in used:
                raise ValueError("ROOT_MIGRATION_UNIT_MULTIPLE_OWNERS")
            used.add(key)
        validated.append(source["source_policy_id"])
    families = set()
    declared: set[tuple[str, str]] = set()
    materialized_ids = [m["policy_id"] for m in registry["materializations"]]
    if len(set(materialized_ids)) != len(materialized_ids):
        raise ValueError("ROOT_MIGRATION_DUPLICATE_MATERIALIZATION")
    source_ids = {s["source_policy_id"] for s in registry["sources"]}
    selected_ids = {pid for pid, entry in entries.items() if entry.get("authority_role") in {"MIGRATION_SOURCE", "CANONICAL_AUTHORITY"}}
    if selected_ids != source_ids | set(materialized_ids):
        raise ValueError("ROOT_MIGRATION_CATALOG_COVERAGE_MISMATCH")
    for materialization in registry["materializations"]:
        pid = materialization["policy_id"]
        entry = entries[pid]
        indexed = entry["path"].removeprefix("developer/policy/") if developer else entry["path"]
        record = raw(materialization["path"])
        if indexed != materialization["path"] or entry.get("authority_role") != "CANONICAL_AUTHORITY" or entry["status"] != materialization["source_status"]:
            raise ValueError("ROOT_MIGRATION_OWNER_CATALOG_MISMATCH")
        if record["policy_class"] != policy_class or record["responsibility_family"] != materialization["family"] or record["policy"]["id"] != pid or record["policy"]["status"] != entry["status"]:
            raise ValueError("ROOT_MIGRATION_MATERIALIZATION_IDENTITY_MISMATCH")
        keys = {key for key in record[field] if key.startswith("unit_")}
        if len(keys) != materialization["unit_count"]:
            raise ValueError("ROOT_MIGRATION_UNIT_COUNT_MISMATCH")
        if materialization["definition_only"] and (keys or entry["status"] != "DRAFT"):
            raise ValueError("ROOT_MIGRATION_DEFINITION_ONLY_PROMOTED")
        declared.update((pid, key) for key in keys)
        families.add(materialization["family"])
    if declared != used:
        raise ValueError("ROOT_MIGRATION_UNIT_COVERAGE_MISMATCH")
    expected_families = set(schema["properties"]["materializations"]["items"]["properties"]["family"]["enum"])
    if families != expected_families:
        raise ValueError("ROOT_MIGRATION_FAMILY_COVERAGE_MISMATCH")
    return tuple(validated)
