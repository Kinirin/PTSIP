from __future__ import annotations

import argparse
import hashlib
import hmac
import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any

from .github_client import GitHubAppClient
from .service import DecisionService
from .store import DecisionStore


class _Handler(BaseHTTPRequestHandler):
    server_version = "PTSIPControlPlane/0.3.1"

    @property
    def service(self) -> DecisionService:
        return self.server.service  # type: ignore[attr-defined]

    @property
    def agent_token(self) -> str:
        return self.server.agent_token  # type: ignore[attr-defined]

    @property
    def webhook_secret(self) -> bytes:
        return self.server.webhook_secret  # type: ignore[attr-defined]

    def _json(self, code: int, payload: dict[str, object]) -> None:
        raw = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def _body(self) -> bytes:
        length = int(self.headers.get("Content-Length", "0"))
        return self.rfile.read(length)

    def _authorized(self) -> bool:
        header = self.headers.get("Authorization", "")
        return bool(self.agent_token) and hmac.compare_digest(header, f"Bearer {self.agent_token}")

    def do_GET(self) -> None:
        if self.path == "/healthz":
            self._json(200, {"status": "ok", "service": "ptsip-control-plane", "version": "0.3.1"})
            return
        self._json(404, {"error": "not found"})

    def do_POST(self) -> None:
        raw = self._body()
        try:
            if self.path == "/github/webhook":
                signature = self.headers.get("X-Hub-Signature-256", "")
                expected = "sha256=" + hmac.new(self.webhook_secret, raw, hashlib.sha256).hexdigest()
                if not self.webhook_secret or not hmac.compare_digest(signature, expected):
                    self._json(401, {"error": "invalid webhook signature"})
                    return
                payload = json.loads(raw.decode("utf-8"))
                if not isinstance(payload, dict):
                    raise ValueError("webhook payload must be an object")
                event = self.headers.get("X-GitHub-Event", "")
                self.service.register_installation_event(payload)
                result = self.service.issue_comment(payload) if event == "issue_comment" else {"status": "ACCEPTED"}
                self._json(200, result)
                return

            if not self._authorized():
                self._json(401, {"error": "unauthorized"})
                return
            payload = json.loads(raw.decode("utf-8"))
            if not isinstance(payload, dict):
                raise ValueError("request body must be an object")
            if self.path == "/v1/gate":
                self._json(200, self.service.gate(payload))
            elif self.path == "/v1/resolve":
                self._json(200, self.service.resolve_agent(payload))
            elif self.path == "/v1/application":
                self._json(200, self.service.application(payload))
            else:
                self._json(404, {"error": "not found"})
        except (KeyError, ValueError) as exc:
            self._json(400, {"error": str(exc)})
        except Exception as exc:
            self._json(500, {"error": str(exc)})

    def log_message(self, format: str, *args: Any) -> None:
        if os.environ.get("PTSIP_APP_QUIET") != "1":
            super().log_message(format, *args)


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="ptsip-app", description="PTSIP GitHub App decision control plane")
    parser.add_argument("--host", default=os.environ.get("PTSIP_APP_HOST", "127.0.0.1"))
    parser.add_argument("--port", type=int, default=int(os.environ.get("PTSIP_APP_PORT", "8080")))
    parser.add_argument("--db", default=os.environ.get("PTSIP_APP_DB", "ptsip-control-plane.sqlite3"))
    return parser


def main(argv: list[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    agent_token = os.environ.get("PTSIP_CONTROL_PLANE_TOKEN", "")
    webhook_secret = os.environ.get("PTSIP_GITHUB_WEBHOOK_SECRET", "")
    if not agent_token:
        raise SystemExit("PTSIP_CONTROL_PLANE_TOKEN is required")
    if not webhook_secret:
        raise SystemExit("PTSIP_GITHUB_WEBHOOK_SECRET is required")
    store = DecisionStore(Path(args.db))
    service = DecisionService(store, GitHubAppClient())
    server = ThreadingHTTPServer((args.host, args.port), _Handler)
    server.service = service  # type: ignore[attr-defined]
    server.agent_token = agent_token  # type: ignore[attr-defined]
    server.webhook_secret = webhook_secret.encode("utf-8")  # type: ignore[attr-defined]
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()
    return 0
