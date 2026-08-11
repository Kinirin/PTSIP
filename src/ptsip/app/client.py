from __future__ import annotations

import json
import os
import urllib.error
import urllib.request
from typing import Any


class ControlPlaneError(RuntimeError):
    pass


class ControlPlaneClient:
    def __init__(self, base_url: str, token: str | None = None):
        self.base_url = base_url.rstrip("/")
        self.token = token or os.environ.get("PTSIP_CONTROL_PLANE_TOKEN")
        if not self.base_url:
            raise ControlPlaneError("PTSIP control plane URL is not configured; use --control-plane")
        if not self.token:
            raise ControlPlaneError("PTSIP_CONTROL_PLANE_TOKEN is required")

    def _post(self, path: str, payload: dict[str, Any]) -> dict[str, Any]:
        request = urllib.request.Request(
            self.base_url + path,
            data=json.dumps(payload, ensure_ascii=False).encode("utf-8"),
            method="POST",
            headers={"Authorization": f"Bearer {self.token}", "Content-Type": "application/json"},
        )
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                raw = response.read()
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode("utf-8", errors="replace")
            raise ControlPlaneError(f"Control plane request failed: HTTP {exc.code}: {detail}") from exc
        except OSError as exc:
            raise ControlPlaneError(f"Control plane request failed: {exc}") from exc
        parsed = json.loads(raw.decode("utf-8"))
        if not isinstance(parsed, dict):
            raise ControlPlaneError("Control plane returned a non-object response")
        return parsed

    def gate(self, payload: dict[str, Any]) -> dict[str, Any]:
        return self._post("/v1/gate", payload)

    def decision(self, payload: dict[str, Any]) -> dict[str, Any]:
        return self._post("/v1/decision", payload)

    def resolve(self, payload: dict[str, Any]) -> dict[str, Any]:
        return self._post("/v1/resolve", payload)

    def application(self, payload: dict[str, Any]) -> dict[str, Any]:
        return self._post("/v1/application", payload)
