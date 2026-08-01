import os
import unittest
from unittest.mock import patch
from loreline_ai.connectors.http_json import HTTPJSONConnector

class Response:
    def raise_for_status(self): pass
    def json(self): return {"items":[{"id":"42","title":"Reset SSO","body":"Wait fifteen minutes."}],"next":"cursor-2"}

class ConnectorTests(unittest.TestCase):
    @patch("loreline_ai.connectors.http_json.httpx.get",return_value=Response())
    def test_generic_json_mapping_and_cursor(self,get):
        os.environ["TEST_CONNECTOR_TOKEN"]="secret"
        connector=HTTPJSONConnector({"url":"https://example.test/api","content_path":"body","cursor_path":"next","credential_env":"TEST_CONNECTOR_TOKEN"})
        records,cursor=connector.fetch({})
        self.assertEqual(records[0].external_id,"42");self.assertEqual(records[0].content,"Wait fifteen minutes.");self.assertEqual(cursor,{"value":"cursor-2"})
        self.assertEqual(get.call_args.kwargs["headers"]["Authorization"],"Bearer secret")

if __name__=="__main__":unittest.main()
