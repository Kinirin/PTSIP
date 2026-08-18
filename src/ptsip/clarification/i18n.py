from __future__ import annotations

import locale
import os

SUPPORTED_LANGUAGES = ("en", "ko")

_TEXT = {
    "en": {
        "title": "[PTSIP clarification] Architecture decision for {component_id}",
        "intro": "PTSIP stopped before speculative classification because the repository does not declare enough architectural intent for this component.",
        "no_llm": "This clarification request is generated from fixed rules and templates. PTSIP does not call an LLM to create it.",
        "classification": "Which primary lifecycle owns this PTSIP component?",
        "purpose": "What is the primary purpose for which this component was created?",
        "shipped": "Is this component distributed or shipped with the product?",
        "runtime_required": "Is this component required by the product at runtime as implementation?",
        "lifecycle_owner": "Which lifecycle owns this component? This transitional compatibility fact must agree with classification.",
        "executable": "Does this component contain executable implementation rather than only a declarative/non-executable responsibility?",
        "reply": "Please reply using the following structured fields. Keep the invariant codes unchanged where applicable.",
        "required": "Clarification required",
        "none": "No clarification is required from the currently detected component candidates.",
    },
    "ko": {
        "title": "[PTSIP 확인 요청] {component_id} 아키텍처 결정",
        "intro": "PTSIP는 이 컴포넌트의 아키텍처 목적을 선언된 저장소 정보만으로 확인할 수 없어 추측성 분류를 중단했습니다.",
        "no_llm": "이 확인 요청은 고정된 규칙과 템플릿으로 생성됩니다. PTSIP는 질문을 만들기 위해 LLM을 호출하지 않습니다.",
        "classification": "이 PTSIP 컴포넌트의 primary lifecycle ownership은 무엇입니까?",
        "purpose": "이 컴포넌트를 만든 주된 목적은 무엇입니까?",
        "shipped": "이 컴포넌트는 제품과 함께 배포되거나 제품 패키지에 포함됩니까?",
        "runtime_required": "제품 런타임 구현으로 이 컴포넌트가 필요합니까?",
        "lifecycle_owner": "이 컴포넌트의 lifecycle owner는 무엇입니까? 이 과도기적 호환성 값은 classification과 일치해야 합니다.",
        "executable": "이 컴포넌트는 선언형/비실행 책임만이 아니라 실행 가능한 구현을 포함합니까?",
        "reply": "다음 구조화된 필드로 답변해 주세요. 해당되는 경우 영문 고정 코드는 변경하지 않는 것이 좋습니다.",
        "required": "확인 필요",
        "none": "현재 탐지된 컴포넌트 후보에는 추가 확인이 필요하지 않습니다.",
    },
}

_OPTIONS = {
    "classification": (
        ("PRODUCT", {"en": "Product lifecycle", "ko": "Product lifecycle"}),
        ("DEVELOPMENT_TOOLING", {"en": "Development tooling lifecycle", "ko": "개발 Tooling lifecycle"}),
        ("DELIVERY", {"en": "Release/distribution/deployment delivery lifecycle", "ko": "릴리스/배포 Delivery lifecycle"}),
        ("OPERATIONS", {"en": "Post-delivery operations lifecycle", "ko": "배포 이후 Operations lifecycle"}),
        ("NEUTRAL_CONTRACT", {"en": "Independent neutral non-executable contract", "ko": "독립 중립 비실행 계약"}),
    ),
    "shipped": (
        ("YES", {"en": "Yes", "ko": "예"}),
        ("NO", {"en": "No", "ko": "아니요"}),
    ),
    "runtime_required": (
        ("YES", {"en": "Yes", "ko": "예"}),
        ("NO", {"en": "No", "ko": "아니요"}),
    ),
    "lifecycle_owner": (
        ("PRODUCT", {"en": "Product lifecycle", "ko": "Product lifecycle"}),
        ("DEVELOPMENT_TOOLING", {"en": "Development tooling lifecycle", "ko": "개발 Tooling lifecycle"}),
        ("DELIVERY", {"en": "Delivery lifecycle", "ko": "Delivery lifecycle"}),
        ("OPERATIONS", {"en": "Operations lifecycle", "ko": "Operations lifecycle"}),
        ("INDEPENDENT", {"en": "Independently governed Neutral Contract", "ko": "독립 관리 Neutral Contract"}),
    ),
    "executable": (
        ("YES", {"en": "Executable implementation is present", "ko": "실행 가능한 구현이 있음"}),
        ("NO", {"en": "Non-executable/declarative responsibility", "ko": "비실행/선언형 책임"}),
    ),
}


def _normalize(value: str | None) -> str | None:
    if not value:
        return None
    lowered = value.strip().replace("_", "-").casefold()
    if lowered == "ko" or lowered.startswith("ko-"):
        return "ko"
    if lowered == "en" or lowered.startswith("en-"):
        return "en"
    return None


def resolve_language(explicit: str | None = None) -> str:
    if explicit is not None:
        normalized = _normalize(explicit)
        if normalized is None:
            raise ValueError("Supported languages are: en, ko")
        return normalized
    configured = _normalize(os.environ.get("PTSIP_LANG"))
    if configured:
        return configured
    try:
        system_locale = locale.getlocale()[0]
    except (ValueError, TypeError):
        system_locale = None
    return _normalize(system_locale) or "en"


def text(language: str, key: str, **values: str) -> str:
    return _TEXT[language][key].format(**values)


def question(field: str, language: str) -> dict[str, object]:
    payload: dict[str, object] = {
        "field": field,
        "type": "free_text" if field == "purpose" else "single_select",
        "prompt": text(language, field),
    }
    if field in _OPTIONS:
        payload["options"] = [
            {"value": code, "label": labels[language]} for code, labels in _OPTIONS[field]
        ]
    return payload
