import os
import unittest
from unittest.mock import patch

from loreline_ai.config import Settings
from loreline_ai.connectors.http_json import HTTPJSONConnector


class Response:
    def raise_for_status(self):
        pass

    def json(self):
        return {"items": [{"id": "42", "title": "Reset SSO", "body": "Wait fifteen minutes."}], "next": "cursor-2"}


class ConnectorTests(unittest.TestCase):
    @patch("loreline_ai.connectors.http_json.httpx.get", return_value=Response())
    def test_generic_json_mapping_and_cursor(self, get):
        os.environ["LORELINE_CONNECTOR_TEST_TOKEN"] = "secret"
        cfg = Settings(connector_allowed_hosts="example.test")
        connector = HTTPJSONConnector(
            {
                "url": "https://example.test/api",
                "content_path": "body",
                "cursor_path": "next",
                "credential_env": "LORELINE_CONNECTOR_TEST_TOKEN",
            },
            cfg,
        )
        records, cursor = connector.fetch({})
        self.assertEqual(records[0].external_id, "42")
        self.assertEqual(records[0].content, "Wait fifteen minutes.")
        self.assertEqual(cursor, {"value": "cursor-2"})
        self.assertEqual(get.call_args.kwargs["headers"]["Authorization"], "Bearer secret")

    def test_rejects_unapproved_host_and_credential_name(self):
        cfg = Settings(connector_allowed_hosts="approved.test")
        with self.assertRaisesRegex(ValueError, "not in LORELINE_CONNECTOR_ALLOWED_HOSTS"):
            HTTPJSONConnector(
                {"url": "https://metadata.invalid/api", "credential_env": "LORELINE_CONNECTOR_TOKEN"}, cfg
            )
        with self.assertRaisesRegex(ValueError, "LORELINE_CONNECTOR_ prefix"):
            HTTPJSONConnector({"url": "https://approved.test/api", "credential_env": "DATABASE_URL"}, cfg)


if __name__ == "__main__":
    unittest.main()
