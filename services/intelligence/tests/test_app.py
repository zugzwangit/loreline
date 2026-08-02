import unittest

from fastapi.testclient import TestClient

from loreline_ai.app import app

PAYLOAD = {
    "question": "How do I reset SSO?",
    "documents": [
        {
            "id": "one",
            "title": "SSO reset",
            "content": "Wait fifteen minutes and retry the identity portal.",
            "source": "Runbook",
        }
    ],
}


class AppTests(unittest.TestCase):
    def test_health_and_grounded_answer(self):
        with TestClient(app) as client:
            self.assertEqual(client.get("/livez").status_code, 200)
            self.assertEqual(client.get("/readyz").status_code, 200)
            response = client.post("/v1/retrieve", json=PAYLOAD)
            self.assertEqual(response.status_code, 200)
            self.assertTrue(response.json()["grounded"])
            self.assertEqual(response.json()["citations"][0]["id"], "one")

    def test_validation_fails_closed(self):
        with TestClient(app) as client:
            response = client.post("/v1/retrieve", json={"question": "", "documents": []})
            self.assertEqual(response.status_code, 422)
            self.assertEqual(response.json()["code"], "validation_error")

    def test_stream_contract(self):
        with TestClient(app) as client:
            response = client.post("/v1/stream", json=PAYLOAD)
            self.assertEqual(response.status_code, 200)
            self.assertIn("event: meta", response.text)
            self.assertIn("event: citations", response.text)
            self.assertIn("event: done", response.text)


if __name__ == "__main__":
    unittest.main()
