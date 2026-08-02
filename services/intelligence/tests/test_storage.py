import unittest
from types import SimpleNamespace
from unittest.mock import MagicMock, patch

from loreline_ai.storage import ObjectStore


class StorageTests(unittest.TestCase):
    @patch("loreline_ai.storage.boto3.client")
    def test_encrypted_object_write(self, client_factory):
        client = MagicMock()
        client_factory.return_value = client
        cfg = SimpleNamespace(
            object_bucket="docs",
            object_endpoint="https://objects.test",
            object_access_key="key",
            object_secret_key="secret",
        )
        store = ObjectStore(cfg)
        result = store.put("tenant/doc/hash", b"content", "text/plain")
        self.assertEqual(result, "tenant/doc/hash")
        client.put_object.assert_called_once_with(
            Bucket="docs",
            Key="tenant/doc/hash",
            Body=b"content",
            ContentType="text/plain",
            ServerSideEncryption="AES256",
        )


if __name__ == "__main__":
    unittest.main()
