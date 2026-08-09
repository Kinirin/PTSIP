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
    reply_lines: list[str] = []
    for field in request.missing_fields:
        if field == "purpose":
            reply_lines.append('purpose: "<short description>"')
        elif field == "shipped":
            reply_lines.append("shipped: YES|NO|CONDITIONAL|UNKNOWN")
        elif field == "runtime_required":
            reply_lines.append("runtime_required: YES|NO|UNKNOWN")
        elif field == "lifecycle_owner":
            reply_lines.append("lifecycle_owner: PRODUCT|DEVELOPMENT_TOOLING|INDEPENDENT|UNKNOWN")
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
