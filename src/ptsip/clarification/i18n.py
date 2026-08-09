from __future__ import annotations

import locale
import os

SUPPORTED_LANGUAGES = ("en", "ko")

_TEXT = {
    "en": {
        "title": "[PTSIP clarification] Purpose of {component_id}",
        "intro": "PTSIP stopped before speculative classification because the repository does not declare enough architectural intent for this component.",
        "no_llm": "This clarification request is generated from fixed rules and templates. PTSIP does not call an LLM to create it.",
        "purpose": "What is the primary purpose for which this component was created?",
        "shipped": "Is this component distributed or shipped with the product?",
        "runtime_required": "Is this component required by the product at runtime?",
        "lifecycle_owner": "Which lifecycle owns versioning and releases for this component?",
        "reply": "Please reply using the following structured fields. Keep the invariant codes unchanged where applicable.",
        "required": "Clarification required",
        "none": "No clarification is required from the currently detected component candidates.",
    },
    "ko": {
        "title": "[PTSIP 확인 요청] {component_id}의 생성 목적",
        "intro": "PTSIP는 이 컴포넌트의 아키텍처 목적을 선언된 저장소 정보만으로 확인할 수 없어 추측성 분류를 중단했습니다.",
        "no_llm": "이 확인 요청은 고정된 규칙과 템플릿으로 생성됩니다. PTSIP는 질문을 만들기 위해 LLM을 호출하지 않습니다.",
        "purpose": "이 컴포넌트를 만든 주된 목적은 무엇입니까?",
        "shipped": "이 컴포넌트는 제품과 함께 배포되거나 제품 패키지에 포함됩니까?",
        "runtime_required": "제품 런타임에서 이 컴포넌트가 필요합니까?",
        "lifecycle_owner": "이 컴포넌트의 버전 관리와 릴리스 생명주기는 누가 소유합니까?",
        "reply": "다음 구조화된 필드로 답변해 주세요. 해당되는 경우 영문 고정 코드는 변경하지 않는 것이 좋습니다.",
        "required": "확인 필요",
        "none": "현재 탐지된 컴포넌트 후보에는 추가 확인이 필요하지 않습니다.",
    },
}

_OPTIONS = {
    "shipped": (
        ("YES", {"en": "Yes", "ko": "예"}),
        ("NO", {"en": "No", "ko": "아니요"}),
        ("CONDITIONAL", {"en": "Sometimes / conditional", "ko": "경우에 따라 다름"}),
        ("UNKNOWN", {"en": "Unknown", "ko": "모름"}),
    ),
    "runtime_required": (
        ("YES", {"en": "Yes", "ko": "예"}),
        ("NO", {"en": "No", "ko": "아니요"}),
        ("UNKNOWN", {"en": "Unknown", "ko": "모름"}),
    ),
    "lifecycle_owner": (
        ("PRODUCT", {"en": "Product lifecycle", "ko": "Product 생명주기"}),
        ("DEVELOPMENT_TOOLING", {"en": "Development tooling lifecycle", "ko": "개발 Tooling 생명주기"}),
        ("INDEPENDENT", {"en": "Independently governed", "ko": "독립적으로 관리"}),
        ("UNKNOWN", {"en": "Unknown", "ko": "모름"}),
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
