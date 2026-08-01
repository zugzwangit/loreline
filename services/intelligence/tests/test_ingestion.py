import unittest
from loreline_ai.ingestion import chunks, embedding, extract_text, fingerprint, normalize

class IngestionTests(unittest.TestCase):
    def test_html_extraction_and_chunk_overlap(self):
        text=extract_text("<h1>Policy</h1><p>Travel &amp; expense rules.</p>","text/html")
        self.assertEqual(text,"Policy Travel & expense rules.")
        parts=chunks("Sentence one. "+("word "*100)+"Sentence two.",200,20)
        self.assertGreater(len(parts),1)
        self.assertTrue(all(len(part)<=200 for part in parts))
    def test_deterministic_embedding(self):
        self.assertEqual(embedding("reset sso"),embedding("reset sso"))
        self.assertAlmostEqual(sum(v*v for v in embedding("reset sso")),1,places=5)
    def test_normalize_rejects_empty_document(self):
        with self.assertRaises(ValueError): normalize("Empty","tiny","text/plain",1200,160)
    def test_fingerprint_changes_with_content(self):
        self.assertNotEqual(fingerprint("a"),fingerprint("b"))

if __name__=="__main__":unittest.main()
