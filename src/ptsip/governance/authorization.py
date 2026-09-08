from __future__ import annotations

import json
from pathlib import Path
from typing import Mapping

import yaml
from jsonschema import Draft202012Validator

from .authority import AuthorityCatalog
from .model import AuthorizationState, AuthorizationTransitionResult, GovernanceAuthorityError


class AuthorizationTransitionEvaluator:
    """Evaluate only Project Owner-preauthorized machine transition rules."""

    REGISTRY = "decisions/AUTHORIZATION-TRANSITION-REGISTRY.yaml"
    SCHEMA = "schemas/ptsip-authorization-transition.schema.json"

    def __init__(self, repository_root: str | Path) -> None:
        self.root = Path(repository_root).resolve()
        self.registry = self._load_yaml(self.REGISTRY)
        self.schema = self._load_json(self.SCHEMA)
        Draft202012Validator.check_schema(self.schema)
        Draft202012Validator(self.schema).validate(self.registry)

    def _path(self, relative: str) -> Path:
        path = Path(relative)
        if path.is_absolute() or ".." in path.parts:
            raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_PATH_INVALID", "authorization transition paths must be repository-relative.", relative)
        resolved=(self.root/path).resolve()
        if self.root not in resolved.parents and resolved != self.root:
            raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_PATH_INVALID", "authorization transition path escaped repository root.", relative)
        return resolved

    def _load_yaml(self, relative: str) -> dict[str, object]:
        path=self._path(relative)
        if not path.is_file():
            raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_REGISTRY_MISSING",f"missing authorization transition registry: {relative}",relative)
        value=yaml.safe_load(path.read_text(encoding="utf-8"))
        if not isinstance(value,dict):
            raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_REGISTRY_INVALID","authorization transition registry must be a mapping.",value)
        return value

    def _load_json(self, relative: str) -> dict[str, object]:
        path=self._path(relative)
        if not path.is_file():
            raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_SCHEMA_MISSING",f"missing authorization transition schema: {relative}",relative)
        value=json.loads(path.read_text(encoding="utf-8"))
        if not isinstance(value,dict):
            raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_SCHEMA_INVALID","authorization transition schema must be a mapping.",value)
        return value

    def derive_project_authority_runtime_readiness(self) -> dict[str, bool]:
        readiness={
            "AUTHORITY_SCHEMA_REGISTRY_VALID":False,
            "AUTHORITY_ROLE_REGISTRY_VALID":False,
            "AUTHORITY_SUBJECT_REGISTRY_VALID":False,
            "CURRENT_ADR_CORPUS_VALID":False,
            "ROLE_EFFECT_VOCABULARY_FROZEN":False,
            "SUBJECT_BINDING_CONTRACT_FROZEN":False,
            "PROJECT_AUTHORITY_RUNTIME_OWNER_PREAUTHORIZED":False,
        }
        try:
            catalog=AuthorityCatalog(self.root)
            readiness["AUTHORITY_SCHEMA_REGISTRY_VALID"]=True
            readiness["AUTHORITY_ROLE_REGISTRY_VALID"]=True
            readiness["AUTHORITY_SUBJECT_REGISTRY_VALID"]=True
            readiness["CURRENT_ADR_CORPUS_VALID"]=len(catalog.validate_current_corpus())==23
            vocabulary=catalog.role_registry["effect_vocabulary"]
            readiness["ROLE_EFFECT_VOCABULARY_FROZEN"]=(
                vocabulary.get("frozen") is True
                and vocabulary.get("count")==99
                and len(vocabulary.get("tokens",[]))==99
            )
            frozen_by=catalog.subject_registry.get("frozen_by")
            readiness["SUBJECT_BINDING_CONTRACT_FROZEN"]=(
                isinstance(frozen_by,Mapping)
                and frozen_by.get("type")=="GIT_COMMIT"
                and isinstance(frozen_by.get("revision"),str)
                and len(frozen_by["revision"])==40
            )
        except (GovernanceAuthorityError,KeyError,TypeError,ValueError):
            pass
        provenance=self.registry.get("authorization_provenance")
        readiness["PROJECT_AUTHORITY_RUNTIME_OWNER_PREAUTHORIZED"]=(
            isinstance(provenance,Mapping)
            and provenance.get("authority")=="PROJECT_OWNER"
            and provenance.get("type")=="GIT_COMMIT"
            and isinstance(provenance.get("revision"),str)
            and len(provenance["revision"])==40
        )
        return readiness

    def evaluate(self, scope: str, readiness: Mapping[str, bool]) -> AuthorizationTransitionResult:
        rules=self.registry.get("rules")
        if not isinstance(rules,Mapping):
            raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_REGISTRY_INVALID","rules must be a mapping.",rules)
        for rule_id,rule_value in rules.items():
            if not isinstance(rule_value,Mapping):
                raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_REGISTRY_INVALID",f"rule {rule_id} must be a mapping.",rule_value)
            if scope not in rule_value.get("target_scopes",[]):
                continue
            required=rule_value.get("required_predicates",{})
            if not isinstance(required,Mapping):
                raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_REGISTRY_INVALID",f"rule {rule_id} required_predicates must be a mapping.",required)
            failed=tuple(predicate for predicate,expected in required.items() if readiness.get(predicate) is not expected)
            state_value=rule_value["when_all_true"] if not failed else rule_value["otherwise"]
            return AuthorizationTransitionResult(
                scope=scope,state=AuthorizationState(state_value),rule_id=str(rule_id),
                failed_predicates=failed,
                blockers=tuple(f"PREDICATE_NOT_SATISFIED:{item}" for item in failed),
            )
        holds=self.registry.get("held_scopes",{})
        if isinstance(holds,Mapping) and scope in holds:
            hold=holds[scope]
            if not isinstance(hold,Mapping):
                raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_REGISTRY_INVALID",f"held scope {scope} must be a mapping.",hold)
            return AuthorizationTransitionResult(
                scope=scope,state=AuthorizationState(hold.get("state","HOLD_NOT_AUTHORIZED")),rule_id=None,
                blockers=tuple(str(item) for item in hold.get("blockers",[])),
            )
        return AuthorizationTransitionResult(
            scope=scope,state=AuthorizationState.HOLD_NOT_AUTHORIZED,rule_id=None,
            blockers=("NO_OWNER_PREAUTHORIZED_TRANSITION_RULE",),
        )

    def evaluate_current_project_authority_runtime(self) -> tuple[AuthorizationTransitionResult, ...]:
        readiness=self.derive_project_authority_runtime_readiness()
        scopes=(
            "PROJECT_AUTHORITY_ELIGIBILITY_RUNTIME",
            "PROJECT_AUTHORITY_PROJECTION_RUNTIME",
            "PROJECT_AUTHORITY_RECORD_MATERIALIZATION",
            "AUTHORIZATION_READINESS_TRANSITION_ENGINE",
        )
        return tuple(self.evaluate(scope,readiness) for scope in scopes)
