from __future__ import annotations

from pathlib import Path
from typing import Mapping

import yaml

from ptsip.governance.authority import AuthorityCatalog
from ptsip.governance.model import AuthorizationState, AuthorizationTransitionResult, GovernanceAuthorityError


class DeveloperAuthorizationTransitionEvaluator:
    """Evaluate PTSIP Project Owner-preauthorized MPD transition rules."""

    REGISTRY = "developer/policy/registries/authorization-transition-registry.yaml"

    def __init__(self, repository_root: str | Path) -> None:
        self.root=Path(repository_root).resolve()
        path=(self.root/self.REGISTRY).resolve()
        if self.root not in path.parents or not path.is_file():
            raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_REGISTRY_MISSING",f"missing developer authorization registry: {self.REGISTRY}",self.REGISTRY)
        value=yaml.safe_load(path.read_text(encoding="utf-8"))
        if not isinstance(value,dict):
            raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_REGISTRY_INVALID","developer authorization registry must be a mapping.",value)
        self.registry=value

    def derive_project_authority_runtime_readiness(self) -> dict[str,bool]:
        catalog=AuthorityCatalog(self.root)
        role=catalog.role_registry.get("effect_vocabulary",{})
        subject=catalog.subject_registry.get("subject_identity_schemes",{})
        provenance=self.registry.get("authorization_provenance")
        return {
            "AUTHORITY_SCHEMA_REGISTRY_VALID": bool(catalog.authority_schema_registry.get("entries")),
            "AUTHORITY_ROLE_REGISTRY_VALID": bool(catalog.role_registry.get("policy_roles")),
            "AUTHORITY_SUBJECT_REGISTRY_VALID": "SUPPORT_POLICY_ID" in subject,
            "CURRENT_SUPPORT_POLICY_CORPUS_VALID": len(catalog.validate_current_corpus()) == 21,
            "ROLE_EFFECT_VOCABULARY_VALID": isinstance(role,Mapping) and role.get("count")==len(role.get("tokens",[])),
            "SUPPORT_POLICY_SUBJECT_CONTRACT_VALID": catalog.subject_registry.get("repository_binding_policy")=="SOLVE_SUBJECT_PROVIDED_NO_BUILTIN_CURRENT_REPOSITORY",
            "PROJECT_AUTHORITY_RUNTIME_OWNER_PREAUTHORIZED": isinstance(provenance,Mapping) and provenance.get("authority")=="PROJECT_OWNER" and provenance.get("type")=="GIT_COMMIT" and isinstance(provenance.get("revision"),str) and len(provenance["revision"])==40,
        }

    def evaluate(self,scope:str,readiness:Mapping[str,bool])->AuthorizationTransitionResult:
        rules=self.registry.get("rules")
        if not isinstance(rules,Mapping):
            raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_REGISTRY_INVALID","rules must be a mapping.",rules)
        for rule_id,rule_value in rules.items():
            if not isinstance(rule_value,Mapping) or scope not in rule_value.get("target_scopes",[]):
                continue
            required=rule_value.get("required_predicates",{})
            if not isinstance(required,Mapping):
                raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_REGISTRY_INVALID",f"rule {rule_id} required_predicates must be a mapping.",required)
            failed=tuple(predicate for predicate,expected in required.items() if readiness.get(predicate) is not expected)
            state_value=rule_value["when_all_true"] if not failed else rule_value["otherwise"]
            return AuthorizationTransitionResult(scope=scope,state=AuthorizationState(state_value),rule_id=str(rule_id),failed_predicates=failed,blockers=tuple(f"PREDICATE_NOT_SATISFIED:{item}" for item in failed))
        holds=self.registry.get("held_scopes",{})
        if isinstance(holds,Mapping) and scope in holds:
            hold=holds[scope]
            return AuthorizationTransitionResult(scope=scope,state=AuthorizationState(hold.get("state","HOLD_NOT_AUTHORIZED")),rule_id=None,blockers=tuple(str(item) for item in hold.get("blockers",[])))
        return AuthorizationTransitionResult(scope=scope,state=AuthorizationState.HOLD_NOT_AUTHORIZED,rule_id=None,blockers=("NO_OWNER_PREAUTHORIZED_TRANSITION_RULE",))

    def evaluate_current_project_authority_runtime(self)->tuple[AuthorizationTransitionResult,...]:
        readiness=self.derive_project_authority_runtime_readiness()
        scopes=("PROJECT_AUTHORITY_ELIGIBILITY_RUNTIME","PROJECT_AUTHORITY_PROJECTION_RUNTIME","PROJECT_AUTHORITY_RECORD_MATERIALIZATION","AUTHORIZATION_READINESS_TRANSITION_ENGINE")
        return tuple(self.evaluate(scope,readiness) for scope in scopes)
