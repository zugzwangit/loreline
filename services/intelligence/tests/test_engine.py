import unittest

from loreline_ai.engine import answer_question, score, tokenize


DOCS = [
    {"id": "KB-1", "title": "SSO access reset", "content": "Wait 15 minutes, then retry the identity portal.", "source": "Runbook"},
    {"id": "KB-2", "title": "Parental leave", "content": "Eligible employees receive sixteen weeks of paid leave.", "source": "Policy"},
]


class EngineTests(unittest.TestCase):
    def test_tokenize_removes_noise(self):
        self.assertEqual(tokenize("How do I reset the SSO?"), ["reset", "sso"])

    def test_relevant_document_scores_higher(self):
        self.assertGreater(score("reset SSO access", DOCS[0]), score("reset SSO access", DOCS[1]))

    def test_answer_is_grounded_and_cited(self):
        result = answer_question("How can I reset SSO access?", DOCS)
        self.assertTrue(result["grounded"])
        self.assertEqual(result["citations"][0]["id"], "KB-1")
        self.assertIn("15 minutes", result["answer"])

    def test_unknown_question_does_not_invent(self):
        result = answer_question("Where is the lunar rover parked?", DOCS)
        self.assertFalse(result["grounded"])
        self.assertEqual(result["citations"], [])


if __name__ == "__main__":
    unittest.main()
