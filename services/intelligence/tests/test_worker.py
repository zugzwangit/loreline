import unittest

from loreline_ai.worker import process_job


class WorkerTests(unittest.TestCase):
    def test_unknown_jobs_fail_closed(self):
        with self.assertRaisesRegex(ValueError, "unsupported job kind"):
            process_job(None, {"kind": "unexpected"})


if __name__ == "__main__":
    unittest.main()
