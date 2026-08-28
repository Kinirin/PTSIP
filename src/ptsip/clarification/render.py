from __future__ import annotations

from .i18n import question, text
from .model import ClarificationRequest


def request_payload(request: ClarificationRequest, language: str) -> dict[str, object]:
    payload = request.as_dict()
    payload["questions"] = [question(field, language) for field in request.missing_fields]
    return payload


def render_issue(request: ClarificationRequest, language: str, repository_revision: str | None) -> tuple[str, str]:
    title = text(language, "title", component_id=request.component_id)
    paths = "\n".join(f"- `{item}`" for item in request.include) or "- `<unknown>`"
    reasons = "\n".join(f"- `{item}`" for item in request.reason_codes)
    questions: list[str] = []
    for index, field in enumerate(request.missing_fields, start=1):
        item = question(field, language)
        lines = [f"{index}. {item['prompt']}"]
        for option in item.get("options", []):
            lines.append(f"   - `{option['value']}` — {option['label']}")
        questions.append("\n".join(lines))
    reply_lines = [
        "format: ptsip-clarification-answer/v2",
        "decision:",
        "  classification: PRODUCT|DEVELOPMENT_TOOLING|DELIVERY|OPERATIONS|NEUTRAL_CONTRACT",
        '  purpose: "<short description>"',
        "  shipped: YES|NO",
        "  runtime_required: YES|NO",
        "  executable: YES|NO",
    ]
    revision = repository_revision or "UNKNOWN"
    body = f"""{text(language, 'intro')}

{text(language, 'no_llm')}

### Component
- ID: `{request.component_id}`
- Repository revision: `{revision}`
- Candidate paths:
{paths}

### Why PTSIP is asking
{reasons}

### Questions

{chr(10).join(questions)}

### Structured reply

{text(language, 'reply')}

```yaml
{chr(10).join(reply_lines)}
```

Tool `0.3.7` uses `classification` as the canonical lifecycle-ownership decision. Historical `ptsip-clarification-answer/v1` replies are compatibility input only and are never emitted by new clarification requests.

A coding agent may also resolve this decision in an active user chat. The first valid resolution wins; after the decision is resolved, late Issue replies are ignored.

<!-- ptsip-clarification-id: {request.id} -->
"""
    return title, body


def render_console(requests: tuple[ClarificationRequest, ...], language: str) -> str:
    if not requests:
        return text(language, "none")
    lines = [f"{text(language, 'required')}: {len(requests)}"]
    for request in requests:
        lines.append("")
        lines.append(f"{request.component_id} [{request.status.value}]")
        for path in request.include:
            lines.append(f"  path: {path}")
        for field in request.missing_fields:
            lines.append(f"  - {question(field, language)['prompt']}")
    return "\n".join(lines)
