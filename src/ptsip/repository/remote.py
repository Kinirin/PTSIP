from __future__ import annotations

import re
from dataclasses import asdict, dataclass
from urllib.parse import urlparse


@dataclass(frozen=True)
class RepositoryRemote:
    name: str
    url: str
    host: str | None
    provider: str | None
    repository: str | None

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


def _strip_git_suffix(path: str) -> str:
    text = path.strip().strip("/")
    return text[:-4] if text.lower().endswith(".git") else text


def parse_remote(name: str, url: str) -> RepositoryRemote:
    text = url.strip()
    host: str | None = None
    repository_path: str | None = None

    scp_like = re.match(r"^(?:[^@\s]+@)?([^:/\s]+):(.+)$", text)
    if scp_like and "://" not in text:
        host = scp_like.group(1)
        repository_path = scp_like.group(2)
    else:
        parsed = urlparse(text)
        if parsed.scheme and parsed.hostname:
            host = parsed.hostname
            repository_path = parsed.path

    repository = None
    provider = None
    cleaned = _strip_git_suffix(repository_path or "")
    parts = [item for item in cleaned.split("/") if item]
    if host and host.casefold() == "github.com" and len(parts) == 2:
        provider = "github"
        repository = f"{parts[0]}/{parts[1]}"

    return RepositoryRemote(
        name=name,
        url=text,
        host=host,
        provider=provider,
        repository=repository,
    )
