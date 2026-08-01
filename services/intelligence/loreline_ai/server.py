from __future__ import annotations

import json
import os
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any

from .engine import answer_question


class Handler(BaseHTTPRequestHandler):
    server_version = "LorelineIntelligence/0.1"

    def do_GET(self) -> None:
        if self.path == "/health":
            self.send_json(HTTPStatus.OK, {"status": "ok", "service": "loreline-intelligence"})
        else:
            self.send_json(HTTPStatus.NOT_FOUND, {"error": {"code": "not_found", "message": "route not found"}})

    def do_POST(self) -> None:
        if self.path != "/v1/retrieve":
            self.send_json(HTTPStatus.NOT_FOUND, {"error": {"code": "not_found", "message": "route not found"}})
            return
        try:
            length = int(self.headers.get("Content-Length", "0"))
            if length <= 0 or length > 1_048_576:
                raise ValueError("request body must be between 1 byte and 1 MB")
            payload = json.loads(self.rfile.read(length))
            question = str(payload.get("question", "")).strip()
            documents = payload.get("documents", [])
            if not question or not isinstance(documents, list):
                raise ValueError("question and documents are required")
            self.send_json(HTTPStatus.OK, answer_question(question, documents))
        except (ValueError, json.JSONDecodeError) as exc:
            self.send_json(HTTPStatus.BAD_REQUEST, {"error": {"code": "invalid_request", "message": str(exc)}})

    def send_json(self, status: HTTPStatus, payload: dict[str, Any]) -> None:
        body = json.dumps(payload).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, format: str, *args: Any) -> None:
        print(f"loreline-intelligence: {format % args}")


def main() -> None:
    address = os.getenv("LORELINE_AI_ADDR", "127.0.0.1:8090")
    host, port = address.rsplit(":", 1)
    server = ThreadingHTTPServer((host, int(port)), Handler)
    print(f"Loreline intelligence listening on {address}")
    server.serve_forever()


if __name__ == "__main__":
    main()
